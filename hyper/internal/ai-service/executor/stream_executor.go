package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/config"
	"hyper/internal/metrics"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// StreamConfig contains all configuration options for the universal stream executor.
// This config makes the executor work for BOTH parent chat and subchat contexts.
type StreamConfig struct {
	// Core identifiers
	SessionID primitive.ObjectID // Chat session ID for message storage
	CompanyID string             // Company ID for multi-tenancy

	// AI configuration
	SystemPrompt string   // System prompt for the AI
	AllowedTools []string // Tool filter (nil = all tools, [] = no tools, ["tool1", "tool2"] = specific tools)

	// Output destination
	OutputSink StreamOutputSink // Where to send stream output (WebSocket or ProgressNotifier)

	// Optional callbacks
	CompletionValidator CompletionValidatorFunc // Custom logic to determine when streaming is complete
	OnMessageSavedWhileDisconnected func(sessionID primitive.ObjectID) // Callback when message is saved but client disconnected

	// Interrupt handling
	InterruptCh <-chan struct{} // Channel to listen for user interrupts

	// Tool result processing
	ToolResultProcessor ToolResultProcessorFunc // Custom tool result processing (for size limits, formatting, etc.)

	// Context
	Logger *zap.Logger
}

// ToolResultProcessorFunc is a function that processes tool results before saving/streaming.
// Returns processed output and whether to save/stream the result.
type ToolResultProcessorFunc func(toolName string, output interface{}) (processedOutput string, shouldSave bool, shouldStream bool)

// StreamExecutor is the universal AI stream executor.
// It handles streaming for BOTH parent chat and subchat contexts using the same core logic.
//
// Key Design Principles:
// - Context-agnostic: Works with any StreamOutputSink implementation
// - Resilient: Continues processing even if client disconnects
// - Observable: Records metrics and logs for monitoring
// - Interruptible: Supports user interrupts via channel
// - Safe: Panic recovery prevents crashes
type StreamExecutor struct {
	config       StreamConfig
	chatService  ChatServiceInterface
	aiService    AIServiceInterface
	logger       *zap.Logger
	maxToolCalls int
}

// NewStreamExecutor creates a new universal stream executor.
func NewStreamExecutor(
	config StreamConfig,
	chatService ChatServiceInterface,
	aiService AIServiceInterface,
) *StreamExecutor {
	return &StreamExecutor{
		config:       config,
		chatService:  chatService,
		aiService:    aiService,
		logger:       config.Logger,
		maxToolCalls: aiService.GetConfig().MaxToolCalls,
	}
}

