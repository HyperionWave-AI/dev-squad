package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/models"
	"hyper/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

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

// ChatWebSocketHandler handles WebSocket connections for real-time chat streaming
type ChatWebSocketHandler struct {
	chatService       *services.ChatService
	aiService         *aiservice.ChatService
	aiSettingsService *services.AISettingsService
	logger            *zap.Logger
}

// NewChatWebSocketHandler creates a new WebSocket handler with ai-service integration
func NewChatWebSocketHandler(chatService *services.ChatService, aiService *aiservice.ChatService, aiSettingsService *services.AISettingsService, logger *zap.Logger) *ChatWebSocketHandler {
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

	// Create context with cancellation
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Start message handling loop
	h.handleMessages(ctx, conn, sessionID, userID, companyID)
}

// handleMessages manages the WebSocket message loop with processing state
func (h *ChatWebSocketHandler) handleMessages(ctx context.Context, conn *websocket.Conn, sessionID primitive.ObjectID, userID, companyID string) {
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
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
	}()

	// Main message loop
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Context cancelled, closing WebSocket")
			close(done)
			return
		default:
			// Read message from client
			_, messageData, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.logger.Warn("WebSocket error", zap.Error(err))
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
			_, err = h.chatService.SaveMessage(ctx, sessionID, "user", userMsg.Content, companyID)
			if err != nil {
				h.logger.Error("Failed to save user message", zap.Error(err))
				h.sendError(conn, "Failed to save message")
				processingMutex.Lock()
				isProcessing = false
				processingMutex.Unlock()
				continue
			}

			// Stream AI response with tool execution events
			h.streamAIResponse(ctx, conn, sessionID, userMsg.Content, companyID)

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
		systemPromptText, _ = h.aiSettingsService.GetSystemPrompt(ctx, session.UserID, companyID)
		if systemPromptText != "" {
			h.logger.Info("Using global system prompt", zap.String("userId", session.UserID))
		}
	}

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
				// Text token from AI
				streamMsg := models.StreamMessage{
					Type:    "token",
					Content: event.Content,
				}
				if err := conn.WriteJSON(streamMsg); err != nil {
					h.logger.Error("Failed to send token to WebSocket", zap.Error(err))
					return
				}
				fullResponse += event.Content
				tokenCount++

			case aiservice.StreamEventToolCall:
				// AI is requesting a tool execution
				toolCallCount++

				// Save tool call to database
				_, err := h.chatService.SaveToolCall(ctx, sessionID, event.ToolCall.ID, event.ToolCall.Name, event.ToolCall.Args, companyID)
				if err != nil {
					h.logger.Error("Failed to save tool call to database", zap.Error(err))
					// Continue streaming even if save fails
				}

				// Send tool call to WebSocket client
				streamMsg := models.StreamMessage{
					Type: "tool_call",
					ToolCall: &models.ToolCallEvent{
						Tool: event.ToolCall.Name,
						Args: event.ToolCall.Args,
						ID:   event.ToolCall.ID,
					},
				}
				if err := conn.WriteJSON(streamMsg); err != nil {
					h.logger.Error("Failed to send tool call to WebSocket", zap.Error(err))
					return
				}

			case aiservice.StreamEventToolResult:
				// Tool execution completed

				// Save tool result to database
				_, err := h.chatService.SaveToolResult(ctx, sessionID, event.ToolResult.ID, event.ToolResult.Name, event.ToolResult.Output, event.ToolResult.Error, event.ToolResult.DurationMs, companyID)
				if err != nil {
					h.logger.Error("Failed to save tool result to database", zap.Error(err))
					// Continue streaming even if save fails
				}

				// Send tool result to WebSocket client
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
					h.logger.Error("Failed to send tool result to WebSocket", zap.Error(err))
					return
				}

			case aiservice.StreamEventError:
				// Error during processing
				h.logger.Error("AI service error during streaming", zap.String("error", event.Error))
				h.sendError(conn, "AI error: "+event.Error)
				return
			}
		}
	}

	// Step 8: Send completion message
	doneMsg := models.StreamMessage{
		Type:    "done",
		Content: "",
	}
	if err := conn.WriteJSON(doneMsg); err != nil {
		h.logger.Error("Failed to send done message", zap.Error(err))
		return
	}

	// Step 9: Save AI response to database
	_, err = h.chatService.SaveMessage(ctx, sessionID, "assistant", fullResponse, companyID)
	if err != nil {
		h.logger.Error("Failed to save AI response", zap.Error(err))
		h.sendError(conn, "Failed to save AI response")
		return
	}

	h.logger.Info("AI response streamed successfully",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("tokensStreamed", tokenCount),
		zap.Int("toolCalls", toolCallCount),
		zap.Int("responseLength", len(fullResponse)))
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
