package dkron

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIAnalytics(t *testing.T) {
	port := getFreePort(t)
	baseURL := fmt.Sprintf("http://localhost:%s/v1", port)
	dir, a := setupAPITest(t, port)
	defer os.RemoveAll(dir)
	defer a.Stop()

	ctx := context.Background()
	// Create Job
	job := &Job{
		Name:     "test_analytics_job",
		Schedule: "@every 1m",
		Executor: "shell",
		ExecutorConfig: map[string]string{
			"command": "date",
		},
	}
	err := a.Store.SetJob(ctx, job, false)
	require.NoError(t, err)

	now := time.Now().UTC()

	// Execution 1: Success, 2 seconds duration
	exec1 := &Execution{
		JobName:    "test_analytics_job",
		StartedAt:  now,
		FinishedAt: now.Add(2 * time.Second),
		Success:    true,
		NodeName:   "node1",
	}
	_, err = a.Store.SetExecution(ctx, exec1)
	require.NoError(t, err)

	// Execution 2: Failed, 4 seconds duration
	exec2 := &Execution{
		JobName:    "test_analytics_job",
		StartedAt:  now.Add(1 * time.Millisecond),
		FinishedAt: now.Add(1 * time.Millisecond).Add(4 * time.Second),
		Success:    false,
		NodeName:   "node2",
	}
	_, err = a.Store.SetExecution(ctx, exec2)
	require.NoError(t, err)

	// Test the API endpoint
	resp, err := http.Get(baseURL + "/analytics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	// Let's assert based on actual returned jobs if a system job exists
	// We expect at least 1 job (the one we created)
	assert.GreaterOrEqual(t, result["total_jobs"].(float64), float64(1))
	assert.Equal(t, float64(2), result["total_executions"])
	assert.Equal(t, float64(1), result["successful_executions"])
	assert.Equal(t, float64(1), result["failed_executions"])
	assert.Equal(t, float64(0.5), result["success_rate"])
	assert.Equal(t, float64(0.5), result["failure_rate"])
	assert.Equal(t, float64(3.0), result["average_duration_sec"]) // (2 + 4) / 2 = 3
	assert.Equal(t, float64(2.0), result["min_duration_sec"])
	assert.Equal(t, float64(4.0), result["max_duration_sec"])
	assert.Equal(t, float64(2), result["duration_sample_count"])
	assert.NotEmpty(t, result["last_execution_at"])

	executionsByNode, ok := result["executions_by_node"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, executionsByNode, "node1")
	require.Contains(t, executionsByNode, "node2")

	node1 := executionsByNode["node1"].(map[string]interface{})
	assert.Equal(t, float64(1), node1["total_executions"])
	assert.Equal(t, float64(1), node1["successful_executions"])
	assert.Equal(t, float64(0), node1["failed_executions"])
	assert.Equal(t, float64(2.0), node1["average_duration_sec"])

	node2 := executionsByNode["node2"].(map[string]interface{})
	assert.Equal(t, float64(1), node2["total_executions"])
	assert.Equal(t, float64(0), node2["successful_executions"])
	assert.Equal(t, float64(1), node2["failed_executions"])
	assert.Equal(t, float64(4.0), node2["average_duration_sec"])
}