// Execute runs the universal streaming loop.
// This is the core method that processes AI stream events and handles:
// - Token streaming and accumulation
// - Tool call execution and saving
// - Tool result processing and streaming
// - User interrupts (non-blocking check)
// - Client disconnect detection
// - Error handling with panic recovery
// - Buffer overflow protection
// - Metrics recording
//
// Returns the full accumulated response and any error.
func (e *StreamExecutor) Execute(ctx context.Context, messages []aiservice.Message) (string, error) {
	e.logger.Info("🚀 Starting universal AI stream execution",
		zap.String("sessionId", e.config.SessionID.Hex()),
		zap.String("companyId", e.config.CompanyID),
		zap.Int("allowedTools", len(e.config.AllowedTools)),
		zap.Int("messageCount", len(messages)))

	// Track stream start time for metrics
	streamStart := time.Now()

	// Add system prompt if provided
	if e.config.SystemPrompt != "" {
		systemMessage := aiservice.Message{
			Role:    "system",
			Content: e.config.SystemPrompt,
		}
		messages = append([]aiservice.Message{systemMessage}, messages...)
		e.logger.Debug("Injected system prompt",
			zap.String("sessionId", e.config.SessionID.Hex()),
			zap.Int("promptLength", len(e.config.SystemPrompt)))
	}

	// Start AI streaming based on tool filter configuration
	var aiStream <-chan aiservice.StreamEvent
	var err error

	if e.config.AllowedTools != nil && len(e.config.AllowedTools) > 0 {
		// Filtered tools mode (subchat or direct subagent)
		e.logger.Info("Starting AI stream with filtered tools",
			zap.String("sessionId", e.config.SessionID.Hex()),
			zap.Strings("allowedTools", e.config.AllowedTools))
		aiStream, err = e.aiService.StreamChatWithToolsFiltered(ctx, messages, e.maxToolCalls, e.config.AllowedTools)
	} else if e.config.AllowedTools == nil {
		// Full tools mode (coordinator)
		e.logger.Info("Starting AI stream with full tool access",
			zap.String("sessionId", e.config.SessionID.Hex()))
		aiStream, err = e.aiService.StreamChatWithTools(ctx, messages, e.maxToolCalls)
	} else {
		// No tools mode (len(AllowedTools) == 0)
		e.logger.Info("Starting AI stream with no tools",
			zap.String("sessionId", e.config.SessionID.Hex()))
		aiStream, err = e.aiService.StreamChatWithTools(ctx, messages, 0)
	}

	if err != nil {
		e.logger.Error("Failed to start AI streaming", zap.Error(err))
		e.config.OutputSink.SendError("Failed to get AI response: " + err.Error())
		return "", fmt.Errorf("failed to start AI streaming: %w", err)
	}

	// Streaming state
	fullResponse := ""
	tokenCount := 0
	toolCallCount := 0

	// Panic recovery for stream processing
	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("Panic during AI stream processing",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.Any("panic", r),
				zap.Int("tokensStreamed", tokenCount),
				zap.Int("toolCalls", toolCallCount))

			// Try to save whatever response we have so far
			if fullResponse != "" {
				if _, err := e.chatService.SaveMessage(ctx, e.config.SessionID, "assistant", fullResponse, e.config.CompanyID); err != nil {
					e.logger.Error("Failed to save partial response after panic", zap.Error(err))
				}
			}

			// Try to notify client if still connected
			if !e.config.OutputSink.IsDisconnected() {
				e.config.OutputSink.SendError("Internal error during AI processing")
			}
		}
	}()

	// Main streaming loop
	for event := range aiStream {
		// PRIORITY CHECK: Non-blocking interrupt detection (runs before every AI event)
		// This ensures user interrupts are processed promptly without blocking AI streaming
		if e.config.InterruptCh != nil {
			select {
			case <-e.config.InterruptCh:
				e.logger.Info("🚨 User interrupt detected during AI streaming",
					zap.String("sessionId", e.config.SessionID.Hex()),
					zap.Int("tokensStreamed", tokenCount))

				// Emit notification to client if still connected
				if !e.config.OutputSink.IsDisconnected() {
					// Send interrupt notification as token (preserves message flow)
					e.config.OutputSink.SendToken("\n\n⏸️ _Interrupt detected - processing your message..._\n\n")
				}

				// Save accumulated response before interrupt
				if fullResponse != "" {
					if _, err := e.chatService.SaveMessage(ctx, e.config.SessionID, "assistant", fullResponse, e.config.CompanyID); err != nil {
						e.logger.Error("Failed to save response before interrupt", zap.Error(err))
					}
				}

				// Send done event so frontend knows streaming has stopped
				if !e.config.OutputSink.IsDisconnected() {
					e.config.OutputSink.SendDone()
				}

				// Return to allow caller to handle interrupt
				return fullResponse, nil
			default:
				// No interrupt, continue with normal processing
			}
		}

		// CONTEXT CHECK: Ensure context hasn't been cancelled
		select {
		case <-ctx.Done():
			e.logger.Info("Context cancelled during streaming",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.Int("tokensStreamed", tokenCount),
				zap.Int("toolCalls", toolCallCount))

			// Save accumulated response before exiting
			if fullResponse != "" {
				if _, err := e.chatService.SaveMessage(ctx, e.config.SessionID, "assistant", fullResponse, e.config.CompanyID); err != nil {
					e.logger.Error("Failed to save response on context cancellation", zap.Error(err))
				}
			}

			// Send done event so frontend knows streaming has stopped
			if !e.config.OutputSink.IsDisconnected() {
				e.config.OutputSink.SendDone()
			}

			return fullResponse, ctx.Err()
		default:
			// Context is still active, process event
		}

		// PROCESS STREAM EVENT
		switch event.Type {
		case aiservice.StreamEventToken:
			if err := e.handleToken(&event, &fullResponse, &tokenCount); err != nil {
				return fullResponse, err
			}

		case aiservice.StreamEventToolCall:
			if err := e.handleToolCall(ctx, &event, &fullResponse, &toolCallCount); err != nil {
				return fullResponse, err
			}

		case aiservice.StreamEventToolResult:
			if err := e.handleToolResult(ctx, &event); err != nil {
				return fullResponse, err
			}

		case aiservice.StreamEventError:
			e.logger.Error("AI service error during streaming", zap.String("error", event.Error))
			e.config.OutputSink.SendError("AI error: " + event.Error)
			return fullResponse, fmt.Errorf("AI error: %s", event.Error)
		}

		// COMPLETION CHECK: Custom validation logic (if provided)
		if e.config.CompletionValidator != nil && e.config.CompletionValidator(fullResponse, toolCallCount) {
			e.logger.Info("Custom completion validator triggered - stopping stream",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.Int("tokensStreamed", tokenCount),
				zap.Int("toolCalls", toolCallCount))
			break
		}
	}

	// RACE CONDITION FIX: Save to database FIRST, then send signals
	// Previous order: done -> save (broken: frontend refreshes before save completes)
	// New order: save -> message_saved -> done (safe: message guaranteed in DB before done)
	var savedMsgID string
	if fullResponse != "" {
		savedMsg, err := e.chatService.SaveMessage(ctx, e.config.SessionID, "assistant", fullResponse, e.config.CompanyID)
		if err != nil {
			e.logger.Error("Failed to save final AI response", zap.Error(err))
			if !e.config.OutputSink.IsDisconnected() {
				e.config.OutputSink.SendError("Failed to save AI response")
			}
			return fullResponse, fmt.Errorf("failed to save final response: %w", err)
		}

		// Extract message ID from saved message for frontend reconciliation
		if savedMsg != nil {
			if msgPtr, ok := (*savedMsg).(*interface{}); ok && msgPtr != nil {
				// The interface wraps *models.ChatMessage, try to extract ID via reflection
				// For safety, we'll use a type switch approach
				savedMsgID = extractMessageID(*savedMsg)
			} else {
				savedMsgID = extractMessageID(*savedMsg)
			}
		}

		e.logger.Debug("Saved final assistant text",
			zap.String("sessionId", e.config.SessionID.Hex()),
			zap.String("messageId", savedMsgID),
			zap.Int("textLength", len(fullResponse)))

		// If WebSocket was disconnected but message was saved, invoke callback to notify
		if e.config.OutputSink.IsDisconnected() && e.config.OnMessageSavedWhileDisconnected != nil {
			e.logger.Info("WebSocket disconnected but message saved - invoking notification callback",
				zap.String("sessionId", e.config.SessionID.Hex()))
			e.config.OnMessageSavedWhileDisconnected(e.config.SessionID)
		}
	} else {
		e.logger.Debug("No remaining assistant text to save (all text saved before tool calls)",
			zap.String("sessionId", e.config.SessionID.Hex()))
	}

	// Send message_saved event for assistant message (for frontend reconciliation)
	if savedMsgID != "" && !e.config.OutputSink.IsDisconnected() {
		// Format: "assistant:{messageId}" to distinguish from user messages
		if err := e.config.OutputSink.SendMessageSaved("assistant:" + savedMsgID); err != nil {
			e.logger.Warn("Failed to send message_saved event",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.Error(err))
		}
	}

	// Send completion signal LAST (after save is guaranteed)
	if !e.config.OutputSink.IsDisconnected() {
		e.config.OutputSink.SendDone()
	}

	// Record AI streaming metrics
	metrics.AIStreamTokens.Add(float64(tokenCount))
	metrics.AIStreamDuration.Observe(time.Since(streamStart).Seconds())

	if e.config.OutputSink.IsDisconnected() {
		e.logger.Info("AI response completed in background after client disconnect",
			zap.String("sessionId", e.config.SessionID.Hex()),
			zap.Int("tokensStreamed", tokenCount),
			zap.Int("toolCalls", toolCallCount),
			zap.Int("responseLength", len(fullResponse)))
	} else {
		e.logger.Info("AI response streamed successfully",
			zap.String("sessionId", e.config.SessionID.Hex()),
			zap.Int("tokensStreamed", tokenCount),
			zap.Int("toolCalls", toolCallCount),
			zap.Int("responseLength", len(fullResponse)))
	}

	return fullResponse, nil
}

