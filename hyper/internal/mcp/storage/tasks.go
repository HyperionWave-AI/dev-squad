package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
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
	ID           string     `json:"id" bson:"id"`
	Description  string     `json:"description" bson:"description"`
	Status       TodoStatus `json:"status" bson:"status"`
	CreatedAt    time.Time  `json:"createdAt" bson:"createdAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	Notes                     string     `json:"notes,omitempty" bson:"notes,omitempty"`
	FilePath                  string     `json:"filePath,omitempty" bson:"filePath,omitempty"`
	FunctionName              string     `json:"functionName,omitempty" bson:"functionName,omitempty"`
	ContextHint               string     `json:"contextHint,omitempty" bson:"contextHint,omitempty"`
	HumanPromptNotes          string     `json:"humanPromptNotes,omitempty" bson:"humanPromptNotes,omitempty"`
	HumanPromptNotesAddedAt   *time.Time `json:"humanPromptNotesAddedAt,omitempty" bson:"humanPromptNotesAddedAt,omitempty"`
	HumanPromptNotesUpdatedAt *time.Time `json:"humanPromptNotesUpdatedAt,omitempty" bson:"humanPromptNotesUpdatedAt,omitempty"`
}

// TodoItemInput represents the input format for creating a TODO item
type TodoItemInput struct {
	Description  string `json:"description"`
	FilePath     string `json:"filePath,omitempty"`
	FunctionName string `json:"functionName,omitempty"`
	ContextHint  string `json:"contextHint,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

// HumanTask represents a task created by a human user
type HumanTask struct {
	ID        string     `json:"taskId" bson:"taskId"`
	Prompt    string     `json:"prompt" bson:"prompt"`
	Summary   string     `json:"summary,omitempty" bson:"summary,omitempty"` // AI-generated summary (max 100 tokens)
	CreatedAt time.Time  `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt" bson:"updatedAt"`
	Status    TaskStatus `json:"status" bson:"status"`
	Notes     string     `json:"notes,omitempty" bson:"notes,omitempty"`
}

// AgentTask represents a task assigned to an agent
type AgentTask struct {
	ID                string     `json:"taskId" bson:"taskId"`
	HumanTaskID       string     `json:"humanTaskId" bson:"humanTaskId"`
	AgentName         string     `json:"agentName" bson:"agentName"`
	Role              string     `json:"role" bson:"role"`
	Todos             []TodoItem `json:"todos" bson:"todos"`
	CreatedAt         time.Time  `json:"createdAt" bson:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt" bson:"updatedAt"`
	Status            TaskStatus `json:"status" bson:"status"`
	Notes             string     `json:"notes,omitempty" bson:"notes,omitempty"`
	ContextSummary    string     `json:"contextSummary,omitempty" bson:"contextSummary,omitempty"`
	Summary           string     `json:"summary,omitempty" bson:"summary,omitempty"` // AI-generated summary (max 100 tokens)
	FilesModified             []string   `json:"filesModified,omitempty" bson:"filesModified,omitempty"`
	QdrantCollections         []string   `json:"qdrantCollections,omitempty" bson:"qdrantCollections,omitempty"`
	PriorWorkSummary          string     `json:"priorWorkSummary,omitempty" bson:"priorWorkSummary,omitempty"`
	HumanPromptNotes          string     `json:"humanPromptNotes,omitempty" bson:"humanPromptNotes,omitempty"`
	HumanPromptNotesAddedAt   *time.Time `json:"humanPromptNotesAddedAt,omitempty" bson:"humanPromptNotesAddedAt,omitempty"`
	HumanPromptNotesUpdatedAt *time.Time `json:"humanPromptNotesUpdatedAt,omitempty" bson:"humanPromptNotesUpdatedAt,omitempty"`
}

// ClearResult contains statistics about cleared tasks
type ClearResult struct {
	HumanTasksDeleted int64     `json:"humanTasksDeleted"`
	AgentTasksDeleted int64     `json:"agentTasksDeleted"`
	ClearedAt         time.Time `json:"clearedAt"`
}

