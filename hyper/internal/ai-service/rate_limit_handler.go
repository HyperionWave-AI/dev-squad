package aiservice

import (
	"strings"
)

// rate_limit_handler.go
//
// This file contains rate limit detection logic for AI provider API calls.
// It checks error messages for common rate limit patterns across different providers.
//
// Supported providers:
// - Anthropic Claude: "rate limit", "429", "usage limit"
// - OpenAI GPT: "rate limit", "429", "too many requests"
// - Ollama: "402" (Payment Required used as rate limit signal)
// - Generic: "quota exceeded", "hourly limit"
//
// The detection is used by the main service to automatically switch to fallback
// models when rate limits are encountered, ensuring uninterrupted service.

// isRateLimitError checks if an error is a rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	// Check for common rate limit error patterns
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "402") || // Payment Required (Ollama rate limit)
		strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "usage limit") || // Matches "hourly usage limit", "daily usage limit", etc.
		strings.Contains(errStr, "hourly limit")
}
