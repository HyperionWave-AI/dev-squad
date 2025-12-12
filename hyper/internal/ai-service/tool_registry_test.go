package aiservice

import (
	"context"
	"testing"
)

// MockToolExecutor is a simple mock tool for testing
type MockToolExecutor struct {
	name        string
	description string
	schema      map[string]interface{}
}

func (m *MockToolExecutor) Name() string {
	return m.name
}

func (m *MockToolExecutor) Description() string {
	return m.description
}

func (m *MockToolExecutor) InputSchema() map[string]interface{} {
	return m.schema
}

func (m *MockToolExecutor) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"result": "mock"}, nil
}

// TestGetFilteredToolsForLangChain_NilAllowedNames tests the coordinator mode (nil = all tools)
func TestGetFilteredToolsForLangChain_NilAllowedNames(t *testing.T) {
	registry := NewToolRegistry()

	// Register some test tools
	tool1 := &MockToolExecutor{
		name:        "discover_tools",
		description: "Discover MCP tools",
		schema:      map[string]interface{}{"type": "object"},
	}
	tool2 := &MockToolExecutor{
		name:        "get_tool_schema",
		description: "Get tool schema",
		schema:      map[string]interface{}{"type": "object"},
	}
	tool3 := &MockToolExecutor{
		name:        "execute_tool",
		description: "Execute a tool",
		schema:      map[string]interface{}{"type": "object"},
	}

	if err := registry.Register(tool1); err != nil {
		t.Fatalf("Failed to register tool1: %v", err)
	}
	if err := registry.Register(tool2); err != nil {
		t.Fatalf("Failed to register tool2: %v", err)
	}
	if err := registry.Register(tool3); err != nil {
		t.Fatalf("Failed to register tool3: %v", err)
	}

	// Test: nil allowedNames should return ALL tools (coordinator mode)
	tools := registry.GetFilteredToolsForLangChain(nil)

	if len(tools) != 3 {
		t.Errorf("Expected 3 tools with nil allowedNames (coordinator mode), got %d", len(tools))
	}

	// Verify all tools are present
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		if tool.Function != nil {
			toolNames[tool.Function.Name] = true
		}
	}

	expectedTools := []string{"discover_tools", "get_tool_schema", "execute_tool"}
	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool '%s' to be present in coordinator mode, but it was missing", expected)
		}
	}
}

// TestGetFilteredToolsForLangChain_WithAllowList tests subagent mode (filtered tools)
func TestGetFilteredToolsForLangChain_WithAllowList(t *testing.T) {
	registry := NewToolRegistry()

	// Register some test tools
	tool1 := &MockToolExecutor{
		name:        "discover_tools",
		description: "Discover MCP tools",
		schema:      map[string]interface{}{"type": "object"},
	}
	tool2 := &MockToolExecutor{
		name:        "bash",
		description: "Execute bash command",
		schema:      map[string]interface{}{"type": "object"},
	}
	tool3 := &MockToolExecutor{
		name:        "execute_subagent",
		description: "Execute subagent (blocked for direct subagents)",
		schema:      map[string]interface{}{"type": "object"},
	}

	if err := registry.Register(tool1); err != nil {
		t.Fatalf("Failed to register tool1: %v", err)
	}
	if err := registry.Register(tool2); err != nil {
		t.Fatalf("Failed to register tool2: %v", err)
	}
	if err := registry.Register(tool3); err != nil {
		t.Fatalf("Failed to register tool3: %v", err)
	}

	// Test: allowedNames should filter tools (subagent mode)
	allowedTools := []string{"discover_tools", "bash"}
	tools := registry.GetFilteredToolsForLangChain(allowedTools)

	if len(tools) != 2 {
		t.Errorf("Expected 2 tools with allowlist, got %d", len(tools))
	}

	// Verify only allowed tools are present
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		if tool.Function != nil {
			toolNames[tool.Function.Name] = true
		}
	}

	if !toolNames["discover_tools"] {
		t.Error("Expected 'discover_tools' to be present in filtered list")
	}
	if !toolNames["bash"] {
		t.Error("Expected 'bash' to be present in filtered list")
	}
	if toolNames["execute_subagent"] {
		t.Error("Did not expect 'execute_subagent' to be present in filtered list")
	}
}

// TestGetFilteredToolsForLangChain_EmptyAllowList tests empty list (no tools mode)
func TestGetFilteredToolsForLangChain_EmptyAllowList(t *testing.T) {
	registry := NewToolRegistry()

	// Register some test tools
	tool1 := &MockToolExecutor{
		name:        "discover_tools",
		description: "Discover MCP tools",
		schema:      map[string]interface{}{"type": "object"},
	}

	if err := registry.Register(tool1); err != nil {
		t.Fatalf("Failed to register tool1: %v", err)
	}

	// Test: empty allowedNames should return NO tools
	tools := registry.GetFilteredToolsForLangChain([]string{})

	if len(tools) != 0 {
		t.Errorf("Expected 0 tools with empty allowlist, got %d", len(tools))
	}
}

// TestGetToolsForLangChain tests the unfiltered method
func TestGetToolsForLangChain(t *testing.T) {
	registry := NewToolRegistry()

	// Register some test tools
	tool1 := &MockToolExecutor{
		name:        "tool1",
		description: "Tool 1",
		schema:      map[string]interface{}{"type": "object"},
	}
	tool2 := &MockToolExecutor{
		name:        "tool2",
		description: "Tool 2",
		schema:      map[string]interface{}{"type": "object"},
	}

	if err := registry.Register(tool1); err != nil {
		t.Fatalf("Failed to register tool1: %v", err)
	}
	if err := registry.Register(tool2); err != nil {
		t.Fatalf("Failed to register tool2: %v", err)
	}

	// Test: GetToolsForLangChain should return ALL tools
	tools := registry.GetToolsForLangChain()

	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}
