package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"hyper/internal/ai-service"
	"hyper/internal/mcp/handlers"
	"hyper/internal/mcp/storage"
	"hyper/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// CoordinatorTools provides MCP coordinator tool executors for LangChain
type CoordinatorTools struct {
	taskStorage      storage.TaskStorage
	knowledgeStorage storage.KnowledgeStorage
}

// NewCoordinatorTools creates a new coordinator tools handler
func NewCoordinatorTools(taskStorage storage.TaskStorage, knowledgeStorage storage.KnowledgeStorage) *CoordinatorTools {
	return &CoordinatorTools{
		taskStorage:      taskStorage,
		knowledgeStorage: knowledgeStorage,
	}
}

// CreateAgentTaskTool implements the ToolExecutor interface
type CreateAgentTaskTool struct {
	storage storage.TaskStorage
}

func (t *CreateAgentTaskTool) Name() string {
	return "create_agent_task"
}

func (t *CreateAgentTaskTool) Description() string {
	return "Create a new agent task linked to a human task. Returns task ID. IMPORTANT: Provide specific file paths in filesModified (e.g., './ui/src/components/TaskCard.tsx') and detailed context in contextSummary including WHAT to change, WHERE (file:line), and HOW. The more specific your context, the less time agents waste exploring. Required: humanTaskId, agentName, role, todos. Optional: contextSummary, filesModified, qdrantCollections, priorWorkSummary."
}

func (t *CreateAgentTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"humanTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Parent human task ID (UUID format)",
			},
			"agentName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the agent assigned to this task",
			},
			"role": map[string]interface{}{
				"type":        "string",
				"description": "Agent's role/responsibility for this task",
			},
			"todos": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "List of TODO items (tasks to complete)",
			},
			"contextSummary": map[string]interface{}{
				"type":        "string",
				"description": "200-word summary with SPECIFICS: What to change, where to change it (file paths, line numbers if known), how to implement it, what patterns to follow. Example: 'Add delete button to TaskCard.tsx around line 45, next to the edit button. Use the existing IconButton component with DeleteIcon. Wire it to deleteTask prop passed from parent. Follow the pattern used for edit button in same file.'",
			},
			"filesModified": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "EXACT file paths that will be modified (e.g., ['./ui/src/components/TaskCard.tsx', './ui/src/components/KanbanTaskCard.tsx']). Be SPECIFIC - this reduces agent exploration time from minutes to seconds. Used for validation.",
			},
			"qdrantCollections": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Suggested Qdrant collections to query if technical patterns needed",
			},
			"priorWorkSummary": map[string]interface{}{
				"type":        "string",
				"description": "Summary of previous agent's work and key decisions (for multi-phase tasks)",
			},
		},
		"required": []string{"humanTaskId", "agentName", "role", "todos"},
	}
}

func (t *CreateAgentTaskTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract and validate required fields
	humanTaskID, ok := input["humanTaskId"].(string)
	if !ok || humanTaskID == "" {
		return nil, fmt.Errorf("humanTaskId is required and must be a string")
	}

	agentName, ok := input["agentName"].(string)
	if !ok || agentName == "" {
		return nil, fmt.Errorf("agentName is required and must be a string")
	}

	role, ok := input["role"].(string)
	if !ok || role == "" {
		return nil, fmt.Errorf("role is required and must be a string")
	}

	todosRaw, ok := input["todos"]
	if !ok {
		return nil, fmt.Errorf("todos is required")
	}

	// Convert todos to []string
	var todos []string
	switch v := todosRaw.(type) {
	case []interface{}:
		todos = make([]string, len(v))
		for i, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("todos[%d] must be a string", i)
			}
			todos[i] = str
		}
	case []string:
		todos = v
	default:
		return nil, fmt.Errorf("todos must be an array of strings")
	}

	if len(todos) == 0 {
		return nil, fmt.Errorf("todos must not be empty")
	}

	// Convert todos to storage format
	todoItems := make([]storage.TodoItemInput, len(todos))
	for i, todo := range todos {
		todoItems[i] = storage.TodoItemInput{
			Description: todo,
		}
	}

	// Extract optional fields
	contextSummary, _ := input["contextSummary"].(string)
	priorWorkSummary, _ := input["priorWorkSummary"].(string)

	var filesModified []string
	if fm, ok := input["filesModified"].([]interface{}); ok {
		filesModified = make([]string, len(fm))
		for i, f := range fm {
			if str, ok := f.(string); ok {
				filesModified[i] = str
			}
		}
	}

	var qdrantCollections []string
	if qc, ok := input["qdrantCollections"].([]interface{}); ok {
		qdrantCollections = make([]string, len(qc))
		for i, c := range qc {
			if str, ok := c.(string); ok {
				qdrantCollections[i] = str
			}
		}
	}

	// Create agent task via storage
	task, err := t.storage.CreateAgentTask(
		humanTaskID,
		agentName,
		role,
		todoItems,
		contextSummary,
		filesModified,
		qdrantCollections,
		priorWorkSummary,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent task: %w", err)
	}

	// Return task summary
	return map[string]interface{}{
		"taskId":     task.ID,
		"agentName":  task.AgentName,
		"role":       task.Role,
		"status":     task.Status,
		"todosCount": len(task.Todos),
		"createdAt":  task.CreatedAt,
	}, nil
}

// ListAgentTasksTool implements the ToolExecutor interface
type ListAgentTasksTool struct {
	storage storage.TaskStorage
}

func (t *ListAgentTasksTool) Name() string {
	return "list_agent_tasks"
}

func (t *ListAgentTasksTool) Description() string {
	return "List agent tasks with optional filters. Returns up to 20 tasks with details. Supports pagination via offset/limit. Use to check task status, find assignments, or review progress."
}

func (t *ListAgentTasksTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentName": map[string]interface{}{
				"type":        "string",
				"description": "Filter by agent name (optional)",
			},
			"humanTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Filter by parent human task ID (optional)",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Number of tasks to skip for pagination (default: 0)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of tasks to return (default: 20, max: 20)",
			},
		},
	}
}

func (t *ListAgentTasksTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract filter parameters
	agentName, _ := input["agentName"].(string)
	humanTaskID, _ := input["humanTaskId"].(string)

	// Extract pagination parameters
	offset := 0
	if o, ok := input["offset"].(float64); ok && o >= 0 {
		offset = int(o)
	}

	limit := 20
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 20 {
			limit = 20 // Enforce max limit per task context
		}
	}

	// Get all tasks
	allTasks := t.storage.ListAllAgentTasks()

	// Apply filters
	var filteredTasks []*storage.AgentTask
	for _, task := range allTasks {
		if humanTaskID != "" && task.HumanTaskID != humanTaskID {
			continue
		}
		if agentName != "" && task.AgentName != agentName {
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}

	// Apply pagination
	totalCount := len(filteredTasks)
	endIndex := offset + limit
	if offset > totalCount {
		offset = totalCount
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	paginatedTasks := filteredTasks[offset:endIndex]

	// Format response
	return map[string]interface{}{
		"tasks":      paginatedTasks,
		"count":      len(paginatedTasks),
		"totalCount": totalCount,
		"offset":     offset,
		"limit":      limit,
	}, nil
}

// QueryKnowledgeTool implements the ToolExecutor interface
type QueryKnowledgeTool struct {
	storage storage.KnowledgeStorage
}

func (t *QueryKnowledgeTool) Name() string {
	return "query_knowledge"
}

func (t *QueryKnowledgeTool) Description() string {
	return "Query the coordinator knowledge base for relevant information. Returns top matches with similarity scores. Limit: 10 results max. Use to find existing solutions, patterns, or context before implementing."
}

func (t *QueryKnowledgeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{
				"type":        "string",
				"description": "Collection name to query (e.g., 'technical-knowledge', 'task:hyperion://task/human/{taskId}')",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query text (natural language)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results (default: 5, max: 10)",
			},
		},
		"required": []string{"collection", "query"},
	}
}

