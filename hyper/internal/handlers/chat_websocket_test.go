package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/mcp/storage"
	"hyper/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// MockChatService is a mock implementation of ChatService for testing
type MockChatService struct {
	mock.Mock
}

func (m *MockChatService) GetSession(ctx context.Context, sessionID primitive.ObjectID, companyID string) (*models.ChatSession, error) {
	args := m.Called(ctx, sessionID, companyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChatSession), args.Error(1)
}

func (m *MockChatService) GetSessionMessages(ctx context.Context, sessionID primitive.ObjectID) ([]models.ChatMessage, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChatMessage), args.Error(1)
}

func (m *MockChatService) SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*models.ChatMessage, error) {
	args := m.Called(ctx, sessionID, role, content, companyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChatMessage), args.Error(1)
}

func (m *MockChatService) SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, id, name string, args map[string]interface{}, companyID string) (*models.ChatMessage, error) {
	callArgs := m.Called(ctx, sessionID, id, name, args, companyID)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(*models.ChatMessage), callArgs.Error(1)
}

func (m *MockChatService) SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, id, name string, output interface{}, errorMsg string, durationMs int64, companyID string) (*models.ChatMessage, error) {
	callArgs := m.Called(ctx, sessionID, id, name, output, errorMsg, durationMs, companyID)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(*models.ChatMessage), callArgs.Error(1)
}

// MockAIService is a mock implementation of AI service for testing
type MockAIService struct {
	mock.Mock
	responseChannel chan string
}

func (m *MockAIService) StreamChat(ctx context.Context, messages []aiservice.Message) (<-chan string, error) {
	args := m.Called(ctx, messages)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan string), args.Error(1)
}

func (m *MockAIService) StreamChatWithTools(ctx context.Context, messages []aiservice.Message, maxToolCalls int) (<-chan aiservice.StreamEvent, error) {
	// Convert simple token channel to StreamEvent channel for backward compatibility
	args := m.Called(ctx, messages, maxToolCalls)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	tokenChan := args.Get(0).(<-chan string)
	eventChan := make(chan aiservice.StreamEvent, 10)

	go func() {
		defer close(eventChan)
		for token := range tokenChan {
			eventChan <- aiservice.StreamEvent{
				Type:    aiservice.StreamEventToken,
				Content: token,
			}
		}
	}()

	return eventChan, nil
}

func (m *MockAIService) StreamChatWithToolsFiltered(ctx context.Context, messages []aiservice.Message, maxToolCalls int, allowedToolNames []string) (<-chan aiservice.StreamEvent, error) {
	// For testing, just delegate to StreamChatWithTools
	return m.StreamChatWithTools(ctx, messages, maxToolCalls)
}

func (m *MockAIService) GetConfig() *aiservice.AIConfig {
	return &aiservice.AIConfig{
		Provider:     "mock",
		Model:        "test-model",
		MaxToolCalls: 3,
	}
}

func (m *MockAIService) GetAllowedToolsForDirectSubagent() []string {
	return []string{} // Return empty list for test
}

// MockAISettingsService is a mock implementation of AISettingsService for testing
type MockAISettingsService struct {
	mock.Mock
}

// MockSubchatStorage is a mock implementation of SubchatStorageInterface for testing
type MockSubchatStorage struct {
	mock.Mock
}

func (m *MockSubchatStorage) GetSubagent(name string) (*storage.Subagent, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.Subagent), args.Error(1)
}

func (m *MockAISettingsService) GetSubagent(ctx context.Context, id primitive.ObjectID, companyID string) (*models.Subagent, error) {
	args := m.Called(ctx, id, companyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subagent), args.Error(1)
}

func (m *MockAISettingsService) GetSystemPrompt(ctx context.Context, userID, companyID string) (string, error) {
	args := m.Called(ctx, userID, companyID)
	return args.String(0), args.Error(1)
}

