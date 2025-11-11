package aiservice

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// ContextKey type for context keys
type contextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "requestID"
	// IdentityKey is the context key for user identity
	IdentityKey contextKey = "identity"
)

// Identity represents user identity extracted from JWT
type Identity struct {
	Type      string `json:"type"`      // "human", "agent", or "service"
	Name      string `json:"name"`      // User or agent name
	ID        string `json:"id"`        // User ID
	Email     string `json:"email"`     // User email
	CompanyID string `json:"companyId"` // Company ID for multi-tenancy
}

// StreamEventType defines the type of streaming event
type StreamEventType string

const (
	StreamEventToken      StreamEventType = "token"       // Text token
	StreamEventToolCall   StreamEventType = "tool_call"   // Tool invocation request
	StreamEventToolResult StreamEventType = "tool_result" // Tool execution result
	StreamEventError      StreamEventType = "error"       // Error during processing
)

// StreamEvent represents a streaming event (token, tool call, or tool result)
type StreamEvent struct {
	Type       StreamEventType `json:"type"`
	Content    string          `json:"content,omitempty"`    // For token events
	ToolCall   *ToolCall       `json:"toolCall,omitempty"`   // For tool_call events
	ToolResult *ToolResult     `json:"toolResult,omitempty"` // For tool_result events
	Error      string          `json:"error,omitempty"`      // For error events
}

// ChatService manages AI chat operations with provider abstraction
type ChatService struct {
	provider     ChatProvider
	config       *AIConfig
	toolRegistry *ToolRegistry

	// Original configuration for fallback recovery
	originalModel    string
	originalProvider string
	originalAPIKey   string
	usingFallback    bool
}

