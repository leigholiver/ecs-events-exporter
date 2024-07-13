package main

import "fmt"

type stdoutEmitter struct{}

func (stdoutEmitter) setEnvVars(config *Config) {
	// nothing to configure
}

func (stdoutEmitter) new(options map[string]interface{}) (emitter, error) {
	// nothing to configure
	return stdoutEmitter{}, nil
}

func (s stdoutEmitter) processLogBatch(batch logBatch) error {
	for _, msg := range batch.logs {
		fmt.Printf("[%v / %v] [%v %v %v %v %v] %v\n", msg.timestamp, msg.timestamp.UnixNano(),
			batch.accountID, batch.roleARN, batch.region, batch.cluster, batch.service.name, msg.msg)
	}
	return nil
}
