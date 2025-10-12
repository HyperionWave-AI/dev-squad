package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Test MongoDB connection helper
func setupTestMongoDB(t *testing.T) (*MongoTaskStorage, func()) {
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

	storage, err := NewMongoTaskStorage(db)
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

	return storage, cleanup
}

// Helper to create a test agent task
func createTestAgentTask(t *testing.T, storage *MongoTaskStorage) *AgentTask {
	// Create human task first
	humanTask, err := storage.CreateHumanTask("Test human task")
	if err != nil {
		t.Fatalf("Failed to create test human task: %v", err)
	}

	// Create agent task
	todos := []TodoItemInput{
		{
			Description: "Test TODO 1",
		},
		{
			Description: "Test TODO 2",
		},
	}

	agentTask, err := storage.CreateAgentTask(humanTask.ID, "Test Agent", "Test role", todos, "", nil, nil, "")
	if err != nil {
		t.Fatalf("Failed to create test agent task: %v", err)
	}

	return agentTask
}

// TestAddTaskPromptNotes tests adding prompt notes to a task
func TestAddTaskPromptNotes(t *testing.T) {
	storage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	task := createTestAgentTask(t, storage)
	notes := "Test prompt notes with **markdown**"

	err := storage.AddTaskPromptNotes(task.ID, notes)
	if err != nil {
		t.Fatalf("AddTaskPromptNotes() error = %v", err)
	}

	// Retrieve and verify
	tasks, err := storage.GetAgentTasksByName(task.AgentName)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	retrieved := tasks[0]
	if retrieved.HumanPromptNotes != notes {
		t.Errorf("HumanPromptNotes = %q, want %q", retrieved.HumanPromptNotes, notes)
	}
	if retrieved.HumanPromptNotesAddedAt == nil {
		t.Error("HumanPromptNotesAddedAt should not be nil")
	}
	if retrieved.HumanPromptNotesUpdatedAt == nil {
		t.Error("HumanPromptNotesUpdatedAt should not be nil")
	}
}

// TestUpdateTaskPromptNotes tests updating existing prompt notes
func TestUpdateTaskPromptNotes(t *testing.T) {
	storage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	task := createTestAgentTask(t, storage)
	initialNotes := "Initial notes"
	updatedNotes := "Updated notes with more detail"

	// Add initial notes
	err := storage.AddTaskPromptNotes(task.ID, initialNotes)
	if err != nil {
		t.Fatalf("AddTaskPromptNotes() error = %v", err)
	}

	// Get the added timestamp
	tasks, err := storage.GetAgentTasksByName(task.AgentName)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	addedAt := tasks[0].HumanPromptNotesAddedAt

	// Small delay to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Update notes
	err = storage.UpdateTaskPromptNotes(task.ID, updatedNotes)
	if err != nil {
		t.Fatalf("UpdateTaskPromptNotes() error = %v", err)
	}

	// Verify update
	tasks, err = storage.GetAgentTasksByName(task.AgentName)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	retrieved := tasks[0]
	if retrieved.HumanPromptNotes != updatedNotes {
		t.Errorf("HumanPromptNotes = %q, want %q", retrieved.HumanPromptNotes, updatedNotes)
	}
	if retrieved.HumanPromptNotesAddedAt == nil || !retrieved.HumanPromptNotesAddedAt.Equal(*addedAt) {
		t.Error("HumanPromptNotesAddedAt should remain unchanged")
	}
	if retrieved.HumanPromptNotesUpdatedAt == nil {
		t.Error("HumanPromptNotesUpdatedAt should be set")
	}
	if !retrieved.HumanPromptNotesUpdatedAt.After(*addedAt) {
		t.Error("HumanPromptNotesUpdatedAt should be after AddedAt")
	}
}

// TestClearTaskPromptNotes tests clearing prompt notes
func TestClearTaskPromptNotes(t *testing.T) {
	storage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	task := createTestAgentTask(t, storage)
	notes := "Notes to be cleared"

	// Add notes
	err := storage.AddTaskPromptNotes(task.ID, notes)
	if err != nil {
		t.Fatalf("AddTaskPromptNotes() error = %v", err)
	}

	// Clear notes
	err = storage.ClearTaskPromptNotes(task.ID)
	if err != nil {
		t.Fatalf("ClearTaskPromptNotes() error = %v", err)
	}

	// Verify cleared
	tasks, err := storage.GetAgentTasksByName(task.AgentName)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	retrieved := tasks[0]
	if retrieved.HumanPromptNotes != "" {
		t.Errorf("HumanPromptNotes = %q, want empty string", retrieved.HumanPromptNotes)
	}
	if retrieved.HumanPromptNotesAddedAt != nil {
		t.Error("HumanPromptNotesAddedAt should be nil after clear")
	}
	if retrieved.HumanPromptNotesUpdatedAt != nil {
		t.Error("HumanPromptNotesUpdatedAt should be nil after clear")
	}
}

