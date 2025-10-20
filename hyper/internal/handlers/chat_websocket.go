package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/tools"
	"hyper/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// DefaultSystemPrompt is the default system prompt for Chat coordinator - guides autonomous behavior
// Exported for use by AI settings service
const DefaultSystemPrompt = `You are a COORDINATOR AI assistant that orchestrates development work through specialist subagents. You delegate implementation work - you do NOT implement yourself.

🎯 YOUR ROLE (CRITICAL):
- **ORCHESTRATOR**: You coordinate work, not implement it
- **CONTEXT PROVIDER**: You gather context and create detailed task specifications
- **DELEGATOR**: You launch specialist subagents (ui-dev, go-dev, sre, etc.) to do the actual work
- **NEVER IMPLEMENT CODE YOURSELF** - use execute_subagent tool to delegate to specialists

🚨 GOLDEN PATH WORKFLOW (MANDATORY):
When user requests code changes, modifications, or implementations:

1. **Create Human Task** (always first):
   - Use coordinator_create_human_task({ prompt: "user's exact request" })

2. **Gather Context** (ALWAYS use semantic search FIRST):
   - ✅ **REQUIRED FIRST STEP**: Use code_index_search("feature description") to find relevant files
   - ✅ Read ONLY the top 1-2 results from code_index_search
   - ✅ Provide exact file paths + line numbers to subagent
   - ❌ **NEVER** start with list_directory or blind file exploration
   - ❌ **NEVER** read more than 3 files total
   - DO NOT explore extensively - code_index_search gives you the answers!

3. **Create Agent Task** (with rich context):
   - Use coordinator_create_agent_task({
       humanTaskId: "<from step 1>",
       agentName: "<ui-dev|go-dev|go-mcp-dev|sre|ui-tester|...>",
       role: "50-100 word mission statement",
       contextSummary: "150-250 words: WHAT to change, WHERE (file:line), HOW (patterns/examples)",
       filesModified: ["exact/file/paths.tsx"],
       todos: [{
         description: "specific task",
         filePath: "exact/path.tsx",
         contextHint: "50-100w guidance on how to implement"
       }]
     })
   - **IMPORTANT**: Put ≥80% of needed info in contextSummary and contextHints
   - Provide specific file paths and line numbers when known
   - Include patterns/examples from your context gathering

4. **Execute Subagent** (delegate the work):
   - Use execute_subagent tool with the agentName and task details
   - The subagent will do ALL the actual implementation work
   - DO NOT do any code modifications yourself

5. **Monitor Progress** (coordinator_list_agent_tasks to check status)

⚠️ CRITICAL RULES - WHEN TO DELEGATE:
- User says "start execution" / "execute now" / "implement this" → **IMMEDIATE**: Create agent task + execute subagent
- User asks to "add feature" / "fix bug" / "modify code" → Follow Golden Path workflow
- User asks "what should we do?" / "explain X" → Answer directly (no delegation needed)
- **MAX 2-3 MINUTES** from user request to subagent execution
- **MAX 3 FILE READS** before creating agent task
- **NEVER IMPLEMENT CODE YOURSELF** - always delegate to execute_subagent

🚨 ANTI-LOOP RULES (PREVENT INFINITE EXPLORATION):
1. **NEVER call the same tool with identical arguments more than ONCE**
2. **If a tool returns a result, USE it immediately** - don't re-call expecting different output
3. **If you find yourself exploring/reading for >2 minutes without creating task → STOP**
4. **Circuit breaker**: System stops you after 3 identical calls - you're looping

Examples of BAD patterns (causes loops):
❌ list_directory(./components) → list_directory(./components) [LOOP!]
❌ read_file(config.ts) → read_file(config.ts) [LOOP!]
❌ Spending 5 minutes reading files before creating agent task [OVER-EXPLORING!]

Examples of GOOD patterns:
✅ code_index_search("delete button") → read 1-2 relevant files → create_agent_task → execute_subagent
✅ User: "start execution" → create_agent_task (with prior context) → execute_subagent [IMMEDIATE DELEGATION]
✅ list_directory(./components) → see TaskCard.tsx → read_file(TaskCard.tsx) → create task [USE RESULTS]

KEY TOOLS (in priority order):
- **code_index_search**: 🔥 PRIMARY TOOL - Find code semantically (use FIRST, before any file reads!)
- **coordinator_create_human_task**: Record user requests
- **coordinator_create_agent_task**: Create detailed task specs for subagents
- **execute_subagent**: Delegate implementation work to specialists
- **read_file**: Read specific files (ONLY after code_index_search, max 3 total)
- **list_directory**: LAST RESORT - only if code_index_search returns nothing
- **coordinator_list_agent_tasks**: Monitor progress

COORDINATOR MINDSET:
✅ "Let me create a task and delegate to ui-dev specialist"
✅ "I'll gather context and launch the appropriate subagent"
✅ "I found the files - now creating agent task with specifics"

❌ "Let me implement this myself using write_file"
❌ "I'll explore all the files first to understand everything"
❌ "Let me make the code changes directly"

Be efficient: Gather minimal context → Create detailed task → Delegate to specialist → Monitor progress.`

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// TODO: Restrict in production based on allowed origins
		return true
	},
}

