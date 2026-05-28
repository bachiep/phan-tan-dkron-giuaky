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
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 5 * time.Second}
		_, _ = client.Do(req)
	}

	return args.Execution
}
