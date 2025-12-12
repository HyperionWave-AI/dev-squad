package aiservice

import (
	"testing"
	"time"
)

// TestNewInMemoryMetricsStore tests the creation of a new metrics store
func TestNewInMemoryMetricsStore(t *testing.T) {
	store := NewInMemoryMetricsStore(1000)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.maxMetricsPerType != 1000 {
		t.Errorf("expected maxMetricsPerType=1000, got %d", store.maxMetricsPerType)
	}

	// Test default max
	store2 := NewInMemoryMetricsStore(0)
	if store2.maxMetricsPerType != 10000 {
		t.Errorf("expected default maxMetricsPerType=10000, got %d", store2.maxMetricsPerType)
	}
}

// TestRecordProviderMetric tests recording provider metrics
func TestRecordProviderMetric(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	metric := &ProviderMetric{
		ID:               "test-1",
		Provider:         "openai",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		Cost:             0.003,
		DurationMs:       1000,
		Success:          true,
		Timestamp:        time.Now(),
	}

	err := store.RecordProviderMetric(metric)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify metric was stored
	metrics, err := store.GetProviderMetrics("openai", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].ID != "test-1" {
		t.Errorf("expected ID=test-1, got %s", metrics[0].ID)
	}
}

// TestRecordProviderMetricNil tests error handling for nil metric
func TestRecordProviderMetricNil(t *testing.T) {
	store := NewInMemoryMetricsStore(100)
	err := store.RecordProviderMetric(nil)
	if err == nil {
		t.Fatal("expected error for nil metric")
	}
}

// TestGetProviderMetrics tests retrieving provider metrics
func TestGetProviderMetrics(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	// Record multiple metrics
	for i := 0; i < 5; i++ {
		metric := &ProviderMetric{
			ID:               "test-" + string(rune(i)),
			Provider:         "openai",
			Model:            "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			Cost:             0.003,
			DurationMs:       1000,
			Success:          true,
			Timestamp:        time.Now(),
		}
		store.RecordProviderMetric(metric)
	}

	// Test retrieval
	metrics, err := store.GetProviderMetrics("openai", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 5 {
		t.Errorf("expected 5 metrics, got %d", len(metrics))
	}

	// Test limit
	metrics, err = store.GetProviderMetrics("openai", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics with limit, got %d", len(metrics))
	}

	// Test non-existent provider
	metrics, err = store.GetProviderMetrics("anthropic", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics for non-existent provider, got %d", len(metrics))
	}
}

// TestRecordToolMetric tests recording tool metrics
func TestRecordToolMetric(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	metric := &ToolMetric{
		ID:           "tool-1",
		ToolName:     "read_file",
		DurationMs:   500,
		Success:      true,
		CacheHit:     false,
		InputSize:    100,
		OutputSize:   1000,
		Timestamp:    time.Now(),
	}

	err := store.RecordToolMetric(metric)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify metric was stored
	metrics, err := store.GetToolMetrics("read_file", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(metrics))
	}
}

