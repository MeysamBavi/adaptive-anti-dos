package analyze

import (
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/monitor"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/plan"
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
}

type Config struct {
	TargetUtilization  float64
	MaxReplicas        int
	MinReplicas        int
	LimitedRequestCost float64
	ReplicaCost        float64
	MinLimit           float64
	UnbanCheckPeriod   time.Duration
	UnbanAfter         time.Duration
}

func NewModule(cfg Config, k knowledge.Base) Module {
	return &impl{
		cfg:           cfg,
		knowledgeBase: k,
	}
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
			log.Println("unbanning", ip)
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
		log.Println("banning ip", ip)
		result = append(result, plan.BanIP(ip))
	}
	return
}

func (i *impl) getResourceAdaptationActions(r monitor.Report) []plan.AdaptationAction {
	replicas := float64(i.knowledgeBase.CurrentReplicas())
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
		return i.adaptResources(y, normalizeX(y/k))
	}

	xUpper = min(xUpper, r.Requests.TotalRate/(r.Requests.NonLimitedRate*k))
	xLower = max(xLower, i.cfg.MinLimit/(float64(i.knowledgeBase.CurrentLimit())*k))

	slope := -k*r.Requests.NonLimitedRate*i.cfg.LimitedRequestCost + replicas*i.cfg.ReplicaCost
	var x float64
	if slope > 0 {
		x = xLower
	} else {
		x = xUpper
	}
	x = normalizeX(x)
	y := k * x

	return i.adaptResources(y, x)
}

func (i *impl) adaptResources(y float64, x float64) (result []plan.AdaptationAction) {
	oldReplicas := i.knowledgeBase.CurrentReplicas()
	nr := float64(oldReplicas) * x
	if math.IsNaN(nr) || int(nr) == 0 {
		log.Println("newReplicas is NaN")
		return nil
	}
	newReplicas := int(nr)
	if newReplicas == oldReplicas {
		log.Println("old replicas is equal to new replicas:", newReplicas)
	} else {
		result = append(result, plan.AdaptReplicas(newReplicas))
		log.Printf("setting new replicas = %d", newReplicas)
	}

	if math.Abs(y-1) > 0.0001 {
		newLimit := int(math.Ceil(float64(i.knowledgeBase.CurrentLimit()) * y))
		result = append(result, plan.AdaptLimit(newLimit))
		log.Printf("setting new limit = %d", newReplicas)
	}

	return
}

func (i *impl) startUnbanner() <-chan string {
	ch := make(chan string)
	go func() {
		for {
			time.Sleep(i.cfg.UnbanCheckPeriod)
			ips, banTime := i.knowledgeBase.CurrentBannedIPs()
			if len(ips) == 0 || banTime.IsZero() || time.Since(banTime) < i.cfg.UnbanAfter {
				continue
			}
			for _, ip := range ips {
				ch <- ip
			}
		}
	}()

	return ch
}
