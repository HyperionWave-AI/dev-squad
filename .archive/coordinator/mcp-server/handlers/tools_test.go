package handlers

import (
	"context"
	"os"
	"strings"
	"testing"

	"hyperion-coordinator-mcp/storage"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Test MongoDB connection helper
func setupTestMongoDB(t *testing.T) (*storage.MongoTaskStorage, func()) {
	// Check if MongoDB test URL is available
	mongoURL := os.Getenv("MONGODB_TEST_URL")
	if mongoURL == "" {
		t.Skip("Skipping MongoDB integration test: MONGODB_TEST_URL not set")
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		t.Fatalf("Failed to connect to test MongoDB: %v", err)
	}

	// Use a unique database name for this test
	dbName := "test_coordinator_" + uuid.New().String()
	db := client.Database(dbName)

	taskStorage, err := storage.NewMongoTaskStorage(db)
	if err != nil {
		t.Fatalf("Failed to create MongoTaskStorage: %v", err)
	}

	cleanup := func() {
		// Drop the test database
		if err := db.Drop(context.Background()); err != nil {
			t.Logf("Warning: failed to drop test database: %v", err)
		}
		if err := client.Disconnect(context.Background()); err != nil {
			t.Logf("Warning: failed to disconnect MongoDB client: %v", err)
		}
	}

	return taskStorage, cleanup
}

// Helper to create a test agent task
func createTestAgentTask(t *testing.T, taskStorage *storage.MongoTaskStorage) *storage.AgentTask {
	// Create human task first
	humanTask, err := taskStorage.CreateHumanTask("Test human task")
	if err != nil {
		t.Fatalf("Failed to create test human task: %v", err)
	}

	// Create agent task
	todos := []storage.TodoItemInput{
		{
			Description: "Test TODO 1",
		},
		{
			Description: "Test TODO 2",
		},
	}

	task, err := taskStorage.CreateAgentTask(humanTask.ID, "Test Agent", "Test role", todos, "", nil, nil, "")
	if err != nil {
		t.Fatalf("Failed to create test agent task: %v", err)
	}

	return task
}

// TestHandleAddTaskPromptNotes tests the add task prompt notes handler
func TestHandleAddTaskPromptNotes(t *testing.T) {
	taskStorage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	handler := NewToolHandler(taskStorage, nil)
	task := createTestAgentTask(t, taskStorage)

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		errorContains string
		wantSuccess   bool
	}{
		{
			name: "Valid prompt notes",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"promptNotes": "Test notes with **markdown**",
			},
			wantSuccess: true,
		},
		{
			name: "Missing agentTaskId",
			args: map[string]interface{}{
				"promptNotes": "Test notes",
			},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
		{
			name: "Empty agentTaskId",
			args: map[string]interface{}{
				"agentTaskId": "",
				"promptNotes": "Test notes",
			},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
		{
			name: "Missing promptNotes",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
			},
			wantError:     true,
			errorContains: "promptNotes parameter is required",
		},
		{
			name: "Empty promptNotes",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"promptNotes": "",
			},
			wantError:     true,
			errorContains: "promptNotes parameter is required",
		},
		{
			name: "Prompt notes exceed max length",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"promptNotes": strings.Repeat("a", 5001),
			},
			wantError:     true,
			errorContains: "exceed maximum length",
		},
		{
			name: "HTML script tags should be sanitized",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"promptNotes": "Safe text <script>alert('xss')</script> more text",
			},
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := handler.handleAddTaskPromptNotes(context.Background(), tt.args)

			if err != nil {
				t.Fatalf("Handler returned unexpected error: %v", err)
			}

			if tt.wantError {
				// Check if result indicates an error
				if len(result.Content) == 0 {
					t.Fatal("Expected error result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in error result")
				}
				if tt.errorContains != "" && !strings.Contains(textContent.Text, tt.errorContains) {
					t.Errorf("Error message = %q, want to contain %q", textContent.Text, tt.errorContains)
				}
			}

			if tt.wantSuccess {
				if len(result.Content) == 0 {
					t.Fatal("Expected success result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in success result")
				}
				if !strings.Contains(textContent.Text, "✓ Added prompt notes") {
					t.Errorf("Success message = %q, want to contain '✓ Added prompt notes'", textContent.Text)
				}
			}
		})
	}
}