// getClaudeSystemPrompt returns the Claude-optimized system prompt
// This is the same prompt used in chat_websocket.go for Claude models
// Uses outcome-focused language, real response examples, and concrete guidance
func getClaudeSystemPrompt() string {
	return `# Task Coordination System

You are a development task coordinator. Your goal: **Get user requests completed by delegating to specialist agents**.

## Your Role

**Coordinate, don't implement.** You create tasks and delegate to specialists who write the actual code.

### What You Do:
1. Understand what the user wants
2. Create tasks to track the work
3. Find relevant files using code search
4. Create detailed instructions for specialists
5. Launch the specialist agent to do the work

### What You Don't Do:
- Write implementation code yourself
- Read multiple files trying to understand the codebase
- Retry searches hoping for better results
- Explore directories without a specific goal

## Workflow

There's no strict step order, but typically:

1. **Check for existing work** (optional): coordinator_list_human_tasks
2. **Record the request**: coordinator_create_human_task
3. **Find relevant files** (if needed): code_index_search
4. **Create agent task**: create_agent_task
5. **Launch specialist**: execute_subagent

### Key Principles:
- **Trust first results**: Accept what tools return on first call
- **One search only**: Do ONE code search, use those results
- **Exact file paths**: Copy paths directly from tool responses
- **Delegate quickly**: After gathering context, hand off to specialist

## Tool Response Examples

### code_index_search Returns:
` + "```json" + `
{
  "results": [
    {
      "filePath": "/Users/name/project/ui/src/components/Settings.tsx",
      "content": "export function Settings() { ... }",
      "score": 0.92,
      "startLine": 15,
      "endLine": 45
    },
    {
      "filePath": "/Users/name/project/ui/src/hooks/useDarkMode.ts",
      "content": "export const useDarkMode = () => { ... }",
      "score": 0.87,
      "startLine": 8,
      "endLine": 25
    }
  ],
  "query": "settings dark mode",
  "totalResults": 2
}
` + "```" + `

**How to use in create_agent_task:**
` + "```json" + `
{
  "filesModified": [
    "/Users/name/project/ui/src/components/Settings.tsx",
    "/Users/name/project/ui/src/hooks/useDarkMode.ts"
  ],
  "todos": [
    {
      "description": "Add dark mode toggle to Settings component",
      "filePath": "/Users/name/project/ui/src/components/Settings.tsx",
      "contextHint": "Line 15-45: Add toggle button, connect to useDarkMode hook"
    }
  ]
}
` + "```" + `

**Important**: Copy the filePath values EXACTLY. Don't shorten, modify, or "fix" them.

**🚨 CRITICAL RULES FOR filesModified AND todos:**

1. **filesModified MUST contain file paths** - NEVER pass an empty array!
   - If you ran code_index_search, use the filePath values from results
   - If no search was done, leave filesModified empty ONLY if task doesn't modify files

2. **NEVER create TODOs that require discovery tools**
   - ❌ BAD: "Search for components using code_index_search"
   - ❌ BAD: "Find the settings file"
   - ❌ BAD: "Locate the auth logic"
   - ✅ GOOD: "Add responsive CSS to Settings.tsx"
   - ✅ GOOD: "Update login form validation"

3. **Discovery is YOUR job, not the subagent's job**
   - code_index_search, list_directory → YOU run these BEFORE creating agent task
   - read_file, write_file, apply_patch → Subagent runs these to implement

4. **TODOs should be implementation steps ONLY**
   - "Read file X and add feature Y"
   - "Update function Z in file W"
   - "Test the changes work correctly"

**Why this matters:** Subagents run in WRITE-ONLY MODE where discovery tools are BLOCKED. If you create a TODO requiring code_index_search, the subagent literally cannot complete it and will do nothing.

### coordinator_create_human_task Returns:
` + "```json" + `
{
  "taskId": "a1b2c3d4-e5f6-4789-a012-b3c4d5e6f789",
  "prompt": "Add dark mode toggle to settings",
  "status": "pending",
  "createdAt": "2025-01-15T10:30:00Z"
}
` + "```" + `

**CRITICAL: Extract the EXACT "taskId" UUID value from the response above.**
- DO NOT generate, make up, or create your own task ID
- DO NOT use descriptive strings like "add-dark-mode" or "change-button-color"
- COPY the exact UUID (e.g., "a1b2c3d4-e5f6-4789-a012-b3c4d5e6f789") from the tool response
- USE that exact taskId value as "humanTaskId" when calling create_agent_task

Example: If the response shows "taskId": "f0205882-473b-49bf-b3ba-adc30ab82fc3", then use EXACTLY "humanTaskId": "f0205882-473b-49bf-b3ba-adc30ab82fc3" in create_agent_task.

### create_agent_task Expects:
` + "```json" + `
{
  "humanTaskId": "a1b2c3d4-e5f6-4789-a012-b3c4d5e6f789",
  "agentName": "ui-dev",
  "role": "Add dark mode toggle to Settings page",
  "contextSummary": "User wants a dark mode toggle in the Settings component. Files found: Settings.tsx (lines 15-45 contain main component), useDarkMode.ts (lines 8-25 contain the hook). Need to add toggle UI element and wire it to the existing useDarkMode hook.",
  "filesModified": [
    "/Users/name/project/ui/src/components/Settings.tsx",
    "/Users/name/project/ui/src/hooks/useDarkMode.ts"
  ],
  "todos": [
    {
      "description": "Add dark mode toggle button to Settings component UI",
      "filePath": "/Users/name/project/ui/src/components/Settings.tsx",
      "contextHint": "Line 15-45: Add <ToggleSwitch> component, place in settings grid. Follow existing pattern for other toggle switches in the file."
    },
    {
      "description": "Connect toggle to useDarkMode hook state",
      "filePath": "/Users/name/project/ui/src/components/Settings.tsx",
      "contextHint": "Import useDarkMode from hooks file, destructure { darkMode, setDarkMode }, pass setDarkMode to toggle onChange handler."
    }
  ]
}
` + "```" + `

## Common Scenarios

### Scenario 1: Simple Feature Addition
` + "```" + `
User: "Add a save button to the editor"

Your approach:
1. coordinator_list_human_tasks (check if similar task exists)
2. coordinator_create_human_task (record request → get task ID)
3. code_index_search("editor save button") → get file paths
4. create_agent_task (humanTaskId from step 2, files from step 3)
5. execute_subagent (launch ui-dev)
6. Done! Specialist takes over from here.
` + "```" + `

### Scenario 2: Bug Fix
` + "```" + `
User: "The login form doesn't validate email format"

Your approach:
1. coordinator_list_human_tasks
2. coordinator_create_human_task → task ID
3. code_index_search("login form email validation")
4. create_agent_task with bug context: "Email validation missing from login form. Found LoginForm.tsx with form logic. Need to add email format regex check before submit."
5. execute_subagent
` + "```" + `

### Scenario 3: Search Returns No Results
` + "```" + `
User: "Update the API endpoint in auth service"

code_index_search("auth service API endpoint") → { results: [], totalResults: 0 }

Your options:
a) Try one more search with different terms: code_index_search("authentication API")
b) Ask user: "I couldn't find auth service files. Can you tell me where they're located?"
c) Create agent task anyway with instruction: "Find auth service API endpoints and update them"

DON'T: Keep searching with variations (auth, authentication, login, etc.) - that triggers circuit breaker
` + "```" + `

## Error Recovery

### Tool Fails Once
Try again with different parameters:
` + "```" + `
code_index_search("dark mode settings") → ERROR: timeout
→ Try: code_index_search("dark mode")
` + "```" + `

### Tool Fails Twice (Same Parameters)
Stop and inform user:
` + "```" + `
"The code search tool failed twice. This might be a system issue. Should I:
a) Try creating the task without file paths (agent will search)
b) Skip this request for now"
` + "```" + `

### File Path Errors
If create_agent_task fails with "file not found":
` + "```" + `
Error: "path does not exist: /old/path/Settings.tsx"

→ Don't retry with same path!
→ Do: Tell user "The search found an outdated path. Can you tell me where Settings.tsx is now?"
` + "```" + `

## Agent Specializations

- **ui-dev**: React/TypeScript UI components, pages, styling
- **go-dev**: Go backend services, APIs, business logic
- **sre**: Deployment, infrastructure, monitoring
- **go-mcp-dev**: MCP protocol implementation in Go
- **ui-tester**: Playwright tests, UI automation

Choose based on the file types and work needed.

## Remember

- **Fast delegation** beats perfect planning
- **First search results** are usually good enough
- **Exact file paths** prevent 90% of errors
- **Clear context** in agent tasks helps specialists succeed
- **You coordinate**, specialists implement`
}

