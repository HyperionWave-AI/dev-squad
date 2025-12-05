package aiservice

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// tool_executor_circuit.go - Circuit Breaker for Tool Execution
//
// This file contains the circuit breaker logic that prevents infinite loops
// during tool execution. It tracks tool call patterns and stops execution
// when repeated failures or duplicates are detected.
//
// Features:
// - Per-tool thresholds (different limits for read vs write operations)
// - Model-specific thresholds (Claude gets more lenient limits)
// - Progressive warnings before circuit breaks
// - Consecutive failure tracking to detect stuck loops

// CircuitBreaker manages tool call loop detection and prevention.
// It tracks recent tool calls and their outcomes to detect infinite loops.
type CircuitBreaker struct {
	recentToolCalls     []string       // Recent tool call signatures
	consecutiveFailures int            // Count of consecutive failures with same signature
	lastFailedSignature string         // Signature of the last failed tool call
	thresholds          map[string]int // Per-tool max duplicate attempts
	isClaudeModel       bool           // Whether using Claude (affects thresholds)
	maxRecentCalls      int            // Maximum recent calls to track
}

// NewCircuitBreaker creates a new CircuitBreaker configured for the given model.
// Claude models get more lenient thresholds as they're better at adapting.
func NewCircuitBreaker(model, provider string) *CircuitBreaker {
	isClaudeModel := strings.Contains(strings.ToLower(model), "claude") ||
		strings.Contains(strings.ToLower(provider), "anthropic")

	cb := &CircuitBreaker{
		recentToolCalls: make([]string, 0, 10),
		isClaudeModel:   isClaudeModel,
		maxRecentCalls:  10,
	}

	if isClaudeModel {
		// Claude-optimized thresholds: More lenient to allow legitimate multi-file operations
		cb.thresholds = map[string]int{
			"read_file":         5, // Allow reading multiple files
			"write_file":        2, // Allow one retry for writes
			"list_directory":    4, // Allow exploring directories
			"bash":              5, // Allow command variations
			"code_index_search": 2, // Strict: one search + one retry max
			"create_agent_task": 4, // Allow retries for parameter refinement
			// Default for other tools: 6 attempts (see GetThreshold)
		}
		log.Printf("[Circuit Breaker] Using Claude-optimized thresholds (more lenient)")
	} else {
		// GPT thresholds: More conservative
		cb.thresholds = map[string]int{
			"read_file":         2, // Stop after 2 attempts (only 1 duplicate allowed)
			"write_file":        1, // Never allow duplicate writes
			"list_directory":    2, // Stop after 2 attempts
			"bash":              3, // Allow more for command variations
			"code_index_search": 3, // Allow query refinement
			// Default for other tools: 4 attempts (see GetThreshold)
		}
		log.Printf("[Circuit Breaker] Using GPT thresholds (conservative)")
	}

	return cb
}

// GetThreshold returns the circuit breaker threshold for a given tool.
func (cb *CircuitBreaker) GetThreshold(toolName string) int {
	if threshold, ok := cb.thresholds[toolName]; ok {
		return threshold
	}
	// Default threshold based on model
	if cb.isClaudeModel {
		return 6
	}
	return 4
}

// GenerateSignature creates a unique signature for a tool call with its arguments.
func GenerateSignature(toolName string, args map[string]interface{}) string {
	argsJSON, _ := json.Marshal(args)
	return fmt.Sprintf("%s(%s)", toolName, string(argsJSON))
}

// RecordToolCall records a tool call in the recent history.
// Returns the count of how many times this exact signature has been called.
func (cb *CircuitBreaker) RecordToolCall(signature string) int {
	cb.recentToolCalls = append(cb.recentToolCalls, signature)
	if len(cb.recentToolCalls) > cb.maxRecentCalls {
		cb.recentToolCalls = cb.recentToolCalls[1:]
	}

	// Count duplicates
	count := 0
	for _, sig := range cb.recentToolCalls {
		if sig == signature {
			count++
		}
	}
	return count
}

// RecordFailure records a failed tool execution.
// Returns true if this is a consecutive failure with the same signature.
func (cb *CircuitBreaker) RecordFailure(signature string) bool {
	if cb.lastFailedSignature == signature {
		cb.consecutiveFailures++
		return true
	}
	cb.consecutiveFailures = 1
	cb.lastFailedSignature = signature
	return false
}

// RecordSuccess resets the failure tracking after a successful execution.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.consecutiveFailures = 0
	cb.lastFailedSignature = ""
}

// GetConsecutiveFailures returns the current consecutive failure count.
func (cb *CircuitBreaker) GetConsecutiveFailures() int {
	return cb.consecutiveFailures
}

// ShouldBreakOnFailure returns true if consecutive failures have reached the limit.
func (cb *CircuitBreaker) ShouldBreakOnFailure() bool {
	return cb.consecutiveFailures >= 3
}

// ShouldBreak checks if the circuit breaker should trigger for this tool.
// Returns (should_break, count) where count is how many times it's been called.
func (cb *CircuitBreaker) ShouldBreak(toolName string, callCount int) bool {
	threshold := cb.GetThreshold(toolName)
	return callCount >= threshold
}

// GetWarning returns a progressive warning message based on duplicate count.
// Returns empty string if no warning is needed.
func (cb *CircuitBreaker) GetWarning(toolName string, count int) string {
	threshold := cb.GetThreshold(toolName)

	if count == 2 {
		// First duplicate - gentle warning
		return fmt.Sprintf("⚠️  WARNING: You already called '%s' with these exact arguments 1 time before. You should use the result from the previous call instead of repeating the same operation.", toolName)
	} else if count == 3 && threshold > 3 {
		// Second duplicate - stronger warning (only if threshold allows)
		return fmt.Sprintf("🔁 LOOP DETECTED: You called '%s' with identical arguments 2 times already. You are stuck in a loop! Use previous results or try a DIFFERENT approach - do NOT call this tool again with the same arguments.", toolName)
	}
	return ""
}

// GetConsecutiveFailureWarning returns a warning message for consecutive failures.
func (cb *CircuitBreaker) GetConsecutiveFailureWarning(toolName string, errorMsg string) string {
	return fmt.Sprintf("❌ CRITICAL: Tool '%s' has FAILED 3 TIMES IN A ROW with identical arguments.\n\n"+
		"Error: %s\n\n"+
		"🛑 This approach is NOT working. You MUST try something different:\n"+
		"   - If file not found: List the directory first to see what files actually exist\n"+
		"   - If path wrong: Try a different path or check your working directory\n"+
		"   - If tool incompatible: Use a completely different tool or approach\n\n"+
		"DO NOT call this tool with these arguments again!", toolName, errorMsg)
}

// GetCircuitBreakerError returns the error message when circuit breaker triggers.
func (cb *CircuitBreaker) GetCircuitBreakerError(toolName string, count int) string {
	return fmt.Sprintf("Circuit breaker triggered: tool '%s' called repeatedly (%d times) with identical arguments. The AI is stuck in an infinite loop and cannot complete this task.", toolName, count)
}

// IsClaudeModel returns whether the circuit breaker is configured for Claude.
func (cb *CircuitBreaker) IsClaudeModel() bool {
	return cb.isClaudeModel
}

// Reset clears all tracking state. Useful for testing or starting fresh.
func (cb *CircuitBreaker) Reset() {
	cb.recentToolCalls = make([]string, 0, 10)
	cb.consecutiveFailures = 0
	cb.lastFailedSignature = ""
}
