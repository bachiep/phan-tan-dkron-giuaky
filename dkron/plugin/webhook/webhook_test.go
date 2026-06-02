package webhook

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/distribworks/dkron/v4/plugin"
	types "github.com/distribworks/dkron/v4/gen/proto/types/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWebhookProcessor(t *testing.T) {
	// 1. Setup mock HTTP server
	var receivedBody []byte
	var requestCount int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		body, _ := ioutil.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// 2. Setup Webhook processor
	processor := &Webhook{}

	// 3. Create Execution data
	now := time.Now()
	exec := &types.Execution{
		JobName:    "test_webhook_job",
		Success:    false, // Simulate a failed job
		NodeName:   "worker-1",
		Output:     []byte("command failed"),
		StartedAt:  timestamppb.New(now.Add(-time.Minute)),
		FinishedAt: timestamppb.New(now),
	}

	// 4. Configure processor with mock URL
	args := &plugin.ProcessorArgs{
		Execution: exec,
		Config: plugin.Config{
			"webhook_url": ts.URL,
		},
	}

	// 5. Call Process
	processor.Process(args)

	// Wait up to 1 second for the asynchronous background goroutine to execute
	for i := 0; i < 20; i++ {
		if requestCount > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 6. Assertions
	assert.Equal(t, 1, requestCount, "Expected exactly 1 HTTP request")
	require.NotEmpty(t, receivedBody, "Expected a non-empty request body")

	var payload map[string]interface{}
	err := json.Unmarshal(receivedBody, &payload)
	require.NoError(t, err)

	assert.Equal(t, "test_webhook_job", payload["job_name"])
	assert.Equal(t, false, payload["success"])
	assert.Equal(t, "worker-1", payload["node_name"])
	assert.Equal(t, "command failed", payload["output"])
	assert.NotEmpty(t, payload["started_at"])
	assert.NotEmpty(t, payload["finished_at"])
}

func TestWebhookProcessor_NoConfig(t *testing.T) {
	processor := &Webhook{}
	exec := &types.Execution{
		JobName: "test_job",
	}
	args := &plugin.ProcessorArgs{
		Execution: exec,
		Config:    plugin.Config{}, // Missing webhook_url
	}

	result := processor.Process(args)
	assert.Equal(t, exec, result, "Expected unmodified execution when no config is provided")
}
