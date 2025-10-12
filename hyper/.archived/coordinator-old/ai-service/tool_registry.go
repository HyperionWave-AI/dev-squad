package aiservice

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms"
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

	// Execute with timeout (30 seconds default)
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return tool.Execute(execCtx, input)
}

// ExecuteToolCall executes a ToolCall and returns ToolResult with timing
func (r *ToolRegistry) ExecuteToolCall(ctx context.Context, toolCall ToolCall) ToolResult {
	startTime := time.Now()

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

// GetToolsForLangChain converts registered tools to LangChain Go format
// This is used to pass tools to LLM providers via llms.WithTools()
func (r *ToolRegistry) GetToolsForLangChain() []llms.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]llms.Tool, 0, len(r.tools))

	for _, tool := range r.tools {
		// Convert to LangChain format
		langChainTool := llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.InputSchema(),
			},
		}
		tools = append(tools, langChainTool)
	}

	return tools
}

// ParseToolCallsFromResponse extracts tool calls from LangChain response
// This handles the LangChain AIChatMessage format
func ParseToolCallsFromResponse(msg llms.ChatMessage) ([]ToolCall, error) {
	aiMsg, ok := msg.(*llms.AIChatMessage)
	if !ok {
		return nil, fmt.Errorf("expected AIChatMessage, got %T", msg)
	}

	// If no tool calls, return empty slice
	if len(aiMsg.ToolCalls) == 0 {
		return []ToolCall{}, nil
	}

	var toolCalls []ToolCall
	for _, tc := range aiMsg.ToolCalls {
		// Parse arguments from JSON string to map
		var args map[string]interface{}
		if tc.FunctionCall != nil && tc.FunctionCall.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
			}
		}

		toolCalls = append(toolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.FunctionCall.Name,
			Args: args,
		})
	}

	return toolCalls, nil
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
