package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeHuman TaskType = "human"
	TaskTypeAgent TaskType = "agent"
)

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusBlocked    TaskStatus = "blocked"
)

// TodoStatus represents the state of an individual TODO item
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
)

// TodoItem represents a single trackable subtask within an agent task
type TodoItem struct {
	ID          string     `json:"id" bson:"id"`
	Description string     `json:"description" bson:"description"`
	Status      TodoStatus `json:"status" bson:"status"`
	CreatedAt   time.Time  `json:"createdAt" bson:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	Notes       string     `json:"notes,omitempty" bson:"notes,omitempty"`
}

// HumanTask represents a task created by a human user
type HumanTask struct {
	ID        string     `json:"id" bson:"taskId"`
	Prompt    string     `json:"prompt" bson:"prompt"`
	CreatedAt time.Time  `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt" bson:"updatedAt"`
	Status    TaskStatus `json:"status" bson:"status"`
	Notes     string     `json:"notes,omitempty" bson:"notes,omitempty"`
}

// AgentTask represents a task assigned to an agent
type AgentTask struct {
	ID          string     `json:"id" bson:"taskId"`
	HumanTaskID string     `json:"humanTaskId" bson:"humanTaskId"`
	AgentName   string     `json:"agentName" bson:"agentName"`
	Role        string     `json:"role" bson:"role"`
	Todos       []TodoItem `json:"todos" bson:"todos"`
	CreatedAt   time.Time  `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt" bson:"updatedAt"`
	Status      TaskStatus `json:"status" bson:"status"`
	Notes       string     `json:"notes,omitempty" bson:"notes,omitempty"`
}

// TaskStorage provides storage interface for tasks
type TaskStorage interface {
	CreateHumanTask(prompt string) (*HumanTask, error)
	CreateAgentTask(humanTaskID, agentName, role string, todos []string) (*AgentTask, error)
	GetHumanTask(taskID string) (*HumanTask, error)
	GetAgentTask(taskID string) (*AgentTask, error)
	GetAgentTasksByName(agentName string) ([]*AgentTask, error)
	ListAllHumanTasks() []*HumanTask
	ListAllAgentTasks() []*AgentTask
	UpdateTaskStatus(taskID string, status TaskStatus, notes string) error
	UpdateTodoStatus(agentTaskID, todoID string, status TodoStatus, notes string) error
}

// MongoTaskStorage implements TaskStorage using MongoDB
type MongoTaskStorage struct {
	humanTasksCollection *mongo.Collection
	agentTasksCollection *mongo.Collection
}

// NewMongoTaskStorage creates a new MongoDB-backed task storage
func NewMongoTaskStorage(db *mongo.Database) (*MongoTaskStorage, error) {
	storage := &MongoTaskStorage{
		humanTasksCollection: db.Collection("human_tasks"),
		agentTasksCollection: db.Collection("agent_tasks"),
	}

	// Create indexes
	ctx := context.Background()

	// Index on humanTasks.taskId
	_, err := storage.humanTasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "taskId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create human tasks index: %w", err)
	}

	// Index on agentTasks.taskId
	_, err = storage.agentTasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "taskId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent tasks index: %w", err)
	}

	// Index on agentTasks.agentName for efficient queries
	_, err = storage.agentTasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "agentName", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent name index: %w", err)
	}

	// Index on agentTasks.humanTaskId for linking
	_, err = storage.agentTasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "humanTaskId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create human task ID index: %w", err)
	}

	return storage, nil
}

// CreateHumanTask creates a new human task
func (s *MongoTaskStorage) CreateHumanTask(prompt string) (*HumanTask, error) {
	ctx := context.Background()

	now := time.Now().UTC()
	task := &HumanTask{
		ID:        uuid.New().String(),
		Prompt:    prompt,
		CreatedAt: now,
		UpdatedAt: now,
		Status:    TaskStatusPending,
	}

	_, err := s.humanTasksCollection.InsertOne(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to insert human task: %w", err)
	}

	return task, nil
}

// CreateAgentTask creates a new agent task
func (s *MongoTaskStorage) CreateAgentTask(humanTaskID, agentName, role string, todos []string) (*AgentTask, error) {
	ctx := context.Background()

	// Validate human task exists
	var humanTask HumanTask
	err := s.humanTasksCollection.FindOne(ctx, bson.M{"taskId": humanTaskID}).Decode(&humanTask)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("human task with ID %s not found", humanTaskID)
		}
		return nil, fmt.Errorf("failed to validate human task: %w", err)
	}

	now := time.Now().UTC()

	// Convert string todos to TodoItem structs
	todoItems := make([]TodoItem, len(todos))
	for i, desc := range todos {
		todoItems[i] = TodoItem{
			ID:          uuid.New().String(),
			Description: desc,
			Status:      TodoStatusPending,
			CreatedAt:   now,
		}
	}

	task := &AgentTask{
		ID:          uuid.New().String(),
		HumanTaskID: humanTaskID,
		AgentName:   agentName,
		Role:        role,
		Todos:       todoItems,
		CreatedAt:   now,
		UpdatedAt:   now,
		Status:      TaskStatusPending,
	}

	_, err = s.agentTasksCollection.InsertOne(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to insert agent task: %w", err)
	}

	return task, nil
}

