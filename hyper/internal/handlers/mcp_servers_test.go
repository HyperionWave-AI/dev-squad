package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockToolsStorage is a mock implementation of ToolsStorageInterface for testing
type MockToolsStorage struct {
	mock.Mock
}

func (m *MockToolsStorage) StoreToolMetadata(ctx context.Context, toolName, description string, schema map[string]interface{}, serverName string) error {
	args := m.Called(ctx, toolName, description, schema, serverName)
	return args.Error(0)
}

func (m *MockToolsStorage) SearchTools(ctx context.Context, query string, limit int) ([]*storage.ToolMatch, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ToolMatch), args.Error(1)
}

func (m *MockToolsStorage) GetToolSchema(ctx context.Context, toolName string) (*storage.ToolMetadata, error) {
	args := m.Called(ctx, toolName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ToolMetadata), args.Error(1)
}

func (m *MockToolsStorage) StoreResourceMetadata(ctx context.Context, uri, name, description, mimeType, serverName string) error {
	args := m.Called(ctx, uri, name, description, mimeType, serverName)
	return args.Error(0)
}

func (m *MockToolsStorage) GetServerResources(ctx context.Context, serverName string) ([]*storage.ResourceMetadata, error) {
	args := m.Called(ctx, serverName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ResourceMetadata), args.Error(1)
}

func (m *MockToolsStorage) StorePromptMetadata(ctx context.Context, name, description string, arguments []map[string]interface{}, serverName string) error {
	args := m.Called(ctx, name, description, arguments, serverName)
	return args.Error(0)
}

func (m *MockToolsStorage) GetServerPrompts(ctx context.Context, serverName string) ([]*storage.PromptMetadata, error) {
	args := m.Called(ctx, serverName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.PromptMetadata), args.Error(1)
}

func (m *MockToolsStorage) AddServer(ctx context.Context, serverName, serverURL, description string, headers map[string]interface{}) error {
	args := m.Called(ctx, serverName, serverURL, description, headers)
	return args.Error(0)
}

func (m *MockToolsStorage) UpdateServer(ctx context.Context, serverName, serverURL, description string, headers map[string]interface{}) error {
	args := m.Called(ctx, serverName, serverURL, description, headers)
	return args.Error(0)
}

func (m *MockToolsStorage) UpdateServerCounts(ctx context.Context, serverName string, toolCount, resourceCount, promptCount int) error {
	args := m.Called(ctx, serverName, toolCount, resourceCount, promptCount)
	return args.Error(0)
}

func (m *MockToolsStorage) RemoveServer(ctx context.Context, serverName string) error {
	args := m.Called(ctx, serverName)
	return args.Error(0)
}

func (m *MockToolsStorage) GetServer(ctx context.Context, serverName string) (*storage.ServerMetadata, error) {
	args := m.Called(ctx, serverName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ServerMetadata), args.Error(1)
}

func (m *MockToolsStorage) GetServerTools(ctx context.Context, serverName string) ([]*storage.ToolMetadata, error) {
	args := m.Called(ctx, serverName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ToolMetadata), args.Error(1)
}

func (m *MockToolsStorage) ListServers(ctx context.Context) ([]*storage.ServerMetadata, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.ServerMetadata), args.Error(1)
}

func (m *MockToolsStorage) RemoveServerTools(ctx context.Context, serverName string) error {
	args := m.Called(ctx, serverName)
	return args.Error(0)
}

func (m *MockToolsStorage) RemoveServerResources(ctx context.Context, serverName string) error {
	args := m.Called(ctx, serverName)
	return args.Error(0)
}

func (m *MockToolsStorage) RemoveServerPrompts(ctx context.Context, serverName string) error {
	args := m.Called(ctx, serverName)
	return args.Error(0)
}

// setupMCPServersTestHandler creates a test handler with mocked dependencies
func setupMCPServersTestHandler() (*MCPServersHandler, *MockToolsStorage) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	mockStorage := new(MockToolsStorage)
	handler := NewMCPServersHandler(mockStorage, nil, logger)
	return handler, mockStorage
}

