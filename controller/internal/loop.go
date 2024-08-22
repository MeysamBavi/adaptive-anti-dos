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
		MetricsPeriod:            12 * time.Second,
		ReportPeriod:             4 * time.Second,
		AttackerPercentThreshold: 0.25,
	}, k)
	a := analyze.NewModule(k)
	p := plan.NewModule(k)
	e := execute.NewModule(k)
	run(m, a, p, e)
}

func run(m monitor.Module, a analyze.Module, p plan.Module, e execute.Module) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// start MAPE-K modules
	reports := m.Start()
	actions := a.Start(reports)
	executables := p.Start(actions)
	e.Start(executables)

	// wait for termination signal
	<-ctx.Done()

	// stop modules
	m.Stop()
	a.Stop()
	p.Stop()
	e.Stop()
}