// setupTestServer creates a test HTTP server with WebSocket handler
func setupTestServer(t *testing.T, chatService *MockChatService, aiService *MockAIService) (*httptest.Server, *ChatWebSocketHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger, _ := zap.NewDevelopment()
	aiSettingsService := new(MockAISettingsService)
	subchatStorage := new(MockSubchatStorage)

	// Mock aiSettingsService to return empty system prompt by default
	aiSettingsService.On("GetSystemPrompt", mock.Anything, mock.Anything, mock.Anything).Return("", nil)

	handler := NewChatWebSocketHandler(chatService, aiService, aiSettingsService, subchatStorage, logger)

	// Mock JWT middleware - set userId and companyId in context
	router.Use(func(c *gin.Context) {
		c.Set("userId", "test-user-123")
		c.Set("companyId", "test-company-456")
		c.Next()
	})

	router.GET("/api/v1/chat/stream", handler.HandleChatWebSocket)

	server := httptest.NewServer(router)
	return server, handler
}

// connectWebSocket creates a WebSocket connection to the test server
func connectWebSocket(t *testing.T, serverURL, sessionID string) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/api/v1/chat/stream?sessionId=" + sessionID
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err, "Failed to connect to WebSocket")
	return conn
}

// TestWebSocketBasicTokenStreaming tests basic token streaming without tools
func TestWebSocketBasicTokenStreaming(t *testing.T) {
	sessionID := primitive.NewObjectID()
	mockChat := new(MockChatService)
	mockAI := new(MockAIService)

	// Setup mocks
	mockChat.On("GetSession", mock.Anything, sessionID, "test-company-456").Return(&models.ChatSession{
		ID:        sessionID,
		UserID:    "test-user-123",
		CompanyID: "test-company-456",
		Title:     "Test Session",
	}, nil)

	mockChat.On("GetSessionMessages", mock.Anything, sessionID).Return([]models.ChatMessage{}, nil)
	mockChat.On("SaveMessage", mock.Anything, sessionID, "user", "Hello AI", "test-company-456").Return(&models.ChatMessage{}, nil)
	mockChat.On("SaveMessage", mock.Anything, sessionID, "assistant", mock.Anything, "test-company-456").Return(&models.ChatMessage{}, nil)

	// Setup AI streaming response
	responseChan := make(chan string, 3)
	responseChan <- "Hello"
	responseChan <- " "
	responseChan <- "World"
	close(responseChan)

	// Mock StreamChatWithTools with correct parameters (ctx, messages, maxToolCalls)
	mockAI.On("StreamChatWithTools", mock.Anything, mock.Anything, mock.Anything).Return((<-chan string)(responseChan), nil)

	// Start test server
	server, _ := setupTestServer(t, mockChat, mockAI)
	defer server.Close()

	// Connect WebSocket
	conn := connectWebSocket(t, server.URL, sessionID.Hex())
	defer conn.Close()

	// Send user message
	userMsg := models.SendMessageRequest{Content: "Hello AI"}
	err := conn.WriteJSON(userMsg)
	assert.NoError(t, err)

	// Read user_message echo
	var userMsgEcho models.StreamMessage
	err = conn.ReadJSON(&userMsgEcho)
	assert.NoError(t, err)
	assert.Equal(t, "user_message", userMsgEcho.Type)
	assert.Equal(t, "Hello AI", userMsgEcho.Content)

	// Read message_saved event (contains database ID)
	var savedMsg models.StreamMessage
	err = conn.ReadJSON(&savedMsg)
	assert.NoError(t, err)
	assert.Equal(t, "message_saved", savedMsg.Type)

	// Read token responses
	tokens := []string{}
	for i := 0; i < 3; i++ {
		var msg models.StreamMessage
		err := conn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.Equal(t, "token", msg.Type)
		tokens = append(tokens, msg.Content)
	}

	// Read done message
	var doneMsg models.StreamMessage
	err = conn.ReadJSON(&doneMsg)
	assert.NoError(t, err)
	assert.Equal(t, "done", doneMsg.Type)

	// Verify tokens received
	assert.Equal(t, []string{"Hello", " ", "World"}, tokens)

	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

// TestWebSocketToolExecution tests tool call and result events (future implementation)
func TestWebSocketToolExecution(t *testing.T) {
	t.Skip("Skipping tool execution test - waiting for Phase 2 ai-service tool integration")

	// This test will be enabled once ai-service returns tool events
	// Test scenario:
	// 1. User message → AI requests bash tool
	// 2. Verify tool_call event sent to client
	// 3. Verify tool_result event sent to client
	// 4. Verify AI response → done
}

// TestWebSocketToolExecutionError tests tool execution error handling (future)
func TestWebSocketToolExecutionError(t *testing.T) {
	t.Skip("Skipping tool error test - waiting for Phase 2 ai-service tool integration")

	// Test scenario:
	// 1. User message → AI requests invalid tool
	// 2. Verify tool_call event
	// 3. Verify tool_result with error field populated
	// 4. Verify AI handles error gracefully
}

// TestWebSocketLargeToolOutput tests chunking of large tool results (future)
func TestWebSocketLargeToolOutput(t *testing.T) {
	t.Skip("Skipping large output test - waiting for Phase 2 ai-service tool integration")

	// Test scenario:
	// 1. User message → AI requests tool with large output (>10KB)
	// 2. Verify tool_call event
	// 3. Verify multiple tool_result_chunk events
	// 4. Verify final chunk has done=true
	// 5. Verify client can reassemble chunks
}

// TestWebSocketConcurrentMessageRejection tests that new messages are rejected during AI processing
func TestWebSocketConcurrentMessageRejection(t *testing.T) {
	t.Skip("Skipping concurrent message rejection test - feature requires refactoring.\n" +
		"Current implementation processes messages sequentially in a single loop.\n" +
		"The message loop blocks on streamAIResponse (line 1560) until AI completes,\n" +
		"so it never reads the second message until the first finishes.\n" +
		"To enable concurrent rejection, message reading must happen in a separate goroutine.\n" +
		"See: internal/handlers/chat_websocket.go lines 1408-1566")

	sessionID := primitive.NewObjectID()
	mockChat := new(MockChatService)
	mockAI := new(MockAIService)

	// Setup mocks
	mockChat.On("GetSession", mock.Anything, sessionID, "test-company-456").Return(&models.ChatSession{
		ID:        sessionID,
		UserID:    "test-user-123",
		CompanyID: "test-company-456",
		Title:     "Test Session",
	}, nil)

	mockChat.On("GetSessionMessages", mock.Anything, sessionID).Return([]models.ChatMessage{}, nil)
	mockChat.On("SaveMessage", mock.Anything, sessionID, "user", "First message", "test-company-456").Return(&models.ChatMessage{}, nil).Once()
	mockChat.On("SaveMessage", mock.Anything, sessionID, "assistant", mock.Anything, "test-company-456").Return(&models.ChatMessage{}, nil)

	// Setup slow AI response to simulate processing (longer delays to ensure concurrency)
	responseChan := make(chan string, 2)
	go func() {
		time.Sleep(500 * time.Millisecond) // Longer initial delay
		responseChan <- "Processing"
		time.Sleep(500 * time.Millisecond)
		responseChan <- "..."
		close(responseChan)
	}()

	mockAI.On("StreamChatWithTools", mock.Anything, mock.Anything, mock.Anything).Return((<-chan string)(responseChan), nil)

	// Start test server
	server, _ := setupTestServer(t, mockChat, mockAI)
	defer server.Close()

	// Connect WebSocket
	conn := connectWebSocket(t, server.URL, sessionID.Hex())
	defer conn.Close()

	// Send first message
	userMsg1 := models.SendMessageRequest{Content: "First message"}
	err := conn.WriteJSON(userMsg1)
	assert.NoError(t, err)

	// Immediately send second message (concurrently, before first completes processing)
	userMsg2 := models.SendMessageRequest{Content: "Second message"}
	err = conn.WriteJSON(userMsg2)
	assert.NoError(t, err)

	// Small delay to ensure server receives and processes the rejection
	time.Sleep(50 * time.Millisecond)

	// Read messages and verify sequence:
	// 1. user_message echo for first
	// 2. message_saved for first
	// 3. error for second (rejected)
	// 4. tokens from first message's AI response

	messages := []models.StreamMessage{}

	// Collect initial messages (user echo, saved, error)
	// Should receive these before AI starts streaming (which has 500ms delay)
	for i := 0; i < 3; i++ {
		var msg models.StreamMessage
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		err = conn.ReadJSON(&msg)
		if err != nil {
			t.Fatalf("Failed to read message %d: %v", i, err)
		}
		messages = append(messages, msg)
	}

	// Verify we got user_message, message_saved, and error
	assert.GreaterOrEqual(t, len(messages), 3, "Should have received at least 3 messages")
	assert.Equal(t, "user_message", messages[0].Type)
	assert.Equal(t, "message_saved", messages[1].Type)
	assert.Equal(t, "error", messages[2].Type)
	assert.Contains(t, messages[2].Error, "wait for current response to complete")

	// Continue reading remaining tokens from first message
	for {
		var msg models.StreamMessage
		err = conn.ReadJSON(&msg)
		if err != nil || msg.Type == "done" {
			break
		}
	}

	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

// TestWebSocketErrorHandling tests error scenarios
func TestWebSocketErrorHandling(t *testing.T) {
	sessionID := primitive.NewObjectID()
	mockChat := new(MockChatService)
	mockAI := new(MockAIService)

	// Setup mocks
	mockChat.On("GetSession", mock.Anything, sessionID, "test-company-456").Return(&models.ChatSession{
		ID:        sessionID,
		UserID:    "test-user-123",
		CompanyID: "test-company-456",
		Title:     "Test Session",
	}, nil)

	mockChat.On("GetSessionMessages", mock.Anything, sessionID).Return([]models.ChatMessage{}, nil)
	mockChat.On("SaveMessage", mock.Anything, sessionID, "user", "Test", "test-company-456").Return(&models.ChatMessage{}, nil)
	mockChat.On("SaveMessage", mock.Anything, sessionID, "assistant", mock.Anything, "test-company-456").Return(&models.ChatMessage{}, nil)

	// Setup AI error response
	responseChan := make(chan string, 1)
	responseChan <- "ERROR: AI service failure"
	close(responseChan)

	mockAI.On("StreamChatWithTools", mock.Anything, mock.Anything, mock.Anything).Return((<-chan string)(responseChan), nil)

	// Start test server
	server, _ := setupTestServer(t, mockChat, mockAI)
	defer server.Close()

	// Connect WebSocket
	conn := connectWebSocket(t, server.URL, sessionID.Hex())
	defer conn.Close()

	// Send user message
	userMsg := models.SendMessageRequest{Content: "Test"}
	err := conn.WriteJSON(userMsg)
	assert.NoError(t, err)

	// Read user_message echo
	var userMsgEcho models.StreamMessage
	err = conn.ReadJSON(&userMsgEcho)
	assert.NoError(t, err)
	assert.Equal(t, "user_message", userMsgEcho.Type)

	// Read message_saved event
	var savedMsg models.StreamMessage
	err = conn.ReadJSON(&savedMsg)
	assert.NoError(t, err)
	assert.Equal(t, "message_saved", savedMsg.Type)

	// Read AI response token (which contains the error)
	var tokenMsg models.StreamMessage
	err = conn.ReadJSON(&tokenMsg)
	assert.NoError(t, err)
	assert.Equal(t, "token", tokenMsg.Type)
	assert.Contains(t, tokenMsg.Content, "ERROR:")

	mockChat.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

// TestWebSocketInvalidMessageFormat tests handling of malformed messages
func TestWebSocketInvalidMessageFormat(t *testing.T) {
	sessionID := primitive.NewObjectID()
	mockChat := new(MockChatService)
	mockAI := new(MockAIService)

	// Setup mocks
	mockChat.On("GetSession", mock.Anything, sessionID, "test-company-456").Return(&models.ChatSession{
		ID:        sessionID,
		UserID:    "test-user-123",
		CompanyID: "test-company-456",
		Title:     "Test Session",
	}, nil)

	// Start test server
	server, _ := setupTestServer(t, mockChat, mockAI)
	defer server.Close()

	// Connect WebSocket
	conn := connectWebSocket(t, server.URL, sessionID.Hex())
	defer conn.Close()

	// Send invalid JSON
	err := conn.WriteMessage(websocket.TextMessage, []byte("{invalid json}"))
	assert.NoError(t, err)

	// Read error response
	var errorMsg models.StreamMessage
	err = conn.ReadJSON(&errorMsg)
	assert.NoError(t, err)
	assert.Equal(t, "error", errorMsg.Type)
	assert.Contains(t, errorMsg.Error, "Invalid message format")

	mockChat.AssertExpectations(t)
}

// TestWebSocketUnauthorizedAccess tests access control
func TestWebSocketUnauthorizedAccess(t *testing.T) {
	sessionID := primitive.NewObjectID()
	mockChat := new(MockChatService)
	mockAI := new(MockAIService)

	// Setup mock - session belongs to different user
	mockChat.On("GetSession", mock.Anything, sessionID, "test-company-456").Return(&models.ChatSession{
		ID:        sessionID,
		UserID:    "different-user-999", // Different user!
		CompanyID: "test-company-456",
		Title:     "Test Session",
	}, nil)

	// Start test server
	server, _ := setupTestServer(t, mockChat, mockAI)
	defer server.Close()

	// Try to connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/chat/stream?sessionId=" + sessionID.Hex()
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	// Should fail with 403 Forbidden
	if err == nil {
		conn.Close()
		t.Fatal("Expected connection to fail for unauthorized access")
	}

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	mockChat.AssertExpectations(t)
}

// TestStreamToolResultChunking tests the streamToolResult helper function directly
func TestStreamToolResultChunking(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := &ChatWebSocketHandler{
		logger: logger,
	}

	// Define our own upgrader for this test
	testUpgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Use channel to signal when server writes are complete
	writeDone := make(chan bool)

	// Create a mock WebSocket connection for testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// Test small result (no chunking)
		smallResult := models.ToolResultEvent{
			ID:         "tool-1",
			Result:     "small output",
			DurationMs: 100,
		}
		err = handler.streamToolResult(conn, smallResult)
		if err != nil {
			t.Errorf("Failed to stream small result: %v", err)
			return
		}

		// Test large result (should chunk)
		largeData := make([]byte, 15*1024) // 15KB
		for i := range largeData {
			largeData[i] = 'A'
		}
		largeResult := models.ToolResultEvent{
			ID:         "tool-2",
			Result:     string(largeData),
			DurationMs: 500,
		}
		err = handler.streamToolResult(conn, largeResult)
		if err != nil {
			t.Errorf("Failed to stream large result: %v", err)
			return
		}

		// Signal that writing is complete
		writeDone <- true

		// Keep connection open until client disconnects
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}))
	defer server.Close()

	// Connect and verify messages
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Set read deadline to prevent hanging
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Read small result message
	var smallMsg models.StreamMessage
	err = conn.ReadJSON(&smallMsg)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "tool_result", smallMsg.Type)
	if assert.NotNil(t, smallMsg.ToolResult) {
		assert.Equal(t, "tool-1", smallMsg.ToolResult.ID)
	}

	// Read large result chunks
	chunkCount := 0
	for {
		var chunkMsg models.StreamMessage
		err = conn.ReadJSON(&chunkMsg)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			}
			t.Logf("Error reading chunk: %v", err)
			break
		}

		// Verify message type
		if !assert.Equal(t, "tool_result_chunk", chunkMsg.Type) {
			break
		}

		chunkCount++

		// Check if ToolResult is present
		if chunkMsg.ToolResult == nil {
			t.Error("ToolResult is nil in chunk message")
			break
		}

		// Check if this is the final chunk
		if chunk, ok := chunkMsg.ToolResult.Result.(map[string]interface{}); ok {
			if done, exists := chunk["done"]; exists && done == true {
				break
			}
		}

		// Safety limit to prevent infinite loop
		if chunkCount > 10 {
			break
		}
	}

	// Wait for server to finish writing
	select {
	case <-writeDone:
		// Server completed successfully
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for server to complete writing")
	}

	assert.Greater(t, chunkCount, 1, "Large result should be split into multiple chunks")
}
