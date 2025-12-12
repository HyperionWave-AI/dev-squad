package handlers

import (
	"testing"
)

func TestDefaultToolResultLimits(t *testing.T) {
	limits := DefaultToolResultLimits()

	// Updated values for provider-agnostic operation
	if limits.DefaultMaxTokens != 3000 {
		t.Errorf("DefaultMaxTokens: got %d, want 3000", limits.DefaultMaxTokens)
	}

	if limits.ContextPercentLimit != 0.20 {
		t.Errorf("ContextPercentLimit: got %f, want 0.20", limits.ContextPercentLimit)
	}

	if limits.MinTokenThreshold != 300 {
		t.Errorf("MinTokenThreshold: got %d, want 300", limits.MinTokenThreshold)
	}

	if len(limits.ToolSpecificLimits) == 0 {
		t.Error("ToolSpecificLimits should not be empty")
	}
}

func TestGetLimit_ToolSpecificLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	// code_index_search has specific limit of 2000 (lowered for provider-agnostic)
	limit := limits.GetLimit("code_index_search", 20000)

	if limit != 2000 {
		t.Errorf("code_index_search limit: got %d, want 2000", limit)
	}
}

func TestGetLimit_DefaultLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	// Unknown tool should use default (3000)
	limit := limits.GetLimit("unknown_tool", 20000)

	if limit != 3000 {
		t.Errorf("Unknown tool limit: got %d, want 3000", limit)
	}
}

func TestGetLimit_ContextPercentLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With 10000 remaining context, 20% = 2000
	// code_index_search has limit 2000, context limit is also 2000
	// Should return min(2000, 2000) = 2000
	limit := limits.GetLimit("code_index_search", 10000)

	if limit != 2000 {
		t.Errorf("Context percent limit: got %d, want 2000", limit)
	}
}

func TestGetLimit_DefaultWithContextLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With 10000 remaining context, 20% = 2000
	// Unknown tool has default 3000, but context limit is 2000
	// Should return min(3000, 2000) = 2000
	limit := limits.GetLimit("unknown_tool", 10000)

	if limit != 2000 {
		t.Errorf("Default with context limit: got %d, want 2000", limit)
	}
}

func TestGetLimit_ZeroRemainingContext(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With 0 remaining context, calculateContextLimit returns DefaultMaxTokens (3000)
	// So min(3000, 3000) = 3000
	limit := limits.GetLimit("unknown_tool", 0)

	if limit != 3000 {
		t.Errorf("Zero remaining context: got %d, want 3000", limit)
	}
}

func TestGetLimit_NegativeRemainingContext(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With negative remaining context, calculateContextLimit returns DefaultMaxTokens (3000)
	// So min(3000, 3000) = 3000
	limit := limits.GetLimit("unknown_tool", -1000)

	if limit != 3000 {
		t.Errorf("Negative remaining context: got %d, want 3000", limit)
	}
}

func TestCalculateContextLimit(t *testing.T) {
	limits := DefaultToolResultLimits()

	tests := []struct {
		remainingContext int
		expected         int
		name             string
	}{
		{10000, 2000, "10000 tokens * 0.20 = 2000"},
		{20000, 4000, "20000 tokens * 0.20 = 4000"},
		{4000, 800, "4000 tokens * 0.20 = 800"},
		{0, 3000, "0 tokens returns DefaultMaxTokens (3000)"},
		{-1000, 3000, "Negative context returns DefaultMaxTokens (3000)"},
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
		{"code_index_search", 2000},
		{"code_index_get_full_content", 4000},
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
		{"read_file", 6000},
		{"file_read", 6000},
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
	if limit != 3000 {
		t.Errorf("bash: got %d, want 3000", limit)
	}
}

func TestToolSpecificLimits_KnowledgeTools(t *testing.T) {
	limits := DefaultToolResultLimits()

	tests := []struct {
		toolName string
		expected int
	}{
		{"knowledge_find", 2000},
		{"coordinator_list_agent_tasks", 1500},
		{"coordinator_list_human_tasks", 1500},
		{"list_agent_tasks", 1500},
		{"knowledge_query", 2000},
		{"coordinator_query_knowledge", 2000},
	}

	for _, tt := range tests {
		limit := limits.GetLimit(tt.toolName, 100000) // Large remaining context
		if limit != tt.expected {
			t.Errorf("%s: got %d, want %d", tt.toolName, limit, tt.expected)
		}
	}
}

func TestToolSpecificLimits_MCPTools(t *testing.T) {
	limits := DefaultToolResultLimits()

	limit := limits.GetLimit("execute_tool", 100000)
	if limit != 3000 {
		t.Errorf("execute_tool: got %d, want 3000", limit)
	}
}

func TestGetLimit_AllToolsWithLowContext(t *testing.T) {
	limits := DefaultToolResultLimits()

	// With only 1000 tokens remaining, context limit is 200 (20%)
	// All tools should be capped at 200
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
		if limit != 200 {
			t.Errorf("%s with 1000 remaining: got %d, want 200", tool, limit)
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
		// Scenario 1: Plenty of context (20% = 20000, no cap)
		{"code_index_search", 100000, 2000, "Code search with 100k context"},
		{"read_file", 100000, 6000, "Read file with 100k context"},

		// Scenario 2: Limited context (20% = 4000)
		{"code_index_search", 20000, 2000, "Code search with 20k context (20% = 4000, tool limit 2000)"},
		{"read_file", 20000, 4000, "Read file with 20k context (20% = 4000, tool limit 6000, min = 4000)"},

		// Scenario 3: Very limited context (20% = 800)
		{"code_index_search", 4000, 800, "Code search with 4k context (20% = 800, tool limit 2000, min = 800)"},
		{"read_file", 4000, 800, "Read file with 4k context (20% = 800, tool limit 6000, min = 800)"},

		// Scenario 4: Critical context (20% = 200)
		{"bash", 1000, 200, "Bash with 1k context (20% = 200, tool limit 3000, min = 200)"},
		{"unknown_tool", 1000, 200, "Unknown tool with 1k context (20% = 200, default 3000, min = 200)"},
	}

	for _, tt := range tests {
		result := limits.GetLimit(tt.toolName, tt.remainingContext)
		if result != tt.expected {
			t.Errorf("%s: got %d, want %d", tt.name, result, tt.expected)
		}
	}
}

// TestProviderAgnosticLimits verifies that the limits are conservative enough
// to work across different AI providers (Claude, GPT-4, Llama, etc.)
func TestProviderAgnosticLimits(t *testing.T) {
	limits := DefaultToolResultLimits()

	// Verify conservative defaults
	if limits.DefaultMaxTokens > 5000 {
		t.Errorf("DefaultMaxTokens should be <= 5000 for provider compatibility, got %d", limits.DefaultMaxTokens)
	}

	if limits.ContextPercentLimit > 0.25 {
		t.Errorf("ContextPercentLimit should be <= 0.25 for provider compatibility, got %f", limits.ContextPercentLimit)
	}

	// Verify all tool-specific limits are reasonable
	for tool, limit := range limits.ToolSpecificLimits {
		if limit > 10000 {
			t.Errorf("Tool %s has limit %d which exceeds safe maximum of 10000", tool, limit)
		}
	}
}