// handleToken processes a token stream event.
// Accumulates the token into fullResponse and streams it to the output sink.
func (e *StreamExecutor) handleToken(event *aiservice.StreamEvent, fullResponse *string, tokenCount *int) error {
	// Accumulate response even if client disconnected
	*fullResponse += event.Content
	*tokenCount++

	// Buffer size protection: Check accumulated response size
	if len(*fullResponse) > config.MaxStreamBufferBytes {
		e.logger.Warn("AI response exceeded buffer limit, truncating stream",
			zap.String("sessionId", e.config.SessionID.Hex()),
			zap.Int("responseSize", len(*fullResponse)),
			zap.Int("maxSize", config.MaxStreamBufferBytes),
			zap.Int("tokensStreamed", *tokenCount))

		// Send truncation notice to client if still connected
		if !e.config.OutputSink.IsDisconnected() {
			e.config.OutputSink.SendToken("\n\n_[Response truncated - exceeded maximum size limit]_")
		}

		// Record truncation metric
		metrics.AIResponseTruncations.Inc()

		// Signal to stop streaming
		return fmt.Errorf("response exceeded buffer limit")
	}

	// Try to send to output sink if client still connected
	if !e.config.OutputSink.IsDisconnected() {
		if err := e.config.OutputSink.SendToken(event.Content); err != nil {
			e.logger.Warn("Failed to send token to output sink",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.Error(err))
			// Don't return error - continue processing for database save
		}
	}

	return nil
}

