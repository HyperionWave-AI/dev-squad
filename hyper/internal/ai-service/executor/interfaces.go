package executor

import (
	"context"

	aiservice "hyper/internal/ai-service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatServiceInterface defines the methods needed from the chat service for message operations.
// This abstraction allows the executor to work with any chat service implementation
// without tight coupling to specific implementations.
type ChatServiceInterface interface {
	// SaveMessage saves a message to the chat session
	SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*interface{}, error)

	// SaveToolCall saves a tool call request to the chat session
	SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName string, args map[string]interface{}, companyID string) (*interface{}, error)

	// SaveToolResult saves a tool execution result to the chat session
	SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName, output, errorMsg string, durationMs int64, companyID string) (*interface{}, error)
}

// AIServiceInterface defines the streaming methods needed from the AI service.
// This abstraction allows the executor to work with different AI service implementations.
type AIServiceInterface interface {
	// StreamChatWithTools streams AI response with full tool access
	StreamChatWithTools(ctx context.Context, messages []aiservice.Message, maxToolCalls int) (<-chan aiservice.StreamEvent, error)

	// StreamChatWithToolsFiltered streams AI response with filtered tool access
	StreamChatWithToolsFiltered(ctx context.Context, messages []aiservice.Message, maxToolCalls int, allowedTools []string) (<-chan aiservice.StreamEvent, error)

	// GetConfig returns AI configuration
	GetConfig() *aiservice.AIConfig
}

// CompletionValidatorFunc is a function type that validates when streaming is complete.
// Returns true if streaming should stop, false to continue.
// Used for custom completion logic (e.g., checking for specific markers, max iterations).
type CompletionValidatorFunc func(fullResponse string, toolCallCount int) bool
