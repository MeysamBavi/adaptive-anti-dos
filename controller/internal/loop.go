package internal

import (
	"context"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/analyze"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/execute"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/monitor"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/plan"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func RunControlLoop(config *Config) {
	log.Printf("Config: %+v\n", *config)

	k := knowledge.NewInMemoryBase()
	m := monitor.NewModule(config.Monitor, k)
	a := analyze.NewModule(config.Analyze, k)
	e := execute.NewModule(config.Execute, k)
	p := plan.NewModule(config.Plan, k, e)
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