// ToolResultCache caches tool execution results by signature to prevent duplicate executions
// Helps reduce circuit breaker hits by reusing results when the same tool is called with identical arguments
type ToolResultCache struct {
	cache map[string]*ToolResult
	mu    sync.RWMutex
}

// NewToolResultCache creates a new tool result cache
func NewToolResultCache() *ToolResultCache {
	return &ToolResultCache{
		cache: make(map[string]*ToolResult),
	}
}

// Get retrieves a cached tool result if it exists
func (c *ToolResultCache) Get(signature string) (*ToolResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result, found := c.cache[signature]
	return result, found
}

// Set stores a tool result in the cache
func (c *ToolResultCache) Set(signature string, result *ToolResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Create a deep copy to avoid mutation issues
	cachedResult := &ToolResult{
		Name:       result.Name,
		Output:     result.Output,
		Error:      result.Error,
		DurationMs: result.DurationMs,
	}
	c.cache[signature] = cachedResult
}

// Delete removes a cached tool result by signature
func (c *ToolResultCache) Delete(signature string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, signature)
}

// DeletePrefix removes all cached tool results with signatures starting with the given prefix
// Returns the count of entries deleted
func (c *ToolResultCache) DeletePrefix(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := 0
	for signature := range c.cache {
		if len(signature) >= len(prefix) && signature[:len(prefix)] == prefix {
			delete(c.cache, signature)
			count++
		}
	}
	return count
}

// NewChatService creates a new ChatService with the given configuration
// Creates an empty tool registry - use RegisterTool() or GetToolRegistry() to add tools
func NewChatService(config *AIConfig) (*ChatService, error) {
	provider, err := NewChatProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Initialize empty tool registry
	// Tools should be registered after creation using RegisterTool() or GetToolRegistry()
	toolRegistry := NewToolRegistry()

	return &ChatService{
		provider:         provider,
		config:           config,
		toolRegistry:     toolRegistry,
		originalModel:    config.Model,
		originalProvider: config.Provider,
		originalAPIKey:   config.APIKey,
		usingFallback:    false,
	}, nil
}

// NewChatServiceWithTools creates a ChatService with a pre-configured tool registry
// Useful when you want to inject a tool registry with pre-registered tools
func NewChatServiceWithTools(config *AIConfig, toolRegistry *ToolRegistry) (*ChatService, error) {
	provider, err := NewChatProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &ChatService{
		provider:         provider,
		config:           config,
		toolRegistry:     toolRegistry,
		originalModel:    config.Model,
		originalProvider: config.Provider,
		originalAPIKey:   config.APIKey,
		usingFallback:    false,
	}, nil
}