// TaskStorage provides storage interface for tasks
type TaskStorage interface {
	CreateHumanTask(prompt string) (*HumanTask, error)
	CreateAgentTask(humanTaskID, agentName, role string, todos []TodoItemInput, contextSummary string, filesModified []string, qdrantCollections []string, priorWorkSummary string) (*AgentTask, error)
	GetHumanTask(taskID string) (*HumanTask, error)
	GetAgentTask(taskID string) (*AgentTask, error)
	GetAgentTasksByName(agentName string) ([]*AgentTask, error)
	ListAllHumanTasks() []*HumanTask
	ListAllAgentTasks() []*AgentTask
	UpdateTaskStatus(taskID string, status TaskStatus, notes string) error
	UpdateTodoStatus(agentTaskID, todoID string, status TodoStatus, notes string) error
	AddTaskPromptNotes(agentTaskID string, notes string) error
	UpdateTaskPromptNotes(agentTaskID string, notes string) error
	ClearTaskPromptNotes(agentTaskID string) error
	AddTodoPromptNotes(agentTaskID string, todoID string, notes string) error
	UpdateTodoPromptNotes(agentTaskID string, todoID string, notes string) error
	ClearTodoPromptNotes(agentTaskID string, todoID string) error
	ClearAllTasks() (*ClearResult, error)
	SearchSimilarHumanTasks(prompt string, limit int, minScore float64) ([]*HumanTask, []float64, error)
}

// MongoTaskStorage implements TaskStorage using MongoDB
type MongoTaskStorage struct {
	humanTasksCollection *mongo.Collection
	agentTasksCollection *mongo.Collection
	knowledgeStorage     KnowledgeStorage
	logger               *zap.Logger
	summarizer           TaskSummarizer // AI summarization service
}

// TaskSummarizer interface for AI-powered task summarization
type TaskSummarizer interface {
	SummarizeTaskWithFallback(ctx context.Context, content string, maxFallbackLength int) string
}