// handleToolCall processes a tool call stream event.
// Saves accumulated text (if any), then saves and streams the tool call.
func (e *StreamExecutor) handleToolCall(ctx context.Context, event *aiservice.StreamEvent, fullResponse *string, toolCallCount *int) error {
	*toolCallCount++

	// CRITICAL: Save accumulated assistant text BEFORE the tool call (if any)
	// This ensures text is persisted even if client refreshes mid-execution
	if *fullResponse != "" {
		_, err := e.chatService.SaveMessage(ctx, e.config.SessionID, "assistant", *fullResponse, e.config.CompanyID)
		if err != nil {
			e.logger.Error("Failed to save assistant text before tool call", zap.Error(err))
			// Continue even if save fails
		} else {
			e.logger.Debug("Saved assistant text before tool call",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.Int("textLength", len(*fullResponse)))
		}
		// Clear accumulated response to start fresh for text after this tool call
		*fullResponse = ""
	}

	// Save tool call to database (always, even if client disconnected)
	e.logger.Info("💾 Saving tool call to database",
		zap.String("sessionId", e.config.SessionID.Hex()),
		zap.String("toolCallID", event.ToolCall.ID),
		zap.String("toolName", event.ToolCall.Name))
	_, err := e.chatService.SaveToolCall(ctx, e.config.SessionID, event.ToolCall.ID, event.ToolCall.Name, event.ToolCall.Args, e.config.CompanyID)
	if err != nil {
		e.logger.Error("Failed to save tool call to database", zap.Error(err))
		// Continue streaming even if save fails
	} else {
		e.logger.Info("✅ Tool call saved successfully",
			zap.String("sessionId", e.config.SessionID.Hex()),
			zap.String("toolCallID", event.ToolCall.ID),
			zap.String("toolName", event.ToolCall.Name))
	}

	// Send tool call to output sink if client still connected
	if !e.config.OutputSink.IsDisconnected() {
		if err := e.config.OutputSink.SendToolCall(event.ToolCall.Name, event.ToolCall.ID, event.ToolCall.Args); err != nil {
			e.logger.Warn("Failed to send tool call to output sink",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.Error(err))
			// Don't return error - continue processing
		}
	}

	return nil
}

