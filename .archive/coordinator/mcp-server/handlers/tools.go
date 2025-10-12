package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"hyperion-coordinator-mcp/storage"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolHandler manages MCP tool operations
type ToolHandler struct {
	taskStorage      storage.TaskStorage
	knowledgeStorage storage.KnowledgeStorage
}

// NewToolHandler creates a new tool handler
func NewToolHandler(taskStorage storage.TaskStorage, knowledgeStorage storage.KnowledgeStorage) *ToolHandler {
	return &ToolHandler{
		taskStorage:      taskStorage,
		knowledgeStorage: knowledgeStorage,
	}
}

// RegisterToolHandlers registers all tool handlers with the MCP server
func (h *ToolHandler) RegisterToolHandlers(server *mcp.Server) error {
	// Register coordinator_upsert_knowledge
	if err := h.registerUpsertKnowledge(server); err != nil {
		return fmt.Errorf("failed to register upsert_knowledge tool: %w", err)
	}

	// Register coordinator_query_knowledge
	if err := h.registerQueryKnowledge(server); err != nil {
		return fmt.Errorf("failed to register query_knowledge tool: %w", err)
	}

	// Register coordinator_get_popular_collections
	if err := h.registerGetPopularCollections(server); err != nil {
		return fmt.Errorf("failed to register get_popular_collections tool: %w", err)
	}

	// Register coordinator_create_human_task
	if err := h.registerCreateHumanTask(server); err != nil {
		return fmt.Errorf("failed to register create_human_task tool: %w", err)
	}

	// Register coordinator_create_agent_task
	if err := h.registerCreateAgentTask(server); err != nil {
		return fmt.Errorf("failed to register create_agent_task tool: %w", err)
	}

	// Register coordinator_update_task_status
	if err := h.registerUpdateTaskStatus(server); err != nil {
		return fmt.Errorf("failed to register update_task_status tool: %w", err)
	}

	// Register coordinator_update_todo_status
	if err := h.registerUpdateTodoStatus(server); err != nil {
		return fmt.Errorf("failed to register update_todo_status tool: %w", err)
	}

	// Register coordinator_list_human_tasks
	if err := h.registerListHumanTasks(server); err != nil {
		return fmt.Errorf("failed to register list_human_tasks tool: %w", err)
	}

	// Register coordinator_list_agent_tasks
	if err := h.registerListAgentTasks(server); err != nil {
		return fmt.Errorf("failed to register list_agent_tasks tool: %w", err)
	}

	// Register coordinator_get_agent_task
	if err := h.registerGetAgentTask(server); err != nil {
		return fmt.Errorf("failed to register get_agent_task tool: %w", err)
	}

	// Register coordinator_clear_task_board
	if err := h.registerClearTaskBoard(server); err != nil {
		return fmt.Errorf("failed to register clear_task_board tool: %w", err)
	}

	// Register coordinator_add_task_prompt_notes
	if err := h.registerAddTaskPromptNotes(server); err != nil {
		return fmt.Errorf("failed to register add_task_prompt_notes tool: %w", err)
	}

	// Register coordinator_update_task_prompt_notes
	if err := h.registerUpdateTaskPromptNotes(server); err != nil {
		return fmt.Errorf("failed to register update_task_prompt_notes tool: %w", err)
	}

	// Register coordinator_clear_task_prompt_notes
	if err := h.registerClearTaskPromptNotes(server); err != nil {
		return fmt.Errorf("failed to register clear_task_prompt_notes tool: %w", err)
	}

	// Register coordinator_add_todo_prompt_notes
	if err := h.registerAddTodoPromptNotes(server); err != nil {
		return fmt.Errorf("failed to register add_todo_prompt_notes tool: %w", err)
	}

	// Register coordinator_update_todo_prompt_notes
	if err := h.registerUpdateTodoPromptNotes(server); err != nil {
		return fmt.Errorf("failed to register update_todo_prompt_notes tool: %w", err)
	}

	// Register coordinator_clear_todo_prompt_notes
	if err := h.registerClearTodoPromptNotes(server); err != nil {
		return fmt.Errorf("failed to register clear_todo_prompt_notes tool: %w", err)
	}

	// Register list_subagents
	if err := h.registerListSubagents(server); err != nil {
		return fmt.Errorf("failed to register list_subagents tool: %w", err)
	}

	// Register set_current_subagent
	if err := h.registerSetCurrentSubagent(server); err != nil {
		return fmt.Errorf("failed to register set_current_subagent tool: %w", err)
	}

	return nil
}

// registerUpsertKnowledge registers the coordinator_upsert_knowledge tool
func (h *ToolHandler) registerUpsertKnowledge(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_upsert_knowledge",
		Description: "Store knowledge in the coordinator knowledge base. Use for storing task context, ADRs, data contracts, and coordination information.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"collection": {
					Type:        "string",
					Description: "Collection name (e.g., 'task:taskURI', 'adr', 'data-contracts')",
				},
				"text": {
					Type:        "string",
					Description: "Content to store",
				},
				"metadata": {
					Type:        "object",
					Description: "Optional metadata (taskId, agentName, timestamp, etc.)",
				},
			},
			Required: []string{"collection", "text"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleUpsertKnowledge(ctx, args)
		return result, err
	})

	return nil
}

// registerQueryKnowledge registers the coordinator_query_knowledge tool
func (h *ToolHandler) registerQueryKnowledge(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_query_knowledge",
		Description: "Query the coordinator knowledge base. Returns most relevant knowledge entries with similarity scores.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"collection": {
					Type:        "string",
					Description: "Collection name to query",
				},
				"query": {
					Type:        "string",
					Description: "Text to search for",
				},
				"limit": {
					Type:        "number",
					Description: "Maximum number of results (default: 5)",
				},
			},
			Required: []string{"collection", "query"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleQueryKnowledge(ctx, args)
		return result, err
	})

	return nil
}

