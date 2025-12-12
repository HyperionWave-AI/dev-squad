package summarizer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ============================================================================
// HTTP Handlers for Summarizer API
// ============================================================================

// SummarizerHandlers contains all HTTP handlers for the summarizer API
type SummarizerHandlers struct {
	summarizer CodeSummarizer
	logger     *zap.Logger
	startTime  time.Time
}

// NewSummarizerHandlers creates a new SummarizerHandlers instance
func NewSummarizerHandlers(summarizer CodeSummarizer, logger *zap.Logger) *SummarizerHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SummarizerHandlers{
		summarizer: summarizer,
		logger:     logger,
		startTime:  time.Now(),
	}
}

// ============================================================================
// Middleware
// ============================================================================

// RequestIDMiddleware adds a unique request ID to each request
func (h *SummarizerHandlers) RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// LoggingMiddleware logs HTTP requests
func (h *SummarizerHandlers) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		duration := time.Since(startTime)

		requestID, _ := c.Get("requestID")
		h.logger.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String("requestID", fmt.Sprintf("%v", requestID)),
		)
	}
}

// ErrorHandlingMiddleware handles panics and errors
func (h *SummarizerHandlers) ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("requestID")
				h.logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("requestID", fmt.Sprintf("%v", requestID)),
				)
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:     "Internal server error",
					Code:      "INTERNAL_ERROR",
					Timestamp: time.Now(),
					RequestID: fmt.Sprintf("%v", requestID),
				})
			}
		}()
		c.Next()
	}
}

// ============================================================================
// Endpoint Handlers
// ============================================================================

// HandleSummarize handles POST /api/summarize requests
func (h *SummarizerHandlers) HandleSummarize(c *gin.Context) {
	requestID, _ := c.Get("requestID")
	requestIDStr := fmt.Sprintf("%v", requestID)

	var req SummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request",
			zap.Error(err),
			zap.String("requestID", requestIDStr),
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "Invalid request",
			Message:   err.Error(),
			Code:      "INVALID_REQUEST",
			Timestamp: time.Now(),
			RequestID: requestIDStr,
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.Warn("Request validation failed",
			zap.Error(err),
			zap.String("requestID", requestIDStr),
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "Validation error",
			Message:   err.Error(),
			Code:      "VALIDATION_ERROR",
			Timestamp: time.Now(),
			RequestID: requestIDStr,
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call summarizer
	summary, err := h.summarizer.Summarize(ctx, req.Code, req.Metadata)
	if err != nil {
		h.logger.Error("Summarization failed",
			zap.Error(err),
			zap.String("requestID", requestIDStr),
		)
		c.JSON(http.StatusInternalServerError, SummarizeResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
			RequestID: requestIDStr,
		})
		return
	}

	h.logger.Info("Summarization successful",
		zap.String("type", summary.Type),
		zap.Int("tokenCount", summary.TokenCount),
		zap.String("requestID", requestIDStr),
	)

	c.JSON(http.StatusOK, SummarizeResponse{
		Success:   true,
		Summary:   summary,
		Timestamp: time.Now(),
		RequestID: requestIDStr,
	})
}

