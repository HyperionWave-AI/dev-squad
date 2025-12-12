package mcp

import (
	"context"
	"testing"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/mcp/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockToolsStorage is a mock implementation of storage.ToolsStorageInterface
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

func TestMCPRegistrySync_SyncAllMCPTools_NoServers(t *testing.T) {
	logger := zap.NewNop()
	registry := aiservice.NewToolRegistry()
	mockStorage := new(MockToolsStorage)

	// Mock: no servers registered
	mockStorage.On("ListServers", mock.Anything).Return([]*storage.ServerMetadata{}, nil)

	sync := NewMCPRegistrySync(registry, mockStorage, logger)
	count, err := sync.SyncAllMCPTools(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	mockStorage.AssertExpectations(t)
}

func TestMCPRegistrySync_SyncAllMCPTools_WithServers(t *testing.T) {
	logger := zap.NewNop()
	registry := aiservice.NewToolRegistry()
	mockStorage := new(MockToolsStorage)

	// Mock servers
	servers := []*storage.ServerMetadata{
		{
			ServerName: "server1",
			ServerURL:  "http://localhost:8001",
			Headers:    nil,
		},
		{
			ServerName: "server2",
			ServerURL:  "http://localhost:8002",
			Headers:    map[string]interface{}{"Authorization": "Bearer token"},
		},
	}
	mockStorage.On("ListServers", mock.Anything).Return(servers, nil)

	// Mock server tools for server1
	server1Tools := []*storage.ToolMetadata{
		{
			ToolName:    "tool1",
			Description: "Tool 1 description",
			Schema:      map[string]interface{}{"type": "object"},
			ServerName:  "server1",
		},
		{
			ToolName:    "tool2",
			Description: "Tool 2 description",
			Schema:      map[string]interface{}{"type": "object"},
			ServerName:  "server1",
		},
	}
	mockStorage.On("GetServerTools", mock.Anything, "server1").Return(server1Tools, nil)

	// Mock server tools for server2
	server2Tools := []*storage.ToolMetadata{
		{
			ToolName:    "tool3",
			Description: "Tool 3 description",
			Schema:      map[string]interface{}{"type": "object"},
			ServerName:  "server2",
		},
	}
	mockStorage.On("GetServerTools", mock.Anything, "server2").Return(server2Tools, nil)

	sync := NewMCPRegistrySync(registry, mockStorage, logger)
	count, err := sync.SyncAllMCPTools(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Verify tools are registered
	registeredTools := registry.List()
	assert.Contains(t, registeredTools, "mcp_server1_tool1")
	assert.Contains(t, registeredTools, "mcp_server1_tool2")
	assert.Contains(t, registeredTools, "mcp_server2_tool3")

	mockStorage.AssertExpectations(t)
}

func TestMCPRegistrySync_SyncServerTools(t *testing.T) {
	logger := zap.NewNop()
	registry := aiservice.NewToolRegistry()
	mockStorage := new(MockToolsStorage)

	server := &storage.ServerMetadata{
		ServerName: "testserver",
		ServerURL:  "http://localhost:8080",
		Headers:    nil,
	}
	mockStorage.On("GetServer", mock.Anything, "testserver").Return(server, nil)

	tools := []*storage.ToolMetadata{
		{
			ToolName:    "search",
			Description: "Search tool",
			Schema:      map[string]interface{}{"type": "object"},
			ServerName:  "testserver",
		},
	}
	mockStorage.On("GetServerTools", mock.Anything, "testserver").Return(tools, nil)

	sync := NewMCPRegistrySync(registry, mockStorage, logger)
	count, err := sync.SyncServerTools(context.Background(), "testserver")

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Contains(t, registry.List(), "mcp_testserver_search")

	mockStorage.AssertExpectations(t)
}

func TestMCPRegistrySync_RemoveServerTools(t *testing.T) {
	logger := zap.NewNop()
	registry := aiservice.NewToolRegistry()
	mockStorage := new(MockToolsStorage)

	// First, sync some tools
	server := &storage.ServerMetadata{
		ServerName: "testserver",
		ServerURL:  "http://localhost:8080",
		Headers:    nil,
	}
	mockStorage.On("GetServer", mock.Anything, "testserver").Return(server, nil)

	tools := []*storage.ToolMetadata{
		{
			ToolName:    "tool1",
			Description: "Tool 1",
			Schema:      map[string]interface{}{"type": "object"},
			ServerName:  "testserver",
		},
	}
	mockStorage.On("GetServerTools", mock.Anything, "testserver").Return(tools, nil)

	sync := NewMCPRegistrySync(registry, mockStorage, logger)
	_, err := sync.SyncServerTools(context.Background(), "testserver")
	require.NoError(t, err)

	// Verify tool was tracked
	assert.Equal(t, 1, sync.GetRegisteredToolCount())
	assert.Contains(t, sync.GetServerToolNames("testserver"), "mcp_testserver_tool1")

	// Remove tools
	err = sync.RemoveServerTools("testserver")
	require.NoError(t, err)

	// Verify tracking was cleared
	assert.Equal(t, 0, sync.GetRegisteredToolCount())
	assert.Nil(t, sync.GetServerToolNames("testserver"))
}

func TestMCPRegistrySync_GetRegisteredToolCount(t *testing.T) {
	logger := zap.NewNop()
	registry := aiservice.NewToolRegistry()
	mockStorage := new(MockToolsStorage)

	sync := NewMCPRegistrySync(registry, mockStorage, logger)

	// Initially zero
	assert.Equal(t, 0, sync.GetRegisteredToolCount())

	// Manually add to internal tracking (simulating a sync)
	sync.mu.Lock()
	sync.serverTools["server1"] = []string{"tool1", "tool2"}
	sync.serverTools["server2"] = []string{"tool3"}
	sync.mu.Unlock()

	assert.Equal(t, 3, sync.GetRegisteredToolCount())
}

func TestMCPRegistrySync_GetServerToolNames(t *testing.T) {
	logger := zap.NewNop()
	registry := aiservice.NewToolRegistry()
	mockStorage := new(MockToolsStorage)

	sync := NewMCPRegistrySync(registry, mockStorage, logger)

	// Non-existent server
	names := sync.GetServerToolNames("nonexistent")
	assert.Nil(t, names)

	// Add tracking
	sync.mu.Lock()
	sync.serverTools["myserver"] = []string{"mcp_myserver_tool1", "mcp_myserver_tool2"}
	sync.mu.Unlock()

	names = sync.GetServerToolNames("myserver")
	assert.Len(t, names, 2)
	assert.Contains(t, names, "mcp_myserver_tool1")
	assert.Contains(t, names, "mcp_myserver_tool2")

	// Verify it returns a copy (modification doesn't affect original)
	names[0] = "modified"
	originalNames := sync.GetServerToolNames("myserver")
	assert.NotEqual(t, "modified", originalNames[0])
}

func TestMCPRegistrySync_DuplicateRegistration(t *testing.T) {
	logger := zap.NewNop()
	registry := aiservice.NewToolRegistry()
	mockStorage := new(MockToolsStorage)

	server := &storage.ServerMetadata{
		ServerName: "testserver",
		ServerURL:  "http://localhost:8080",
		Headers:    nil,
	}
	mockStorage.On("GetServer", mock.Anything, "testserver").Return(server, nil)

	tools := []*storage.ToolMetadata{
		{
			ToolName:    "tool1",
			Description: "Tool 1",
			Schema:      map[string]interface{}{"type": "object"},
			ServerName:  "testserver",
		},
	}
	mockStorage.On("GetServerTools", mock.Anything, "testserver").Return(tools, nil)

	sync := NewMCPRegistrySync(registry, mockStorage, logger)

	// First sync
	count1, err := sync.SyncServerTools(context.Background(), "testserver")
	require.NoError(t, err)
	assert.Equal(t, 1, count1)

	// Second sync (same server) - should handle duplicates gracefully
	count2, err := sync.SyncServerTools(context.Background(), "testserver")
	require.NoError(t, err)
	assert.Equal(t, 1, count2) // Still counts as 1 (duplicate handled)
}