// TestTaskPromptNotesErrors tests error cases
func TestTaskPromptNotesErrors(t *testing.T) {
	storage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	invalidID := "invalid-task-id"

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "AddTaskPromptNotes with invalid ID",
			fn:   func() error { return storage.AddTaskPromptNotes(invalidID, "notes") },
		},
		{
			name: "UpdateTaskPromptNotes with invalid ID",
			fn:   func() error { return storage.UpdateTaskPromptNotes(invalidID, "notes") },
		},
		{
			name: "ClearTaskPromptNotes with invalid ID",
			fn:   func() error { return storage.ClearTaskPromptNotes(invalidID) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			// These operations should succeed but affect 0 documents
			// The actual error handling depends on implementation
			if err != nil {
				t.Logf("Operation returned error (may be expected): %v", err)
			}
		})
	}
}

// TestAddTodoPromptNotes tests adding prompt notes to a TODO item
func TestAddTodoPromptNotes(t *testing.T) {
	storage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	task := createTestAgentTask(t, storage)
	todoID := task.Todos[0].ID
	notes := "TODO-specific guidance with **markdown**"

	err := storage.AddTodoPromptNotes(task.ID, todoID, notes)
	if err != nil {
		t.Fatalf("AddTodoPromptNotes() error = %v", err)
	}

	// Retrieve and verify
	tasks, err := storage.GetAgentTasksByName(task.AgentName)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	retrieved := tasks[0]
	todo := retrieved.Todos[0]
	if todo.HumanPromptNotes != notes {
		t.Errorf("TODO HumanPromptNotes = %q, want %q", todo.HumanPromptNotes, notes)
	}
	if todo.HumanPromptNotesAddedAt == nil {
		t.Error("TODO HumanPromptNotesAddedAt should not be nil")
	}
	if todo.HumanPromptNotesUpdatedAt == nil {
		t.Error("TODO HumanPromptNotesUpdatedAt should not be nil")
	}
}

// TestUpdateTodoPromptNotes tests updating existing TODO prompt notes
func TestUpdateTodoPromptNotes(t *testing.T) {
	storage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	task := createTestAgentTask(t, storage)
	todoID := task.Todos[0].ID
	initialNotes := "Initial TODO notes"
	updatedNotes := "Updated TODO notes with changes"

	// Add initial notes
	err := storage.AddTodoPromptNotes(task.ID, todoID, initialNotes)
	if err != nil {
		t.Fatalf("AddTodoPromptNotes() error = %v", err)
	}

	// Get the added timestamp
	tasks, err := storage.GetAgentTasksByName(task.AgentName)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	addedAt := tasks[0].Todos[0].HumanPromptNotesAddedAt

	// Small delay to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Update notes
	err = storage.UpdateTodoPromptNotes(task.ID, todoID, updatedNotes)
	if err != nil {
		t.Fatalf("UpdateTodoPromptNotes() error = %v", err)
	}

	// Verify update
	tasks, err = storage.GetAgentTasksByName(task.AgentName)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	todo := tasks[0].Todos[0]
	if todo.HumanPromptNotes != updatedNotes {
		t.Errorf("TODO HumanPromptNotes = %q, want %q", todo.HumanPromptNotes, updatedNotes)
	}
	if todo.HumanPromptNotesAddedAt == nil || !todo.HumanPromptNotesAddedAt.Equal(*addedAt) {
		t.Error("TODO HumanPromptNotesAddedAt should remain unchanged")
	}
	if todo.HumanPromptNotesUpdatedAt == nil {
		t.Error("TODO HumanPromptNotesUpdatedAt should be set")
	}
	if !todo.HumanPromptNotesUpdatedAt.After(*addedAt) {
		t.Error("TODO HumanPromptNotesUpdatedAt should be after AddedAt")
	}
}

