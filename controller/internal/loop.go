package internal

import (
	"context"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/analyze"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/execute"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/monitor"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/plan"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
}

func RunControlLoop(config Config) {
	k := knowledge.NewInMemoryBase()
	m := monitor.NewModule(monitor.Config{
		MetricsPeriod:            15 * time.Second,
		ReportPeriod:             6 * time.Second,
		CpuQuota:                 0.01,
		AttackerPercentThreshold: 0.25,
	}, k)
	a := analyze.NewModule(analyze.Config{
		TargetUtilization:  0.7,
		MaxReplicas:        4,
		MinReplicas:        1,
		LimitedRequestCost: 50,
		ReplicaCost:        200,
		MinLimit:           5,
		UnbanCheckPeriod:   10 * time.Second,
		UnbanAfter:         time.Minute,
	}, k)
	e := execute.NewModule(execute.Config{
		InitialLimit: 50,
	}, k)
	p := plan.NewModule(plan.Config{
		MergeTimeout:     3 * time.Second,
		ExecutionTimeout: 10 * time.Second,
	}, k, e)
	run(m, a, p, e)
}

func run(m monitor.Module, a analyze.Module, p plan.Module, e execute.Module) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// start MAPE-K modules
	reports := m.Start()
	actions := a.Start(reports)
	p.Start(actions)
	e.Start()

	// wait for termination signal
	<-ctx.Done()

	// stop modules
	m.Stop()
	a.Stop()
	p.Stop()
	e.Stop()
}
