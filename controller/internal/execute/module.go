package execute

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Module interface {
	Start()
	ScaleService(ctx context.Context, replicas int) error
	SetRateLimit(limit int)
	BanIP(ip string)
	UnbanIP(ip string)
	Stop()
}

type impl struct {
	knowledgeBase knowledge.Base
	dockerClient  *client.Client
	limit         atomic.Int32
	banOrUnban    *sync.Map
}

type Config struct {
	InitialLimit    int
	InitialReplicas int
}

func NewModule(config Config, k knowledge.Base) Module {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	i := &impl{
		knowledgeBase: k,
		dockerClient:  dockerClient,
		banOrUnban:    &sync.Map{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = i.ScaleService(ctx, config.InitialReplicas)
	if err != nil {
		panic(fmt.Errorf("failed to set initial replicas: %w", err))
	}
	i.SetRateLimit(config.InitialLimit)
	i.knowledgeBase.SetLimit(config.InitialLimit)

	return i
}

func (i *impl) Start() {
	go func() {
		http.HandleFunc("/gateway", i.handleGatewayRequest)
		err := http.ListenAndServe(":6041", nil)
		if err != nil {
			log.Fatal("Error starting HTTP server", err)
		}
	}()
}

func (i *impl) Stop() {
}

const (
	serviceName = "file-server"
)

func (i *impl) ScaleService(ctx context.Context, replicas int) error {
	// Get the service details
	services, err := i.dockerClient.ServiceList(ctx, types.ServiceListOptions{
		Filters: filters.NewArgs(filters.Arg("name", serviceName)),
	})
	if err != nil {
		return err
	}

	if len(services) == 0 {
		return fmt.Errorf("service %s not found", serviceName)
	}

	// Update the service with the new replica count
	service := services[0]
	serviceSpec := service.Spec
	r := uint64(replicas)
	serviceSpec.Mode.Replicated.Replicas = &r

	_, err = i.dockerClient.ServiceUpdate(ctx, service.ID, service.Version, serviceSpec, types.ServiceUpdateOptions{})
	if err != nil {
		return err
	}

	i.knowledgeBase.SetReplicas(replicas)
	log.Printf("Service %s scaled to %d replicas", serviceName, replicas)
	return nil
}

func (i *impl) SetRateLimit(limit int) {
	i.limit.Store(int32(limit))
}

func (i *impl) BanIP(ip string) {
	i.banOrUnban.Store(ip, true)
}

func (i *impl) UnbanIP(ip string) {
	i.banOrUnban.Store(ip, false)
}

func (i *impl) handleGatewayRequest(w http.ResponseWriter, _ *http.Request) {
	limit := i.limit.Load()

	oldBannedIPs, _ := i.knowledgeBase.CurrentBannedIPs()
	for _, oldBanned := range oldBannedIPs {
		_, ok := i.banOrUnban.Load(oldBanned)
		if !ok {
			i.banOrUnban.Store(oldBanned, true)
		}
	}
	newBannedIPs := make([]string, 0, len(oldBannedIPs))
	i.banOrUnban.Range(func(ip, ban any) bool {
		if ban.(bool) {
			newBannedIPs = append(newBannedIPs, ip.(string))
		}
		i.banOrUnban.Delete(ip)
		return true
	})

	response := map[string]any{
		"http": map[string]any{
			"middlewares": map[string]any{
				"fs-rate-limit": map[string]any{
					"rateLimit": map[string]any{
						"average": limit,
						"burst":   limit,
						"period":  1,
						"sourceCriterion": map[string]any{
							"ipStrategy": map[string]any{
								"depth": 1,
							},
						},
					},
				},
				"fs-deny-ip": map[string]any{
					"plugin": map[string]any{
						"denyip": map[string]any{
							"ipDenyList": newBannedIPs,
						},
					},
				},
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Println("Error encoding json, serving gateway config response", err)
		return
	}
	i.knowledgeBase.SetLimit(int(limit))
	i.knowledgeBase.SetBannedIPs(newBannedIPs)
	log.Printf("succesfully set limit: %v", limit)
	log.Printf("succesfully set banned IPs: %v", newBannedIPs)
}
