package handlers

import (
	"encoding/json"
)

// TokenEstimator provides token counting utilities
type TokenEstimator struct {
	charsPerToken int
}

// NewTokenEstimator creates a new token estimator
func NewTokenEstimator() *TokenEstimator {
	return &TokenEstimator{
		charsPerToken: 4, // Heuristic: ~4 characters = 1 token
	}
}

// EstimateTokens estimates token count for any output type
func (te *TokenEstimator) EstimateTokens(output interface{}) int {
	switch v := output.(type) {
	case string:
		return (len(v) + te.charsPerToken - 1) / te.charsPerToken
	case []interface{}:
		total := 0
		for _, item := range v {
			total += te.EstimateTokens(item)
		}
		return total
	case []string:
		total := 0
		for _, s := range v {
			total += (len(s) + te.charsPerToken - 1) / te.charsPerToken
		}
		return total
	case map[string]interface{}:
		total := 0
		for k, val := range v {
			total += (len(k) + te.charsPerToken - 1) / te.charsPerToken
			total += te.EstimateTokens(val)
		}
		return total
	default:
		if b, err := json.Marshal(v); err == nil {
			return (len(b) + te.charsPerToken - 1) / te.charsPerToken
		}
		return 0
	}
}

// ShouldUseSummary determines if summary should be used instead of full result
func (te *TokenEstimator) ShouldUseSummary(resultTokens int, remainingContextTokens int) bool {
	// Use summary if result > 50% of remaining context
	if resultTokens > remainingContextTokens/2 {
		return true
	}

	// Use summary if result would leave < 500 tokens remaining
	if remainingContextTokens-resultTokens < 500 {
		return true
	}

	// Use summary if result > 3000 tokens (absolute threshold)
	if resultTokens > 3000 {
		return true
	}

	return false
}
