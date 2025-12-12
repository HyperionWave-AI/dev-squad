package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hyper/internal/mcp/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test Agent Communication Endpoints
func TestAgentCommunicate(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockTaskStorage)
		expectedStatus int
		expectedError  string
		validateResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success - status communication",
			requestBody: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "status",
			},
			mockSetup:      func(m *MockTaskStorage) {},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response AgentCommunicationResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.Equal(t, "go-dev", response.AgentType)
				assert.NotEmpty(t, response.Timestamp)
			},
		},
		{
			name: "success - status with task ID",
			requestBody: AgentCommunicationRequest{
				AgentType:         "ui-dev",
				CommunicationType: "status",
				TaskID:            "task-123",
			},
			mockSetup: func(m *MockTaskStorage) {
				task := &storage.AgentTask{
					ID:        "task-123",
					AgentName: "ui-dev",
					Status:    storage.TaskStatusInProgress,
				}
				m.On("GetAgentTask", "task-123").Return(task, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response AgentCommunicationResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.Equal(t, "ui-dev", response.AgentType)
			},
		},
		{
			name: "success - execute communication with create_agent_task",
			requestBody: AgentCommunicationRequest{
				AgentType:         "coordinator",
				CommunicationType: "execute",
				Parameters: map[string]interface{}{
					"command":      "create_agent_task",
					"humanTaskId":  "human-123",
					"role":         "Backend Developer",
					"todos":        []storage.TodoItemInput{},
				},
			},
			mockSetup: func(m *MockTaskStorage) {
				task := &storage.AgentTask{
					ID:          "agent-123",
					HumanTaskID: "human-123",
					AgentName:   "coordinator",
					Role:        "Backend Developer",
					Status:      storage.TaskStatusPending,
				}
				m.On("CreateAgentTask", "human-123", "coordinator", "Backend Developer", 
					mock.AnythingOfType("[]storage.TodoItemInput"), "", 
					mock.AnythingOfType("[]string"), mock.AnythingOfType("[]string"), "").Return(task, nil)
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response AgentCommunicationResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.Equal(t, "coordinator", response.AgentType)
				assert.Contains(t, response.Message, "created successfully")
			},
		},
		{
			name: "success - direct message communication",
			requestBody: AgentCommunicationRequest{
				AgentType:         "sre",
				CommunicationType: "direct_message",
				Message:           "Hello from another agent",
			},
			mockSetup:      func(m *MockTaskStorage) {},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response AgentCommunicationResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
				assert.Equal(t, "sre", response.AgentType)
				assert.Contains(t, response.Message, "received")
			},
		},
		{
			name:           "error - invalid agent type",
			requestBody: AgentCommunicationRequest{
				AgentType:         "invalid-agent",
				CommunicationType: "status",
			},
			mockSetup:      func(m *MockTaskStorage) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid agent type",
		},
		{
			name: "error - invalid communication type",
			requestBody: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "invalid-comm",
			},
			mockSetup:      func(m *MockTaskStorage) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid communication type",
		},
		{
			name: "error - execute without command",
			requestBody: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "execute",
			},
			mockSetup:      func(m *MockTaskStorage) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "execute communication requires",
		},
		{
			name: "error - direct message without message",
			requestBody: AgentCommunicationRequest{
				AgentType:         "ui-dev",
				CommunicationType: "direct_message",
			},
			mockSetup:      func(m *MockTaskStorage) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "direct_message communication requires message content",
		},
		{
			name: "error - message too long",
			requestBody: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "direct_message",
				Message:           string(make([]byte, 10001)), // Exceeds 10000 char limit
			},
			mockSetup:      func(m *MockTaskStorage) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "message length exceeds maximum",
		},
		{
			name: "error - unsupported command",
			requestBody: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "execute",
				Parameters: map[string]interface{}{
					"command": "unsupported_command",
				},
			},
			mockSetup:      func(m *MockTaskStorage) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Unsupported command",
		},
		{
			name: "error - task not found for status",
			requestBody: AgentCommunicationRequest{
				AgentType:         "go-dev",
				CommunicationType: "status",
				TaskID:            "nonexistent-task",
			},
			mockSetup: func(m *MockTaskStorage) {
				m.On("GetAgentTask", "nonexistent-task").Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "task not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockTaskStorage, _ := setupTestHandler()
			tt.mockSetup(mockTaskStorage)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/api/v1/agents/communicate", handler.AgentCommunicate)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/communicate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response AgentCommunicationResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.False(t, response.Success)
				assert.Contains(t, response.Message, tt.expectedError)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, w)
			}

			mockTaskStorage.AssertExpectations(t)
		})
	}
}

