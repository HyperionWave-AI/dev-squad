package aiservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// ContextKey type for context keys
type contextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "requestID"
	// IdentityKey is the context key for user identity
	IdentityKey contextKey = "identity"
)

// Identity represents user identity extracted from JWT
type Identity struct {
	Type      string `json:"type"`      // "human", "agent", or "service"
	Name      string `json:"name"`      // User or agent name
	ID        string `json:"id"`        // User ID
	Email     string `json:"email"`     // User email
	CompanyID string `json:"companyId"` // Company ID for multi-tenancy
}

// StreamEventType defines the type of streaming event
type StreamEventType string

const (
	StreamEventToken      StreamEventType = "token"       // Text token
	StreamEventToolCall   StreamEventType = "tool_call"   // Tool invocation request
	StreamEventToolResult StreamEventType = "tool_result" // Tool execution result
	StreamEventError      StreamEventType = "error"       // Error during processing
)

// StreamEvent represents a streaming event (token, tool call, or tool result)
type StreamEvent struct {
	Type       StreamEventType `json:"type"`
	Content    string          `json:"content,omitempty"`    // For token events
	ToolCall   *ToolCall       `json:"toolCall,omitempty"`   // For tool_call events
	ToolResult *ToolResult     `json:"toolResult,omitempty"` // For tool_result events
	Error      string          `json:"error,omitempty"`      // For error events
}

// ChatService manages AI chat operations with provider abstraction
type ChatService struct {
	provider     ChatProvider
	config       *AIConfig
	toolRegistry *ToolRegistry
}

// ToolResultCache caches tool execution results by signature to prevent duplicate executions
// Helps reduce circuit breaker hits by reusing results when the same tool is called with identical arguments
type ToolResultCache struct {
	cache map[string]*ToolResult
	mu    sync.RWMutex
}

// NewToolResultCache creates a new tool result cache
func NewToolResultCache() *ToolResultCache {
	return &ToolResultCache{
		cache: make(map[string]*ToolResult),
	}
}

// Get retrieves a cached tool result if it exists
func (c *ToolResultCache) Get(signature string) (*ToolResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result, found := c.cache[signature]
	return result, found
}

// Set stores a tool result in the cache
func (c *ToolResultCache) Set(signature string, result *ToolResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Create a deep copy to avoid mutation issues
	cachedResult := &ToolResult{
		Name:       result.Name,
		Output:     result.Output,
		Error:      result.Error,
		DurationMs: result.DurationMs,
	}
	c.cache[signature] = cachedResult
}

// NewChatService creates a new ChatService with the given configuration
// Creates an empty tool registry - use RegisterTool() or GetToolRegistry() to add tools
func NewChatService(config *AIConfig) (*ChatService, error) {
	provider, err := NewChatProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Initialize empty tool registry
	// Tools should be registered after creation using RegisterTool() or GetToolRegistry()
	toolRegistry := NewToolRegistry()

	return &ChatService{
		provider:     provider,
		config:       config,
		toolRegistry: toolRegistry,
	}, nil
}

// NewChatServiceWithTools creates a ChatService with a pre-configured tool registry
// Useful when you want to inject a tool registry with pre-registered tools
func NewChatServiceWithTools(config *AIConfig, toolRegistry *ToolRegistry) (*ChatService, error) {
	provider, err := NewChatProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &ChatService{
		provider:     provider,
		config:       config,
		toolRegistry: toolRegistry,
	}, nil
}

// RegisterTool adds a tool to the service's tool registry
func (s *ChatService) RegisterTool(tool ToolExecutor) error {
	return s.toolRegistry.Register(tool)
}

// GetToolRegistry returns the tool registry for external tool registration
func (s *ChatService) GetToolRegistry() *ToolRegistry {
	return s.toolRegistry
}

// StreamChat sends messages to AI provider and streams the response (legacy text-only method)
// For tool-enabled streaming, use StreamChatWithTools
// Extracts user identity from context for logging and multi-tenancy
func (s *ChatService) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	// Extract identity from context (for logging and multi-tenancy)
	identity := s.getIdentityFromContext(ctx)
	requestID := s.getRequestIDFromContext(ctx)

	// Log the request
	if identity != nil {
		log.Printf("[ChatService] Request from %s (%s) - RequestID: %s - Provider: %s Model: %s",
			identity.Name, identity.Type, requestID, s.config.Provider, s.config.Model)
	} else {
		log.Printf("[ChatService] Request (no identity) - RequestID: %s - Provider: %s Model: %s",
			requestID, s.config.Provider, s.config.Model)
	}

	// Validate messages
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	// Call provider's StreamChat
	outputChan, err := s.provider.StreamChat(ctx, messages)
	if err != nil {
		log.Printf("[ChatService] ERROR - RequestID: %s - Failed to stream: %v", requestID, err)
		return nil, fmt.Errorf("failed to stream chat: %w", err)
	}

	// Wrap the output channel to handle context cancellation and logging
	wrappedChan := make(chan string, 100)

	go func() {
		defer close(wrappedChan)

		tokenCount := 0
		for {
			select {
			case <-ctx.Done():
				// Context cancelled
				log.Printf("[ChatService] Context cancelled - RequestID: %s - Tokens streamed: %d",
					requestID, tokenCount)
				return

			case chunk, ok := <-outputChan:
				if !ok {
					// Provider channel closed
					log.Printf("[ChatService] Stream complete - RequestID: %s - Total tokens: %d",
						requestID, tokenCount)
					return
				}

				tokenCount++

				// Forward chunk to wrapped channel
				select {
				case <-ctx.Done():
					return
				case wrappedChan <- chunk:
					// Chunk sent successfully
				}
			}
		}
	}()

	return wrappedChan, nil
}