// TestRecordCacheHit tests recording cache hits
func TestRecordCacheHit(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	err := store.RecordCacheHit("read_file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metrics, err := store.GetCacheMetrics("read_file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.CacheHits != 1 {
		t.Errorf("expected 1 cache hit, got %d", metrics.CacheHits)
	}
	if metrics.TotalRequests != 1 {
		t.Errorf("expected 1 total request, got %d", metrics.TotalRequests)
	}
	if metrics.HitRate != 100.0 {
		t.Errorf("expected 100%% hit rate, got %f%%", metrics.HitRate)
	}
}

// TestRecordCacheMiss tests recording cache misses
func TestRecordCacheMiss(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	err := store.RecordCacheMiss("read_file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metrics, err := store.GetCacheMetrics("read_file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.CacheMisses != 1 {
		t.Errorf("expected 1 cache miss, got %d", metrics.CacheMisses)
	}
	if metrics.TotalRequests != 1 {
		t.Errorf("expected 1 total request, got %d", metrics.TotalRequests)
	}
	if metrics.HitRate != 0.0 {
		t.Errorf("expected 0%% hit rate, got %f%%", metrics.HitRate)
	}
}

// TestCacheHitRate tests cache hit rate calculation
func TestCacheHitRate(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	// Record 3 hits and 2 misses
	for i := 0; i < 3; i++ {
		store.RecordCacheHit("read_file")
	}
	for i := 0; i < 2; i++ {
		store.RecordCacheMiss("read_file")
	}

	metrics, err := store.GetCacheMetrics("read_file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedHitRate := 60.0 // 3 hits / 5 total = 60%
	if metrics.HitRate != expectedHitRate {
		t.Errorf("expected %f%% hit rate, got %f%%", expectedHitRate, metrics.HitRate)
	}
}

// TestGetDailyCostBreakdown tests daily cost aggregation
func TestGetDailyCostBreakdown(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Record metrics for today
	metric1 := &ProviderMetric{
		ID:               "test-1",
		Provider:         "openai",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		Cost:             0.003,
		DurationMs:       1000,
		Success:          true,
		Timestamp:        today.Add(2 * time.Hour),
	}

	metric2 := &ProviderMetric{
		ID:               "test-2",
		Provider:         "anthropic",
		Model:            "claude-3-opus",
		PromptTokens:     200,
		CompletionTokens: 100,
		TotalTokens:      300,
		Cost:             0.005,
		DurationMs:       2000,
		Success:          true,
		Timestamp:        today.Add(4 * time.Hour),
	}

	store.RecordProviderMetric(metric1)
	store.RecordProviderMetric(metric2)

	breakdown, err := store.GetDailyCostBreakdown(today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedTotalCost := 0.008
	if breakdown.TotalCost != expectedTotalCost {
		t.Errorf("expected total cost %f, got %f", expectedTotalCost, breakdown.TotalCost)
	}

	if breakdown.SuccessfulCalls != 2 {
		t.Errorf("expected 2 successful calls, got %d", breakdown.SuccessfulCalls)
	}

	if breakdown.TokensUsed != 450 {
		t.Errorf("expected 450 tokens, got %d", breakdown.TokensUsed)
	}
}

// TestGetCostByProvider tests cost aggregation by provider
func TestGetCostByProvider(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	now := time.Now()
	startDate := now.Add(-24 * time.Hour)
	endDate := now.Add(24 * time.Hour)

	// Record metrics for different providers
	metric1 := &ProviderMetric{
		ID:               "test-1",
		Provider:         "openai",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		Cost:             0.003,
		DurationMs:       1000,
		Success:          true,
		Timestamp:        now,
	}

	metric2 := &ProviderMetric{
		ID:               "test-2",
		Provider:         "anthropic",
		Model:            "claude-3-opus",
		PromptTokens:     200,
		CompletionTokens: 100,
		TotalTokens:      300,
		Cost:             0.005,
		DurationMs:       2000,
		Success:          true,
		Timestamp:        now,
	}

	store.RecordProviderMetric(metric1)
	store.RecordProviderMetric(metric2)

	costs, err := store.GetCostByProvider(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if costs["openai"] != 0.003 {
		t.Errorf("expected openai cost 0.003, got %f", costs["openai"])
	}

	if costs["anthropic"] != 0.005 {
		t.Errorf("expected anthropic cost 0.005, got %f", costs["anthropic"])
	}
}

// TestGetCostByModel tests cost aggregation by model
func TestGetCostByModel(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	now := time.Now()
	startDate := now.Add(-24 * time.Hour)
	endDate := now.Add(24 * time.Hour)

	// Record metrics for different models
	metric1 := &ProviderMetric{
		ID:               "test-1",
		Provider:         "openai",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		Cost:             0.003,
		DurationMs:       1000,
		Success:          true,
		Timestamp:        now,
	}

	metric2 := &ProviderMetric{
		ID:               "test-2",
		Provider:         "openai",
		Model:            "gpt-3.5-turbo",
		PromptTokens:     200,
		CompletionTokens: 100,
		TotalTokens:      300,
		Cost:             0.0005,
		DurationMs:       500,
		Success:          true,
		Timestamp:        now,
	}

	store.RecordProviderMetric(metric1)
	store.RecordProviderMetric(metric2)

	costs, err := store.GetCostByModel(startDate, endDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if costs["gpt-4"] != 0.003 {
		t.Errorf("expected gpt-4 cost 0.003, got %f", costs["gpt-4"])
	}

	if costs["gpt-3.5-turbo"] != 0.0005 {
		t.Errorf("expected gpt-3.5-turbo cost 0.0005, got %f", costs["gpt-3.5-turbo"])
	}
}

// TestGetAverageCostPerCall tests average cost calculation
func TestGetAverageCostPerCall(t *testing.T) {
	store := NewInMemoryMetricsStore(100)

	// Record 3 metrics with different costs
	for i := 0; i < 3; i++ {
		metric := &ProviderMetric{
			ID:               "test-" + string(rune(i)),
			Provider:         "openai",
			Model:            "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			Cost:             0.003,
			DurationMs:       1000,
			Success:          true,
			Timestamp:        time.Now(),
		}
		store.RecordProviderMetric(metric)
	}

	avgCost, err := store.GetAverageCostPerCall("openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedAvg := 0.003
	if avgCost != expectedAvg {
		// Allow for floating point precision errors
		diff := avgCost - expectedAvg
		if diff < -0.0001 || diff > 0.0001 {
			t.Errorf("expected average cost %f, got %f", expectedAvg, avgCost)
		}
	}
}
func TestCalculateOpenAICost(t *testing.T) {
	tests := []struct {
		model            string
		promptTokens     int
		completionTokens int
		expectedCost     float64
	}{
		{"gpt-4", 1000, 1000, 0.09},                    // (1000/1000)*0.03 + (1000/1000)*0.06 = 0.09
		{"gpt-4-turbo", 1000, 1000, 0.04},              // (1000/1000)*0.01 + (1000/1000)*0.03 = 0.04
		{"gpt-3.5-turbo", 1000, 1000, 0.002},           // (1000/1000)*0.0005 + (1000/1000)*0.0015 = 0.002
		{"unknown-model", 1000, 1000, 0.002},           // Default to gpt-3.5 pricing
	}

	for _, test := range tests {
		cost := CalculateOpenAICost(test.model, test.promptTokens, test.completionTokens)
		if cost != test.expectedCost {
			t.Errorf("model=%s: expected cost %f, got %f", test.model, test.expectedCost, cost)
		}
	}
}

// TestCalculateAnthropicCost tests Anthropic cost calculation
func TestCalculateAnthropicCost(t *testing.T) {
	tests := []struct {
		model        string
		inputTokens  int
		outputTokens int
		expectedCost float64
	}{
		{"claude-3-opus", 1000000, 1000000, 90.0},           // (1000000/1000000)*15 + (1000000/1000000)*75 = 90
		{"claude-3-sonnet", 1000000, 1000000, 18.0},         // (1000000/1000000)*3 + (1000000/1000000)*15 = 18
		{"claude-3-haiku", 1000000, 1000000, 1.5},           // (1000000/1000000)*0.25 + (1000000/1000000)*1.25 = 1.5
		{"unknown-model", 1000000, 1000000, 18.0},           // Default to Sonnet pricing
	}

	for _, test := range tests {
		cost := CalculateAnthropicCost(test.model, test.inputTokens, test.outputTokens)
		if cost != test.expectedCost {
			t.Errorf("model=%s: expected cost %f, got %f", test.model, test.expectedCost, cost)
		}
	}
}

// TestMetricsMaxCapacity tests that metrics store respects max capacity
func TestMetricsMaxCapacity(t *testing.T) {
	store := NewInMemoryMetricsStore(5) // Max 5 metrics

	// Record 10 metrics
	for i := 0; i < 10; i++ {
		metric := &ProviderMetric{
			ID:               "test-" + string(rune(i)),
			Provider:         "openai",
			Model:            "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			Cost:             0.003,
			DurationMs:       1000,
			Success:          true,
			Timestamp:        time.Now(),
		}
		store.RecordProviderMetric(metric)
	}

	// Should only have last 5
	metrics, _ := store.GetProviderMetrics("openai", 100)
	if len(metrics) != 5 {
		t.Errorf("expected 5 metrics (max capacity), got %d", len(metrics))
	}
}

// TestConcurrentMetricsRecording tests thread-safe metrics recording
func TestConcurrentMetricsRecording(t *testing.T) {
	store := NewInMemoryMetricsStore(1000)

	// Record metrics concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				metric := &ProviderMetric{
					ID:               "test-" + string(rune(id*10+j)),
					Provider:         "openai",
					Model:            "gpt-4",
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
					Cost:             0.003,
					DurationMs:       1000,
					Success:          true,
					Timestamp:        time.Now(),
				}
				store.RecordProviderMetric(metric)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all metrics were recorded
	metrics, _ := store.GetProviderMetrics("openai", 1000)
	if len(metrics) != 100 {
		t.Errorf("expected 100 metrics, got %d", len(metrics))
	}
}
