package aiservice

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ToolMetricsTracker tracks metrics for tool execution including cache hits/misses
type ToolMetricsTracker struct {
	metricsRecorder *MetricsRecorder
	logger          *zap.Logger
	mu              sync.Mutex

	// Per-request tracking
	requestMetrics map[string]*RequestMetrics
}

// RequestMetrics tracks metrics for a single request
type RequestMetrics struct {
	RequestID      string
	StartTime      time.Time
	ToolExecutions []ToolExecution
	CacheHits      int
	CacheMisses    int
}

// ToolExecution tracks a single tool execution
type ToolExecution struct {
	ToolName  string
	StartTime time.Time
	Duration  int64
	CacheHit  bool
	Error     string
}

// NewToolMetricsTracker creates a new tool metrics tracker
func NewToolMetricsTracker(metricsRecorder *MetricsRecorder, logger *zap.Logger) *ToolMetricsTracker {
	return &ToolMetricsTracker{
		metricsRecorder: metricsRecorder,
		logger:          logger,
		requestMetrics:  make(map[string]*RequestMetrics),
	}
}

// StartRequest initializes tracking for a new request
func (t *ToolMetricsTracker) StartRequest(requestID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.requestMetrics[requestID] = &RequestMetrics{
		RequestID:      requestID,
		StartTime:      time.Now(),
		ToolExecutions: make([]ToolExecution, 0),
	}
}

// RecordToolExecution records a tool execution with cache status
func (t *ToolMetricsTracker) RecordToolExecution(ctx context.Context, requestID string, toolName string, duration int64, cacheHit bool, errorMsg string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get or create request metrics
	metrics, exists := t.requestMetrics[requestID]
	if !exists {
		// Create new if doesn't exist
		metrics = &RequestMetrics{
			RequestID:      requestID,
			StartTime:      time.Now(),
			ToolExecutions: make([]ToolExecution, 0),
		}
		t.requestMetrics[requestID] = metrics
	}

	// Track cache hit/miss
	if cacheHit {
		metrics.CacheHits++
	} else {
		metrics.CacheMisses++
	}

	// Record tool execution
	execution := ToolExecution{
		ToolName:  toolName,
		StartTime: time.Now(),
		Duration:  duration,
		CacheHit:  cacheHit,
		Error:     errorMsg,
	}
	metrics.ToolExecutions = append(metrics.ToolExecutions, execution)

	// Record to metrics storage
	if t.metricsRecorder != nil {
		err := t.metricsRecorder.RecordToolMetrics(ctx, toolName, cacheHit, duration, requestID)
		if err != nil {
			t.logger.Warn("failed to record tool metrics",
				zap.String("toolName", toolName),
				zap.String("requestID", requestID),
				zap.Error(err))
		}
	}

	t.logger.Debug("recorded tool execution",
		zap.String("toolName", toolName),
		zap.String("requestID", requestID),
		zap.Bool("cacheHit", cacheHit),
		zap.Int64("duration", duration))

	return nil
}

// GetRequestMetrics returns metrics for a request
func (t *ToolMetricsTracker) GetRequestMetrics(requestID string) *RequestMetrics {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.requestMetrics[requestID]
}

// FinalizeRequest completes tracking for a request and returns summary
func (t *ToolMetricsTracker) FinalizeRequest(requestID string) map[string]interface{} {
	t.mu.Lock()
	defer t.mu.Unlock()

	metrics, exists := t.requestMetrics[requestID]
	if !exists {
		return nil
	}

	// Calculate cache hit rate
	totalOps := metrics.CacheHits + metrics.CacheMisses
	cacheHitRate := 0.0
	if totalOps > 0 {
		cacheHitRate = float64(metrics.CacheHits) / float64(totalOps)
	}

	// Calculate total duration
	totalDuration := int64(0)
	for _, exec := range metrics.ToolExecutions {
		totalDuration += exec.Duration
	}

	// Build summary
	summary := map[string]interface{}{
		"requestID":       requestID,
		"toolCount":       len(metrics.ToolExecutions),
		"cacheHits":       metrics.CacheHits,
		"cacheMisses":     metrics.CacheMisses,
		"cacheHitRate":    cacheHitRate,
		"totalDuration":   totalDuration,
		"averageDuration": totalDuration / int64(len(metrics.ToolExecutions)),
	}

	// Clean up
	delete(t.requestMetrics, requestID)

	return summary
}

// CacheTracker tracks cache statistics for tool result caching
type CacheTracker struct {
	mu         sync.Mutex
	hitCount   int
	missCount  int
	totalSize  int64
	maxSize    int64
}

// NewCacheTracker creates a new cache tracker
func NewCacheTracker(maxSize int64) *CacheTracker {
	return &CacheTracker{
		maxSize: maxSize,
	}
}

// RecordHit records a cache hit
func (c *CacheTracker) RecordHit(size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.hitCount++
	c.totalSize += size
}

// RecordMiss records a cache miss
func (c *CacheTracker) RecordMiss(size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.missCount++
	c.totalSize += size
}

// GetStats returns cache statistics
func (c *CacheTracker) GetStats() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	totalOps := c.hitCount + c.missCount
	hitRate := 0.0
	if totalOps > 0 {
		hitRate = float64(c.hitCount) / float64(totalOps)
	}

	return map[string]interface{}{
		"hitCount":   c.hitCount,
		"missCount":  c.missCount,
		"totalOps":   totalOps,
		"hitRate":    hitRate,
		"totalSize":  c.totalSize,
		"maxSize":    c.maxSize,
		"utilization": float64(c.totalSize) / float64(c.maxSize),
	}
}

// Reset resets cache statistics
func (c *CacheTracker) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.hitCount = 0
	c.missCount = 0
	c.totalSize = 0
}
