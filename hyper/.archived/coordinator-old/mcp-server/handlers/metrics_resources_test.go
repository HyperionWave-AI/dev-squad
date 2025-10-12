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

// MockTaskStorage implements storage.TaskStorage for testing
type MockMetricsTaskStorage struct {
	tasks []*storage.AgentTask
}

func (m *MockMetricsTaskStorage) CreateHumanTask(prompt string) (*storage.HumanTask, error) {
	return nil, nil
}

func (m *MockMetricsTaskStorage) CreateAgentTask(humanTaskID, agentName, role string, todos []storage.TodoItemInput, contextSummary string, filesModified []string, qdrantCollections []string, priorWorkSummary string) (*storage.AgentTask, error) {
	return nil, nil
}

func (m *MockMetricsTaskStorage) GetHumanTask(taskID string) (*storage.HumanTask, error) {
	return nil, nil
}

func (m *MockMetricsTaskStorage) GetAgentTask(taskID string) (*storage.AgentTask, error) {
	return nil, nil
}

func (m *MockMetricsTaskStorage) GetAgentTasksByName(agentName string) ([]*storage.AgentTask, error) {
	var result []*storage.AgentTask
	for _, task := range m.tasks {
		if task.AgentName == agentName {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *MockMetricsTaskStorage) ListAllHumanTasks() []*storage.HumanTask {
	return []*storage.HumanTask{}
}

func (m *MockMetricsTaskStorage) ListAllAgentTasks() []*storage.AgentTask {
	return m.tasks
}

func (m *MockMetricsTaskStorage) UpdateTaskStatus(taskID string, status storage.TaskStatus, notes string) error {
	return nil
}

func (m *MockMetricsTaskStorage) UpdateTodoStatus(agentTaskID, todoID string, status storage.TodoStatus, notes string) error {
	return nil
}

func (m *MockMetricsTaskStorage) AddTaskPromptNotes(agentTaskID string, notes string) error {
	return nil
}

func (m *MockMetricsTaskStorage) UpdateTaskPromptNotes(agentTaskID string, notes string) error {
	return nil
}

func (m *MockMetricsTaskStorage) ClearTaskPromptNotes(agentTaskID string) error {
	return nil
}

func (m *MockMetricsTaskStorage) AddTodoPromptNotes(agentTaskID string, todoID string, notes string) error {
	return nil
}

func (m *MockMetricsTaskStorage) UpdateTodoPromptNotes(agentTaskID string, todoID string, notes string) error {
	return nil
}

func (m *MockMetricsTaskStorage) ClearTodoPromptNotes(agentTaskID string, todoID string) error {
	return nil
}

func (m *MockMetricsTaskStorage) ClearAllTasks() (*storage.ClearResult, error) {
	return nil, nil
}

func TestMetricsResourceHandler_SquadVelocity(t *testing.T) {
	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := todayStart.AddDate(0, 0, -7)

	completedAt := now.Add(-1 * time.Hour)

	// Create mock storage with test data
	mockStorage := &MockMetricsTaskStorage{
		tasks: []*storage.AgentTask{
			// Backend squad - completed today
			{
				ID:          "task1",
				HumanTaskID: "human1",
				AgentName:   "backend-services",
				Role:        "Implement API endpoint",
				Status:      storage.TaskStatusCompleted,
				CreatedAt:   todayStart.Add(1 * time.Hour),
				UpdatedAt:   completedAt,
				Todos: []storage.TodoItem{
					{
						ID:          "todo1",
						Description: "Create handler",
						Status:      storage.TodoStatusCompleted,
						CreatedAt:   todayStart.Add(1 * time.Hour),
						CompletedAt: &completedAt,
					},
				},
			},
			// Frontend squad - completed this week
			{
				ID:          "task2",
				HumanTaskID: "human1",
				AgentName:   "frontend-experience",
				Role:        "Update UI component",
				Status:      storage.TaskStatusCompleted,
				CreatedAt:   weekStart.Add(1 * time.Hour),
				UpdatedAt:   weekStart.Add(2 * time.Hour),
				Todos: []storage.TodoItem{
					{
						ID:          "todo2",
						Description: "Update component",
						Status:      storage.TodoStatusCompleted,
						CreatedAt:   weekStart.Add(1 * time.Hour),
						CompletedAt: &completedAt,
					},
				},
			},
			// Backend squad - in progress
			{
				ID:          "task3",
				HumanTaskID: "human2",
				AgentName:   "backend-services",
				Role:        "Add validation",
				Status:      storage.TaskStatusInProgress,
				CreatedAt:   todayStart.Add(30 * time.Minute),
				UpdatedAt:   todayStart.Add(45 * time.Minute),
				Todos: []storage.TodoItem{
					{
						ID:          "todo3",
						Description: "Add validator",
						Status:      storage.TodoStatusInProgress,
						CreatedAt:   todayStart.Add(30 * time.Minute),
					},
				},
			},
		},
	}

	handler := NewMetricsResourceHandler(mockStorage)

	// Create MCP server and register resources
	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	server := mcp.NewServer(impl, &mcp.ServerOptions{HasResources: true})

	err := handler.RegisterMetricsResources(server)
	require.NoError(t, err)

	// Test squad velocity resource
	ctx := context.Background()
	req := &mcp.ReadResourceRequest{}

	result, err := handler.handleSquadVelocity(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Contents, 1)

	// Parse JSON response
	var response map[string]interface{}
	err = json.Unmarshal([]byte(result.Contents[0].Text), &response)
	require.NoError(t, err)

	// Verify response structure
	assert.Contains(t, response, "squads")
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "windows")

	squads := response["squads"].([]interface{})
	assert.Len(t, squads, 2) // backend-services and frontend-experience

	// Find backend-services squad
	var backendSquad map[string]interface{}
	for _, s := range squads {
		squad := s.(map[string]interface{})
		if squad["squadName"] == "backend-services" {
			backendSquad = squad
			break
		}
	}

	require.NotNil(t, backendSquad)
	assert.Equal(t, float64(1), backendSquad["todayCompleted"])
	assert.Equal(t, float64(1), backendSquad["weekCompleted"])
	assert.Equal(t, float64(1), backendSquad["allTimeCompleted"])
	assert.Equal(t, float64(2), backendSquad["allTimeTaskCount"]) // 1 completed + 1 in progress
}

func TestMetricsResourceHandler_ContextEfficiency(t *testing.T) {
	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	mockStorage := &MockMetricsTaskStorage{
		tasks: []*storage.AgentTask{
			// Quick completion (1 hour)
			{
				ID:          "task1",
				AgentName:   "backend-services",
				Status:      storage.TaskStatusCompleted,
				CreatedAt:   todayStart.Add(1 * time.Hour),
				UpdatedAt:   todayStart.Add(2 * time.Hour),
				Todos:       []storage.TodoItem{{}, {}, {}}, // 3 TODOs
			},
			// Longer completion (4 hours)
			{
				ID:          "task2",
				AgentName:   "frontend-experience",
				Status:      storage.TaskStatusCompleted,
				CreatedAt:   todayStart.Add(1 * time.Hour),
				UpdatedAt:   todayStart.Add(5 * time.Hour),
				Todos:       []storage.TodoItem{{}, {}, {}, {}, {}, {}, {}}, // 7 TODOs (complex)
			},
			// In progress
			{
				ID:          "task3",
				AgentName:   "backend-services",
				Status:      storage.TaskStatusInProgress,
				CreatedAt:   todayStart.Add(6 * time.Hour),
				UpdatedAt:   now,
				Todos:       []storage.TodoItem{{}, {}}, // 2 TODOs (simple)
			},
		},
	}

	handler := NewMetricsResourceHandler(mockStorage)

	ctx := context.Background()
	req := &mcp.ReadResourceRequest{}

	result, err := handler.handleContextEfficiency(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Contents, 1)

	// Parse JSON response
	var response map[string]interface{}
	err = json.Unmarshal([]byte(result.Contents[0].Text), &response)
	require.NoError(t, err)

	// Verify response structure
	assert.Contains(t, response, "metrics")
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "analysis")

	metrics := response["metrics"].(map[string]interface{})

	// Check overall stats
	overallStats := metrics["overallStats"].(map[string]interface{})
	assert.Equal(t, float64(2), overallStats["totalTasksCompleted"])
	assert.Equal(t, float64(2), overallStats["activeSquads"])
	assert.Greater(t, overallStats["averageCompletionTime"], float64(0))

	// Check squad stats
	squadStats := metrics["squadStats"].([]interface{})
	assert.Len(t, squadStats, 2)

	// Check task complexity
	complexity := metrics["taskComplexity"].(map[string]interface{})
	assert.Contains(t, complexity, "averageTodoCount")
	assert.Contains(t, complexity, "todoDistribution")
	assert.Contains(t, complexity, "complexTasksPercent")
	assert.Contains(t, complexity, "simpleTasksPercent")

	// Verify analysis
	analysis := response["analysis"].(map[string]interface{})
	assert.Contains(t, analysis, "efficiency")
	assert.Contains(t, analysis, "trend")
}

