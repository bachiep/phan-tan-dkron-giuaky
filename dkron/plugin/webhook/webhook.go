package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/distribworks/dkron/v4/plugin"
	types "github.com/distribworks/dkron/v4/gen/proto/types/v1"
)

// Webhook is the actual processor implementation
type Webhook struct{}

// Process sends a webhook POST request when a job is done
func (w *Webhook) Process(args *plugin.ProcessorArgs) *types.Execution {
	url, ok := args.Config["webhook_url"]
	if !ok || url == "" {
		return args.Execution
	}

	payload := map[string]interface{}{
		"job_name":   args.Execution.JobName,
		"success":    args.Execution.Success,
		"node_name":  args.Execution.NodeName,
		"output":     string(args.Execution.Output),
	}
	
	if args.Execution.StartedAt != nil {
		payload["started_at"] = args.Execution.StartedAt.AsTime().Format(time.RFC3339)
	}
	if args.Execution.FinishedAt != nil {
		payload["finished_at"] = args.Execution.FinishedAt.AsTime().Format(time.RFC3339)
	}

	body, err := json.Marshal(payload)
	if err == nil {
		// Run asynchronously in a background goroutine so that the webhook call
		// doesn't block the main Dkron process execution pipeline.
		go func() {
			maxRetries := 3
			backoff := 500 * time.Millisecond
			client := &http.Client{Timeout: 3 * time.Second}

			for i := 0; i < maxRetries; i++ {
				req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
				if err != nil {
					break
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err == nil {
					_ = resp.Body.Close()
					// Check for successful HTTP status codes
					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						break
					}
				}

				// Exponential backoff sleep, except on last attempt
				if i < maxRetries-1 {
					time.Sleep(backoff)
					backoff *= 2
				}
			}
		}()
	}

	return args.Execution
}