// StreamChatWithTools sends messages to AI provider with tool support and streams events
// Handles tool calls automatically: when AI requests a tool, executes it and returns result
// Returns channel of StreamEvent which can be tokens, tool calls, or tool results
func (s *ChatService) StreamChatWithTools(ctx context.Context, messages []Message, maxToolCalls int) (<-chan StreamEvent, error) {
	identity := s.getIdentityFromContext(ctx)
	requestID := s.getRequestIDFromContext(ctx)

	// Log the request
	if identity != nil {
		log.Printf("[ChatService] Tool-enabled request from %s (%s) - RequestID: %s - Provider: %s Model: %s",
			identity.Name, identity.Type, requestID, s.config.Provider, s.config.Model)
	} else {
		log.Printf("[ChatService] Tool-enabled request (no identity) - RequestID: %s - Provider: %s Model: %s",
			requestID, s.config.Provider, s.config.Model)
	}

	// Validate messages
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	// Default max tool calls to prevent loops
	if maxToolCalls <= 0 {
		maxToolCalls = 5
	}

	// Create output channel for events
	eventChan := make(chan StreamEvent, 100)

	// Get tools for LangChain
	tools := s.toolRegistry.GetToolsForLangChain()

	// Check if provider supports tools
	supportsTools := false
	if toolProvider, ok := s.provider.(ToolCapableProvider); ok {
		supportsTools = toolProvider.SupportsTools()
	}

	if !supportsTools || len(tools) == 0 {
		// Fallback to text-only streaming
		log.Printf("[ChatService] Provider doesn't support tools or no tools registered - RequestID: %s", requestID)
		go func() {
			defer close(eventChan)
			textChan, err := s.provider.StreamChat(ctx, messages)
			if err != nil {
				eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
				return
			}
			for chunk := range textChan {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
			}
		}()
		return eventChan, nil
	}

	// Start tool-enabled streaming
	go func() {
		defer close(eventChan)

		toolCallCount := 0
		iterationCount := 0
		currentMessages := append([]Message{}, messages...) // Copy messages

		// Tool result cache: prevent duplicate tool executions
		resultCache := NewToolResultCache()

		// Circuit breaker: track recent tool calls to detect infinite loops
		recentToolCalls := make([]string, 0, 10)
		failedToolCalls := make(map[string]int) // Track failed attempts separately
		toolCallSignature := func(name string, args map[string]interface{}) string {
			argsJSON, _ := json.Marshal(args)
			return fmt.Sprintf("%s(%s)", name, string(argsJSON))
		}

		// Per-tool circuit breaker thresholds (max duplicate attempts before stopping)
		circuitBreakerThresholds := map[string]int{
			"read_file":         2, // Stop after 2 attempts (only 1 duplicate allowed)
			"write_file":        1, // Never allow duplicate writes
			"list_directory":    2, // Stop after 2 attempts
			"bash":              3, // Allow more for command variations
			"code_index_search": 3, // Allow query refinement
			// Default for other tools: 4 attempts
		}

		for toolCallCount < maxToolCalls && iterationCount < s.config.MaxIterations {
			iterationCount++

			// Calculate context size BEFORE applying sliding window
			contextSize := 0
			for _, msg := range currentMessages {
				contextSize += len(msg.Content)
			}

			// Apply sliding window BEFORE context exceeds model's token limit
			// Model token limit ≈ 32K total tokens (24K input + 8K output)
			// 24K tokens × 4 chars/token ≈ 96K chars theoretical max
			// Set threshold at 40K chars (≈10K tokens) to leave safety margin
			const maxContextSize = 40000 // 40KB threshold (safe for most models)
			if contextSize > maxContextSize {
				log.Printf("[Sliding Window] Context size %d chars exceeds threshold %d chars, applying window",
					contextSize, maxContextSize)
				currentMessages = applySlidingWindow(currentMessages, 6) // max 6 messages total

				// Recalculate after trimming
				contextSize = 0
				for _, msg := range currentMessages {
					contextSize += len(msg.Content)
				}
			}

			// Log iteration details
			log.Printf("[AI Processing] Iteration: %d, Request: %d chars, Context: %d chars, Tool calls so far: %d",
				iterationCount, contextSize, contextSize, toolCallCount)

			// DEBUG: Log context details before LLM API call to identify accumulation
			contextSize = calculateContextSize(currentMessages)
			toolResultPreview := getToolResultPreview(currentMessages, 200)
			log.Printf("[DEBUG Context] Before LLM call - Messages: %d, Total size: %d chars, Tool result preview: %s",
				len(currentMessages), contextSize, toolResultPreview)

			// DEBUG: Log tools being passed to LLM
			log.Printf("[DEBUG Tools] Passing %d tools to LLM provider %s", len(tools), s.config.Provider)
			if len(tools) > 0 {
				toolNames := make([]string, 0, 3)
				for i := 0; i < len(tools) && i < 3; i++ {
					if tools[i].Function != nil {
						toolNames = append(toolNames, tools[i].Function.Name)
					}
				}
				log.Printf("[DEBUG Tools] Sample tools: %v", toolNames)
			}

			// Call provider with tools
			toolProvider := s.provider.(ToolCapableProvider)
			response, err := toolProvider.StreamChatWithTools(ctx, currentMessages, tools)
			if err != nil {
				// Check if this is a rate limit error and we have a fallback model configured
				if isRateLimitError(err) && s.config.FallbackModel != "" {
					log.Printf("[Rate Limit] Detected rate limit error, switching to fallback model: %s → %s",
						s.config.Model, s.config.FallbackModel)

					// Notify user about fallback
					fallbackMsg := fmt.Sprintf("\n\n⚠️  RATE LIMIT DETECTED: Primary model '%s' has hit its rate limit.\n"+
						"🔄 Automatically switching to fallback model '%s' (local, no rate limits)...\n\n",
						s.config.Model, s.config.FallbackModel)
					eventChan <- StreamEvent{Type: StreamEventToken, Content: fallbackMsg}

					// Save original model, provider, and API key
					originalModel := s.config.Model
					originalProvider := s.config.Provider
					originalAPIKey := s.config.APIKey

					// Switch to fallback model
					s.config.Model = s.config.FallbackModel

					// Switch to Anthropic provider for Claude models
					if strings.Contains(strings.ToLower(s.config.FallbackModel), "claude") {
						s.config.Provider = "anthropic"
						// Load Anthropic API key from environment
						anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
						if anthropicKey == "" {
							log.Printf("[ChatService] ERROR - RequestID: %s - ANTHROPIC_API_KEY not found in environment", requestID)
							eventChan <- StreamEvent{Type: StreamEventError,
								Error: "Rate limit error and ANTHROPIC_API_KEY not configured for fallback"}
							return
						}
						s.config.APIKey = anthropicKey
					}

					// Recreate provider with fallback model
					fallbackProvider, err := NewChatProvider(s.config)
					if err != nil {
						log.Printf("[ChatService] ERROR - RequestID: %s - Failed to create fallback provider: %v", requestID, err)
						eventChan <- StreamEvent{Type: StreamEventError,
							Error: fmt.Sprintf("Rate limit error and failed to switch to fallback model: %v", err)}
						return
					}

					// Update provider
					s.provider = fallbackProvider

					// Retry with fallback provider
					if toolProvider, ok := s.provider.(ToolCapableProvider); ok {
						response, err = toolProvider.StreamChatWithTools(ctx, currentMessages, tools)
						if err != nil {
							log.Printf("[ChatService] ERROR - RequestID: %s - Fallback model also failed: %v", requestID, err)
							eventChan <- StreamEvent{Type: StreamEventError,
								Error: fmt.Sprintf("Both primary and fallback models failed: %v", err)}
							return
						}
						log.Printf("[Rate Limit] Successfully switched to fallback model '%s'", s.config.FallbackModel)

						// Send success notification
						successMsg := fmt.Sprintf("✅ Successfully switched to '%s'. Continuing with your request...\n\n", s.config.FallbackModel)
						eventChan <- StreamEvent{Type: StreamEventToken, Content: successMsg}

						// Note: We keep using the fallback model for the rest of this session
						// The original model, provider, and API key are saved in case we need them later
						_, _, _ = originalModel, originalProvider, originalAPIKey // Mark as used to avoid compiler warning
					} else {
						log.Printf("[ChatService] ERROR - Fallback provider doesn't support tools")
						eventChan <- StreamEvent{Type: StreamEventError, Error: "Fallback provider doesn't support tools"}
						return
					}
				} else {
					// Not a rate limit error or no fallback configured - just fail
					log.Printf("[ChatService] ERROR - RequestID: %s - Tool call failed: %v", requestID, err)
					eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
					return
				}
			}

			// Stream response tokens
			var responseText string
			responseTokens := 0
			for chunk := range response.TextChannel {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
				responseText += chunk
				responseTokens++
			}

			// Log iteration response details
			log.Printf("[AI Processing] Iteration: %d complete, Response: %d tokens, Tool calls requested: %d",
				iterationCount, responseTokens, len(response.ToolCalls))

			// Check for tool calls
			if len(response.ToolCalls) == 0 {
				// No more tool calls, we're done
				log.Printf("[ChatService] Stream complete - RequestID: %s - Total iterations: %d, Tool calls: %d",
					requestID, iterationCount, toolCallCount)
				return
			}

			// Process each tool call
			for _, toolCall := range response.ToolCalls {
				toolCallCount++
				if toolCallCount > maxToolCalls {
					log.Printf("[ChatService] Max tool calls reached (%d) - RequestID: %s", maxToolCalls, requestID)
					eventChan <- StreamEvent{Type: StreamEventError, Error: fmt.Sprintf("maximum tool calls (%d) exceeded", maxToolCalls)}
					return
				}

				// Log tool request with arguments
				argsJSON, _ := json.Marshal(toolCall.Args)
				log.Printf("[Tool Request] AI requested tool '%s' with args: %s",
					toolCall.Name, string(argsJSON))

				// Send tool call event
				eventChan <- StreamEvent{Type: StreamEventToolCall, ToolCall: &toolCall}

				// Generate signature for cache and circuit breaker
				signature := toolCallSignature(toolCall.Name, toolCall.Args)

				// Check tool result cache BEFORE execution
				var result ToolResult
				cachedResult, found := resultCache.Get(signature)
				if found {
					// Use cached result - avoid redundant execution
					result = *cachedResult
					log.Printf("[Tool Cache HIT] Using cached result for '%s' - skipping execution", toolCall.Name)

					// Add cache hit notice to the result so AI knows it's cached
					cacheNotice := fmt.Sprintf("🔁 CACHED RESULT: You already called '%s' with these exact arguments. Using previous result instead of re-executing.", toolCall.Name)

					// Prepend cache notice to the output
					if outputMap, ok := result.Output.(map[string]interface{}); ok {
						// Clone the map to avoid mutating the cached version
						newOutput := make(map[string]interface{})
						for k, v := range outputMap {
							newOutput[k] = v
						}
						newOutput["_cacheNotice"] = cacheNotice
						result.Output = newOutput
					}
				} else {
					// Execute tool (no cached result available)
					result = s.toolRegistry.ExecuteToolCall(ctx, toolCall)
					log.Printf("[Tool Cache MISS] Executed '%s' - storing result in cache", toolCall.Name)

					// Store in cache for future duplicate calls
					resultCache.Set(signature, &result)
				}

				// Send tool result event (full result to client for display)
				eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

				// CRITICAL: Track failed tool calls separately - stop immediately on retry of failed operation
				if result.Error != "" {
					failedToolCalls[signature]++
					if failedToolCalls[signature] >= 2 {
						// Second failure with same args - stop immediately!
						log.Printf("[Circuit Breaker - Failed Tool] Tool '%s' failed twice with identical arguments - stopping", toolCall.Name)
						eventChan <- StreamEvent{
							Type: StreamEventError,
							Error: fmt.Sprintf("❌ CRITICAL ERROR: Tool '%s' failed TWICE with identical arguments. Error: %s\n\n"+
								"🛑 You are retrying a FAILED operation. This will never work!\n"+
								"✅ Try a DIFFERENT approach:\n"+
								"   - If file not found: check directory listing or search results for the ACTUAL file name\n"+
								"   - If path wrong: try different path or create the file\n"+
								"   - If tool incompatible: use a different tool\n\n"+
								"DO NOT retry the same failed operation again!", toolCall.Name, result.Error),
						}
						return
					}
				}

				// Circuit breaker: check for repeated tool calls AND warn the AI
				recentToolCalls = append(recentToolCalls, signature)
				if len(recentToolCalls) > 10 {
					recentToolCalls = recentToolCalls[1:]
				}

				// Count how many times this exact tool+args was called in ALL history
				totalCount := 0
				for _, sig := range recentToolCalls {
					if sig == signature {
						totalCount++
					}
				}

				// Get tool-specific threshold
				threshold := circuitBreakerThresholds[toolCall.Name]
				if threshold == 0 {
					threshold = 4 // Default threshold
				}

				// Progressive warnings to AI (inject into context so AI sees them)
				var loopWarning string
				if totalCount == 2 {
					// First duplicate - gentle warning
					loopWarning = fmt.Sprintf("⚠️  WARNING: You already called '%s' with these exact arguments 1 time before. You should use the result from the previous call instead of repeating the same operation.", toolCall.Name)
				} else if totalCount == 3 && threshold > 3 {
					// Second duplicate - stronger warning (only if threshold allows)
					loopWarning = fmt.Sprintf("🔁 LOOP DETECTED: You called '%s' with identical arguments 2 times already. You are stuck in a loop! Use previous results or try a DIFFERENT approach - do NOT call this tool again with the same arguments.", toolCall.Name)
				} else if totalCount >= threshold {
					// Threshold reached - trigger circuit breaker
					log.Printf("[Circuit Breaker] Tool '%s' called %d times (threshold: %d) - stopping infinite loop", toolCall.Name, totalCount, threshold)
					eventChan <- StreamEvent{
						Type:  StreamEventError,
						Error: fmt.Sprintf("Circuit breaker triggered: tool '%s' called repeatedly (%d times) with identical arguments. The AI is stuck in an infinite loop and cannot complete this task.", toolCall.Name, totalCount),
					}
					return
				}

				// Log tool execution
				if result.Error != "" {
					log.Printf("[ChatService] Tool '%s' failed - RequestID: %s - Error: %s - Duration: %dms",
						result.Name, requestID, result.Error, result.DurationMs)
				} else {
					log.Printf("[ChatService] Tool '%s' succeeded - RequestID: %s - Duration: %dms",
						result.Name, requestID, result.DurationMs)
				}

				// Add assistant response to history (brief)
				currentMessages = append(currentMessages, Message{
					Role:    "assistant",
					Content: responseText,
				})

				// Add tool result to message history
				var toolResultMsg string
				if result.Error != "" {
					// Check if this is a permanent failure that shouldn't be retried
					errorLower := strings.ToLower(result.Error)
					isPermanentError := strings.Contains(errorLower, "requires mcp endpoint") ||
						strings.Contains(errorLower, "not supported") ||
						strings.Contains(errorLower, "cannot be used") ||
						strings.Contains(errorLower, "requires direct mcp")

					if isPermanentError {
						toolResultMsg = fmt.Sprintf("PERMANENT ERROR - Tool '%s' cannot be used in this context: %s. DO NOT retry this tool - it will not work.", result.Name, result.Error)
					} else {
						toolResultMsg = fmt.Sprintf("Tool '%s' error: %s", result.Name, result.Error)
					}
				} else {
					// Marshal output to JSON for context
					outputJSON, err := json.Marshal(result.Output)
					if err != nil {
						toolResultMsg = fmt.Sprintf("Tool '%s' result: <serialization error: %v>", result.Name, err)
					} else {
						// Check if output contains an error field (common pattern for tools returning error in response)
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if errorField, hasError := outputMap["error"]; hasError && errorField != nil {
								errorStr := fmt.Sprintf("%v", errorField)
								toolResultMsg = fmt.Sprintf("PERMANENT ERROR - Tool '%s' returned error: %s. DO NOT retry this tool.", result.Name, errorStr)
							} else {
								toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))
							}
						} else {
							toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))
						}
					}
				}

				// CRITICAL FIX: If we generated a loop warning, EMBED it in JSON result
				if loopWarning != "" {
					log.Printf("[Loop Detection] %s", loopWarning)

					// Send warning as a visible message to the user
					eventChan <- StreamEvent{Type: StreamEventToken, Content: "\n\n" + loopWarning + "\n\n"}

					// Try to embed warning into the JSON result instead of prepending as text
					if strings.HasPrefix(toolResultMsg, fmt.Sprintf("Tool '%s' result: ", result.Name)) {
						jsonPart := strings.TrimPrefix(toolResultMsg, fmt.Sprintf("Tool '%s' result: ", result.Name))
						var resultData map[string]interface{}
						if err := json.Unmarshal([]byte(jsonPart), &resultData); err == nil {
							resultData["_loopWarning"] = loopWarning
							if newJSON, err := json.Marshal(resultData); err == nil {
								toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(newJSON))
							} else {
								// fallback: text prepend if reserialization fails
								toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
							}
						} else {
							// fallback: text prepend if JSON parse fails
							toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
						}
					} else {
						// fallback: if message doesn’t match JSON result format
						toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
					}
				}

				// CRITICAL FIX: Truncate tool results that are too large to prevent token limit errors
				// Individual tool results can be HUGE (e.g., bash ls -R output = 1.98MB)
				// Even if sliding window triggers, if one recent message is huge, it doesn't help
				const maxToolResultSize = 10000 // 10KB per tool result
				if len(toolResultMsg) > maxToolResultSize {
					originalSize := len(toolResultMsg)
					// Keep first 9KB and add truncation notice
					toolResultMsg = toolResultMsg[:maxToolResultSize-500] + fmt.Sprintf("\n\n... [TRUNCATED: Result was %d chars, showing first %d chars to prevent token limit. If you need more, use a more specific query or process the data in smaller chunks.] ...", originalSize, maxToolResultSize-500)
					log.Printf("[Tool Result Truncation] Truncated tool '%s' result from %d to %d chars to prevent token limit",
						result.Name, originalSize, len(toolResultMsg))
				}

				currentMessages = append(currentMessages, Message{
					Role:    "system",
					Content: toolResultMsg,
				})

				log.Printf("[AI Processing] Context after tool %d: %d messages, %d total chars",
					toolCallCount, len(currentMessages), func() int {
						sum := 0
						for _, m := range currentMessages {
							sum += len(m.Content)
						}
						return sum
					}())
			}
		}

		// Max iterations reached
		log.Printf("[ChatService] Max tool calls reached - RequestID: %s - Total iterations: %d, Tool calls: %d",
			requestID, iterationCount, toolCallCount)
	}()

	return eventChan, nil
}