func (t *QueryKnowledgeTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract and validate required fields
	collection, ok := input["collection"].(string)
	if !ok || collection == "" {
		return nil, fmt.Errorf("collection is required and must be a string")
	}

	query, ok := input["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required and must be a string")
	}

	// Extract optional limit
	limit := 5
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 10 {
			limit = 10 // Enforce max limit per task context
		}
	}

	// Query knowledge storage
	results, err := t.storage.Query(collection, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge: %w", err)
	}

	// Format results
	type KnowledgeResult struct {
		ID         string                 `json:"id"`
		Collection string                 `json:"collection"`
		Text       string                 `json:"text"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
		Score      float64                `json:"score"`
	}

	formattedResults := make([]KnowledgeResult, len(results))
	for i, result := range results {
		formattedResults[i] = KnowledgeResult{
			ID:         result.Entry.ID,
			Collection: result.Entry.Collection,
			Text:       result.Entry.Text,
			Metadata:   result.Entry.Metadata,
			Score:      result.Score,
		}
	}

	return formattedResults, nil
}

// UpsertKnowledgeTool implements the ToolExecutor interface
type UpsertKnowledgeTool struct {
	storage storage.KnowledgeStorage
}

func (t *UpsertKnowledgeTool) Name() string {
	return "coordinator_upsert_knowledge"
}

func (t *UpsertKnowledgeTool) Description() string {
	return "Store knowledge in the coordinator knowledge base. Use for storing task context, ADRs, data contracts, and coordination information. Returns entry ID and collection."
}

func (t *UpsertKnowledgeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{
				"type":        "string",
				"description": "Collection name (e.g., 'task:taskURI', 'adr', 'data-contracts')",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Content to store",
			},
			"metadata": map[string]interface{}{
				"type":        "object",
				"description": "Optional metadata (taskId, agentName, timestamp, etc.)",
			},
		},
		"required": []string{"collection", "text"},
	}
}

func (t *UpsertKnowledgeTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	collection, ok := input["collection"].(string)
	if !ok || collection == "" {
		return nil, fmt.Errorf("collection is required and must be a string")
	}

	text, ok := input["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required and must be a string")
	}

	var metadata map[string]interface{}
	if m, ok := input["metadata"].(map[string]interface{}); ok {
		metadata = m
	}

	entry, err := t.storage.Upsert(collection, text, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert knowledge: %w", err)
	}

	return map[string]interface{}{
		"id":         entry.ID,
		"collection": entry.Collection,
		"createdAt":  entry.CreatedAt,
	}, nil
}

// GetPopularCollectionsTool implements the ToolExecutor interface
type GetPopularCollectionsTool struct {
	storage storage.KnowledgeStorage
}

func (t *GetPopularCollectionsTool) Name() string {
	return "coordinator_get_popular_collections"
}

func (t *GetPopularCollectionsTool) Description() string {
	return "Get top N knowledge collections by entry count. Use for discovering which collections contain the most knowledge. Returns collection names with entry counts."
}

func (t *GetPopularCollectionsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of collections to return (default: 5)",
			},
		},
	}
}

func (t *GetPopularCollectionsTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	limit := 5
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	stats, err := t.storage.GetPopularCollections(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular collections: %w", err)
	}

	if stats == nil || len(stats) == 0 {
		return map[string]interface{}{
			"collections":  []interface{}{},
			"message":      "No collections with entries yet",
			"totalDefined": 14,
		}, nil
	}

	return stats, nil
}

// CreateHumanTaskTool implements the ToolExecutor interface
type CreateHumanTaskTool struct {
	storage storage.TaskStorage
}

func (t *CreateHumanTaskTool) Name() string {
	return "coordinator_create_human_task"
}

func (t *CreateHumanTaskTool) Description() string {
	return "Create a new human task with the original user prompt. Returns task ID. Use this as the first step when a user makes a request."
}

func (t *CreateHumanTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "Original human request/prompt",
			},
			"forceCreate": map[string]interface{}{
				"type":        "boolean",
				"description": "Set to true to create task despite similar existing tasks (default: false)",
			},
		},
		"required": []string{"prompt"},
	}
}

func (t *CreateHumanTaskTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	prompt, ok := input["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("prompt is required and must be a string")
	}

	forceCreate := false
	if fc, ok := input["forceCreate"].(bool); ok {
		forceCreate = fc
	}

	// Check for similar tasks unless forceCreate is true
	if !forceCreate {
		similarTasks, scores, err := t.storage.SearchSimilarHumanTasks(prompt, 5, 0.75)
		if err == nil && len(similarTasks) > 0 {
			// Found similar tasks - return them instead of creating
			formattedTasks := make([]map[string]interface{}, len(similarTasks))
			for i, task := range similarTasks {
				formattedTasks[i] = map[string]interface{}{
					"taskId":     task.ID,
					"prompt":     task.Prompt,
					"status":     task.Status,
					"createdAt":  task.CreatedAt,
					"similarity": scores[i],
				}
			}

			return map[string]interface{}{
				"similarTasksFound": true,
				"similarTasks":      formattedTasks,
				"message":           fmt.Sprintf("Found %d similar task(s). Set forceCreate=true to create anyway, or use an existing task.", len(similarTasks)),
			}, nil
		}
	}

	// No similar tasks or forceCreate=true - proceed with creation
	task, err := t.storage.CreateHumanTask(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create human task: %w", err)
	}

	return map[string]interface{}{
		"similarTasksFound": false,
		"taskId":            task.ID,
		"status":            task.Status,
		"prompt":            task.Prompt,
		"createdAt":         task.CreatedAt,
	}, nil
}

// UpdateTaskStatusTool implements the ToolExecutor interface
type UpdateTaskStatusTool struct {
	storage storage.TaskStorage
}

func (t *UpdateTaskStatusTool) Name() string {
	return "coordinator_update_task_status"
}

func (t *UpdateTaskStatusTool) Description() string {
	return "Update the status of any task (human or agent). Status values: pending, in_progress, completed, blocked. Use to track task progress."
}

func (t *UpdateTaskStatusTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"taskId": map[string]interface{}{
				"type":        "string",
				"description": "Task ID to update (UUID)",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "New status (pending, in_progress, completed, blocked)",
				"enum":        []string{"pending", "in_progress", "completed", "blocked"},
			},
			"notes": map[string]interface{}{
				"type":        "string",
				"description": "Optional progress notes",
			},
		},
		"required": []string{"taskId", "status"},
	}
}

func (t *UpdateTaskStatusTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	taskID, ok := input["taskId"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("taskId is required and must be a string")
	}

	statusStr, ok := input["status"].(string)
	if !ok || statusStr == "" {
		return nil, fmt.Errorf("status is required and must be one of: pending, in_progress, completed, blocked")
	}

	status := storage.TaskStatus(statusStr)
	notes, _ := input["notes"].(string)

	err := t.storage.UpdateTaskStatus(taskID, status, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	return map[string]interface{}{
		"taskId": taskID,
		"status": status,
		"notes":  notes,
	}, nil
}

// UpdateTodoStatusTool implements the ToolExecutor interface
type UpdateTodoStatusTool struct {
	storage storage.TaskStorage
}

func (t *UpdateTodoStatusTool) Name() string {
	return "coordinator_update_todo_status"
}

func (t *UpdateTodoStatusTool) Description() string {
	return "Update the status of a specific TODO item within an agent task. Status values: pending, in_progress, completed. When all TODOs are completed, the agent task is automatically marked as completed."
}

func (t *UpdateTodoStatusTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task ID (UUID)",
			},
			"todoId": map[string]interface{}{
				"type":        "string",
				"description": "TODO item ID (UUID)",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "New status (pending, in_progress, completed)",
				"enum":        []string{"pending", "in_progress", "completed"},
			},
			"notes": map[string]interface{}{
				"type":        "string",
				"description": "Optional progress notes for this TODO",
			},
		},
		"required": []string{"agentTaskId", "todoId", "status"},
	}
}

func (t *UpdateTodoStatusTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	todoID, ok := input["todoId"].(string)
	if !ok || todoID == "" {
		return nil, fmt.Errorf("todoId is required and must be a string")
	}

	statusStr, ok := input["status"].(string)
	if !ok || statusStr == "" {
		return nil, fmt.Errorf("status is required and must be one of: pending, in_progress, completed")
	}

	status := storage.TodoStatus(statusStr)
	notes, _ := input["notes"].(string)

	err := t.storage.UpdateTodoStatus(agentTaskID, todoID, status, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to update TODO status: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"todoId":      todoID,
		"status":      status,
		"notes":       notes,
	}, nil
}

// ListHumanTasksTool implements the ToolExecutor interface
type ListHumanTasksTool struct {
	storage storage.TaskStorage
}

func (t *ListHumanTasksTool) Name() string {
	return "coordinator_list_human_tasks"
}

func (t *ListHumanTasksTool) Description() string {
	return "List all human tasks from the coordinator database. Returns array of tasks with all fields."
}

func (t *ListHumanTasksTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListHumanTasksTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	tasks := t.storage.ListAllHumanTasks()
	return map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}, nil
}

// GetAgentTaskTool implements the ToolExecutor interface
type GetAgentTaskTool struct {
	storage storage.TaskStorage
}

func (t *GetAgentTaskTool) Name() string {
	return "coordinator_get_agent_task"
}

func (t *GetAgentTaskTool) Description() string {
	return "Get a single agent task by ID with full, untruncated content. Use this to retrieve complete task details when coordinator_list_agent_tasks shows truncated fields."
}

func (t *GetAgentTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"taskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task ID (UUID)",
			},
		},
		"required": []string{"taskId"},
	}
}

func (t *GetAgentTaskTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	taskID, ok := input["taskId"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("taskId is required and must be a string")
	}

	task, err := t.storage.GetAgentTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent task: %w", err)
	}

	return map[string]interface{}{
		"task": task,
	}, nil
}

// AddTaskPromptNotesTool implements the ToolExecutor interface
type AddTaskPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *AddTaskPromptNotesTool) Name() string {
	return "coordinator_add_task_prompt_notes"
}

func (t *AddTaskPromptNotesTool) Description() string {
	return "Add human guidance notes to an agent task. Use to provide additional context or instructions to the agent working on the task."
}

func (t *AddTaskPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"promptNotes": map[string]interface{}{
				"type":        "string",
				"description": "Human guidance notes, markdown supported",
			},
		},
		"required": []string{"agentTaskId", "promptNotes"},
	}
}

func (t *AddTaskPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	promptNotes, ok := input["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return nil, fmt.Errorf("promptNotes is required and must be a string")
	}

	// Validate and sanitize prompt notes
	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	err = t.storage.AddTaskPromptNotes(agentTaskID, sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to add prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"message":     "Prompt notes added successfully",
	}, nil
}

// UpdateTaskPromptNotesTool implements the ToolExecutor interface
type UpdateTaskPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *UpdateTaskPromptNotesTool) Name() string {
	return "coordinator_update_task_prompt_notes"
}

func (t *UpdateTaskPromptNotesTool) Description() string {
	return "Update existing human guidance notes on an agent task. Use to modify previously added guidance."
}

func (t *UpdateTaskPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"promptNotes": map[string]interface{}{
				"type":        "string",
				"description": "Human guidance notes, markdown supported",
			},
		},
		"required": []string{"agentTaskId", "promptNotes"},
	}
}

func (t *UpdateTaskPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	promptNotes, ok := input["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return nil, fmt.Errorf("promptNotes is required and must be a string")
	}

	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	err = t.storage.UpdateTaskPromptNotes(agentTaskID, sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to update prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"message":     "Prompt notes updated successfully",
	}, nil
}

// ClearTaskPromptNotesTool implements the ToolExecutor interface
type ClearTaskPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *ClearTaskPromptNotesTool) Name() string {
	return "coordinator_clear_task_prompt_notes"
}

func (t *ClearTaskPromptNotesTool) Description() string {
	return "Clear/remove human guidance notes from an agent task."
}

func (t *ClearTaskPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
		},
		"required": []string{"agentTaskId"},
	}
}

func (t *ClearTaskPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	err := t.storage.ClearTaskPromptNotes(agentTaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"message":     "Prompt notes cleared successfully",
	}, nil
}

// AddTodoPromptNotesTool implements the ToolExecutor interface
type AddTodoPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *AddTodoPromptNotesTool) Name() string {
	return "coordinator_add_todo_prompt_notes"
}

func (t *AddTodoPromptNotesTool) Description() string {
	return "Add human guidance notes to a specific TODO item. Use to provide specific instructions for a single TODO."
}

func (t *AddTodoPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"todoId": map[string]interface{}{
				"type":        "string",
				"description": "TODO item UUID",
			},
			"promptNotes": map[string]interface{}{
				"type":        "string",
				"description": "Human guidance notes, markdown supported",
			},
		},
		"required": []string{"agentTaskId", "todoId", "promptNotes"},
	}
}

func (t *AddTodoPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	todoID, ok := input["todoId"].(string)
	if !ok || todoID == "" {
		return nil, fmt.Errorf("todoId is required and must be a string")
	}

	promptNotes, ok := input["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return nil, fmt.Errorf("promptNotes is required and must be a string")
	}

	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	err = t.storage.AddTodoPromptNotes(agentTaskID, todoID, sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to add TODO prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"todoId":      todoID,
		"message":     "TODO prompt notes added successfully",
	}, nil
}

// UpdateTodoPromptNotesTool implements the ToolExecutor interface
type UpdateTodoPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *UpdateTodoPromptNotesTool) Name() string {
	return "coordinator_update_todo_prompt_notes"
}

func (t *UpdateTodoPromptNotesTool) Description() string {
	return "Update existing human guidance notes on a TODO item."
}

func (t *UpdateTodoPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"todoId": map[string]interface{}{
				"type":        "string",
				"description": "TODO item UUID",
			},
			"promptNotes": map[string]interface{}{
				"type":        "string",
				"description": "Human guidance notes, markdown supported",
			},
		},
		"required": []string{"agentTaskId", "todoId", "promptNotes"},
	}
}

func (t *UpdateTodoPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	todoID, ok := input["todoId"].(string)
	if !ok || todoID == "" {
		return nil, fmt.Errorf("todoId is required and must be a string")
	}

	promptNotes, ok := input["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return nil, fmt.Errorf("promptNotes is required and must be a string")
	}

	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	err = t.storage.UpdateTodoPromptNotes(agentTaskID, todoID, sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to update TODO prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"todoId":      todoID,
		"message":     "TODO prompt notes updated successfully",
	}, nil
}

// ClearTodoPromptNotesTool implements the ToolExecutor interface
type ClearTodoPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *ClearTodoPromptNotesTool) Name() string {
	return "coordinator_clear_todo_prompt_notes"
}

func (t *ClearTodoPromptNotesTool) Description() string {
	return "Clear/remove human guidance notes from a TODO item."
}

func (t *ClearTodoPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"todoId": map[string]interface{}{
				"type":        "string",
				"description": "TODO item UUID",
			},
		},
		"required": []string{"agentTaskId", "todoId"},
	}
}

func (t *ClearTodoPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	todoID, ok := input["todoId"].(string)
	if !ok || todoID == "" {
		return nil, fmt.Errorf("todoId is required and must be a string")
	}

	err := t.storage.ClearTodoPromptNotes(agentTaskID, todoID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear TODO prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"todoId":      todoID,
		"message":     "TODO prompt notes cleared successfully",
	}, nil
}

// ListSubagentsTool implements the ToolExecutor interface
type ListSubagentsTool struct {
	mongoDatabase *interface{} // Will be *mongo.Database but using interface{} to avoid import cycle
}

func (t *ListSubagentsTool) Name() string {
	return "list_subagents"
}

func (t *ListSubagentsTool) Description() string {
	return "Returns available subagents from CLAUDE.md agent list with names, descriptions, tools, and categories"
}

func (t *ListSubagentsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListSubagentsTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// For now, return a hardcoded list of subagents from CLAUDE.md
	// In the future, this should query MongoDB's subagents collection
	// But since we can't import mongo.Database here, we'll use a simple approach

	subagents := []map[string]interface{}{
		{
			"name":        "go-dev",
			"description": "Go microservices, REST APIs, business logic",
		},
		{
			"name":        "go-mcp-dev",
			"description": "MCP tools and integrations (Model Context Protocol)",
		},
		{
			"name":        "ui-dev",
			"description": "React/TypeScript implementation, components",
		},
		{
			"name":        "ui-tester",
			"description": "Playwright E2E tests, accessibility validation",
		},
		{
			"name":        "sre",
			"description": "Deployment to dev/prod environments",
		},
		{
			"name":        "k8s-deployment-expert",
			"description": "Kubernetes manifests, rollouts, scaling",
		},
	}

	return map[string]interface{}{
		"subagents": subagents,
		"count":     len(subagents),
	}, nil
}

// SetCurrentSubagentTool implements the ToolExecutor interface
type SetCurrentSubagentTool struct {
	mongoDatabase *interface{} // Will be *mongo.Database but using interface{} to avoid import cycle
}

func (t *SetCurrentSubagentTool) Name() string {
	return "set_current_subagent"
}

func (t *SetCurrentSubagentTool) Description() string {
	return "Associate a subagent with the current chat session. Stores subagent name in chat metadata."
}

func (t *SetCurrentSubagentTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"subagentName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the subagent to associate with chat (must match list from list_subagents)",
			},
		},
		"required": []string{"subagentName"},
	}
}

func (t *SetCurrentSubagentTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	subagentName, ok := input["subagentName"].(string)
	if !ok || subagentName == "" {
		return nil, fmt.Errorf("subagentName is required and must be a string")
	}

	// Validate subagent name against known list
	validSubagents := map[string]bool{
		"go-dev":                             true,
		"go-mcp-dev":                         true,
		"Backend Services Specialist":        true,
		"Event Systems Specialist":           true,
		"Data Platform Specialist":           true,
		"ui-dev":                             true,
		"ui-tester":                          true,
		"Frontend Experience Specialist":     true,
		"AI Integration Specialist":          true,
		"Real-time Systems Specialist":       true,
		"sre":                                true,
		"k8s-deployment-expert":              true,
		"Infrastructure Automation Specialist": true,
		"Security & Auth Specialist":         true,
		"Observability Specialist":           true,
		"End-to-End Testing Coordinator":     true,
	}

	if !validSubagents[subagentName] {
		return nil, fmt.Errorf("invalid subagent name '%s'. Use list_subagents to see available subagents", subagentName)
	}

	// Return success - actual chat session association will be handled by the chat service
	return map[string]interface{}{
		"subagentName": subagentName,
		"valid":        true,
		"message":      fmt.Sprintf("Subagent '%s' validated successfully. Chat session association requires chat context.", subagentName),
	}, nil
}

// DiscoverToolsExecutor implements the discover_tools tool executor
type DiscoverToolsExecutor struct {
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
}

func (e *DiscoverToolsExecutor) Name() string {
	return "discover_tools"
}

func (e *DiscoverToolsExecutor) Description() string {
	return "Discover MCP tools using natural language semantic search. Returns matching tool names with descriptions and similarity scores. Use this to find tools by description (e.g., 'video tools', 'database tools', 'file operations')."
}

func (e *DiscoverToolsExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Natural language search query describing the tools you're looking for (e.g., 'tools for video processing', 'database operations', 'file management')",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of results to return (default: 5, max: 20)",
			},
		},
		"required": []string{"query"},
	}
}

func (e *DiscoverToolsExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleDiscoverTools(ctx, args)
	return data, err
}

// GetToolSchemaExecutor implements the get_tool_schema tool executor
type GetToolSchemaExecutor struct {
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
}

func (e *GetToolSchemaExecutor) Name() string {
	return "get_tool_schema"
}

func (e *GetToolSchemaExecutor) Description() string {
	return "Get the complete JSON schema for a specific MCP tool. Returns the full tool definition including parameters, types, and descriptions. Use this after discovering tools to understand how to call them."
}

func (e *GetToolSchemaExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"toolName": map[string]interface{}{
				"type":        "string",
				"description": "Exact tool name to get schema for (use discover_tools first to find tool names)",
			},
		},
		"required": []string{"toolName"},
	}
}

func (e *GetToolSchemaExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleGetToolSchema(ctx, args)
	return data, err
}

// ExecuteToolExecutor implements the execute_tool tool executor
type ExecuteToolExecutor struct {
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
}

func (e *ExecuteToolExecutor) Name() string {
	return "execute_tool"
}

func (e *ExecuteToolExecutor) Description() string {
	return "Execute an MCP tool by name with specified arguments. This tool looks up the tool's server from the registry and makes an HTTP call to that server's MCP endpoint. Works with external MCP servers registered via mcp_add_server. Built-in tools cannot be executed via this tool. Use get_tool_schema first to understand required parameters."
}

func (e *ExecuteToolExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"toolName": map[string]interface{}{
				"type":        "string",
				"description": "Exact tool name to execute (from discover_tools)",
			},
			"args": map[string]interface{}{
				"type":        "object",
				"description": "Tool-specific arguments as a JSON object (see get_tool_schema for parameter details)",
			},
		},
		"required": []string{"toolName", "args"},
	}
}

func (e *ExecuteToolExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleExecuteTool(ctx, args)
	return data, err
}

// McpAddServerExecutor implements the mcp_add_server tool executor
type McpAddServerExecutor struct {
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
}

func (e *McpAddServerExecutor) Name() string {
	return "mcp_add_server"
}

func (e *McpAddServerExecutor) Description() string {
	return "Add a new MCP server to the registry, discover its tools, and store them in MongoDB and Qdrant for semantic search. The server must be accessible via HTTP/HTTPS and expose the MCP protocol."
}

func (e *McpAddServerExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"serverName": map[string]interface{}{
				"type":        "string",
				"description": "Unique name for this MCP server (e.g., 'openai-mcp', 'github-mcp')",
			},
			"serverUrl": map[string]interface{}{
				"type":        "string",
				"description": "HTTP/HTTPS URL of the MCP server (e.g., 'http://localhost:3000/mcp')",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Human-readable description of what this server provides",
			},
		},
		"required": []string{"serverName", "serverUrl"},
	}
}

func (e *McpAddServerExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleMCPAddServer(ctx, args)
	return data, err
}

// McpRediscoverServerExecutor implements the mcp_rediscover_server tool executor
type McpRediscoverServerExecutor struct {
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
}

func (e *McpRediscoverServerExecutor) Name() string {
	return "mcp_rediscover_server"
}

func (e *McpRediscoverServerExecutor) Description() string {
	return "Rediscover and refresh tools from an existing MCP server. This removes old tools and discovers the current set of tools available on the server."
}

func (e *McpRediscoverServerExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"serverName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the MCP server to rediscover (must already be registered)",
			},
		},
		"required": []string{"serverName"},
	}
}

func (e *McpRediscoverServerExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleMCPRediscoverServer(ctx, args)
	return data, err
}

// McpRemoveServerExecutor implements the mcp_remove_server tool executor
type McpRemoveServerExecutor struct {
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
}

func (e *McpRemoveServerExecutor) Name() string {
	return "mcp_remove_server"
}

func (e *McpRemoveServerExecutor) Description() string {
	return "Remove an MCP server and all its tools from the registry. This deletes the server metadata and all associated tool data from MongoDB and Qdrant."
}

func (e *McpRemoveServerExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"serverName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the MCP server to remove",
			},
		},
		"required": []string{"serverName"},
	}
}

func (e *McpRemoveServerExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleMCPRemoveServer(ctx, args)
	return data, err
}

// ExecuteSubagentTool implements the ToolExecutor interface
// This tool creates a subchat, links it to an agent task, and executes the subagent in background
type ExecuteSubagentTool struct {
	subchatStorage    *storage.SubchatStorage
	taskStorage       storage.TaskStorage
	aiService         AIServiceInterface
	chatService       ChatServiceInterface
	aiSettingsService AISettingsServiceInterface
	logger            *zap.Logger
}

// AIServiceInterface defines methods needed from the AI chat service
type AIServiceInterface interface {
	StreamChatWithTools(ctx context.Context, messages []aiservice.Message, maxToolCalls int) (<-chan aiservice.StreamEvent, error)
	GetConfig() *aiservice.AIConfig
}

// ChatServiceInterface defines methods needed from the chat service
type ChatServiceInterface interface {
	CreateSession(ctx context.Context, userID, companyID, title string) (*models.ChatSession, error)
	SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*models.ChatMessage, error)
	SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, id, name string, args map[string]interface{}, companyID string) (*models.ChatMessage, error)
	SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, id, name string, output interface{}, errorMsg string, durationMs int64, companyID string) (*models.ChatMessage, error)
}

// AISettingsServiceInterface defines methods needed from AI settings service
type AISettingsServiceInterface interface {
	GetSubagent(ctx context.Context, id primitive.ObjectID, companyID string) (*models.Subagent, error)
}

func (t *ExecuteSubagentTool) Name() string {
	return "execute_subagent"
}

func (t *ExecuteSubagentTool) Description() string {
	return "Execute a subagent to handle an agent task. Creates a subchat, links it to the task, and spawns the subagent in a separate execution context. The subagent will work independently and update task status. Returns immediately with subchat ID for tracking."
}

func (t *ExecuteSubagentTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task ID (UUID) to execute",
			},
			"parentChatId": map[string]interface{}{
				"type":        "string",
				"description": "Parent chat session ID",
			},
		},
		"required": []string{"agentTaskId", "parentChatId"},
	}
}

func (t *ExecuteSubagentTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	parentChatID, ok := input["parentChatId"].(string)
	if !ok || parentChatID == "" {
		return nil, fmt.Errorf("parentChatId is required and must be a string")
	}

	t.logger.Info("🚀 execute_subagent tool called",
		zap.String("agentTaskId", agentTaskID),
		zap.String("parentChatId", parentChatID))

	// Get the agent task to extract subagent name and details
	agentTask, err := t.taskStorage.GetAgentTask(agentTaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent task: %w", err)
	}

	t.logger.Info("📋 Retrieved agent task",
		zap.String("agentTaskId", agentTaskID),
		zap.String("agentName", agentTask.AgentName),
		zap.Int("todoCount", len(agentTask.Todos)))

	// Update task status to in_progress
	err = t.taskStorage.UpdateTaskStatus(agentTaskID, storage.TaskStatusInProgress, "Subagent execution initiated")
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Create subchat for this execution
	subchat, err := t.subchatStorage.CreateSubchat(
		parentChatID,
		agentTask.AgentName,
		&agentTaskID,
		nil, // todoID
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create subchat: %w", err)
	}

	t.logger.Info("💬 Created subchat for background execution",
		zap.String("subchatId", subchat.ID),
		zap.String("agentName", agentTask.AgentName))

	// Spawn background goroutine to execute the subagent
	go t.executeSubagentInBackground(subchat.ID, agentTask, parentChatID)

	return map[string]interface{}{
		"subchatId":    subchat.ID,
		"agentName":    subchat.SubagentName,
		"agentTaskId":  agentTaskID,
		"status":       "executing",
		"message":      fmt.Sprintf("Subchat created and %s is now executing in background. Check subchat messages for progress.", agentTask.AgentName),
		"createdAt":    subchat.CreatedAt,
	}, nil
}

// FileOperationTracker tracks file operations and duplicate tool calls during subagent execution
type FileOperationTracker struct {
	DirectoriesListed map[string]int    // path -> count
	FilesRead         map[string]int    // path -> count
	FilesWritten      map[string]int    // path -> count
	ToolCallHistory   []string          // chronological list of tool calls

	// Track full argument sets for loop detection
	ToolCallSignatures map[string]int   // signature (toolName + argsJSON) -> count
}

// NewFileOperationTracker creates a new tracker
func NewFileOperationTracker() *FileOperationTracker {
	return &FileOperationTracker{
		DirectoriesListed:  make(map[string]int),
		FilesRead:          make(map[string]int),
		FilesWritten:       make(map[string]int),
		ToolCallHistory:    make([]string, 0),
		ToolCallSignatures: make(map[string]int),
	}
}

// RecordOperation records a file operation and detects duplicate tool calls with identical arguments
func (f *FileOperationTracker) RecordOperation(toolName string, args map[string]interface{}) string {
	warning := ""

	// ENHANCED LOOP DETECTION: Track full argument sets for all tool calls
	// Serialize arguments to JSON for signature comparison
	argsJSON, err := json.Marshal(args)
	if err == nil {
		// Create signature: toolName + argsJSON
		signature := toolName + ":" + string(argsJSON)

		// Track this signature
		f.ToolCallSignatures[signature]++

		// Generate warning if this exact call (tool + args) was seen before
		if f.ToolCallSignatures[signature] > 1 {
			count := f.ToolCallSignatures[signature] - 1
			warning = fmt.Sprintf("🔁 LOOP DETECTED: You already called '%s' with these exact arguments %d time(s). You are repeating the same operation. Use the results from previous calls instead of repeating.", toolName, count)
		}
	}

	// SPECIFIC FILE OPERATION TRACKING: Keep detailed file path tracking for progress summaries
	switch toolName {
	case "list_directory":
		if path, ok := args["path"].(string); ok {
			f.DirectoriesListed[path]++
			// Only show file-specific warning if no general loop warning was generated
			if warning == "" && f.DirectoriesListed[path] > 1 {
				warning = fmt.Sprintf("⚠️  WARNING: You already listed directory '%s' %d time(s). Review previous results before repeating.", path, f.DirectoriesListed[path]-1)
			}
		}
	case "read_file":
		if path, ok := args["filePath"].(string); ok {
			f.FilesRead[path]++
			// Only show file-specific warning if no general loop warning was generated
			if warning == "" && f.FilesRead[path] > 1 {
				warning = fmt.Sprintf("⚠️  WARNING: You already read file '%s' %d time(s). You should have the content from previous calls.", path, f.FilesRead[path]-1)
			}
		}
	case "write_file", "apply_patch":
		if path, ok := args["filePath"].(string); ok {
			f.FilesWritten[path]++
		}
	}

	f.ToolCallHistory = append(f.ToolCallHistory, toolName)
	return warning
}

// GetProgressSummary returns a formatted summary of file operations
func (f *FileOperationTracker) GetProgressSummary() string {
	var summary strings.Builder

	summary.WriteString("\n\n📊 PROGRESS TRACKING - Files You've Already Seen:\n")

	if len(f.DirectoriesListed) > 0 {
		summary.WriteString("\nDirectories Listed:\n")
		for path, count := range f.DirectoriesListed {
			summary.WriteString(fmt.Sprintf("  • %s (%d times)\n", path, count))
		}
	}

	if len(f.FilesRead) > 0 {
		summary.WriteString("\nFiles Read:\n")
		for path, count := range f.FilesRead {
			summary.WriteString(fmt.Sprintf("  • %s (%d times)\n", path, count))
		}
	}

	if len(f.FilesWritten) > 0 {
		summary.WriteString("\nFiles Written/Modified:\n")
		for path, count := range f.FilesWritten {
			summary.WriteString(fmt.Sprintf("  • %s (%d times)\n", path, count))
		}
	}

	if len(f.DirectoriesListed) == 0 && len(f.FilesRead) == 0 && len(f.FilesWritten) == 0 {
		summary.WriteString("  (No file operations yet)\n")
	}

	summary.WriteString("\n⚠️  IMPORTANT: Do not repeat operations on files you've already seen. Use the information from previous tool calls.\n")

	return summary.String()
}

// validateFileModifications checks if expected files were actually modified using git diff
func (t *ExecuteSubagentTool) validateFileModifications(agentTask *storage.AgentTask) (bool, []string, error) {
	// If no files expected to be modified, skip validation
	if len(agentTask.FilesModified) == 0 {
		return true, []string{}, nil
	}

	// Run: git diff --name-only HEAD
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil, fmt.Errorf("git diff failed: %w (output: %s)", err, string(output))
	}

	modifiedFilesStr := strings.TrimSpace(string(output))
	if modifiedFilesStr == "" {
		return false, []string{}, fmt.Errorf("expected files not modified: wanted %v, but git diff shows no changes", agentTask.FilesModified)
	}

	modifiedFiles := strings.Split(modifiedFilesStr, "\n")

	// Check if any expected files were modified
	expectedFiles := make(map[string]bool)
	for _, file := range agentTask.FilesModified {
		expectedFiles[file] = true
	}

	matchedFiles := []string{}
	for _, modFile := range modifiedFiles {
		if modFile != "" && expectedFiles[modFile] {
			matchedFiles = append(matchedFiles, modFile)
		}
	}

	// Require at least 1 expected file to be modified
	if len(matchedFiles) == 0 {
		return false, matchedFiles, fmt.Errorf("expected files not modified: wanted %v, got %v instead", agentTask.FilesModified, modifiedFiles)
	}

	return true, matchedFiles, nil
}

// executeSubagentInBackground runs the subagent AI streaming in a background goroutine
func (t *ExecuteSubagentTool) executeSubagentInBackground(subchatID string, agentTask *storage.AgentTask, parentChatID string) {
	// Create a new background context with generous timeout for long-running tasks
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	t.logger.Info("⚡ Starting subagent execution in background goroutine",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName),
		zap.String("agentTaskId", agentTask.ID))

	// Initialize progress tracker
	progressTracker := NewFileOperationTracker()

	// Create a chat session for this subchat
	companyID := "dev-company" // TODO: Extract from context
	userID := "coordinator"    // Subagent executions are associated with the coordinator
	sessionTitle := fmt.Sprintf("Subchat: %s - %s", agentTask.AgentName, agentTask.Role)

	chatSession, err := t.chatService.CreateSession(ctx, userID, companyID, sessionTitle)
	if err != nil {
		t.logger.Error("Failed to create chat session for subchat",
			zap.String("subchatId", subchatID),
			zap.Error(err))
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Failed to create chat session: %v", err))
		return
	}

	t.logger.Info("💬 Created chat session for subchat",
		zap.String("subchatId", subchatID),
		zap.String("sessionId", chatSession.ID.Hex()))

	// Update subchat with session ID for linking
	sessionIDHex := chatSession.ID.Hex()
	err = t.subchatStorage.UpdateSubchatSessionID(subchatID, sessionIDHex)
	if err != nil {
		t.logger.Warn("Failed to link chat session to subchat",
			zap.String("subchatId", subchatID),
			zap.String("sessionId", sessionIDHex),
			zap.Error(err))
		// Continue execution even if linking fails
	}

	// Build subagent prompt from agent task
	subagentPrompt := t.buildSubagentPrompt(agentTask)

	t.logger.Info("📜 Built subagent prompt",
		zap.String("subchatId", subchatID),
		zap.Int("promptLength", len(subagentPrompt)),
		zap.Int("todoCount", len(agentTask.Todos)))

	// Save initial user message
	_, err = t.chatService.SaveMessage(ctx, chatSession.ID, "user", subagentPrompt, companyID)
	if err != nil {
		t.logger.Warn("Failed to save initial user message",
			zap.String("subchatId", subchatID),
			zap.Error(err))
		// Continue execution even if message save fails
	}

	// Create initial message with the task prompt
	messages := []aiservice.Message{
		{
			Role:    "user",
			Content: subagentPrompt,
		},
	}

	// Stream AI response with tools
	maxToolCalls := t.aiService.GetConfig().MaxToolCalls
	aiStream, err := t.aiService.StreamChatWithTools(ctx, messages, maxToolCalls)
	if err != nil {
		t.logger.Error("Failed to start AI streaming for subagent",
			zap.String("subchatId", subchatID),
			zap.Error(err))
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("AI streaming failed: %v", err))
		return
	}

	// Process stream events and save to subchat
	fullResponse := ""
	toolCallCount := 0
	completedTodos := 0

	t.logger.Info("📡 Subagent AI stream started - processing events...",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName))

	for event := range aiStream {
		select {
		case <-ctx.Done():
			t.logger.Warn("⏱️ Subagent execution cancelled by timeout",
				zap.String("subchatId", subchatID))
			t.handleExecutionFailure(agentTask.ID, "Execution timeout")
			return
		default:
			switch event.Type {
			case aiservice.StreamEventToken:
				fullResponse += event.Content

			case aiservice.StreamEventToolCall:
				toolCallCount++

				// Record file operation and get warning if duplicate
				warning := progressTracker.RecordOperation(event.ToolCall.Name, event.ToolCall.Args)
				if warning != "" {
					t.logger.Warn("Duplicate file operation detected",
						zap.String("subchatId", subchatID),
						zap.String("toolName", event.ToolCall.Name),
						zap.String("warning", warning))

					// Save warning as a visible assistant message in Chat UI
					_, err := t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", warning, companyID)
					if err != nil {
						t.logger.Warn("Failed to save duplicate operation warning to chat",
							zap.String("subchatId", subchatID),
							zap.Error(err))
					}
				}

				t.logger.Info("🔧 Subagent calling tool",
					zap.String("subchatId", subchatID),
					zap.String("agentName", agentTask.AgentName),
					zap.String("toolName", event.ToolCall.Name),
					zap.Int("toolCallNumber", toolCallCount))

				// Save tool call to subchat messages
				_, err := t.chatService.SaveToolCall(ctx, chatSession.ID, event.ToolCall.ID, event.ToolCall.Name, event.ToolCall.Args, companyID)
				if err != nil {
					t.logger.Error("Failed to save tool call",
						zap.String("subchatId", subchatID),
						zap.Error(err))
				}

				// Check if this is a todo status update - track completion
				if event.ToolCall.Name == "coordinator_update_todo_status" {
					if status, ok := event.ToolCall.Args["status"].(string); ok && status == "completed" {
						completedTodos++
						t.logger.Info("✅ TODO marked as completed",
							zap.String("subchatId", subchatID),
							zap.String("agentName", agentTask.AgentName),
							zap.Int("completedCount", completedTodos),
							zap.Int("totalTodos", len(agentTask.Todos)))
					} else if status, ok := event.ToolCall.Args["status"].(string); ok && status == "in_progress" {
						t.logger.Info("▶️ TODO started",
							zap.String("subchatId", subchatID),
							zap.String("agentName", agentTask.AgentName))
					}
				}

			case aiservice.StreamEventToolResult:
				// Summarize tool result to prevent context bloat
				var originalSize int
				var summarizedOutput string

				if event.ToolResult.Output != nil {
					// Calculate original size for logging
					if str, ok := event.ToolResult.Output.(string); ok {
						originalSize = len(str)
					} else {
						outputBytes, _ := json.Marshal(event.ToolResult.Output)
						originalSize = len(outputBytes)
					}

					// Apply summarization
					summarizedOutput = t.summarizeToolResult(event.ToolResult.Name, event.ToolResult.Output)
				}

				// Use error message if tool call failed
				if event.ToolResult.Error != "" {
					summarizedOutput = fmt.Sprintf("Error: %s", event.ToolResult.Error)
				}

				// Inject progress summary every 3 tool calls for file operations
				if toolCallCount%3 == 0 && (len(progressTracker.FilesRead) > 0 || len(progressTracker.DirectoriesListed) > 0) {
					progressSummary := progressTracker.GetProgressSummary()
					summarizedOutput += progressSummary

					// Save progress summary as a visible assistant message in Chat UI
					_, err := t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", progressSummary, companyID)
					if err != nil {
						t.logger.Warn("Failed to save progress summary message to chat",
							zap.String("subchatId", subchatID),
							zap.Error(err))
					}

					t.logger.Info("📊 Injected progress tracking summary",
						zap.String("subchatId", subchatID),
						zap.Int("filesRead", len(progressTracker.FilesRead)),
						zap.Int("dirsListed", len(progressTracker.DirectoriesListed)),
						zap.Int("filesWritten", len(progressTracker.FilesWritten)))
				}

				// Log tool result with success/failure indicator and context size info
				if event.ToolResult.Error != "" {
					t.logger.Warn("❌ Tool call failed",
						zap.String("subchatId", subchatID),
						zap.String("toolName", event.ToolResult.Name),
						zap.String("error", event.ToolResult.Error))
				} else {
					t.logger.Info("✓ Tool call completed",
						zap.String("subchatId", subchatID),
						zap.String("toolName", event.ToolResult.Name),
						zap.Int64("durationMs", event.ToolResult.DurationMs),
						zap.Int("originalSize", originalSize),
						zap.Int("summarizedSize", len(summarizedOutput)))
				}

				_, err := t.chatService.SaveToolResult(ctx, chatSession.ID, event.ToolResult.ID, event.ToolResult.Name, summarizedOutput, event.ToolResult.Error, event.ToolResult.DurationMs, companyID)
				if err != nil {
					t.logger.Error("Failed to save tool result",
						zap.String("subchatId", subchatID),
						zap.Error(err))
				}

			case aiservice.StreamEventError:
				t.logger.Error("❌ AI service error during subagent execution",
					zap.String("subchatId", subchatID),
					zap.String("error", event.Error))
				t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("AI error: %s", event.Error))
				return
			}
		}
	}

	t.logger.Info("📝 Saving final AI response to subchat",
		zap.String("subchatId", subchatID),
		zap.Int("responseLength", len(fullResponse)))

	// Save final AI response to subchat
	_, err = t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", fullResponse, companyID)
	if err != nil {
		t.logger.Error("Failed to save subagent final response",
			zap.String("subchatId", subchatID),
			zap.Error(err))
	}

	// ========================================
	// VALIDATION LAYER 1: File Modification Validation
	// ========================================
	var modifiedFiles []string
	if len(agentTask.FilesModified) > 0 {
		filesOK, files, validationErr := t.validateFileModifications(agentTask)
		modifiedFiles = files

		if !filesOK {
			t.logger.Warn("❌ File modification validation FAILED",
				zap.String("subchatId", subchatID),
				zap.String("agentTaskId", agentTask.ID),
				zap.Error(validationErr))

			// Mark as BLOCKED instead of completed
			blockReason := fmt.Sprintf("Validation failed: %v. Tool calls: %d, Claimed TODOs: %d/%d",
				validationErr, toolCallCount, completedTodos, len(agentTask.Todos))

			err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusBlocked, blockReason)
			if err != nil {
				t.logger.Error("Failed to update task status to blocked",
					zap.String("agentTaskId", agentTask.ID),
					zap.Error(err))
			}

			err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusFailed)
			if err != nil {
				t.logger.Error("Failed to update subchat status to failed",
					zap.String("subchatId", subchatID),
					zap.Error(err))
			}

			t.logger.Error("🚨 PHANTOM COMPLETION PREVENTED",
				zap.String("subchatId", subchatID),
				zap.String("agentTaskId", agentTask.ID),
				zap.String("reason", "no files modified"))
			return
		}

		// Log proof of work
		t.logger.Info("✅ File modification validation PASSED",
			zap.String("subchatId", subchatID),
			zap.String("agentTaskId", agentTask.ID),
			zap.Strings("modifiedFiles", modifiedFiles))
	}

	// ========================================
	// VALIDATION LAYER 2: TODO Completion Verification
	// ========================================
	if completedTodos < len(agentTask.Todos) {
		t.logger.Warn("❌ TODO completion validation FAILED",
			zap.String("subchatId", subchatID),
			zap.String("agentTaskId", agentTask.ID),
			zap.Int("completed", completedTodos),
			zap.Int("total", len(agentTask.Todos)))

		summaryNotes := fmt.Sprintf("Incomplete: %d/%d TODOs done, %d tool calls. Files modified: %v",
			completedTodos, len(agentTask.Todos), toolCallCount, modifiedFiles)

		err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusInProgress, summaryNotes)
		if err != nil {
			t.logger.Error("Failed to update task status to in_progress",
				zap.String("agentTaskId", agentTask.ID),
				zap.Error(err))
		}

		err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusActive)
		if err != nil {
			t.logger.Error("Failed to update subchat status to active",
				zap.String("subchatId", subchatID),
				zap.Error(err))
		}

		return
	}

	// ========================================
	// ALL VALIDATIONS PASSED - Safe to mark as completed
	// ========================================
	summaryNotes := fmt.Sprintf("✅ VALIDATED completion: %d/%d TODOs, %d tool calls, %d files modified: %v",
		completedTodos, len(agentTask.Todos), toolCallCount, len(modifiedFiles), modifiedFiles)

	t.logger.Info("🎉 Task completion validated successfully",
		zap.String("subchatId", subchatID),
		zap.String("agentTaskId", agentTask.ID),
		zap.Int("toolCalls", toolCallCount),
		zap.Int("completedTodos", completedTodos),
		zap.Strings("filesModified", modifiedFiles))

	err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusCompleted, summaryNotes)
	if err != nil {
		t.logger.Error("Failed to update task status to completed",
			zap.String("agentTaskId", agentTask.ID),
			zap.Error(err))
	}

	// Update subchat status to completed
	err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusCompleted)
	if err != nil {
		t.logger.Error("Failed to update subchat status",
			zap.String("subchatId", subchatID),
			zap.Error(err))
	}

	t.logger.Info("🎉 Subagent execution completed successfully!",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName),
		zap.Int("toolCalls", toolCallCount),
		zap.Int("completedTodos", completedTodos),
		zap.Int("totalTodos", len(agentTask.Todos)))
}

// buildSubagentPrompt constructs a detailed prompt for the subagent based on the agent task
func (t *ExecuteSubagentTool) buildSubagentPrompt(agentTask *storage.AgentTask) string {
	prompt := fmt.Sprintf(`You are %s. You have been assigned a task to complete.

ROLE: %s

TASK CONTEXT:
%s

YOUR TODOs:
`, agentTask.AgentName, agentTask.Role, agentTask.ContextSummary)

	for i, todo := range agentTask.Todos {
		status := "PENDING"
		if todo.Status == "completed" {
			status = "✓ DONE"
		} else if todo.Status == "in_progress" {
			status = "IN PROGRESS"
		}

		prompt += fmt.Sprintf("\n%d. [%s] %s", i+1, status, todo.Description)

		if todo.FilePath != "" {
			prompt += fmt.Sprintf("\n   File: %s", todo.FilePath)
		}
		if todo.FunctionName != "" {
			prompt += fmt.Sprintf("\n   Function: %s", todo.FunctionName)
		}
		if todo.ContextHint != "" {
			prompt += fmt.Sprintf("\n   Hint: %s", todo.ContextHint)
		}
		if todo.HumanPromptNotes != "" {
			prompt += fmt.Sprintf("\n   Notes: %s", todo.HumanPromptNotes)
		}
	}

	if len(agentTask.FilesModified) > 0 {
		prompt += "\n\nFILES TO MODIFY:\n"
		for _, file := range agentTask.FilesModified {
			prompt += fmt.Sprintf("- %s\n", file)
		}
	}

	if len(agentTask.QdrantCollections) > 0 {
		prompt += "\n\nRELEVANT KNOWLEDGE COLLECTIONS:\n"
		for _, coll := range agentTask.QdrantCollections {
			prompt += fmt.Sprintf("- %s\n", coll)
		}
	}

	if agentTask.HumanPromptNotes != "" {
		prompt += fmt.Sprintf("\n\nADDITIONAL GUIDANCE:\n%s\n", agentTask.HumanPromptNotes)
	}

	prompt += fmt.Sprintf(`

🎯 EXECUTION DEADLINE: You must START IMPLEMENTING within 2 MINUTES of reading this prompt.
No planning phase. No extended exploration. Read context → Start coding → Complete TODOs.

INSTRUCTIONS:
1. Start working on the TODOs immediately - no planning phase needed
2. Use coordinator_update_todo_status to mark each TODO as 'in_progress' when you start it
3. Use coordinator_update_todo_status to mark each TODO as 'completed' when done
4. Use tools like read_file, write_file, bash to complete your work
5. Update coordinator_upsert_knowledge with key decisions and contracts for handoff
6. When ALL TODOs are completed, summarize what you accomplished

🚨 CRITICAL ANTI-LOOP RULES (CIRCUIT BREAKER WILL STOP YOU):
• NEVER call the same tool with identical arguments more than ONCE
• If you call bash with "grep -R delete", use those results - DO NOT call it again
• If you call list_directory on "./ui/src", use those results - DO NOT call it again
• If you get results from a tool, USE THEM. Don't re-run hoping for different output
• After 4 identical tool calls, you will be STOPPED and marked as FAILED
• Tool results are stored in your context - refer back to them instead of re-calling

🎯 EFFICIENT FILE DISCOVERY (ANTI-LOOP):
• ✅ IF context/TODO specifies exact file path: read_file directly (NO exploration needed!)
• ✅ IF file location unclear: Use code_index_search("feature name") ONCE, then read top result
• ❌ NEVER start with list_directory - it leads to exploration loops
• ❌ NEVER read same file twice - use the result from your first read

🎯 WORKFLOW GUIDANCE (MANDATORY):
• code_index_search: Read TOP result immediately - NO MORE SEARCHING
• After list_directory: IMMEDIATELY use read_file on the target file - NO MORE EXPLORATION
• After read_file: IMMEDIATELY make changes with write_file or apply_patch - NO MORE READING
• After grep/search: USE THE RESULTS to guide your implementation - DO NOT SEARCH AGAIN
• After write_file: Move to next TODO or verify - DO NOT re-read what you just wrote
• Progress through: discover location → read → modify → test → next TODO
• Each tool call MUST move you closer to completing a TODO
• If you find yourself calling the same tool repeatedly, STOP and use previous results

⚡ FAST ITERATION REQUIREMENT:
• Maximum 3 tool calls before first file modification
• If you read more than 3 files before editing, you're over-exploring
• Start implementing within 2 minutes - context is already provided above
• Trust the context summary and TODO hints - they contain what you need

Task ID: %s
Start now! Remember: USE TOOL RESULTS, DON'T REPEAT CALLS`, agentTask.ID)

	return prompt
}

