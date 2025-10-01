package handlers

import (
	"context"
	"encoding/json"
	"fmt"

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

	// Register coordinator_clear_task_board
	if err := h.registerClearTaskBoard(server); err != nil {
		return fmt.Errorf("failed to register clear_task_board tool: %w", err)
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

	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("No knowledge found in collection '%s' for query: %s", collection, query)},
			},
		}, results, nil
	}

	// Format results
	resultText := fmt.Sprintf("Found %d knowledge entries:\n\n", len(results))
	for i, result := range results {
		resultText += fmt.Sprintf("Result %d (Score: %.2f)\n", i+1, result.Score)
		resultText += fmt.Sprintf("ID: %s\n", result.Entry.ID)
		resultText += fmt.Sprintf("Created: %s\n", result.Entry.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
		resultText += fmt.Sprintf("Text: %s\n", result.Entry.Text)
		if len(result.Entry.Metadata) > 0 {
			metadataJSON, _ := json.MarshalIndent(result.Entry.Metadata, "", "  ")
			resultText += fmt.Sprintf("Metadata: %s\n", string(metadataJSON))
		}
		resultText += "\n---\n\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
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
		Description: "List all agent tasks from the coordinator database. Returns array of tasks with all fields.",
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

// handleListAgentTasks retrieves all agent tasks with optional filters
func (h *ToolHandler) handleListAgentTasks(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, map[string]interface{}, error) {
	humanTaskID, _ := args["humanTaskId"].(string)
	agentName, _ := args["agentName"].(string)

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

	tasksJSON, err := json.MarshalIndent(filteredTasks, "", "  ")
	if err != nil {
		return createErrorResult(fmt.Sprintf("failed to marshal tasks: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Retrieved %d agent tasks", len(filteredTasks))
	if humanTaskID != "" {
		resultText += fmt.Sprintf(" (filtered by humanTaskId: %s)", humanTaskID)
	}
	if agentName != "" {
		resultText += fmt.Sprintf(" (filtered by agentName: %s)", agentName)
	}
	resultText += fmt.Sprintf("\n\nTasks:\n%s", string(tasksJSON))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"tasks": filteredTasks,
		"count": len(filteredTasks),
	}, nil
}

// extractArguments safely extracts arguments from CallToolRequest
func (h *ToolHandler) extractArguments(req *mcp.CallToolRequest) (map[string]interface{}, error) {
	if req.Params.Arguments == nil {
		return make(map[string]interface{}), nil
	}

	// First try direct type assertion
	args, ok := req.Params.Arguments.(map[string]interface{})
	if ok {
		return args, nil
	}

	// If that fails, try JSON round-trip for proper type conversion
	jsonBytes, err := json.Marshal(req.Params.Arguments)
	if err != nil {
		return nil, fmt.Errorf("arguments must be serializable: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("arguments must be unmarshable to map[string]interface{}: %w", err)
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