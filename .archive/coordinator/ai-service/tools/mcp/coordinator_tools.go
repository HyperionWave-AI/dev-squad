package mcp

import (
	"context"
	"fmt"

	"hyperion-coordinator/ai-service"
	"hyperion-coordinator-mcp/storage"
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
	return "Create a new agent task linked to a human task. Returns task ID. Use this to assign work to specialist agents with context-rich task descriptions. Required: humanTaskId, agentName, role, todos. Optional: contextSummary, filesModified, qdrantCollections, priorWorkSummary."
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
				"description": "200-word summary of what agent needs to know (business context, constraints, pattern references)",
			},
			"filesModified": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "List of file paths this task will create or modify",
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
		},
		"required": []string{"prompt"},
	}
}

func (t *CreateHumanTaskTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	prompt, ok := input["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("prompt is required and must be a string")
	}

	task, err := t.storage.CreateHumanTask(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create human task: %w", err)
	}

	return map[string]interface{}{
		"taskId":    task.ID,
		"status":    task.Status,
		"prompt":    task.Prompt,
		"createdAt": task.CreatedAt,
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

// RegisterCoordinatorTools registers all coordinator tools with the tool registry
func RegisterCoordinatorTools(registry *aiservice.ToolRegistry, taskStorage storage.TaskStorage, knowledgeStorage storage.KnowledgeStorage) error {
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
		// Note: coordinator_clear_task_board excluded (destructive operation)
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
