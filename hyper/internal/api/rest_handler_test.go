package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hyper/internal/mcp/scanner"
	"hyper/internal/mcp/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Mock implementations
type MockTaskStorage struct {
	mock.Mock
}

func (m *MockTaskStorage) CreateHumanTask(prompt string) (*storage.HumanTask, error) {
	args := m.Called(prompt)
	if args.Get(0) != nil {
		return args.Get(0).(*storage.HumanTask), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTaskStorage) ListAllHumanTasks() []*storage.HumanTask {
	args := m.Called()
	return args.Get(0).([]*storage.HumanTask)
}

func (m *MockTaskStorage) GetHumanTask(id string) (*storage.HumanTask, error) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*storage.HumanTask), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTaskStorage) UpdateTaskStatus(taskID string, status storage.TaskStatus, notes string) error {
	args := m.Called(taskID, status, notes)
	return args.Error(0)
}

func (m *MockTaskStorage) CreateAgentTask(humanTaskID, agentName, role string, todos []storage.TodoItemInput, contextSummary string, filesModified, qdrantCollections []string, priorWorkSummary string) (*storage.AgentTask, error) {
	args := m.Called(humanTaskID, agentName, role, todos, contextSummary, filesModified, qdrantCollections, priorWorkSummary)
	if args.Get(0) != nil {
		return args.Get(0).(*storage.AgentTask), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTaskStorage) ListAllAgentTasks() []*storage.AgentTask {
	args := m.Called()
	return args.Get(0).([]*storage.AgentTask)
}

func (m *MockTaskStorage) GetAgentTask(id string) (*storage.AgentTask, error) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*storage.AgentTask), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTaskStorage) UpdateTodoStatus(agentTaskID, todoID string, status storage.TodoStatus, notes string) error {
	args := m.Called(agentTaskID, todoID, status, notes)
	return args.Error(0)
}

func (m *MockTaskStorage) ClearAllTasks() (*storage.ClearResult, error) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*storage.ClearResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTaskStorage) AddTaskPromptNotes(agentTaskID, notes string) error {
	args := m.Called(agentTaskID, notes)
	return args.Error(0)
}

func (m *MockTaskStorage) UpdateTaskPromptNotes(agentTaskID, notes string) error {
	args := m.Called(agentTaskID, notes)
	return args.Error(0)
}

func (m *MockTaskStorage) ClearTaskPromptNotes(agentTaskID string) error {
	args := m.Called(agentTaskID)
	return args.Error(0)
}

func (m *MockTaskStorage) AddTodoPromptNotes(agentTaskID, todoID, notes string) error {
	args := m.Called(agentTaskID, todoID, notes)
	return args.Error(0)
}

func (m *MockTaskStorage) UpdateTodoPromptNotes(agentTaskID, todoID, notes string) error {
	args := m.Called(agentTaskID, todoID, notes)
	return args.Error(0)
}

func (m *MockTaskStorage) ClearTodoPromptNotes(agentTaskID, todoID string) error {
	args := m.Called(agentTaskID, todoID)
	return args.Error(0)
}

type MockKnowledgeStorage struct {
	mock.Mock
}

