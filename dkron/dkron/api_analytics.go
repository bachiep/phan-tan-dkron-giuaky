package dkron

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AnalyticsResult defines the JSON structure of the new Analytics endpoint
type AnalyticsResult struct {
	TotalJobs          int     `json:"total_jobs"`
	TotalExecutions    int     `json:"total_executions"`
	SuccessRate        float64 `json:"success_rate"`
	AverageDurationSec float64 `json:"average_duration_sec"`
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
	totalDuration := time.Duration(0)

	for _, ex := range execs {
		if ex.Success {
			successCount++
		}
		if !ex.FinishedAt.IsZero() && !ex.StartedAt.IsZero() {
			totalDuration += ex.FinishedAt.Sub(ex.StartedAt)
		}
	}

	successRate := 0.0
	avgDuration := 0.0

	if totalExecutions > 0 {
		successRate = float64(successCount) / float64(totalExecutions)
		avgDuration = totalDuration.Seconds() / float64(totalExecutions)
	}

	result := AnalyticsResult{
		TotalJobs:          len(jobs),
		TotalExecutions:    totalExecutions,
		SuccessRate:        successRate,
		AverageDurationSec: avgDuration,
	}

	renderJSON(c, http.StatusOK, result)
}
