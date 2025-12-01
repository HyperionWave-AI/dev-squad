package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatSession represents a conversation session with AI
type ChatSession struct {
	ID                    primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	UserID                string              `json:"userId" bson:"userId"`
	CompanyID             string              `json:"companyId" bson:"companyId"`
	Title                 string              `json:"title" bson:"title"`
	ContextTokenCount     int                 `json:"contextTokenCount" bson:"contextTokenCount"`                 // Total tokens used in session
	ContextPercentage     float64             `json:"contextPercentage" bson:"contextPercentage"`                 // Percentage of max context used
	ParentChatID          *primitive.ObjectID `json:"parentChatId,omitempty" bson:"parentChatId,omitempty"`      // For subchats - links to parent session
	ActiveSubagentID      *primitive.ObjectID `json:"activeSubagentId,omitempty" bson:"activeSubagentId,omitempty"` // For user-created subagents
	ActiveSubagentName    *string             `json:"activeSubagentName,omitempty" bson:"activeSubagentName,omitempty"` // For system subagents (go-dev, ui-dev, etc.)
	ErrorPreventionMode   bool                `json:"errorPreventionMode" bson:"errorPreventionMode"` // Toggle for validation plugin (default: false)
	ComplexityAnalysisMode bool               `json:"complexityAnalysisMode" bson:"complexityAnalysisMode"` // Toggle for complexity analysis and task splitting (default: false)
	CreatedAt             time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt             time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// ChatMessage represents a single message in a conversation
type ChatMessage struct {
	ID                  primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SessionID           primitive.ObjectID `json:"sessionId" bson:"sessionId"`
	Role                string             `json:"role" bson:"role"` // "user", "assistant", "system", "tool_call", "tool_result", "summary"
	Content             string             `json:"content" bson:"content"`
	IsPending           bool               `json:"isPending" bson:"isPending"`                         // Whether this message is still being processed (optimistic update)
	Timestamp           time.Time          `json:"timestamp" bson:"timestamp"`
	TokenCount          int                `json:"tokenCount" bson:"tokenCount"`                       // Estimated token count for this message
	IsArchived          bool               `json:"isArchived" bson:"isArchived"`                       // Whether this message has been archived
	IsSummary           bool               `json:"isSummary" bson:"isSummary"`                         // Whether this message is a summary of older messages
	OriginalMessageCount int               `json:"originalMessageCount" bson:"originalMessageCount"`   // Number of messages this summary represents

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

// UpdateSessionRequest represents the request to update a chat session
type UpdateSessionRequest struct {
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
	Type         string              `json:"type"` // "token", "tool_call", "tool_result", "tool_result_chunk", "done", "error", "session_created", "user_message", "message_saved", "system_notification"
	Content      string              `json:"content,omitempty"` // For session_created: contains new session ID
	Error        string              `json:"error,omitempty"`
	ToolCall     *ToolCallEvent      `json:"toolCall,omitempty"`
	ToolResult   *ToolResultEvent    `json:"toolResult,omitempty"`
	Notification *SystemNotification `json:"notification,omitempty"` // For system events (compaction, deflection, summarization)
}

// SystemNotification represents background system events for frontend display
type SystemNotification struct {
	Category string                 `json:"category"` // "compaction", "deflection", "summarization"
	Title    string                 `json:"title"`
	Message  string                 `json:"message"`
	Severity string                 `json:"severity"` // "info", "warning", "success"
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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

// ContextErrorCode represents different types of context errors
type ContextErrorCode string

const (
	ContextWarning    ContextErrorCode = "CONTEXT_WARNING"
	ContextCritical   ContextErrorCode = "CONTEXT_CRITICAL"
	ContextFull       ContextErrorCode = "CONTEXT_FULL"
	SummarizationFail ContextErrorCode = "SUMMARIZATION_FAILED"
)

// RecoveryOption represents an action the user can take to recover from context error
type RecoveryOption struct {
	Label       string `json:"label"`
	Action      string `json:"action"` // "archive", "new_chat", "summarize", "clear"
	Description string `json:"description"`
}

// ContextError represents a context-related error with recovery suggestions
type ContextError struct {
	Code             ContextErrorCode `json:"code"`
	Message          string           `json:"message"`
	Suggestion       string           `json:"suggestion"`
	RecoveryOptions  []RecoveryOption `json:"recoveryOptions"`
	ContextMetadata  *ContextMetadata `json:"contextMetadata,omitempty"`
	Timestamp        time.Time        `json:"timestamp"`
}

// ContextMetadata represents context usage information
type ContextMetadata struct {
	TokenCount    int       `json:"tokenCount"`
	MaxTokens     int       `json:"maxTokens"`
	PercentageUsed float64  `json:"percentageUsed"`
	IsNearLimit   bool      `json:"isNearLimit"`
	IsFull        bool      `json:"isFull"`
	LastUpdated   time.Time `json:"lastUpdated"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// STRUCTURED TOOL RESULT PRESENTATION TYPES
// ═══════════════════════════════════════════════════════════════════════════════

// KeyMatch represents a highlighted match within a result
type KeyMatch struct {
	Text      string `json:"text"`      // The matched text
	LineNum   int    `json:"lineNum"`   // Line number where match appears
	Context   string `json:"context"`   // Surrounding context (50 chars before/after)
	Relevance float64 `json:"relevance"` // How relevant this match is (0-1)
}

// FileSummary provides a brief overview of a file's contents
type FileSummary struct {
	FilePath    string `json:"filePath"`    // Full path to the file
	FileType    string `json:"fileType"`    // File extension/type (e.g., "go", "tsx", "json")
	LineCount   int    `json:"lineCount"`   // Total lines in file
	Description string `json:"description"` // Brief description of what file contains
	Module      string `json:"module"`      // Module/package name if applicable
}

// StructuredResult represents a single search result with enhanced presentation
type StructuredResult struct {
	ID          string       `json:"id"`          // Unique result ID
	FilePath    string       `json:"filePath"`    // File path
	Score       float64      `json:"score"`       // Relevance score (0-1)
	Text        string       `json:"text"`        // Full result text
	KeyMatches  []KeyMatch   `json:"keyMatches"` // Highlighted key matches
	Summary     FileSummary  `json:"summary"`     // File summary
	Rank        int          `json:"rank"`        // Rank within group (1-based)
	Priority    string       `json:"priority"`    // "high", "medium", "low"
}

// ResultGroup represents results grouped by file/module
type ResultGroup struct {
	GroupName   string              `json:"groupName"`   // File path or module name
	GroupType   string              `json:"groupType"`   // "file", "module", "package"
	ResultCount int                 `json:"resultCount"` // Number of results in group
	AvgScore    float64             `json:"avgScore"`    // Average relevance score
	Results     []StructuredResult  `json:"results"`     // Results in this group
	Priority    string              `json:"priority"`    // "high", "medium", "low"
}

// PriorityRecommendation suggests which files to focus on
type PriorityRecommendation struct {
	Rank        int    `json:"rank"`        // Priority rank (1 = highest)
	FilePath    string `json:"filePath"`    // Recommended file path
	Reason      string `json:"reason"`      // Why this file is recommended
	Score       float64 `json:"score"`      // Relevance score
	MatchCount  int    `json:"matchCount"`  // Number of matches in this file
	Confidence  float64 `json:"confidence"` // Confidence in recommendation (0-1)
}

// StructuredToolResultResponse represents the complete structured presentation
type StructuredToolResultResponse struct {
	// Summary statistics
	TotalResults    int `json:"totalResults"`    // Total number of results
	GroupCount      int `json:"groupCount"`      // Number of groups
	HighPriorityCount int `json:"highPriorityCount"` // Count of high-priority results

	// Organized results
	Groups []ResultGroup `json:"groups"` // Results grouped by file/module

	// AI decision support
	Recommendations []PriorityRecommendation `json:"recommendations"` // Suggested files to focus on
	
	// Metadata
	SearchQuery string `json:"searchQuery"` // Original search query
	ExecutionTime int64 `json:"executionTime"` // Query execution time in ms
	Timestamp   time.Time `json:"timestamp"`   // When results were generated
}
