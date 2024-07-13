package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type ecsClientConfig struct {
	client    *ecs.Client
	accountID string
	roleARN   string
	region    string
}

type ecsService struct {
	name string
	tags []ecsTypes.Tag
}

func getClientsForRole(clients []*ecsClientConfig, regions []string, roleARN string) []*ecsClientConfig {
	for _, region := range regions {
		client, err := getECSClient(roleARN, region)
		if err != nil {
			fmt.Printf("failed to create ECS client: %v\n", err)
		} else {
			clients = append(clients, client)
		}
	}
	return clients
}

func getECSClient(roleARN string, region string) (*ecsClientConfig, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	stsClient := sts.NewFromConfig(cfg)
	if roleARN != "" {
		stsProvider := stscreds.NewAssumeRoleProvider(stsClient, roleARN)
		cfg.Credentials = aws.NewCredentialsCache(stsProvider)
		stsClient = sts.NewFromConfig(cfg)
	}

	account, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	return &ecsClientConfig{
		ecs.NewFromConfig(cfg),
		*account.Account,
		*account.Arn,
		region,
	}, nil
}

func listClusters(client *ecs.Client) ([]string, error) {
	clusters := []string{}
	input := &ecs.ListClustersInput{}
	for {
		result, err := client.ListClusters(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, result.ClusterArns...)
		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}
	return clusters, nil
}

func describeClusters(client *ecs.Client, clusters []string, tags []map[string]string) ([]string, error) {
	outputClusters := []string{}
	input := &ecs.DescribeClustersInput{
		Clusters: clusters,
		Include:  []ecsTypes.ClusterField{ecsTypes.ClusterFieldTags},
	}
	result, err := client.DescribeClusters(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	for _, cluster := range result.Clusters {
		if matchesTagFilters(tags, cluster.Tags) {
			outputClusters = append(outputClusters, *cluster.ClusterName)
		}
	}
	for _, f := range result.Failures {
		log.Printf("warning: %v %v\n", *f.Reason, *f.Arn)
	}
	return outputClusters, nil
}

func getClusterList(client *ecs.Client, clusters []string, tags []map[string]string) ([]string, error) {
	if len(clusters) == 0 {
		resp, err := listClusters(client)
		if err != nil {
			return nil, err
		}
		clusters = resp
	}
	return describeClusters(client, clusters, tags)
}

func listServices(client *ecs.Client, cluster string) ([]string, error) {
	services := []string{}
	input := &ecs.ListServicesInput{Cluster: &cluster}
	for {
		result, err := client.ListServices(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		services = append(services, result.ServiceArns...)
		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}
	return services, nil
}

func describeServices(client *ecs.Client, cluster string, services []string, tags []map[string]string) ([]ecsService, error) {
	outputServices := []ecsService{}
	chunkSize := 10 // can only describe 10 services at a time
	length := len(services)
	var input *ecs.DescribeServicesInput
	for start := 0; start < length; start += chunkSize {
		end := start + chunkSize
		if end > length {
			end = length
		}
		chunk := services[start:end]

		input = &ecs.DescribeServicesInput{
			Cluster:  &cluster,
			Services: chunk,
			Include:  []ecsTypes.ServiceField{ecsTypes.ServiceFieldTags},
		}
		result, err := client.DescribeServices(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		for _, service := range result.Services {
			if matchesTagFilters(tags, service.Tags) {
				outputServices = append(outputServices, ecsService{*service.ServiceName, service.Tags})
			}
		}
		for _, f := range result.Failures {
			log.Printf("warning: %v %v\n", *f.Reason, *f.Arn)
		}
	}

	return outputServices, nil
}

func getServiceList(client *ecs.Client, cluster string, services []string, tags []map[string]string) ([]ecsService, error) {
	if len(services) == 0 {
		resp, err := listServices(client, cluster)
		if err != nil {
			return nil, err
		}
		services = resp
	}
	return describeServices(client, cluster, services, tags)
}

func getDeploymentLogs(client *ecs.Client, cluster string, service string, timestamp time.Time) ([]logRow, error) {
	input := &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: []string{service},
	}

	output, err := client.DescribeServices(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	var logs []logRow
	for _, svc := range output.Services {
		for _, event := range svc.Events {
			if event.CreatedAt.After(timestamp) {
				logs = append(logs, logRow{*event.Message, *event.CreatedAt})
			}
		}
	}

	return logs, nil
}

func getCurrentRegion() (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", err
	}
	return cfg.Region, nil
}

func matchesTagFilters(filterRules []map[string]string, tags []ecsTypes.Tag) bool {
	if len(filterRules) == 0 {
		return true
	}
	for _, rule := range filterRules {
		match := true
		for key, value := range rule {
			found := false
			for _, tag := range tags {
				if *tag.Key == key && *tag.Value == value {
					found = true
					break
				}
			}
			if !found {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
