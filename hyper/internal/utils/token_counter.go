package utils

import (
	"encoding/json"
	"fmt"

	"hyper/internal/models"
)

// TokenCounter provides utilities for counting tokens in messages
// Uses character-based estimation: 1 token ≈ 4 characters
type TokenCounter struct {
	// TokensPerCharacter is the estimated ratio of tokens to characters
	// Standard OpenAI estimation: 1 token ≈ 4 characters
	TokensPerCharacter float64
}

// NewTokenCounter creates a new token counter with default settings
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		TokensPerCharacter: 0.25, // 1 token per 4 characters
	}
}

// CountTokens estimates the number of tokens in a string
// Uses character count * TokensPerCharacter ratio
func (tc *TokenCounter) CountTokens(text string) int {
	if text == "" {
		return 0
	}
	// Round up to ensure we don't underestimate
	tokens := int(float64(len(text)) * tc.TokensPerCharacter)
	if tokens == 0 && len(text) > 0 {
		tokens = 1 // Minimum 1 token for non-empty text
	}
	return tokens
}

// CountMessageTokens counts tokens in a single message
// Includes role, content, and metadata
func (tc *TokenCounter) CountMessageTokens(msg *models.ChatMessage) int {
	tokens := 0

	// Count role (typically 5-10 tokens)
	tokens += tc.CountTokens(msg.Role)

	// Count content
	tokens += tc.CountTokens(msg.Content)

	// Count tool call data if present
	if msg.ToolCall != nil {
		tokens += tc.CountTokens(msg.ToolCall.Name)
		tokens += tc.CountTokens(msg.ToolCall.ID)
		// Count args as JSON
		if argsJSON, err := json.Marshal(msg.ToolCall.Args); err == nil {
			tokens += tc.CountTokens(string(argsJSON))
		}
	}

	// Count tool result data if present
	if msg.ToolResult != nil {
		tokens += tc.CountTokens(msg.ToolResult.Name)
		tokens += tc.CountTokens(msg.ToolResult.ID)
		tokens += tc.CountTokens(msg.ToolResult.Error)
		// Count output as JSON
		if outputJSON, err := json.Marshal(msg.ToolResult.Output); err == nil {
			tokens += tc.CountTokens(string(outputJSON))
		}
	}

	// Add overhead for message structure (typically 4-5 tokens per message)
	tokens += 5

	return tokens
}

// CountSessionTokens counts total tokens in a session's messages
func (tc *TokenCounter) CountSessionTokens(messages []models.ChatMessage) int {
	total := 0
	for i := range messages {
		total += tc.CountMessageTokens(&messages[i])
	}
	return total
}

// EstimateTokensForContent estimates tokens for raw content string
// Useful for checking if a message will fit before saving
func (tc *TokenCounter) EstimateTokensForContent(content string) int {
	return tc.CountTokens(content)
}

// FormatTokenCount formats a token count as a human-readable string
func (tc *TokenCounter) FormatTokenCount(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	return fmt.Sprintf("%.1fK", float64(tokens)/1000)
}

// CalculatePercentage calculates what percentage of max tokens are used
func (tc *TokenCounter) CalculatePercentage(used, max int) float64 {
	if max == 0 {
		return 0
	}
	return (float64(used) / float64(max)) * 100
}
