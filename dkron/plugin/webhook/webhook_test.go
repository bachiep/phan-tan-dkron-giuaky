package webhook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	types "github.com/distribworks/dkron/v4/gen/proto/types/v1"
	"github.com/distribworks/dkron/v4/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWebhookProcessorSendsPayloadAndHeaders(t *testing.T) {
	received := make(chan webhookRequest, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- webhookRequest{
			body:      body,
			content:   r.Header.Get("Content-Type"),
			auth:      r.Header.Get("X-Demo-Token"),
			userAgent: r.Header.Get("User-Agent"),
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	now := time.Now()
	exec := &types.Execution{
		JobName:    "test_webhook_job",
		Success:    false,
		NodeName:   "worker-1",
		Output:     []byte("command failed"),
		Group:      42,
		Attempt:    2,
		StartedAt:  timestamppb.New(now.Add(-2 * time.Second)),
		FinishedAt: timestamppb.New(now),
	}

	processor := &Webhook{}
	result := processor.Process(&plugin.ProcessorArgs{
		Execution: exec,
		Config: plugin.Config{
			"webhook_url": ts.URL,
			"headers":     `{"X-Demo-Token":"secret","User-Agent":"dkron-webhook-test"}`,
			"timeout":     "500ms",
			"backoff":     "1ms",
		},
	})

	assert.Equal(t, exec, result)

	req := waitForWebhookRequest(t, received)
	assert.Equal(t, "application/json", req.content)
	assert.Equal(t, "secret", req.auth)
	assert.Equal(t, "dkron-webhook-test", req.userAgent)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(req.body, &payload))

	assert.Equal(t, "dkron.execution.finished", payload["event"])
	assert.Equal(t, "test_webhook_job", payload["job_name"])
	assert.Equal(t, false, payload["success"])
	assert.Equal(t, "worker-1", payload["node_name"])
	assert.Equal(t, "command failed", payload["output"])
	assert.Equal(t, float64(42), payload["group"])
	assert.Equal(t, float64(2), payload["attempt"])
	assert.Equal(t, float64(2), payload["duration_sec"])
	assert.NotEmpty(t, payload["started_at"])
	assert.NotEmpty(t, payload["finished_at"])
}

func TestWebhookProcessorRetriesFailedResponses(t *testing.T) {
	var requestCount atomic.Int32
	done := make(chan struct{}, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		done <- struct{}{}
	}))
	defer ts.Close()

	processor := &Webhook{}
	processor.Process(&plugin.ProcessorArgs{
		Execution: &types.Execution{
			JobName: "retry_webhook_job",
			Success: false,
		},
		Config: plugin.Config{
			"webhook_url": ts.URL,
			"max_retries": "3",
			"backoff":     "1ms",
			"timeout":     "500ms",
		},
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook retry success")
	}

	assert.Equal(t, int32(3), requestCount.Load())
}

func TestWebhookProcessorOnlyOnFailureSkipsSuccessfulExecution(t *testing.T) {
	var requestCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	processor := &Webhook{}
	exec := &types.Execution{
		JobName: "success_job",
		Success: true,
	}

	result := processor.Process(&plugin.ProcessorArgs{
		Execution: exec,
		Config: plugin.Config{
			"webhook_url":     ts.URL,
			"only_on_failure": "true",
		},
	})

	assert.Equal(t, exec, result)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(0), requestCount.Load())
}

func TestWebhookProcessorNoConfig(t *testing.T) {
	processor := &Webhook{}
	exec := &types.Execution{
		JobName: "test_job",
	}
	args := &plugin.ProcessorArgs{
		Execution: exec,
		Config:    plugin.Config{},
	}

	result := processor.Process(args)
	assert.Equal(t, exec, result, "Expected unmodified execution when no config is provided")
}

type webhookRequest struct {
	body      []byte
	content   string
	auth      string
	userAgent string
}

func waitForWebhookRequest(t *testing.T, received <-chan webhookRequest) webhookRequest {
	t.Helper()

	select {
	case req := <-received:
		return req
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook request")
		return webhookRequest{}
	}
}
