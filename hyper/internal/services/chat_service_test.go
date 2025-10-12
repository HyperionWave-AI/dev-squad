package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// setupTestChatService creates a test service with in-memory MongoDB
func setupTestChatService(t *testing.T) (*ChatService, func()) {
	// Use MongoDB test client (requires MongoDB running locally or use mongomock)
	// For unit tests, we'll use a real MongoDB connection to localhost
	// In CI/CD, you'd use docker-compose or testcontainers
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	// Connect to local MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v. Skipping integration test.", err)
		return nil, nil
	}

	// Use unique database for this test
	dbName := "chat_service_test_" + primitive.NewObjectID().Hex()
	db := client.Database(dbName)

	service, err := NewChatService(db, logger)
	if err != nil {
		t.Fatalf("Failed to create chat service: %v", err)
	}

	cleanup := func() {
		db.Drop(ctx)
		client.Disconnect(ctx)
	}

	return service, cleanup
}

// Test CreateSession
func TestCreateSession(t *testing.T) {
	service, cleanup := setupTestChatService(t)
	if service == nil {
		return // Skipped
	}
	defer cleanup()

	tests := []struct {
		name      string
		userID    string
		companyID string
		title     string
		wantErr   bool
	}{
		{
			name:      "success - valid session creation",
			userID:    "user-123",
			companyID: "company-456",
			title:     "Test Chat Session",
			wantErr:   false,
		},
		{
			name:      "success - empty title",
			userID:    "user-789",
			companyID: "company-456",
			title:     "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			session, err := service.CreateSession(ctx, tt.userID, tt.companyID, tt.title)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
				assert.NotEqual(t, primitive.NilObjectID, session.ID)
				assert.Equal(t, tt.userID, session.UserID)
				assert.Equal(t, tt.companyID, session.CompanyID)
				assert.Equal(t, tt.title, session.Title)
			}
		})
	}
}

// Test GetUserSessions
func TestGetUserSessions(t *testing.T) {
	service, cleanup := setupTestChatService(t)
	if service == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	userID := "user-test-123"
	companyID := "company-test-456"

	// Create multiple sessions
	session1, _ := service.CreateSession(ctx, userID, companyID, "Session 1")
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	session2, _ := service.CreateSession(ctx, userID, companyID, "Session 2")

	// Retrieve sessions
	sessions, err := service.GetUserSessions(ctx, userID, companyID)
	assert.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Check ordering (latest first)
	assert.Equal(t, session2.ID, sessions[0].ID)
	assert.Equal(t, session1.ID, sessions[1].ID)
}

// Test SaveMessage
func TestSaveMessage(t *testing.T) {
	service, cleanup := setupTestChatService(t)
	if service == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	session, _ := service.CreateSession(ctx, "user-123", "company-456", "Test Session")

	tests := []struct {
		name      string
		sessionID primitive.ObjectID
		role      string
		content   string
		companyID string
		wantErr   bool
	}{
		{
			name:      "success - user message",
			sessionID: session.ID,
			role:      "user",
			content:   "Hello, how are you?",
			companyID: "company-456",
			wantErr:   false,
		},
		{
			name:      "success - assistant message",
			sessionID: session.ID,
			role:      "assistant",
			content:   "I'm doing great, thank you!",
			companyID: "company-456",
			wantErr:   false,
		},
		{
			name:      "error - wrong company",
			sessionID: session.ID,
			role:      "user",
			content:   "Test",
			companyID: "wrong-company",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, err := service.SaveMessage(ctx, tt.sessionID, tt.role, tt.content, tt.companyID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, message)
				assert.NotEqual(t, primitive.NilObjectID, message.ID)
				assert.Equal(t, tt.sessionID, message.SessionID)
				assert.Equal(t, tt.role, message.Role)
				assert.Equal(t, tt.content, message.Content)
			}
		})
	}
}

