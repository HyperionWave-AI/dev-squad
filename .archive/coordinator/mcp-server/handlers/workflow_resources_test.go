package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"hyperion-coordinator-mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTaskStorage for testing workflow resources
type MockWorkflowTaskStorage struct {
	agentTasks []*storage.AgentTask
}

func (m *MockWorkflowTaskStorage) CreateHumanTask(prompt string) (*storage.HumanTask, error) {
	return nil, nil
}

func (m *MockWorkflowTaskStorage) CreateAgentTask(humanTaskID, agentName, role string, todos []storage.TodoItemInput, contextSummary string, filesModified []string, qdrantCollections []string, priorWorkSummary string) (*storage.AgentTask, error) {
	return nil, nil
}

func (m *MockWorkflowTaskStorage) GetHumanTask(taskID string) (*storage.HumanTask, error) {
	return nil, nil
}

func (m *MockWorkflowTaskStorage) GetAgentTask(taskID string) (*storage.AgentTask, error) {
	return nil, nil
}

func (m *MockWorkflowTaskStorage) GetAgentTasksByName(agentName string) ([]*storage.AgentTask, error) {
	return nil, nil
}

func (m *MockWorkflowTaskStorage) ListAllHumanTasks() []*storage.HumanTask {
	return []*storage.HumanTask{}
}

func (m *MockWorkflowTaskStorage) ListAllAgentTasks() []*storage.AgentTask {
	return m.agentTasks
}

func (m *MockWorkflowTaskStorage) UpdateTaskStatus(taskID string, status storage.TaskStatus, notes string) error {
	return nil
}

func (m *MockWorkflowTaskStorage) UpdateTodoStatus(agentTaskID, todoID string, status storage.TodoStatus, notes string) error {
	return nil
}

func (m *MockWorkflowTaskStorage) AddTaskPromptNotes(agentTaskID string, notes string) error {
	return nil
}

func (m *MockWorkflowTaskStorage) UpdateTaskPromptNotes(agentTaskID string, notes string) error {
	return nil
}

func (m *MockWorkflowTaskStorage) ClearTaskPromptNotes(agentTaskID string) error {
	return nil
}

func (m *MockWorkflowTaskStorage) AddTodoPromptNotes(agentTaskID string, todoID string, notes string) error {
	return nil
}

func (m *MockWorkflowTaskStorage) UpdateTodoPromptNotes(agentTaskID string, todoID string, notes string) error {
	return nil
}

func (m *MockWorkflowTaskStorage) ClearTodoPromptNotes(agentTaskID string, todoID string) error {
	return nil
}

func (m *MockWorkflowTaskStorage) ClearAllTasks() (*storage.ClearResult, error) {
	return nil, nil
}

func TestWorkflowResourceHandler_ActiveAgents(t *testing.T) {
	now := time.Now().UTC()

	// Create mock storage with sample tasks
	mockStorage := &MockWorkflowTaskStorage{
		agentTasks: []*storage.AgentTask{
			{
				ID:          "task-1",
				AgentName:   "go-mcp-dev",
				Role:        "Implement MCP tools",
				Status:      storage.TaskStatusInProgress,
				CreatedAt:   now.Add(-2 * time.Hour),
				UpdatedAt:   now.Add(-10 * time.Minute),
				Todos:       []storage.TodoItem{{ID: "todo-1", Status: storage.TodoStatusInProgress}},
				HumanTaskID: "human-1",
			},
			{
				ID:          "task-2",
				AgentName:   "ui-dev",
				Role:        "Build task board",
				Status:      storage.TaskStatusPending,
				CreatedAt:   now.Add(-1 * time.Hour),
				UpdatedAt:   now.Add(-1 * time.Hour),
				Todos:       []storage.TodoItem{{ID: "todo-2", Status: storage.TodoStatusPending}},
				HumanTaskID: "human-1",
			},
			{
				ID:          "task-3",
				AgentName:   "go-mcp-dev",
				Role:        "Add validation",
				Status:      storage.TaskStatusBlocked,
				CreatedAt:   now.Add(-30 * time.Minute),
				UpdatedAt:   now.Add(-5 * time.Minute),
				Todos:       []storage.TodoItem{{ID: "todo-3", Status: storage.TodoStatusPending}},
				HumanTaskID: "human-1",
			},
		},
	}

	// Create handler
	handler := NewWorkflowResourceHandler(mockStorage)

	// Create MCP server and register resources
	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	opts := &mcp.ServerOptions{
		HasResources: true,
	}
	server := mcp.NewServer(impl, opts)

	err := handler.RegisterWorkflowResources(server)
	require.NoError(t, err)

	// Test active-agents resource directly
	ctx := context.Background()
	req := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "hyperion://workflow/active-agents",
		},
	}

	result, err := handler.handleActiveAgents(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Contents, 1)

	// Parse JSON response
	var response map[string]interface{}
	err = json.Unmarshal([]byte(result.Contents[0].Text), &response)
	require.NoError(t, err)

	// Verify response structure
	assert.Contains(t, response, "agents")
	assert.Contains(t, response, "totalCount")
	assert.Contains(t, response, "timestamp")

	agents := response["agents"].([]interface{})
	assert.Len(t, agents, 2) // go-mcp-dev and ui-dev

	// Verify agent status computation
	for _, agentData := range agents {
		agent := agentData.(map[string]interface{})
		agentName := agent["agentName"].(string)

		if agentName == "go-mcp-dev" {
			// Should be "blocked" because most recent task is blocked
			assert.Equal(t, "blocked", agent["status"])
			assert.Equal(t, float64(2), agent["taskCount"])
			assert.Equal(t, float64(1), agent["blockedCount"])
		} else if agentName == "ui-dev" {
			// Should be "idle" because task is pending
			assert.Equal(t, "idle", agent["status"])
			assert.Equal(t, float64(1), agent["taskCount"])
		}
	}
}