// StreamChatWithToolsFiltered sends messages to AI provider with restricted tool access
// This is used for subagents to prevent them from calling coordinator tools
// Only the specified tools in allowedToolNames will be available to the AI
func (s *ChatService) StreamChatWithToolsFiltered(ctx context.Context, messages []Message, maxToolCalls int, allowedToolNames []string) (<-chan StreamEvent, error) {
	identity := s.getIdentityFromContext(ctx)
	requestID := s.getRequestIDFromContext(ctx)

	// Log the request
	if identity != nil {
		log.Printf("[ChatService] Tool-filtered request from %s (%s) - RequestID: %s - Provider: %s Model: %s - Allowed tools: %v",
			identity.Name, identity.Type, requestID, s.config.Provider, s.config.Model, allowedToolNames)
	} else {
		log.Printf("[ChatService] Tool-filtered request (no identity) - RequestID: %s - Provider: %s Model: %s - Allowed tools: %v",
			requestID, s.config.Provider, s.config.Model, allowedToolNames)
	}

	// Validate messages
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	// Default max tool calls to prevent loops
	if maxToolCalls <= 0 {
		maxToolCalls = 5
	}

	// Create output channel for events
	eventChan := make(chan StreamEvent, 100)

	// Get FILTERED tools for LangChain (only allowed tools)
	tools := s.toolRegistry.GetFilteredToolsForLangChain(allowedToolNames)

	log.Printf("[ChatService] Filtered %d tools for subagent (from allowlist of %d tools)", len(tools), len(allowedToolNames))

	// Check if provider supports tools
	supportsTools := false
	if toolProvider, ok := s.provider.(ToolCapableProvider); ok {
		supportsTools = toolProvider.SupportsTools()
	}

	if !supportsTools || len(tools) == 0 {
		// Fallback to text-only streaming
		log.Printf("[ChatService] Provider doesn't support tools or no tools registered - RequestID: %s", requestID)
		go func() {
			defer close(eventChan)
			textChan, err := s.provider.StreamChat(ctx, messages)
			if err != nil {
				eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
				return
			}
			for chunk := range textChan {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
			}
		}()
		return eventChan, nil
	}

	// Start tool-enabled streaming
	go func() {
		defer close(eventChan)

		toolCallCount := 0
		iterationCount := 0
		currentMessages := append([]Message{}, messages...) // Copy messages

		// Tool result cache: prevent duplicate tool executions
		resultCache := NewToolResultCache()

		// Circuit breaker: track recent tool calls to detect infinite loops
		recentToolCalls := make([]string, 0, 10)
		failedToolCalls := make(map[string]int) // Track failed attempts separately
		toolCallSignature := func(name string, args map[string]interface{}) string {
			argsJSON, _ := json.Marshal(args)
			return fmt.Sprintf("%s(%s)", name, string(argsJSON))
		}

		// Per-tool circuit breaker thresholds (max duplicate attempts before stopping)
		circuitBreakerThresholds := map[string]int{
			"read_file":         2, // Stop after 2 attempts (only 1 duplicate allowed)
			"write_file":        1, // Never allow duplicate writes
			"list_directory":    2, // Stop after 2 attempts
			"bash":              3, // Allow more for command variations
			"code_index_search": 3, // Allow query refinement
			// Default for other tools: 4 attempts
		}

		for toolCallCount < maxToolCalls && iterationCount < s.config.MaxIterations {
			iterationCount++

			// Calculate context size BEFORE applying sliding window
			contextSize := 0
			for _, msg := range currentMessages {
				contextSize += len(msg.Content)
			}

			// Apply sliding window BEFORE context exceeds model's token limit
			const maxContextSize = 40000 // 40KB threshold (safe for most models)
			if contextSize > maxContextSize {
				log.Printf("[Sliding Window] Context size %d chars exceeds threshold %d chars, applying window",
					contextSize, maxContextSize)
				currentMessages = applySlidingWindow(currentMessages, 6) // max 6 messages total

				// Recalculate after trimming
				contextSize = 0
				for _, msg := range currentMessages {
					contextSize += len(msg.Content)
				}
			}

			// Log iteration details
			log.Printf("[AI Processing - Filtered Tools] Iteration: %d, Request: %d chars, Context: %d chars, Tool calls so far: %d",
				iterationCount, contextSize, contextSize, toolCallCount)

			// DEBUG: Log context details before LLM API call
			contextSize = calculateContextSize(currentMessages)
			toolResultPreview := getToolResultPreview(currentMessages, 200)
			log.Printf("[DEBUG Context - Filtered] Before LLM call - Messages: %d, Total size: %d chars, Tool result preview: %s",
				len(currentMessages), contextSize, toolResultPreview)

			// Call provider with FILTERED tools
			toolProvider := s.provider.(ToolCapableProvider)
			response, err := toolProvider.StreamChatWithTools(ctx, currentMessages, tools)
			if err != nil {
				log.Printf("[ChatService] ERROR - RequestID: %s - Tool call failed: %v", requestID, err)
				eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
				return
			}

			// Stream response tokens
			var responseText string
			responseTokens := 0
			for chunk := range response.TextChannel {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
				responseText += chunk
				responseTokens++
			}

			// Log iteration response details
			log.Printf("[AI Processing - Filtered] Iteration: %d complete, Response: %d tokens, Tool calls requested: %d",
				iterationCount, responseTokens, len(response.ToolCalls))

			// Check for tool calls
			if len(response.ToolCalls) == 0 {
				// No more tool calls, we're done
				log.Printf("[ChatService - Filtered] Stream complete - RequestID: %s - Total iterations: %d, Tool calls: %d",
					requestID, iterationCount, toolCallCount)
				return
			}

			// Process each tool call
			for _, toolCall := range response.ToolCalls {
				toolCallCount++
				if toolCallCount > maxToolCalls {
					log.Printf("[ChatService - Filtered] Max tool calls reached (%d) - RequestID: %s", maxToolCalls, requestID)
					eventChan <- StreamEvent{Type: StreamEventError, Error: fmt.Sprintf("maximum tool calls (%d) exceeded", maxToolCalls)}
					return
				}

				// Log tool request with arguments
				argsJSON, _ := json.Marshal(toolCall.Args)
				log.Printf("[Tool Request - Filtered] AI requested tool '%s' with args: %s",
					toolCall.Name, string(argsJSON))

				// Send tool call event
				eventChan <- StreamEvent{Type: StreamEventToolCall, ToolCall: &toolCall}

				// Generate signature for cache and circuit breaker
				signature := toolCallSignature(toolCall.Name, toolCall.Args)

				// Check tool result cache BEFORE execution
				var result ToolResult
				cachedResult, found := resultCache.Get(signature)
				if found {
					// Use cached result - avoid redundant execution
					result = *cachedResult
					log.Printf("[Tool Cache HIT] Using cached result for '%s' - skipping execution", toolCall.Name)

					// Add cache hit notice to the result
					cacheNotice := fmt.Sprintf("🔁 CACHED RESULT: You already called '%s' with these exact arguments. Using previous result instead of re-executing.", toolCall.Name)
					if outputMap, ok := result.Output.(map[string]interface{}); ok {
						newOutput := make(map[string]interface{})
						for k, v := range outputMap {
							newOutput[k] = v
						}
						newOutput["_cacheNotice"] = cacheNotice
						result.Output = newOutput
					}
				} else {
					// Execute tool (no cached result available)
					result = s.toolRegistry.ExecuteToolCall(ctx, toolCall)
					log.Printf("[Tool Cache MISS] Executed '%s' - storing result in cache", toolCall.Name)

					// Store in cache for future duplicate calls
					resultCache.Set(signature, &result)
				}

				// Send tool result event
				eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

				// CRITICAL: Track failed tool calls separately
				if result.Error != "" {
					failedToolCalls[signature]++
					if failedToolCalls[signature] >= 2 {
						log.Printf("[Circuit Breaker - Failed Tool] Tool '%s' failed twice with identical arguments - stopping", toolCall.Name)
						eventChan <- StreamEvent{
							Type: StreamEventError,
							Error: fmt.Sprintf("❌ CRITICAL ERROR: Tool '%s' failed TWICE with identical arguments. Error: %s\n\n"+
								"🛑 You are retrying a FAILED operation. This will never work!\n"+
								"✅ Try a DIFFERENT approach - DO NOT retry the same failed operation!", toolCall.Name, result.Error),
						}
						return
					}
				}

				// Circuit breaker: check for repeated tool calls
				recentToolCalls = append(recentToolCalls, signature)
				if len(recentToolCalls) > 10 {
					recentToolCalls = recentToolCalls[1:]
				}

				// Count duplicates
				totalCount := 0
				for _, sig := range recentToolCalls {
					if sig == signature {
						totalCount++
					}
				}

				// Get tool-specific threshold
				threshold := circuitBreakerThresholds[toolCall.Name]
				if threshold == 0 {
					threshold = 4 // Default threshold
				}

				// Progressive warnings
				var loopWarning string
				if totalCount == 2 {
					loopWarning = fmt.Sprintf("⚠️  WARNING: You already called '%s' with these exact arguments 1 time before. Use the previous result instead of repeating.", toolCall.Name)
				} else if totalCount == 3 && threshold > 3 {
					loopWarning = fmt.Sprintf("🔁 LOOP DETECTED: You called '%s' with identical arguments 2 times already. You are stuck in a loop! Use previous results or try a DIFFERENT approach.", toolCall.Name)
				} else if totalCount >= threshold {
					log.Printf("[Circuit Breaker] Tool '%s' called %d times (threshold: %d) - stopping infinite loop", toolCall.Name, totalCount, threshold)
					eventChan <- StreamEvent{
						Type:  StreamEventError,
						Error: fmt.Sprintf("Circuit breaker triggered: tool '%s' called repeatedly (%d times) with identical arguments. The AI is stuck in an infinite loop.", toolCall.Name, totalCount),
					}
					return
				}

				// Add assistant response to history
				currentMessages = append(currentMessages, Message{
					Role:    "assistant",
					Content: responseText,
				})

				// Add tool result to message history
				var toolResultMsg string
				if result.Error != "" {
					toolResultMsg = fmt.Sprintf("Tool '%s' error: %s", result.Name, result.Error)
				} else {
					outputJSON, err := json.Marshal(result.Output)
					if err != nil {
						toolResultMsg = fmt.Sprintf("Tool '%s' result: <serialization error: %v>", result.Name, err)
					} else {
						toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))
					}
				}

				// CRITICAL FIX: If we generated a loop warning, EMBED it in JSON result
				if loopWarning != "" {
					log.Printf("[Loop Detection] %s", loopWarning)

					// Send warning as a visible message to the user
					eventChan <- StreamEvent{Type: StreamEventToken, Content: "\n\n" + loopWarning + "\n\n"}

					// Try to embed warning into the JSON result instead of prepending as text
					if strings.HasPrefix(toolResultMsg, fmt.Sprintf("Tool '%s' result: ", result.Name)) {
						jsonPart := strings.TrimPrefix(toolResultMsg, fmt.Sprintf("Tool '%s' result: ", result.Name))
						var resultData map[string]interface{}
						if err := json.Unmarshal([]byte(jsonPart), &resultData); err == nil {
							resultData["_loopWarning"] = loopWarning
							if newJSON, err := json.Marshal(resultData); err == nil {
								toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(newJSON))
							} else {
								// fallback: text prepend if reserialization fails
								toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
							}
						} else {
							// fallback: text prepend if JSON parse fails
							toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
						}
					} else {
						// fallback: if message doesn't match JSON result format
						toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
					}
				}

				// Truncate large tool results
				const maxToolResultSize = 10000
				if len(toolResultMsg) > maxToolResultSize {
					originalSize := len(toolResultMsg)
					toolResultMsg = toolResultMsg[:maxToolResultSize-500] + fmt.Sprintf("\n\n... [TRUNCATED: Result was %d chars, showing first %d chars] ...", originalSize, maxToolResultSize-500)
					log.Printf("[Tool Result Truncation] Truncated tool '%s' result from %d to %d chars",
						result.Name, originalSize, len(toolResultMsg))
				}

				currentMessages = append(currentMessages, Message{
					Role:    "system",
					Content: toolResultMsg,
				})
			}
		}

		// Max iterations reached
		log.Printf("[ChatService - Filtered] Max tool calls reached - RequestID: %s - Total iterations: %d, Tool calls: %d",
			requestID, iterationCount, toolCallCount)
	}()

	return eventChan, nil
}