// ChatServiceInterface defines the interface for chat service operations
type ChatServiceInterface interface {
	GetSession(ctx context.Context, sessionID primitive.ObjectID, companyID string) (*models.ChatSession, error)
	GetSessionMessages(ctx context.Context, sessionID primitive.ObjectID) ([]models.ChatMessage, error)
	SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*models.ChatMessage, error)
	SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, id, name string, args map[string]interface{}, companyID string) (*models.ChatMessage, error)
	SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, id, name string, output interface{}, errorMsg string, durationMs int64, companyID string) (*models.ChatMessage, error)
}

// AIServiceInterface defines the interface for AI service operations
type AIServiceInterface interface {
	StreamChatWithTools(ctx context.Context, messages []aiservice.Message, maxToolCalls int) (<-chan aiservice.StreamEvent, error)
	GetConfig() *aiservice.AIConfig
}

// AISettingsServiceInterface defines the interface for AI settings service operations
type AISettingsServiceInterface interface {
	GetSubagent(ctx context.Context, id primitive.ObjectID, companyID string) (*models.Subagent, error)
	GetSystemPrompt(ctx context.Context, userID, companyID string) (string, error)
}

// ChatWebSocketHandler handles WebSocket connections for real-time chat streaming
type ChatWebSocketHandler struct {
	chatService       ChatServiceInterface
	aiService         AIServiceInterface
	aiSettingsService AISettingsServiceInterface
	logger            *zap.Logger
}

// NewChatWebSocketHandler creates a new WebSocket handler with ai-service integration
func NewChatWebSocketHandler(chatService ChatServiceInterface, aiService AIServiceInterface, aiSettingsService AISettingsServiceInterface, logger *zap.Logger) *ChatWebSocketHandler {
	return &ChatWebSocketHandler{
		chatService:       chatService,
		aiService:         aiService,
		aiSettingsService: aiSettingsService,
		logger:            logger,
	}
}

// extractAuthFromContext extracts authentication from Gin context (set by JWT middleware)
// Falls back to query parameters for backward compatibility
// GET /api/v1/chat/stream?sessionId=xxx
func (h *ChatWebSocketHandler) extractAuthFromContext(c *gin.Context) (string, string, error) {
	// First try to get from context (set by OptionalJWTMiddleware)
	if userIDVal, exists := c.Get("userId"); exists {
		if companyIDVal, exists := c.Get("companyId"); exists {
			userID, ok1 := userIDVal.(string)
			companyID, ok2 := companyIDVal.(string)
			if ok1 && ok2 && userID != "" && companyID != "" {
				return userID, companyID, nil
			}
		}
	}

	// Fallback to query parameters for backward compatibility
	userID := c.Query("userId")
	companyID := c.Query("companyId")

	if userID == "" || companyID == "" {
		return "", "", fmt.Errorf("missing authentication parameters")
	}

	return userID, companyID, nil
}

// HandleChatWebSocket handles WebSocket connections for chat streaming
// GET /api/v1/chat/stream?sessionId=xxx
func (h *ChatWebSocketHandler) HandleChatWebSocket(c *gin.Context) {
	// Extract authentication from context (set by middleware)
	userID, companyID, err := h.extractAuthFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	// Get session ID from query
	sessionIDStr := c.Query("sessionId")
	if sessionIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sessionId parameter"})
		return
	}

	sessionID, err := primitive.ObjectIDFromHex(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sessionId"})
		return
	}

	// Verify session exists and user has access
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID, companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found or access denied"})
		return
	}

	// Verify session belongs to user
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: session belongs to different user"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}
	defer conn.Close()

	h.logger.Info("WebSocket connection established",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("userId", userID))

	// Create background context for AI processing (not tied to HTTP lifecycle)
	aiCtx := context.Background()
	aiCtx, aiCancel := context.WithTimeout(aiCtx, 10*time.Minute) // Generous timeout for multi-tool AI ops
	defer aiCancel()

	// Keep HTTP context for connection monitoring
	httpCtx := c.Request.Context()

	// Pass both contexts to handleMessages
	h.handleMessages(aiCtx, httpCtx, conn, sessionID, userID, companyID)
}