// TestAddMCPServer_NewServer tests adding a new MCP server
func TestAddMCPServer_NewServer(t *testing.T) {
	handler, mockStorage := setupMCPServersTestHandler()

	// Mock GetServer to return error (server doesn't exist)
	mockStorage.On("GetServer", mock.Anything, "test-server").Return(nil, assert.AnError)
	// Mock AddServer to succeed
	mockStorage.On("AddServer", mock.Anything, "test-server", "http://localhost:3000", "Test server", mock.Anything).Return(nil)

	// Create test request
	reqBody := AddMCPServerRequest{
		ServerName:  "test-server",
		ServerURL:   "http://localhost:3000",
		Description: "Test server",
		Headers:     map[string]interface{}{"Authorization": "Bearer token"},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/mcp/servers", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute handler
	handler.AddMCPServer(c)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response AddMCPServerResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "added successfully")

	// Verify mock expectations
	mockStorage.AssertExpectations(t)
}

// TestAddMCPServer_ReplaceExistingServer tests replacing an existing MCP server
func TestAddMCPServer_ReplaceExistingServer(t *testing.T) {
	handler, mockStorage := setupMCPServersTestHandler()

	// Mock GetServer to return existing server
	existingServer := &storage.ServerMetadata{
		ServerName:    "test-server",
		ServerURL:     "http://old-url:3000",
		Description:   "Old description",
		ToolCount:     5,
		ResourceCount: 3,
		PromptCount:   2,
		CreatedAt:     time.Now().Add(-24 * time.Hour),
		UpdatedAt:     time.Now().Add(-1 * time.Hour),
	}
	mockStorage.On("GetServer", mock.Anything, "test-server").Return(existingServer, nil)

	// Mock removal of old server and associated data
	mockStorage.On("RemoveServer", mock.Anything, "test-server").Return(nil)
	mockStorage.On("RemoveServerTools", mock.Anything, "test-server").Return(nil)
	mockStorage.On("RemoveServerResources", mock.Anything, "test-server").Return(nil)
	mockStorage.On("RemoveServerPrompts", mock.Anything, "test-server").Return(nil)

	// Mock AddServer to succeed with new configuration
	mockStorage.On("AddServer", mock.Anything, "test-server", "http://new-url:4000", "New description", mock.Anything).Return(nil)

	// Create test request with new server configuration
	reqBody := AddMCPServerRequest{
		ServerName:  "test-server",
		ServerURL:   "http://new-url:4000",
		Description: "New description",
		Headers:     map[string]interface{}{"Authorization": "Bearer new-token"},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/mcp/servers", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute handler
	handler.AddMCPServer(c)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response AddMCPServerResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "updated successfully")

	// Verify all cleanup methods were called
	mockStorage.AssertCalled(t, "RemoveServer", mock.Anything, "test-server")
	mockStorage.AssertCalled(t, "RemoveServerTools", mock.Anything, "test-server")
	mockStorage.AssertCalled(t, "RemoveServerResources", mock.Anything, "test-server")
	mockStorage.AssertCalled(t, "RemoveServerPrompts", mock.Anything, "test-server")

	// Verify new server was added
	mockStorage.AssertCalled(t, "AddServer", mock.Anything, "test-server", "http://new-url:4000", "New description", mock.Anything)

	mockStorage.AssertExpectations(t)
}

// TestAddMCPServer_InvalidServerName tests validation of server name
func TestAddMCPServer_InvalidServerName(t *testing.T) {
	handler, _ := setupMCPServersTestHandler()

	testCases := []struct {
		name       string
		serverName string
		errorMsg   string
	}{
		{"with spaces", "invalid server", "Invalid server name"},
		{"with special chars", "server@name", "Invalid server name"},
		{"with dots", "server.name", "Invalid server name"},
		{"empty name", "", "Invalid request body"}, // Caught by JSON binding validation
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := AddMCPServerRequest{
				ServerName:  tc.serverName,
				ServerURL:   "http://localhost:3000",
				Description: "Test",
			}
			bodyBytes, _ := json.Marshal(reqBody)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/v1/mcp/servers", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.AddMCPServer(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["error"], tc.errorMsg)
		})
	}
}

