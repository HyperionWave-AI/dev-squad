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
	ID           string     `json:"taskId" bson:"taskId"`
	Prompt       string     `json:"prompt" bson:"prompt"`
	Summary      string     `json:"summary,omitempty" bson:"summary,omitempty"` // AI-generated summary (max 100 tokens)
	AgentTaskIDs []string   `json:"agentTaskIds,omitempty" bson:"agentTaskIds,omitempty"` // Bidirectional traceability to agent tasks
	CreatedAt    time.Time  `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt" bson:"updatedAt"`
	Status       TaskStatus `json:"status" bson:"status"`
	Notes        string     `json:"notes,omitempty" bson:"notes,omitempty"`
}

// AgentTask represents a task assigned to an agent
type AgentTask struct {
	// Hierarchy fields
	ParentTaskID     *string  `json:"parentTaskId,omitempty" bson:"parentTaskId,omitempty"`
	ChildTaskIDs     []string `json:"childTaskIds,omitempty" bson:"childTaskIds,omitempty"`
	SplitReason      string   `json:"splitReason,omitempty" bson:"splitReason,omitempty"`
	SplitStrategy    string   `json:"splitStrategy,omitempty" bson:"splitStrategy,omitempty"`
	OrderIndex       int      `json:"orderIndex,omitempty" bson:"orderIndex,omitempty"`
	DependsOn        []string `json:"dependsOn,omitempty" bson:"dependsOn,omitempty"`
	
	// Complexity tracking fields
	EstimatedComplexity  int      `json:"estimatedComplexity,omitempty" bson:"estimatedComplexity,omitempty"`
	ComplexityFactors    []string `json:"complexityFactors,omitempty" bson:"complexityFactors,omitempty"`
	EstimatedLineCount   int      `json:"estimatedLineCount,omitempty" bson:"estimatedLineCount,omitempty"`
	
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

// TaskHierarchy represents the hierarchical structure of tasks
type TaskHierarchy struct {
	RootTask     *AgentTask       `json:"rootTask"`
	ChildTasks   []*TaskHierarchy `json:"childTasks,omitempty"`
	Depth        int              `json:"depth"`
	TotalTasks   int              `json:"totalTasks"`
	CompletedTasks int            `json:"completedTasks"`
}

// HierarchicalProgress represents progress tracking for hierarchical tasks
type HierarchicalProgress struct {
	RootTaskID       string                           `json:"rootTaskId"`
	TotalTasks       int                              `json:"totalTasks"`
	CompletedTasks   int                              `json:"completedTasks"`
	InProgressTasks  int                              `json:"inProgressTasks"`
	BlockedTasks     int                              `json:"blockedTasks"`
	PendingTasks     int                              `json:"pendingTasks"`
	ProgressPercent  float64                          `json:"progressPercent"`
	EstimatedTotal   int                              `json:"estimatedTotal"`
	ActualTotal      int                              `json:"actualTotal"`
	TasksByLevel     map[int][]*AgentTask             `json:"tasksByLevel"`
	DependencyGraph  map[string][]string              `json:"dependencyGraph"`
	CriticalPath     []string                         `json:"criticalPath"`
	UpdatedAt        time.Time                        `json:"updatedAt"`
}

// ChildTaskParams represents parameters for creating a child task
type ChildTaskParams struct {
	AgentName           string          `json:"agentName"`
	Role                string          `json:"role"`
	Todos               []TodoItemInput `json:"todos"`
	ContextSummary      string          `json:"contextSummary"`
	FilesModified       []string        `json:"filesModified,omitempty"`
	QdrantCollections   []string        `json:"qdrantCollections,omitempty"`
	PriorWorkSummary    string          `json:"priorWorkSummary,omitempty"`
	SplitReason         string          `json:"splitReason"`
	SplitStrategy       string          `json:"splitStrategy"`
	OrderIndex          int             `json:"orderIndex"`
	DependsOn           []string        `json:"dependsOn,omitempty"`
	EstimatedComplexity int             `json:"estimatedComplexity,omitempty"`
	ComplexityFactors   []string        `json:"complexityFactors,omitempty"`
	EstimatedLineCount  int             `json:"estimatedLineCount,omitempty"`
}

// TaskStorage provides storage interface for tasks
type TaskStorage interface {
	CreateHumanTask(prompt string) (*HumanTask, error)
	CreateAgentTask(humanTaskID, agentName, role string, todos []TodoItemInput, contextSummary string, filesModified []string, qdrantCollections []string, priorWorkSummary string) (*AgentTask, error)
	GetHumanTask(taskID string) (*HumanTask, error)
	GetAgentTask(taskID string) (*AgentTask, error)
	GetAgentTasksByName(agentName string) ([]*AgentTask, error)
	ListAllHumanTasks() []*HumanTask // Deprecated: Use ListHumanTasks with filter
	ListAllAgentTasks() []*AgentTask // Deprecated: Use ListAgentTasks with filter
	ListHumanTasks(filter bson.M) ([]*HumanTask, error)
	ListAgentTasks(filter bson.M, offset, limit int) ([]*AgentTask, int, error)
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
	
	// Hierarchical task management methods
	CreateChildAgentTask(parentTaskID string, params ChildTaskParams) (*AgentTask, error)
	GetChildTasks(parentTaskID string) ([]*AgentTask, error)
	GetTaskHierarchy(rootTaskID string) (*TaskHierarchy, error)
	AddTaskDependency(taskID, dependsOnTaskID string) error
	RemoveTaskDependency(taskID, dependsOnTaskID string) error
	GetBlockedTasks() ([]*AgentTask, error)
	GetDependencyChain(taskID string) ([]string, error)
	GetHierarchicalProgress(rootTaskID string) (*HierarchicalProgress, error)
	UpdateHierarchicalStatus(taskID string, status TaskStatus) error
}
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

	// Index on agentTasks.parentTaskId for hierarchical queries
	_, err = storage.agentTasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "parentTaskId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create parent task ID index: %w", err)
	}

	// Index on agentTasks.dependsOn for dependency queries
	_, err = storage.agentTasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "dependsOn", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create depends on index: %w", err)
	}

	// Compound index for hierarchical progress queries (parentTaskId + status + orderIndex)
	_, err = storage.agentTasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "parentTaskId", Value: 1},
			{Key: "status", Value: 1},
			{Key: "orderIndex", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create hierarchical progress index: %w", err)
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
		// Pass taskId as a separate parameter
		taskIdPtr := &task.ID
		_, err := s.knowledgeStorage.Upsert(collection, task.Prompt, metadata, taskIdPtr)
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

	// Update human task with bidirectional traceability
	// Push this agent task ID to the human task's AgentTaskIDs array
	_, err = s.humanTasksCollection.UpdateOne(
		ctx,
		bson.M{"taskId": humanTaskID},
		bson.M{
			"$push": bson.M{"agentTaskIds": task.ID},
			"$set":  bson.M{"updatedAt": now},
		},
	)
	if err != nil {
		// Log error but don't fail task creation since agent task is already created
		s.logger.Warn("Failed to update human task with agent task ID",
			zap.String("humanTaskId", humanTaskID),
			zap.String("agentTaskId", task.ID),
			zap.Error(err))
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
// Deprecated: Use ListHumanTasks with filter for better performance
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

// ListHumanTasks returns human tasks matching the given filter, sorted by createdAt descending (newest first)
func (s *MongoTaskStorage) ListHumanTasks(filter bson.M) ([]*HumanTask, error) {
	ctx := context.Background()

	// If filter is nil, use empty filter to get all
	if filter == nil {
		filter = bson.M{}
	}

	// Sort by createdAt descending (newest first)
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := s.humanTasksCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to query human tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*HumanTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode human tasks: %w", err)
	}

	return tasks, nil
}

// ListAllAgentTasks returns all agent tasks
// Deprecated: Use ListAgentTasks with filter for better performance
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

// ListAgentTasks returns agent tasks matching the given filter with pagination
// Returns tasks, total count, and error
func (s *MongoTaskStorage) ListAgentTasks(filter bson.M, offset, limit int) ([]*AgentTask, int, error) {
	ctx := context.Background()

	// If filter is nil, use empty filter to get all
	if filter == nil {
		filter = bson.M{}
	}

	// Count total matching documents
	totalCount, err := s.agentTasksCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count agent tasks: %w", err)
	}

	// Apply pagination options
	findOptions := options.Find()
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limit))
	// Sort by createdAt descending (newest first)
	findOptions.SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := s.agentTasksCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query agent tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*AgentTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, 0, fmt.Errorf("failed to decode agent tasks: %w", err)
	}

	return tasks, int(totalCount), nil
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

	// No taskId filtering for global search across all tasks
	results, err := s.knowledgeStorage.Query("human_tasks_search", prompt, limit, nil)
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
// CreateChildAgentTask creates a new child agent task with hierarchical relationships
func (s *MongoTaskStorage) CreateChildAgentTask(parentTaskID string, params ChildTaskParams) (*AgentTask, error) {
	ctx := context.Background()

	// Verify parent task exists
	parentTask, err := s.GetAgentTask(parentTaskID)
	if err != nil {
		return nil, fmt.Errorf("parent task not found: %w", err)
	}

	now := time.Now().UTC()
	childTask := &AgentTask{
		ID:                  uuid.New().String(),
		HumanTaskID:         parentTask.HumanTaskID,
		ParentTaskID:        &parentTaskID,
		AgentName:           params.AgentName,
		Role:                params.Role,
		SplitReason:         params.SplitReason,
		SplitStrategy:       params.SplitStrategy,
		OrderIndex:          params.OrderIndex,
		DependsOn:           params.DependsOn,
		EstimatedComplexity: params.EstimatedComplexity,
		ComplexityFactors:   params.ComplexityFactors,
		EstimatedLineCount:  params.EstimatedLineCount,
		CreatedAt:           now,
		UpdatedAt:           now,
		Status:              TaskStatusPending,
		ContextSummary:      params.ContextSummary,
		FilesModified:       params.FilesModified,
		QdrantCollections:   params.QdrantCollections,
		PriorWorkSummary:    params.PriorWorkSummary,
	}

	// Convert TodoItemInput to TodoItem
	for _, todoInput := range params.Todos {
		todo := TodoItem{
			ID:           uuid.New().String(),
			Description:  todoInput.Description,
			Status:       TodoStatusPending,
			CreatedAt:    now,
			FilePath:     todoInput.FilePath,
			FunctionName: todoInput.FunctionName,
			ContextHint:  todoInput.ContextHint,
			Notes:        todoInput.Notes,
		}
		childTask.Todos = append(childTask.Todos, todo)
	}

	// Generate summary if summarizer is available
	if s.summarizer != nil {
		content := fmt.Sprintf("Child task for %s: %s. Context: %s", params.AgentName, params.Role, params.ContextSummary)
		childTask.Summary = s.summarizer.SummarizeTaskWithFallback(ctx, content, 100)
	}

	// Insert child task
	_, err = s.agentTasksCollection.InsertOne(ctx, childTask)
	if err != nil {
		return nil, fmt.Errorf("failed to create child task: %w", err)
	}

	// Update parent task to include child ID
	update := bson.M{
		"$push": bson.M{"childTaskIds": childTask.ID},
		"$set":  bson.M{"updatedAt": now},
	}
	_, err = s.agentTasksCollection.UpdateOne(ctx, bson.M{"taskId": parentTaskID}, update)
	if err != nil {
		s.logger.Error("Failed to update parent task with child ID", zap.Error(err))
	}

	return childTask, nil
}

