package aiservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	Content    string          `json:"content,omitempty"`     // For token events
	ToolCall   *ToolCall       `json:"toolCall,omitempty"`    // For tool_call events
	ToolResult *ToolResult     `json:"toolResult,omitempty"`  // For tool_result events
	Error      string          `json:"error,omitempty"`       // For error events
}

// ChatService manages AI chat operations with provider abstraction
type ChatService struct {
	provider     ChatProvider
	config       *AIConfig
	toolRegistry *ToolRegistry
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

		for toolCallCount < maxToolCalls {
			iterationCount++

			// Calculate context size BEFORE applying sliding window
			contextSize := 0
			for _, msg := range currentMessages {
				contextSize += len(msg.Content)
			}

			// Apply sliding window ONLY if context exceeds 500KB (500,000 chars)
			const maxContextSize = 500000 // 500KB threshold
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

			// Call provider with tools
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

				// Execute tool
				result := s.toolRegistry.ExecuteToolCall(ctx, toolCall)

				// Send tool result event (full result to client for display)
				eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

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
					toolResultMsg = fmt.Sprintf("Tool '%s' error: %s", result.Name, result.Error)
				} else {
					// Marshal output to JSON for context
					outputJSON, err := json.Marshal(result.Output)
					if err != nil {
						toolResultMsg = fmt.Sprintf("Tool '%s' result: <serialization error: %v>", result.Name, err)
					} else {
						toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))
					}
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
