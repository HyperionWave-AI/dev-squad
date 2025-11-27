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

// DefaultToolResultLimits returns sensible defaults
func DefaultToolResultLimits() *ToolResultLimits {
	return &ToolResultLimits{
		DefaultMaxTokens: 5000,
		ToolSpecificLimits: map[string]int{
			// Code search tools - prefer summaries
			"code_index_search":           3000,
			"code_index_get_full_content": 6000,

			// File operations
			"read_file": 8000,
			"file_read": 8000,

			// Shell commands can be verbose
			"bash": 4000,

			// Knowledge/search tools
			"knowledge_find":              2000,
			"coordinator_list_agent_tasks": 2000,

			// Grep results
			"grep": 3000,
		},
		ContextPercentLimit: 0.25, // Max 25% of remaining context
		MinTokenThreshold:   500,  // Don't deflect tiny results
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
