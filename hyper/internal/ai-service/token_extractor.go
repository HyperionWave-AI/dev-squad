package aiservice

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// TokenUsage represents token consumption for a single API call
type TokenUsage struct {
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	Provider         string    `json:"provider"` // "openai", "anthropic", "groq", etc.
	Model            string    `json:"model"`
	Timestamp        time.Time `json:"timestamp"`
}

// TokenUsageLogger handles logging and storage of token usage metrics
type TokenUsageLogger struct {
	usageHistory []TokenUsage
}

// NewTokenUsageLogger creates a new TokenUsageLogger
func NewTokenUsageLogger() *TokenUsageLogger {
	return &TokenUsageLogger{
		usageHistory: make([]TokenUsage, 0),
	}
}

// LogUsage records token usage and logs it
func (l *TokenUsageLogger) LogUsage(usage *TokenUsage) {
	if usage == nil {
		return
	}

	l.usageHistory = append(l.usageHistory, *usage)

	// Log to console
	fmt.Printf("[TOKEN USAGE] Provider: %s, Model: %s, Prompt: %d, Completion: %d, Total: %d\n",
		usage.Provider, usage.Model, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
}

// GetTotalUsage returns aggregated token usage statistics
func (l *TokenUsageLogger) GetTotalUsage() map[string]interface{} {
	totalPrompt := 0
	totalCompletion := 0
	totalTokens := 0
	providerStats := make(map[string]map[string]int)

	for _, usage := range l.usageHistory {
		totalPrompt += usage.PromptTokens
		totalCompletion += usage.CompletionTokens
		totalTokens += usage.TotalTokens

		if _, exists := providerStats[usage.Provider]; !exists {
			providerStats[usage.Provider] = make(map[string]int)
		}
		providerStats[usage.Provider]["prompt_tokens"] += usage.PromptTokens
		providerStats[usage.Provider]["completion_tokens"] += usage.CompletionTokens
		providerStats[usage.Provider]["total_tokens"] += usage.TotalTokens
		providerStats[usage.Provider]["call_count"]++
	}

	return map[string]interface{}{
		"total_prompt_tokens":     totalPrompt,
		"total_completion_tokens": totalCompletion,
		"total_tokens":            totalTokens,
		"total_calls":             len(l.usageHistory),
		"provider_stats":          providerStats,
	}
}

// GetUsageHistory returns the complete usage history
func (l *TokenUsageLogger) GetUsageHistory() []TokenUsage {
	return l.usageHistory
}

// TokenUsageExtractor extracts token counts from API responses
type TokenUsageExtractor struct {
	provider string
}

// NewTokenUsageExtractor creates a new token usage extractor
func NewTokenUsageExtractor(provider string) *TokenUsageExtractor {
	return &TokenUsageExtractor{
		provider: provider,
	}
}

// ExtractClaudeTokens extracts token usage from Claude API response
// Claude returns usage in the response body with input_tokens and output_tokens
func (e *TokenUsageExtractor) ExtractClaudeTokens(responseBody []byte) (inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int, err error) {
	var resp struct {
		Usage struct {
			InputTokens       int `json:"input_tokens"`
			OutputTokens      int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	log.Printf("[TokenExtractor] Claude tokens - Input: %d, Output: %d, CacheCreation: %d, CacheRead: %d",
		resp.Usage.InputTokens,
		resp.Usage.OutputTokens,
		resp.Usage.CacheCreationInputTokens,
		resp.Usage.CacheReadInputTokens)

	return resp.Usage.InputTokens,
		resp.Usage.OutputTokens,
		resp.Usage.CacheCreationInputTokens,
		resp.Usage.CacheReadInputTokens,
		nil
}

// ExtractOpenAITokens extracts token usage from OpenAI API response
// OpenAI returns usage in the response body with prompt_tokens and completion_tokens
func (e *TokenUsageExtractor) ExtractOpenAITokens(responseBody []byte) (inputTokens, outputTokens int, err error) {
	var resp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return 0, 0, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	log.Printf("[TokenExtractor] OpenAI tokens - Input: %d, Output: %d, Total: %d",
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		resp.Usage.TotalTokens)

	return resp.Usage.PromptTokens, resp.Usage.CompletionTokens, nil
}

// ExtractGroqTokens extracts token usage from Groq API response
// Groq uses the same format as OpenAI (prompt_tokens, completion_tokens)
func (e *TokenUsageExtractor) ExtractGroqTokens(responseBody []byte) (inputTokens, outputTokens int, err error) {
	var resp struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return 0, 0, fmt.Errorf("failed to parse Groq response: %w", err)
	}

	log.Printf("[TokenExtractor] Groq tokens - Input: %d, Output: %d, Total: %d",
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		resp.Usage.TotalTokens)

	return resp.Usage.PromptTokens, resp.Usage.CompletionTokens, nil
}

// ExtractTokensByProvider extracts tokens based on the provider type
func (e *TokenUsageExtractor) ExtractTokensByProvider(responseBody []byte) (inputTokens, outputTokens int, err error) {
	switch e.provider {
	case "anthropic":
		input, output, _, _, err := e.ExtractClaudeTokens(responseBody)
		return input, output, err
	case "openai":
		return e.ExtractOpenAITokens(responseBody)
	case "groq":
		return e.ExtractGroqTokens(responseBody)
	default:
		return 0, 0, fmt.Errorf("unsupported provider: %s", e.provider)
	}
}
