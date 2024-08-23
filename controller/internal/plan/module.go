package plan

import (
	"context"
	"fmt"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/execute"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/knowledge"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/utils"
	"log"
	"sync"
	"time"
)

type Module interface {
	Start(actions <-chan AdaptationAction)
	Stop()
}

type impl struct {
	knowledgeBase knowledge.Base
	wg            *sync.WaitGroup
	cfg           Config
	executeModule execute.Module
	log           *log.Logger
}

type Config struct {
	MergeTimeout     time.Duration `config:"merge_timeout"`
	ExecutionTimeout time.Duration `config:"execution_timeout"`
}

func NewModule(cfg Config, k knowledge.Base, e execute.Module) Module {
	return &impl{
		executeModule: e,
		cfg:           cfg,
		knowledgeBase: k,
		log:           utils.GetLogger("plan"),
	}
}

func (i *impl) Start(actions <-chan AdaptationAction) {
	i.wg = &sync.WaitGroup{}

	i.wg.Add(1)
	go i.planAndExecute(actions)
}

func (i *impl) Stop() {
	i.wg.Wait()
}

func (i *impl) planAndExecute(actions <-chan AdaptationAction) {
	defer i.wg.Done()
	ticker := time.NewTicker(i.cfg.MergeTimeout)
	ch := &changes{BanOrUnban: make(map[string]bool)}
	mergedChanges := 0
	for {
		select {
		case a, ok := <-actions:
			if !ok {
				return
			}
			ticker.Reset(i.cfg.MergeTimeout)
			a(ch)
			mergedChanges++
		case <-ticker.C:
			if mergedChanges == 0 {
				continue
			}
			err := i.executeChanges(ch)
			if err != nil {
				i.log.Printf("Error executing changes: %s", err)
			}
			mergedChanges = 0
			ch = &changes{BanOrUnban: make(map[string]bool)}
		}
	}
}

func (i *impl) executeChanges(ch *changes) error {
	ch.lock.Lock()
	defer ch.lock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), i.cfg.ExecutionTimeout)
	defer cancel()

	var err error
	for ip, ban := range ch.BanOrUnban {
		if ban {
			i.executeModule.BanIP(ip)
		} else {
			i.executeModule.UnbanIP(ip)
		}
	}
	if ch.Replicas != 0 {
		err = i.executeModule.ScaleService(ctx, ch.Replicas)
		if err != nil {
			err = fmt.Errorf("failed to execute scale change: %s", err)
		}
	}
	if ch.Limit != 0 {
		i.executeModule.SetRateLimit(ch.Limit)
	}
	return err
}