// GetConfig returns the current AI configuration
func (s *ChatService) GetConfig() *AIConfig {
	return s.config
}

// getIdentityFromContext extracts user identity from context
func (s *ChatService) getIdentityFromContext(ctx context.Context) *Identity {
	identity, ok := ctx.Value(IdentityKey).(*Identity)
	if !ok {
		return nil
	}
	return identity
}

// getRequestIDFromContext extracts request ID from context
func (s *ChatService) getRequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(RequestIDKey).(string)
	if !ok || requestID == "" {
		return "unknown"
	}
	return requestID
}

// WithIdentity adds identity to context
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, IdentityKey, identity)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetIdentityFromContext is a helper to extract identity from context
func GetIdentityFromContext(ctx context.Context) (*Identity, error) {
	identity, ok := ctx.Value(IdentityKey).(*Identity)
	if !ok || identity == nil {
		return nil, fmt.Errorf("identity not found in context")
	}
	return identity, nil
}

// calculateContextSize returns the total character count of all messages
func calculateContextSize(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content)
	}
	return total
}

// getToolResultPreview extracts the first maxChars of tool result content from messages
// Useful for debugging to see what tool results are being accumulated
func getToolResultPreview(messages []Message, maxChars int) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		// Look for tool result messages (role=system with "Tool '...' result:" pattern)
		if msg.Role == "system" && len(msg.Content) > 0 {
			if len(msg.Content) <= maxChars {
				return msg.Content
			}
			return msg.Content[:maxChars] + "..."
		}
	}
	return "(no tool results found)"
}

