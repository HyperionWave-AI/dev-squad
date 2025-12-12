package summarizer

import (
	"fmt"
	"time"
)

// ============================================================================
// HTTP Request Types
// ============================================================================

// SummarizeRequest represents a request to summarize code
type SummarizeRequest struct {
	Code     string       `json:"code" binding:"required"`
	Metadata CodeMetadata `json:"metadata,omitempty"`
	UserID   string       `json:"userId,omitempty"`
}

// BatchSummarizeRequest represents a request to summarize multiple code snippets
type BatchSummarizeRequest struct {
	Items  []BatchItem `json:"items" binding:"required,min=1,max=100"`
	UserID string      `json:"userId,omitempty"`
}

// BatchItem represents a single item in a batch summarization request
type BatchItem struct {
	ID       string       `json:"id" binding:"required"`
	Code     string       `json:"code" binding:"required"`
	Metadata CodeMetadata `json:"metadata,omitempty"`
}

// ============================================================================
// HTTP Response Types
// ============================================================================

// SummarizeResponse represents the response from a summarization request
type SummarizeResponse struct {
	Success   bool          `json:"success"`
	Summary   *CodeSummary  `json:"summary,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	RequestID string        `json:"requestId,omitempty"`
}

// BatchSummarizeResponse represents the response from a batch summarization request
type BatchSummarizeResponse struct {
	Success    bool                    `json:"success"`
	Results    []BatchSummarizeResult  `json:"results,omitempty"`
	Errors     []BatchError            `json:"errors,omitempty"`
	Statistics BatchStatistics         `json:"statistics"`
	Timestamp  time.Time               `json:"timestamp"`
	RequestID  string                  `json:"requestId,omitempty"`
}

// BatchSummarizeResult represents the result for a single item in batch processing
type BatchSummarizeResult struct {
	ID       string       `json:"id"`
	Summary  *CodeSummary `json:"summary"`
	Success  bool         `json:"success"`
	Error    string       `json:"error,omitempty"`
	Duration int64        `json:"durationMs"`
}

// BatchError represents an error for a batch item
type BatchError struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// BatchStatistics contains statistics about batch processing
type BatchStatistics struct {
	TotalItems      int   `json:"totalItems"`
	SuccessfulItems int   `json:"successfulItems"`
	FailedItems     int   `json:"failedItems"`
	TotalTokens     int   `json:"totalTokens"`
	TotalDurationMs int64 `json:"totalDurationMs"`
	AverageDurationMs float64 `json:"averageDurationMs"`
}

// MetricsResponse represents the response from the metrics endpoint
type MetricsResponse struct {
	Success   bool                  `json:"success"`
	Metrics   SummarizationMetrics  `json:"metrics,omitempty"`
	Cache     CacheStats            `json:"cache,omitempty"`
	Tokens    TokenMetrics          `json:"tokens,omitempty"`
	Timestamp time.Time             `json:"timestamp"`
}

// HealthResponse represents the response from the health check endpoint
type HealthResponse struct {
	Status    string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp time.Time         `json:"timestamp"`
	Uptime    int64             `json:"uptimeSeconds"`
	Version   string            `json:"version,omitempty"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// ErrorResponse represents a generic error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message,omitempty"`
	Code      string    `json:"code,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"requestId,omitempty"`
}

// ============================================================================
// Validation Helpers
// ============================================================================

// Validate validates the SummarizeRequest
func (r *SummarizeRequest) Validate() error {
	if r.Code == "" {
		return fmt.Errorf("code is required")
	}

	if len(r.Code) > 100000 { // 100KB limit
		return fmt.Errorf("code exceeds maximum size of 100KB")
	}

	return nil
}

// Validate validates the BatchSummarizeRequest
func (r *BatchSummarizeRequest) Validate() error {
	if len(r.Items) == 0 {
		return fmt.Errorf("items cannot be empty")
	}

	if len(r.Items) > 100 {
		return fmt.Errorf("batch size cannot exceed 100 items")
	}

	seenIDs := make(map[string]bool)
	for i, item := range r.Items {
		if item.ID == "" {
			return fmt.Errorf("item at index %d has empty id", i)
		}

		if seenIDs[item.ID] {
			return fmt.Errorf("duplicate item id: %s", item.ID)
		}
		seenIDs[item.ID] = true

		if item.Code == "" {
			return fmt.Errorf("item %s has empty code", item.ID)
		}

		if len(item.Code) > 100000 {
			return fmt.Errorf("item %s code exceeds maximum size of 100KB", item.ID)
		}
	}

	return nil
}