// GetChildTasks retrieves all child tasks for a given parent task
func (s *MongoTaskStorage) GetChildTasks(parentTaskID string) ([]*AgentTask, error) {
	ctx := context.Background()

	filter := bson.M{"parentTaskId": parentTaskID}
	cursor, err := s.agentTasksCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query child tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*AgentTask
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode child tasks: %w", err)
	}

	return tasks, nil
}

// GetTaskHierarchy retrieves the complete hierarchical structure starting from a root task
func (s *MongoTaskStorage) GetTaskHierarchy(rootTaskID string) (*TaskHierarchy, error) {
	rootTask, err := s.GetAgentTask(rootTaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get root task: %w", err)
	}

	hierarchy := &TaskHierarchy{
		RootTask:   rootTask,
		Depth:      0,
		TotalTasks: 1,
	}

	if rootTask.Status == TaskStatusCompleted {
		hierarchy.CompletedTasks = 1
	}

	// Recursively build hierarchy
	err = s.buildHierarchy(hierarchy, rootTaskID, 1)
	if err != nil {
		return nil, err
	}

	return hierarchy, nil
}

// buildHierarchy is a helper function to recursively build task hierarchy
func (s *MongoTaskStorage) buildHierarchy(parent *TaskHierarchy, taskID string, depth int) error {
	childTasks, err := s.GetChildTasks(taskID)
	if err != nil {
		return err
	}

	for _, childTask := range childTasks {
		childHierarchy := &TaskHierarchy{
			RootTask:   childTask,
			Depth:      depth,
			TotalTasks: 1,
		}

		if childTask.Status == TaskStatusCompleted {
			childHierarchy.CompletedTasks = 1
		}

		// Recursively build child hierarchy
		err = s.buildHierarchy(childHierarchy, childTask.ID, depth+1)
		if err != nil {
			return err
		}

		parent.ChildTasks = append(parent.ChildTasks, childHierarchy)
		parent.TotalTasks += childHierarchy.TotalTasks
		parent.CompletedTasks += childHierarchy.CompletedTasks
	}

	return nil
}

