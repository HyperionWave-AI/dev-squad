package config

import (
	"os"
	"strconv"
)

// ProviderCapabilities defines AI provider limits for tool result management.
// This enables provider-agnostic tool result size management that works across
// ANY AI provider (Claude, GPT-4, Llama, etc.)
type ProviderCapabilities struct {
	// MaxContextTokens is the max tokens for provider (200k Claude, 128k GPT-4)
	MaxContextTokens int

	// CharsPerToken is the average characters per token (3.5 Claude, 4.0 GPT)
	CharsPerToken float64

	// ReservedForResponse is tokens to reserve for AI response output
	ReservedForResponse int

	// SafeToolResultBytes is a universal byte limit for tool results
	// This is a hard limit that works for ANY provider
	SafeToolResultBytes int
}

// DefaultProviderCapabilities returns safe defaults that work for any provider.
// These conservative defaults ensure compatibility across all AI providers
// without requiring explicit configuration.
func DefaultProviderCapabilities() *ProviderCapabilities {
	return &ProviderCapabilities{
		MaxContextTokens:    100000,    // Conservative default (works for most models)
		CharsPerToken:       3.5,       // Conservative (Claude uses ~3.5)
		ReservedForResponse: 8000,      // Reserve for AI output
		SafeToolResultBytes: 50 * 1024, // 50KB - safe for ANY provider
	}
}

// GetProviderCapabilities returns capabilities based on provider/model.
// Environment variables can override all other settings for custom configurations.
//
// Priority order (highest to lowest):
// 1. Environment variables (final override)
// 2. Model-specific defaults
// 3. Provider-specific defaults
// 4. Default values
//
// Supported environment variables:
//   - MAX_CONTEXT_TOKENS: Override max context window size
//   - SAFE_TOOL_RESULT_BYTES: Override max tool result size
func GetProviderCapabilities(provider, model string) *ProviderCapabilities {
	caps := DefaultProviderCapabilities()

	// Provider-specific defaults
	switch provider {
	case "anthropic":
		caps.MaxContextTokens = 200000
		caps.CharsPerToken = 3.5
	case "openai":
		caps.MaxContextTokens = 128000
		caps.CharsPerToken = 4.0
	case "google":
		caps.MaxContextTokens = 32000 // Gemini Pro
		caps.CharsPerToken = 4.0
	case "meta", "llama":
		caps.MaxContextTokens = 8000 // Llama 2 default
		caps.CharsPerToken = 4.0
	}

	// Model-specific overrides (for finer-grained control)
	switch model {
	case "gpt-4-turbo", "gpt-4-turbo-preview":
		caps.MaxContextTokens = 128000
	case "gpt-4":
		caps.MaxContextTokens = 8192
	case "gpt-4-32k":
		caps.MaxContextTokens = 32768
	case "gpt-3.5-turbo":
		caps.MaxContextTokens = 16385
	case "gpt-3.5-turbo-16k":
		caps.MaxContextTokens = 16385
	case "claude-3-opus", "claude-3-sonnet", "claude-3-haiku":
		caps.MaxContextTokens = 200000
	case "claude-2", "claude-2.1":
		caps.MaxContextTokens = 100000
	case "gemini-pro":
		caps.MaxContextTokens = 32000
	case "gemini-1.5-pro":
		caps.MaxContextTokens = 1000000 // 1M context
	case "llama-2-70b":
		caps.MaxContextTokens = 4096
	case "llama-3-70b":
		caps.MaxContextTokens = 8192
	}

	// Environment overrides (final, highest priority)
	// These allow runtime customization to override all other settings
	if maxCtx := os.Getenv("MAX_CONTEXT_TOKENS"); maxCtx != "" {
		if val, err := strconv.Atoi(maxCtx); err == nil && val > 0 {
			caps.MaxContextTokens = val
		}
	}
	if safeBytes := os.Getenv("SAFE_TOOL_RESULT_BYTES"); safeBytes != "" {
		if val, err := strconv.Atoi(safeBytes); err == nil && val > 0 {
			caps.SafeToolResultBytes = val
		}
	}

	return caps
}

// EstimateTokens converts bytes to estimated tokens using the provider's char-to-token ratio
func (p *ProviderCapabilities) EstimateTokens(bytes int) int {
	return int(float64(bytes) / p.CharsPerToken)
}

// EstimateTokensFromString estimates tokens from a string
func (p *ProviderCapabilities) EstimateTokensFromString(s string) int {
	return p.EstimateTokens(len(s))
}

// MaxSafeResultTokens returns max tokens for a single tool result
func (p *ProviderCapabilities) MaxSafeResultTokens() int {
	return p.EstimateTokens(p.SafeToolResultBytes)
}

// CalculateRemainingContext calculates remaining context after accounting for
// system prompt, messages, and reserved response tokens
func (p *ProviderCapabilities) CalculateRemainingContext(systemPromptBytes, messagesBytes int) int {
	systemPromptTokens := p.EstimateTokens(systemPromptBytes)
	messagesTokens := p.EstimateTokens(messagesBytes)

	remaining := p.MaxContextTokens - systemPromptTokens - messagesTokens - p.ReservedForResponse
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsResultTooLarge checks if a result exceeds the universal byte limit
func (p *ProviderCapabilities) IsResultTooLarge(resultBytes int) bool {
	return resultBytes > p.SafeToolResultBytes
}
