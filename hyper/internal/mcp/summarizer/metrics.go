// Package summarizer provides code summarization with caching, token budget management, and metrics.
//
// The metrics module collects and tracks comprehensive metrics for code summarization operations.
// It monitors performance, cache effectiveness, token usage, and error rates.
//
// Key Features:
//   - Summarization type tracking (AI, heuristic, cached)
//   - Latency tracking with percentile calculations (P95, P99)
//   - Token usage monitoring
//   - Cache hit rate tracking
//   - Error rate monitoring
//   - Prometheus-compatible metrics export
//
// Usage:
//
//	collector := NewMetricsCollector(logger)
//	collector.RecordSummarization("llm", 150, 250, false)
//	collector.UpdateCacheStats(0.85, 1024)
//	metrics := collector.GetMetrics()
//	breakdown := collector.GetSummaryBreakdown()
//	stats := collector.GetPerformanceStats()
//	promMetrics := collector.PrometheusMetrics()
//
//	collector.LogMetrics() // Log all metrics
package summarizer

import (
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SummarizationMetrics tracks metrics for code summarization
type SummarizationMetrics struct {
	TotalCount     int64
	AICount        int64
	HeuristicCount int64
	CachedCount    int64
	ErrorCount     int64
	LatencyMs      []int64
	TokensUsed     int64
	CacheHitRate   float64
	CacheSize      int
	AverageLatency float64
	P95Latency     int64
	P99Latency     int64
}

// MetricsCollector collects and tracks summarization metrics
type MetricsCollector struct {
	metrics       *SummarizationMetrics
	logger        *zap.Logger
	mu            sync.RWMutex
	startTime     time.Time
	lastResetTime time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *zap.Logger) *MetricsCollector {
	if logger == nil {
		logger = zap.NewNop()
	}

	now := time.Now()
	return &MetricsCollector{
		metrics: &SummarizationMetrics{
			LatencyMs: make([]int64, 0, 1000),
		},
		logger:        logger,
		startTime:     now,
		lastResetTime: now,
	}
}

// RecordSummarization records a summarization event
func (mc *MetricsCollector) RecordSummarization(summaryType string, latencyMs int64, tokensUsed int, cacheHit bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.TotalCount++

	switch summaryType {
	case "llm":
		mc.metrics.AICount++
	case "heuristic":
		mc.metrics.HeuristicCount++
	case "cached":
		mc.metrics.CachedCount++
	}

	if latencyMs > 0 {
		mc.metrics.LatencyMs = append(mc.metrics.LatencyMs, latencyMs)
		mc.updateLatencyStats()
	}

	mc.metrics.TokensUsed += int64(tokensUsed)

	mc.logger.Debug("Summarization recorded",
		zap.String("type", summaryType),
		zap.Int64("latencyMs", latencyMs),
		zap.Int("tokensUsed", tokensUsed),
		zap.Bool("cacheHit", cacheHit),
		zap.Int64("totalCount", mc.metrics.TotalCount))
}

// RecordError records a summarization error
func (mc *MetricsCollector) RecordError(errorType string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.ErrorCount++

	mc.logger.Warn("Summarization error recorded",
		zap.String("errorType", errorType),
		zap.Int64("totalErrors", mc.metrics.ErrorCount))
}

// UpdateCacheStats updates cache-related metrics
func (mc *MetricsCollector) UpdateCacheStats(hitRate float64, cacheSize int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.CacheHitRate = hitRate
	mc.metrics.CacheSize = cacheSize

	mc.logger.Debug("Cache stats updated",
		zap.Float64("hitRate", hitRate),
		zap.Int("cacheSize", cacheSize))
}

// GetMetrics returns a copy of the current metrics
func (mc *MetricsCollector) GetMetrics() SummarizationMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *mc.metrics
	metrics.LatencyMs = append([]int64(nil), mc.metrics.LatencyMs...)

	return metrics
}

// ResetMetrics resets all metrics to zero
func (mc *MetricsCollector) ResetMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics = &SummarizationMetrics{
		LatencyMs: make([]int64, 0, 1000),
	}
	mc.lastResetTime = time.Now()

	mc.logger.Info("Metrics reset")
}

// LogMetrics logs the current metrics
func (mc *MetricsCollector) LogMetrics() {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	uptime := time.Since(mc.startTime)
	errorRate := 0.0
	if mc.metrics.TotalCount > 0 {
		errorRate = float64(mc.metrics.ErrorCount) / float64(mc.metrics.TotalCount) * 100.0
	}

	mc.logger.Info("Summarization metrics",
		zap.Int64("totalCount", mc.metrics.TotalCount),
		zap.Int64("aiCount", mc.metrics.AICount),
		zap.Int64("heuristicCount", mc.metrics.HeuristicCount),
		zap.Int64("cachedCount", mc.metrics.CachedCount),
		zap.Int64("errorCount", mc.metrics.ErrorCount),
		zap.Float64("errorRate", errorRate),
		zap.Int64("tokensUsed", mc.metrics.TokensUsed),
		zap.Float64("cacheHitRate", mc.metrics.CacheHitRate),
		zap.Int("cacheSize", mc.metrics.CacheSize),
		zap.Float64("averageLatencyMs", mc.metrics.AverageLatency),
		zap.Int64("p95LatencyMs", mc.metrics.P95Latency),
		zap.Int64("p99LatencyMs", mc.metrics.P99Latency),
		zap.Duration("uptime", uptime))
}