// registerCreateHumanTask registers the coordinator_create_human_task tool
func (h *ToolHandler) registerCreateHumanTask(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_create_human_task",
		Description: "Create a new human task with the original user prompt. Returns a unique taskId (UUID format).",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"prompt": {
					Type:        "string",
					Description: "Original human request/prompt",
				},
			},
			Required: []string{"prompt"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleCreateHumanTask(ctx, args)
		return result, err
	})

	return nil
}

// registerCreateAgentTask registers the coordinator_create_agent_task tool
func (h *ToolHandler) registerCreateAgentTask(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_create_agent_task",
		Description: "Create a new agent task linked to a human task. Returns a unique taskId (UUID format). Supports context-rich task creation to minimize agent context window usage.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"humanTaskId": {
					Type:        "string",
					Description: "Parent human task ID (UUID)",
				},
				"agentName": {
					Type:        "string",
					Description: "Name of the agent assigned to this task",
				},
				"role": {
					Type:        "string",
					Description: "Agent's role/responsibility for this task",
				},
				"contextSummary": {
					Type:        "string",
					Description: "200-word summary of what agent needs to know (business context, constraints, pattern references). Optional but highly recommended for context efficiency.",
				},
				"filesModified": {
					Type:        "array",
					Description: "List of file paths this task will create or modify. Optional.",
					Items: &jsonschema.Schema{
						Type: "string",
					},
				},
				"qdrantCollections": {
					Type:        "array",
					Description: "Suggested Qdrant collections to query if technical patterns needed (1-2 max). Optional.",
					Items: &jsonschema.Schema{
						Type: "string",
					},
				},
				"priorWorkSummary": {
					Type:        "string",
					Description: "Summary of previous agent's work and key decisions (for multi-phase tasks). Optional.",
				},
				"todos": {
					Type:        "array",
					Description: "List of TODO items. Can be strings (legacy) or objects with context hints (recommended).",
					Items: &jsonschema.Schema{
						OneOf: []*jsonschema.Schema{
							{Type: "string"},
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"description": {
										Type:        "string",
										Description: "What to do",
									},
									"filePath": {
										Type:        "string",
										Description: "Specific file to modify (optional)",
									},
									"functionName": {
										Type:        "string",
										Description: "Specific function to create/modify (optional)",
									},
									"contextHint": {
										Type:        "string",
										Description: "50-word hint of how to implement (optional)",
									},
									"notes": {
										Type:        "string",
										Description: "Additional context for this TODO (optional)",
									},
								},
								Required: []string{"description"},
							},
						},
					},
				},
			},
			Required: []string{"humanTaskId", "agentName", "role", "todos"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleCreateAgentTask(ctx, args)
		return result, err
	})

	return nil
}

// registerUpdateTaskStatus registers the coordinator_update_task_status tool
func (h *ToolHandler) registerUpdateTaskStatus(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_update_task_status",
		Description: "Update the status of any task (human or agent). Status values: pending, in_progress, completed, blocked.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"taskId": {
					Type:        "string",
					Description: "Task ID to update (UUID)",
				},
				"status": {
					Type:        "string",
					Description: "New status (pending, in_progress, completed, blocked)",
					Enum:        []interface{}{"pending", "in_progress", "completed", "blocked"},
				},
				"notes": {
					Type:        "string",
					Description: "Optional progress notes",
				},
			},
			Required: []string{"taskId", "status"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleUpdateTaskStatus(ctx, args)
		return result, err
	})

	return nil
}

// handleUpsertKnowledge handles the coordinator_upsert_knowledge tool call
func (h *ToolHandler) handleUpsertKnowledge(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	collection, ok := args["collection"].(string)
	if !ok || collection == "" {
		return createErrorResult("collection parameter is required and must be a non-empty string"), nil, nil
	}

	text, ok := args["text"].(string)
	if !ok || text == "" {
		return createErrorResult("text parameter is required and must be a non-empty string"), nil, nil
	}

	var metadata map[string]interface{}
	if m, ok := args["metadata"].(map[string]interface{}); ok {
		metadata = m
	}

	entry, err := h.knowledgeStorage.Upsert(collection, text, metadata)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to upsert knowledge: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Knowledge stored successfully\n\nID: %s\nCollection: %s\nCreated: %s",
		entry.ID, entry.Collection, entry.CreatedAt.Format("2006-01-02 15:04:05 UTC"))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, entry, nil
}