// TestHandleUpdateTaskPromptNotes tests the update task prompt notes handler
func TestHandleUpdateTaskPromptNotes(t *testing.T) {
	taskStorage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	handler := NewToolHandler(taskStorage, nil)
	task := createTestAgentTask(t, taskStorage)

	// First add some notes
	err := taskStorage.AddTaskPromptNotes(task.ID, "Initial notes")
	if err != nil {
		t.Fatalf("Failed to add initial notes: %v", err)
	}

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		errorContains string
		wantSuccess   bool
	}{
		{
			name: "Valid update",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"promptNotes": "Updated notes with changes",
			},
			wantSuccess: true,
		},
		{
			name: "Missing agentTaskId",
			args: map[string]interface{}{
				"promptNotes": "Test notes",
			},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
		{
			name: "Missing promptNotes",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
			},
			wantError:     true,
			errorContains: "promptNotes parameter is required",
		},
		{
			name: "Exceed max length",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"promptNotes": strings.Repeat("b", 5001),
			},
			wantError:     true,
			errorContains: "exceed maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := handler.handleUpdateTaskPromptNotes(context.Background(), tt.args)

			if err != nil {
				t.Fatalf("Handler returned unexpected error: %v", err)
			}

			if tt.wantError {
				if len(result.Content) == 0 {
					t.Fatal("Expected error result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in error result")
				}
				if tt.errorContains != "" && !strings.Contains(textContent.Text, tt.errorContains) {
					t.Errorf("Error message = %q, want to contain %q", textContent.Text, tt.errorContains)
				}
			}

			if tt.wantSuccess {
				if len(result.Content) == 0 {
					t.Fatal("Expected success result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in success result")
				}
				if !strings.Contains(textContent.Text, "✓ Updated prompt notes") {
					t.Errorf("Success message = %q, want to contain '✓ Updated prompt notes'", textContent.Text)
				}
			}
		})
	}
}

// TestHandleClearTaskPromptNotes tests the clear task prompt notes handler
func TestHandleClearTaskPromptNotes(t *testing.T) {
	taskStorage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	handler := NewToolHandler(taskStorage, nil)
	task := createTestAgentTask(t, taskStorage)

	// First add some notes
	err := taskStorage.AddTaskPromptNotes(task.ID, "Notes to clear")
	if err != nil {
		t.Fatalf("Failed to add initial notes: %v", err)
	}

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		errorContains string
		wantSuccess   bool
	}{
		{
			name: "Valid clear",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
			},
			wantSuccess: true,
		},
		{
			name:          "Missing agentTaskId",
			args:          map[string]interface{}{},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
		{
			name: "Empty agentTaskId",
			args: map[string]interface{}{
				"agentTaskId": "",
			},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := handler.handleClearTaskPromptNotes(context.Background(), tt.args)

			if err != nil {
				t.Fatalf("Handler returned unexpected error: %v", err)
			}

			if tt.wantError {
				if len(result.Content) == 0 {
					t.Fatal("Expected error result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in error result")
				}
				if tt.errorContains != "" && !strings.Contains(textContent.Text, tt.errorContains) {
					t.Errorf("Error message = %q, want to contain %q", textContent.Text, tt.errorContains)
				}
			}

			if tt.wantSuccess {
				if len(result.Content) == 0 {
					t.Fatal("Expected success result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in success result")
				}
				if !strings.Contains(textContent.Text, "✓ Cleared prompt notes") {
					t.Errorf("Success message = %q, want to contain '✓ Cleared prompt notes'", textContent.Text)
				}
			}
		})
	}
}

// TestHandleAddTodoPromptNotes tests the add TODO prompt notes handler
func TestHandleAddTodoPromptNotes(t *testing.T) {
	taskStorage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	handler := NewToolHandler(taskStorage, nil)
	task := createTestAgentTask(t, taskStorage)
	todoID := task.Todos[0].ID

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		errorContains string
		wantSuccess   bool
	}{
		{
			name: "Valid TODO prompt notes",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      todoID,
				"promptNotes": "TODO-specific notes with **markdown**",
			},
			wantSuccess: true,
		},
		{
			name: "Missing agentTaskId",
			args: map[string]interface{}{
				"todoId":      todoID,
				"promptNotes": "Test notes",
			},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
		{
			name: "Missing todoId",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"promptNotes": "Test notes",
			},
			wantError:     true,
			errorContains: "todoId parameter is required",
		},
		{
			name: "Missing promptNotes",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      todoID,
			},
			wantError:     true,
			errorContains: "promptNotes parameter is required",
		},
		{
			name: "Empty agentTaskId",
			args: map[string]interface{}{
				"agentTaskId": "",
				"todoId":      todoID,
				"promptNotes": "Test notes",
			},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
		{
			name: "Empty todoId",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      "",
				"promptNotes": "Test notes",
			},
			wantError:     true,
			errorContains: "todoId parameter is required",
		},
		{
			name: "Empty promptNotes",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      todoID,
				"promptNotes": "",
			},
			wantError:     true,
			errorContains: "promptNotes parameter is required",
		},
		{
			name: "Exceed max length",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      todoID,
				"promptNotes": strings.Repeat("c", 5001),
			},
			wantError:     true,
			errorContains: "exceed maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := handler.handleAddTodoPromptNotes(context.Background(), tt.args)

			if err != nil {
				t.Fatalf("Handler returned unexpected error: %v", err)
			}

			if tt.wantError {
				if len(result.Content) == 0 {
					t.Fatal("Expected error result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in error result")
				}
				if tt.errorContains != "" && !strings.Contains(textContent.Text, tt.errorContains) {
					t.Errorf("Error message = %q, want to contain %q", textContent.Text, tt.errorContains)
				}
			}

			if tt.wantSuccess {
				if len(result.Content) == 0 {
					t.Fatal("Expected success result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in success result")
				}
				if !strings.Contains(textContent.Text, "✓ Added prompt notes to TODO") {
					t.Errorf("Success message = %q, want to contain '✓ Added prompt notes to TODO'", textContent.Text)
				}
			}
		})
	}
}

