package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatSession represents a conversation session with AI
type ChatSession struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    string             `json:"userId" bson:"userId"`
	CompanyID string             `json:"companyId" bson:"companyId"`
	Title     string             `json:"title" bson:"title"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`

	// Subchat fields for parallel agent workflows
	ParentChatID   *primitive.ObjectID `json:"parentChatId,omitempty" bson:"parentChatId,omitempty"`     // Reference to parent chat (if subchat)
	SubagentName   string              `json:"subagentName,omitempty" bson:"subagentName,omitempty"`     // Assigned subagent name
	AssignedTaskID string              `json:"assignedTaskId,omitempty" bson:"assignedTaskId,omitempty"` // UUID reference to agent task
	AssignedTodoID string              `json:"assignedTodoId,omitempty" bson:"assignedTodoId,omitempty"` // UUID reference to specific TODO
}

// ChatMessage represents a single message in a conversation
type ChatMessage struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SessionID primitive.ObjectID `json:"sessionId" bson:"sessionId"`
	Role      string             `json:"role" bson:"role"` // "user", "assistant", "system", "tool_call", "tool_result"
	Content   string             `json:"content" bson:"content"`
	Timestamp time.Time          `json:"timestamp" bson:"timestamp"`

	// Tool-related fields (optional, only for tool_call and tool_result roles)
	ToolCall   *ToolCallData   `json:"toolCall,omitempty" bson:"toolCall,omitempty"`
	ToolResult *ToolResultData `json:"toolResult,omitempty" bson:"toolResult,omitempty"`
}

// ToolCallData represents tool call information stored in database
type ToolCallData struct {
	ID   string                 `json:"id" bson:"id"`
	Name string                 `json:"name" bson:"name"`
	Args map[string]interface{} `json:"args" bson:"args"`
}

// ToolResultData represents tool result information stored in database
type ToolResultData struct {
	ID         string      `json:"id" bson:"id"`
	Name       string      `json:"name" bson:"name"` // Tool name for reference
	Output     interface{} `json:"output" bson:"output"`
	Error      string      `json:"error,omitempty" bson:"error,omitempty"`
	DurationMs int64       `json:"durationMs" bson:"durationMs"`
}

// CreateSessionRequest represents the request to create a new chat session
type CreateSessionRequest struct {
	Title string `json:"title" binding:"required"`
}

// GetMessagesResponse represents paginated message response
type GetMessagesResponse struct {
	Messages   []ChatMessage `json:"messages"`
	Total      int64         `json:"total"`
	Limit      int           `json:"limit"`
	Offset     int           `json:"offset"`
	HasMore    bool          `json:"hasMore"`
}

// SendMessageRequest represents a message sent from user
type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// StreamMessage represents a streaming AI response message
type StreamMessage struct {
	Type       string            `json:"type"` // "token", "tool_call", "tool_result", "tool_result_chunk", "done", "error"
	Content    string            `json:"content,omitempty"`
	Error      string            `json:"error,omitempty"`
	ToolCall   *ToolCallEvent    `json:"toolCall,omitempty"`
	ToolResult *ToolResultEvent  `json:"toolResult,omitempty"`
}

// ToolCallEvent represents an AI tool call event
type ToolCallEvent struct {
	Tool string                 `json:"tool"`
	Args map[string]interface{} `json:"args"`
	ID   string                 `json:"id"`
}

// ToolResultEvent represents the result of a tool execution
type ToolResultEvent struct {
	ID         string      `json:"id"`
	Result     interface{} `json:"result"`
	Error      string      `json:"error,omitempty"`
	DurationMs int         `json:"durationMs"`
}

// ToolResultChunk represents a chunk of a large tool result
type ToolResultChunk struct {
	ID    string `json:"id"`
	Chunk string `json:"chunk"`
	Index int    `json:"index"`
	Total int    `json:"total"`
	Done  bool   `json:"done"`
}
