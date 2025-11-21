# How to Register MCP Tools

**Collection:** howto
**Tags:** mcp, tools, ai-integration, model-context-protocol, go
**Version:** 1.0
**Last Updated:** 2025-11-21

---

## Overview

This guide explains how to register and implement Model Context Protocol (MCP) tools that extend AI agent capabilities. MCP tools allow AI assistants to interact with your system through well-defined interfaces.

## Prerequisites

- Understanding of function calling in LLMs
- Go 1.25 with MCP SDK
- Knowledge of JSON Schema
- Familiarity with [MCP Server Architecture](../ai-integration/mcp-server-architecture.md)

## When to Use This Guide

- Creating custom tools for AI agents
- Extending agent capabilities with domain-specific operations
- Integrating external systems with AI workflows
- Building tool-based AI applications

---

## Steps

### Step 1: Define Tool Schema

Create tool definition with JSON Schema for parameters:

```go
package tools

import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Define your tool
var ListTasksTool = mcp.Tool{
    Name:        "list_tasks",
    Description: "List tasks with optional filtering by status",
    InputSchema: mcp.ToolInputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "status": map[string]interface{}{
                "type":        "string",
                "description": "Filter tasks by status",
                "enum":        []string{"pending", "in_progress", "completed", "blocked"},
            },
            "limit": map[string]interface{}{
                "type":        "number",
                "description": "Maximum number of tasks to return",
                "minimum":     1,
                "maximum":     100,
                "default":     20,
            },
        },
    },
}
```

**Schema Best Practices:**
- Use clear, descriptive names (`list_tasks` not `lt`)
- Provide detailed descriptions for each parameter
- Use enums for fixed value sets
- Set constraints (min/max) where applicable
- Define default values for optional parameters

### Step 2: Create Tool Handler

Implement the handler that executes the tool:

```go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "go.uber.org/zap"
)

type TaskToolHandler struct {
    taskService TaskService
    logger      *zap.Logger
}

func NewTaskToolHandler(service TaskService, logger *zap.Logger) *TaskToolHandler {
    return &TaskToolHandler{
        taskService: service,
        logger:      logger,
    }
}

// HandleListTasks processes the list_tasks tool call
func (h *TaskToolHandler) HandleListTasks(
    ctx context.Context,
    params map[string]interface{},
) (*mcp.ToolResponse, error) {
    // Step 1: Extract and validate parameters
    request, err := h.parseListTasksParams(params)
    if err != nil {
        return nil, fmt.Errorf("invalid parameters: %w", err)
    }

    // Step 2: Execute business logic
    tasks, err := h.taskService.List(ctx, request.Status, request.Limit)
    if err != nil {
        h.logger.Error("Failed to list tasks",
            zap.Error(err),
            zap.String("status", request.Status),
        )
        return nil, fmt.Errorf("failed to list tasks: %w", err)
    }

    // Step 3: Format response
    return h.formatTasksResponse(tasks)
}

type ListTasksRequest struct {
    Status string
    Limit  int
}

func (h *TaskToolHandler) parseListTasksParams(
    params map[string]interface{},
) (*ListTasksRequest, error) {
    request := &ListTasksRequest{
        Limit: 20, // default
    }

    // Extract status (optional)
    if status, ok := params["status"].(string); ok {
        // Validate enum
        if !isValidStatus(status) {
            return nil, fmt.Errorf("invalid status: %s", status)
        }
        request.Status = status
    }

    // Extract limit (optional)
    if limit, ok := params["limit"].(float64); ok {
        if limit < 1 || limit > 100 {
            return nil, fmt.Errorf("limit must be between 1 and 100")
        }
        request.Limit = int(limit)
    }

    return request, nil
}

func (h *TaskToolHandler) formatTasksResponse(
    tasks []Task,
) (*mcp.ToolResponse, error) {
    // Convert to JSON for AI consumption
    data, err := json.MarshalIndent(tasks, "", "  ")
    if err != nil {
        return nil, fmt.Errorf("failed to marshal tasks: %w", err)
    }

    return &mcp.ToolResponse{
        Content: []mcp.Content{
            {
                Type: "text",
                Text: string(data),
            },
        },
    }, nil
}

func isValidStatus(status string) bool {
    validStatuses := []string{"pending", "in_progress", "completed", "blocked"}
    for _, v := range validStatuses {
        if v == status {
            return true
        }
    }
    return false
}
```

### Step 3: Register Tool with MCP Server

Add tool to MCP server's tool registry:

