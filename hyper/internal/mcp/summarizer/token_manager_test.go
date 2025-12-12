package summarizer

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewTokenManager(t *testing.T) {
	logger := zap.NewNop()
	tm := NewTokenManager(5000, 100, nil, logger)

	if tm == nil {
		t.Fatal("NewTokenManager returned nil")
	}

	if tm.TotalBudget() != 5000 {
		t.Errorf("Expected budget 5000, got %d", tm.TotalBudget())
	}

	if tm.UsedBudget() != 0 {
		t.Errorf("Expected used 0, got %d", tm.UsedBudget())
	}

	if tm.RemainingBudget() != 5000 {
		t.Errorf("Expected remaining 5000, got %d", tm.RemainingBudget())
	}
}

func TestCanSummarize(t *testing.T) {
	logger := zap.NewNop()
	tm := NewTokenManager(1000, 100, nil, logger)

	// Should be able to summarize small code
	code := "func test() {}"
	if !tm.CanSummarize(code) {
		t.Fatal("Expected to be able to summarize small code")
	}

	// Record usage
	tm.RecordUsage(900)

	// Should not be able to summarize large code
	largeCode := "func test() { " + string(make([]byte, 500)) + " }"
	if tm.CanSummarize(largeCode) {
		t.Fatal("Expected to not be able to summarize large code when budget exhausted")
	}
}

func TestRecordUsage(t *testing.T) {
	logger := zap.NewNop()
	tm := NewTokenManager(5000, 100, nil, logger)

	tm.RecordUsage(100)
	if tm.UsedBudget() != 100 {
		t.Errorf("Expected used 100, got %d", tm.UsedBudget())
	}

	if tm.RemainingBudget() != 4900 {
		t.Errorf("Expected remaining 4900, got %d", tm.RemainingBudget())
	}

	tm.RecordUsage(200)
	if tm.UsedBudget() != 300 {
		t.Errorf("Expected used 300, got %d", tm.UsedBudget())
	}

	if tm.RemainingBudget() != 4700 {
		t.Errorf("Expected remaining 4700, got %d", tm.RemainingBudget())
	}
}

func TestReset(t *testing.T) {
	logger := zap.NewNop()
	tm := NewTokenManager(5000, 100, nil, logger)

	tm.RecordUsage(1000)
	if tm.UsedBudget() != 1000 {
		t.Errorf("Expected used 1000, got %d", tm.UsedBudget())
	}

	tm.Reset()
	if tm.UsedBudget() != 0 {
		t.Errorf("Expected used 0 after reset, got %d", tm.UsedBudget())
	}

	if tm.RemainingBudget() != 5000 {
		t.Errorf("Expected remaining 5000 after reset, got %d", tm.RemainingBudget())
	}
}

func TestGetMetrics(t *testing.T) {
	logger := zap.NewNop()
	tm := NewTokenManager(5000, 100, nil, logger)

	tm.RecordUsage(100)
	tm.CanSummarize("small code")
	tm.CanSummarize(string(make([]byte, 10000))) // This should fail

	metrics := tm.GetMetrics()

	if metrics.TotalBudget != 5000 {
		t.Errorf("Expected total budget 5000, got %d", metrics.TotalBudget)
	}

	if metrics.CurrentUsage != 100 {
		t.Errorf("Expected current usage 100, got %d", metrics.CurrentUsage)
	}

	if metrics.RemainingBudget != 4900 {
		t.Errorf("Expected remaining 4900, got %d", metrics.RemainingBudget)
	}
}

func TestSimpleTokenEstimator(t *testing.T) {
	estimator := &SimpleTokenEstimator{}

	// Empty string
	if estimator.Estimate("") != 0 {
		t.Errorf("Expected 0 tokens for empty string")
	}

	// 4 characters = 1 token
	if estimator.Estimate("test") != 1 {
		t.Errorf("Expected 1 token for 4 characters")
	}

	// 8 characters = 2 tokens
	if estimator.Estimate("testtest") != 2 {
		t.Errorf("Expected 2 tokens for 8 characters")
	}

	// 5 characters = 2 tokens (rounded up)
	if estimator.Estimate("tests") != 2 {
		t.Errorf("Expected 2 tokens for 5 characters")
	}
}

func TestEstimateTokensForCode(t *testing.T) {
	code := "func test() { return 42; }"
	tokens := EstimateTokensForCode(code)

	if tokens <= 0 {
		t.Errorf("Expected positive token count, got %d", tokens)
	}

	// Should be roughly len(code) / 4
	expectedMin := len(code) / 5
	expectedMax := len(code) / 3

	if tokens < expectedMin || tokens > expectedMax {
		t.Errorf("Expected tokens between %d and %d, got %d", expectedMin, expectedMax, tokens)
	}
}

func TestEstimateTokensForSummary(t *testing.T) {
	code := "func test() { return 42; }"
	summaryTokens := EstimateTokensForSummary(code)
	codeTokens := EstimateTokensForCode(code)

	// Summary should be roughly 25% of code tokens
	if summaryTokens >= codeTokens {
		t.Errorf("Expected summary tokens (%d) to be less than code tokens (%d)", summaryTokens, codeTokens)
	}
}

func TestTokenManagerBudgetExhaustion(t *testing.T) {
	logger := zap.NewNop()
	tm := NewTokenManager(100, 50, nil, logger)

	// Use most of the budget
	tm.RecordUsage(90)

	// Should be able to summarize small code
	if !tm.CanSummarize("x") {
		t.Fatal("Expected to be able to summarize with remaining budget")
	}

	// Use remaining budget
	tm.RecordUsage(10)

	// Should not be able to summarize anything
	if tm.CanSummarize("x") {
		t.Fatal("Expected to not be able to summarize with exhausted budget")
	}
}

func TestTokenManagerMetricsBlockRate(t *testing.T) {
	logger := zap.NewNop()
	tm := NewTokenManager(100, 50, nil, logger)

	tm.RecordUsage(100)

	// Try to summarize (should fail)
	tm.CanSummarize("large code that exceeds budget")
	tm.CanSummarize("another large code")

	metrics := tm.GetMetrics()

	// Should have some blocked requests
	if metrics.RequestsBlocked == 0 {
		t.Fatal("Expected some blocked requests")
	}

	if metrics.BlockRate <= 0 {
		t.Errorf("Expected positive block rate, got %f", metrics.BlockRate)
	}
}

func TestTokenManagerThreadSafety(t *testing.T) {
	logger := zap.NewNop()
	tm := NewTokenManager(10000, 100, nil, logger)

	done := make(chan bool, 10)

	// Concurrent record usage
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				tm.RecordUsage(1)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have recorded all usage
	if tm.UsedBudget() != 1000 {
		t.Errorf("Expected used 1000, got %d", tm.UsedBudget())
	}
}

func BenchmarkCanSummarize(b *testing.B) {
	logger := zap.NewNop()
	tm := NewTokenManager(100000, 100, nil, logger)

	code := "func test() { return 42; }"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.CanSummarize(code)
	}
}

func BenchmarkRecordUsage(b *testing.B) {
	logger := zap.NewNop()
	tm := NewTokenManager(1000000, 100, nil, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.RecordUsage(1)
	}
}

func BenchmarkEstimateTokens(b *testing.B) {
	code := "func test() { return 42; }"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokensForCode(code)
	}
}