// NewMongoTaskStorage creates a new MongoDB-backed task storage
// summarizer can be nil for backward compatibility (will skip AI summarization)
func NewMongoTaskStorage(db *mongo.Database, knowledgeStorage KnowledgeStorage, summarizer TaskSummarizer, logger *zap.Logger) (*MongoTaskStorage, error) {
	storage := &MongoTaskStorage{
		humanTasksCollection: db.Collection("human_tasks"),
		agentTasksCollection: db.Collection("agent_tasks"),
		knowledgeStorage:     knowledgeStorage,
		logger:               logger,
		summarizer:           summarizer,
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

	// Generate AI summary if summarizer is available
	if s.summarizer != nil {
		summary := s.summarizer.SummarizeTaskWithFallback(ctx, prompt, 200)
		task.Summary = summary
		s.logger.Debug("Generated task summary",
			zap.String("taskId", task.ID),
			zap.Int("summaryLength", len(summary)))
	} else {
		// Fallback: use first 200 chars if no summarizer
		if len(prompt) > 200 {
			task.Summary = prompt[:200] + "..."
		} else {
			task.Summary = prompt
		}
	}

	_, err := s.humanTasksCollection.InsertOne(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to insert human task: %w", err)
	}

	// Index task prompt in knowledge base for similarity search
	if s.knowledgeStorage != nil {
		collection := "human_tasks_search"
		metadata := map[string]interface{}{
			"taskId":    task.ID,
			"status":    string(task.Status),
			"createdAt": task.CreatedAt,
		}
		_, err := s.knowledgeStorage.Upsert(collection, task.Prompt, metadata)
		if err != nil {
			// Log error but don't fail task creation
			s.logger.Warn("Failed to index human task in knowledge base",
				zap.String("taskId", task.ID),
				zap.String("collection", collection),
				zap.Error(err))
		}
	}

	return task, nil
}

// CreateAgentTask creates a new agent task
func (s *MongoTaskStorage) CreateAgentTask(humanTaskID, agentName, role string, todos []TodoItemInput, contextSummary string, filesModified []string, qdrantCollections []string, priorWorkSummary string) (*AgentTask, error) {
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

	// Convert TodoItemInput to TodoItem structs
	todoItems := make([]TodoItem, len(todos))
	for i, input := range todos {
		todoItems[i] = TodoItem{
			ID:           uuid.New().String(),
			Description:  input.Description,
			Status:       TodoStatusPending,
			CreatedAt:    now,
			FilePath:     input.FilePath,
			FunctionName: input.FunctionName,
			ContextHint:  input.ContextHint,
			Notes:        input.Notes,
		}
	}

	task := &AgentTask{
		ID:                uuid.New().String(),
		HumanTaskID:       humanTaskID,
		AgentName:         agentName,
		Role:              role,
		Todos:             todoItems,
		CreatedAt:         now,
		UpdatedAt:         now,
		Status:            TaskStatusPending,
		ContextSummary:    contextSummary,
		FilesModified:     filesModified,
		QdrantCollections: qdrantCollections,
		PriorWorkSummary:  priorWorkSummary,
	}

	// Generate AI summary from context if summarizer is available
	if s.summarizer != nil && contextSummary != "" {
		summary := s.summarizer.SummarizeTaskWithFallback(ctx, contextSummary, 200)
		task.Summary = summary
		s.logger.Debug("Generated agent task summary",
			zap.String("taskId", task.ID),
			zap.String("agentName", agentName),
			zap.Int("summaryLength", len(summary)))
	} else if contextSummary != "" {
		// Fallback: use first 200 chars of context summary
		if len(contextSummary) > 200 {
			task.Summary = contextSummary[:200] + "..."
		} else {
			task.Summary = contextSummary
		}
	} else if role != "" {
		// If no context, use role as summary
		task.Summary = role
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

// AddTaskPromptNotes adds human prompt notes to an agent task
func (s *MongoTaskStorage) AddTaskPromptNotes(agentTaskID string, notes string) error {
	ctx := context.Background()
	now := time.Now().UTC()

	update := bson.M{
		"$set": bson.M{
			"humanPromptNotes":          notes,
			"humanPromptNotesAddedAt":   &now,
			"humanPromptNotesUpdatedAt": &now,
			"updatedAt":                 now,
		},
	}

	result := s.agentTasksCollection.FindOneAndUpdate(
		ctx,
		bson.M{"taskId": agentTaskID},
		update,
	)

	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return fmt.Errorf("agent task with ID %s not found", agentTaskID)
		}
		return fmt.Errorf("failed to add task prompt notes: %w", result.Err())
	}

	return nil
}

// UpdateTaskPromptNotes updates existing human prompt notes on an agent task
func (s *MongoTaskStorage) UpdateTaskPromptNotes(agentTaskID string, notes string) error {
	ctx := context.Background()
	now := time.Now().UTC()

	update := bson.M{
		"$set": bson.M{
			"humanPromptNotes":          notes,
			"humanPromptNotesUpdatedAt": &now,
			"updatedAt":                 now,
		},
	}

	result := s.agentTasksCollection.FindOneAndUpdate(
		ctx,
		bson.M{"taskId": agentTaskID},
		update,
	)

	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return fmt.Errorf("agent task with ID %s not found", agentTaskID)
		}
		return fmt.Errorf("failed to update task prompt notes: %w", result.Err())
	}

	return nil
}

// ClearTaskPromptNotes removes human prompt notes from an agent task
func (s *MongoTaskStorage) ClearTaskPromptNotes(agentTaskID string) error {
	ctx := context.Background()
	now := time.Now().UTC()

	update := bson.M{
		"$unset": bson.M{
			"humanPromptNotes":          "",
			"humanPromptNotesAddedAt":   "",
			"humanPromptNotesUpdatedAt": "",
		},
		"$set": bson.M{
			"updatedAt": now,
		},
	}

	result := s.agentTasksCollection.FindOneAndUpdate(
		ctx,
		bson.M{"taskId": agentTaskID},
		update,
	)

	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return fmt.Errorf("agent task with ID %s not found", agentTaskID)
		}
		return fmt.Errorf("failed to clear task prompt notes: %w", result.Err())
	}

	return nil
}

