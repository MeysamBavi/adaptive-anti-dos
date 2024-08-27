package analyze

import (
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/monitor"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/plan"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/utils"
	"log"
	"math"
	"sync"
	"time"
)

type Module interface {
	Start(symptoms <-chan monitor.Report) <-chan plan.AdaptationAction
	Stop()
}

type impl struct {
	knowledgeBase knowledge.Base
	wg            *sync.WaitGroup
	cfg           Config
	log           *log.Logger
}

type Config struct {
	TargetUtilization  float64       `config:"target_utilization"`
	MaxReplicas        int           `config:"max_replicas"`
	MinReplicas        int           `config:"min_replicas"`
	LimitedRequestCost float64       `config:"limited_request_cost"`
	ReplicaCost        float64       `config:"replica_cost"`
	MinLimit           float64       `config:"min_limit"`
	UnbanCheckPeriod   time.Duration `config:"unban_check_period"`
	UnbanAfter         time.Duration `config:"unban_after"`
}

func NewModule(cfg Config, k knowledge.Base) Module {
	i := &impl{
		cfg:           cfg,
		knowledgeBase: k,
		log:           utils.GetLogger("analyze"),
	}
	return i
}

func (i *impl) Start(reports <-chan monitor.Report) <-chan plan.AdaptationAction {
	actions := make(chan plan.AdaptationAction)
	i.wg = &sync.WaitGroup{}

	i.wg.Add(1)
	unbans := i.startUnbanner()
	go i.analyze(reports, unbans, actions)

	return actions
}

func (i *impl) Stop() {
	i.wg.Wait()
}

func (i *impl) analyze(reports <-chan monitor.Report, unbans <-chan string, actions chan<- plan.AdaptationAction) {
	defer i.wg.Done()
	defer close(actions)

	for {
		select {
		case r, ok := <-reports:
			if !ok {
				return
			}
			for _, a := range i.getActions(r) {
				actions <- a
			}
		case ip := <-unbans:
			i.log.Println("unbanning", ip)
			actions <- plan.UnbanIP(ip)
		}
	}
}

func (i *impl) getActions(r monitor.Report) []plan.AdaptationAction {
	var actions []plan.AdaptationAction
	actions = append(actions, i.getBanAdaptationActions(r)...)
	actions = append(actions, i.getResourceAdaptationActions(r)...)
	return actions
}

func (i *impl) getBanAdaptationActions(r monitor.Report) (result []plan.AdaptationAction) {
	for ip := range r.PotentialAttackerIPs {
		i.log.Println("banning ip", ip)
		result = append(result, plan.BanIP(ip))
	}
	return
}

func (i *impl) getResourceAdaptationActions(r monitor.Report) []plan.AdaptationAction {
	replicas := float64(i.knowledgeBase.CurrentReplicas())
	limit := float64(i.knowledgeBase.CurrentLimit())

	xUpper := float64(i.cfg.MaxReplicas) / replicas
	xLower := float64(i.cfg.MinReplicas) / replicas
	normalizeX := func(x float64) float64 {
		if x >= xUpper {
			x = xUpper
			x *= math.Floor(x*replicas) / (x * replicas)
		} else if x <= xLower {
			x = xLower
			x *= math.Ceil(x*replicas) / (x * replicas)
		} else {
			x *= math.RoundToEven(x*replicas) / (x * replicas)
		}
		return x
	}

	k := math.Sqrt((i.cfg.TargetUtilization / r.AverageCpuUtilization) * r.Requests.GoodLatencyPercent)

	if r.Requests.TotalRate-r.Requests.NonLimitedRate < 0.1 ||
		math.IsNaN(r.Requests.LimitedRatesStdDev) || r.Requests.LimitedRatesStdDev > 4 {
		y := 1.0
		return i.adaptResources(y, normalizeX(y/k), limit, replicas)
	}

	xUpper = min(xUpper, r.Requests.TotalRate/(r.Requests.NonLimitedRate*k))
	xLower = max(xLower, i.cfg.MinLimit/(limit*k))

	if xLower > xUpper {
		i.log.Println("NO SOLUTION!!!")
		return nil
	}

	slope := -k*r.Requests.NonLimitedRate*i.cfg.LimitedRequestCost + replicas*i.cfg.ReplicaCost
	var x float64
	if slope > 0 {
		x = xLower
	} else {
		x = xUpper
	}
	x = normalizeX(x)
	y := k * x

	return i.adaptResources(y, x, limit, replicas)
}

func (i *impl) adaptResources(y, x, limit, oldReplicas float64) (result []plan.AdaptationAction) {
	nr := math.Round(oldReplicas * x)
	if math.IsNaN(nr) || int(nr) == 0 {
		i.log.Println("newReplicas is NaN")
		return nil
	}
	newReplicas := int(nr)
	if newReplicas == int(oldReplicas) {
		i.log.Println("old replicas is equal to new replicas:", newReplicas)
	} else {
		result = append(result, plan.AdaptReplicas(newReplicas))
		i.log.Printf("setting new replicas = %d", newReplicas)
	}

	if math.Abs(y-1) > 0.0001 {
		newLimit := int(math.Ceil(limit * y))
		result = append(result, plan.AdaptLimit(newLimit))
		i.log.Printf("setting new limit = %d", newLimit)
	}

	return
}

func (i *impl) startUnbanner() <-chan string {
	ch := make(chan string)
	go func() {
		for {
			time.Sleep(i.cfg.UnbanCheckPeriod)
			i.knowledgeBase.RangeBannedIPs(func(ip string, t time.Time) {
				if time.Since(t) >= i.cfg.UnbanAfter {
					ch <- ip
				}
			})
		}
	}()

	return ch
}
