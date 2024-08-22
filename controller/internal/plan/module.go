package plan

import (
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
)

type Executable interface {
}

type Module interface {
	Start(actions <-chan AdaptationAction) <-chan Executable
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

func (i *impl) Start(actions <-chan AdaptationAction) <-chan Executable {
	executables := make(chan Executable)
	go i.planAndExecute(actions)
	return executables
}

func (i *impl) Stop() {
}

func (i *impl) planAndExecute(actions <-chan AdaptationAction) {
	for range actions {
	}
}