func TestCalculateComplexityMetrics(t *testing.T) {
	tasks := []*storage.AgentTask{
		{Todos: []storage.TodoItem{{}, {}}},                       // Simple (2 TODOs)
		{Todos: []storage.TodoItem{{}, {}, {}}},                   // Simple (3 TODOs)
		{Todos: []storage.TodoItem{{}, {}, {}, {}}},               // Medium (4 TODOs)
		{Todos: []storage.TodoItem{{}, {}, {}, {}, {}}},           // Medium (5 TODOs)
		{Todos: []storage.TodoItem{{}, {}, {}, {}, {}, {}}},       // Complex (6 TODOs)
		{Todos: []storage.TodoItem{{}, {}, {}, {}, {}, {}, {}, {}}}, // Complex (8 TODOs)
	}

	handler := &MetricsResourceHandler{}
	metrics := handler.calculateComplexityMetrics(tasks)

	// Average: (2 + 3 + 4 + 5 + 6 + 8) / 6 = 4.67
	assert.InDelta(t, 4.67, metrics.AverageTodoCount, 0.1)

	// Distribution
	assert.Equal(t, 2, metrics.TodoDistribution["1-3"])
	assert.Equal(t, 2, metrics.TodoDistribution["4-5"])
	assert.Equal(t, 1, metrics.TodoDistribution["6-7"])
	assert.Equal(t, 1, metrics.TodoDistribution["8+"])

	// Complex (>5 TODOs): 2 out of 6 = 33.33%
	assert.InDelta(t, 33.33, metrics.ComplexTasksPercent, 0.1)

	// Simple (<=3 TODOs): 2 out of 6 = 33.33%
	assert.InDelta(t, 33.33, metrics.SimpleTasksPercent, 0.1)
}

func TestEfficiencyAnalysis(t *testing.T) {
	handler := &MetricsResourceHandler{}

	tests := []struct {
		score    float64
		expected string
	}{
		{95.0, "Excellent"},
		{85.0, "Good"},
		{70.0, "Fair"},
		{50.0, "Poor"},
		{30.0, "Critical"},
	}

	for _, tt := range tests {
		result := handler.getEfficiencyAnalysis(tt.score)
		assert.Contains(t, result, tt.expected)
	}
}
