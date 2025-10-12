package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	aiservice "hyperion-coordinator/ai-service"
	"hyperion-coordinator/internal/models"
	"hyperion-coordinator/internal/services"

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

func (m *MockAIService) GetConfig() *aiservice.AIConfig {
	return &aiservice.AIConfig{
		Provider: "mock",
		Model:    "test-model",
	}
}

// setupTestServer creates a test HTTP server with WebSocket handler
func setupTestServer(t *testing.T, chatService *MockChatService, aiService *MockAIService) (*httptest.Server, *ChatWebSocketHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger, _ := zap.NewDevelopment()
	handler := NewChatWebSocketHandler(chatService, aiService, logger)

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

	mockAI.On("StreamChat", mock.Anything, mock.Anything).Return((<-chan string)(responseChan), nil)

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
	mockChat.On("SaveMessage", mock.Anything, sessionID, "user", "First message", "test-company-456").Return(&models.ChatMessage{}, nil)

	// Setup slow AI response to simulate processing
	responseChan := make(chan string, 2)
	go func() {
		time.Sleep(100 * time.Millisecond)
		responseChan <- "Processing"
		time.Sleep(100 * time.Millisecond)
		responseChan <- "..."
		close(responseChan)
	}()

	mockAI.On("StreamChat", mock.Anything, mock.Anything).Return((<-chan string)(responseChan), nil)

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

	// Immediately try to send second message (should be rejected)
	time.Sleep(50 * time.Millisecond)
	userMsg2 := models.SendMessageRequest{Content: "Second message"}
	err = conn.WriteJSON(userMsg2)
	assert.NoError(t, err)

	// Read first token
	var msg1 models.StreamMessage
	err = conn.ReadJSON(&msg1)
	assert.NoError(t, err)

	// Should receive error for second message
	var errorMsg models.StreamMessage
	err = conn.ReadJSON(&errorMsg)
	assert.NoError(t, err)
	assert.Equal(t, "error", errorMsg.Type)
	assert.Contains(t, errorMsg.Error, "wait for current response to complete")

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

	// Setup AI error response
	responseChan := make(chan string, 1)
	responseChan <- "ERROR: AI service failure"
	close(responseChan)

	mockAI.On("StreamChat", mock.Anything, mock.Anything).Return((<-chan string)(responseChan), nil)

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

	// Read error response
	var errorMsg models.StreamMessage
	err = conn.ReadJSON(&errorMsg)
	assert.NoError(t, err)
	assert.Equal(t, "error", errorMsg.Type)
	assert.Contains(t, errorMsg.Error, "ERROR:")

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

	// Create a mock WebSocket connection for testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		// Test small result (no chunking)
		smallResult := models.ToolResultEvent{
			ID:         "tool-1",
			Result:     "small output",
			DurationMs: 100,
		}
		err = handler.streamToolResult(conn, smallResult)
		assert.NoError(t, err)

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
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Connect and verify messages
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Read small result message
	var smallMsg models.StreamMessage
	err = conn.ReadJSON(&smallMsg)
	assert.NoError(t, err)
	assert.Equal(t, "tool_result", smallMsg.Type)
	assert.Equal(t, "tool-1", smallMsg.ToolResult.ID)

	// Read large result chunks
	chunkCount := 0
	for {
		var chunkMsg models.StreamMessage
		err = conn.ReadJSON(&chunkMsg)
		assert.NoError(t, err)
		assert.Equal(t, "tool_result_chunk", chunkMsg.Type)

		chunkCount++

		// Check if this is the final chunk
		if chunk, ok := chunkMsg.ToolResult.Result.(models.ToolResultChunk); ok {
			if chunk.Done {
				break
			}
		}
	}

	assert.Greater(t, chunkCount, 1, "Large result should be split into multiple chunks")
}