// TestHandleUpdateTodoPromptNotes tests the update TODO prompt notes handler
func TestHandleUpdateTodoPromptNotes(t *testing.T) {
	taskStorage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	handler := NewToolHandler(taskStorage, nil)
	task := createTestAgentTask(t, taskStorage)
	todoID := task.Todos[0].ID

	// First add some notes
	err := taskStorage.AddTodoPromptNotes(task.ID, todoID, "Initial TODO notes")
	if err != nil {
		t.Fatalf("Failed to add initial notes: %v", err)
	}

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		errorContains string
		wantSuccess   bool
	}{
		{
			name: "Valid update",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      todoID,
				"promptNotes": "Updated TODO notes",
			},
			wantSuccess: true,
		},
		{
			name: "Missing agentTaskId",
			args: map[string]interface{}{
				"todoId":      todoID,
				"promptNotes": "Test notes",
			},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
		{
			name: "Missing todoId",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"promptNotes": "Test notes",
			},
			wantError:     true,
			errorContains: "todoId parameter is required",
		},
		{
			name: "Missing promptNotes",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      todoID,
			},
			wantError:     true,
			errorContains: "promptNotes parameter is required",
		},
		{
			name: "Exceed max length",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      todoID,
				"promptNotes": strings.Repeat("d", 5001),
			},
			wantError:     true,
			errorContains: "exceed maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := handler.handleUpdateTodoPromptNotes(context.Background(), tt.args)

			if err != nil {
				t.Fatalf("Handler returned unexpected error: %v", err)
			}

			if tt.wantError {
				if len(result.Content) == 0 {
					t.Fatal("Expected error result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in error result")
				}
				if tt.errorContains != "" && !strings.Contains(textContent.Text, tt.errorContains) {
					t.Errorf("Error message = %q, want to contain %q", textContent.Text, tt.errorContains)
				}
			}

			if tt.wantSuccess {
				if len(result.Content) == 0 {
					t.Fatal("Expected success result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in success result")
				}
				if !strings.Contains(textContent.Text, "✓ Updated prompt notes for TODO") {
					t.Errorf("Success message = %q, want to contain '✓ Updated prompt notes for TODO'", textContent.Text)
				}
			}
		})
	}
}

// TestHandleClearTodoPromptNotes tests the clear TODO prompt notes handler
func TestHandleClearTodoPromptNotes(t *testing.T) {
	taskStorage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	handler := NewToolHandler(taskStorage, nil)
	task := createTestAgentTask(t, taskStorage)
	todoID := task.Todos[0].ID

	// First add some notes
	err := taskStorage.AddTodoPromptNotes(task.ID, todoID, "TODO notes to clear")
	if err != nil {
		t.Fatalf("Failed to add initial notes: %v", err)
	}

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantError     bool
		errorContains string
		wantSuccess   bool
	}{
		{
			name: "Valid clear",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      todoID,
			},
			wantSuccess: true,
		},
		{
			name: "Missing agentTaskId",
			args: map[string]interface{}{
				"todoId": todoID,
			},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
		{
			name: "Missing todoId",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
			},
			wantError:     true,
			errorContains: "todoId parameter is required",
		},
		{
			name: "Empty agentTaskId",
			args: map[string]interface{}{
				"agentTaskId": "",
				"todoId":      todoID,
			},
			wantError:     true,
			errorContains: "agentTaskId parameter is required",
		},
		{
			name: "Empty todoId",
			args: map[string]interface{}{
				"agentTaskId": task.ID,
				"todoId":      "",
			},
			wantError:     true,
			errorContains: "todoId parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := handler.handleClearTodoPromptNotes(context.Background(), tt.args)

			if err != nil {
				t.Fatalf("Handler returned unexpected error: %v", err)
			}

			if tt.wantError {
				if len(result.Content) == 0 {
					t.Fatal("Expected error result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in error result")
				}
				if tt.errorContains != "" && !strings.Contains(textContent.Text, tt.errorContains) {
					t.Errorf("Error message = %q, want to contain %q", textContent.Text, tt.errorContains)
				}
			}

			if tt.wantSuccess {
				if len(result.Content) == 0 {
					t.Fatal("Expected success result but got empty content")
				}
				textContent, ok := result.Content[0].(*mcp.TextContent)
				if !ok {
					t.Fatal("Expected TextContent in success result")
				}
				if !strings.Contains(textContent.Text, "✓ Cleared prompt notes from TODO") {
					t.Errorf("Success message = %q, want to contain '✓ Cleared prompt notes from TODO'", textContent.Text)
				}
			}
		})
	}
}
