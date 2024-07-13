package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	// exit gracefully on ctrl+c
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		os.Exit(0)
	}()

	// load in the config
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// create a bunch of clients
	var clients []*ecsClientConfig
	if !config.IgnoreDefaultCreds {
		clients = getClientsForRole(clients, config.Regions, "")
	}
	for _, role := range config.Roles {
		clients = getClientsForRole(clients, role.Regions, role.RoleARN)
	}
	// bail out if we have no clients to scrape
	if len(clients) == 0 {
		log.Fatalln("fatal: no aws credentials to use")
	}

	// main loop
	var timestamp time.Time
	for {
		timestamp = time.Now().Add(-time.Duration(config.ScanInterval) * time.Second)
		errCh := make(chan error, 1)
		var wg sync.WaitGroup
		for _, client := range clients {
			wg.Add(1)
			go func(client ecsClientConfig) {
				defer wg.Done()

				clusters, err := getClusterList(client.client, config.Clusters, config.ClusterTags)
				if err != nil {
					errCh <- err
					return
				}

				for _, cluster := range clusters {
					wg.Add(1)
					go func(client ecsClientConfig, cluster string) {
						defer wg.Done()

						services, err := getServiceList(client.client, cluster, config.Services, config.ServiceTags)
						if err != nil {
							errCh <- err
							return
						}

						for _, service := range services {
							wg.Add(1)
							go func(client ecsClientConfig, cluster string, service ecsService) {
								defer wg.Done()

								logs, err := getDeploymentLogs(client.client, cluster, service.name, timestamp)
								if err != nil {
									errCh <- err
									return
								}
								if len(logs) > 0 {
									err = config.Logger.processLogBatch(logBatch{client.accountID, client.roleARN, client.region, cluster, service, logs})
									if err != nil {
										errCh <- err
										return
									}
								}
							}(client, cluster, service)
						}
					}(client, cluster)
				}
			}(*client)
		}

		go func() {
			wg.Wait()
			close(errCh)
		}()
		for err := range errCh {
			if err != nil {
				log.Printf("error: %v\n", err)
			}
		}

		time.Sleep(time.Duration(config.ScanInterval) * time.Second)
	}
}
