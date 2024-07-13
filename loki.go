package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

type lokiEmitter struct {
	URL   string `json:"url"`
	OrgID string `json:"org_id"`
}

type lokiLogRow struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

type lokiPayload struct {
	Streams []lokiLogRow `json:"streams"`
}

func (lokiEmitter) new(options map[string]interface{}) (emitter, error) {
	config := lokiEmitter{}
	err := configToStruct(options, &config)
	if err != nil {
		return nil, err
	}

	// validate configuration
	if config.URL == "" {
		return nil, errors.New("must set a loki URL to send logs to")
	}

	if config.OrgID == "" {
		fmt.Println("warning: no loki org id set")
	}

	return config, nil
}

func (s lokiEmitter) processLogBatch(batch logBatch) error {
	payload := logBatchToLokiPayload(batch)
	err := s.pushPayloadToLoki(payload)
	if err != nil {
		return err
	}
	return nil
}

func (s lokiEmitter) setEnvVars(config *Config) {
	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL != "" {
		config.Logging.Logger = "loki"
		if config.Logging.Options == nil {
			config.Logging.Options = map[string]interface{}{
				"url": lokiURL,
			}
		} else {
			config.Logging.Options["url"] = lokiURL
		}
	}

	lokiOrgID := os.Getenv("LOKI_ORG_ID")
	if lokiOrgID != "" {
		config.Logging.Logger = "loki"
		if config.Logging.Options == nil {
			config.Logging.Options = map[string]interface{}{
				"org_id": lokiOrgID,
			}
		} else {
			config.Logging.Options["org_id"] = lokiOrgID
		}
	}
}

func logBatchToLokiPayload(batch logBatch) lokiPayload {
	labels := map[string]string{
		"aws_account":  batch.accountID,
		"role_arn":     batch.roleARN,
		"aws_region":   batch.region,
		"ecs_cluster":  batch.cluster,
		"service_name": batch.service.name,
	}

	// use promutil from yace to ensure tag naming consistency
	for _, tag := range batch.service.tags {
		_, key := promutil.PromStringTag(fmt.Sprintf("tag_%v", *tag.Key), true)
		labels[key] = *tag.Value
	}

	var values [][2]string
	for _, msg := range batch.logs {
		values = append(values, [2]string{strconv.FormatInt(msg.timestamp.UnixNano(), 10), msg.msg})
	}

	var payloadStreams []lokiLogRow
	payloadStreams = append(payloadStreams, lokiLogRow{labels, values})
	return lokiPayload{payloadStreams}
}

func (s lokiEmitter) pushPayloadToLoki(payload lokiPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", s.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.OrgID != "" {
		req.Header.Set("X-Scope-OrgID", s.OrgID)
	}

	// Add Basic Auth credentials
	//req.SetBasicAuth("username", "password")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("error: recieved response status: %v\n%v", resp.Status, string(body))
	}
	return nil
}
