package handlers

import (
	"log"
)

// ContextTracker tracks token usage across different parts of the context window
type ContextTracker struct {
	SystemPromptTokens int
	MessagesTokens     int
	ToolResultsTokens  int
	TotalTokens        int
	RemainingTokens    int
}

// CalculateRemaining calculates remaining tokens in the context window
func (ct *ContextTracker) CalculateRemaining(modelContextWindow int) {
	ct.TotalTokens = ct.SystemPromptTokens + ct.MessagesTokens + ct.ToolResultsTokens
	ct.RemainingTokens = modelContextWindow - ct.TotalTokens - 1000 // 1000 token safety margin
	if ct.RemainingTokens < 0 {
		ct.RemainingTokens = 0
	}
}

// LogUsage logs the current context usage
func (ct *ContextTracker) LogUsage(modelName string) {
	log.Printf(
		"[Context] Model: %s | System: %d | Messages: %d | Tools: %d | Total: %d | Remaining: %d",
		modelName,
		ct.SystemPromptTokens,
		ct.MessagesTokens,
		ct.ToolResultsTokens,
		ct.TotalTokens,
		ct.RemainingTokens,
	)
}

// NewContextTracker creates a new context tracker
func NewContextTracker() *ContextTracker {
	return &ContextTracker{
		SystemPromptTokens: 0,
		MessagesTokens:     0,
		ToolResultsTokens:  0,
		TotalTokens:        0,
		RemainingTokens:    0,
	}
}