```go
package main

import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "your-project/internal/handlers"
    "your-project/internal/tools"
)

func initializeMCPServer(
    taskHandler *handlers.TaskToolHandler,
    logger *zap.Logger,
) *mcp.Server {
    server := mcp.NewServer()

    // Register list_tasks tool
    server.RegisterTool(tools.ListTasksTool, func(ctx context.Context, params map[string]interface{}) (*mcp.ToolResponse, error) {
        return taskHandler.HandleListTasks(ctx, params)
    })

    // Register more tools...
    // server.RegisterTool(tools.CreateTaskTool, taskHandler.HandleCreateTask)
    // server.RegisterTool(tools.SearchCodeTool, codeHandler.HandleSearchCode)

    logger.Info("MCP server initialized",
        zap.Int("toolCount", len(server.Tools())),
    )

    return server
}
```

### Step 4: Implement Complex Tool (Multi-Step)

For tools requiring multiple operations:

```go
var CreateAgentTaskTool = mcp.Tool{
    Name:        "create_agent_task",
    Description: "Create a new agent task with todos and context",
    InputSchema: mcp.ToolInputSchema{
        Type:     "object",
        Required: []string{"humanTaskId", "agentName", "role", "todos"},
        Properties: map[string]interface{}{
            "humanTaskId": map[string]interface{}{
                "type":        "string",
                "description": "Parent human task ID",
            },
            "agentName": map[string]interface{}{
                "type":        "string",
                "description": "Name of the agent (e.g., 'go-dev', 'ui-dev')",
            },
            "role": map[string]interface{}{
                "type":        "string",
                "description": "Agent's role/mission (50-100 words)",
            },
            "todos": map[string]interface{}{
                "type":        "array",
                "description": "List of todo items",
                "items": map[string]interface{}{
                    "type": "object",
                    "required": []string{"description"},
                    "properties": map[string]interface{}{
                        "description": map[string]interface{}{
                            "type": "string",
                        },
                        "filePath": map[string]interface{}{
                            "type": "string",
                        },
                        "contextHint": map[string]interface{}{
                            "type": "string",
                        },
                    },
                },
            },
            "contextSummary": map[string]interface{}{
                "type":        "string",
                "description": "Detailed context (150-250 words)",
            },
        },
    },
}

func (h *TaskToolHandler) HandleCreateAgentTask(
    ctx context.Context,
    params map[string]interface{},
) (*mcp.ToolResponse, error) {
    // Parse complex nested structure
    request, err := h.parseCreateAgentTaskParams(params)
    if err != nil {
        return nil, err
    }

    // Validate parent task exists
    parentTask, err := h.taskService.GetHumanTask(ctx, request.HumanTaskID)
    if err != nil {
        return nil, fmt.Errorf("parent task not found: %w", err)
    }

    // Create agent task
    agentTask, err := h.taskService.CreateAgentTask(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("failed to create agent task: %w", err)
    }

    // Update parent task reference
    err = h.taskService.LinkAgentTask(ctx, parentTask.ID, agentTask.ID)
    if err != nil {
        h.logger.Warn("Failed to link agent task to parent",
            zap.Error(err),
            zap.String("parentId", parentTask.ID),
            zap.String("agentTaskId", agentTask.ID),
        )
    }

    // Return formatted response
    return h.formatAgentTaskResponse(agentTask)
}
```

### Step 5: Implement Error Handling

Provide clear error messages for AI consumption:

```go
func (h *TaskToolHandler) HandleListTasks(
    ctx context.Context,
    params map[string]interface{},
) (*mcp.ToolResponse, error) {
    request, err := h.parseListTasksParams(params)
    if err != nil {
        // Return user-friendly error
        return &mcp.ToolResponse{
            IsError: true,
            Content: []mcp.Content{
                {
                    Type: "text",
                    Text: fmt.Sprintf("Parameter validation failed: %s", err.Error()),
                },
            },
        }, nil
    }

    tasks, err := h.taskService.List(ctx, request.Status, request.Limit)
    if err != nil {
        // Log technical error
        h.logger.Error("Task service error", zap.Error(err))
        
        // Return generic error to AI
        return &mcp.ToolResponse{
            IsError: true,
            Content: []mcp.Content{
                {
                    Type: "text",
                    Text: "Failed to retrieve tasks. Please try again or contact support.",
                },
            },
        }, nil
    }

    return h.formatTasksResponse(tasks)
}
```

### Step 6: Add Tool Documentation

Document tool behavior for AI and developers:

```go
var SearchCodeTool = mcp.Tool{
    Name: "search_code",
    Description: `Perform semantic code search across indexed repositories.
    
Returns relevant code chunks with file paths, line numbers, and surrounding context.
Use this when you need to:
- Find implementation of specific functionality
- Locate where a particular API is used
- Understand code structure and patterns
- Search for similar code examples