// AddTodoPromptNotes adds human prompt notes to a specific TODO item
func (s *MongoTaskStorage) AddTodoPromptNotes(agentTaskID string, todoID string, notes string) error {
	ctx := context.Background()
	now := time.Now().UTC()

	update := bson.M{
		"$set": bson.M{
			"todos.$[elem].humanPromptNotes":          notes,
			"todos.$[elem].humanPromptNotesAddedAt":   &now,
			"todos.$[elem].humanPromptNotesUpdatedAt": &now,
			"updatedAt": now,
		},
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"elem.id": todoID},
		},
	})

	result, err := s.agentTasksCollection.UpdateOne(
		ctx,
		bson.M{"taskId": agentTaskID},
		update,
		arrayFilters,
	)

	if err != nil {
		return fmt.Errorf("failed to add todo prompt notes: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("agent task with ID %s not found", agentTaskID)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("todo item with ID %s not found in agent task %s", todoID, agentTaskID)
	}

	return nil
}

// UpdateTodoPromptNotes updates existing human prompt notes on a specific TODO item
func (s *MongoTaskStorage) UpdateTodoPromptNotes(agentTaskID string, todoID string, notes string) error {
	ctx := context.Background()
	now := time.Now().UTC()

	update := bson.M{
		"$set": bson.M{
			"todos.$[elem].humanPromptNotes":          notes,
			"todos.$[elem].humanPromptNotesUpdatedAt": &now,
			"updatedAt": now,
		},
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"elem.id": todoID},
		},
	})

	result, err := s.agentTasksCollection.UpdateOne(
		ctx,
		bson.M{"taskId": agentTaskID},
		update,
		arrayFilters,
	)

	if err != nil {
		return fmt.Errorf("failed to update todo prompt notes: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("agent task with ID %s not found", agentTaskID)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("todo item with ID %s not found in agent task %s", todoID, agentTaskID)
	}

	return nil
}

// ClearTodoPromptNotes removes human prompt notes from a specific TODO item
func (s *MongoTaskStorage) ClearTodoPromptNotes(agentTaskID string, todoID string) error {
	ctx := context.Background()
	now := time.Now().UTC()

	update := bson.M{
		"$unset": bson.M{
			"todos.$[elem].humanPromptNotes":          "",
			"todos.$[elem].humanPromptNotesAddedAt":   "",
			"todos.$[elem].humanPromptNotesUpdatedAt": "",
		},
		"$set": bson.M{
			"updatedAt": now,
		},
	}

	arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"elem.id": todoID},
		},
	})

	result, err := s.agentTasksCollection.UpdateOne(
		ctx,
		bson.M{"taskId": agentTaskID},
		update,
		arrayFilters,
	)

	if err != nil {
		return fmt.Errorf("failed to clear todo prompt notes: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("agent task with ID %s not found", agentTaskID)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("todo item with ID %s not found in agent task %s", todoID, agentTaskID)
	}

	return nil
}

// ClearAllTasks removes all tasks from the database
func (s *MongoTaskStorage) ClearAllTasks() (*ClearResult, error) {
	ctx := context.Background()
	result := &ClearResult{
		ClearedAt: time.Now(),
	}

	// Delete all human tasks
	humanResult, err := s.humanTasksCollection.DeleteMany(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to delete human tasks: %w", err)
	}
	result.HumanTasksDeleted = humanResult.DeletedCount

	// Delete all agent tasks
	agentResult, err := s.agentTasksCollection.DeleteMany(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to delete agent tasks: %w", err)
	}
	result.AgentTasksDeleted = agentResult.DeletedCount

	return result, nil
}

// SearchSimilarHumanTasks searches for similar human tasks using semantic similarity
func (s *MongoTaskStorage) SearchSimilarHumanTasks(prompt string, limit int, minScore float64) ([]*HumanTask, []float64, error) {
	// Query knowledge base for similar prompts
	if s.knowledgeStorage == nil {
		return nil, nil, fmt.Errorf("knowledge storage not configured")
	}

	results, err := s.knowledgeStorage.Query("human_tasks_search", prompt, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query similar tasks: %w", err)
	}

	// Filter by score and fetch full tasks
	var tasks []*HumanTask
	var scores []float64

	for _, result := range results {
		if result.Score < minScore {
			continue
		}

		taskID, ok := result.Entry.Metadata["taskId"].(string)
		if !ok {
			continue
		}

		task, err := s.GetHumanTask(taskID)
		if err != nil {
			// Skip tasks that can't be retrieved
			continue
		}

		tasks = append(tasks, task)
		scores = append(scores, result.Score)
	}

	return tasks, scores, nil
}