func (m *MockKnowledgeStorage) GetCollectionStatsWithMetadata() ([]*storage.CollectionWithMetadata, error) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).([]*storage.CollectionWithMetadata), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockKnowledgeStorage) ListKnowledge(collection string, limit int) ([]*storage.KnowledgeEntry, error) {
	args := m.Called(collection, limit)
	if args.Get(0) != nil {
		return args.Get(0).([]*storage.KnowledgeEntry), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockKnowledgeStorage) Upsert(collection, text string, metadata map[string]interface{}) (*storage.KnowledgeEntry, error) {
	args := m.Called(collection, text, metadata)
	if args.Get(0) != nil {
		return args.Get(0).(*storage.KnowledgeEntry), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockKnowledgeStorage) Query(collection, query string, limit int) ([]*storage.QueryResult, error) {
	args := m.Called(collection, query, limit)
	if args.Get(0) != nil {
		return args.Get(0).([]*storage.QueryResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockKnowledgeStorage) GetPopularCollections(limit int) ([]*storage.CollectionStats, error) {
	args := m.Called(limit)
	if args.Get(0) != nil {
		return args.Get(0).([]*storage.CollectionStats), args.Error(1)
	}
	return nil, args.Error(1)
}

// Setup helper
func setupTestHandler() (*RESTAPIHandler, *MockTaskStorage, *MockKnowledgeStorage) {
	logger, _ := zap.NewDevelopment()
	mockTaskStorage := new(MockTaskStorage)
	mockKnowledgeStorage := new(MockKnowledgeStorage)

	handler := &RESTAPIHandler{
		taskStorage:      mockTaskStorage,
		knowledgeStorage: mockKnowledgeStorage,
		fileScanner:      scanner.NewFileScanner(),
		logger:           logger,
	}

	return handler, mockTaskStorage, mockKnowledgeStorage
}

// Test CreateHumanTask
func TestCreateHumanTask(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockTaskStorage)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "success - valid task creation",
			requestBody: CreateHumanTaskRequest{
				Prompt: "Test task prompt",
			},
			mockSetup: func(m *MockTaskStorage) {
				task := &storage.HumanTask{
					ID:     "task-123",
					Prompt: "Test task prompt",
					Status: storage.TaskStatusPending,
				}
				m.On("CreateHumanTask", "Test task prompt").Return(task, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "error - missing prompt",
			requestBody:    map[string]string{},
			mockSetup:      func(m *MockTaskStorage) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request",
		},
		{
			name: "error - storage failure",
			requestBody: CreateHumanTaskRequest{
				Prompt: "Test prompt",
			},
			mockSetup: func(m *MockTaskStorage) {
				m.On("CreateHumanTask", "Test prompt").Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to create task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockTaskStorage, _ := setupTestHandler()
			tt.mockSetup(mockTaskStorage)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/api/v1/tasks", handler.CreateHumanTask)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]string
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Contains(t, response["error"], tt.expectedError)
			}

			mockTaskStorage.AssertExpectations(t)
		})
	}
}

// Test ListHumanTasks
func TestListHumanTasks(t *testing.T) {
	handler, mockTaskStorage, _ := setupTestHandler()

	mockTaskStorage.On("ListAllHumanTasks").Return([]*storage.HumanTask{
		{ID: "task-1", Prompt: "Task 1"},
		{ID: "task-2", Prompt: "Task 2"},
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/tasks", handler.ListHumanTasks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ListHumanTasksResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, 2, response.Count)
	assert.Len(t, response.Tasks, 2)

	mockTaskStorage.AssertExpectations(t)
}

// Test UpdateTaskStatus
func TestUpdateTaskStatus(t *testing.T) {
	handler, mockTaskStorage, _ := setupTestHandler()

	mockTaskStorage.On("UpdateTaskStatus", "task-123", storage.TaskStatusInProgress, "Working on it").Return(nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/api/v1/tasks/:id/status", handler.UpdateTaskStatus)

	body := UpdateTaskStatusRequest{
		Status: "in_progress",
		Notes:  "Working on it",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/task-123/status", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response UpdateTaskStatusResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response.Success)

	mockTaskStorage.AssertExpectations(t)
}

// Test ListCollections
func TestListCollections(t *testing.T) {
	handler, _, mockKnowledgeStorage := setupTestHandler()

	mockKnowledgeStorage.On("GetCollectionStatsWithMetadata").Return([]*storage.CollectionWithMetadata{
		{CollectionName: "technical-knowledge", Category: "technical", EntryCount: 10},
		{CollectionName: "adr", Category: "architecture", EntryCount: 5},
	}, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/knowledge/collections", handler.ListCollections)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/collections", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ListCollectionsResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Len(t, response.Collections, 2)
	assert.Equal(t, "technical-knowledge", response.Collections[0].Name)

	mockKnowledgeStorage.AssertExpectations(t)
}

// Test RegisterRESTRoutes
func TestRegisterRESTRoutes(t *testing.T) {
	handler, _, _ := setupTestHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRESTRoutes(router)

	// Test that routes are registered
	routes := router.Routes()
	assert.Greater(t, len(routes), 10, "Should have registered multiple routes")

	// Check for specific route patterns
	foundTasksRoute := false
	for _, route := range routes {
		if route.Path == "/api/v1/tasks" {
			foundTasksRoute = true
			break
		}
	}
	assert.True(t, foundTasksRoute, "Should register /api/v1/tasks route")
}
