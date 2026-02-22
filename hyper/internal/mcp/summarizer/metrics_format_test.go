package summarizer

import (
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestMetricFormatHelpers(t *testing.T) {
	if got := formatInt64(12345); got != "12345" {
		t.Fatalf("formatInt64 returned %q", got)
	}
	if got := formatInt(-7); got != "-7" {
		t.Fatalf("formatInt returned %q", got)
	}
	if got := formatFloat64(3.25); got != "3.25" {
		t.Fatalf("formatFloat64 returned %q", got)
	}
}

func TestPrometheusMetrics_ContainsNumericValues(t *testing.T) {
	mc := NewMetricsCollector(zap.NewNop())
	mc.RecordSummarization("llm", 120, 42, false)
	mc.UpdateCacheStats(0.5, 256)

	metrics := mc.PrometheusMetrics()

	for _, expected := range []string{
		"summarization_total_count 1",
		"summarization_ai_count 1",
		"summarization_tokens_used_total 42",
		"summarization_cache_hit_rate 0.5",
		"summarization_cache_size_bytes 256",
	} {
		if !strings.Contains(metrics, expected) {
			t.Fatalf("expected %q in metrics output:\n%s", expected, metrics)
		}
	}
}

func TestNewMetricsCollector_NilLoggerAndLatencyNoop(t *testing.T) {
	mc := NewMetricsCollector(nil)
	if mc == nil {
		t.Fatal("expected metrics collector")
	}

	// Covers early return branch in updateLatencyStats when no latencies exist.
	mc.updateLatencyStats()
}

func TestCalculatePercentile_Boundaries(t *testing.T) {
	if got := calculatePercentile(nil, 95); got != 0 {
		t.Fatalf("expected 0 for empty latencies, got %d", got)
	}

	latencies := []int64{10, 20, 30}
	if got := calculatePercentile(latencies, 100); got != 30 {
		t.Fatalf("expected last element for p100, got %d", got)
	}

	// Large negative percentile triggers the index < 0 clamp branch.
	if got := calculatePercentile(latencies, -200); got != 10 {
		t.Fatalf("expected first element for negative percentile, got %d", got)
	}
}