// AddTaskDependency adds a dependency relationship between tasks
func (s *MongoTaskStorage) AddTaskDependency(taskID, dependsOnTaskID string) error {
	ctx := context.Background()
	now := time.Now().UTC()

	// Verify both tasks exist
	_, err := s.GetAgentTask(taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	_, err = s.GetAgentTask(dependsOnTaskID)
	if err != nil {
		return fmt.Errorf("dependency task not found: %w", err)
	}

	// Check for circular dependencies
	chain, err := s.GetDependencyChain(dependsOnTaskID)
	if err != nil {
		return fmt.Errorf("failed to check dependency chain: %w", err)
	}

	for _, depID := range chain {
		if depID == taskID {
			return fmt.Errorf("circular dependency detected")
		}
	}

	// Add dependency
	update := bson.M{
		"$addToSet": bson.M{"dependsOn": dependsOnTaskID},
		"$set":      bson.M{"updatedAt": now},
	}
	_, err = s.agentTasksCollection.UpdateOne(ctx, bson.M{"taskId": taskID}, update)
	if err != nil {
		return fmt.Errorf("failed to add dependency: %w", err)
	}

	return nil
}

// RemoveTaskDependency removes a dependency relationship between tasks
func (s *MongoTaskStorage) RemoveTaskDependency(taskID, dependsOnTaskID string) error {
	ctx := context.Background()
	now := time.Now().UTC()

	update := bson.M{
		"$pull": bson.M{"dependsOn": dependsOnTaskID},
		"$set":  bson.M{"updatedAt": now},
	}
	_, err := s.agentTasksCollection.UpdateOne(ctx, bson.M{"taskId": taskID}, update)
	if err != nil {
		return fmt.Errorf("failed to remove dependency: %w", err)
	}

	return nil
}

// GetBlockedTasks retrieves all tasks that are blocked by dependencies
func (s *MongoTaskStorage) GetBlockedTasks() ([]*AgentTask, error) {
	ctx := context.Background()

	// Find tasks that have dependencies and are not completed
	filter := bson.M{
		"dependsOn": bson.M{"$exists": true, "$ne": []string{}},
		"status":    bson.M{"$ne": TaskStatusCompleted},
	}

	cursor, err := s.agentTasksCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query blocked tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var allTasks []*AgentTask
	if err = cursor.All(ctx, &allTasks); err != nil {
		return nil, fmt.Errorf("failed to decode tasks: %w", err)
	}

	var blockedTasks []*AgentTask
	for _, task := range allTasks {
		isBlocked, err := s.isTaskBlocked(task)
		if err != nil {
			s.logger.Error("Failed to check if task is blocked", zap.String("taskId", task.ID), zap.Error(err))
			continue
		}
		if isBlocked {
			blockedTasks = append(blockedTasks, task)
		}
	}

	return blockedTasks, nil
}

// isTaskBlocked checks if a task is blocked by incomplete dependencies
func (s *MongoTaskStorage) isTaskBlocked(task *AgentTask) (bool, error) {
	for _, depID := range task.DependsOn {
		depTask, err := s.GetAgentTask(depID)
		if err != nil {
			return true, err // If dependency doesn't exist, task is blocked
		}
		if depTask.Status != TaskStatusCompleted {
			return true, nil // If dependency is not completed, task is blocked
		}
	}
	return false, nil
}

// GetDependencyChain retrieves the complete dependency chain for a task
func (s *MongoTaskStorage) GetDependencyChain(taskID string) ([]string, error) {
	visited := make(map[string]bool)
	var chain []string

	err := s.buildDependencyChain(taskID, visited, &chain)
	if err != nil {
		return nil, err
	}

	return chain, nil
}

// buildDependencyChain recursively builds the dependency chain
func (s *MongoTaskStorage) buildDependencyChain(taskID string, visited map[string]bool, chain *[]string) error {
	if visited[taskID] {
		return nil // Already processed
	}

	visited[taskID] = true
	*chain = append(*chain, taskID)

	task, err := s.GetAgentTask(taskID)
	if err != nil {
		return err
	}

	for _, depID := range task.DependsOn {
		err = s.buildDependencyChain(depID, visited, chain)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetHierarchicalProgress calculates and returns progress information for a hierarchical task structure
func (s *MongoTaskStorage) GetHierarchicalProgress(rootTaskID string) (*HierarchicalProgress, error) {
	hierarchy, err := s.GetTaskHierarchy(rootTaskID)
	if err != nil {
		return nil, err
	}

	progress := &HierarchicalProgress{
		RootTaskID:      rootTaskID,
		TotalTasks:      hierarchy.TotalTasks,
		CompletedTasks:  hierarchy.CompletedTasks,
		TasksByLevel:    make(map[int][]*AgentTask),
		DependencyGraph: make(map[string][]string),
		UpdatedAt:       time.Now().UTC(),
	}

	// Calculate progress statistics
	s.calculateProgressStats(hierarchy, progress)

	// Calculate progress percentage
	if progress.TotalTasks > 0 {
		progress.ProgressPercent = float64(progress.CompletedTasks) / float64(progress.TotalTasks) * 100
	}

	return progress, nil
}

// calculateProgressStats recursively calculates progress statistics
func (s *MongoTaskStorage) calculateProgressStats(hierarchy *TaskHierarchy, progress *HierarchicalProgress) {
	task := hierarchy.RootTask

	// Add task to level map
	if progress.TasksByLevel[hierarchy.Depth] == nil {
		progress.TasksByLevel[hierarchy.Depth] = []*AgentTask{}
	}
	progress.TasksByLevel[hierarchy.Depth] = append(progress.TasksByLevel[hierarchy.Depth], task)

	// Add to dependency graph
	progress.DependencyGraph[task.ID] = task.DependsOn

	// Count task status
	switch task.Status {
	case TaskStatusInProgress:
		progress.InProgressTasks++
	case TaskStatusBlocked:
		progress.BlockedTasks++
	case TaskStatusPending:
		progress.PendingTasks++
	}

	// Process child tasks
	for _, child := range hierarchy.ChildTasks {
		s.calculateProgressStats(child, progress)
	}
}

// UpdateHierarchicalStatus updates a task's status and propagates changes up the hierarchy
func (s *MongoTaskStorage) UpdateHierarchicalStatus(taskID string, status TaskStatus) error {
	// Update the task status
	err := s.UpdateTaskStatus(taskID, status, "")
	if err != nil {
		return err
	}

	// Get the task to check for parent
	task, err := s.GetAgentTask(taskID)
	if err != nil {
		return err
	}

	// If task has a parent, check if parent status needs updating
	if task.ParentTaskID != nil {
		err = s.updateParentStatus(*task.ParentTaskID)
		if err != nil {
			s.logger.Error("Failed to update parent status", zap.String("parentId", *task.ParentTaskID), zap.Error(err))
		}
	}

	return nil
}

// updateParentStatus checks and updates parent task status based on child task statuses
func (s *MongoTaskStorage) updateParentStatus(parentTaskID string) error {
	childTasks, err := s.GetChildTasks(parentTaskID)
	if err != nil {
		return err
	}

	if len(childTasks) == 0 {
		return nil // No children, no update needed
	}

	// Check child statuses
	allCompleted := true
	anyInProgress := false
	anyBlocked := false

	for _, child := range childTasks {
		switch child.Status {
		case TaskStatusCompleted:
			// Keep checking
		case TaskStatusInProgress:
			allCompleted = false
			anyInProgress = true
		case TaskStatusBlocked:
			allCompleted = false
			anyBlocked = true
		case TaskStatusPending:
			allCompleted = false
		}
	}

	// Determine parent status
	var newStatus TaskStatus
	if allCompleted {
		newStatus = TaskStatusCompleted
	} else if anyBlocked {
		newStatus = TaskStatusBlocked
	} else if anyInProgress {
		newStatus = TaskStatusInProgress
	} else {
		newStatus = TaskStatusPending
	}

	// Update parent status
	return s.UpdateTaskStatus(parentTaskID, newStatus, "Status updated based on child task progress")
}
