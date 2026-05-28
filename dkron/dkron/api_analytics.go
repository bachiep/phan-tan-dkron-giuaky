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

	totalExecutions := 0
	successCount := 0
	totalDuration := time.Duration(0)

	for _, job := range jobs {
		execs, err := h.agent.Store.GetExecutions(c.Request.Context(), job.Name, &ExecutionOptions{})
		if err == nil {
			for _, ex := range execs {
				totalExecutions++
				if ex.Success {
					successCount++
				}
				if !ex.FinishedAt.IsZero() && !ex.StartedAt.IsZero() {
					totalDuration += ex.FinishedAt.Sub(ex.StartedAt)
				}
			}
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
