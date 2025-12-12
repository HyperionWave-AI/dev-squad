package aiservice

import (
	"strings"
)

// tool_executor_fallback.go - Model Limits and Fallback Utilities
//
// This file contains utilities for handling model-specific limits and fallback behavior.
// Rate limit detection is in rate_limit_handler.go.
// Claude system prompt is in langchain_service.go.

// ModelLimits contains context window and message limits for different models.
type ModelLimits struct {
	MaxContextSize int // Maximum context size in characters
	MaxMessages    int // Maximum messages to keep
}

// GetModelLimits returns the appropriate limits for a given model configuration.
func GetModelLimits(model, provider string) ModelLimits {
	if IsClaudeModel(model, provider) {
		return ModelLimits{
			MaxContextSize: 150000, // 150KB for Claude (≈37K tokens, leaves room for output)
			MaxMessages:    20,     // Keep more messages for Claude
		}
	}

	// GPT limits (default)
	return ModelLimits{
		MaxContextSize: 40000, // 40KB for GPT (≈10K tokens)
		MaxMessages:    6,     // Conservative for GPT
	}
}

// IsClaudeModel checks if the model/provider combination is a Claude model.
func IsClaudeModel(model, provider string) bool {
	return strings.Contains(strings.ToLower(model), "claude") ||
		strings.Contains(strings.ToLower(provider), "anthropic")
}