// handleToolResult processes a tool result stream event.
// Optionally processes the result (for size limits, formatting), then saves and streams it.
func (e *StreamExecutor) handleToolResult(ctx context.Context, event *aiservice.StreamEvent) error {
	// Apply custom tool result processing if configured
	var outputStr string
	var shouldSave bool = true
	var shouldStream bool = true

	if e.config.ToolResultProcessor != nil {
		outputStr, shouldSave, shouldStream = e.config.ToolResultProcessor(event.ToolResult.Name, event.ToolResult.Output)
	} else {
		// Default: Convert output to string
		if str, ok := event.ToolResult.Output.(string); ok {
			outputStr = str
		} else {
			// Marshal non-string outputs to JSON
			outputBytes, _ := json.Marshal(event.ToolResult.Output)
			outputStr = string(outputBytes)
		}
	}

	// Save tool result to database if allowed
	if shouldSave {
		// CRITICAL LOGGING: Track tool call ID through result save
		e.logger.Info("💾 Saving tool result to database",
			zap.String("sessionId", e.config.SessionID.Hex()),
			zap.String("toolResultID", event.ToolResult.ID),
			zap.String("toolName", event.ToolResult.Name),
			zap.Int("outputLength", len(outputStr)),
			zap.String("error", event.ToolResult.Error))

		// Check if ID is empty BEFORE saving
		if event.ToolResult.ID == "" {
			e.logger.Error("🚨 BUG DETECTED: ToolResult.ID is EMPTY before SaveToolResult!",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.String("toolName", event.ToolResult.Name),
				zap.Int64("durationMs", event.ToolResult.DurationMs))
		}

		_, err := e.chatService.SaveToolResult(
			ctx,
			e.config.SessionID,
			event.ToolResult.ID,
			event.ToolResult.Name,
			outputStr,
			event.ToolResult.Error,
			event.ToolResult.DurationMs,
			e.config.CompanyID,
		)
		if err != nil {
			e.logger.Error("Failed to save tool result to database", zap.Error(err))
			// Continue streaming even if save fails
		} else {
			e.logger.Info("✅ Tool result saved successfully",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.String("toolResultID", event.ToolResult.ID),
				zap.String("toolName", event.ToolResult.Name))
		}
	}

	// Send tool result to output sink if allowed and client still connected
	if shouldStream && !e.config.OutputSink.IsDisconnected() {
		if err := e.config.OutputSink.SendToolResult(
			event.ToolResult.ID,
			outputStr,
			event.ToolResult.Error,
			int(event.ToolResult.DurationMs),
		); err != nil {
			e.logger.Warn("Failed to send tool result to output sink",
				zap.String("sessionId", e.config.SessionID.Hex()),
				zap.Error(err))
			// Don't return error - continue processing
		}
	}

	return nil
}

// extractMessageID extracts the hex ID from a saved message.
// The saved message is wrapped in interface{} and may be *models.ChatMessage.
// Uses reflection to safely extract the ID field.
func extractMessageID(msg interface{}) string {
	if msg == nil {
		return ""
	}

	// Use reflection to get the ID field
	val := reflect.ValueOf(msg)

	// Handle pointer types
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ""
		}
		val = val.Elem()
	}

	// Look for ID field (case-insensitive check for common patterns)
	if val.Kind() == reflect.Struct {
		idField := val.FieldByName("ID")
		if idField.IsValid() {
			// Check if it's a primitive.ObjectID (has Hex method)
			if idField.Type().String() == "primitive.ObjectID" {
				hexMethod := idField.MethodByName("Hex")
				if hexMethod.IsValid() {
					results := hexMethod.Call(nil)
					if len(results) > 0 {
						return results[0].String()
					}
				}
			}
		}
	}

	return ""
}
