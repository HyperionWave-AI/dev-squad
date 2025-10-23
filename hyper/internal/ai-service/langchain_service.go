package aiservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
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

// StreamChatWithTools sends messages to AI provider with tool support and streams events
// Handles tool calls automatically: when AI requests a tool, executes it and returns result
// Returns channel of StreamEvent which can be tokens, tool calls, or tool results
func (s *ChatService) StreamChatWithTools(ctx context.Context, messages []Message, maxToolCalls int) (<-chan StreamEvent, error) {
	identity := s.getIdentityFromContext(ctx)
	requestID := s.getRequestIDFromContext(ctx)

	// Log the request
	if identity != nil {
		log.Printf("[ChatService] Tool-enabled request from %s (%s) - RequestID: %s - Provider: %s Model: %s",
			identity.Name, identity.Type, requestID, s.config.Provider, s.config.Model)
	} else {
		log.Printf("[ChatService] Tool-enabled request (no identity) - RequestID: %s - Provider: %s Model: %s",
			requestID, s.config.Provider, s.config.Model)
	}

	// Validate messages
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	// Try to switch back to primary model if we're using fallback
	if s.usingFallback {
		log.Printf("[Rate Limit Recovery] Attempting to switch back to primary model: %s", s.originalModel)

		// Restore original configuration
		s.config.Model = s.originalModel
		s.config.Provider = s.originalProvider
		s.config.APIKey = s.originalAPIKey

		// Recreate provider with original config
		primaryProvider, err := NewChatProvider(s.config)
		if err != nil {
			log.Printf("[Rate Limit Recovery] Failed to recreate primary provider, staying on fallback: %v", err)
		} else {
			// Update provider and mark as no longer using fallback
			s.provider = primaryProvider
			s.usingFallback = false
			log.Printf("[Rate Limit Recovery] Successfully switched back to primary model: %s", s.originalModel)
		}
	}

	// Default max tool calls to prevent loops
	if maxToolCalls <= 0 {
		maxToolCalls = 5
	}

	// Create output channel for events
	eventChan := make(chan StreamEvent, 100)

	// Get tools for LangChain
	tools := s.toolRegistry.GetToolsForLangChain()

	// Check if provider supports tools
	supportsTools := false
	if toolProvider, ok := s.provider.(ToolCapableProvider); ok {
		supportsTools = toolProvider.SupportsTools()
	}

	if !supportsTools || len(tools) == 0 {
		// Fallback to text-only streaming
		log.Printf("[ChatService] Provider doesn't support tools or no tools registered - RequestID: %s", requestID)
		go func() {
			defer close(eventChan)
			textChan, err := s.provider.StreamChat(ctx, messages)
			if err != nil {
				eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
				return
			}
			for chunk := range textChan {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
			}
		}()
		return eventChan, nil
	}

	// Start tool-enabled streaming
	go func() {
		defer close(eventChan)

		toolCallCount := 0
		iterationCount := 0
		currentMessages := append([]Message{}, messages...) // Copy messages

		// Tool result cache: prevent duplicate tool executions
		resultCache := NewToolResultCache()

		// Circuit breaker: track recent tool calls to detect infinite loops
		recentToolCalls := make([]string, 0, 10)
		failedToolCalls := make(map[string]int)        // Track failed attempts separately
		pathValidationRetries := make(map[string]bool) // Track file path validation retries for code_index_search
		taskIdValidationAttempts := 0                   // Track taskId validation attempts for create_agent_task (max 3)

		// Tool call history: track all executed tools for smart filtering (reduces token usage by ~70%)
		toolCallHistory := make([]ToolResult, 0, 20)
		toolCallSignature := func(name string, args map[string]interface{}) string {
			argsJSON, _ := json.Marshal(args)
			return fmt.Sprintf("%s(%s)", name, string(argsJSON))
		}

		// WORKFLOW STATE ENFORCEMENT: Track coordinator workflow progress (fallback model only)
		workflowState := map[string]interface{}{
			"step":            0,     // 0=initial, 1=listed, 2=created, 3=searched, 4=agent_task, 5=done
			"humanTaskId":     "",    // Store taskId from step 2
			"searchCompleted": false, // Prevent multiple searches
			"agentTaskId":     "",    // Store agentTaskId from step 4
		}

		// Function to validate workflow tool calls (only enforced for fallback model)
		validateWorkflowTool := func(toolName string) (bool, string) {
			if !s.usingFallback {
				return true, "" // No enforcement for primary model
			}

			step := workflowState["step"].(int)
			humanTaskId := workflowState["humanTaskId"].(string)
			searchCompleted := workflowState["searchCompleted"].(bool)

			switch toolName {
			case "coordinator_list_human_tasks":
				if step == 0 || step == 1 || step == 2 {
					// Allow listing tasks at any early step to retrieve exact taskId
					return true, ""
				}
				return false, "❌ BLOCKED: You already have taskId. NEXT: Call appropriate tool for current step."

			case "coordinator_create_human_task":
				if step == 1 && humanTaskId == "" {
					return true, ""
				}
				if humanTaskId != "" {
					// State-aware guidance based on workflow progress
					if searchCompleted {
						return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'create_agent_task'.", humanTaskId)
					}
					return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'code_index_search'.", humanTaskId)
				}
				return false, "❌ BLOCKED: Call 'coordinator_list_human_tasks' first."

			case "code_index_search":
				if step == 2 && humanTaskId != "" && !searchCompleted {
					return true, ""
				}
				if searchCompleted {
					return false, "❌ BLOCKED: Search done. NEXT: Call 'create_agent_task'."
				}
				return false, "❌ BLOCKED: Create human task first."

			case "create_agent_task":
				if step == 3 && humanTaskId != "" && searchCompleted {
					return true, ""
				}
				return false, "❌ BLOCKED: Run 'code_index_search' first to get file paths."

			case "execute_subagent":
				agentTaskId := workflowState["agentTaskId"].(string)
				if step == 4 && agentTaskId != "" {
					return true, ""
				}
				return false, "❌ BLOCKED: Call 'create_agent_task' first."
			}

			return true, ""
		}

		// Per-tool circuit breaker thresholds (max duplicate attempts before stopping)
		// Claude models get higher thresholds as they're better at adapting
		isClaudeModel := strings.Contains(strings.ToLower(s.config.Model), "claude") ||
			strings.Contains(strings.ToLower(s.config.Provider), "anthropic")

		var circuitBreakerThresholds map[string]int
		if isClaudeModel {
			// Claude-optimized thresholds: More lenient to allow legitimate multi-file operations
			circuitBreakerThresholds = map[string]int{
				"read_file":         5, // Allow reading multiple files
				"write_file":        2, // Allow one retry for writes
				"list_directory":    4, // Allow exploring directories
				"bash":              5, // Allow command variations
				"code_index_search": 2, // Strict: one search + one retry max
				"create_agent_task": 4, // Allow retries for parameter refinement
				// Default for other tools: 6 attempts
			}
			log.Printf("[Circuit Breaker] Using Claude-optimized thresholds (more lenient)")
		} else {
			// GPT thresholds: More conservative
			circuitBreakerThresholds = map[string]int{
				"read_file":         2, // Stop after 2 attempts (only 1 duplicate allowed)
				"write_file":        1, // Never allow duplicate writes
				"list_directory":    2, // Stop after 2 attempts
				"bash":              3, // Allow more for command variations
				"code_index_search": 3, // Allow query refinement
				// Default for other tools: 4 attempts
			}
			log.Printf("[Circuit Breaker] Using GPT thresholds (conservative)")
		}

		for toolCallCount < maxToolCalls && iterationCount < s.config.MaxIterations {
			iterationCount++

			// CRITICAL: Reload full tools array at start of each iteration
			// This prevents the filtered tools from previous iteration being reused
			tools = s.toolRegistry.GetToolsForLangChain()

			// Calculate context size BEFORE applying sliding window
			contextSize := 0
			for _, msg := range currentMessages {
				contextSize += len(msg.Content)
			}

			// Apply sliding window BEFORE context exceeds model's token limit
			// Claude: 200K tokens (≈800KB text) - use 150KB threshold
			// GPT: 32K tokens (≈128KB text) - use 40KB threshold to be safe
			var maxContextSize int
			var maxMessages int
			if isClaudeModel {
				maxContextSize = 150000 // 150KB for Claude (≈37K tokens, leaves room for output)
				maxMessages = 20        // Keep more messages for Claude
				log.Printf("[Context Window] Using Claude limits: %d chars, max %d messages", maxContextSize, maxMessages)
			} else {
				maxContextSize = 40000 // 40KB for GPT (≈10K tokens)
				maxMessages = 6        // Conservative for GPT
				log.Printf("[Context Window] Using GPT limits: %d chars, max %d messages", maxContextSize, maxMessages)
			}

			if contextSize > maxContextSize {
				log.Printf("[Sliding Window] Context size %d chars exceeds threshold %d chars, applying window",
					contextSize, maxContextSize)
				currentMessages = applySlidingWindow(currentMessages, maxMessages)

				// Recalculate after trimming
				contextSize = 0
				for _, msg := range currentMessages {
					contextSize += len(msg.Content)
				}
			}

			// Log iteration details
			log.Printf("[AI Processing] Iteration: %d, Request: %d chars, Context: %d chars, Tool calls so far: %d",
				iterationCount, contextSize, contextSize, toolCallCount)

			// DEBUG: Log context details before LLM API call to identify accumulation
			contextSize = calculateContextSize(currentMessages)
			toolResultPreview := getToolResultPreview(currentMessages, 200)
			log.Printf("[DEBUG Context] Before LLM call - Messages: %d, Total size: %d chars, Tool result preview: %s",
				len(currentMessages), contextSize, toolResultPreview)

			// SMART TOOL FILTERING: Reduce token usage by 70% by sending only relevant tools
			// This applies to ALL models to reduce rate limit issues
			originalToolCount := len(tools)
			relevantToolNames := filterToolsByWorkflowState(toolCallHistory)
			filteredTools := s.toolRegistry.GetFilteredToolsForLangChain(relevantToolNames)

			// Only apply smart filtering if it actually reduces the tool count
			// Keep all tools if filtering would include most of them anyway (>30 tools)
			if len(filteredTools) < originalToolCount && len(filteredTools) <= 30 {
				tools = filteredTools
				log.Printf("[Smart Tool Filter] Reduced from %d to %d tools (%.0f%% reduction) - Tool history: %d calls",
					originalToolCount, len(tools), 100.0*(1.0-float64(len(tools))/float64(originalToolCount)), len(toolCallHistory))

				// Log which tools are being sent for debugging
				toolNames := make([]string, 0, len(tools))
				for _, tool := range tools {
					if tool.Function != nil {
						toolNames = append(toolNames, tool.Function.Name)
					}
				}
				log.Printf("[Smart Tool Filter] Sending tools: %v", toolNames)
			} else {
				log.Printf("[Smart Tool Filter] Keeping all %d tools (filtering not beneficial)", originalToolCount)
			}

			// PHASE 3: PRESCRIPTIVE STATE MACHINE - Only allow ONE tool per workflow step
			// This forces ALL models into a linear workflow with zero ambiguity
			// Each step unlocks exactly ONE required tool - model has no choice but to follow the sequence
			// Applied to ALL models (not just Claude) to ensure consistent coordinator workflow
			if true { // Enable workflow enforcement for all models (GPT, Claude, Groq, etc.)
				step := workflowState["step"].(int)
				originalCount := len(tools)

				// Define allowed tools per step (WHITELIST approach)
				var allowedTools []string
				switch step {
				case 0: // Step 0: ONLY allow coordinator_list_human_tasks
					allowedTools = []string{"coordinator_list_human_tasks"}
				case 1: // Step 1: ONLY allow coordinator_create_human_task
					allowedTools = []string{"coordinator_create_human_task"}
				case 2: // Step 2: ONLY allow code_index_search
					allowedTools = []string{"code_index_search"}
				case 3: // Step 3: ONLY allow create_agent_task
					allowedTools = []string{"create_agent_task"}
				case 4: // Step 4: ONLY allow execute_subagent
					allowedTools = []string{"execute_subagent"}
				case 5: // Step 5: Workflow complete - NO TOOLS NEEDED (subagent is executing)
					// DO NOT provide list_agent_tasks - it causes hallucinated humanTaskIds
					// The subagent is executing in background, coordinator should inform user and STOP
					allowedTools = []string{}
				default:
					allowedTools = nil // Unknown step, allow all
				}

				// Filter tools using whitelist
				if allowedTools != nil {
					// Create set for O(1) lookup
					allowedSet := make(map[string]bool)
					for _, name := range allowedTools {
						allowedSet[name] = true
					}

					filteredTools := make([]llms.Tool, 0, len(allowedTools))
					for _, tool := range tools {
						if tool.Function != nil && allowedSet[tool.Function.Name] {
							filteredTools = append(filteredTools, tool)
						}
					}
					tools = filteredTools
				}

				if originalCount != len(tools) {
					log.Printf("[Phase 3 Prescriptive Filter] Step %d: Filtered %d → %d tools (allowed: %v)",
						step, originalCount, len(tools), allowedTools)
				}
			}

			// DEBUG: Log tools being passed to LLM
			log.Printf("[DEBUG Tools] Passing %d tools to LLM provider %s", len(tools), s.config.Provider)
			if len(tools) > 0 {
				toolNames := make([]string, 0, 3)
				for i := 0; i < len(tools) && i < 3; i++ {
					if tools[i].Function != nil {
						toolNames = append(toolNames, tools[i].Function.Name)
					}
				}
				log.Printf("[DEBUG Tools] Sample tools: %v", toolNames)
			}

			// WORKFLOW STATE GUIDANCE: Inject iteration and tool call progress into system prompt
			// This helps the LLM understand where it is in the workflow and avoid loops
			if len(currentMessages) > 0 && currentMessages[0].Role == "system" {
				// Build workflow state summary
				workflowGuidance := fmt.Sprintf(`

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WORKFLOW PROGRESS (Iteration %d)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Current Iteration: %d / %d
Total Tool Calls Made: %d / %d

⚠️  IMPORTANT: If you see "BLOCKED" or "NEXT:" in recent tool results,
    those messages contain CRITICAL guidance about what to do next.
    You MUST follow the "NEXT:" instructions - they tell you exactly which tool to call.

⚠️  AVOID LOOPS: Do NOT retry the same tool that was just BLOCKED.
    Follow the "NEXT:" guidance instead.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, iterationCount, iterationCount, s.config.MaxIterations, toolCallCount, maxToolCalls)

				// INJECT CAPTURED HUMAN TASK ID: If we captured a humanTaskId from coordinator_create_human_task,
				// inject it directly into the system prompt so Claude can use it without extraction
				if humanTaskID, ok := workflowState["humanTaskId"].(string); ok && humanTaskID != "" {
					taskIDGuidance := fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎯 CAPTURED TASK ID
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

The human task has been created/found. Use this EXACT taskId value:

**humanTaskId**: "%s"

When calling create_agent_task, use EXACTLY:
{
  "humanTaskId": "%s",
  ...
}

DO NOT generate or make up a different task ID. Use the value shown above.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, humanTaskID, humanTaskID)
					workflowGuidance += taskIDGuidance
					log.Printf("[Workflow State] Injected humanTaskId into system prompt: %s", humanTaskID)
				}

				// Append to existing system prompt (don't replace it)
				currentMessages[0].Content = currentMessages[0].Content + workflowGuidance
				log.Printf("[Workflow Guidance] Injected iteration %d progress into system prompt", iterationCount)
			}

			// Call provider with tools
			toolProvider := s.provider.(ToolCapableProvider)
			response, err := toolProvider.StreamChatWithTools(ctx, currentMessages, tools)
			if err != nil {
				// Check if this is a rate limit error and we have a fallback model configured
				if isRateLimitError(err) && s.config.FallbackModel != "" {
					log.Printf("[Rate Limit] Detected rate limit error, switching to fallback model: %s → %s",
						s.config.Model, s.config.FallbackModel)

					// Notify user about fallback
					fallbackMsg := fmt.Sprintf("\n\n⚠️  RATE LIMIT DETECTED: Primary model '%s' has hit its rate limit.\n"+
						"🔄 Automatically switching to fallback model '%s' (local, no rate limits)...\n\n",
						s.config.Model, s.config.FallbackModel)
					eventChan <- StreamEvent{Type: StreamEventToken, Content: fallbackMsg}

					// Save original model, provider, and API key
					originalModel := s.config.Model
					originalProvider := s.config.Provider
					originalAPIKey := s.config.APIKey

					// Switch to fallback model
					s.config.Model = s.config.FallbackModel

					// Switch to Anthropic provider for Claude models
					if strings.Contains(strings.ToLower(s.config.FallbackModel), "claude") {
						s.config.Provider = "anthropic"
						// Load Anthropic API key from environment
						anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
						if anthropicKey == "" {
							log.Printf("[ChatService] ERROR - RequestID: %s - ANTHROPIC_API_KEY not found in environment", requestID)
							eventChan <- StreamEvent{Type: StreamEventError,
								Error: "Rate limit error and ANTHROPIC_API_KEY not configured for fallback"}
							return
						}
						s.config.APIKey = anthropicKey
					}

					// Recreate provider with fallback model
					fallbackProvider, err := NewChatProvider(s.config)
					if err != nil {
						log.Printf("[ChatService] ERROR - RequestID: %s - Failed to create fallback provider: %v", requestID, err)
						eventChan <- StreamEvent{Type: StreamEventError,
							Error: fmt.Sprintf("Rate limit error and failed to switch to fallback model: %v", err)}
						return
					}

					// Update provider and mark as using fallback
					s.provider = fallbackProvider
					s.usingFallback = true

					// CRITICAL FIX: If we fell back to Claude, swap to Claude-optimized system prompt and circuit breakers
					if strings.Contains(strings.ToLower(s.config.FallbackModel), "claude") {
						log.Printf("[Rate Limit] Detected Claude fallback - swapping to Claude-optimized system prompt and thresholds")

						// Replace the system prompt in currentMessages with Claude-optimized version
						for i := range currentMessages {
							if currentMessages[i].Role == "system" {
								// Swap out FILE_PATHS_TO_USE fiction for real JSON examples
								claudePrompt := getClaudeSystemPrompt()

								// Extract session context from old prompt (the critical guidance section)
								oldPrompt := currentMessages[i].Content
								sessionContextStart := strings.Index(oldPrompt, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\nCRITICAL SYSTEM BEHAVIOR")
								if sessionContextStart > 0 {
									sessionContext := oldPrompt[sessionContextStart:]
									currentMessages[i].Content = claudePrompt + "\n\n" + sessionContext
								} else {
									currentMessages[i].Content = claudePrompt
								}

								log.Printf("[Rate Limit] Swapped system prompt: %d chars → %d chars", len(oldPrompt), len(currentMessages[i].Content))
								break
							}
						}

						// Re-calculate circuit breaker thresholds for Claude (more lenient)
						circuitBreakerThresholds = map[string]int{
							"read_file":         5, // Allow reading multiple files
							"write_file":        2, // Allow one retry for writes
							"list_directory":    4, // Allow exploring directories
							"bash":              5, // Allow command variations
							"code_index_search": 2, // Strict: one search + one retry max
							"create_agent_task": 4, // Allow retries for parameter refinement
							// Default for other tools: 6 attempts
						}
						log.Printf("[Circuit Breaker] Re-applied Claude thresholds after fallback")
					}

					// Retry with fallback provider
					if toolProvider, ok := s.provider.(ToolCapableProvider); ok {
						response, err = toolProvider.StreamChatWithTools(ctx, currentMessages, tools)
						if err != nil {
							log.Printf("[ChatService] ERROR - RequestID: %s - Fallback model also failed: %v", requestID, err)
							eventChan <- StreamEvent{Type: StreamEventError,
								Error: fmt.Sprintf("Both primary and fallback models failed: %v", err)}
							return
						}
						log.Printf("[Rate Limit] Successfully switched to fallback model '%s'", s.config.FallbackModel)

						// Send success notification
						successMsg := fmt.Sprintf("✅ Successfully switched to '%s'. Will automatically retry primary model on next request...\n\n", s.config.FallbackModel)
						eventChan <- StreamEvent{Type: StreamEventToken, Content: successMsg}

						// Note: We will automatically try to switch back to the primary model on the next request
						// The original configuration is saved in s.originalModel, s.originalProvider, s.originalAPIKey
						_, _, _ = originalModel, originalProvider, originalAPIKey // Mark as used to avoid compiler warning
					} else {
						log.Printf("[ChatService] ERROR - Fallback provider doesn't support tools")
						eventChan <- StreamEvent{Type: StreamEventError, Error: "Fallback provider doesn't support tools"}
						return
					}
				} else {
					// Not a rate limit error or no fallback configured - just fail
					log.Printf("[ChatService] ERROR - RequestID: %s - Tool call failed: %v", requestID, err)
					eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
					return
				}
			}

			// Stream response tokens
			var responseText string
			responseTokens := 0
			for chunk := range response.TextChannel {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
				responseText += chunk
				responseTokens++
			}

			// Log iteration response details
			log.Printf("[AI Processing] Iteration: %d complete, Response: %d tokens, Tool calls requested: %d",
				iterationCount, responseTokens, len(response.ToolCalls))

			// Check for tool calls
			if len(response.ToolCalls) == 0 {
				// No more tool calls, we're done
				log.Printf("[ChatService] Stream complete - RequestID: %s - Total iterations: %d, Tool calls: %d",
					requestID, iterationCount, toolCallCount)
				return
			}

			// Process each tool call
			for _, toolCall := range response.ToolCalls {
				toolCallCount++
				if toolCallCount > maxToolCalls {
					log.Printf("[ChatService] Max tool calls reached (%d) - RequestID: %s", maxToolCalls, requestID)
					eventChan <- StreamEvent{Type: StreamEventError, Error: fmt.Sprintf("maximum tool calls (%d) exceeded", maxToolCalls)}
					return
				}

				// Log tool request with arguments
				argsJSON, _ := json.Marshal(toolCall.Args)
				log.Printf("[Tool Request] AI requested tool '%s' with args: %s",
					toolCall.Name, string(argsJSON))

				// WORKFLOW VALIDATION: Check if this tool call is allowed in current workflow state
				var result ToolResult
				if s.usingFallback {
					allowed, blockMessage := validateWorkflowTool(toolCall.Name)
					if !allowed {
						log.Printf("[Workflow Enforcer] BLOCKED tool '%s' - %s", toolCall.Name, blockMessage)

						// Create a blocking error result so model understands the tool failed
						result = ToolResult{
							Name:       toolCall.Name,
							Output:     nil,
							Error:      blockMessage,
							DurationMs: 0,
						}

						// Send the error result to the model
						eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

						// Add tool result to message history as an error
						currentMessages = append(currentMessages, Message{
							Role:    "assistant",
							Content: fmt.Sprintf("I attempted to call '%s' but it was blocked.", toolCall.Name),
						})
						currentMessages = append(currentMessages, Message{
							Role:    "system",
							Content: fmt.Sprintf("TOOL ERROR from '%s': %s", toolCall.Name, blockMessage),
						})

						// Continue to next iteration - don't execute the blocked tool
						continue
					}
				}

				// Send tool call event
				eventChan <- StreamEvent{Type: StreamEventToolCall, ToolCall: &toolCall}

				// Generate signature for cache and circuit breaker
				signature := toolCallSignature(toolCall.Name, toolCall.Args)

				// Check tool result cache BEFORE execution
				cachedResult, found := resultCache.Get(signature)
				if found {
					// Use cached result - avoid redundant execution
					result = *cachedResult
					log.Printf("[Tool Cache HIT] Using cached result for '%s' - skipping execution", toolCall.Name)

					// Add cache hit notice to the result so AI knows it's cached
					cacheNotice := fmt.Sprintf("🔁 CACHED RESULT: You already called '%s' with these exact arguments. Using previous result instead of re-executing.", toolCall.Name)

					// Prepend cache notice to the output
					if outputMap, ok := result.Output.(map[string]interface{}); ok {
						// Clone the map to avoid mutating the cached version
						newOutput := make(map[string]interface{})
						for k, v := range outputMap {
							newOutput[k] = v
						}
						newOutput["_cacheNotice"] = cacheNotice
						result.Output = newOutput
					}
				} else {
					// Execute tool (no cached result available)
					// Inject humanTaskId from workflowState into context for auto-population
					toolCtx := ctx
					if humanTaskID, ok := workflowState["humanTaskId"].(string); ok && humanTaskID != "" {
						toolCtx = context.WithValue(ctx, "lastHumanTaskId", humanTaskID)
					}
					result = s.toolRegistry.ExecuteToolCall(toolCtx, toolCall)
					log.Printf("[Tool Cache MISS] Executed '%s' - storing result in cache", toolCall.Name)

					// Store in cache for future duplicate calls
					resultCache.Set(signature, &result)
				}

				// GPT FILE PATH VALIDATION: For code_index_search, validate file paths before proceeding
				// This ONLY applies to GPT models, NOT Claude (Claude has its own file validation in create_agent_task)
				if toolCall.Name == "code_index_search" && result.Error == "" && !s.usingFallback {
					isGPTModel := !strings.Contains(strings.ToLower(s.config.Model), "claude")

					if isGPTModel {
						// Extract file paths from search results
						filePaths := extractFilePathsFromCodeIndexResult(result)

						if len(filePaths) > 0 {
							validPaths, invalidPaths := validateFilePaths(filePaths)

							log.Printf("[GPT Path Validator] code_index_search returned %d paths: %d valid, %d invalid",
								len(filePaths), len(validPaths), len(invalidPaths))

							if len(invalidPaths) > 0 {
								// Some paths are invalid - check if this is already a retry
								retryAttemptKey := fmt.Sprintf("code_index_search_retry_%v", toolCall.Args["query"])
								hasRetried := pathValidationRetries[retryAttemptKey]

								if hasRetried {
									// Already retried once - stop execution with clear error
									log.Printf("[GPT Path Validator] code_index_search retry also returned invalid paths - stopping execution")

									errorMsg := fmt.Sprintf("❌ CRITICAL ERROR: code_index_search returned invalid file paths (files don't exist on filesystem):\n\n"+
										"INVALID PATHS:\n"+
										"%s\n\n"+
										"🛑 Even after retrying the search, the code index is returning paths to files that don't exist.\n"+
										"✅ This means:\n"+
										"   1. The code index may be stale or out of sync with the filesystem\n"+
										"   2. The files may have been moved or deleted\n"+
										"   3. The search query may be finding old/archived files\n\n"+
										"🔍 NEXT STEPS:\n"+
										"   - Ask the user to verify the file locations\n"+
										"   - Try a different search query\n"+
										"   - Check if files exist in a different directory\n"+
										"   - Consider re-indexing the codebase\n\n"+
										"DO NOT proceed with create_agent_task using these invalid paths!",
										strings.Join(invalidPaths, "\n"))

									eventChan <- StreamEvent{Type: StreamEventError, Error: errorMsg}
									return
								} else {
									// First time encountering invalid paths - try automatic retry with refined query
									log.Printf("[GPT Path Validator] First invalid path detection - attempting automatic retry with refined query")

									// Mark this as a retry attempt
									pathValidationRetries[retryAttemptKey] = true

									// Send warning to user about automatic retry
									warningMsg := fmt.Sprintf("\n\n⚠️  FILE PATH VALIDATION WARNING:\n"+
										"code_index_search returned %d invalid file paths (files don't exist).\n"+
										"🔄 Automatically retrying search with refined query...\n\n"+
										"Invalid paths found:\n%s\n\n",
										len(invalidPaths), strings.Join(invalidPaths, "\n"))
									eventChan <- StreamEvent{Type: StreamEventToken, Content: warningMsg}

									// Modify the tool result to indicate validation failure
									// Inject validation warning into result so GPT knows to refine the search
									if outputMap, ok := result.Output.(map[string]interface{}); ok {
										outputMap["_validationWarning"] = fmt.Sprintf(
											"⚠️ VALIDATION FAILED: %d out of %d file paths are INVALID (files don't exist on filesystem). "+
											"You MUST retry code_index_search with a different query to find the CORRECT file paths. "+
											"DO NOT proceed with create_agent_task using these invalid paths. "+
											"Invalid paths: %s",
											len(invalidPaths), len(filePaths), strings.Join(invalidPaths, "\n"))
										outputMap["invalidPaths"] = invalidPaths
										outputMap["validPaths"] = validPaths
										result.Output = outputMap
									}

									log.Printf("[GPT Path Validator] Injected validation warning into result - GPT should retry search")
								}
							} else {
								log.Printf("[GPT Path Validator] All paths valid - proceeding normally")
							}
						}
					}
				}

				// TASK ID VALIDATION: For create_agent_task, validate humanTaskId exists before proceeding
				if toolCall.Name == "create_agent_task" && result.Error == "" {
					// Extract humanTaskId from arguments
					var humanTaskId string
					if id, ok := toolCall.Args["humanTaskId"].(string); ok {
						humanTaskId = id
					}

					// Validate humanTaskId if provided
					if humanTaskId != "" {
						// Call coordinator_list_human_tasks to get all tasks
						listTasksCall := ToolCall{
							ID:   "taskid_validation",
							Name: "coordinator_list_human_tasks",
							Args: map[string]interface{}{},
						}
						listResult := s.toolRegistry.ExecuteToolCall(ctx, listTasksCall)

						taskExists := false
						if listResult.Error == "" {
							if outputMap, ok := listResult.Output.(map[string]interface{}); ok {
								if tasks, ok := outputMap["tasks"].([]interface{}); ok {
									for _, task := range tasks {
										if taskMap, ok := task.(map[string]interface{}); ok {
											if taskId, ok := taskMap["taskId"].(string); ok && taskId == humanTaskId {
												taskExists = true
												break
											}
										}
									}
								}
							}
						}

						if !taskExists {
							// TaskId is invalid - increment attempt counter
							taskIdValidationAttempts++
							log.Printf("[TaskId Validator] Invalid humanTaskId '%s' - Attempt %d/3", humanTaskId, taskIdValidationAttempts)

							if taskIdValidationAttempts >= 3 {
								// After 3 attempts, stop execution with clear error
								log.Printf("[TaskId Validator] Failed 3 times - stopping execution")
								errorMsg := fmt.Sprintf("❌ CRITICAL ERROR: create_agent_task called with INVALID humanTaskId 3 times.\n\n"+
									"INVALID humanTaskId PROVIDED: '%s'\n\n"+
									"🛑 The humanTaskId you are providing DOES NOT EXIST in the database.\n"+
									"✅ This means:\n"+
									"   1. You are hallucinating or generating the task ID instead of copying it from tool responses\n"+
									"   2. The task ID may have been typed incorrectly\n"+
									"   3. The task may have been deleted\n\n"+
									"🔍 CORRECT APPROACH:\n"+
									"   1. Call coordinator_list_human_tasks to see ALL existing tasks\n"+
									"   2. Find the task that matches the user's request\n"+
									"   3. COPY the EXACT 'taskId' field from the task object\n"+
									"   4. Use that EXACT UUID as 'humanTaskId' when calling create_agent_task\n\n"+
									"❌ DO NOT:\n"+
									"   - Generate UUIDs yourself\n"+
									"   - Use descriptive names like 'add-feature' or 'fix-bug'\n"+
									"   - Try to guess or construct the task ID\n\n"+
									"⚠️ Execution stopped after 3 invalid attempts. Please review the instructions above and try again.",
									humanTaskId)

								eventChan <- StreamEvent{Type: StreamEventError, Error: errorMsg}
								return
							}

							// First or second attempt - inject warning and ask model to list tasks
							warningMsg := fmt.Sprintf("\n\n⚠️  TASK ID VALIDATION ERROR (Attempt %d/3):\n"+
								"The humanTaskId '%s' DOES NOT EXIST in the database.\n"+
								"🔄 You MUST call coordinator_list_human_tasks to get the correct taskId.\n\n",
								taskIdValidationAttempts, humanTaskId)
							eventChan <- StreamEvent{Type: StreamEventToken, Content: warningMsg}

							// Replace the result with an error result
							result = ToolResult{
								ID:   result.ID,
								Name: "create_agent_task",
								Args: result.Args,
								Output: map[string]interface{}{
									"_validationError": "BLOCKED",
									"_reason": fmt.Sprintf("Invalid humanTaskId: '%s' does not exist", humanTaskId),
									"NEXT":    "Call coordinator_list_human_tasks to see all tasks, find the correct task, and COPY its exact 'taskId' field",
								},
								Error: fmt.Sprintf("❌ BLOCKED: humanTaskId '%s' is INVALID (does not exist). "+
									"You MUST call coordinator_list_human_tasks to get all tasks and find the EXACT taskId. "+
									"DO NOT hallucinate or generate task IDs. COPY the exact UUID from the tool response. "+
									"This is attempt %d/3. After 3 failures, execution will stop.",
									humanTaskId, taskIdValidationAttempts),
							}

							log.Printf("[TaskId Validator] Blocked create_agent_task - injected error asking model to list tasks")
						} else {
							log.Printf("[TaskId Validator] humanTaskId '%s' is valid - proceeding", humanTaskId)
						}
					}
				}

				// Send tool result event (full result to client for display)
				eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

				// Track tool execution in history for smart filtering (keep last 20)
				toolCallHistory = append(toolCallHistory, result)
				if len(toolCallHistory) > 20 {
					toolCallHistory = toolCallHistory[1:] // Remove oldest
				}

				// CRITICAL: Track failed tool calls separately - stop immediately on retry of failed operation
				if result.Error != "" {
					failedToolCalls[signature]++
					if failedToolCalls[signature] >= 2 {
						// Second failure with same args - stop immediately!
						log.Printf("[Circuit Breaker - Failed Tool] Tool '%s' failed twice with identical arguments - stopping", toolCall.Name)
						eventChan <- StreamEvent{
							Type: StreamEventError,
							Error: fmt.Sprintf("❌ CRITICAL ERROR: Tool '%s' failed TWICE with identical arguments. Error: %s\n\n"+
								"🛑 You are retrying a FAILED operation. This will never work!\n"+
								"✅ Try a DIFFERENT approach:\n"+
								"   - If file not found: check directory listing or search results for the ACTUAL file name\n"+
								"   - If path wrong: try different path or create the file\n"+
								"   - If tool incompatible: use a different tool\n\n"+
								"DO NOT retry the same failed operation again!", toolCall.Name, result.Error),
						}
						return
					}
				}

				// Circuit breaker: check for repeated tool calls AND warn the AI
				recentToolCalls = append(recentToolCalls, signature)
				if len(recentToolCalls) > 10 {
					recentToolCalls = recentToolCalls[1:]
				}

				// Count how many times this exact tool+args was called in ALL history
				totalCount := 0
				for _, sig := range recentToolCalls {
					if sig == signature {
						totalCount++
					}
				}

				// Get tool-specific threshold
				threshold := circuitBreakerThresholds[toolCall.Name]
				if threshold == 0 {
					threshold = 4 // Default threshold
				}

				// Progressive warnings to AI (inject into context so AI sees them)
				var loopWarning string
				if totalCount == 2 {
					// First duplicate - gentle warning
					loopWarning = fmt.Sprintf("⚠️  WARNING: You already called '%s' with these exact arguments 1 time before. You should use the result from the previous call instead of repeating the same operation.", toolCall.Name)
				} else if totalCount == 3 && threshold > 3 {
					// Second duplicate - stronger warning (only if threshold allows)
					loopWarning = fmt.Sprintf("🔁 LOOP DETECTED: You called '%s' with identical arguments 2 times already. You are stuck in a loop! Use previous results or try a DIFFERENT approach - do NOT call this tool again with the same arguments.", toolCall.Name)
				} else if totalCount >= threshold {
					// Threshold reached - trigger circuit breaker
					log.Printf("[Circuit Breaker] Tool '%s' called %d times (threshold: %d) - stopping infinite loop", toolCall.Name, totalCount, threshold)
					eventChan <- StreamEvent{
						Type:  StreamEventError,
						Error: fmt.Sprintf("Circuit breaker triggered: tool '%s' called repeatedly (%d times) with identical arguments. The AI is stuck in an infinite loop and cannot complete this task.", toolCall.Name, totalCount),
					}
					return
				}

				// Log tool execution with comprehensive response data (Claude optimization)
				if result.Error != "" {
					log.Printf("[ChatService] Tool '%s' failed - RequestID: %s - Error: %s - Duration: %dms",
						result.Name, requestID, result.Error, result.DurationMs)
					// Log complete error response for analysis
					argsJSON, _ := json.Marshal(toolCall.Args)
					log.Printf("[Tool Response - ERROR] Tool: %s | Args: %s | Error: %s",
						result.Name, string(argsJSON), result.Error)
				} else {
					log.Printf("[ChatService] Tool '%s' succeeded - RequestID: %s - Duration: %dms",
						result.Name, requestID, result.DurationMs)
					// Log complete success response for analysis
					argsJSON, _ := json.Marshal(toolCall.Args)
					outputJSON, _ := json.Marshal(result.Output)
					log.Printf("[Tool Response - SUCCESS] Tool: %s | Args: %s | Output: %s",
						result.Name, string(argsJSON), string(outputJSON))
				}

				// Add assistant response to history (brief)
				currentMessages = append(currentMessages, Message{
					Role:    "assistant",
					Content: responseText,
				})

				// Add tool result to message history
				var toolResultMsg string
				if result.Error != "" {
					// Check if this is a permanent failure that shouldn't be retried
					errorLower := strings.ToLower(result.Error)
					isPermanentError := strings.Contains(errorLower, "requires mcp endpoint") ||
						strings.Contains(errorLower, "not supported") ||
						strings.Contains(errorLower, "cannot be used") ||
						strings.Contains(errorLower, "requires direct mcp")

					if isPermanentError {
						toolResultMsg = fmt.Sprintf("PERMANENT ERROR - Tool '%s' cannot be used in this context: %s. DO NOT retry this tool - it will not work.", result.Name, result.Error)
					} else {
						toolResultMsg = fmt.Sprintf("Tool '%s' error: %s", result.Name, result.Error)
					}
				} else {
					// Marshal output to JSON for context
					outputJSON, err := json.Marshal(result.Output)
					if err != nil {
						toolResultMsg = fmt.Sprintf("Tool '%s' result: <serialization error: %v>", result.Name, err)
					} else {
						// Check if output contains an error field (common pattern for tools returning error in response)
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if errorField, hasError := outputMap["error"]; hasError && errorField != nil {
								errorStr := fmt.Sprintf("%v", errorField)
								toolResultMsg = fmt.Sprintf("PERMANENT ERROR - Tool '%s' returned error: %s. DO NOT retry this tool.", result.Name, errorStr)
							} else {
								toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))
							}
						} else {
							toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))
						}
					}
				}

				// CRITICAL FIX: If we generated a loop warning, EMBED it in JSON result
				if loopWarning != "" {
					log.Printf("[Loop Detection] %s", loopWarning)

					// Send warning as a visible message to the user
					eventChan <- StreamEvent{Type: StreamEventToken, Content: "\n\n" + loopWarning + "\n\n"}

					// Try to embed warning into the JSON result instead of prepending as text
					if strings.HasPrefix(toolResultMsg, fmt.Sprintf("Tool '%s' result: ", result.Name)) {
						jsonPart := strings.TrimPrefix(toolResultMsg, fmt.Sprintf("Tool '%s' result: ", result.Name))
						var resultData map[string]interface{}
						if err := json.Unmarshal([]byte(jsonPart), &resultData); err == nil {
							resultData["_loopWarning"] = loopWarning
							if newJSON, err := json.Marshal(resultData); err == nil {
								toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(newJSON))
							} else {
								// fallback: text prepend if reserialization fails
								toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
							}
						} else {
							// fallback: text prepend if JSON parse fails
							toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
						}
					} else {
						// fallback: if message doesn’t match JSON result format
						toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
					}
				}

				// CRITICAL FIX: Truncate tool results that are too large to prevent token limit errors
				// Individual tool results can be HUGE (e.g., bash ls -R output = 1.98MB)
				// Even if sliding window triggers, if one recent message is huge, it doesn't help
				// For fallback model (Haiku), use higher limit to preserve more context
				maxToolResultSize := 10000 // 10KB per tool result (default)
				if s.usingFallback {
					maxToolResultSize = 30000 // 30KB for fallback model - preserve more context
				}
				if len(toolResultMsg) > maxToolResultSize {
					originalSize := len(toolResultMsg)
					// Keep first portion and add truncation notice
					truncatedSize := maxToolResultSize - 500
					toolResultMsg = toolResultMsg[:truncatedSize] + fmt.Sprintf("\n\n... [TRUNCATED: Result was %d chars, showing first %d chars to prevent token limit. If you need more, use a more specific query or process the data in smaller chunks.] ...", originalSize, truncatedSize)
					log.Printf("[Tool Result Truncation] Truncated tool '%s' result from %d to %d chars to prevent token limit",
						result.Name, originalSize, len(toolResultMsg))
				}

				currentMessages = append(currentMessages, Message{
					Role:    "system",
					Content: toolResultMsg,
				})

				// WORKFLOW STATE UPDATE: Update workflow state after successful tool execution
				// Apply to ALL models to match prescriptive filter behavior (line 714)
				if result.Error == "" {
					switch toolCall.Name {
					case "coordinator_list_human_tasks":
						if workflowState["step"].(int) == 0 {
							workflowState["step"] = 1
							log.Printf("[Workflow State] Step 1 complete: listed tasks")
						}

					case "coordinator_create_human_task":
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if taskID, hasTaskID := outputMap["taskId"].(string); hasTaskID && taskID != "" {
								workflowState["step"] = 2
								workflowState["humanTaskId"] = taskID
								log.Printf("[Workflow State] Step 2 complete: created human task %s", taskID)
							} else if similarTasksFound, _ := outputMap["similarTasksFound"].(bool); similarTasksFound {
								// Case 2: Similar task found - use existing task instead of creating new one
								if similarTasks, ok := outputMap["similarTasks"].([]interface{}); ok && len(similarTasks) > 0 {
									if firstTask, ok := similarTasks[0].(map[string]interface{}); ok {
										if existingTaskID, ok := firstTask["taskId"].(string); ok && existingTaskID != "" {
											workflowState["step"] = 2
											workflowState["humanTaskId"] = existingTaskID
											log.Printf("[Workflow State] Step 2 complete: using existing similar task %s", existingTaskID)
										}
									}
								}
							}
						}

					case "code_index_search":
						workflowState["step"] = 3
						workflowState["searchCompleted"] = true
						log.Printf("[Workflow State] Step 3 complete: code search done")

					case "create_agent_task":
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if agentTaskID, hasAgentTaskID := outputMap["taskId"].(string); hasAgentTaskID && agentTaskID != "" {
								workflowState["step"] = 4
								workflowState["agentTaskId"] = agentTaskID
								log.Printf("[Workflow State] Step 4 complete: created agent task %s", agentTaskID)
							}
						}

					case "execute_subagent":
						workflowState["step"] = 5
						log.Printf("[Workflow State] Step 5 complete: subagent launched")
					}
				}

				// FALLBACK MODEL ENHANCEMENT: Add explicit state tracking for workflow comprehension
				// Haiku (smaller model) benefits from explicit guidance on workflow state and next steps
				if s.usingFallback {
					stateGuidance := s.generateWorkflowStateGuidance(toolCall.Name, result, toolCallCount)
					if stateGuidance != "" {
						log.Printf("[Fallback State Tracking] Injecting workflow guidance after tool '%s'", toolCall.Name)
						currentMessages = append(currentMessages, Message{
							Role:    "system",
							Content: stateGuidance,
						})
					}
				}

				log.Printf("[AI Processing] Context after tool %d: %d messages, %d total chars",
					toolCallCount, len(currentMessages), func() int {
						sum := 0
						for _, m := range currentMessages {
							sum += len(m.Content)
						}
						return sum
					}())
			}
		}

		// Max iterations reached
		log.Printf("[ChatService] Max tool calls reached - RequestID: %s - Total iterations: %d, Tool calls: %d",
			requestID, iterationCount, toolCallCount)
	}()

	return eventChan, nil
}

// StreamChatWithToolsFiltered sends messages to AI provider with restricted tool access
// This is used for subagents to prevent them from calling coordinator tools
// Only the specified tools in allowedToolNames will be available to the AI
func (s *ChatService) StreamChatWithToolsFiltered(ctx context.Context, messages []Message, maxToolCalls int, allowedToolNames []string) (<-chan StreamEvent, error) {
	identity := s.getIdentityFromContext(ctx)
	requestID := s.getRequestIDFromContext(ctx)

	// Log the request
	if identity != nil {
		log.Printf("[ChatService] Tool-filtered request from %s (%s) - RequestID: %s - Provider: %s Model: %s - Allowed tools: %v",
			identity.Name, identity.Type, requestID, s.config.Provider, s.config.Model, allowedToolNames)
	} else {
		log.Printf("[ChatService] Tool-filtered request (no identity) - RequestID: %s - Provider: %s Model: %s - Allowed tools: %v",
			requestID, s.config.Provider, s.config.Model, allowedToolNames)
	}

	// Validate messages
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	// Default max tool calls to prevent loops
	if maxToolCalls <= 0 {
		maxToolCalls = 5
	}

	// Create output channel for events
	eventChan := make(chan StreamEvent, 100)

	// Get FILTERED tools for LangChain (only allowed tools)
	tools := s.toolRegistry.GetFilteredToolsForLangChain(allowedToolNames)

	log.Printf("[ChatService] Filtered %d tools for subagent (from allowlist of %d tools)", len(tools), len(allowedToolNames))

	// Check if provider supports tools
	supportsTools := false
	if toolProvider, ok := s.provider.(ToolCapableProvider); ok {
		supportsTools = toolProvider.SupportsTools()
	}

	if !supportsTools || len(tools) == 0 {
		// Fallback to text-only streaming
		log.Printf("[ChatService] Provider doesn't support tools or no tools registered - RequestID: %s", requestID)
		go func() {
			defer close(eventChan)
			textChan, err := s.provider.StreamChat(ctx, messages)
			if err != nil {
				eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
				return
			}
			for chunk := range textChan {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
			}
		}()
		return eventChan, nil
	}

	// Start tool-enabled streaming
	go func() {
		defer close(eventChan)

		toolCallCount := 0
		iterationCount := 0
		currentMessages := append([]Message{}, messages...) // Copy messages

		// Tool result cache: prevent duplicate tool executions
		resultCache := NewToolResultCache()

		// Circuit breaker: track recent tool calls to detect infinite loops
		recentToolCalls := make([]string, 0, 10)
		failedToolCalls := make(map[string]int)        // Track failed attempts separately
		// pathValidationRetries not needed in fallback model (Claude handles its own validation)
		toolCallSignature := func(name string, args map[string]interface{}) string {
			argsJSON, _ := json.Marshal(args)
			return fmt.Sprintf("%s(%s)", name, string(argsJSON))
		}

		// WORKFLOW STATE ENFORCEMENT: Track coordinator workflow progress (fallback model only)
		workflowState := map[string]interface{}{
			"step":            0,     // 0=initial, 1=listed, 2=created, 3=searched, 4=agent_task, 5=done
			"humanTaskId":     "",    // Store taskId from step 2
			"searchCompleted": false, // Prevent multiple searches
			"agentTaskId":     "",    // Store agentTaskId from step 4
		}

		// Function to validate workflow tool calls (only enforced for fallback model)
		validateWorkflowTool := func(toolName string) (bool, string) {
			if !s.usingFallback {
				return true, "" // No enforcement for primary model
			}

			step := workflowState["step"].(int)
			humanTaskId := workflowState["humanTaskId"].(string)
			searchCompleted := workflowState["searchCompleted"].(bool)

			switch toolName {
			case "coordinator_list_human_tasks":
				if step == 0 || step == 1 || step == 2 {
					// Allow listing tasks at any early step to retrieve exact taskId
					return true, ""
				}
				return false, "❌ BLOCKED: You already have taskId. NEXT: Call appropriate tool for current step."

			case "coordinator_create_human_task":
				if step == 1 && humanTaskId == "" {
					return true, ""
				}
				if humanTaskId != "" {
					// State-aware guidance based on workflow progress
					if searchCompleted {
						return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'create_agent_task'.", humanTaskId)
					}
					return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'code_index_search'.", humanTaskId)
				}
				return false, "❌ BLOCKED: Call 'coordinator_list_human_tasks' first."

			case "code_index_search":
				if step == 2 && humanTaskId != "" && !searchCompleted {
					return true, ""
				}
				if searchCompleted {
					return false, "❌ BLOCKED: Search done. NEXT: Call 'create_agent_task'."
				}
				return false, "❌ BLOCKED: Create human task first."

			case "create_agent_task":
				if step == 3 && humanTaskId != "" && searchCompleted {
					return true, ""
				}
				return false, "❌ BLOCKED: Run 'code_index_search' first to get file paths."

			case "execute_subagent":
				agentTaskId := workflowState["agentTaskId"].(string)
				if step == 4 && agentTaskId != "" {
					return true, ""
				}
				return false, "❌ BLOCKED: Call 'create_agent_task' first."
			}

			return true, ""
		}

		// Per-tool circuit breaker thresholds (max duplicate attempts before stopping)
		// Claude models get higher thresholds as they're better at adapting
		isClaudeModel := strings.Contains(strings.ToLower(s.config.Model), "claude") ||
			strings.Contains(strings.ToLower(s.config.Provider), "anthropic")

		var circuitBreakerThresholds map[string]int
		if isClaudeModel {
			// Claude-optimized thresholds: More lenient to allow legitimate multi-file operations
			circuitBreakerThresholds = map[string]int{
				"read_file":         5, // Allow reading multiple files
				"write_file":        2, // Allow one retry for writes
				"list_directory":    4, // Allow exploring directories
				"bash":              5, // Allow command variations
				"code_index_search": 2, // Strict: one search + one retry max
				"create_agent_task": 4, // Allow retries for parameter refinement
				// Default for other tools: 6 attempts
			}
			log.Printf("[Circuit Breaker] Using Claude-optimized thresholds (more lenient)")
		} else {
			// GPT thresholds: More conservative
			circuitBreakerThresholds = map[string]int{
				"read_file":         2, // Stop after 2 attempts (only 1 duplicate allowed)
				"write_file":        1, // Never allow duplicate writes
				"list_directory":    2, // Stop after 2 attempts
				"bash":              3, // Allow more for command variations
				"code_index_search": 3, // Allow query refinement
				// Default for other tools: 4 attempts
			}
			log.Printf("[Circuit Breaker] Using GPT thresholds (conservative)")
		}

		for toolCallCount < maxToolCalls && iterationCount < s.config.MaxIterations {
			iterationCount++

			// Calculate context size BEFORE applying sliding window
			contextSize := 0
			for _, msg := range currentMessages {
				contextSize += len(msg.Content)
			}

			// Apply sliding window BEFORE context exceeds model's token limit
			// Claude: 200K tokens (≈800KB text) - use 150KB threshold
			// GPT: 32K tokens (≈128KB text) - use 40KB threshold to be safe
			var maxContextSize int
			var maxMessages int
			if isClaudeModel {
				maxContextSize = 150000 // 150KB for Claude (≈37K tokens, leaves room for output)
				maxMessages = 20        // Keep more messages for Claude
			} else {
				maxContextSize = 40000 // 40KB for GPT (≈10K tokens)
				maxMessages = 6        // Conservative for GPT
			}

			if contextSize > maxContextSize {
				log.Printf("[Sliding Window - Filtered] Context size %d chars exceeds threshold %d chars, applying window",
					contextSize, maxContextSize)
				currentMessages = applySlidingWindow(currentMessages, maxMessages)

				// Recalculate after trimming
				contextSize = 0
				for _, msg := range currentMessages {
					contextSize += len(msg.Content)
				}
			}

			// Log iteration details
			log.Printf("[AI Processing - Filtered Tools] Iteration: %d, Request: %d chars, Context: %d chars, Tool calls so far: %d",
				iterationCount, contextSize, contextSize, toolCallCount)

			// DEBUG: Log context details before LLM API call
			contextSize = calculateContextSize(currentMessages)
			toolResultPreview := getToolResultPreview(currentMessages, 200)
			log.Printf("[DEBUG Context - Filtered] Before LLM call - Messages: %d, Total size: %d chars, Tool result preview: %s",
				len(currentMessages), contextSize, toolResultPreview)

			// Call provider with FILTERED tools
			toolProvider := s.provider.(ToolCapableProvider)
			response, err := toolProvider.StreamChatWithTools(ctx, currentMessages, tools)
			if err != nil {
				log.Printf("[ChatService] ERROR - RequestID: %s - Tool call failed: %v", requestID, err)
				eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
				return
			}

			// Stream response tokens
			var responseText string
			responseTokens := 0
			for chunk := range response.TextChannel {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
				responseText += chunk
				responseTokens++
			}

			// Log iteration response details
			log.Printf("[AI Processing - Filtered] Iteration: %d complete, Response: %d tokens, Tool calls requested: %d",
				iterationCount, responseTokens, len(response.ToolCalls))

			// Check for tool calls
			if len(response.ToolCalls) == 0 {
				// No more tool calls, we're done
				log.Printf("[ChatService - Filtered] Stream complete - RequestID: %s - Total iterations: %d, Tool calls: %d",
					requestID, iterationCount, toolCallCount)
				return
			}

			// Process each tool call
			for _, toolCall := range response.ToolCalls {
				toolCallCount++
				if toolCallCount > maxToolCalls {
					log.Printf("[ChatService - Filtered] Max tool calls reached (%d) - RequestID: %s", maxToolCalls, requestID)
					eventChan <- StreamEvent{Type: StreamEventError, Error: fmt.Sprintf("maximum tool calls (%d) exceeded", maxToolCalls)}
					return
				}

				// Log tool request with arguments
				argsJSON, _ := json.Marshal(toolCall.Args)
				log.Printf("[Tool Request - Filtered] AI requested tool '%s' with args: %s",
					toolCall.Name, string(argsJSON))

				// WORKFLOW VALIDATION: Check if this tool call is allowed in current workflow state
				var result ToolResult
				if s.usingFallback {
					allowed, blockMessage := validateWorkflowTool(toolCall.Name)
					if !allowed {
						log.Printf("[Workflow Enforcer - Filtered] BLOCKED tool '%s' - %s", toolCall.Name, blockMessage)

						// Create a blocking error result so model understands the tool failed
						result = ToolResult{
							Name:       toolCall.Name,
							Output:     nil,
							Error:      blockMessage,
							DurationMs: 0,
						}

						// Send the error result to the model
						eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

						// Add tool result to message history as an error
						currentMessages = append(currentMessages, Message{
							Role:    "assistant",
							Content: fmt.Sprintf("I attempted to call '%s' but it was blocked.", toolCall.Name),
						})
						currentMessages = append(currentMessages, Message{
							Role:    "system",
							Content: fmt.Sprintf("TOOL ERROR from '%s': %s", toolCall.Name, blockMessage),
						})

						// Continue to next iteration - don't execute the blocked tool
						continue
					}
				}

				// Send tool call event
				eventChan <- StreamEvent{Type: StreamEventToolCall, ToolCall: &toolCall}

				// Generate signature for cache and circuit breaker
				signature := toolCallSignature(toolCall.Name, toolCall.Args)

				// Check tool result cache BEFORE execution
				cachedResult, found := resultCache.Get(signature)
				if found {
					// Use cached result - avoid redundant execution
					result = *cachedResult
					log.Printf("[Tool Cache HIT] Using cached result for '%s' - skipping execution", toolCall.Name)

					// Add cache hit notice to the result
					cacheNotice := fmt.Sprintf("🔁 CACHED RESULT: You already called '%s' with these exact arguments. Using previous result instead of re-executing.", toolCall.Name)
					if outputMap, ok := result.Output.(map[string]interface{}); ok {
						newOutput := make(map[string]interface{})
						for k, v := range outputMap {
							newOutput[k] = v
						}
						newOutput["_cacheNotice"] = cacheNotice
						result.Output = newOutput
					}
				} else {
					// Execute tool (no cached result available)
					// Inject humanTaskId from workflowState into context for auto-population
					toolCtx := ctx
					if humanTaskID, ok := workflowState["humanTaskId"].(string); ok && humanTaskID != "" {
						toolCtx = context.WithValue(ctx, "lastHumanTaskId", humanTaskID)
					}
					result = s.toolRegistry.ExecuteToolCall(toolCtx, toolCall)
					log.Printf("[Tool Cache MISS] Executed '%s' - storing result in cache", toolCall.Name)

					// Store in cache for future duplicate calls
					resultCache.Set(signature, &result)
				}

				// Send tool result event
				eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

				// Log tool execution with comprehensive response data (Claude optimization)
				if result.Error != "" {
					log.Printf("[ChatService - Filtered] Tool '%s' failed - RequestID: %s - Error: %s - Duration: %dms",
						result.Name, requestID, result.Error, result.DurationMs)
					// Log complete error response for analysis
					argsJSON, _ := json.Marshal(toolCall.Args)
					log.Printf("[Tool Response - ERROR - Filtered] Tool: %s | Args: %s | Error: %s",
						result.Name, string(argsJSON), result.Error)
				} else {
					log.Printf("[ChatService - Filtered] Tool '%s' succeeded - RequestID: %s - Duration: %dms",
						result.Name, requestID, result.DurationMs)
					// Log complete success response for analysis
					argsJSON, _ := json.Marshal(toolCall.Args)
					outputJSON, _ := json.Marshal(result.Output)
					log.Printf("[Tool Response - SUCCESS - Filtered] Tool: %s | Args: %s | Output: %s",
						result.Name, string(argsJSON), string(outputJSON))
				}

				// CRITICAL: Track failed tool calls separately
				if result.Error != "" {
					failedToolCalls[signature]++
					if failedToolCalls[signature] >= 2 {
						log.Printf("[Circuit Breaker - Failed Tool] Tool '%s' failed twice with identical arguments - stopping", toolCall.Name)
						eventChan <- StreamEvent{
							Type: StreamEventError,
							Error: fmt.Sprintf("❌ CRITICAL ERROR: Tool '%s' failed TWICE with identical arguments. Error: %s\n\n"+
								"🛑 You are retrying a FAILED operation. This will never work!\n"+
								"✅ Try a DIFFERENT approach - DO NOT retry the same failed operation!", toolCall.Name, result.Error),
						}
						return
					}
				}

				// Circuit breaker: check for repeated tool calls
				recentToolCalls = append(recentToolCalls, signature)
				if len(recentToolCalls) > 10 {
					recentToolCalls = recentToolCalls[1:]
				}

				// Count duplicates
				totalCount := 0
				for _, sig := range recentToolCalls {
					if sig == signature {
						totalCount++
					}
				}

				// Get tool-specific threshold
				threshold := circuitBreakerThresholds[toolCall.Name]
				if threshold == 0 {
					threshold = 4 // Default threshold
				}

				// Progressive warnings
				var loopWarning string
				if totalCount == 2 {
					loopWarning = fmt.Sprintf("⚠️  WARNING: You already called '%s' with these exact arguments 1 time before. Use the previous result instead of repeating.", toolCall.Name)
				} else if totalCount == 3 && threshold > 3 {
					loopWarning = fmt.Sprintf("🔁 LOOP DETECTED: You called '%s' with identical arguments 2 times already. You are stuck in a loop! Use previous results or try a DIFFERENT approach.", toolCall.Name)
				} else if totalCount >= threshold {
					log.Printf("[Circuit Breaker] Tool '%s' called %d times (threshold: %d) - stopping infinite loop", toolCall.Name, totalCount, threshold)
					eventChan <- StreamEvent{
						Type:  StreamEventError,
						Error: fmt.Sprintf("Circuit breaker triggered: tool '%s' called repeatedly (%d times) with identical arguments. The AI is stuck in an infinite loop.", toolCall.Name, totalCount),
					}
					return
				}

				// CRITICAL FIX: Add tool_call message with tool_use block (not plain text)
				// This ensures the model has conversation memory of making the tool call
				// Role must be "tool_call" to match provider.go:429 check
				currentMessages = append(currentMessages, Message{
					Role:    "tool_call",
					Content: responseText,
					ToolCall: &ToolCall{
						ID:   toolCall.ID,
						Name: toolCall.Name,
						Args: toolCall.Args,
					},
				})

				// CRITICAL FIX: Embed loop warning into the result.Output (not as separate text)
				// This ensures it's part of the structured response the model sees
				if loopWarning != "" {
					log.Printf("[Loop Detection] %s", loopWarning)

					// Send warning as a visible message to the user
					eventChan <- StreamEvent{Type: StreamEventToken, Content: "\n\n" + loopWarning + "\n\n"}

					// Embed warning directly into result.Output
					if outputMap, ok := result.Output.(map[string]interface{}); ok {
						// Create a new map with warning injected
						newOutput := make(map[string]interface{})
						for k, v := range outputMap {
							newOutput[k] = v
						}
						newOutput["_loopWarning"] = loopWarning
						result.Output = newOutput
					} else {
						// For non-map outputs, wrap in a map
						result.Output = map[string]interface{}{
							"result":       result.Output,
							"_loopWarning": loopWarning,
						}
					}
				}

				// Truncate large tool results (apply to result.Output, not string)
				const maxToolResultSize = 10000
				if outputJSON, err := json.Marshal(result.Output); err == nil {
					if len(outputJSON) > maxToolResultSize {
						originalSize := len(outputJSON)
						truncated := string(outputJSON[:maxToolResultSize-500])
						result.Output = map[string]interface{}{
							"_truncated": true,
							"_message":   fmt.Sprintf("Result was %d chars, showing first %d chars", originalSize, maxToolResultSize-500),
							"_preview":   truncated,
						}
						log.Printf("[Tool Result Truncation] Truncated tool '%s' result from %d to %d chars",
							result.Name, originalSize, len(truncated))
					}
				}

				// CRITICAL FIX: Add tool_result message with proper role (user, not system)
				// This matches Anthropic's API format and ensures conversation continuity
				currentMessages = append(currentMessages, Message{
					Role:    "tool_result",
					Content: "", // Content is in ToolResult
					ToolResult: &ToolResult{
						ID:         toolCall.ID,
						Name:       toolCall.Name,
						Output:     result.Output,
						Error:      result.Error,
						DurationMs: result.DurationMs,
					},
				})

				// WORKFLOW STATE UPDATE: Update workflow state after successful tool execution (filtered function)
				// Apply to ALL models to match prescriptive filter behavior
				if result.Error == "" {
					switch toolCall.Name {
					case "coordinator_list_human_tasks":
						if workflowState["step"].(int) == 0 {
							workflowState["step"] = 1
							log.Printf("[Workflow State - Filtered] Step 1 complete: listed tasks")
						}

					case "coordinator_create_human_task":
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if taskID, hasTaskID := outputMap["taskId"].(string); hasTaskID && taskID != "" {
								workflowState["step"] = 2
								workflowState["humanTaskId"] = taskID
								log.Printf("[Workflow State - Filtered] Step 2 complete: created human task %s", taskID)
							} else if similarTasksFound, _ := outputMap["similarTasksFound"].(bool); similarTasksFound {
								// Case 2: Similar task found - use existing task instead of creating new one
								if similarTasks, ok := outputMap["similarTasks"].([]interface{}); ok && len(similarTasks) > 0 {
									if firstTask, ok := similarTasks[0].(map[string]interface{}); ok {
										if existingTaskID, ok := firstTask["taskId"].(string); ok && existingTaskID != "" {
											workflowState["step"] = 2
											workflowState["humanTaskId"] = existingTaskID
											log.Printf("[Workflow State - Filtered] Step 2 complete: using existing similar task %s", existingTaskID)
										}
									}
								}
							}
						}

					case "code_index_search":
						workflowState["step"] = 3
						workflowState["searchCompleted"] = true
						log.Printf("[Workflow State - Filtered] Step 3 complete: code search done")

					case "create_agent_task":
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if agentTaskID, hasAgentTaskID := outputMap["taskId"].(string); hasAgentTaskID && agentTaskID != "" {
								workflowState["step"] = 4
								workflowState["agentTaskId"] = agentTaskID
								log.Printf("[Workflow State - Filtered] Step 4 complete: created agent task %s", agentTaskID)
							}
						}

					case "execute_subagent":
						workflowState["step"] = 5
						log.Printf("[Workflow State - Filtered] Step 5 complete: subagent launched")
					}
				}
			}
		}

		// Max iterations reached
		log.Printf("[ChatService - Filtered] Max tool calls reached - RequestID: %s - Total iterations: %d, Tool calls: %d",
			requestID, iterationCount, toolCallCount)
	}()

	return eventChan, nil
}

// GetConfig returns the current AI configuration
func (s *ChatService) GetConfig() *AIConfig {
	return s.config
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

// WithIdentity adds identity to context
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, IdentityKey, identity)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetIdentityFromContext is a helper to extract identity from context
func GetIdentityFromContext(ctx context.Context) (*Identity, error) {
	identity, ok := ctx.Value(IdentityKey).(*Identity)
	if !ok || identity == nil {
		return nil, fmt.Errorf("identity not found in context")
	}
	return identity, nil
}

// calculateContextSize returns the total character count of all messages
func calculateContextSize(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content)
	}
	return total
}

// getToolResultPreview extracts the first maxChars of tool result content from messages
// Useful for debugging to see what tool results are being accumulated
func getToolResultPreview(messages []Message, maxChars int) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		// Look for tool result messages (role=system with "Tool '...' result:" pattern)
		if msg.Role == "system" && len(msg.Content) > 0 {
			if len(msg.Content) <= maxChars {
				return msg.Content
			}
			return msg.Content[:maxChars] + "..."
		}
	}
	return "(no tool results found)"
}

// MessagePriority represents the importance of a message for context preservation
type MessagePriority int

const (
	PriorityVeryLow  MessagePriority = 10  // list operations - trim first
	PriorityLow      MessagePriority = 25  // status checks, directory listings
	PriorityMedium   MessagePriority = 50  // read_file, write_file, coordinator ops
	PriorityHigh     MessagePriority = 75  // code_index_search, create_agent_task
	PriorityCritical MessagePriority = 100 // system prompts, user messages - NEVER trim
)

// getMessagePriority classifies messages by importance for context preservation
func getMessagePriority(msg Message) MessagePriority {
	// CRITICAL: Workflow enforcer error messages - contain guidance for next steps
	// These messages guide the LLM on what to do next and must NEVER be trimmed
	if strings.Contains(msg.Content, "BLOCKED") || strings.Contains(msg.Content, "NEXT:") {
		return PriorityCritical
	}

	// CRITICAL: Tool error messages - indicate failures that LLM must respond to
	if strings.Contains(msg.Content, "TOOL ERROR") || strings.Contains(msg.Content, "error:") {
		return PriorityCritical
	}

	// System prompts and user messages are critical - never trim
	if msg.Role == "system" {
		// Check if it's a tool result
		if strings.HasPrefix(msg.Content, "Tool '") {
			// Extract tool name from "Tool 'toolname' result: ..."
			if idx := strings.Index(msg.Content, "' result:"); idx > 6 {
				toolName := msg.Content[6:idx]
				return getToolResultPriority(toolName)
			}
			return PriorityMedium // Default for unparseable tool results
		}
		return PriorityCritical // System prompts are critical
	}
	if msg.Role == "user" {
		return PriorityCritical
	}

	// Assistant messages - check content for tool calls
	if msg.Role == "assistant" {
		// Assistant messages are medium priority by default
		return PriorityMedium
	}

	return PriorityLow
}

// getToolResultPriority assigns priority to tool results based on tool name
func getToolResultPriority(toolName string) MessagePriority {
	// HIGH PRIORITY: Critical for workflow - preserve strongly
	highPriorityTools := map[string]bool{
		"code_index_search":           true, // File discovery results - critical for agent tasks
		"create_agent_task":           true, // Agent task creation - core workflow
		"coordinator_create_human_task": true, // Human task creation - core workflow
		"execute_subagent":            true, // Subagent execution - core workflow
	}

	// MEDIUM PRIORITY: Important but can be re-retrieved if needed
	mediumPriorityTools := map[string]bool{
		"read_file":                   true,
		"write_file":                  true,
		"apply_patch":                 true,
		"coordinator_update_todo_status": true,
		"coordinator_update_task_status": true,
		"coordinator_get_agent_task":  true,
		"bash":                        true,
	}

	// LOW PRIORITY: Informational - can be trimmed
	lowPriorityTools := map[string]bool{
		"list_directory":              true,
		"code_index_status":           true,
		"coordinator_get_popular_collections": true,
		"coordinator_find_similar_tasks": true,
	}

	if highPriorityTools[toolName] {
		return PriorityHigh
	}
	if mediumPriorityTools[toolName] {
		return PriorityMedium
	}
	if lowPriorityTools[toolName] {
		return PriorityLow
	}

	// VERY LOW PRIORITY: List operations - trim first
	if strings.HasPrefix(toolName, "list_") || strings.HasPrefix(toolName, "coordinator_list_") {
		return PriorityVeryLow
	}

	return PriorityMedium // Default for unknown tools
}

// applySlidingWindow keeps only recent messages to prevent context accumulation
// ENHANCED: Uses priority-based trimming to preserve critical tool results
// Strategy: Keep high-priority messages (code_index_search) and trim low-priority first
func applySlidingWindow(messages []Message, maxMessages int) []Message {
	if len(messages) <= maxMessages {
		return messages // No need to trim
	}

	// Identify system prompt (if exists at index 0)
	hasSystemPrompt := len(messages) > 0 && messages[0].Role == "system" && !strings.HasPrefix(messages[0].Content, "Tool '")

	// Find original user message (first "user" role after system prompt)
	var systemMsg, userMsg *Message
	userMsgIdx := -1

	if hasSystemPrompt {
		systemMsg = &messages[0]
		// Find first user message after system
		for i := 1; i < len(messages); i++ {
			if messages[i].Role == "user" {
				userMsg = &messages[i]
				userMsgIdx = i
				break
			}
		}
	} else {
		// No system prompt - first message should be user
		if len(messages) > 0 && messages[0].Role == "user" {
			userMsg = &messages[0]
			userMsgIdx = 0
		}
	}

	// Calculate how many slots we need for critical messages
	reservedSlots := 0
	if systemMsg != nil {
		reservedSlots++
	}
	if userMsg != nil {
		reservedSlots++
	}

	// Get messages after the original user message (the conversation)
	var conversationMsgs []Message
	startIdx := userMsgIdx + 1
	if startIdx < len(messages) {
		conversationMsgs = messages[startIdx:]
	}

	// Calculate target size for conversation messages
	targetConversationSize := maxMessages - reservedSlots
	if targetConversationSize < 0 {
		targetConversationSize = 0
	}

	// SMART TRIMMING: If we need to reduce conversation messages, use priority-based selection
	var selectedConversationMsgs []Message
	if len(conversationMsgs) <= targetConversationSize {
		// No trimming needed for conversation
		selectedConversationMsgs = conversationMsgs
	} else {
		// Need to trim - use priority-based selection
		selectedConversationMsgs = selectMessagesByPriority(conversationMsgs, targetConversationSize)
	}

	// Build final message list
	result := make([]Message, 0, maxMessages)

	// Add system prompt if exists
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}

	// Add original user message if exists
	if userMsg != nil {
		result = append(result, *userMsg)
	}

	// Add selected conversation messages
	result = append(result, selectedConversationMsgs...)

	log.Printf("[Smart Sliding Window] Reduced from %d to %d messages (system: %v, user: %v, conversation: %d, preserved high-priority: %d)",
		len(messages), len(result), systemMsg != nil, userMsg != nil, len(selectedConversationMsgs),
		countHighPriorityMessages(selectedConversationMsgs))

	return result
}

// selectMessagesByPriority selects messages using priority-based algorithm
// Keeps: recent messages + high-priority messages (code_index_search, create_agent_task)
func selectMessagesByPriority(messages []Message, targetSize int) []Message {
	if len(messages) <= targetSize {
		return messages
	}

	// Classify messages by priority
	type indexedMessage struct {
		msg      Message
		index    int
		priority MessagePriority
	}

	indexed := make([]indexedMessage, len(messages))
	for i, msg := range messages {
		indexed[i] = indexedMessage{
			msg:      msg,
			index:    i,
			priority: getMessagePriority(msg),
		}
	}

	// Strategy: Always keep last N messages + highest priority older messages
	// This ensures recent context + critical historical context (code_index_search)

	// Keep last 40% of target as recent messages (maintains conversation flow)
	recentCount := targetSize * 4 / 10
	if recentCount < 2 {
		recentCount = 2 // Minimum recent messages
	}
	if recentCount > len(messages) {
		recentCount = len(messages)
	}

	// Reserve remaining slots for high-priority older messages
	prioritySlots := targetSize - recentCount

	// Always include last N messages (recent context)
	recentStartIdx := len(messages) - recentCount
	selected := make(map[int]bool)
	for i := recentStartIdx; i < len(messages); i++ {
		selected[i] = true
	}

	// Add high-priority older messages (not already selected)
	if prioritySlots > 0 {
		// Sort older messages by priority (descending)
		olderMessages := indexed[:recentStartIdx]
		sort.Slice(olderMessages, func(i, j int) bool {
			// First by priority (higher first), then by recency (later first)
			if olderMessages[i].priority != olderMessages[j].priority {
				return olderMessages[i].priority > olderMessages[j].priority
			}
			return olderMessages[i].index > olderMessages[j].index
		})

		// Add top priority messages up to available slots
		added := 0
		for _, im := range olderMessages {
			if added >= prioritySlots {
				break
			}
			// Only add high or critical priority messages
			if im.priority >= PriorityHigh {
				selected[im.index] = true
				added++
			}
		}
	}

	// Build result maintaining original order
	result := make([]Message, 0, targetSize)
	for i, msg := range messages {
		if selected[i] {
			result = append(result, msg)
		}
	}

	return result
}

// countHighPriorityMessages counts messages with HIGH or CRITICAL priority
func countHighPriorityMessages(messages []Message) int {
	count := 0
	for _, msg := range messages {
		if p := getMessagePriority(msg); p >= PriorityHigh {
			count++
		}
	}
	return count
}

// isRateLimitError checks if an error is a rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	// Check for common rate limit error patterns
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "402") || // Payment Required (Ollama rate limit)
		strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "usage limit") || // Matches "hourly usage limit", "daily usage limit", etc.
		strings.Contains(errStr, "hourly limit")
}

// Tool category definitions for smart filtering
var (
	// Phase 1 - Initial Assessment: List/search for existing work
	workflowPhase1Tools = []string{
		"coordinator_list_human_tasks",
		"coordinator_list_agent_tasks",
		"coordinator_get_agent_task",
		"knowledge_find",
		"coordinator_query_knowledge",
	}

	// Phase 2 - Task Creation: Create and manage tasks
	workflowPhase2Tools = []string{
		"coordinator_create_human_task",
		"create_agent_task",   // FIXED: No coordinator_ prefix
		"list_agent_tasks",    // FIXED: No coordinator_ prefix
		"execute_subagent",    // FIXED: No coordinator_ prefix
		"list_subagents",      // FIXED: No coordinator_ prefix
		"coordinator_update_task_status",
		"coordinator_update_todo_status",
	}

	// Phase 3 - Code Discovery: Search and index code
	workflowPhase3Tools = []string{
		"code_index_search",
		"code_index_status",
		"code_index_add_folder",
		"code_index_scan",
		"code_index_remove_folder",
	}

	// Phase 4 - Knowledge Management: Store and retrieve knowledge
	workflowPhase4Tools = []string{
		"coordinator_upsert_knowledge",
		"knowledge_store",
		"coordinator_get_popular_collections",
	}

	// Core tools - always included in every request
	coreTools = []string{
		"bash",
		"file_read",
		"file_write",
		"apply_patch",
		"coordinator_add_task_prompt_notes",
		"coordinator_update_task_prompt_notes",
		"coordinator_add_todo_prompt_notes",
		"coordinator_update_todo_prompt_notes",
	}
)

// filterToolsByWorkflowState analyzes tool call history and returns relevant tools
// This reduces token usage by ~70% (from 38 tools to 8-12 tools per request)
func filterToolsByWorkflowState(toolCallHistory []ToolResult) []string {
	// Start with core tools (always included)
	relevantTools := make(map[string]bool)
	for _, tool := range coreTools {
		relevantTools[tool] = true
	}

	// Analyze recent tool calls to determine workflow phase
	recentCalls := make(map[string]bool)
	lookbackLimit := 3 // Look at last 3 tool calls

	// Collect recent tool names
	for i := len(toolCallHistory) - 1; i >= 0 && len(recentCalls) < lookbackLimit; i-- {
		recentCalls[toolCallHistory[i].Name] = true
	}

	// Determine which phases to include based on recent activity
	includePhase1 := len(toolCallHistory) == 0 // First request - include listing tools
	includePhase2 := false
	includePhase3 := false
	includePhase4 := false

	// Check recent calls to determine active phases
	for toolName := range recentCalls {
		// If we just listed tasks, include task creation tools
		if toolName == "coordinator_list_human_tasks" || toolName == "list_agent_tasks" {
			includePhase2 = true
		}
		// If we just created a task, include code search tools
		if toolName == "coordinator_create_human_task" || toolName == "create_agent_task" {
			includePhase3 = true
		}
		// If we searched code, include knowledge and task creation tools
		if toolName == "code_index_search" {
			includePhase2 = true // Can create agent tasks
			includePhase4 = true // Can store knowledge
		}
		// If we're managing knowledge, keep those tools
		if toolName == "coordinator_upsert_knowledge" || toolName == "knowledge_store" || toolName == "knowledge_find" {
			includePhase4 = true
		}
		// If we're updating task status, keep task management tools
		if toolName == "coordinator_update_task_status" || toolName == "coordinator_update_todo_status" {
			includePhase2 = true
		}
	}

	// Add tools for active phases
	if includePhase1 {
		for _, tool := range workflowPhase1Tools {
			relevantTools[tool] = true
		}
	}
	if includePhase2 {
		for _, tool := range workflowPhase2Tools {
			relevantTools[tool] = true
		}
	}
	if includePhase3 {
		for _, tool := range workflowPhase3Tools {
			relevantTools[tool] = true
		}
	}
	if includePhase4 {
		for _, tool := range workflowPhase4Tools {
			relevantTools[tool] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(relevantTools))
	for tool := range relevantTools {
		result = append(result, tool)
	}

	return result
}

// extractFilePathsFromCodeIndexResult extracts file paths from code_index_search result
func extractFilePathsFromCodeIndexResult(result ToolResult) []string {
	if result.Error != "" {
		return nil
	}

	outputMap, ok := result.Output.(map[string]interface{})
	if !ok {
		return nil
	}

	resultsArray, ok := outputMap["results"].([]interface{})
	if !ok {
		return nil
	}

	var paths []string
	for _, item := range resultsArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if path, ok := itemMap["path"].(string); ok && path != "" {
			paths = append(paths, path)
		}
	}

	return paths
}

// validateFilePaths checks if all provided file paths exist on the filesystem
// Returns: (validPaths, invalidPaths)
func validateFilePaths(paths []string) ([]string, []string) {
	var validPaths []string
	var invalidPaths []string

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			validPaths = append(validPaths, path)
		} else {
			invalidPaths = append(invalidPaths, path)
		}
	}

	return validPaths, invalidPaths
}

// generateWorkflowStateGuidance creates explicit state tracking messages for the fallback model
// to help it understand the 5-step coordinator workflow and what to do next.
// This is crucial for smaller models like Haiku that need more explicit guidance.
func (s *ChatService) generateWorkflowStateGuidance(toolName string, result ToolResult, toolCallCount int) string {
	// Skip guidance if tool failed
	if result.Error != "" {
		return ""
	}

	// Extract data from tool result for guidance
	var guidance string

	switch toolName {
	case "coordinator_list_human_tasks":
		// Step 1 complete - guide to Step 2
		guidance = "✅ STEP 1 COMPLETE: You checked existing tasks.\n" +
			"➡️ NEXT ACTION: Call 'coordinator_create_human_task' with the user's exact request.\n" +
			"   Example: {\"prompt\": \"<user's exact words>\"}\n" +
			"🔒 DO NOT call coordinator_list_human_tasks again - you already have the results."

	case "coordinator_create_human_task":
		// Step 2 complete - extract taskId and guide to Step 3
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			taskID, hasTaskID := outputMap["taskId"].(string)
			similarTasksFound, _ := outputMap["similarTasksFound"].(bool)

			if similarTasksFound {
				// Duplicate detected - extract first similar task's ID and proceed
				var firstTaskID string
				if similarTasks, ok := outputMap["similarTasks"].([]interface{}); ok && len(similarTasks) > 0 {
					if firstTask, ok := similarTasks[0].(map[string]interface{}); ok {
						if taskID, ok := firstTask["taskId"].(string); ok {
							firstTaskID = taskID
						}
					}
				}

				if firstTaskID != "" {
					// Found a similar task - use it and proceed to Step 2
					guidance = fmt.Sprintf("⚠️ SIMILAR TASK FOUND: A task with similar intent already exists.\n"+
						"📝 SAVE THIS: humanTaskId = \"%s\" (using existing similar task)\n"+
						"➡️ NEXT ACTION: Call 'code_index_search' ONCE to find relevant files.\n"+
						"   Example: {\"query\": \"<what user wants to change>\", \"limit\": 15}\n"+
						"🔒 DO NOT call coordinator_create_human_task again - you already have a taskId.\n"+
						"💡 To create a NEW task instead, call coordinator_create_human_task with forceCreate=true", firstTaskID)
				} else {
					// No task ID found in similar tasks - tell model to force create
					guidance = "⚠️ DUPLICATE TASK DETECTED: Similar tasks exist but no ID was found.\n" +
						"➡️ NEXT ACTION: Call coordinator_create_human_task with forceCreate=true to create anyway.\n" +
						"   Example: {\"prompt\": \"<user's exact words>\", \"forceCreate\": true}"
				}
			} else if hasTaskID {
				// Task created successfully - guide to Step 3
				guidance = fmt.Sprintf("✅ STEP 2 COMPLETE: Human task created successfully.\n"+
					"📝 SAVE THIS: humanTaskId = \"%s\"\n"+
					"➡️ NEXT ACTION: Call 'code_index_search' ONCE to find relevant files.\n"+
					"   Example: {\"query\": \"<what user wants to change>\", \"limit\": 15}\n"+
					"🔒 DO NOT call coordinator_create_human_task again - you already have the taskId.\n"+
					"🔒 You will need this taskId for Step 4 (create_agent_task).", taskID)
			}
		}

	case "code_index_search":
		// Step 3 complete - extract file paths and guide to Step 4
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			filePathsRaw, hasFilePaths := outputMap["FILE_PATHS_TO_USE"]
			_, hasResults := outputMap["results"]

			if hasFilePaths || hasResults {
				filePathsCount := 0
				if filePaths, ok := filePathsRaw.([]interface{}); ok {
					filePathsCount = len(filePaths)
				}

				guidance = fmt.Sprintf("✅ STEP 3 COMPLETE: Code search returned %d file(s).\n"+
					"📝 EXTRACT: Copy file paths from FILE_PATHS_TO_USE array above.\n"+
					"➡️ NEXT ACTION: Call 'create_agent_task' with:\n"+
					"   - humanTaskId: \"<taskId from Step 2>\"\n"+
					"   - agentName: \"ui-dev\" (for UI changes) or \"go-dev\" (for backend)\n"+
					"   - role: \"Brief mission description\"\n"+
					"   - contextSummary: \"WHAT to change, WHERE (file:line from search), HOW\"\n"+
					"   - filesModified: [\"<COPY exact paths from FILE_PATHS_TO_USE>\"]\n"+
					"   - todos: [{description: \"Implement X in file Y\", filePath, contextHint}]\n\n"+
					"🚨 CRITICAL:\n"+
					"   • filesModified MUST NOT be empty - populate with paths from FILE_PATHS_TO_USE\n"+
					"   • TODOs must be implementation steps, NOT discovery steps\n"+
					"   • DO NOT create TODOs like 'Search for...' or 'Find...'\n"+
					"   • Subagent CANNOT run code_index_search - it's blocked in write-only mode\n\n"+
					"🔒 DO NOT call code_index_search again - you already have the file paths.\n"+
					"🔒 Use EXACT paths from FILE_PATHS_TO_USE array - do NOT type paths manually!", filePathsCount)
			}
		}

	case "create_agent_task":
		// Step 4 complete - extract agent task ID and guide to Step 5
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			agentTaskID, hasAgentTaskID := outputMap["taskId"].(string)
			agentName, _ := outputMap["agentName"].(string)

			if hasAgentTaskID {
				guidance = fmt.Sprintf("✅ STEP 4 COMPLETE: Agent task created successfully.\n"+
					"📝 Agent Task ID: \"%s\"\n"+
					"➡️ NEXT ACTION (FINAL): Call 'execute_subagent' to launch the agent:\n"+
					"   {\"agentTaskId\": \"%s\", \"parentChatId\": \"<from session context>\"}\n"+
					"🔒 DO NOT call create_agent_task again - the task is created.\n"+
					"✅ After execute_subagent, the %s agent will implement the changes - YOU ARE DONE!", agentTaskID, agentTaskID, agentName)
			}
		}

	case "execute_subagent":
		// Step 5 complete - extract agentTaskId and tell coordinator to STOP
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			agentTaskID, hasAgentTaskID := outputMap["agentTaskId"].(string)
			agentName, _ := outputMap["agentName"].(string)
			subchatID, _ := outputMap["subchatId"].(string)

			if hasAgentTaskID {
				guidance = fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
					"✅ WORKFLOW COMPLETE - YOUR JOB IS DONE!\n"+
					"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
					"The %s agent is executing your request in background.\n"+
					"   • Agent Task ID: %s\n"+
					"   • Subchat ID: %s\n\n"+
					"🛑 STOP HERE - DO NOT CALL ANY MORE TOOLS\n"+
					"🛑 DO NOT call list_agent_tasks, coordinator_get_agent_task, or any monitoring tools\n"+
					"🛑 DO NOT try to check status - the agent is working independently\n\n"+
					"✅ YOUR ONLY ACTION: Inform the user that work has begun.\n"+
					"   Example: \"I've delegated this to the %s agent. They're working on it now.\"\n\n"+
					"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
					agentName, agentTaskID, subchatID, agentName)
			} else {
				// Fallback if agentTaskId not found
				guidance = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
					"✅ WORKFLOW COMPLETE - YOUR JOB IS DONE!\n" +
					"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
					"The specialist agent is executing the request in background.\n\n" +
					"🛑 STOP HERE - DO NOT CALL ANY MORE TOOLS\n" +
					"✅ YOUR ONLY ACTION: Inform the user that work has begun.\n\n" +
					"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
			}
		}
	}

	// If we generated guidance, add a clear separator
	if guidance != "" {
		guidance = "\n" + strings.Repeat("━", 70) + "\n" +
			"🤖 WORKFLOW STATE TRACKER (for your guidance)\n" +
			strings.Repeat("━", 70) + "\n" +
			guidance + "\n" +
			strings.Repeat("━", 70)
	}

	return guidance
}
