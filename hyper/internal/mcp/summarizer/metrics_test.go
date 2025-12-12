package summarizer

import (
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestNewMetricsCollector(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	if mc == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}

	metrics := mc.GetMetrics()
	if metrics.TotalCount != 0 {
		t.Errorf("Expected initial TotalCount 0, got %d", metrics.TotalCount)
	}
}

func TestRecordSummarization(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.RecordSummarization("llm", 100, 50, false)

	metrics := mc.GetMetrics()
	if metrics.TotalCount != 1 {
		t.Errorf("Expected TotalCount 1, got %d", metrics.TotalCount)
	}

	if metrics.AICount != 1 {
		t.Errorf("Expected AICount 1, got %d", metrics.AICount)
	}

	if metrics.TokensUsed != 50 {
		t.Errorf("Expected TokensUsed 50, got %d", metrics.TokensUsed)
	}
}

func TestRecordSummarizationTypes(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.RecordSummarization("llm", 100, 50, false)
	mc.RecordSummarization("heuristic", 50, 30, false)
	mc.RecordSummarization("cached", 10, 0, true)

	metrics := mc.GetMetrics()

	if metrics.TotalCount != 3 {
		t.Errorf("Expected TotalCount 3, got %d", metrics.TotalCount)
	}

	if metrics.AICount != 1 {
		t.Errorf("Expected AICount 1, got %d", metrics.AICount)
	}

	if metrics.HeuristicCount != 1 {
		t.Errorf("Expected HeuristicCount 1, got %d", metrics.HeuristicCount)
	}

	if metrics.CachedCount != 1 {
		t.Errorf("Expected CachedCount 1, got %d", metrics.CachedCount)
	}
}

func TestRecordError(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.RecordError("timeout")
	mc.RecordError("api_error")

	metrics := mc.GetMetrics()
	if metrics.ErrorCount != 2 {
		t.Errorf("Expected ErrorCount 2, got %d", metrics.ErrorCount)
	}
}

func TestUpdateCacheStats(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.UpdateCacheStats(0.85, 500)

	metrics := mc.GetMetrics()
	if metrics.CacheHitRate != 0.85 {
		t.Errorf("Expected CacheHitRate 0.85, got %f", metrics.CacheHitRate)
	}

	if metrics.CacheSize != 500 {
		t.Errorf("Expected CacheSize 500, got %d", metrics.CacheSize)
	}
}

func TestResetMetrics(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.RecordSummarization("llm", 100, 50, false)
	mc.RecordError("timeout")

	metrics := mc.GetMetrics()
	if metrics.TotalCount != 1 {
		t.Errorf("Expected TotalCount 1 before reset, got %d", metrics.TotalCount)
	}

	mc.ResetMetrics()

	metrics = mc.GetMetrics()
	if metrics.TotalCount != 0 {
		t.Errorf("Expected TotalCount 0 after reset, got %d", metrics.TotalCount)
	}

	if metrics.ErrorCount != 0 {
		t.Errorf("Expected ErrorCount 0 after reset, got %d", metrics.ErrorCount)
	}
}

func TestGetSummaryBreakdown(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.RecordSummarization("llm", 100, 50, false)
	mc.RecordSummarization("llm", 100, 50, false)
	mc.RecordSummarization("heuristic", 50, 30, false)
	mc.RecordError("timeout")

	breakdown := mc.GetSummaryBreakdown()

	if breakdown["total"] != int64(3) {
		t.Errorf("Expected total 3, got %v", breakdown["total"])
	}

	if breakdown["ai"] != int64(2) {
		t.Errorf("Expected ai 2, got %v", breakdown["ai"])
	}

	if breakdown["heuristic"] != int64(1) {
		t.Errorf("Expected heuristic 1, got %v", breakdown["heuristic"])
	}

	if breakdown["errors"] != int64(1) {
		t.Errorf("Expected errors 1, got %v", breakdown["errors"])
	}
}

