package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	types "github.com/distribworks/dkron/v4/gen/proto/types/v1"
	"github.com/distribworks/dkron/v4/plugin"
)

const (
	defaultMaxRetries = 3
	defaultTimeout    = 3 * time.Second
	defaultBackoff    = 500 * time.Millisecond
)

// Webhook is the actual processor implementation
type Webhook struct{}

type webhookConfig struct {
	URL           string
	MaxRetries    int
	Timeout       time.Duration
	Backoff       time.Duration
	OnlyOnFailure bool
	Headers       map[string]string
}

// Process sends a webhook POST request when a job is done
func (w *Webhook) Process(args *plugin.ProcessorArgs) *types.Execution {
	cfg := parseWebhookConfig(args.Config)
	if cfg.URL == "" {
		return args.Execution
	}
	if cfg.OnlyOnFailure && args.Execution.Success {
		return args.Execution
	}

	body, err := json.Marshal(buildPayload(args.Execution))
	if err == nil {
		// Run asynchronously in a background goroutine so that the webhook call
		// doesn't block the main Dkron process execution pipeline.
		go sendWebhook(cfg, body)
	}

	return args.Execution
}

func buildPayload(execution *types.Execution) map[string]interface{} {
	payload := map[string]interface{}{
		"event":     "dkron.execution.finished",
		"job_name":  execution.JobName,
		"success":   execution.Success,
		"node_name": execution.NodeName,
		"output":    string(execution.Output),
		"group":     execution.Group,
		"attempt":   execution.Attempt,
	}

	if execution.StartedAt != nil {
		payload["started_at"] = execution.StartedAt.AsTime().Format(time.RFC3339)
	}
	if execution.FinishedAt != nil {
		payload["finished_at"] = execution.FinishedAt.AsTime().Format(time.RFC3339)
	}
	if execution.StartedAt != nil && execution.FinishedAt != nil {
		startedAt := execution.StartedAt.AsTime()
		finishedAt := execution.FinishedAt.AsTime()
		if !finishedAt.Before(startedAt) {
			payload["duration_sec"] = finishedAt.Sub(startedAt).Seconds()
		}
	}

	return payload
}

func sendWebhook(cfg webhookConfig, body []byte) {
	client := &http.Client{Timeout: cfg.Timeout}
	backoff := cfg.Backoff

	for i := 0; i < cfg.MaxRetries; i++ {
		req, err := http.NewRequest(http.MethodPost, cfg.URL, bytes.NewBuffer(body))
		if err != nil {
			break
		}
		req.Header.Set("Content-Type", "application/json")
		for key, value := range cfg.Headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				break
			}
		}

		if i < cfg.MaxRetries-1 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
}

func parseWebhookConfig(config plugin.Config) webhookConfig {
	cfg := webhookConfig{
		URL:        config["webhook_url"],
		MaxRetries: defaultMaxRetries,
		Timeout:    defaultTimeout,
		Backoff:    defaultBackoff,
		Headers:    map[string]string{},
	}

	if maxRetries, err := strconv.Atoi(config["max_retries"]); err == nil && maxRetries > 0 {
		cfg.MaxRetries = maxRetries
	}
	if timeout, err := time.ParseDuration(config["timeout"]); err == nil && timeout > 0 {
		cfg.Timeout = timeout
	}
	if backoff, err := time.ParseDuration(config["backoff"]); err == nil && backoff > 0 {
		cfg.Backoff = backoff
	}
	if onlyOnFailure, err := strconv.ParseBool(config["only_on_failure"]); err == nil {
		cfg.OnlyOnFailure = onlyOnFailure
	}
	if rawHeaders := config["headers"]; rawHeaders != "" {
		_ = json.Unmarshal([]byte(rawHeaders), &cfg.Headers)
	}

	return cfg
}
