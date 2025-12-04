package mcp

import (
	"context"
	"fmt"

	"hyper/internal/mcp/storage"
)

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