// TestAddMCPServer_ValidServerNames tests valid server name patterns
func TestAddMCPServer_ValidServerNames(t *testing.T) {
	testCases := []string{
		"simple-server",
		"server_name",
		"ServerName123",
		"server-123_test",
	}

	for _, serverName := range testCases {
		t.Run(serverName, func(t *testing.T) {
			handler, mockStorage := setupMCPServersTestHandler()

			mockStorage.On("GetServer", mock.Anything, serverName).Return(nil, assert.AnError)
			mockStorage.On("AddServer", mock.Anything, serverName, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			reqBody := AddMCPServerRequest{
				ServerName:  serverName,
				ServerURL:   "http://localhost:3000",
				Description: "Test",
			}
			bodyBytes, _ := json.Marshal(reqBody)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/api/v1/mcp/servers", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.AddMCPServer(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockStorage.AssertExpectations(t)
		})
	}
}

// TestAddMCPServer_ReplacementFailureOnRemove tests handling of removal failure during replacement
func TestAddMCPServer_ReplacementFailureOnRemove(t *testing.T) {
	handler, mockStorage := setupMCPServersTestHandler()

	// Mock GetServer to return existing server
	existingServer := &storage.ServerMetadata{
		ServerName: "test-server",
		ServerURL:  "http://old-url:3000",
	}
	mockStorage.On("GetServer", mock.Anything, "test-server").Return(existingServer, nil)

	// Mock RemoveServer to fail
	mockStorage.On("RemoveServer", mock.Anything, "test-server").Return(assert.AnError)

	// Create test request
	reqBody := AddMCPServerRequest{
		ServerName:  "test-server",
		ServerURL:   "http://new-url:4000",
		Description: "New description",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/mcp/servers", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute handler
	handler.AddMCPServer(c)

	// Verify error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Failed to replace existing server")

	mockStorage.AssertExpectations(t)
}

// TestListMCPServers tests listing all MCP servers
func TestListMCPServers(t *testing.T) {
	handler, mockStorage := setupMCPServersTestHandler()

	// Mock ListServers to return test data
	servers := []*storage.ServerMetadata{
		{
			ServerName:    "server-1",
			ServerURL:     "http://localhost:3000",
			Description:   "Server 1",
			ToolCount:     5,
			ResourceCount: 3,
			PromptCount:   2,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ServerName:    "server-2",
			ServerURL:     "http://localhost:4000",
			Description:   "Server 2",
			ToolCount:     10,
			ResourceCount: 5,
			PromptCount:   3,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}
	mockStorage.On("ListServers", mock.Anything).Return(servers, nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/mcp/servers", nil)

	// Execute handler
	handler.ListMCPServers(c)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response ListMCPServersResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 2, response.Total)
	assert.Len(t, response.Servers, 2)
	assert.Equal(t, "server-1", response.Servers[0].ServerName)
	assert.Equal(t, "server-2", response.Servers[1].ServerName)

	mockStorage.AssertExpectations(t)
}

// TestRemoveMCPServer tests removing an MCP server
func TestRemoveMCPServer(t *testing.T) {
	handler, mockStorage := setupMCPServersTestHandler()

	// Mock RemoveServer to succeed
	mockStorage.On("RemoveServer", mock.Anything, "test-server").Return(nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/api/v1/mcp/servers/test-server", nil)
	c.Params = []gin.Param{{Key: "serverName", Value: "test-server"}}

	// Execute handler
	handler.RemoveMCPServer(c)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Contains(t, response["message"], "removed successfully")

	mockStorage.AssertExpectations(t)
}
