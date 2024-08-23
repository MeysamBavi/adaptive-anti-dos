package internal

import (
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/analyze"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/execute"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/monitor"
	"github.com/MeysamBavi/adaptive-anti-dos/controller/internal/plan"
	"github.com/knadh/koanf/providers/env"
	"log"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

const (
	tag       = "config"
	delimiter = "."
	prefix    = "AAD__"
	separator = "__"
)

type Config struct {
	Monitor monitor.Config `config:"monitor"`
	Analyze analyze.Config `config:"analyze"`
	Plan    plan.Config    `config:"plan"`
	Execute execute.Config `config:"execute"`
}

func Default() *Config {
	return &Config{
		Monitor: monitor.Config{
			MetricsAddress:           "http://localhost:9090",
			MetricsPeriod:            15 * time.Second,
			ReportPeriod:             6 * time.Second,
			CpuQuota:                 0.01,
			AttackerPercentThreshold: 0.25,
		},
		Analyze: analyze.Config{
			TargetUtilization:  0.7,
			MaxReplicas:        4,
			MinReplicas:        1,
			LimitedRequestCost: 50,
			ReplicaCost:        200,
			MinLimit:           5,
			UnbanCheckPeriod:   10 * time.Second,
			UnbanAfter:         time.Minute,
		},
		Plan: plan.Config{
			MergeTimeout:     3 * time.Second,
			ExecutionTimeout: 10 * time.Second,
		},
		Execute: execute.Config{
			InitialLimit: 50,
		},
	}
}

func LoadConfig() *Config {
	k := koanf.New(delimiter)
	{
		err := k.Load(structs.Provider(Default(), tag), nil)
		if err != nil {
			log.Fatalf("could not load default config: %s", err)
		}
	}

	{
		err := k.Load(file.Provider("/etc/config.yaml"), yaml.Parser())
		if err != nil {
			log.Printf("could not load yaml config: %s\n", err)
		}
	}

	{
		err := k.Load(env.Provider(prefix, delimiter, envCallBack), nil)
		if err != nil {
			log.Printf("could not load env variables for config: %s\n", err)
		}
	}

	var instance Config
	err := k.UnmarshalWithConf("", &instance, koanf.UnmarshalConf{
		Tag: tag,
	})

	if err != nil {
		log.Fatalf("could not unmarshal config: %s\n", err)
	}

	return &instance
}

func envCallBack(s string) string {
	base := strings.ToLower(strings.TrimPrefix(s, prefix))

	return strings.ReplaceAll(base, separator, delimiter)
}