func TestWorkflowResourceHandler_TaskQueue(t *testing.T) {
	now := time.Now().UTC()

	// Create mock storage with pending tasks
	mockStorage := &MockWorkflowTaskStorage{
		agentTasks: []*storage.AgentTask{
			{
				ID:                "task-1",
				AgentName:         "go-mcp-dev",
				Role:              "High priority task",
				Status:            storage.TaskStatusPending,
				CreatedAt:         now.Add(-2 * time.Hour),
				ContextSummary:    "Complete context provided",
				FilesModified:     []string{"file1.go", "file2.go"},
				PriorWorkSummary:  "Prior work completed",
				Todos:             []storage.TodoItem{{ID: "1"}, {ID: "2"}, {ID: "3"}},
				HumanTaskID:       "human-1",
			},
			{
				ID:          "task-2",
				AgentName:   "ui-dev",
				Role:        "Low priority task",
				Status:      storage.TaskStatusPending,
				CreatedAt:   now.Add(-30 * time.Minute),
				Todos:       []storage.TodoItem{{ID: "1"}},
				HumanTaskID: "human-1",
			},
			{
				ID:          "task-3",
				AgentName:   "backend-services",
				Role:        "Completed task",
				Status:      storage.TaskStatusCompleted,
				CreatedAt:   now.Add(-3 * time.Hour),
				Todos:       []storage.TodoItem{{ID: "1"}},
				HumanTaskID: "human-1",
			},
		},
	}

	handler := NewWorkflowResourceHandler(mockStorage)

	ctx := context.Background()
	req := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "hyperion://workflow/task-queue",
		},
	}

	result, err := handler.handleTaskQueue(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result.Contents[0].Text), &response)
	require.NoError(t, err)

	queue := response["queue"].([]interface{})
	assert.Len(t, queue, 2) // Only pending tasks

	// Verify first task has higher priority
	firstTask := queue[0].(map[string]interface{})
	secondTask := queue[1].(map[string]interface{})

	assert.Greater(t, firstTask["priority"].(float64), secondTask["priority"].(float64))
	assert.Equal(t, "High priority task", firstTask["role"])
}

func TestWorkflowResourceHandler_Dependencies(t *testing.T) {
	now := time.Now().UTC()

	mockStorage := &MockWorkflowTaskStorage{
		agentTasks: []*storage.AgentTask{
			{
				ID:               "task-1",
				AgentName:        "go-mcp-dev",
				Role:             "Foundation task",
				Status:           storage.TaskStatusCompleted,
				CreatedAt:        now.Add(-2 * time.Hour),
				Notes:            "Completed successfully",
				PriorWorkSummary: "",
				HumanTaskID:      "human-1",
			},
			{
				ID:               "task-2",
				AgentName:        "ui-dev",
				Role:             "Depends on task-1",
				Status:           storage.TaskStatusPending,
				CreatedAt:        now.Add(-1 * time.Hour),
				Notes:            "Waiting for task-1 completion",
				PriorWorkSummary: "Task task-1 provides the API",
				HumanTaskID:      "human-1",
			},
		},
	}

	handler := NewWorkflowResourceHandler(mockStorage)

	ctx := context.Background()
	req := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "hyperion://workflow/dependencies",
		},
	}

	result, err := handler.handleDependencies(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result.Contents[0].Text), &response)
	require.NoError(t, err)

	dependencies := response["dependencies"].([]interface{})
	assert.Len(t, dependencies, 2)

	// Verify dependency structure
	for _, depData := range dependencies {
		dep := depData.(map[string]interface{})
		assert.Contains(t, dep, "taskId")
		assert.Contains(t, dep, "agentName")
		assert.Contains(t, dep, "status")
	}
}
