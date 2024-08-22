package analyze

import (
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/monitor"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/plan"
)

type Module interface {
	Start(symptoms <-chan monitor.Report) <-chan plan.AdaptationAction
	Stop()
}

type impl struct {
	knowledgeBase knowledge.Base
}

func NewModule(k knowledge.Base) Module {
	return &impl{
		knowledgeBase: k,
	}
}

func (i *impl) Start(reports <-chan monitor.Report) <-chan plan.AdaptationAction {
	actions := make(chan plan.AdaptationAction)
	go i.analyze(reports)
	return actions
}

func (i *impl) Stop() {
}

func (i *impl) analyze(reports <-chan monitor.Report) {
	for range reports {
	}
}
