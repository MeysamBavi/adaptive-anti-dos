package execute

import (
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/plan"
)

type Module interface {
	Start(executables <-chan plan.Executable)
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

func (i impl) Start(executables <-chan plan.Executable) {
}

func (i impl) Stop() {
}