// summarizeToolResult creates a concise summary of a tool result to prevent context bloat
func (t *ExecuteSubagentTool) summarizeToolResult(toolName string, output interface{}) string {
	const maxChars = 500 // Maximum characters to keep for most outputs

	var outputStr string
	if str, ok := output.(string); ok {
		outputStr = str
	} else {
		outputBytes, _ := json.Marshal(output)
		outputStr = string(outputBytes)
	}

	// Special handling for different tool types
	switch toolName {
	case "list_directory":
		// Extract just file count and first few files
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(outputStr), &result); err == nil {
			if files, ok := result["files"].([]interface{}); ok {
				count := len(files)
				preview := files
				if count > 10 {
					preview = files[:10]
				}
				previewJSON, _ := json.Marshal(preview)
				return fmt.Sprintf("Directory listing: %d files found. First 10: %s", count, string(previewJSON))
			}
		}

	case "read_file":
		// Summarize file reading - show first/last lines
		lines := strings.Split(outputStr, "\n")
		if len(lines) > 20 {
			firstLines := strings.Join(lines[:10], "\n")
			lastLines := strings.Join(lines[len(lines)-5:], "\n")
			return fmt.Sprintf("File read (%d lines). First 10 lines:\n%s\n... (truncated %d lines) ...\nLast 5 lines:\n%s",
				len(lines), firstLines, len(lines)-15, lastLines)
		}

	case "bash":
		// Summarize bash output - show success/failure and brief output
		if len(outputStr) > maxChars {
			return fmt.Sprintf("Bash command completed. Output (truncated to %d chars): %s...", maxChars, outputStr[:maxChars])
		}

	case "coordinator_update_todo_status":
		// Just confirm the update without full details
		return "TODO status updated successfully"

	case "coordinator_update_task_status":
		// Just confirm the update
		return "Task status updated successfully"

	case "write_file":
		// Confirm write without showing full content
		return "File written successfully"

	case "apply_patch":
		// Confirm patch without showing full diff
		return "Patch applied successfully"
	}

	// Default: truncate long outputs
	if len(outputStr) > maxChars {
		return outputStr[:maxChars] + fmt.Sprintf("... (truncated, original length: %d chars)", len(outputStr))
	}

	return outputStr
}