// Test Agent Type Validation
func TestValidateAgentType(t *testing.T) {
	handler, _, _ := setupTestHandler()

	validTypes := []string{"ui-dev", "go-dev", "sre", "coordinator", "data-analyst", "qa"}
	invalidTypes := []string{"invalid", "unknown-agent", "", "UI-DEV", "Go-Dev"}

	for _, validType := range validTypes {
		t.Run("valid_"+validType, func(t *testing.T) {
			err := handler.validateAgentType(validType)
			assert.NoError(t, err)
		})
	}

	for _, invalidType := range invalidTypes {
		t.Run("invalid_"+invalidType, func(t *testing.T) {
			err := handler.validateAgentType(invalidType)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid agent type")
		})
	}
}

// Test Communication Type Validation
func TestValidateCommunicationType(t *testing.T) {
	handler, _, _ := setupTestHandler()

	validTypes := []string{"execute", "status", "direct_message"}
	invalidTypes := []string{"invalid", "unknown-comm", "", "EXECUTE", "Status"}

	for _, validType := range validTypes {
		t.Run("valid_"+validType, func(t *testing.T) {
			err := handler.validateCommunicationType(validType)
			assert.NoError(t, err)
		})
	}

	for _, invalidType := range invalidTypes {
		t.Run("invalid_"+invalidType, func(t *testing.T) {
			err := handler.validateCommunicationType(invalidType)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid communication type")
		})
	}
}

// Test Request Validation
func TestValidateAndLogAgentRequest(t *testing.T) {
	handler, _, _ := setupTestHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		if handler.validateAndLogAgentRequest(c) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		}
	})

	tests := []struct {
		name           string
		method         string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "valid request",
			method:         http.MethodPost,
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid method",
			method:         http.MethodGet,
			contentType:    "application/json",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid content type",
			method:         http.MethodPost,
			contentType:    "text/plain",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "no content type (should pass)",
			method:         http.MethodPost,
			contentType:    "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Test Execute Request Validation
func TestValidateExecuteRequest(t *testing.T) {
	handler, _, _ := setupTestHandler()

	tests := []struct {
		name        string
		request     AgentCommunicationRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid with message",
			request: AgentCommunicationRequest{
				Message: "test command",
			},
			expectError: false,
		},
		{
			name: "valid with parameters command",
			request: AgentCommunicationRequest{
				Parameters: map[string]interface{}{
					"command": "create_agent_task",
				},
			},
			expectError: false,
		},
		{
			name:        "invalid - no message or parameters",
			request:     AgentCommunicationRequest{},
			expectError: true,
			errorMsg:    "execute communication requires",
		},
		{
			name: "invalid - empty command in parameters",
			request: AgentCommunicationRequest{
				Parameters: map[string]interface{}{
					"command": "",
				},
			},
			expectError: true,
			errorMsg:    "must be a non-empty string",
		},
		{
			name: "invalid - message too long",
			request: AgentCommunicationRequest{
				Message: string(make([]byte, 10001)),
			},
			expectError: true,
			errorMsg:    "message length exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateExecuteRequest(tt.request)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test Direct Message Request Validation
func TestValidateDirectMessageRequest(t *testing.T) {
	handler, _, _ := setupTestHandler()

	tests := []struct {
		name        string
		request     AgentCommunicationRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid with message",
			request: AgentCommunicationRequest{
				Message: "Hello from agent",
			},
			expectError: false,
		},
		{
			name: "valid with message in parameters",
			request: AgentCommunicationRequest{
				Parameters: map[string]interface{}{
					"message": "Hello from parameters",
				},
			},
			expectError: false,
		},
		{
			name:        "invalid - no message",
			request:     AgentCommunicationRequest{},
			expectError: true,
			errorMsg:    "direct_message communication requires message content",
		},
		{
			name: "invalid - empty message in parameters",
			request: AgentCommunicationRequest{
				Parameters: map[string]interface{}{
					"message": "",
				},
			},
			expectError: true,
			errorMsg:    "direct_message communication requires message content",
		},
		{
			name: "invalid - message too long",
			request: AgentCommunicationRequest{
				Message: string(make([]byte, 10001)),
			},
			expectError: true,
			errorMsg:    "message length exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateDirectMessageRequest(tt.request)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}