Best practices:
- Use descriptive queries ("authentication middleware implementation")
- Specify file types to narrow results (fileTypes: [".go", ".ts"])
- Start with chunk-m retrieve mode for balanced results
- Use minScore >= 0.7 for high-quality matches`,
    InputSchema: mcp.ToolInputSchema{
        Type:     "object",
        Required: []string{"query"},
        Properties: map[string]interface{}{
            "query": map[string]interface{}{
                "type":        "string",
                "description": "Semantic search query describing what code you're looking for",
            },
            "limit": map[string]interface{}{
                "type":        "number",
                "description": "Maximum results to return",
                "default":     10,
                "minimum":     1,
                "maximum":     50,
            },
            "minScore": map[string]interface{}{
                "type":        "number",
                "description": "Minimum similarity score (0.0-1.0)",
                "default":     0.0,
                "minimum":     0.0,
                "maximum":     1.0,
            },
            "retrieve": map[string]interface{}{
                "type":        "string",
                "description": "How much context to return",
                "enum":        []string{"chunk-s", "chunk-m", "chunk-l", "full"},
                "default":     "chunk-m",
            },
        },
    },
}
```

### Step 7: Test Tool Registration

Create tests for tool handlers:

```go
package handlers_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestHandleListTasks(t *testing.T) {
    // Setup
    mockService := new(MockTaskService)
    logger, _ := zap.NewDevelopment()
    handler := handlers.NewTaskToolHandler(mockService, logger)

    // Mock service response
    expectedTasks := []Task{
        {ID: "1", Status: "pending"},
        {ID: "2", Status: "pending"},
    }
    mockService.On("List", mock.Anything, "pending", 20).
        Return(expectedTasks, nil)

    // Execute
    params := map[string]interface{}{
        "status": "pending",
        "limit":  float64(20),
    }
    
    response, err := handler.HandleListTasks(context.Background(), params)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.False(t, response.IsError)
    mockService.AssertExpectations(t)
}

func TestHandleListTasks_InvalidStatus(t *testing.T) {
    mockService := new(MockTaskService)
    logger, _ := zap.NewDevelopment()
    handler := handlers.NewTaskToolHandler(mockService, logger)

    params := map[string]interface{}{
        "status": "invalid_status",
    }
    
    response, err := handler.HandleListTasks(context.Background(), params)

    // Should return error in response
    assert.NoError(t, err)
    assert.True(t, response.IsError)
    assert.Contains(t, response.Content[0].Text, "invalid status")
}
```

---

## Best Practices

### 1. Clear Tool Names
Use descriptive verb_noun pattern:
- ✅ `create_task`, `list_users`, `search_code`
- ❌ `task`, `users`, `search`

### 2. Comprehensive Descriptions
Help AI understand when and how to use the tool:
```go
Description: "Create a new task. Use when the user requests creating, adding, or starting a new task. Returns the created task with ID."
```

### 3. Parameter Validation
Validate all inputs before processing:
```go
if limit < 1 || limit > 100 {
    return nil, fmt.Errorf("limit must be between 1 and 100, got %d", limit)
}
```

### 4. Structured Responses
Return well-formatted JSON for AI parsing:
```go
response := map[string]interface{}{
    "taskId": task.ID,
    "status": task.Status,
    "createdAt": task.CreatedAt,
}
```

### 5. Error Messages
Provide actionable error messages:
```go
// ✅ GOOD
"Task 'abc-123' not found. Use list_tasks to see available tasks."

// ❌ BAD
"Not found"
```

---

## Common Pitfalls

### 1. Missing Required Fields
```go
// Declare required fields
Required: []string{"humanTaskId", "agentName"},
```

### 2. Type Mismatches
JSON numbers are float64 in Go:
```go
// ❌ BAD
limit := params["limit"].(int) // panic

// ✅ GOOD
limit := int(params["limit"].(float64))
```

### 3. Not Handling Optional Parameters
```go
// Check existence before assertion
if status, ok := params["status"].(string); ok {
    request.Status = status
}
```

### 4. Returning Go Errors Directly
```go
// ❌ BAD - Internal error exposed
return nil, err

// ✅ GOOD - User-friendly message
return nil, fmt.Errorf("failed to create task: please check task name is unique")
```

---

## Related Documentation

- [MCP Server Architecture](../ai-integration/mcp-server-architecture.md) - Overall MCP design
- [Task Coordination System](../ai-integration/task-coordination-system.md) - Task management tools
- [Data Contracts](../data-contracts.md) - Type definitions

---

## Troubleshooting

### Issue: "Tool not found"

**Cause:** Tool not registered in MCP server

**Solution:**
```go
// Verify registration
server.RegisterTool(tools.MyTool, handler.HandleMyTool)

// List registered tools
fmt.Println("Registered tools:", server.Tools())
```

### Issue: "Invalid parameters"

**Cause:** Schema mismatch or parsing error

**Solution:**
- Validate parameters match schema
- Check JSON types (numbers are float64)
- Add logging to see what AI sends

### Issue: "Tool times out"

**Cause:** Handler takes too long

**Solution:**
```go
// Add timeout to context
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

result, err := handler.Execute(ctx, params)
```
