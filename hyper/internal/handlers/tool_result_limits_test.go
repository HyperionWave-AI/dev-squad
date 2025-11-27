package handlers

import (
	"testing"
)

func TestDefaultToolResultLimits(t *testing.T) {
	limits := DefaultToolResultLimits()

	if limits.DefaultMaxTokens != 5000 {
		t.Errorf("DefaultMaxTokens: got %d, want 5000", limits.DefaultMaxTokens)
	}

	if limits.ContextPercentLimit != 0.25 {
		t.Errorf("ContextPercentLimit: got %f, want 0.25", limits.ContextPercentLimit)
	}

	if limits.MinTokenThreshold != 500 {
		t.Errorf("MinTokenThreshold: got %d, want 500", limits.MinTokenThreshold)
	}

	if len(limits.ToolSpecificLimits) == 0 {
		t.Error("ToolSpecificLimits should not be empty")
	}
}

func TestGetLimit_ToolSpecificLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	// code_index_search has specific limit of 3000
	limit := limits.GetLimit("code_index_search", 20000)

	if limit != 3000 {
		t.Errorf("code_index_search limit: got %d, want 3000", limit)
	}
}

func TestGetLimit_DefaultLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	// Unknown tool should use default
	limit := limits.GetLimit("unknown_tool", 20000)

	if limit != 5000 {
		t.Errorf("Unknown tool limit: got %d, want 5000", limit)
	}
}

func TestGetLimit_ContextPercentLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With 10000 remaining context, 25% = 2500
	// code_index_search has limit 3000, but context limit is 2500
	// Should return min(3000, 2500) = 2500
	limit := limits.GetLimit("code_index_search", 10000)

	if limit != 2500 {
		t.Errorf("Context percent limit: got %d, want 2500", limit)
	}
}

func TestGetLimit_DefaultWithContextLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With 10000 remaining context, 25% = 2500
	// Unknown tool has default 5000, but context limit is 2500
	// Should return min(5000, 2500) = 2500
	limit := limits.GetLimit("unknown_tool", 10000)

	if limit != 2500 {
		t.Errorf("Default with context limit: got %d, want 2500", limit)
	}
}

func TestGetLimit_ZeroRemainingContext(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With 0 remaining context, calculateContextLimit returns DefaultMaxTokens
	// So min(5000, 5000) = 5000
	limit := limits.GetLimit("unknown_tool", 0)

	if limit != 5000 {
		t.Errorf("Zero remaining context: got %d, want 5000", limit)
	}
}

func TestGetLimit_NegativeRemainingContext(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With negative remaining context, calculateContextLimit returns DefaultMaxTokens
	// So min(5000, 5000) = 5000
	limit := limits.GetLimit("unknown_tool", -1000)

	if limit != 5000 {
		t.Errorf("Negative remaining context: got %d, want 5000", limit)
	}
}

func TestCalculateContextLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	tests := []struct {
		remainingContext int
		expected         int
		name             string
	}{
		{10000, 2500, "10000 tokens * 0.25 = 2500"},
		{20000, 5000, "20000 tokens * 0.25 = 5000"},
		{4000, 1000, "4000 tokens * 0.25 = 1000"},
		{0, 5000, "0 tokens returns DefaultMaxTokens (5000)"},
		{-1000, 5000, "Negative context returns DefaultMaxTokens (5000)"},
	}

	for _, tt := range tests {
		result := limits.calculateContextLimit(tt.remainingContext)
		if result != tt.expected {
			t.Errorf("%s: got %d, want %d", tt.name, result, tt.expected)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a        int
		b        int
		expected int
		name     string
	}{
		{5, 10, 5, "5 < 10"},
		{10, 5, 5, "10 > 5"},
		{5, 5, 5, "5 == 5"},
		{0, 10, 0, "0 < 10"},
		{-5, 5, -5, "-5 < 5"},
		{-10, -5, -10, "-10 < -5"},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("%s: min(%d, %d) = %d, want %d", tt.name, tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestToolSpecificLimits_CodeTools(t *testing.T) {
	limits := DefaultToolResultLimits()

	tests := []struct {
		toolName string
		expected int
	}{
		{"code_index_search", 3000},
		{"code_index_get_full_content", 6000},
	}

	for _, tt := range tests {
		limit := limits.GetLimit(tt.toolName, 100000) // Large remaining context
		if limit != tt.expected {
			t.Errorf("%s: got %d, want %d", tt.toolName, limit, tt.expected)
		}
	}
}

func TestToolSpecificLimits_FileOperations(t *testing.T) {
	limits := DefaultToolResultLimits()

	tests := []struct {
		toolName string
		expected int
	}{
		{"read_file", 8000},
		{"file_read", 8000},
	}

	for _, tt := range tests {
		limit := limits.GetLimit(tt.toolName, 100000) // Large remaining context
		if limit != tt.expected {
			t.Errorf("%s: got %d, want %d", tt.toolName, limit, tt.expected)
		}
	}
}

func TestToolSpecificLimits_ShellCommands(t *testing.T) {
	limits := DefaultToolResultLimits()

	limit := limits.GetLimit("bash", 100000)
	if limit != 4000 {
		t.Errorf("bash: got %d, want 4000", limit)
	}
}

func TestToolSpecificLimits_KnowledgeTools(t *testing.T) {
	limits := DefaultToolResultLimits()

	tests := []struct {
		toolName string
		expected int
	}{
		{"knowledge_find", 2000},
		{"coordinator_list_agent_tasks", 2000},
	}

	for _, tt := range tests {
		limit := limits.GetLimit(tt.toolName, 100000) // Large remaining context
		if limit != tt.expected {
			t.Errorf("%s: got %d, want %d", tt.toolName, limit, tt.expected)
		}
	}
}

func TestGetLimit_AllToolsWithLowContext(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With only 1000 tokens remaining, context limit is 250
	// All tools should be capped at 250
	remainingContext := 1000

	tools := []string{
		"code_index_search",
		"code_index_get_full_content",
		"read_file",
		"bash",
		"knowledge_find",
		"unknown_tool",
	}

	for _, tool := range tools {
		limit := limits.GetLimit(tool, remainingContext)
		if limit != 250 {
			t.Errorf("%s with 1000 remaining: got %d, want 250", tool, limit)
		}
	}
}

func TestGetLimit_RealWorldScenarios(t *testing.T) {
	limits := DefaultToolResultLimits()

	tests := []struct {
		toolName         string
		remainingContext int
		expected         int
		name             string
	}{
		// Scenario 1: Plenty of context (25% = 25000, no cap)
		{"code_index_search", 100000, 3000, "Code search with 100k context"},
		{"read_file", 100000, 8000, "Read file with 100k context"},

		// Scenario 2: Limited context (25% = 5000)
		{"code_index_search", 20000, 3000, "Code search with 20k context (25% = 5000, tool limit 3000)"},
		{"read_file", 20000, 5000, "Read file with 20k context (25% = 5000, tool limit 8000, min = 5000)"},

		// Scenario 3: Very limited context (25% = 1000)
		{"code_index_search", 4000, 1000, "Code search with 4k context (25% = 1000, tool limit 3000, min = 1000)"},
		{"read_file", 4000, 1000, "Read file with 4k context (25% = 1000, tool limit 8000, min = 1000)"},

		// Scenario 4: Critical context (25% = 250)
		{"bash", 1000, 250, "Bash with 1k context (25% = 250, tool limit 4000, min = 250)"},
		{"unknown_tool", 1000, 250, "Unknown tool with 1k context (25% = 250, default 5000, min = 250)"},
	}

	for _, tt := range tests {
		result := limits.GetLimit(tt.toolName, tt.remainingContext)
		if result != tt.expected {
			t.Errorf("%s: got %d, want %d", tt.name, result, tt.expected)
		}
	}
}
