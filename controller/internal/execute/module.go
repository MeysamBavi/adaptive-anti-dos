package execute

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"log"
	"net/http"
	"slices"
	"strings"
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
	log           *log.Logger
}

type Config struct {
	InitialLimit int `config:"initial_limit"`
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
		log:           utils.GetLogger("execute"),
	}
	err = i.refreshReplicas()
	if err != nil {
		panic(fmt.Errorf("failed to get initial replicas: %w", err))
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

func (i *impl) findService(ctx context.Context) (*swarm.Service, error) {
	services, err := i.dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, err
	}
	idx := slices.IndexFunc(services, func(s swarm.Service) bool {
		return strings.Contains(s.Spec.Name, serviceName)
	})

	if idx < 0 {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}
	service := services[idx]
	spec := service.Spec
	if spec.Mode.Replicated == nil || spec.Mode.Replicated.Replicas == nil {
		return nil, fmt.Errorf("replicated service %s not found", serviceName)
	}
	return &service, nil
}

func (i *impl) refreshReplicas() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := i.findService(ctx)
	if err != nil {
		return err
	}

	r := int(*s.Spec.Mode.Replicated.Replicas)
	i.knowledgeBase.SetReplicas(r)
	i.log.Println("successfully refreshed replicas:", r)
	return nil
}

func (i *impl) ScaleService(ctx context.Context, replicas int) error {
	service, err := i.findService(ctx)
	if err != nil {
		return err
	}

	// Update the service with the new replica count
	serviceSpec := service.Spec
	if serviceSpec.Mode.Replicated == nil {
		return fmt.Errorf("replicated service %s not found", serviceName)
	}
	r := uint64(replicas)
	serviceSpec.Mode.Replicated.Replicas = &r

	_, err = i.dockerClient.ServiceUpdate(ctx, service.ID, service.Version, serviceSpec, types.ServiceUpdateOptions{})
	if err != nil {
		return err
	}

	i.knowledgeBase.SetReplicas(replicas)
	i.log.Printf("Service %s scaled to %d replicas", serviceName, replicas)
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

	i.knowledgeBase.RangeBannedIPs(func(oldBanned string, _ time.Time) {
		_, ok := i.banOrUnban.Load(oldBanned)
		if !ok {
			i.banOrUnban.Store(oldBanned, true)
		}
	})
	newBannedIPs := make([]string, 0)
	i.banOrUnban.Range(func(ip, ban any) bool {
		if ban.(bool) {
			newBannedIPs = append(newBannedIPs, ip.(string))
		}
		return true
	})

	empty := false
	if len(newBannedIPs) == 0 {
		newBannedIPs = append(newBannedIPs, "11.0.0.0")
		empty = true
	}
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
	if empty {
		newBannedIPs = nil
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		i.log.Println("Error encoding json, serving gateway config response", err)
		return
	}
	i.knowledgeBase.SetLimit(int(limit))
	i.banOrUnban.Range(func(ip, value any) bool {
		if value.(bool) {
			i.knowledgeBase.BanIP(ip.(string))
		} else {
			i.knowledgeBase.UnbanIP(ip.(string))
		}
		return true
	})
	i.log.Printf("succesfully set limit: %v", limit)
	i.log.Printf("succesfully set banned IPs: %v", newBannedIPs)
}