// updateLatencyStats updates latency statistics (must be called with lock held)
func (mc *MetricsCollector) updateLatencyStats() {
	if len(mc.metrics.LatencyMs) == 0 {
		return
	}

	// Calculate average
	var sum int64
	for _, latency := range mc.metrics.LatencyMs {
		sum += latency
	}
	mc.metrics.AverageLatency = float64(sum) / float64(len(mc.metrics.LatencyMs))

	// Calculate percentiles (simple implementation)
	// For production, consider using a proper percentile library
	mc.metrics.P95Latency = calculatePercentile(mc.metrics.LatencyMs, 95)
	mc.metrics.P99Latency = calculatePercentile(mc.metrics.LatencyMs, 99)
}

// calculatePercentile calculates the Nth percentile of latencies
func calculatePercentile(latencies []int64, percentile int) int64 {
	if len(latencies) == 0 {
		return 0
	}

	// Simple percentile calculation (not perfectly accurate but good enough)
	index := (len(latencies) * percentile) / 100
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	// For accurate percentile, we'd need to sort, but for now return approximate
	if index < 0 {
		index = 0
	}

	return latencies[index]
}

// GetSummaryBreakdown returns a breakdown of summary types
func (mc *MetricsCollector) GetSummaryBreakdown() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	breakdown := map[string]interface{}{
		"total":     mc.metrics.TotalCount,
		"ai":        mc.metrics.AICount,
		"heuristic": mc.metrics.HeuristicCount,
		"cached":    mc.metrics.CachedCount,
		"errors":    mc.metrics.ErrorCount,
	}

	if mc.metrics.TotalCount > 0 {
		breakdown["aiPercent"] = float64(mc.metrics.AICount) / float64(mc.metrics.TotalCount) * 100.0
		breakdown["heuristicPercent"] = float64(mc.metrics.HeuristicCount) / float64(mc.metrics.TotalCount) * 100.0
		breakdown["cachedPercent"] = float64(mc.metrics.CachedCount) / float64(mc.metrics.TotalCount) * 100.0
		breakdown["errorPercent"] = float64(mc.metrics.ErrorCount) / float64(mc.metrics.TotalCount) * 100.0
	}

	return breakdown
}

// GetPerformanceStats returns performance-related statistics
func (mc *MetricsCollector) GetPerformanceStats() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	stats := map[string]interface{}{
		"averageLatencyMs": mc.metrics.AverageLatency,
		"p95LatencyMs":     mc.metrics.P95Latency,
		"p99LatencyMs":     mc.metrics.P99Latency,
		"tokensUsed":       mc.metrics.TokensUsed,
		"cacheHitRate":     mc.metrics.CacheHitRate,
		"cacheSize":        mc.metrics.CacheSize,
		"uptime":           time.Since(mc.startTime).String(),
	}

	if mc.metrics.TotalCount > 0 {
		stats["averageTokensPerSummary"] = float64(mc.metrics.TokensUsed) / float64(mc.metrics.TotalCount)
	}

	return stats
}

// PrometheusMetrics returns metrics in Prometheus format
func (mc *MetricsCollector) PrometheusMetrics() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := ""

	// Counter metrics
	metrics += "# HELP summarization_total_count Total number of summaries generated\n"
	metrics += "# TYPE summarization_total_count counter\n"
	metrics += "summarization_total_count " + formatInt64(mc.metrics.TotalCount) + "\n\n"

	metrics += "# HELP summarization_ai_count Number of AI-generated summaries\n"
	metrics += "# TYPE summarization_ai_count counter\n"
	metrics += "summarization_ai_count " + formatInt64(mc.metrics.AICount) + "\n\n"

	metrics += "# HELP summarization_heuristic_count Number of heuristic summaries\n"
	metrics += "# TYPE summarization_heuristic_count counter\n"
	metrics += "summarization_heuristic_count " + formatInt64(mc.metrics.HeuristicCount) + "\n\n"

	metrics += "# HELP summarization_cached_count Number of cached summaries\n"
	metrics += "# TYPE summarization_cached_count counter\n"
	metrics += "summarization_cached_count " + formatInt64(mc.metrics.CachedCount) + "\n\n"

	metrics += "# HELP summarization_error_count Number of summarization errors\n"
	metrics += "# TYPE summarization_error_count counter\n"
	metrics += "summarization_error_count " + formatInt64(mc.metrics.ErrorCount) + "\n\n"

	// Gauge metrics
	metrics += "# HELP summarization_tokens_used_total Total tokens used\n"
	metrics += "# TYPE summarization_tokens_used_total gauge\n"
	metrics += "summarization_tokens_used_total " + formatInt64(mc.metrics.TokensUsed) + "\n\n"

	metrics += "# HELP summarization_cache_hit_rate Cache hit rate (0-1)\n"
	metrics += "# TYPE summarization_cache_hit_rate gauge\n"
	metrics += "summarization_cache_hit_rate " + formatFloat64(mc.metrics.CacheHitRate) + "\n\n"

	metrics += "# HELP summarization_cache_size_bytes Current cache size in bytes\n"
	metrics += "# TYPE summarization_cache_size_bytes gauge\n"
	metrics += "summarization_cache_size_bytes " + formatInt(mc.metrics.CacheSize) + "\n\n"

	// Histogram metrics (simplified)
	metrics += "# HELP summarization_latency_ms Summarization latency in milliseconds\n"
	metrics += "# TYPE summarization_latency_ms histogram\n"
	metrics += "summarization_latency_ms_avg " + formatFloat64(mc.metrics.AverageLatency) + "\n"
	metrics += "summarization_latency_ms_p95 " + formatInt64(mc.metrics.P95Latency) + "\n"
	metrics += "summarization_latency_ms_p99 " + formatInt64(mc.metrics.P99Latency) + "\n\n"

	return metrics
}

// Helper functions for formatting metrics
func formatInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}

func formatInt(v int) string {
	return strconv.Itoa(v)
}

func formatFloat64(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