// HandleBatchSummarize handles POST /api/summarize/batch requests
func (h *SummarizerHandlers) HandleBatchSummarize(c *gin.Context) {
	requestID, _ := c.Get("requestID")
	requestIDStr := fmt.Sprintf("%v", requestID)

	var req BatchSummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid batch request",
			zap.Error(err),
			zap.String("requestID", requestIDStr),
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "Invalid request",
			Message:   err.Error(),
			Code:      "INVALID_REQUEST",
			Timestamp: time.Now(),
			RequestID: requestIDStr,
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.Warn("Batch request validation failed",
			zap.Error(err),
			zap.String("requestID", requestIDStr),
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "Validation error",
			Message:   err.Error(),
			Code:      "VALIDATION_ERROR",
			Timestamp: time.Now(),
			RequestID: requestIDStr,
		})
		return
	}

	startTime := time.Now()
	results := make([]BatchSummarizeResult, 0, len(req.Items))
	errors := make([]BatchError, 0)
	totalTokens := 0
	successCount := 0

	// Process each item
	for _, item := range req.Items {
		itemStartTime := time.Now()

		// Create context with timeout for each item
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		summary, err := h.summarizer.Summarize(ctx, item.Code, item.Metadata)
		cancel()

		duration := time.Since(itemStartTime).Milliseconds()

		if err != nil {
			h.logger.Warn("Batch item summarization failed",
				zap.String("itemID", item.ID),
				zap.Error(err),
				zap.String("requestID", requestIDStr),
			)
			errors = append(errors, BatchError{
				ID:    item.ID,
				Error: err.Error(),
			})
			results = append(results, BatchSummarizeResult{
				ID:       item.ID,
				Success:  false,
				Error:    err.Error(),
				Duration: duration,
			})
		} else {
			successCount++
			totalTokens += summary.TokenCount
			results = append(results, BatchSummarizeResult{
				ID:       item.ID,
				Summary:  summary,
				Success:  true,
				Duration: duration,
			})
		}
	}

	totalDuration := time.Since(startTime).Milliseconds()
	avgDuration := float64(totalDuration) / float64(len(req.Items))

	h.logger.Info("Batch summarization completed",
		zap.Int("totalItems", len(req.Items)),
		zap.Int("successfulItems", successCount),
		zap.Int("failedItems", len(errors)),
		zap.Int("totalTokens", totalTokens),
		zap.Int64("totalDurationMs", totalDuration),
		zap.String("requestID", requestIDStr),
	)

	c.JSON(http.StatusOK, BatchSummarizeResponse{
		Success: len(errors) == 0,
		Results: results,
		Errors:  errors,
		Statistics: BatchStatistics{
			TotalItems:        len(req.Items),
			SuccessfulItems:   successCount,
			FailedItems:       len(errors),
			TotalTokens:       totalTokens,
			TotalDurationMs:   totalDuration,
			AverageDurationMs: avgDuration,
		},
		Timestamp: time.Now(),
		RequestID: requestIDStr,
	})
}

// HandleMetrics handles GET /api/metrics requests
func (h *SummarizerHandlers) HandleMetrics(c *gin.Context) {
	requestID, _ := c.Get("requestID")
	requestIDStr := fmt.Sprintf("%v", requestID)

	// Get metrics from summarizer if it supports it
	var metrics SummarizationMetrics
	var cache CacheStats
	var tokens TokenMetrics

	if llmSummarizer, ok := h.summarizer.(*LLMSummarizer); ok {
		metrics = llmSummarizer.GetMetrics()
		cache = llmSummarizer.GetCacheStats()
		tokens = llmSummarizer.GetTokenMetrics()
	}

	h.logger.Info("Metrics retrieved",
		zap.String("requestID", requestIDStr),
	)

	c.JSON(http.StatusOK, MetricsResponse{
		Success:   true,
		Metrics:   metrics,
		Cache:     cache,
		Tokens:    tokens,
		Timestamp: time.Now(),
	})
}

// HandleHealth handles GET /api/health requests
func (h *SummarizerHandlers) HandleHealth(c *gin.Context) {
	requestID, _ := c.Get("requestID")
	requestIDStr := fmt.Sprintf("%v", requestID)

	uptime := int64(time.Since(h.startTime).Seconds())

	checks := map[string]string{
		"summarizer": "ok",
	}

	status := "healthy"

	h.logger.Debug("Health check performed",
		zap.String("status", status),
		zap.Int64("uptime", uptime),
		zap.String("requestID", requestIDStr),
	)

	c.JSON(http.StatusOK, HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Uptime:    uptime,
		Checks:    checks,
	})
}

// HandleReadiness handles GET /api/ready requests
func (h *SummarizerHandlers) HandleReadiness(c *gin.Context) {
	requestID, _ := c.Get("requestID")
	requestIDStr := fmt.Sprintf("%v", requestID)

	// Check if summarizer is ready
	ready := h.summarizer != nil

	if !ready {
		h.logger.Warn("Readiness check failed - summarizer not initialized",
			zap.String("requestID", requestIDStr),
		)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":     false,
			"message":   "Summarizer not initialized",
			"timestamp": time.Now(),
		})
		return
	}

	h.logger.Debug("Readiness check passed",
		zap.String("requestID", requestIDStr),
	)

	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"timestamp": time.Now(),
	})
}

// HandleLiveness handles GET /api/live requests
func (h *SummarizerHandlers) HandleLiveness(c *gin.Context) {
	requestID, _ := c.Get("requestID")
	requestIDStr := fmt.Sprintf("%v", requestID)

	h.logger.Debug("Liveness check performed",
		zap.String("requestID", requestIDStr),
	)

	c.JSON(http.StatusOK, gin.H{
		"alive":     true,
		"timestamp": time.Now(),
	})
}
