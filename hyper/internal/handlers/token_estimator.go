package handlers

import (
	"encoding/json"
	"sync"
	"time"

	"hyper/internal/models"
)

// TokenEstimator provides token counting utilities
// TokenEstimator provides token counting utilities
type TokenEstimator struct {
	charsPerToken int
}

// cachedTokenCount represents a cached token count with metadata
type cachedTokenCount struct {
	tokens    int
	version   int64
	timestamp time.Time
}

// MessageTokenCache caches token counts per message to avoid recalculation
type MessageTokenCache struct {
	cache map[string]cachedTokenCount
	mu    sync.RWMutex
}

// NewMessageTokenCache creates a new thread-safe message token cache
func NewMessageTokenCache() *MessageTokenCache {
	return &MessageTokenCache{
		cache: make(map[string]cachedTokenCount),
	}
}

// Clear removes all cached entries
func (c *MessageTokenCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]cachedTokenCount)
}

// GetOrEstimate returns cached token count or estimates and caches the result
// It uses the message's cached TokenCount if available, otherwise estimates and caches
func (c *MessageTokenCache) GetOrEstimate(msg *models.ChatMessage, estimator *TokenEstimator) int {
	messageID := msg.ID.Hex()

	// Check cache first
	c.mu.RLock()
	if cached, ok := c.cache[messageID]; ok {
		c.mu.RUnlock()
		return cached.tokens
	}
	c.mu.RUnlock()

	// Estimate tokens for message
	tokens := estimator.EstimateMessageTokens(msg)

	// Store in cache
	c.mu.Lock()
	c.cache[messageID] = cachedTokenCount{tokens: tokens, timestamp: time.Now()}
	c.mu.Unlock()

	return tokens
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

// EstimateMessageTokens estimates the token count for a ChatMessage
// This includes the message content, role/metadata overhead, and any tool calls/results
func (te *TokenEstimator) EstimateMessageTokens(msg *models.ChatMessage) int {
	// Use cached value if available
	if msg.TokenCount > 0 {
		return msg.TokenCount
	}

	// Start with content tokens
	total := te.EstimateTokens(msg.Content)

	// Add overhead for role and metadata (~10 tokens)
	// This accounts for: role field, timestamp, message structure
	total += 10

	// Add tokens for tool call if present
	if msg.ToolCall != nil {
		// Tool name + structure overhead
		total += te.EstimateTokens(msg.ToolCall.Name) + 5
		// Tool arguments
		total += te.EstimateTokens(msg.ToolCall.Args)
		// Tool call ID
		total += te.EstimateTokens(msg.ToolCall.ID) + 5
	}

	// Add tokens for tool result if present
	if msg.ToolResult != nil {
		// Tool result structure overhead
		total += 10
		// Tool result output
		total += te.EstimateTokens(msg.ToolResult.Output)
		// Tool result ID and error (if present)
		total += te.EstimateTokens(msg.ToolResult.ID)
		if msg.ToolResult.Error != "" {
			total += te.EstimateTokens(msg.ToolResult.Error)
		}
	}

	return total
}

// EstimateTotalTokens estimates the total token count for a slice of ChatMessages
// This is useful for calculating context usage across multiple messages
func (te *TokenEstimator) EstimateTotalTokens(messages []models.ChatMessage) int {
	total := 0
	for i := range messages {
		total += te.EstimateMessageTokens(&messages[i])
	}
	return total
}