// handleExecutionFailure marks the task as blocked with error details
func (t *ExecuteSubagentTool) handleExecutionFailure(agentTaskID, errorMsg string) {
	err := t.taskStorage.UpdateTaskStatus(agentTaskID, storage.TaskStatusBlocked, fmt.Sprintf("Execution failed: %s", errorMsg))
	if err != nil {
		t.logger.Error("Failed to update task status to blocked",
			zap.String("agentTaskId", agentTaskID),
			zap.Error(err))
	}
}

// RegisterCoordinatorTools registers all coordinator tools with the tool registry
func RegisterCoordinatorTools(
	registry *aiservice.ToolRegistry,
	taskStorage storage.TaskStorage,
	knowledgeStorage storage.KnowledgeStorage,
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler,
	subchatStorage *storage.SubchatStorage,
	aiService AIServiceInterface,
	chatService ChatServiceInterface,
	aiSettingsService AISettingsServiceInterface,
	logger *zap.Logger,
) error {
	tools := []aiservice.ToolExecutor{
		// Existing tools
		&CreateAgentTaskTool{storage: taskStorage},
		&ListAgentTasksTool{storage: taskStorage},
		&QueryKnowledgeTool{storage: knowledgeStorage},

		// New tools
		&UpsertKnowledgeTool{storage: knowledgeStorage},
		&GetPopularCollectionsTool{storage: knowledgeStorage},
		&CreateHumanTaskTool{storage: taskStorage},
		&UpdateTaskStatusTool{storage: taskStorage},
		&UpdateTodoStatusTool{storage: taskStorage},
		&ListHumanTasksTool{storage: taskStorage},
		&GetAgentTaskTool{storage: taskStorage},
		&AddTaskPromptNotesTool{storage: taskStorage},
		&UpdateTaskPromptNotesTool{storage: taskStorage},
		&ClearTaskPromptNotesTool{storage: taskStorage},
		&AddTodoPromptNotesTool{storage: taskStorage},
		&UpdateTodoPromptNotesTool{storage: taskStorage},
		&ClearTodoPromptNotesTool{storage: taskStorage},

		// Subagent tools
		&ListSubagentsTool{mongoDatabase: nil},
		&SetCurrentSubagentTool{mongoDatabase: nil},
		&ExecuteSubagentTool{
			subchatStorage:    subchatStorage,
			taskStorage:       taskStorage,
			aiService:         aiService,
			chatService:       chatService,
			aiSettingsService: aiSettingsService,
			logger:            logger,
		},

		// MCP tools discovery and management (6 new tools)
		&DiscoverToolsExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&GetToolSchemaExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&ExecuteToolExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&McpAddServerExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&McpRediscoverServerExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&McpRemoveServerExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		// Note: coordinator_clear_task_board excluded (destructive operation)
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