// RegisterTool adds a tool to the service's tool registry
func (s *ChatService) RegisterTool(tool ToolExecutor) error {
	return s.toolRegistry.Register(tool)
}

// GetToolRegistry returns the tool registry for external tool registration
func (s *ChatService) GetToolRegistry() *ToolRegistry {
	return s.toolRegistry
}

// StreamChat sends messages to AI provider and streams the response (legacy text-only method)
// For tool-enabled streaming, use StreamChatWithTools
// Extracts user identity from context for logging and multi-tenancy
func (s *ChatService) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	// Extract identity from context (for logging and multi-tenancy)
	identity := s.getIdentityFromContext(ctx)
	requestID := s.getRequestIDFromContext(ctx)

	// Log the request
	if identity != nil {
		log.Printf("[ChatService] Request from %s (%s) - RequestID: %s - Provider: %s Model: %s",
			identity.Name, identity.Type, requestID, s.config.Provider, s.config.Model)
	} else {
		log.Printf("[ChatService] Request (no identity) - RequestID: %s - Provider: %s Model: %s",
			requestID, s.config.Provider, s.config.Model)
	}

	// Validate messages
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	// Call provider's StreamChat
	outputChan, err := s.provider.StreamChat(ctx, messages)
	if err != nil {
		log.Printf("[ChatService] ERROR - RequestID: %s - Failed to stream: %v", requestID, err)
		return nil, fmt.Errorf("failed to stream chat: %w", err)
	}

	// Wrap the output channel to handle context cancellation and logging
	wrappedChan := make(chan string, 100)

	go func() {
		defer close(wrappedChan)

		tokenCount := 0
		for {
			select {
			case <-ctx.Done():
				// Context cancelled
				log.Printf("[ChatService] Context cancelled - RequestID: %s - Tokens streamed: %d",
					requestID, tokenCount)
				return

			case chunk, ok := <-outputChan:
				if !ok {
					// Provider channel closed
					log.Printf("[ChatService] Stream complete - RequestID: %s - Total tokens: %d",
						requestID, tokenCount)
					return
				}

				tokenCount++

				// Forward chunk to wrapped channel
				select {
				case <-ctx.Done():
					return
				case wrappedChan <- chunk:
					// Chunk sent successfully
				}
			}
		}
	}()

	return wrappedChan, nil
}

// NOTE: StreamChatWithTools, StreamChatWithToolsFiltered, and generateWorkflowStateGuidance
// have been extracted to tool_executor.go for better code organization.
// See tool_executor.go for the full implementation.

// GetConfig returns the AI configuration for this service
func (s *ChatService) GetConfig() *AIConfig {
	return s.config
}

// GetAllowedToolsForDirectSubagent returns the list of tools that direct subagent chats can use
// This excludes coordinator delegation tools to prevent subchats from being created
// Direct subagent mode is when user communicates directly with a specific agent (go-dev, ui-dev, etc.)
// In this mode, the agent should work autonomously without delegating to other agents
func (s *ChatService) GetAllowedToolsForDirectSubagent() []string {
	// Get all registered tool names
	allTools := s.toolRegistry.List()

	// Define blocked tools (delegation and coordinator management tools)
	blockedTools := map[string]bool{
		"execute_subagent":              true, // CRITICAL: Prevent direct subagents from creating subchats
		"coordinator_create_human_task": true, // Direct subagents cannot create human tasks
		"coordinator_create_agent_task": true, // Direct subagents cannot create agent tasks
		"coordinator_list_human_tasks":  true, // Direct subagents should not list human tasks
		"coordinator_list_agent_tasks":  true, // Direct subagents should not list agent tasks
		"create_agent_task":             true, // Direct subagents use autonomous execution only
	}

	// Filter out blocked tools
	allowedTools := make([]string, 0, len(allTools))
	for _, toolName := range allTools {
		if !blockedTools[toolName] {
			allowedTools = append(allowedTools, toolName)
		}
	}

	return allowedTools
}

// getIdentityFromContext extracts user identity from context
func (s *ChatService) getIdentityFromContext(ctx context.Context) *Identity {
	identity, ok := ctx.Value(IdentityKey).(*Identity)
	if !ok {
		return nil
	}
	return identity
}

// getRequestIDFromContext extracts request ID from context
func (s *ChatService) getRequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(RequestIDKey).(string)
	if !ok || requestID == "" {
		return "unknown"
	}
	return requestID
}