func TestGetPerformanceStats(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.RecordSummarization("llm", 100, 50, false)
	mc.RecordSummarization("llm", 200, 100, false)
	mc.UpdateCacheStats(0.75, 250)

	stats := mc.GetPerformanceStats()

	if stats["cacheHitRate"] != 0.75 {
		t.Errorf("Expected cacheHitRate 0.75, got %v", stats["cacheHitRate"])
	}

	if stats["cacheSize"] != 250 {
		t.Errorf("Expected cacheSize 250, got %v", stats["cacheSize"])
	}

	if stats["tokensUsed"] != int64(150) {
		t.Errorf("Expected tokensUsed 150, got %v", stats["tokensUsed"])
	}
}

func TestPrometheusMetrics(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.RecordSummarization("llm", 100, 50, false)
	mc.RecordSummarization("heuristic", 50, 30, false)
	mc.RecordError("timeout")
	mc.UpdateCacheStats(0.8, 300)

	prometheusMetrics := mc.PrometheusMetrics()

	// Check that it contains expected metrics
	if !strings.Contains(prometheusMetrics, "summarization_total_count") {
		t.Fatal("Expected summarization_total_count in Prometheus metrics")
	}

	if !strings.Contains(prometheusMetrics, "summarization_ai_count") {
		t.Fatal("Expected summarization_ai_count in Prometheus metrics")
	}

	if !strings.Contains(prometheusMetrics, "summarization_error_count") {
		t.Fatal("Expected summarization_error_count in Prometheus metrics")
	}

	if !strings.Contains(prometheusMetrics, "summarization_cache_hit_rate") {
		t.Fatal("Expected summarization_cache_hit_rate in Prometheus metrics")
	}
}

func TestLatencyStats(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	// Record multiple latencies
	latencies := []int64{50, 100, 150, 200, 250}
	for _, latency := range latencies {
		mc.RecordSummarization("llm", latency, 50, false)
	}

	metrics := mc.GetMetrics()

	// Average should be around 150
	if metrics.AverageLatency < 100 || metrics.AverageLatency > 200 {
		t.Errorf("Expected average latency around 150, got %f", metrics.AverageLatency)
	}

	// P95 and P99 should be set
	if metrics.P95Latency == 0 {
		t.Errorf("Expected P95Latency to be set")
	}

	if metrics.P99Latency == 0 {
		t.Errorf("Expected P99Latency to be set")
	}
}

func TestMetricsThreadSafety(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	done := make(chan bool, 10)

	// Concurrent record operations
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				if j%3 == 0 {
					mc.RecordSummarization("llm", 100, 50, false)
				} else if j%3 == 1 {
					mc.RecordError("timeout")
				} else {
					mc.UpdateCacheStats(0.8, 300)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := mc.GetMetrics()
	if metrics.TotalCount == 0 {
		t.Fatal("Expected metrics to be recorded")
	}
}

func TestLogMetrics(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.RecordSummarization("llm", 100, 50, false)
	mc.RecordError("timeout")

	// Should not panic
	mc.LogMetrics()
}

func TestMetricsWithZeroLatency(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	mc.RecordSummarization("cached", 0, 0, true)

	metrics := mc.GetMetrics()
	if metrics.CachedCount != 1 {
		t.Errorf("Expected CachedCount 1, got %d", metrics.CachedCount)
	}
}

func TestMetricsErrorRate(t *testing.T) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	// Record 10 successful summaries
	for i := 0; i < 10; i++ {
		mc.RecordSummarization("llm", 100, 50, false)
	}
	// Record 2 errors (errors are tracked separately, not in TotalCount)
	mc.RecordError("timeout")
	mc.RecordError("api_error")

	metrics := mc.GetMetrics()
	if metrics.TotalCount != 10 {
		t.Errorf("Expected TotalCount 10, got %d", metrics.TotalCount)
	}

	if metrics.ErrorCount != 2 {
		t.Errorf("Expected ErrorCount 2, got %d", metrics.ErrorCount)
	}
}

func BenchmarkRecordSummarization(b *testing.B) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordSummarization("llm", 100, 50, false)
	}
}

func BenchmarkGetMetrics(b *testing.B) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		mc.RecordSummarization("llm", 100, 50, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.GetMetrics()
	}
}

func BenchmarkPrometheusMetrics(b *testing.B) {
	logger := zap.NewNop()
	mc := NewMetricsCollector(logger)

	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		mc.RecordSummarization("llm", 100, 50, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.PrometheusMetrics()
	}
}