// applySlidingWindow keeps only recent messages to prevent context accumulation
// Strategy: Keep system prompt + original user message + last N tool exchanges
// This prevents sending 100+200+300 accumulated messages, instead sending 100+100+100
func applySlidingWindow(messages []Message, maxMessages int) []Message {
	if len(messages) <= maxMessages {
		return messages // No need to trim
	}

	// Identify system prompt (if exists at index 0)
	hasSystemPrompt := len(messages) > 0 && messages[0].Role == "system"

	// Find original user message (first "user" role after system prompt)
	var systemMsg, userMsg *Message
	userMsgIdx := -1

	if hasSystemPrompt {
		systemMsg = &messages[0]
		// Find first user message after system
		for i := 1; i < len(messages); i++ {
			if messages[i].Role == "user" {
				userMsg = &messages[i]
				userMsgIdx = i
				break
			}
		}
	} else {
		// No system prompt - first message should be user
		if len(messages) > 0 && messages[0].Role == "user" {
			userMsg = &messages[0]
			userMsgIdx = 0
		}
	}

	// Calculate how many recent messages to keep
	reservedSlots := 0
	if systemMsg != nil {
		reservedSlots++
	}
	if userMsg != nil {
		reservedSlots++
	}

	recentCount := maxMessages - reservedSlots
	if recentCount < 0 {
		recentCount = 0
	}

	// Build new message list
	result := make([]Message, 0, maxMessages)

	// Add system prompt if exists
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}

	// Add original user message if exists
	if userMsg != nil {
		result = append(result, *userMsg)
	}

	// Add last N messages (tool exchanges)
	if recentCount > 0 && len(messages) > userMsgIdx+1 {
		// Get messages after the original user message
		afterUserMsg := messages[userMsgIdx+1:]

		// Take last recentCount messages
		startIdx := len(afterUserMsg) - recentCount
		if startIdx < 0 {
			startIdx = 0
		}

		result = append(result, afterUserMsg[startIdx:]...)
	}

	log.Printf("[Sliding Window] Reduced from %d to %d messages (system: %v, user: %v, recent: %d)",
		len(messages), len(result), systemMsg != nil, userMsg != nil, recentCount)

	return result
}

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
