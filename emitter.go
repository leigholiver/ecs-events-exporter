package main

import (
	"fmt"
	"time"
)

type emitter interface {
	setEnvVars(config *Config)
	new(options map[string]interface{}) (emitter, error)
	processLogBatch(batch logBatch) error
}

type logRow struct {
	msg       string
	timestamp time.Time
}

type logBatch struct {
	accountID string
	roleARN   string
	region    string
	cluster   string
	service   ecsService
	logs      []logRow
}

// small implementation note as im sure ill forget.. we configure the env vars for
// every emitter before instantiating the emitter. this allows the emitter type to be
// inferred from a particular set of env vars, ie running with LOKI_URL=http://example.com
// will automatically set up the lokiEmitter
func configureEmitterEnvVars(config *Config) {
	emitters := []emitter{
		stdoutEmitter{},
		lokiEmitter{},
	}

	for _, e := range emitters {
		e.setEnvVars(config)
	}
}

func emitterFromConfig(emittertype string, options map[string]interface{}) (emitter, error) {
	switch emittertype {
	case "loki":
		return lokiEmitter{}.new(options)
	case "stdout":
		return stdoutEmitter{}.new(options)
	}
	return nil, fmt.Errorf("unknown emitter '%v'", emittertype)
}