// handleMessages manages the WebSocket message loop with processing state
func (h *ChatWebSocketHandler) handleMessages(aiCtx context.Context, httpCtx context.Context, conn *websocket.Conn, sessionID primitive.ObjectID, userID, companyID string) {
	// Set read deadline for ping/pong
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Channel for handling graceful shutdown
	done := make(chan struct{})

	// Processing state to prevent concurrent messages during AI response
	isProcessing := false
	var processingMutex sync.Mutex

	// Goroutine for sending pings
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					h.logger.Warn("Failed to send ping", zap.Error(err))
					return
				}
			case <-httpCtx.Done():
				return
			case <-done:
				return
			}
		}
	}()

	// Main message loop
	for {
		select {
		case <-httpCtx.Done():
			h.logger.Info("HTTP context cancelled, closing WebSocket")
			close(done)
			return
		default:
			// Read message from client
			_, messageData, err := conn.ReadMessage()
			if err != nil {
				// Check if this is a normal disconnection
				if websocket.IsCloseError(err,
					websocket.CloseGoingAway,           // 1001: browser navigation
					websocket.CloseAbnormalClosure,     // 1006: abnormal closure
					websocket.CloseNormalClosure,       // 1000: normal closure
					websocket.CloseNoStatusReceived) {  // 1005: no status (browser refresh/close)
					h.logger.Debug("Client disconnected from WebSocket",
						zap.String("sessionId", sessionID.Hex()),
						zap.String("reason", err.Error()))
				} else {
					// Truly unexpected error
					h.logger.Warn("WebSocket unexpected error",
						zap.String("sessionId", sessionID.Hex()),
						zap.Error(err))
				}
				close(done)
				return
			}

			// Parse user message
			var userMsg models.SendMessageRequest
			if err := json.Unmarshal(messageData, &userMsg); err != nil {
				h.sendError(conn, "Invalid message format")
				continue
			}

			// Check if already processing a message
			processingMutex.Lock()
			if isProcessing {
				processingMutex.Unlock()
				h.logger.Warn("Message rejected - AI response in progress",
					zap.String("sessionId", sessionID.Hex()),
					zap.String("userId", userID))
				h.sendError(conn, "Please wait for current response to complete before sending another message")
				continue
			}
			isProcessing = true
			processingMutex.Unlock()

			// Save user message to database
			_, err = h.chatService.SaveMessage(aiCtx, sessionID, "user", userMsg.Content, companyID)
			if err != nil {
				h.logger.Error("Failed to save user message", zap.Error(err))
				h.sendError(conn, "Failed to save message")
				processingMutex.Lock()
				isProcessing = false
				processingMutex.Unlock()
				continue
			}

			// Stream AI response with tool execution events
			h.streamAIResponse(aiCtx, conn, sessionID, userMsg.Content, companyID)

			// Reset processing state after response complete
			processingMutex.Lock()
			isProcessing = false
			processingMutex.Unlock()
		}
	}
}