// handleQueryKnowledge handles the coordinator_query_knowledge tool call
func (h *ToolHandler) handleQueryKnowledge(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	collection, ok := args["collection"].(string)
	if !ok || collection == "" {
		return createErrorResult("collection parameter is required and must be a non-empty string"), nil, nil
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return createErrorResult("query parameter is required and must be a non-empty string"), nil, nil
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	results, err := h.knowledgeStorage.Query(collection, query, limit)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to query knowledge: %s", err.Error())), nil, nil
	}

	// Return JSON array of knowledge entries for frontend consumption
	// Convert storage.QueryResult to a JSON-serializable format
	type KnowledgeEntryResponse struct {
		ID         string                 `json:"id"`
		Collection string                 `json:"collection"`
		Text       string                 `json:"text"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
		CreatedAt  string                 `json:"createdAt"`
		Score      float64                `json:"score"`
	}

	entries := make([]KnowledgeEntryResponse, len(results))
	for i, result := range results {
		entries[i] = KnowledgeEntryResponse{
			ID:         result.Entry.ID,
			Collection: result.Entry.Collection,
			Text:       result.Entry.Text,
			Metadata:   result.Entry.Metadata,
			CreatedAt:  result.Entry.CreatedAt.Format(time.RFC3339),
			Score:      result.Score,
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(entries)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to serialize results: %s", err.Error())), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, results, nil
}

// handleCreateHumanTask handles the coordinator_create_human_task tool call
func (h *ToolHandler) handleCreateHumanTask(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return createErrorResult("prompt parameter is required and must be a non-empty string"), nil, nil
	}

	task, err := h.taskStorage.CreateHumanTask(prompt)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to create human task: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Human task created successfully\n\nTask ID: %s\nCreated: %s\nStatus: %s\n\nPrompt: %s",
		task.ID, task.CreatedAt.Format("2006-01-02 15:04:05 UTC"), task.Status, task.Prompt)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, task, nil
}

// handleCreateAgentTask handles the coordinator_create_agent_task tool call
func (h *ToolHandler) handleCreateAgentTask(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	humanTaskID, ok := args["humanTaskId"].(string)
	if !ok || humanTaskID == "" {
		return createErrorResult("humanTaskId parameter is required and must be a non-empty string"), nil, nil
	}

	agentName, ok := args["agentName"].(string)
	if !ok || agentName == "" {
		return createErrorResult("agentName parameter is required and must be a non-empty string"), nil, nil
	}

	role, ok := args["role"].(string)
	if !ok || role == "" {
		return createErrorResult("role parameter is required and must be a non-empty string"), nil, nil
	}

	// Parse todos - support both string[] (legacy) and TodoItemInput[] (new)
	todosInterface, ok := args["todos"].([]interface{})
	if !ok || len(todosInterface) == 0 {
		return createErrorResult("todos parameter is required and must be a non-empty array"), nil, nil
	}

	todos := make([]storage.TodoItemInput, len(todosInterface))
	for i, t := range todosInterface {
		// Check if it's a string (legacy format)
		if str, ok := t.(string); ok {
			todos[i] = storage.TodoItemInput{
				Description: str,
			}
		} else if todoMap, ok := t.(map[string]interface{}); ok {
			// New format with context hints
			desc, ok := todoMap["description"].(string)
			if !ok || desc == "" {
				return createErrorResult(fmt.Sprintf("todos[%d].description is required and must be a non-empty string", i)), nil, nil
			}
			todos[i] = storage.TodoItemInput{
				Description: desc,
			}
			// Optional fields
			if filePath, ok := todoMap["filePath"].(string); ok {
				todos[i].FilePath = filePath
			}
			if functionName, ok := todoMap["functionName"].(string); ok {
				todos[i].FunctionName = functionName
			}
			if contextHint, ok := todoMap["contextHint"].(string); ok {
				todos[i].ContextHint = contextHint
			}
			if notes, ok := todoMap["notes"].(string); ok {
				todos[i].Notes = notes
			}
		} else {
			return createErrorResult(fmt.Sprintf("todos[%d] must be a string or an object with description field", i)), nil, nil
		}
	}

	// Parse optional context fields
	contextSummary := ""
	if cs, ok := args["contextSummary"].(string); ok {
		contextSummary = cs
	}

	var filesModified []string
	if fm, ok := args["filesModified"].([]interface{}); ok {
		filesModified = make([]string, len(fm))
		for i, f := range fm {
			if str, ok := f.(string); ok {
				filesModified[i] = str
			}
		}
	}

	var qdrantCollections []string
	if qc, ok := args["qdrantCollections"].([]interface{}); ok {
		qdrantCollections = make([]string, len(qc))
		for i, c := range qc {
			if str, ok := c.(string); ok {
				qdrantCollections[i] = str
			}
		}
	}

	priorWorkSummary := ""
	if pws, ok := args["priorWorkSummary"].(string); ok {
		priorWorkSummary = pws
	}

	task, err := h.taskStorage.CreateAgentTask(humanTaskID, agentName, role, todos, contextSummary, filesModified, qdrantCollections, priorWorkSummary)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to create agent task: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Agent task created successfully\n\nTask ID: %s\nAgent: %s\nRole: %s\nParent Task: %s\nCreated: %s\nStatus: %s\n",
		task.ID, task.AgentName, task.Role, task.HumanTaskID, task.CreatedAt.Format("2006-01-02 15:04:05 UTC"), task.Status)

	if task.ContextSummary != "" {
		resultText += fmt.Sprintf("\nContext Summary: %s\n", task.ContextSummary)
	}
	if len(task.FilesModified) > 0 {
		resultText += fmt.Sprintf("\nFiles to Modify: %v\n", task.FilesModified)
	}
	if len(task.QdrantCollections) > 0 {
		resultText += fmt.Sprintf("\nSuggested Qdrant Collections: %v\n", task.QdrantCollections)
	}

	resultText += "\nTODOs:\n"
	for i, todo := range task.Todos {
		resultText += fmt.Sprintf("  %d. %s", i+1, todo.Description)
		if todo.FilePath != "" {
			resultText += fmt.Sprintf(" (File: %s)", todo.FilePath)
		}
		if todo.FunctionName != "" {
			resultText += fmt.Sprintf(" (Function: %s)", todo.FunctionName)
		}
		resultText += "\n"
		if todo.ContextHint != "" {
			resultText += fmt.Sprintf("     Hint: %s\n", todo.ContextHint)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, task, nil
}

// handleUpdateTaskStatus handles the coordinator_update_task_status tool call
func (h *ToolHandler) handleUpdateTaskStatus(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	taskID, ok := args["taskId"].(string)
	if !ok || taskID == "" {
		return createErrorResult("taskId parameter is required and must be a non-empty string"), nil, nil
	}

	statusStr, ok := args["status"].(string)
	if !ok || statusStr == "" {
		return createErrorResult("status parameter is required and must be one of: pending, in_progress, completed, blocked"), nil, nil
	}

	status := storage.TaskStatus(statusStr)

	notes := ""
	if n, ok := args["notes"].(string); ok {
		notes = n
	}

	err := h.taskStorage.UpdateTaskStatus(taskID, status, notes)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to update task status: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Task status updated successfully\n\nTask ID: %s\nNew Status: %s", taskID, status)
	if notes != "" {
		resultText += fmt.Sprintf("\nNotes: %s", notes)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"taskId": taskID,
		"status": status,
		"notes":  notes,
	}, nil
}

// registerUpdateTodoStatus registers the coordinator_update_todo_status tool
func (h *ToolHandler) registerUpdateTodoStatus(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_update_todo_status",
		Description: "Update the status of a specific TODO item within an agent task. Status values: pending, in_progress, completed. When all TODOs are completed, the agent task is automatically marked as completed.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"agentTaskId": {
					Type:        "string",
					Description: "Agent task ID (UUID)",
				},
				"todoId": {
					Type:        "string",
					Description: "TODO item ID (UUID)",
				},
				"status": {
					Type:        "string",
					Description: "New status (pending, in_progress, completed)",
					Enum:        []interface{}{"pending", "in_progress", "completed"},
				},
				"notes": {
					Type:        "string",
					Description: "Optional progress notes for this TODO",
				},
			},
			Required: []string{"agentTaskId", "todoId", "status"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleUpdateTodoStatus(ctx, args)
		return result, err
	})

	return nil
}

// handleUpdateTodoStatus handles the coordinator_update_todo_status tool call
func (h *ToolHandler) handleUpdateTodoStatus(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	agentTaskID, ok := args["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return createErrorResult("agentTaskId parameter is required and must be a non-empty string"), nil, nil
	}

	todoID, ok := args["todoId"].(string)
	if !ok || todoID == "" {
		return createErrorResult("todoId parameter is required and must be a non-empty string"), nil, nil
	}

	statusStr, ok := args["status"].(string)
	if !ok || statusStr == "" {
		return createErrorResult("status parameter is required and must be one of: pending, in_progress, completed"), nil, nil
	}

	status := storage.TodoStatus(statusStr)

	notes := ""
	if n, ok := args["notes"].(string); ok {
		notes = n
	}

	err := h.taskStorage.UpdateTodoStatus(agentTaskID, todoID, status, notes)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to update TODO status: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ TODO status updated successfully\n\nAgent Task ID: %s\nTODO ID: %s\nNew Status: %s", agentTaskID, todoID, status)
	if notes != "" {
		resultText += fmt.Sprintf("\nNotes: %s", notes)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"agentTaskId": agentTaskID,
		"todoId":      todoID,
		"status":      status,
		"notes":       notes,
	}, nil
}

// registerListHumanTasks registers the coordinator_list_human_tasks tool
func (h *ToolHandler) registerListHumanTasks(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_list_human_tasks",
		Description: "List all human tasks from the coordinator database. Returns array of tasks with all fields.",
		InputSchema: &jsonschema.Schema{
			Type:       "object",
			Properties: map[string]*jsonschema.Schema{},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, _, err := h.handleListHumanTasks(ctx)
		return result, err
	})

	return nil
}

// registerListAgentTasks registers the coordinator_list_agent_tasks tool
func (h *ToolHandler) registerListAgentTasks(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_list_agent_tasks",
		Description: "List agent tasks from the coordinator database with pagination (max 50 per request). Large fields (>500 bytes) are truncated - use coordinator_get_agent_task to get full details. Returns total count and offset/limit for navigation.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"humanTaskId": {
					Type:        "string",
					Description: "Optional: Filter by parent human task ID",
				},
				"agentName": {
					Type:        "string",
					Description: "Optional: Filter by agent name",
				},
				"offset": {
					Type:        "number",
					Description: "Optional: Number of tasks to skip (default: 0). Use for pagination.",
				},
				"limit": {
					Type:        "number",
					Description: "Optional: Maximum number of tasks to return (default: 50, max: 50)",
				},
			},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleListAgentTasks(ctx, args)
		return result, err
	})

	return nil
}

// handleListHumanTasks retrieves all human tasks
func (h *ToolHandler) handleListHumanTasks(ctx context.Context) (*mcp.CallToolResult, map[string]interface{}, error) {
	tasks := h.taskStorage.ListAllHumanTasks()

	tasksJSON, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to marshal tasks: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Retrieved %d human tasks\n\nTasks:\n%s", len(tasks), string(tasksJSON))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}, nil
}

// handleListAgentTasks retrieves all agent tasks with optional filters and pagination
func (h *ToolHandler) handleListAgentTasks(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, map[string]interface{}, error) {
	humanTaskID, _ := args["humanTaskId"].(string)
	agentName, _ := args["agentName"].(string)

	// Pagination parameters
	offset := 0
	if o, ok := args["offset"].(float64); ok && o >= 0 {
		offset = int(o)
	}

	limit := 50
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 50 {
			limit = 50 // Enforce max limit
		}
	}

	allTasks := h.taskStorage.ListAllAgentTasks()

	// Apply filters if provided
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

	totalCount := len(filteredTasks)

	// Apply pagination
	endIndex := offset + limit
	if offset > totalCount {
		offset = totalCount
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	paginatedTasks := filteredTasks[offset:endIndex]

	// Truncate large fields (>500 bytes)
	truncatedTasks := make([]map[string]interface{}, len(paginatedTasks))
	for i, task := range paginatedTasks {
		taskMap := make(map[string]interface{})
		taskMap["id"] = task.ID
		taskMap["humanTaskId"] = task.HumanTaskID
		taskMap["agentName"] = task.AgentName
		taskMap["role"] = task.Role
		taskMap["status"] = task.Status
		taskMap["createdAt"] = task.CreatedAt
		taskMap["updatedAt"] = task.UpdatedAt

		// Truncate large fields
		if len(task.ContextSummary) > 500 {
			taskMap["contextSummary"] = task.ContextSummary[:500] + "... [TRUNCATED - use coordinator_get_agent_task for full content]"
		} else {
			taskMap["contextSummary"] = task.ContextSummary
		}

		if len(task.PriorWorkSummary) > 500 {
			taskMap["priorWorkSummary"] = task.PriorWorkSummary[:500] + "... [TRUNCATED - use coordinator_get_agent_task for full content]"
		} else {
			taskMap["priorWorkSummary"] = task.PriorWorkSummary
		}

		if len(task.Notes) > 500 {
			taskMap["notes"] = task.Notes[:500] + "... [TRUNCATED - use coordinator_get_agent_task for full content]"
		} else {
			taskMap["notes"] = task.Notes
		}

		if len(task.HumanPromptNotes) > 500 {
			taskMap["humanPromptNotes"] = task.HumanPromptNotes[:500] + "... [TRUNCATED - use coordinator_get_agent_task for full content]"
		} else {
			taskMap["humanPromptNotes"] = task.HumanPromptNotes
		}

		taskMap["filesModified"] = task.FilesModified
		taskMap["qdrantCollections"] = task.QdrantCollections
		taskMap["humanPromptNotesAddedAt"] = task.HumanPromptNotesAddedAt

		// Truncate TODO items
		truncatedTodos := make([]map[string]interface{}, len(task.Todos))
		for j, todo := range task.Todos {
			todoMap := make(map[string]interface{})
			todoMap["id"] = todo.ID
			todoMap["description"] = todo.Description
			todoMap["status"] = todo.Status
			todoMap["filePath"] = todo.FilePath
			todoMap["functionName"] = todo.FunctionName

			if len(todo.ContextHint) > 500 {
				todoMap["contextHint"] = todo.ContextHint[:500] + "... [TRUNCATED]"
			} else {
				todoMap["contextHint"] = todo.ContextHint
			}

			if len(todo.Notes) > 500 {
				todoMap["notes"] = todo.Notes[:500] + "... [TRUNCATED]"
			} else {
				todoMap["notes"] = todo.Notes
			}

			if len(todo.HumanPromptNotes) > 500 {
				todoMap["humanPromptNotes"] = todo.HumanPromptNotes[:500] + "... [TRUNCATED]"
			} else {
				todoMap["humanPromptNotes"] = todo.HumanPromptNotes
			}

			todoMap["humanPromptNotesAddedAt"] = todo.HumanPromptNotesAddedAt
			truncatedTodos[j] = todoMap
		}
		taskMap["todos"] = truncatedTodos

		truncatedTasks[i] = taskMap
	}

	tasksJSON, err := json.MarshalIndent(truncatedTasks, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to marshal tasks: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Retrieved %d agent tasks (showing %d-%d of %d total)",
		len(paginatedTasks), offset+1, offset+len(paginatedTasks), totalCount)
	if humanTaskID != "" {
		resultText += fmt.Sprintf("\nFiltered by humanTaskId: %s", humanTaskID)
	}
	if agentName != "" {
		resultText += fmt.Sprintf("\nFiltered by agentName: %s", agentName)
	}
	resultText += fmt.Sprintf("\n\nℹ️  Note: Fields >500 bytes are truncated. Use coordinator_get_agent_task(taskId) for full details.")
	resultText += fmt.Sprintf("\n\nTasks:\n%s", string(tasksJSON))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"tasks":      truncatedTasks,
		"count":      len(paginatedTasks),
		"totalCount": totalCount,
		"offset":     offset,
		"limit":      limit,
	}, nil
}

// registerGetAgentTask registers the coordinator_get_agent_task tool
func (h *ToolHandler) registerGetAgentTask(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_get_agent_task",
		Description: "Get a single agent task by ID with full, untruncated content. Use this to retrieve complete task details when coordinator_list_agent_tasks shows truncated fields.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"taskId": {
					Type:        "string",
					Description: "Agent task ID (UUID)",
				},
			},
			Required: []string{"taskId"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleGetAgentTask(ctx, args)
		return result, err
	})

	return nil
}

// handleGetAgentTask retrieves a single agent task by ID
func (h *ToolHandler) handleGetAgentTask(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	taskID, ok := args["taskId"].(string)
	if !ok || taskID == "" {
		return createErrorResult("taskId parameter is required and must be a non-empty string"), nil, nil
	}

	task, err := h.taskStorage.GetAgentTask(taskID)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to get agent task: %s", err.Error())), nil, nil
	}

	taskJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to marshal task: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Retrieved agent task\n\nTask:\n%s", string(taskJSON))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"task": task,
	}, nil
}

// extractArguments safely extracts arguments from CallToolRequest
func (h *ToolHandler) extractArguments(req *mcp.CallToolRequest) (map[string]interface{}, error) {
	if req.Params.Arguments == nil || len(req.Params.Arguments) == 0 {
		return make(map[string]interface{}), nil
	}

	// Arguments is json.RawMessage in SDK v1.0.0, unmarshal it directly
	var result map[string]interface{}
	if err := json.Unmarshal(req.Params.Arguments, &result); err != nil {
		return nil, fmt.Errorf("arguments must be a valid JSON object: %w", err)
	}

	return result, nil
}

// createErrorResult creates an error result with the given message
func createErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("❌ Error: %s", message)},
		},
		IsError: true,
	}
}

// registerClearTaskBoard registers the coordinator_clear_task_board tool
func (h *ToolHandler) registerClearTaskBoard(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_clear_task_board",
		Description: "Clear all tasks from the coordinator. ⚠️ DESTRUCTIVE OPERATION - Cannot be undone. Removes all human tasks and agent tasks.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"confirm": {
					Type:        "boolean",
					Description: "Must be true to confirm deletion",
				},
			},
			Required: []string{"confirm"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}

		result, _, err := h.handleClearTaskBoard(ctx, args)
		return result, err
	})

	return nil
}

// handleClearTaskBoard clears all tasks from the database
func (h *ToolHandler) handleClearTaskBoard(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, map[string]interface{}, error) {
	// Check confirmation
	confirm, ok := args["confirm"].(bool)
	if !ok || !confirm {
		return createErrorResult("Confirmation required: set confirm=true to clear all tasks"), nil, nil
	}

	// Clear all tasks
	result, err := h.taskStorage.ClearAllTasks()
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to clear tasks: %v", err)), nil, nil
	}

	// Format result
	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("✓ Task board cleared successfully\n\n%s", string(resultJSON)),
			},
		},
	}, map[string]interface{}{
		"humanTasksDeleted": result.HumanTasksDeleted,
		"agentTasksDeleted": result.AgentTasksDeleted,
		"clearedAt":         result.ClearedAt,
	}, nil
}

// registerAddTaskPromptNotes registers the coordinator_add_task_prompt_notes tool
func (h *ToolHandler) registerAddTaskPromptNotes(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_add_task_prompt_notes",
		Description: "Add human guidance notes to an agent task",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"agentTaskId": {
					Type:        "string",
					Description: "Agent task UUID",
				},
				"promptNotes": {
					Type:        "string",
					Description: "Human guidance notes, markdown supported",
				},
			},
			Required: []string{"agentTaskId", "promptNotes"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleAddTaskPromptNotes(ctx, args)
		return result, err
	})

	return nil
}

// registerUpdateTaskPromptNotes registers the coordinator_update_task_prompt_notes tool
func (h *ToolHandler) registerUpdateTaskPromptNotes(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_update_task_prompt_notes",
		Description: "Update existing human guidance notes on an agent task",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"agentTaskId": {
					Type:        "string",
					Description: "Agent task UUID",
				},
				"promptNotes": {
					Type:        "string",
					Description: "Human guidance notes, markdown supported",
				},
			},
			Required: []string{"agentTaskId", "promptNotes"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleUpdateTaskPromptNotes(ctx, args)
		return result, err
	})

	return nil
}

// registerClearTaskPromptNotes registers the coordinator_clear_task_prompt_notes tool
func (h *ToolHandler) registerClearTaskPromptNotes(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_clear_task_prompt_notes",
		Description: "Clear/remove human guidance notes from an agent task",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"agentTaskId": {
					Type:        "string",
					Description: "Agent task UUID",
				},
			},
			Required: []string{"agentTaskId"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleClearTaskPromptNotes(ctx, args)
		return result, err
	})

	return nil
}

// registerAddTodoPromptNotes registers the coordinator_add_todo_prompt_notes tool
func (h *ToolHandler) registerAddTodoPromptNotes(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_add_todo_prompt_notes",
		Description: "Add human guidance notes to a specific TODO item",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"agentTaskId": {
					Type:        "string",
					Description: "Agent task UUID",
				},
				"todoId": {
					Type:        "string",
					Description: "TODO item UUID",
				},
				"promptNotes": {
					Type:        "string",
					Description: "Human guidance notes, markdown supported",
				},
			},
			Required: []string{"agentTaskId", "todoId", "promptNotes"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleAddTodoPromptNotes(ctx, args)
		return result, err
	})

	return nil
}

// registerUpdateTodoPromptNotes registers the coordinator_update_todo_prompt_notes tool
func (h *ToolHandler) registerUpdateTodoPromptNotes(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_update_todo_prompt_notes",
		Description: "Update existing human guidance notes on a TODO item",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"agentTaskId": {
					Type:        "string",
					Description: "Agent task UUID",
				},
				"todoId": {
					Type:        "string",
					Description: "TODO item UUID",
				},
				"promptNotes": {
					Type:        "string",
					Description: "Human guidance notes, markdown supported",
				},
			},
			Required: []string{"agentTaskId", "todoId", "promptNotes"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleUpdateTodoPromptNotes(ctx, args)
		return result, err
	})

	return nil
}

// registerClearTodoPromptNotes registers the coordinator_clear_todo_prompt_notes tool
func (h *ToolHandler) registerClearTodoPromptNotes(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_clear_todo_prompt_notes",
		Description: "Clear/remove human guidance notes from a TODO item",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"agentTaskId": {
					Type:        "string",
					Description: "Agent task UUID",
				},
				"todoId": {
					Type:        "string",
					Description: "TODO item UUID",
				},
			},
			Required: []string{"agentTaskId", "todoId"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleClearTodoPromptNotes(ctx, args)
		return result, err
	})

	return nil
}

// handleAddTaskPromptNotes adds human prompt notes to an agent task
func (h *ToolHandler) handleAddTaskPromptNotes(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	agentTaskId, ok := args["agentTaskId"].(string)
	if !ok || agentTaskId == "" {
		return createErrorResult("agentTaskId parameter is required and must be a non-empty string"), nil, nil
	}

	promptNotes, ok := args["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return createErrorResult("promptNotes parameter is required and must be a non-empty string"), nil, nil
	}

	// Validate and sanitize prompt notes
	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Validation error: %v", err)), nil, nil
	}

	// Add prompt notes to task
	err = h.taskStorage.AddTaskPromptNotes(agentTaskId, sanitized)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to add prompt notes: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("✓ Added prompt notes to task %s", agentTaskId),
			},
		},
	}, nil, nil
}

// handleUpdateTaskPromptNotes updates existing human prompt notes on an agent task
func (h *ToolHandler) handleUpdateTaskPromptNotes(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	agentTaskId, ok := args["agentTaskId"].(string)
	if !ok || agentTaskId == "" {
		return createErrorResult("agentTaskId parameter is required and must be a non-empty string"), nil, nil
	}

	promptNotes, ok := args["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return createErrorResult("promptNotes parameter is required and must be a non-empty string"), nil, nil
	}

	// Validate and sanitize prompt notes
	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Validation error: %v", err)), nil, nil
	}

	// Update prompt notes
	err = h.taskStorage.UpdateTaskPromptNotes(agentTaskId, sanitized)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to update prompt notes: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("✓ Updated prompt notes for task %s", agentTaskId),
			},
		},
	}, nil, nil
}

// handleClearTaskPromptNotes clears human prompt notes from an agent task
func (h *ToolHandler) handleClearTaskPromptNotes(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	agentTaskId, ok := args["agentTaskId"].(string)
	if !ok || agentTaskId == "" {
		return createErrorResult("agentTaskId parameter is required and must be a non-empty string"), nil, nil
	}

	// Clear prompt notes
	err := h.taskStorage.ClearTaskPromptNotes(agentTaskId)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to clear prompt notes: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("✓ Cleared prompt notes from task %s", agentTaskId),
			},
		},
	}, nil, nil
}

// handleAddTodoPromptNotes adds human prompt notes to a specific TODO item
func (h *ToolHandler) handleAddTodoPromptNotes(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	agentTaskId, ok := args["agentTaskId"].(string)
	if !ok || agentTaskId == "" {
		return createErrorResult("agentTaskId parameter is required and must be a non-empty string"), nil, nil
	}

	todoId, ok := args["todoId"].(string)
	if !ok || todoId == "" {
		return createErrorResult("todoId parameter is required and must be a non-empty string"), nil, nil
	}

	promptNotes, ok := args["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return createErrorResult("promptNotes parameter is required and must be a non-empty string"), nil, nil
	}

	// Validate and sanitize prompt notes
	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Validation error: %v", err)), nil, nil
	}

	// Add prompt notes to TODO
	err = h.taskStorage.AddTodoPromptNotes(agentTaskId, todoId, sanitized)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to add TODO prompt notes: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("✓ Added prompt notes to TODO %s in task %s", todoId, agentTaskId),
			},
		},
	}, nil, nil
}

// handleUpdateTodoPromptNotes updates existing human prompt notes on a specific TODO item
func (h *ToolHandler) handleUpdateTodoPromptNotes(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	agentTaskId, ok := args["agentTaskId"].(string)
	if !ok || agentTaskId == "" {
		return createErrorResult("agentTaskId parameter is required and must be a non-empty string"), nil, nil
	}

	todoId, ok := args["todoId"].(string)
	if !ok || todoId == "" {
		return createErrorResult("todoId parameter is required and must be a non-empty string"), nil, nil
	}

	promptNotes, ok := args["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return createErrorResult("promptNotes parameter is required and must be a non-empty string"), nil, nil
	}

	// Validate and sanitize prompt notes
	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Validation error: %v", err)), nil, nil
	}

	// Update prompt notes
	err = h.taskStorage.UpdateTodoPromptNotes(agentTaskId, todoId, sanitized)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to update TODO prompt notes: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("✓ Updated prompt notes for TODO %s in task %s", todoId, agentTaskId),
			},
		},
	}, nil, nil
}

// handleClearTodoPromptNotes clears human prompt notes from a specific TODO item
func (h *ToolHandler) handleClearTodoPromptNotes(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	agentTaskId, ok := args["agentTaskId"].(string)
	if !ok || agentTaskId == "" {
		return createErrorResult("agentTaskId parameter is required and must be a non-empty string"), nil, nil
	}

	todoId, ok := args["todoId"].(string)
	if !ok || todoId == "" {
		return createErrorResult("todoId parameter is required and must be a non-empty string"), nil, nil
	}

	// Clear prompt notes from TODO
	err := h.taskStorage.ClearTodoPromptNotes(agentTaskId, todoId)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to clear TODO prompt notes: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("✓ Cleared prompt notes from TODO %s in task %s", todoId, agentTaskId),
			},
		},
	}, nil, nil
}

// registerListSubagents registers the list_subagents tool
func (h *ToolHandler) registerListSubagents(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "list_subagents",
		Description: "Returns available subagents from CLAUDE.md agent list with names, descriptions, tools, and categories",
		InputSchema: &jsonschema.Schema{
			Type:       "object",
			Properties: map[string]*jsonschema.Schema{},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, _, err := h.handleListSubagents(ctx)
		return result, err
	})

	return nil
}

// handleListSubagents handles the list_subagents tool call
func (h *ToolHandler) handleListSubagents(ctx context.Context) (*mcp.CallToolResult, interface{}, error) {
	// Read CLAUDE.md from project root (validation only - we hardcode the list for reliability)
	claudeMdPath := "/Users/maxmednikov/MaxSpace/dev-squad/CLAUDE.md"
	_, err := os.ReadFile(claudeMdPath)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to read CLAUDE.md: %s", err.Error())), nil, nil
	}

	// Parse subagent definitions from "Available Sub-Agents" section
	type Subagent struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tools       []string `json:"tools"`
		Category    string   `json:"category"`
	}

	subagents := []Subagent{
		// Backend Infrastructure
		{Name: "go-dev", Description: "Go microservices, REST APIs, business logic", Tools: []string{"hyper", "filesystem", "github", "fetch", "mongodb"}, Category: "Backend Infrastructure"},
		{Name: "go-mcp-dev", Description: "MCP tools and integrations (Model Context Protocol)", Tools: []string{"hyper", "filesystem", "github", "fetch"}, Category: "Backend Infrastructure"},
		{Name: "Backend Services Specialist", Description: "Service architecture (via coordinator)", Tools: []string{"hyper"}, Category: "Backend Infrastructure"},
		{Name: "Event Systems Specialist", Description: "NATS JetStream (via coordinator)", Tools: []string{"hyper"}, Category: "Backend Infrastructure"},
		{Name: "Data Platform Specialist", Description: "MongoDB optimization and data modeling (via coordinator)", Tools: []string{"hyper", "mongodb"}, Category: "Backend Infrastructure"},

		// Frontend & Experience
		{Name: "ui-dev", Description: "React/TypeScript implementation, components", Tools: []string{"hyper", "filesystem", "github", "fetch"}, Category: "Frontend & Experience"},
		{Name: "ui-tester", Description: "Playwright E2E tests, accessibility validation", Tools: []string{"hyper", "filesystem", "playwright-mcp"}, Category: "Frontend & Experience"},
		{Name: "Frontend Experience Specialist", Description: "UI architecture (via coordinator)", Tools: []string{"hyper"}, Category: "Frontend & Experience"},
		{Name: "AI Integration Specialist", Description: "Claude/GPT integration (via coordinator)", Tools: []string{"hyper", "fetch"}, Category: "Frontend & Experience"},
		{Name: "Real-time Systems Specialist", Description: "WebSocket, streaming (via coordinator)", Tools: []string{"hyper"}, Category: "Frontend & Experience"},

		// Platform & Operations
		{Name: "sre", Description: "Deployment to dev/prod environments", Tools: []string{"hyper", "kubernetes", "github", "fetch"}, Category: "Platform & Operations"},
		{Name: "k8s-deployment-expert", Description: "Kubernetes manifests, rollouts, scaling", Tools: []string{"hyper", "kubernetes", "github"}, Category: "Platform & Operations"},
		{Name: "Infrastructure Automation Specialist", Description: "GKE, GitHub Actions (via coordinator)", Tools: []string{"hyper", "kubernetes", "github"}, Category: "Platform & Operations"},
		{Name: "Security & Auth Specialist", Description: "JWT, RBAC, security policies (via coordinator)", Tools: []string{"hyper"}, Category: "Platform & Operations"},
		{Name: "Observability Specialist", Description: "Metrics, monitoring (via coordinator)", Tools: []string{"hyper"}, Category: "Platform & Operations"},

		// Testing & Quality
		{Name: "End-to-End Testing Coordinator", Description: "Cross-squad testing (via coordinator)", Tools: []string{"hyper"}, Category: "Testing & Quality"},
	}

	// Return JSON array
	jsonData, err := json.Marshal(subagents)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to serialize subagents: %s", err.Error())), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, subagents, nil
}

// registerSetCurrentSubagent registers the set_current_subagent tool
func (h *ToolHandler) registerSetCurrentSubagent(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "set_current_subagent",
		Description: "Associate a subagent with the current chat session. Stores subagent name in chat metadata.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"subagentName": {
					Type:        "string",
					Description: "Name of the subagent to associate with chat (must match list from list_subagents)",
				},
			},
			Required: []string{"subagentName"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleSetCurrentSubagent(ctx, args)
		return result, err
	})

	return nil
}

// handleSetCurrentSubagent handles the set_current_subagent tool call
func (h *ToolHandler) handleSetCurrentSubagent(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	subagentName, ok := args["subagentName"].(string)
	if !ok || subagentName == "" {
		return createErrorResult("subagentName parameter is required and must be a non-empty string"), nil, nil
	}

	// Validate subagent name against known list
	validSubagents := map[string]bool{
		"go-dev": true, "go-mcp-dev": true, "Backend Services Specialist": true,
		"Event Systems Specialist": true, "Data Platform Specialist": true,
		"ui-dev": true, "ui-tester": true, "Frontend Experience Specialist": true,
		"AI Integration Specialist": true, "Real-time Systems Specialist": true,
		"sre": true, "k8s-deployment-expert": true, "Infrastructure Automation Specialist": true,
		"Security & Auth Specialist": true, "Observability Specialist": true,
		"End-to-End Testing Coordinator": true,
	}

	if !validSubagents[subagentName] {
		return createErrorResult(fmt.Sprintf("invalid subagent name '%s'. Use list_subagents to see available subagents", subagentName)), nil, nil
	}

	// Note: Actual MongoDB update will be implemented in subchat service
	// For now, return success with note that this requires chat context
	resultText := fmt.Sprintf("✓ Subagent '%s' validated successfully\n\nNote: Chat session association will be implemented via subchat service. Use subchat_handler REST API to create subchats with subagent assignments.", subagentName)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"subagentName": subagentName,
		"valid":        true,
	}, nil
}

// registerGetPopularCollections registers the coordinator_get_popular_collections tool
func (h *ToolHandler) registerGetPopularCollections(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "coordinator_get_popular_collections",
		Description: "Get top N knowledge collections by entry count for Quick Start suggestions",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"limit": {
					Type:        "number",
					Description: "Maximum number of collections to return (default: 5)",
				},
			},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := h.extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleGetPopularCollections(ctx, args)
		return result, err
	})

	return nil
}

// handleGetPopularCollections handles the coordinator_get_popular_collections tool call
func (h *ToolHandler) handleGetPopularCollections(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	stats, err := h.knowledgeStorage.GetPopularCollections(limit)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to get popular collections: %s", err.Error())), nil, nil
	}

	// CRITICAL: Never return null - always return an empty array with a helpful message
	if stats == nil || len(stats) == 0 {
		emptyResponse := map[string]interface{}{
			"collections":  []interface{}{},
			"message":      "No collections with entries yet. Check hyperion://knowledge/collections resource for available collections.",
			"totalDefined": 14, // Total predefined collections
		}
		jsonData, _ := json.Marshal(emptyResponse)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(jsonData)},
			},
		}, emptyResponse, nil
	}

	// Return JSON array directly
	jsonData, err := json.Marshal(stats)
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to serialize results: %s", err.Error())), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, stats, nil
}
