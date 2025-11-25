package storage

import (
	"fmt"
	"strings"
)

// ModelPricing defines the pricing for different AI models
type ModelPricing struct {
	Provider         string  // "claude", "openai", "groq", etc.
	Model            string  // Full model name
	InputCostPer1M   float64 // Cost per 1 million input tokens (USD)
	OutputCostPer1M  float64 // Cost per 1 million output tokens (USD)
	CacheCostPer1M   float64 // Cost per 1 million cache creation tokens (Claude only)
	CacheReadPer1M   float64 // Cost per 1 million cache read tokens (Claude only)
}

// PricingTable contains pricing for all supported models
var PricingTable = []ModelPricing{
	// Claude models
	{
		Provider:        "anthropic",
		Model:           "claude-3-5-sonnet-20241022",
		InputCostPer1M:  3.0,
		OutputCostPer1M: 15.0,
		CacheCostPer1M:  0.30,   // 90% discount on cache creation
		CacheReadPer1M:  0.30,   // 90% discount on cache reads
	},
	{
		Provider:        "anthropic",
		Model:           "claude-3-opus-20250219",
		InputCostPer1M:  15.0,
		OutputCostPer1M: 75.0,
		CacheCostPer1M:  1.50,
		CacheReadPer1M:  1.50,
	},
	{
		Provider:        "anthropic",
		Model:           "claude-3-haiku-20250307",
		InputCostPer1M:  0.80,
		OutputCostPer1M: 4.0,
		CacheCostPer1M:  0.08,
		CacheReadPer1M:  0.08,
	},
	// OpenAI models
	{
		Provider:        "openai",
		Model:           "gpt-4-turbo",
		InputCostPer1M:  10.0,
		OutputCostPer1M: 30.0,
	},
	{
		Provider:        "openai",
		Model:           "gpt-4",
		InputCostPer1M:  30.0,
		OutputCostPer1M: 60.0,
	},
	{
		Provider:        "openai",
		Model:           "gpt-3.5-turbo",
		InputCostPer1M:  0.50,
		OutputCostPer1M: 1.50,
	},
	{
		Provider:        "openai",
		Model:           "gpt-4o",
		InputCostPer1M:  5.0,
		OutputCostPer1M: 15.0,
	},
	{
		Provider:        "openai",
		Model:           "gpt-4o-mini",
		InputCostPer1M:  0.15,
		OutputCostPer1M: 0.60,
	},
	// Groq models (free tier)
	{
		Provider:        "groq",
		Model:           "mixtral-8x7b-32768",
		InputCostPer1M:  0.0,
		OutputCostPer1M: 0.0,
	},
	{
		Provider:        "groq",
		Model:           "llama-3.1-70b-versatile",
		InputCostPer1M:  0.0,
		OutputCostPer1M: 0.0,
	},
}

// GetModelPricing returns the pricing for a specific model
func GetModelPricing(provider string, model string) *ModelPricing {
	provider = strings.ToLower(provider)
	model = strings.ToLower(model)

	for i := range PricingTable {
		if strings.ToLower(PricingTable[i].Provider) == provider &&
			strings.ToLower(PricingTable[i].Model) == model {
			return &PricingTable[i]
		}
	}

	// Return default pricing if not found
	return &ModelPricing{
		Provider:        provider,
		Model:           model,
		InputCostPer1M:  0.0,
		OutputCostPer1M: 0.0,
	}
}

// CalculateCost calculates the cost for a given number of tokens
func CalculateCost(provider string, model string, inputTokens int, outputTokens int, cacheCreationTokens int, cacheReadTokens int) (float64, float64, float64) {
	pricing := GetModelPricing(provider, model)

	// Calculate input cost
	inputCost := float64(inputTokens) * (pricing.InputCostPer1M / 1_000_000)

	// Calculate output cost
	outputCost := float64(outputTokens) * (pricing.OutputCostPer1M / 1_000_000)

	// Calculate cache costs (Claude only)
	cacheCost := 0.0
	if cacheCreationTokens > 0 {
		cacheCost += float64(cacheCreationTokens) * (pricing.CacheCostPer1M / 1_000_000)
	}
	if cacheReadTokens > 0 {
		cacheCost += float64(cacheReadTokens) * (pricing.CacheReadPer1M / 1_000_000)
	}

	totalCost := inputCost + outputCost + cacheCost

	return inputCost, outputCost, totalCost
}

// FormatCost formats a cost value as a string with proper currency formatting
func FormatCost(cost float64) string {
	if cost < 0.0001 {
		return "$0.00"
	}
	return fmt.Sprintf("$%.4f", cost)
}

// FormatCostUSD formats a cost value as USD with 2 decimal places
func FormatCostUSD(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}