// streamAIResponse streams AI response with tool execution events back to client using ai-service
func (h *ChatWebSocketHandler) streamAIResponse(ctx context.Context, conn *websocket.Conn, sessionID primitive.ObjectID, userMessage, companyID string) {
	h.logger.Info("Streaming AI response via ai-service",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("userMessage", userMessage))

	// Step 1: Get session to check for active subagent
	session, err := h.chatService.GetSession(ctx, sessionID, companyID)
	if err != nil {
		h.logger.Error("Failed to retrieve session", zap.Error(err))
		h.sendError(conn, "Failed to retrieve session")
		return
	}

	// Step 2: Determine active agent and fetch system prompt
	var systemPromptText string
	if session.ActiveSubagentID != nil {
		// Using custom subagent - fetch subagent's prompt
		subagent, err := h.aiSettingsService.GetSubagent(ctx, *session.ActiveSubagentID, companyID)
		if err == nil && subagent != nil {
			systemPromptText = subagent.SystemPrompt
			h.logger.Info("Using subagent prompt",
				zap.String("subagentId", session.ActiveSubagentID.Hex()),
				zap.String("subagentName", subagent.Name))
		} else {
			h.logger.Warn("Failed to fetch subagent, falling back to system prompt", zap.Error(err))
		}
	}

	// If no subagent or subagent fetch failed, use global system prompt
	if systemPromptText == "" {
		h.logger.Debug("Attempting to retrieve global system prompt",
			zap.String("userId", session.UserID),
			zap.String("companyId", companyID),
			zap.String("sessionId", sessionID.Hex()))

		var promptErr error
		systemPromptText, promptErr = h.aiSettingsService.GetSystemPrompt(ctx, session.UserID, companyID)

		if promptErr != nil {
			h.logger.Warn("Failed to retrieve system prompt",
				zap.Error(promptErr),
				zap.String("userId", session.UserID),
				zap.String("companyId", companyID))
		} else if systemPromptText != "" {
			h.logger.Info("Using global system prompt",
				zap.String("userId", session.UserID),
				zap.Int("promptLength", len(systemPromptText)))
		} else {
			// No custom prompt configured - use default autonomous prompt
			systemPromptText = DefaultSystemPrompt
			h.logger.Info("Using default autonomous system prompt",
				zap.String("userId", session.UserID),
				zap.String("companyId", companyID),
				zap.Int("promptLength", len(systemPromptText)))
		}
	}

	// ALWAYS append critical system guidance (filesystem context + anti-loop rules + session context)
	// This is appended regardless of custom prompts to ensure consistent behavior
	projectRoot := tools.GetProjectRoot()
	criticalGuidance := fmt.Sprintf(`

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CRITICAL SYSTEM BEHAVIOR (NON-OVERRIDABLE)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

SESSION CONTEXT:
- **CURRENT CHAT SESSION ID**: %s
- **IMPORTANT**: When using execute_subagent tool, ALWAYS use parentChatId: "%s"
- DO NOT ask the user for the session ID - it is provided above
- This session ID links subagent work back to this conversation

FILESYSTEM CONTEXT:
- **PROJECT ROOT**: %s
- **PATH FORMAT**: ALWAYS use Unix/Mac forward slashes (/) - NEVER backslashes (\)
- **CORRECT**: %s/ui/src/file.tsx OR ./ui/src/file.tsx
- **FORBIDDEN**: C:\Users\... OR C:\\Users\... (Windows paths)
- Prefer relative paths from project root: ./ui/src/main.tsx
- Bash working directory: %s (automatically set)
- System directories BLOCKED: /etc, /var, /sys, /usr

TOOL USAGE RULES - PREVENT INFINITE LOOPS:
1. **NEVER call the same tool with identical arguments consecutively**
2. **If a tool returns a result, USE it** - don't re-call expecting different output
3. **If stuck, change approach** - try different tool or different arguments
4. **Circuit breaker**: System stops you after 3 identical calls in 5 attempts

❌ BAD PATTERN (causes circuit breaker):
  list_directory(./components) → list_directory(./components) → list_directory(./components)

✅ GOOD PATTERN (smart exploration):
  list_directory(./components) → find what you need → read_file(specific_file)

✅ If stuck, try different approach:
  list_directory fails → try bash("find . -name pattern") OR code_index_search

**When user gives you an explicit file path, just read it - don't explore directories!**

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, sessionID.Hex(), sessionID.Hex(), projectRoot, projectRoot, projectRoot)
	systemPromptText += criticalGuidance

	// Step 3: Get conversation history for context
	messages, err := h.chatService.GetSessionMessages(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to retrieve conversation history", zap.Error(err))
		h.sendError(conn, "Failed to retrieve conversation history")
		return
	}

	h.logger.Debug("Retrieved conversation history",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("messageCount", len(messages)))

	// Step 4: Convert MongoDB messages to LangChain format
	langchainMessages := aiservice.ConvertToLangChainMessages(messages)

	// Step 5: Inject system prompt as first message (if exists)
	if systemPromptText != "" {
		// Prepend system message
		systemMessage := aiservice.Message{
			Role:    "system",
			Content: systemPromptText,
		}
		langchainMessages = append([]aiservice.Message{systemMessage}, langchainMessages...)

		h.logger.Debug("Injected system prompt",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("promptLength", len(systemPromptText)))
	}

	// Step 6: Stream AI response via ai-service with tool support
	maxToolCalls := h.aiService.GetConfig().MaxToolCalls
	aiStream, err := h.aiService.StreamChatWithTools(ctx, langchainMessages, maxToolCalls)
	if err != nil {
		h.logger.Error("Failed to get AI response", zap.Error(err))
		h.sendError(conn, "Failed to get AI response: "+err.Error())
		return
	}

	// Step 7: Stream mixed content (tokens and tool events) to WebSocket client
	fullResponse := ""
	tokenCount := 0
	toolCallCount := 0
	clientDisconnected := false // Track client disconnect state

	for event := range aiStream {
		select {
		case <-ctx.Done():
			h.logger.Info("Context cancelled during streaming",
				zap.String("sessionId", sessionID.Hex()),
				zap.Int("tokensStreamed", tokenCount),
				zap.Int("toolCalls", toolCallCount))
			return
		default:
			// Handle different event types
			switch event.Type {
			case aiservice.StreamEventToken:
				// Accumulate response even if client disconnected
				fullResponse += event.Content
				tokenCount++

				// Try to send to WebSocket if client still connected
				if !clientDisconnected {
					streamMsg := models.StreamMessage{
						Type:    "token",
						Content: event.Content,
					}
					if err := conn.WriteJSON(streamMsg); err != nil {
						// Check if this is a normal disconnection (client closed browser/refreshed)
						if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
							h.logger.Debug("Client disconnected during streaming - continuing processing in background",
								zap.String("sessionId", sessionID.Hex()),
								zap.Int("tokensStreamed", tokenCount))
							clientDisconnected = true // Set flag and continue processing
						} else {
							h.logger.Warn("Failed to send token to WebSocket - continuing processing",
								zap.String("sessionId", sessionID.Hex()),
								zap.Error(err))
							clientDisconnected = true // Assume client is gone
						}
						// Don't return - continue processing to save to database
					}
				}

			case aiservice.StreamEventToolCall:
				// AI is requesting a tool execution
				toolCallCount++

				// Save tool call to database (always, even if client disconnected)
				_, err := h.chatService.SaveToolCall(ctx, sessionID, event.ToolCall.ID, event.ToolCall.Name, event.ToolCall.Args, companyID)
				if err != nil {
					h.logger.Error("Failed to save tool call to database", zap.Error(err))
					// Continue streaming even if save fails
				}

				// Send tool call to WebSocket client if still connected
				if !clientDisconnected {
					streamMsg := models.StreamMessage{
						Type: "tool_call",
						ToolCall: &models.ToolCallEvent{
							Tool: event.ToolCall.Name,
							Args: event.ToolCall.Args,
							ID:   event.ToolCall.ID,
						},
					}
					if err := conn.WriteJSON(streamMsg); err != nil {
						// Check if this is a normal disconnection
						if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
							h.logger.Debug("Client disconnected during tool call streaming - continuing processing",
								zap.String("sessionId", sessionID.Hex()))
							clientDisconnected = true
						} else {
							h.logger.Warn("Failed to send tool call to WebSocket - continuing processing",
								zap.String("sessionId", sessionID.Hex()),
								zap.Error(err))
							clientDisconnected = true
						}
						// Don't return - continue processing
					}
				}

			case aiservice.StreamEventToolResult:
				// Tool execution completed

				// Convert output to string for database storage
				outputStr := ""
				if event.ToolResult.Output != nil {
					if str, ok := event.ToolResult.Output.(string); ok {
						outputStr = str
					} else {
						// Marshal non-string outputs to JSON
						outputBytes, _ := json.Marshal(event.ToolResult.Output)
						outputStr = string(outputBytes)
					}
				}

				// Save tool result to database (always, even if client disconnected)
				_, err := h.chatService.SaveToolResult(ctx, sessionID, event.ToolResult.ID, event.ToolResult.Name, outputStr, event.ToolResult.Error, event.ToolResult.DurationMs, companyID)
				if err != nil {
					h.logger.Error("Failed to save tool result to database", zap.Error(err))
					// Continue streaming even if save fails
				}

				// Send tool result to WebSocket client if still connected
				if !clientDisconnected {
					streamMsg := models.StreamMessage{
						Type: "tool_result",
						ToolResult: &models.ToolResultEvent{
							ID:         event.ToolResult.ID,
							Result:     event.ToolResult.Output,
							Error:      event.ToolResult.Error,
							DurationMs: int(event.ToolResult.DurationMs),
						},
					}
					if err := conn.WriteJSON(streamMsg); err != nil {
						// Check if this is a normal disconnection
						if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
							h.logger.Debug("Client disconnected during tool result streaming - continuing processing",
								zap.String("sessionId", sessionID.Hex()))
							clientDisconnected = true
						} else {
							h.logger.Warn("Failed to send tool result to WebSocket - continuing processing",
								zap.String("sessionId", sessionID.Hex()),
								zap.Error(err))
							clientDisconnected = true
						}
						// Don't return - continue processing
					}
				}

			case aiservice.StreamEventError:
				// Error during processing
				h.logger.Error("AI service error during streaming", zap.String("error", event.Error))
				h.sendError(conn, "AI error: "+event.Error)
				return
			}
		}
	}

	// Step 8: Send completion message (if client still connected)
	if !clientDisconnected {
		doneMsg := models.StreamMessage{
			Type:    "done",
			Content: "",
		}
		if err := conn.WriteJSON(doneMsg); err != nil {
			// Check if this is a normal disconnection
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				h.logger.Debug("Client disconnected before completion message",
					zap.String("sessionId", sessionID.Hex()))
			} else {
				h.logger.Warn("Failed to send done message", zap.Error(err))
			}
			clientDisconnected = true
			// Don't return - continue to save response to database
		}
	}

	// Step 9: Save AI response to database (ALWAYS, even if client disconnected)
	_, err = h.chatService.SaveMessage(ctx, sessionID, "assistant", fullResponse, companyID)
	if err != nil {
		h.logger.Error("Failed to save AI response", zap.Error(err))
		// Only try to send error if client still connected
		if !clientDisconnected {
			h.sendError(conn, "Failed to save AI response")
		}
		return
	}

	if clientDisconnected {
		h.logger.Info("AI response completed in background after client disconnect",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("tokensStreamed", tokenCount),
			zap.Int("toolCalls", toolCallCount),
			zap.Int("responseLength", len(fullResponse)))
	} else {
		h.logger.Info("AI response streamed successfully",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("tokensStreamed", tokenCount),
			zap.Int("toolCalls", toolCallCount),
			zap.Int("responseLength", len(fullResponse)))
	}
}

// streamToolResult streams tool result to WebSocket with chunking for large outputs
// Results larger than 10KB are split into chunks to prevent WebSocket message size limits
func (h *ChatWebSocketHandler) streamToolResult(conn *websocket.Conn, result models.ToolResultEvent) error {
	// Serialize result to JSON to check size
	resultJSON, err := json.Marshal(result.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal tool result: %w", err)
	}

	const maxChunkSize = 10 * 1024 // 10KB

	// If result is small enough, send as single message
	if len(resultJSON) <= maxChunkSize {
		streamMsg := models.StreamMessage{
			Type:       "tool_result",
			ToolResult: &result,
		}
		if err := conn.WriteJSON(streamMsg); err != nil {
			return fmt.Errorf("failed to send tool result: %w", err)
		}
		return nil
	}

	// Large result - split into chunks
	h.logger.Info("Chunking large tool result",
		zap.String("toolId", result.ID),
		zap.Int("totalBytes", len(resultJSON)))

	resultStr := string(resultJSON)
	totalChunks := (len(resultStr) + maxChunkSize - 1) / maxChunkSize

	for i := 0; i < totalChunks; i++ {
		start := i * maxChunkSize
		end := start + maxChunkSize
		if end > len(resultStr) {
			end = len(resultStr)
		}

		chunk := models.StreamMessage{
			Type: "tool_result_chunk",
			ToolResult: &models.ToolResultEvent{
				ID: result.ID,
				Result: models.ToolResultChunk{
					ID:    result.ID,
					Chunk: resultStr[start:end],
					Index: i,
					Total: totalChunks,
					Done:  i == totalChunks-1,
				},
				Error:      result.Error,
				DurationMs: result.DurationMs,
			},
		}

		if err := conn.WriteJSON(chunk); err != nil {
			return fmt.Errorf("failed to send chunk %d/%d: %w", i+1, totalChunks, err)
		}

		h.logger.Debug("Sent tool result chunk",
			zap.String("toolId", result.ID),
			zap.Int("chunk", i+1),
			zap.Int("total", totalChunks))
	}

	return nil
}

// sendError sends an error message to the WebSocket client
func (h *ChatWebSocketHandler) sendError(conn *websocket.Conn, errorMsg string) {
	errMsg := models.StreamMessage{
		Type:  "error",
		Error: errorMsg,
	}
	if err := conn.WriteJSON(errMsg); err != nil {
		h.logger.Error("Failed to send error message", zap.Error(err))
	}
}