// GetHumanTask retrieves a human task by ID
func (s *MongoTaskStorage) GetHumanTask(taskID string) (*HumanTask, error) {
	ctx := context.Background()

	var task HumanTask
	err := s.humanTasksCollection.FindOne(ctx, bson.M{"taskId": taskID}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("human task with ID %s not found", taskID)
		}
		return nil, fmt.Errorf("failed to retrieve human task: %w", err)
	}

	return &task, nil
}

// GetAgentTask retrieves an agent task by ID
func (s *MongoTaskStorage) GetAgentTask(taskID string) (*AgentTask, error) {
	ctx := context.Background()

	var task AgentTask
	err := s.agentTasksCollection.FindOne(ctx, bson.M{"taskId": taskID}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("agent task with ID %s not found", taskID)
		}
		return nil, fmt.Errorf("failed to retrieve agent task: %w", err)
	}

	return &task, nil
}

// GetAgentTasksByName retrieves all agent tasks for a specific agent
func (s *MongoTaskStorage) GetAgentTasksByName(agentName string) ([]*AgentTask, error) {
	ctx := context.Background()

	cursor, err := s.agentTasksCollection.Find(ctx, bson.M{"agentName": agentName})
	if err != nil {
		return nil, fmt.Errorf("failed to query agent tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*AgentTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode agent tasks: %w", err)
	}

	return tasks, nil
}

// ListAllHumanTasks returns all human tasks
func (s *MongoTaskStorage) ListAllHumanTasks() []*HumanTask {
	ctx := context.Background()

	cursor, err := s.humanTasksCollection.Find(ctx, bson.M{})
	if err != nil {
		return []*HumanTask{}
	}
	defer cursor.Close(ctx)

	var tasks []*HumanTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return []*HumanTask{}
	}

	return tasks
}

// ListAllAgentTasks returns all agent tasks
func (s *MongoTaskStorage) ListAllAgentTasks() []*AgentTask {
	ctx := context.Background()

	cursor, err := s.agentTasksCollection.Find(ctx, bson.M{})
	if err != nil {
		return []*AgentTask{}
	}
	defer cursor.Close(ctx)

	var tasks []*AgentTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return []*AgentTask{}
	}

	return tasks
}

// UpdateTaskStatus updates the status and notes of any task (human or agent)
func (s *MongoTaskStorage) UpdateTaskStatus(taskID string, status TaskStatus, notes string) error {
	ctx := context.Background()

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now().UTC(),
		},
	}

	if notes != "" {
		update["$set"].(bson.M)["notes"] = notes
	}

	// Try human tasks first
	result := s.humanTasksCollection.FindOneAndUpdate(
		ctx,
		bson.M{"taskId": taskID},
		update,
	)
	if result.Err() == nil {
		return nil
	}

	// If not found in human tasks, try agent tasks
	result = s.agentTasksCollection.FindOneAndUpdate(
		ctx,
		bson.M{"taskId": taskID},
		update,
	)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return fmt.Errorf("task with ID %s not found", taskID)
		}
		return fmt.Errorf("failed to update task status: %w", result.Err())
	}

	return nil
}

// UpdateTodoStatus updates the status of a specific TODO item within an agent task
func (s *MongoTaskStorage) UpdateTodoStatus(agentTaskID, todoID string, status TodoStatus, notes string) error {
	ctx := context.Background()

	// First, get the agent task to find the todo item
	var agentTask AgentTask
	err := s.agentTasksCollection.FindOne(ctx, bson.M{"taskId": agentTaskID}).Decode(&agentTask)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("agent task with ID %s not found", agentTaskID)
		}
		return fmt.Errorf("failed to retrieve agent task: %w", err)
	}

	// Find the todo item index
	todoIndex := -1
	for i, todo := range agentTask.Todos {
		if todo.ID == todoID {
			todoIndex = i
			break
		}
	}

	if todoIndex == -1 {
		return fmt.Errorf("todo item with ID %s not found in agent task %s", todoID, agentTaskID)
	}

	// Prepare the update for the specific todo item
	now := time.Now().UTC()
	updateFields := bson.M{
		fmt.Sprintf("todos.%d.status", todoIndex):    status,
		"updatedAt":                                   now,
	}

	// Add completion timestamp if status is completed
	if status == TodoStatusCompleted {
		updateFields[fmt.Sprintf("todos.%d.completedAt", todoIndex)] = now
	}

	// Add notes if provided
	if notes != "" {
		updateFields[fmt.Sprintf("todos.%d.notes", todoIndex)] = notes
	}

	update := bson.M{"$set": updateFields}

	// Update the agent task
	result := s.agentTasksCollection.FindOneAndUpdate(
		ctx,
		bson.M{"taskId": agentTaskID},
		update,
	)

	if result.Err() != nil {
		return fmt.Errorf("failed to update todo status: %w", result.Err())
	}

	// Check if all todos are completed, and if so, auto-complete the agent task
	var updatedTask AgentTask
	err = s.agentTasksCollection.FindOne(ctx, bson.M{"taskId": agentTaskID}).Decode(&updatedTask)
	if err == nil {
		allCompleted := true
		for _, todo := range updatedTask.Todos {
			if todo.Status != TodoStatusCompleted {
				allCompleted = false
				break
			}
		}

		// Auto-complete the agent task if all todos are done
		if allCompleted && updatedTask.Status != TaskStatusCompleted {
			s.UpdateTaskStatus(agentTaskID, TaskStatusCompleted, "All TODO items completed")
		}
	}

	return nil
}