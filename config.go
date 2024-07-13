package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"

	"gopkg.in/yaml.v2"
)

// Config file schema definition
type Config struct {
	Logging struct {
		Logger  string                 `yaml:"logger"`
		Options map[string]interface{} `yaml:"options"`
	} `yaml:"logging"`
	ScanInterval       int      `yaml:"scan_interval"`
	IgnoreDefaultCreds bool     `yaml:"ignore_default_credentials"`
	Regions            []string `yaml:"regions"`
	Roles              []struct {
		RoleARN string   `yaml:"role_arn"`
		Regions []string `yaml:"regions"`
	} `yaml:"roles"`
	Clusters    []string            `yaml:"clusters"`
	ClusterTags []map[string]string `yaml:"cluster_tags"`
	Services    []string            `yaml:"services"`
	ServiceTags []map[string]string `yaml:"service_tags"`
	Logger      emitter
}

func loadConfig() (*Config, error) {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = "config.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("warning: config file %v does not exist - using empty configuration\n", path)
			data = []byte{}
		} else {
			return nil, err
		}
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	setEnvVars(&config)
	configureEmitterEnvVars(&config)
	setDefaults(&config)
	return &config, nil
}

func setEnvVars(config *Config) {
	// scanning interval
	scanInterval := os.Getenv("SCAN_INTERVAL")
	if scanInterval != "" {
		scanInt, err := strconv.Atoi(scanInterval)
		if err != nil {
			log.Fatalf("Invalid scan interval %s: %v\n", scanInterval, err)
		}
		config.ScanInterval = scanInt
	}
}

func setDefaults(config *Config) {
	// make sure we have something to scan
	if config.IgnoreDefaultCreds && len(config.Roles) == 0 {
		log.Fatal("Nothing to scan for - either add roles to your config file, or re-enable the default credentials")
	}

	// set scan interval
	if config.ScanInterval == 0 {
		config.ScanInterval = 60
	}

	// set up logging
	if config.Logging.Logger == "" {
		config.Logging.Logger = "stdout"
	}
	loggerObj, err := emitterFromConfig(config.Logging.Logger, config.Logging.Options)
	if err != nil {
		log.Fatal(err)
	}
	config.Logger = loggerObj

	// set up accounts/roles/regions
	if len(config.Regions) == 0 {
		defaultRegion, err := getCurrentRegion()
		if err != nil {
			log.Fatal(err)
		}
		config.Regions = []string{defaultRegion}
	}

	for i := range config.Roles {
		if len(config.Roles[i].Regions) == 0 {
			config.Roles[i].Regions = config.Regions
		}
	}
}

// Generic function to unmarshal map[string]interface{} into a struct
func configToStruct(m map[string]interface{}, result interface{}) error {
	optionsJSON, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("error marshalling map: %v", err)
	}

	val := reflect.ValueOf(result)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("result argument must be a pointer to a struct")
	}

	err = json.Unmarshal(optionsJSON, result)
	if err != nil {
		return fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return nil
}