// Test GetMessages with pagination
func TestGetMessages(t *testing.T) {
	service, cleanup := setupTestChatService(t)
	if service == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	session, _ := service.CreateSession(ctx, "user-123", "company-456", "Test Session")

	// Create 5 messages
	for i := 0; i < 5; i++ {
		service.SaveMessage(ctx, session.ID, "user", "Message "+string(rune('A'+i)), "company-456")
		time.Sleep(5 * time.Millisecond) // Ensure different timestamps
	}

	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedCount int
		expectedHasMore bool
	}{
		{
			name:            "first page",
			limit:           2,
			offset:          0,
			expectedCount:   2,
			expectedHasMore: true,
		},
		{
			name:            "second page",
			limit:           2,
			offset:          2,
			expectedCount:   2,
			expectedHasMore: true,
		},
		{
			name:            "last page",
			limit:           2,
			offset:          4,
			expectedCount:   1,
			expectedHasMore: false,
		},
		{
			name:            "all messages",
			limit:           10,
			offset:          0,
			expectedCount:   5,
			expectedHasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetMessages(ctx, session.ID, "company-456", tt.limit, tt.offset)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(result.Messages))
			assert.Equal(t, int64(5), result.Total)
			assert.Equal(t, tt.expectedHasMore, result.HasMore)
		})
	}
}

// Test DeleteSession
func TestDeleteSession(t *testing.T) {
	service, cleanup := setupTestChatService(t)
	if service == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	userID := "user-123"
	companyID := "company-456"
	session, _ := service.CreateSession(ctx, userID, companyID, "Test Session")

	// Add some messages
	service.SaveMessage(ctx, session.ID, "user", "Test message", companyID)

	// Delete session
	err := service.DeleteSession(ctx, session.ID, userID, companyID)
	assert.NoError(t, err)

	// Verify session is deleted
	_, err = service.GetSession(ctx, session.ID, companyID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Test SaveToolCall and SaveToolResult
func TestSaveToolCallAndResult(t *testing.T) {
	service, cleanup := setupTestChatService(t)
	if service == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	session, _ := service.CreateSession(ctx, "user-123", "company-456", "Test Session")

	// Save tool call
	toolCall, err := service.SaveToolCall(ctx, session.ID, "call-123", "search_documents", map[string]interface{}{
		"query": "test query",
	}, "company-456")

	assert.NoError(t, err)
	assert.NotNil(t, toolCall)
	assert.Equal(t, "tool_call", toolCall.Role)
	assert.NotNil(t, toolCall.ToolCall)
	assert.Equal(t, "call-123", toolCall.ToolCall.ID)
	assert.Equal(t, "search_documents", toolCall.ToolCall.Name)

	// Save tool result
	toolResult, err := service.SaveToolResult(ctx, session.ID, "call-123", "search_documents", map[string]interface{}{
		"results": []string{"doc1", "doc2"},
	}, "", 150, "company-456")

	assert.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.Equal(t, "tool_result", toolResult.Role)
	assert.NotNil(t, toolResult.ToolResult)
	assert.Equal(t, "call-123", toolResult.ToolResult.ID)
	assert.Equal(t, int64(150), toolResult.ToolResult.DurationMs)
}

// Test SetSessionSubagent
func TestSetSessionSubagent(t *testing.T) {
	service, cleanup := setupTestChatService(t)
	if service == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	session, _ := service.CreateSession(ctx, "user-123", "company-456", "Test Session")

	subagentID := primitive.NewObjectID()

	// Set subagent
	err := service.SetSessionSubagent(ctx, session.ID, &subagentID, "company-456")
	assert.NoError(t, err)

	// Verify subagent is set (would need to query session to verify)
	// For now, just check no error

	// Clear subagent
	err = service.SetSessionSubagent(ctx, session.ID, nil, "company-456")
	assert.NoError(t, err)
}

// Test authorization - access control
func TestSessionAccessControl(t *testing.T) {
	service, cleanup := setupTestChatService(t)
	if service == nil {
		return
	}
	defer cleanup()

	ctx := context.Background()
	session, _ := service.CreateSession(ctx, "user-123", "company-456", "Test Session")

	t.Run("wrong company cannot access session", func(t *testing.T) {
		_, err := service.GetSession(ctx, session.ID, "wrong-company")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found or access denied")
	})

	t.Run("wrong user cannot delete session", func(t *testing.T) {
		err := service.DeleteSession(ctx, session.ID, "wrong-user", "company-456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}
