package aiservice

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ToolExecutor defines the interface for tools that can be called by AI
type ToolExecutor interface {
	// Name returns the tool name (must be lowercase_snake_case)
	Name() string
	// Description returns a human-readable description for the AI
	Description() string
	// InputSchema returns JSON schema for the tool's input parameters
	InputSchema() map[string]interface{}
	// Execute runs the tool with given input and returns result
	Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

// ToolCall represents a tool invocation request from the AI
type ToolCall struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Args       map[string]interface{} `json:"args"`
	Output     interface{} `json:"output,omitempty"`
	Error      string      `json:"error,omitempty"`
	DurationMs int64       `json:"durationMs"`
}

// ToolRegistry manages available tools and provides methods for tool execution
type ToolRegistry struct {
	tools map[string]ToolExecutor
	mu    sync.RWMutex
}

// NewToolRegistry creates a new empty ToolRegistry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolExecutor),
	}
}

// NewToolRegistryWithMCPTools creates a ToolRegistry with all MCP tools registered
// This registers coordinator, Qdrant, and code index tools for use in chat
func NewToolRegistryWithMCPTools(
	taskStorage interface{}, // storage.TaskStorage
	knowledgeStorage interface{}, // storage.KnowledgeStorage
	codeIndexStorage interface{}, // *storage.CodeIndexStorage
	qdrantClient interface{}, // storage.QdrantClientInterface
) (*ToolRegistry, error) {
	registry := NewToolRegistry()

	// Import MCP tool packages dynamically to avoid circular dependencies
	// The actual registration is done by the calling code which has access to the packages
	// This is just a helper that creates the registry structure

	return registry, nil
}

// Register adds a tool to the registry
// Tool names must be lowercase_snake_case and unique
func (r *ToolRegistry) Register(tool ToolExecutor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate tool name format (lowercase_snake_case)
	if !isValidToolName(tool.Name()) {
		return fmt.Errorf("invalid tool name '%s': must be lowercase_snake_case", tool.Name())
	}

	// Check for duplicates
	if _, exists := r.tools[tool.Name()]; exists {
		return fmt.Errorf("tool '%s' already registered", tool.Name())
	}

	r.tools[tool.Name()] = tool
	return nil
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (ToolExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return tool, nil
}

// List returns all registered tool names
func (r *ToolRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute runs a tool by name with the given input
func (r *ToolRegistry) Execute(ctx context.Context, name string, input map[string]interface{}) (interface{}, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	// Execute with timeout (5 minutes default)
	execCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return tool.Execute(execCtx, input)
}

// ExecuteToolCall executes a ToolCall and returns ToolResult with timing
// OPTION 2 FIX: Ensures ToolResult.ID is ALWAYS populated from toolCall.ID
func (r *ToolRegistry) ExecuteToolCall(ctx context.Context, toolCall ToolCall) ToolResult {
	startTime := time.Now()

	// VALIDATION: Check if toolCall.ID is empty (root cause of SaveToolResult bug)
	if toolCall.ID == "" {
		fmt.Printf("[⚠️  BUG DETECTED] ExecuteToolCall: toolCall.ID is EMPTY for tool '%s'\n", toolCall.Name)
		fmt.Printf("[⚠️  BUG DETECTED] This will cause SaveToolResult to fail with empty toolCallID\n")
		fmt.Printf("[⚠️  BUG DETECTED] Tool: %s, Args: %v\n", toolCall.Name, toolCall.Args)
	}

	result := ToolResult{
		ID:   toolCall.ID,
		Name: toolCall.Name,
		Args: toolCall.Args,
	}

	output, err := r.Execute(ctx, toolCall.Name, toolCall.Args)
	result.DurationMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Error = err.Error()
	} else {
		result.Output = output
	}

	return result
}

// GetTools converts registered tools to the native Tool format
// This is used to pass tools to LLM providers
func (r *ToolRegistry) GetTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.getToolsUnlocked()
}

// getToolsUnlocked is an internal helper that assumes the lock is already held
func (r *ToolRegistry) getToolsUnlocked() []Tool {
	tools := make([]Tool, 0, len(r.tools))

	for _, tool := range r.tools {
		nativeTool := Tool{
			Type: "function",
			Function: &FunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.InputSchema(),
			},
		}
		tools = append(tools, nativeTool)
	}

	return tools
}

// GetFilteredTools converts only specified tools to the native Tool format
// This is used to restrict which tools are available to subagents or other contexts
// allowedNames: list of tool names to include (e.g., []string{"read_file", "write_file"})
// If allowedNames is nil, returns ALL tools (coordinator mode with full access)
func (r *ToolRegistry) GetFilteredTools(allowedNames []string) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If allowedNames is nil, return ALL tools (coordinator mode)
	if allowedNames == nil {
		return r.getToolsUnlocked()
	}

	// Create a set for O(1) lookup
	allowedSet := make(map[string]bool, len(allowedNames))
	for _, name := range allowedNames {
		allowedSet[name] = true
	}

	tools := make([]Tool, 0, len(allowedNames))

	for name, tool := range r.tools {
		// Only include if in allowed list
		if allowedSet[name] {
			nativeTool := Tool{
				Type: "function",
				Function: &FunctionDefinition{
					Name:        tool.Name(),
					Description: tool.Description(),
					Parameters:  tool.InputSchema(),
				},
			}
			tools = append(tools, nativeTool)
		}
	}

	return tools
}

// GetToolsForLangChain is a backwards-compatible alias for GetTools
// Deprecated: Use GetTools instead
func (r *ToolRegistry) GetToolsForLangChain() []Tool {
	return r.GetTools()
}

// GetFilteredToolsForLangChain is a backwards-compatible alias for GetFilteredTools
// Deprecated: Use GetFilteredTools instead
func (r *ToolRegistry) GetFilteredToolsForLangChain(allowedNames []string) []Tool {
	return r.GetFilteredTools(allowedNames)
}

// isValidToolName checks if a tool name follows lowercase_snake_case convention
func isValidToolName(name string) bool {
	// Must be lowercase letters, numbers, and underscores only
	// Must start with a letter
	// Cannot have consecutive underscores
	pattern := `^[a-z][a-z0-9]*(_[a-z0-9]+)*$`
	matched, _ := regexp.MatchString(pattern, name)
	return matched && !strings.Contains(name, "__")
}
