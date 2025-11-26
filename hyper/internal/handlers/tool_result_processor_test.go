package handlers

import (
	"testing"
)

func TestProcessToolResult_SmallResult(t *testing.T) {
	processor := NewToolResultProcessor()

	result := "small result"
	processed := processor.ProcessToolResult("test_tool", result, 5000)

	if processed.WasSummarized {
		t.Error("Small result should not be summarized")
	}
}

func TestProcessToolResult_LargeResult(t *testing.T) {
	processor := NewToolResultProcessor()

	// Create a large result (> 3000 tokens)
	largeResult := ""
	for i := 0; i < 15000; i++ {
		largeResult += "x"
	}

	processed := processor.ProcessToolResult("test_tool", largeResult, 5000)

	if !processed.WasSummarized {
		t.Error("Large result should be summarized")
	}
}

func TestShouldUseSummary(t *testing.T) {
	estimator := NewTokenEstimator()

	tests := []struct {
		resultTokens    int
		remainingTokens int
		expected        bool
		name            string
	}{
		{1000, 5000, false, "Small result, plenty of context"},
		{3000, 5000, true, "Result > 50% of remaining"},
		{4600, 5000, true, "Would leave < 500 tokens"},
		{3500, 5000, true, "Result > 3000 tokens absolute"},
	}

	for _, tt := range tests {
		result := estimator.ShouldUseSummary(tt.resultTokens, tt.remainingTokens)
		if result != tt.expected {
			t.Errorf("%s: got %v, want %v", tt.name, result, tt.expected)
		}
	}
}

func TestEstimateTokens_String(t *testing.T) {
	estimator := NewTokenEstimator()

	// 4 chars per token heuristic
	result := estimator.EstimateTokens("hello world")
	expected := (11 + 3) / 4 // 11 chars, round up
	if result != expected {
		t.Errorf("String estimation: got %d, want %d", result, expected)
	}
}

func TestEstimateTokens_Array(t *testing.T) {
	estimator := NewTokenEstimator()

	arr := []interface{}{"hello", "world"}
	result := estimator.EstimateTokens(arr)

	// Each string estimated separately
	hello := (5 + 3) / 4
	world := (5 + 3) / 4
	expected := hello + world

	if result != expected {
		t.Errorf("Array estimation: got %d, want %d", result, expected)
	}
}

func TestContextTracker_CalculateRemaining(t *testing.T) {
	tracker := NewContextTracker()
	tracker.SystemPromptTokens = 1000
	tracker.MessagesTokens = 2000
	tracker.ToolResultsTokens = 1500

	tracker.CalculateRemaining(8000)

	expectedTotal := 1000 + 2000 + 1500
	expectedRemaining := 8000 - expectedTotal - 1000

	if tracker.TotalTokens != expectedTotal {
		t.Errorf("Total tokens: got %d, want %d", tracker.TotalTokens, expectedTotal)
	}

	if tracker.RemainingTokens != expectedRemaining {
		t.Errorf("Remaining tokens: got %d, want %d", tracker.RemainingTokens, expectedRemaining)
	}
}

func TestContextTracker_NegativeRemaining(t *testing.T) {
	tracker := NewContextTracker()
	tracker.SystemPromptTokens = 5000
	tracker.MessagesTokens = 3000
	tracker.ToolResultsTokens = 1000

	tracker.CalculateRemaining(8000)

	if tracker.RemainingTokens < 0 {
		t.Errorf("Remaining tokens should not be negative, got %d", tracker.RemainingTokens)
	}
}

func TestProcessMultipleToolResults(t *testing.T) {
	processor := NewToolResultProcessor()

	toolResults := map[string]interface{}{
		"tool1": "result1",
		"tool2": "result2",
	}

	processed := processor.ProcessMultipleToolResults(toolResults, 5000)

	if len(processed) != 2 {
		t.Errorf("Expected 2 processed results, got %d", len(processed))
	}

	for toolName, result := range processed {
		if result.OriginalResult == nil {
			t.Errorf("Tool %s has nil original result", toolName)
		}
	}
}
