package dkron

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AnalyticsResult defines the JSON structure of the new Analytics endpoint
type AnalyticsResult struct {
	TotalJobs            int                      `json:"total_jobs"`
	TotalExecutions      int                      `json:"total_executions"`
	SuccessfulExecutions int                      `json:"successful_executions"`
	FailedExecutions     int                      `json:"failed_executions"`
	SuccessRate          float64                  `json:"success_rate"`
	FailureRate          float64                  `json:"failure_rate"`
	AverageDurationSec   float64                  `json:"average_duration_sec"`
	MinDurationSec       float64                  `json:"min_duration_sec"`
	MaxDurationSec       float64                  `json:"max_duration_sec"`
	DurationSampleCount  int                      `json:"duration_sample_count"`
	LastExecutionAt      *time.Time               `json:"last_execution_at,omitempty"`
	ExecutionsByNode     map[string]NodeAnalytics `json:"executions_by_node"`
}

// NodeAnalytics summarizes execution outcomes for one Dkron node.
type NodeAnalytics struct {
	TotalExecutions      int     `json:"total_executions"`
	SuccessfulExecutions int     `json:"successful_executions"`
	FailedExecutions     int     `json:"failed_executions"`
	AverageDurationSec   float64 `json:"average_duration_sec"`
}

func (h *HTTPTransport) analyticsHandler(c *gin.Context) {
	jobs, err := h.agent.Store.GetJobs(c.Request.Context(), &JobOptions{})
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var execs []*Execution

	// Optimize: Avoid N+1 query bottleneck by fetching all execution records
	// in a single database prefix scan instead of querying each job individually.
	// Since we are using the local BuntDB Store, we can assert it and use its internal methods.
	if localStore, ok := h.agent.Store.(*Store); ok {
		kvs, err := localStore.list(executionsPrefix+":", false, &ExecutionOptions{})
		if err == nil {
			execs, _ = localStore.unmarshalExecutions(kvs, nil)
		}
	}

	// Fallback to N+1 query if using a non-standard Storage implementation
	if execs == nil {
		for _, job := range jobs {
			jExecs, err := h.agent.Store.GetExecutions(c.Request.Context(), job.Name, &ExecutionOptions{})
			if err == nil {
				execs = append(execs, jExecs...)
			}
		}
	}

	totalExecutions := len(execs)
	successCount := 0
	failedCount := 0
	totalDuration := time.Duration(0)
	minDuration := time.Duration(0)
	maxDuration := time.Duration(0)
	durationSampleCount := 0
	var lastExecutionAt *time.Time
	nodeStats := make(map[string]NodeAnalytics)
	nodeDurations := make(map[string]time.Duration)
	nodeDurationSamples := make(map[string]int)

	for _, ex := range execs {
		if ex.Success {
			successCount++
		} else {
			failedCount++
		}

		nodeName := ex.NodeName
		if nodeName == "" {
			nodeName = "unknown"
		}
		nodeStat := nodeStats[nodeName]
		nodeStat.TotalExecutions++
		if ex.Success {
			nodeStat.SuccessfulExecutions++
		} else {
			nodeStat.FailedExecutions++
		}
		nodeStats[nodeName] = nodeStat

		if !ex.FinishedAt.IsZero() {
			finishedAt := ex.FinishedAt
			if lastExecutionAt == nil || finishedAt.After(*lastExecutionAt) {
				lastExecutionAt = &finishedAt
			}
		}

		if !ex.FinishedAt.IsZero() && !ex.StartedAt.IsZero() && !ex.FinishedAt.Before(ex.StartedAt) {
			duration := ex.FinishedAt.Sub(ex.StartedAt)
			totalDuration += duration
			durationSampleCount++
			nodeDurations[nodeName] += duration
			nodeDurationSamples[nodeName]++

			if minDuration == 0 || duration < minDuration {
				minDuration = duration
			}
			if duration > maxDuration {
				maxDuration = duration
			}
		}
	}

	successRate := 0.0
	failureRate := 0.0
	avgDuration := 0.0

	if totalExecutions > 0 {
		successRate = float64(successCount) / float64(totalExecutions)
		failureRate = float64(failedCount) / float64(totalExecutions)
	}

	if durationSampleCount > 0 {
		avgDuration = totalDuration.Seconds() / float64(durationSampleCount)
	}

	for nodeName, nodeStat := range nodeStats {
		if samples := nodeDurationSamples[nodeName]; samples > 0 {
			nodeStat.AverageDurationSec = nodeDurations[nodeName].Seconds() / float64(samples)
			nodeStats[nodeName] = nodeStat
		}
	}

	result := AnalyticsResult{
		TotalJobs:            len(jobs),
		TotalExecutions:      totalExecutions,
		SuccessfulExecutions: successCount,
		FailedExecutions:     failedCount,
		SuccessRate:          successRate,
		FailureRate:          failureRate,
		AverageDurationSec:   avgDuration,
		MinDurationSec:       minDuration.Seconds(),
		MaxDurationSec:       maxDuration.Seconds(),
		DurationSampleCount:  durationSampleCount,
		LastExecutionAt:      lastExecutionAt,
		ExecutionsByNode:     nodeStats,
	}

	renderJSON(c, http.StatusOK, result)
}
