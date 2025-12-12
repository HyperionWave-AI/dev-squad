package handlers

// ToolResultLimits defines token limits for tool results
type ToolResultLimits struct {
	// DefaultMaxTokens is the default limit for any tool
	DefaultMaxTokens int

	// ToolSpecificLimits overrides for specific tools
	ToolSpecificLimits map[string]int

	// ContextPercentLimit is max percentage of remaining context a single result can use
	// e.g., 0.25 = max 25% of remaining context
	ContextPercentLimit float64

	// MinTokenThreshold is the minimum tokens before deflection kicks in
	// Results smaller than this are always allowed
	MinTokenThreshold int
}

// DefaultToolResultLimits returns sensible defaults optimized for provider-agnostic operation.
// These limits are conservative to work across ANY AI provider (Claude, GPT-4, Llama, etc.)
func DefaultToolResultLimits() *ToolResultLimits {
	return &ToolResultLimits{
		DefaultMaxTokens: 3000, // Lowered from 5000 for broader compatibility
		ToolSpecificLimits: map[string]int{
			// Code search tools - prefer summaries
			"code_index_search":           2000,
			"code_index_get_full_content": 4000,

			// File operations
			"read_file": 6000,
			"file_read": 6000,

			// Shell commands
			"bash": 3000,

			// Knowledge/search tools
			"knowledge_find":              2000,
			"knowledge_query":             2000,
			"coordinator_query_knowledge": 2000,

			// Coordinator list tools - LOW limits since they're paginated now
			"coordinator_list_agent_tasks": 1500,
			"coordinator_list_human_tasks": 1500,
			"list_agent_tasks":             1500,

			// Grep results
			"grep": 2000,

			// MCP tools (external) - conservative limits
			"execute_tool": 3000,
		},
		ContextPercentLimit: 0.20, // Lowered from 0.25 - max 20% of remaining context
		MinTokenThreshold:   300,  // Lowered from 500
	}
}

// GetLimit returns the token limit for a specific tool
func (l *ToolResultLimits) GetLimit(toolName string, remainingContext int) int {
	// Check tool-specific limit first
	if limit, exists := l.ToolSpecificLimits[toolName]; exists {
		return min(limit, l.calculateContextLimit(remainingContext))
	}

	// Fall back to default
	return min(l.DefaultMaxTokens, l.calculateContextLimit(remainingContext))
}

// calculateContextLimit returns max tokens based on remaining context
func (l *ToolResultLimits) calculateContextLimit(remainingContext int) int {
	if remainingContext <= 0 {
		return l.DefaultMaxTokens
	}
	return int(float64(remainingContext) * l.ContextPercentLimit)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
