package monitor

import (
	"context"
	"fmt"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/utils"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"log"
	"math"
	"sync"
	"time"
)

type Module interface {
	Start() <-chan Report
	Stop()
}

type Requests struct {
	TotalRate          float64
	NonLimitedRate     float64
	LimitedRatesStdDev float64
	GoodLatencyPercent float64
}

type Report struct {
	AverageCpuUtilization float64
	Requests              Requests
	PotentialAttackerIPs  map[string]float64
}

type impl struct {
	cfg           Config
	knowledgeBase knowledge.Base
	stop          context.CancelFunc
	wg            *sync.WaitGroup
	metricsClient v1.API
	log           *log.Logger
}

type Config struct {
	MetricsAddress           string        `config:"metrics_address"`
	MetricsPeriod            time.Duration `config:"metrics_period"`
	ReportPeriod             time.Duration `config:"report_period"`
	CpuQuota                 float64       `config:"cpu_quota"`
	AttackerPercentThreshold float64       `config:"attacker_percent_threshold"`
}

func NewModule(cfg Config, k knowledge.Base) Module {
	client, err := api.NewClient(api.Config{
		Address: cfg.MetricsAddress,
	})
	if err != nil {
		panic(err)
	}

	return &impl{
		cfg:           cfg,
		knowledgeBase: k,
		metricsClient: v1.NewAPI(client),
		log:           utils.GetLogger("monitor"),
	}
}

func (i *impl) Start() <-chan Report {
	reports := make(chan Report)
	ctx, cancel := context.WithCancel(context.Background())
	i.stop = cancel
	i.wg = &sync.WaitGroup{}

	i.wg.Add(1)
	go i.monitor(ctx, reports)

	return reports
}

func (i *impl) Stop() {
	i.stop()
	i.wg.Wait()
}

func (i *impl) monitor(ctx context.Context, reports chan<- Report) {
	ticker := time.NewTicker(i.cfg.ReportPeriod)
	defer i.wg.Done()
	defer ticker.Stop()
	defer close(reports)

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			requests, err := i.getRequestsReport(t)
			if err != nil {
				i.log.Println("failed to get requests report:", err)
				continue
			}
			i.log.Printf("requests: %+v\n", requests)
			cpu, err := i.getAverageCpuUtil(t)
			if err != nil {
				i.log.Println("failed to get cpu report:", err)
				continue
			}
			i.log.Printf("cpu util: %+v\n", cpu)
			attackerIPs, err := i.getPotentialAttackerIPs(t)
			if err != nil {
				i.log.Println("failed to get potential attacker ip report:", err)
				continue
			}
			i.log.Printf("attackers: %+v\n", attackerIPs)

			reports <- Report{
				AverageCpuUtilization: cpu,
				Requests:              requests,
				PotentialAttackerIPs:  attackerIPs,
			}
		}
	}
}

func (i *impl) getAverageCpuUtil(now time.Time) (float64, error) {
	query := fmt.Sprintf(`avg(rate(process_cpu_seconds_total{job="file-server"}[%s]))`, i.cfg.MetricsPeriod)
	value, err := singleValue(i.queryPrometheus(query, now))
	if value == 0 || value == math.NaN() {
		value = i.cfg.CpuQuota
	}
	return value / i.cfg.CpuQuota, err
}

func (i *impl) getRequestsReport(now time.Time) (Requests, error) {
	var err error
	result := Requests{}

	query1 := fmt.Sprintf(`sum(rate(traefik_entrypoint_requests_total{code!="403"}[%s]))`, i.cfg.MetricsPeriod)
	result.TotalRate, err = singleValue(i.queryPrometheus(query1, now))
	if err != nil {
		return result, err
	}

	query2 := fmt.Sprintf(`sum(rate(traefik_entrypoint_requests_total{code!="403", code!="429"}[%s]))`, i.cfg.MetricsPeriod)
	result.NonLimitedRate, err = singleValue(i.queryPrometheus(query2, now))
	if err != nil {
		return result, err
	}

	query3 := fmt.Sprintf(`sum(rate(traefik_entrypoint_request_duration_seconds_bucket{code!="403", code!="429", le="1.2"}[%s])) / sum(rate(traefik_entrypoint_request_duration_seconds_count{code!="403", code!="429"}[%s]))`, i.cfg.MetricsPeriod, i.cfg.MetricsPeriod)
	result.GoodLatencyPercent, err = singleValue(i.queryPrometheus(query3, now))
	if result.GoodLatencyPercent == 0 || math.IsNaN(result.GoodLatencyPercent) {
		result.GoodLatencyPercent = 1
	}
	if err != nil {
		return result, err
	}

	query4 := fmt.Sprintf(`stddev(rate(traefik_entrypoint_requests_total{code="429"}[%s]))`, i.cfg.MetricsPeriod)
	result.LimitedRatesStdDev, err = singleValue(i.queryPrometheus(query4, now))
	if err != nil {
		return result, err
	}
	return result, nil
}

func (i *impl) getPotentialAttackerIPs(now time.Time) (map[string]float64, error) {
	query := fmt.Sprintf(`sum(rate(traefik_entrypoint_requests_total{code="429"}[%s])) by (ip) / sum(rate(traefik_entrypoint_requests_total[%s])) by (ip) > %f`,
		i.cfg.MetricsPeriod, i.cfg.MetricsPeriod, i.cfg.AttackerPercentThreshold)
	return ipValues(i.queryPrometheus(query, now))
}

func (i *impl) queryPrometheus(query string, now time.Time) (model.Vector, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, warnings, err := i.metricsClient.Query(ctx, query, now)
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		i.log.Printf("Warnings: %v\n", warnings)
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return vector, fmt.Errorf("unexpected result format, expected vector")
	}

	return vector, nil
}

func singleValue(vector model.Vector, err error) (float64, error) {
	if err != nil {
		return 0, err
	}
	if len(vector) > 1 {
		return 0, fmt.Errorf("unexpected result format, len vector not 1 or 0: %v", len(vector))
	}
	if len(vector) == 0 {
		return 0, nil
	}
	return float64(vector[0].Value), nil
}

func ipValues(vector model.Vector, err error) (map[string]float64, error) {
	if err != nil {
		return nil, err
	}
	result := make(map[string]float64)
	for _, v := range vector {
		ip, ok := v.Metric["ip"]
		if !ok {
			continue
		}
		result[string(ip)] = float64(v.Value)
	}
	return result, nil
}