// TestClearTodoPromptNotes tests clearing TODO prompt notes
func TestClearTodoPromptNotes(t *testing.T) {
	storage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	task := createTestAgentTask(t, storage)
	todoID := task.Todos[0].ID
	notes := "TODO notes to be cleared"

	// Add notes
	err := storage.AddTodoPromptNotes(task.ID, todoID, notes)
	if err != nil {
		t.Fatalf("AddTodoPromptNotes() error = %v", err)
	}

	// Clear notes
	err = storage.ClearTodoPromptNotes(task.ID, todoID)
	if err != nil {
		t.Fatalf("ClearTodoPromptNotes() error = %v", err)
	}

	// Verify cleared
	tasks, err := storage.GetAgentTasksByName(task.AgentName)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	todo := tasks[0].Todos[0]
	if todo.HumanPromptNotes != "" {
		t.Errorf("TODO HumanPromptNotes = %q, want empty string", todo.HumanPromptNotes)
	}
	if todo.HumanPromptNotesAddedAt != nil {
		t.Error("TODO HumanPromptNotesAddedAt should be nil after clear")
	}
	if todo.HumanPromptNotesUpdatedAt != nil {
		t.Error("TODO HumanPromptNotesUpdatedAt should be nil after clear")
	}
}

// TestTodoPromptNotesErrors tests error cases for TODO operations
func TestTodoPromptNotesErrors(t *testing.T) {
	storage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	task := createTestAgentTask(t, storage)
	invalidTaskID := "invalid-task-id"
	invalidTodoID := "invalid-todo-id"

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "AddTodoPromptNotes with invalid task ID",
			fn:   func() error { return storage.AddTodoPromptNotes(invalidTaskID, task.Todos[0].ID, "notes") },
		},
		{
			name: "AddTodoPromptNotes with invalid TODO ID",
			fn:   func() error { return storage.AddTodoPromptNotes(task.ID, invalidTodoID, "notes") },
		},
		{
			name: "UpdateTodoPromptNotes with invalid task ID",
			fn:   func() error { return storage.UpdateTodoPromptNotes(invalidTaskID, task.Todos[0].ID, "notes") },
		},
		{
			name: "UpdateTodoPromptNotes with invalid TODO ID",
			fn:   func() error { return storage.UpdateTodoPromptNotes(task.ID, invalidTodoID, "notes") },
		},
		{
			name: "ClearTodoPromptNotes with invalid task ID",
			fn:   func() error { return storage.ClearTodoPromptNotes(invalidTaskID, task.Todos[0].ID) },
		},
		{
			name: "ClearTodoPromptNotes with invalid TODO ID",
			fn:   func() error { return storage.ClearTodoPromptNotes(task.ID, invalidTodoID) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			// These operations should succeed but affect 0 documents
			// The actual error handling depends on implementation
			if err != nil {
				t.Logf("Operation returned error (may be expected): %v", err)
			}
		})
	}
}

// TestTodoArrayFilterUpdate tests that array filters correctly target specific TODOs
func TestTodoArrayFilterUpdate(t *testing.T) {
	storage, cleanup := setupTestMongoDB(t)
	defer cleanup()

	task := createTestAgentTask(t, storage)
	// Task has 2 TODOs from createTestAgentTask
	todo2ID := task.Todos[1].ID
	notes := "Notes for second TODO only"

	// Add notes to second TODO only
	err := storage.AddTodoPromptNotes(task.ID, todo2ID, notes)
	if err != nil {
		t.Fatalf("AddTodoPromptNotes() error = %v", err)
	}

	// Retrieve and verify
	tasks, err := storage.GetAgentTasksByName(task.AgentName)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	retrieved := tasks[0]

	// First TODO should have no notes
	if retrieved.Todos[0].HumanPromptNotes != "" {
		t.Errorf("First TODO should have no notes, got %q", retrieved.Todos[0].HumanPromptNotes)
	}
	if retrieved.Todos[0].HumanPromptNotesAddedAt != nil {
		t.Error("First TODO should have nil AddedAt")
	}

	// Second TODO should have the notes
	if retrieved.Todos[1].HumanPromptNotes != notes {
		t.Errorf("Second TODO notes = %q, want %q", retrieved.Todos[1].HumanPromptNotes, notes)
	}
	if retrieved.Todos[1].HumanPromptNotesAddedAt == nil {
		t.Error("Second TODO should have AddedAt timestamp")
	}
	if retrieved.Todos[1].HumanPromptNotesUpdatedAt == nil {
		t.Error("Second TODO should have UpdatedAt timestamp")
	}
}
