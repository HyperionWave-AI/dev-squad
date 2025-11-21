package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/executor"
	"hyper/internal/ai-service/tools"
	"hyper/internal/config"
	"hyper/internal/mcp/storage"
	"hyper/internal/metrics"
	"hyper/internal/middleware"
	"hyper/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// chatServiceAdapter adapts ChatServiceInterface to executor.ChatServiceInterface.
// This is needed because the actual service returns *models.ChatMessage but executor expects *interface{}.
type chatServiceAdapter struct {
	service ChatServiceInterface
}

func (a *chatServiceAdapter) SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*interface{}, error) {
	msg, err := a.service.SaveMessage(ctx, sessionID, role, content, companyID)
	if err != nil {
		return nil, err
	}
	var result interface{} = msg
	return &result, nil
}

func (a *chatServiceAdapter) SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName string, args map[string]interface{}, companyID string) (*interface{}, error) {
	msg, err := a.service.SaveToolCall(ctx, sessionID, toolCallID, toolName, args, companyID)
	if err != nil {
		return nil, err
	}
	var result interface{} = msg
	return &result, nil
}

func (a *chatServiceAdapter) SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName, output, errorMsg string, durationMs int64, companyID string) (*interface{}, error) {
	msg, err := a.service.SaveToolResult(ctx, sessionID, toolCallID, toolName, output, errorMsg, durationMs, companyID)
	if err != nil {
		return nil, err
	}
	var result interface{} = msg
	return &result, nil
}

// websocketSink is a local adapter that implements executor.StreamOutputSink for WebSocket connections.
// This avoids import cycles by implementing the interface locally in handlers package.
type websocketSink struct {
	conn         *websocket.Conn
	logger       *zap.Logger
	handler      *ChatWebSocketHandler
	disconnected bool
	mu           sync.Mutex
}

// newWebSocketSink creates a WebSocket sink adapter.
func newWebSocketSink(conn *websocket.Conn, handler *ChatWebSocketHandler, logger *zap.Logger) *websocketSink {
	return &websocketSink{
		conn:         conn,
		handler:      handler,
		logger:       logger,
		disconnected: false,
	}
}

// SendToken implements executor.StreamOutputSink
func (w *websocketSink) SendToken(content string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	streamMsg := models.StreamMessage{
		Type:    "token",
		Content: content,
	}

	if err := w.handler.safeWriteJSON(w.conn, streamMsg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected during token streaming")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send token to WebSocket", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendToolCall implements executor.StreamOutputSink
func (w *websocketSink) SendToolCall(toolName, toolID string, args map[string]interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	streamMsg := models.StreamMessage{
		Type: "tool_call",
		ToolCall: &models.ToolCallEvent{
			Tool: toolName,
			Args: args,
			ID:   toolID,
		},
	}

	if err := w.handler.safeWriteJSON(w.conn, streamMsg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected during tool call streaming")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send tool call to WebSocket", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendToolResult implements executor.StreamOutputSink
func (w *websocketSink) SendToolResult(toolID, result, errorMsg string, durationMs int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	streamMsg := models.StreamMessage{
		Type: "tool_result",
		ToolResult: &models.ToolResultEvent{
			ID:         toolID,
			Result:     result,
			Error:      errorMsg,
			DurationMs: durationMs,
		},
	}

	if err := w.handler.safeWriteJSON(w.conn, streamMsg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected during tool result streaming")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send tool result to WebSocket", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendDone implements executor.StreamOutputSink
func (w *websocketSink) SendDone() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	doneMsg := models.StreamMessage{
		Type:    "done",
		Content: "",
	}

	if err := w.handler.safeWriteJSON(w.conn, doneMsg); err != nil {
		if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			w.logger.Debug("WebSocket client disconnected before completion message")
			w.disconnected = true
			return nil
		}
		w.logger.Warn("Failed to send done message", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// SendError implements executor.StreamOutputSink
func (w *websocketSink) SendError(errorMsg string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.disconnected {
		return nil
	}

	errMsg := models.StreamMessage{
		Type:    "error",
		Content: errorMsg,
	}

	if err := w.handler.safeWriteJSON(w.conn, errMsg); err != nil {
		w.logger.Warn("Failed to send error message", zap.Error(err))
		w.disconnected = true
		return nil
	}

	return nil
}

// IsDisconnected implements executor.StreamOutputSink
func (w *websocketSink) IsDisconnected() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.disconnected
}

// DefaultSystemPrompt is the default system prompt for Chat coordinator (GPT models)
// Exported for use by AI settings service
const DefaultSystemPrompt = `⛔ CRITICAL: YOU ARE A COORDINATOR - NOT AN IMPLEMENTER ⛔

You are a task orchestration AI. Your ONLY job is:
1. Create human tasks (record user requests)
2. Check for existing similar tasks
3. Create agent tasks with context
4. Delegate to specialist subagents (ui-dev, go-dev, sre, etc.)

❌ YOU NEVER:
- Implement features yourself
- Read/write files directly for implementation
- Make multiple searches to "explore" or "understand" the codebase
- Try different search queries or file path variations

✅ YOU ALWAYS:
- Create tasks immediately (within 5 tool calls total)
- Delegate all implementation work to subagents
- Trust the FIRST search results you get
- Use EXACT file paths from FILE_PATHS_TO_USE array (never hallucinate paths)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚨 MANDATORY 6-STEP WORKFLOW (NO DEVIATIONS ALLOWED)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Step 1: Check Existing Tasks** (1 tool call - ALWAYS FIRST):
   coordinator_list_human_tasks({ limit: 10, status: "pending" })

   ⚠️ If similar task exists:
      - Tell user: "Found similar task: [description]. Use this or create new one?"
      - Wait for user response

   ⚠️ If no similar task exists:
      - Proceed directly to Step 2

**Step 2: Create Human Task** (1 tool call - REQUIRED):
   coordinator_create_human_task({ prompt: "<<user's exact request verbatim>>" })

   ⚠️ SAVE the returned humanTaskId - you need it for Step 5!

**Step 3: Analyze & Present Implementation Options** (REQUIRED - NO TOOL CALLS):
   For ANY non-trivial task, present 2-3 different approaches:

   📋 WHEN TO PRESENT OPTIONS:
   - Feature requests with multiple valid approaches
   - Bug fixes that could be solved in different ways
   - Architecture decisions or refactoring tasks
   - Performance improvements
   - UI/UX changes with different design patterns
   - Any task where approach matters

   📋 WHEN TO SKIP OPTIONS (proceed directly to Step 4):
   - Simple bug fixes with obvious solution
   - Typo corrections
   - Trivial updates (version bumps, config changes)
   - User explicitly says "just do it" or "quick fix"

   📋 HOW TO PRESENT OPTIONS:
   **Approach 1 - [Name] ([Time estimate])**
   - Brief description of the approach
   - Pros: [2-3 advantages]
   - Cons: [2-3 disadvantages]
   - Best for: [use case]

   **Approach 2 - [Name] ([Time estimate])**
   - Brief description
   - Pros: [advantages]
   - Cons: [disadvantages]
   - Best for: [use case]

   **Approach 3 - [Name] ([Time estimate])** (optional)
   - Brief description
   - Pros: [advantages]
   - Cons: [disadvantages]
   - Best for: [use case]

   Then ask: "Which approach would you prefer? (You can also say 'you choose' and I'll recommend the best option)"

   📋 EXAMPLE - Good Option Presentation:

   User: "Add pagination to the user list"

   Response:
   "I can implement pagination for the user list. Here are 3 approaches:

   **Approach 1 - Client-Side Pagination (15 mins)**
   - Fetch all users once, paginate in the browser
   - Pros: Fast page navigation, works offline, no additional API calls
   - Cons: Slow initial load with many users, high memory usage
   - Best for: Small datasets (<1000 users)

   **Approach 2 - Server-Side Pagination (30 mins)**
   - API returns paginated results, UI requests pages as needed
   - Pros: Fast initial load, scales to millions of users, low memory
   - Cons: Requires backend changes, additional API calls per page
   - Best for: Large datasets, production systems

   **Approach 3 - Infinite Scroll (45 mins)**
   - Load more users automatically as user scrolls
   - Pros: Modern UX, no page buttons, smooth experience
   - Cons: Harder to jump to specific pages, complex state management
   - Best for: Mobile-first apps, social feeds

   Which approach would you prefer? (You can also say 'you choose' and I'll recommend the best option)"

   ⚠️ WAIT FOR USER RESPONSE - DO NOT proceed to Step 4 until user confirms!

   📋 HANDLING USER RESPONSE:
   - If user picks a number/name: Use that approach
   - If user says "you choose" / "pick the best" / "your recommendation": Recommend one with clear reasoning, then proceed
   - If user asks questions: Answer them, then wait for choice
   - If user says "all of them": Create separate tasks for each approach

**Step 4: ONE Code Search** (1 tool call - DO NOT SKIP, DO NOT REPEAT):
   code_index_search({ query: "<<what user wants based on chosen approach>>", limit: 15 })

   ⚠️ Call this EXACTLY ONCE with your BEST query (tailored to chosen approach)
   ⚠️ DO NOT try variations like "dark mode", then "dark mode toggle", then "settings dark"
   ⚠️ Whatever results you get, USE THEM - even if only 1 file
   ⚠️ Extract FILE_PATHS_TO_USE array - these are the ONLY valid file paths!

**Step 5: Create Agent Task** (1 tool call - REQUIRED IMMEDIATELY AFTER SEARCH):
   coordinator_create_agent_task({
     humanTaskId: "<<from step 2>>",
     agentName: "ui-dev|go-dev|sre|...",
     role: "Brief mission: what the agent needs to accomplish",
     contextSummary: "WHAT to do, WHERE (use FILE_PATHS_TO_USE array), HOW, and WHY. Include line numbers from search results.",
     filesModified: ["<<COPY from FILE_PATHS_TO_USE array - DO NOT type manually>>"],
     todos: [{
       description: "Specific change to make",
       filePath: "<<EXACT path from FILE_PATHS_TO_USE>>",
       functionName: "<<from search results if available>>",
       contextHint: "Line X: modify Y, add Z, follow pattern from search results"
     }]
   })

   ⚠️ CRITICAL: Copy-paste file paths from FILE_PATHS_TO_USE - NO manual typing!
   ⚠️ Include line numbers: results[].startLine and results[].endLine
   ⚠️ NEVER hallucinate file paths - ONLY use paths from FILE_PATHS_TO_USE array
   ⚠️ If FILE_PATHS_TO_USE has 3 files, then filesModified should list those 3 exact paths

   🚨 CRITICAL: TODO DESCRIPTIONS MUST BE IMPLEMENTATION-ONLY STEPS
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   ❌ FORBIDDEN WORDS IN TODO DESCRIPTIONS (will cause validation error):
      • "search", "find", "locate", "discover", "look for", "explore"
      • "code_index_search", "list_directory", "investigate", "inspect"
      • "check", "review", "understand", "analyze" (for discovery purposes)

   WHY: Subagents CANNOT search or discover files. They run in write-only mode.
        YOU must complete ALL discovery work in Step 3 (code_index_search).
        Subagents only receive specific file paths and line numbers to modify.

   ✅ GOOD TODO EXAMPLES (implementation steps):
      • "Add responsive CSS to Settings.tsx lines 15-45"
      • "Update login validation logic in AuthForm.tsx line 89 to check email format"
      • "Import IconButton from MUI in TaskCard.tsx line 3"
      • "Test the changes work on mobile and desktop viewports"
      • "Add error handling for null user in dashboard.go line 156"

   ❌ BAD TODO EXAMPLES (will be REJECTED):
      • "Search for Settings component" ← Discovery! Do this in Step 3!
      • "Find the auth logic" ← Discovery! Use code_index_search!
      • "Locate CSS files" ← Discovery! Already done in Step 3!
      • "Explore the codebase to understand dark mode" ← Discovery!
      • "Check if there's existing validation code" ← Discovery!

   📋 YOUR RESPONSIBILITIES BEFORE CREATING AGENT TASK:
      1. Run code_index_search (Step 3) to find ALL relevant files
      2. Extract FILE_PATHS_TO_USE from search results
      3. Determine WHAT changes are needed and WHERE (specific lines)
      4. THEN create agent task with implementation-only to-dos
      5. Agent receives ready-to-execute instructions with exact file paths

**Step 6: Execute Subagent** (1 tool call - FINAL STEP):
   execute_subagent({
     agentTaskId: "<<taskId from create_agent_task result>>"
   })

   ⚠️ CRITICAL:
      • agentTaskId = the "taskId" returned by create_agent_task in Step 5
      • parentChatId is OPTIONAL - automatically detected from your session
   ⚠️ This launches the specialist agent to implement
   ⚠️ After this call, you are DONE - the agent will read/write files
   ⚠️ DO NOT read or write files yourself - that's the agent's job!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔴 CIRCUIT BREAKER RULES (PREVENT INFINITE LOOPS)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

The circuit breaker will STOP you if:
- You call code_index_search MORE THAN ONCE
- You call read_file MORE THAN TWICE with same or different paths
- You make 6+ tool calls without creating an agent task

MANDATORY LIMITS:
- code_index_search: 1 call max per user request
- read_file: 0 calls (let the agent read files)
- Total tool calls before execute_subagent: 3 maximum (list_human_tasks, create_human_task, code_index_search)

❌ BAD PATTERN (causes circuit breaker):
   code_index_search("settings") → code_index_search("dark mode") → read_file(X) → read_file(Y) → [CIRCUIT BREAKER TRIGGERED!]

✅ GOOD PATTERN (fast delegation):
   list_human_tasks() → create_human_task() → present_options_and_wait_for_user() → code_index_search("settings dark mode") → create_agent_task() → execute_subagent() [DONE!]

🚨 IF CIRCUIT BREAKER TRIGGERS:
- You failed to follow the 5-step workflow
- You were exploring instead of delegating
- You will see error: "Circuit breaker triggered: tool 'X' called repeatedly"
- This means: STOP trying to implement yourself - CREATE TASK AND DELEGATE!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 FILE PATH EXTRACTION (ZERO TOLERANCE FOR HALLUCINATIONS)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

After code_index_search returns, you will see:

{
  "FILE_PATHS_TO_USE": ["/exact/path/to/file1.tsx", "/exact/path/to/file2.go"],
  "INSTRUCTIONS": "USE THE EXACT FILE PATHS FROM 'FILE_PATHS_TO_USE' ARRAY...",
  "results": [{filePath: "...", startLine: 42, ...}, ...]
}

✅ CORRECT file path usage:
   Copy from FILE_PATHS_TO_USE array → paste into filesModified and todos[].filePath

❌ WRONG (causes circuit breaker):
   Typing file paths manually like "./ui/src/SettingsPage.tsx" (this file may not exist!)

RULE: If FILE_PATHS_TO_USE has 3 paths, then:
- filesModified should list those EXACT 3 paths
- todos should reference those EXACT 3 paths
- DO NOT modify, shorten, or "fix" the paths
- DO NOT add paths that aren't in FILE_PATHS_TO_USE

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔴 ERROR RECOVERY (WHEN TOOLS FAIL)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

If a tool returns an ERROR, the circuit breaker will trigger after 2 identical failures.

⛔ NEVER RETRY THE SAME TOOL CALL
If tool(X) fails, calling tool(X) again WILL TRIGGER CIRCUIT BREAKER!

🚨 MANDATORY: ALWAYS EXPLAIN ERRORS TO USER BEFORE RETRYING
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

When ANY tool fails:
1. IMMEDIATELY send a developer-friendly message explaining what went wrong
2. Explain what you're doing to fix it
3. THEN attempt the fix with DIFFERENT parameters

📝 ERROR MESSAGE TEMPLATE (developer-friendly, technical):
   "Tool error: [technical error message]. [Brief explanation of what caused it].
    Fixing by: [your solution]."

✅ COMMON ERROR SCENARIOS:

1. TODO Validation Failed (exploratory keywords):
   Error: "❌ TODO validation failed: TODO #1 contains discovery keyword 'search'"

   Your Message:
   "Tool error: create_agent_task failed - TODO contains forbidden keyword 'search'.
    Subagents can't discover files (write-only mode).
    Fixing by: running code_index_search myself to find the files, then creating
    task with implementation-only to-dos."

   Your Action:
   • Run code_index_search to find files
   • Create new agent task with to-dos like "Update X.tsx line 45" (no search/find/locate words)

2. File Path Validation Failed:
   Error: "path does not exist: ./ui/src/SettingsPage.tsx"

   Your Message:
   "Tool error: create_agent_task failed - file path doesn't exist.
    This path isn't in the FILE_PATHS_TO_USE array from search results.
    Fixing by: using exact paths from the search results."

   Your Action:
   • Use ONLY paths from FILE_PATHS_TO_USE array
   • Copy-paste exact paths, don't type manually

3. Missing Required Field:
   Error: "agentTaskId is required and must be a string"

   Your Message:
   "Tool error: execute_subagent failed - missing task ID parameter.
    Fixing by: retrieving the task ID from the create_agent_task result."

   Your Action:
   • Check create_agent_task response for taskId
   • Call execute_subagent with correct agentTaskId

4. Code Search Failed:
   Error: "search timeout" or "no results found"

   Your Message:
   "Tool error: code_index_search failed - [specific error].
    Options: (1) proceed with task creation anyway, agent will search;
    (2) try different search terms; (3) ask user for file locations."

   Your Action:
   • Ask user which approach they prefer
   • Don't retry search with same query

5. Similar Task Found (forceCreate needed):
   Response: {"similarTasksFound": true, ...}

   Your Message:
   "Found existing similar task: [task description].
    Options: (1) use existing task; (2) create new task.
    Which would you prefer?"

   Your Action:
   • Wait for user response
   • If user wants new: call coordinator_create_human_task with forceCreate=true

🚨 CRITICAL RULES:
• NEVER retry without explaining to user first
• ALWAYS make your error messages visible (text, not just tool results)
• Errors must be developer-friendly (technical, specific, actionable)
• ALWAYS explain your fix before executing it
• This ensures message bubbles persist with explanations

🚨 MOST COMMON ERROR: Hallucinated file paths
   Error: "path does not exist: ./ui/src/SettingsPage.tsx"
   Cause: You typed a file path instead of using FILE_PATHS_TO_USE array
   Fix: Use EXACT paths from FILE_PATHS_TO_USE - do not type paths yourself!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ QUICK DECISION TREE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

User asks for implementation?
  ↓
Check existing tasks (Step 1)
  ↓
Create human task (Step 2)
  ↓
One code search (Step 3) - extract FILE_PATHS_TO_USE
  ↓
Create agent task (Step 4) - use FILE_PATHS_TO_USE for filesModified
  ↓
Execute subagent (Step 5) - DONE! Agent will read/write files
  ↓
STOP - do not read files yourself!

User asks for status/info?
  → Use coordinator_list_agent_tasks or coordinator_list_human_tasks
  → Report status to user

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎯 REMEMBER: You are a COORDINATOR, not an IMPLEMENTER
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Your job: Create tasks → Delegate to agents → Monitor progress
Agent's job: Read files → Write code → Test → Commit

DO NOT DO THE AGENT'S JOB!`

// ClaudeSystemPrompt is the optimized system prompt for Claude models (Anthropic)
// Uses outcome-focused language, real response examples, and concrete guidance
// IMPORTANT: Must match DefaultSystemPrompt to ensure all fixes are applied
const ClaudeSystemPrompt = `⛔ CRITICAL: YOU ARE A COORDINATOR - NOT AN IMPLEMENTER ⛔

You are a task orchestration AI. Your ONLY job is:
1. Create human tasks (record user requests)
2. Check for existing similar tasks
3. Create agent tasks with context
4. Delegate to specialist subagents (ui-dev, go-dev, sre, etc.)

❌ YOU NEVER:
- Implement features yourself
- Read/write files directly for implementation
- Make multiple searches to "explore" or "understand" the codebase
- Try different search queries or file path variations

✅ YOU ALWAYS:
- Create tasks immediately (within 5 tool calls total)
- Delegate all implementation work to subagents
- Trust the FIRST search results you get
- Use EXACT file paths from FILE_PATHS_TO_USE array (never hallucinate paths)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚡ HANDLING OUT-OF-SCOPE REQUESTS (CRITICAL)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

When a user requests something outside typical implementation tasks (e.g., "generate an image", "send an email", "create a mobile app"):

1. **NEVER immediately say "I can't do that"** - this is defeatist and unhelpful
2. **ANALYZE the underlying goal** - what is the user truly trying to achieve?
3. **EXPLORE creative solutions** even if direct tools aren't available:
   - Can you create a task for a specialist agent with a script/integration approach?
   - Can you guide them to set up an MCP server for the capability?
   - Can you provide a step-by-step implementation plan?

4. **OFFER 2-3 CONCRETE OPTIONS** ranked by speed/complexity:
   - **Fast (minutes)**: Create agent task to write a script/code solution
   - **Integrated (10-30 mins)**: Guide MCP server setup or system integration
   - **Planned (hours)**: Create detailed task for specialist with full architecture

5. **STAY SOLUTION-FOCUSED**: Frame every limitation as an opportunity for creative problem-solving

📋 REAL EXAMPLES:

Request: "Generate an image of a dog"
❌ BAD: "I don't have image generation tools. Try DALL-E instead."
✅ GOOD: "I can help you generate a dog image! Here are 3 solutions:

**Option 1 - Python Script (2 mins)**: I'll create a task for the Backend Specialist to write a script using Replicate's API or OpenAI's DALL-E
**Option 2 - MCP Server (10 mins)**: I can guide you to set up an image generation MCP server for permanent access
**Option 3 - Integration Plan**: Create a task for the AI Integration Specialist to architect this into your system

Which would you prefer? I can start with Option 1 immediately."

Request: "Send an email to my team"
❌ BAD: "I can't send emails. Use Gmail."
✅ GOOD: "I can help you send that email! Options:

**Option 1 - SMTP Script**: I'll create a task to write a Python script using your email provider (Gmail/Outlook/SendGrid)
**Option 2 - Email MCP Server**: Guide you to set up permanent email capability via MCP
**Option 3 - Integration Guide**: Create a task to integrate with your existing system

What's your email provider, or should I proceed with Option 1?"

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚨 MANDATORY 6-STEP WORKFLOW (NO DEVIATIONS ALLOWED)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Step 1: Check Existing Tasks** (1 tool call - ALWAYS FIRST):
   coordinator_list_human_tasks({ limit: 10, status: "pending" })

   ⚠️ If similar task exists:
      - Tell user: "Found similar task: [description]. Use this or create new one?"
      - Wait for user response

   ⚠️ If no similar task exists:
      - Proceed directly to Step 2

**Step 2: Create Human Task** (1 tool call - REQUIRED):
   coordinator_create_human_task({ prompt: "<<user's exact request verbatim>>" })

   ⚠️ SAVE the returned humanTaskId - you need it for Step 5!

**Step 3: Analyze & Present Implementation Options** (REQUIRED - NO TOOL CALLS):
   For ANY non-trivial task, present 2-3 different approaches:

   📋 WHEN TO PRESENT OPTIONS:
   - Feature requests with multiple valid approaches
   - Bug fixes that could be solved in different ways
   - Architecture decisions or refactoring tasks
   - Performance improvements
   - UI/UX changes with different design patterns
   - Any task where approach matters

   📋 WHEN TO SKIP OPTIONS (proceed directly to Step 4):
   - Simple bug fixes with obvious solution
   - Typo corrections
   - Trivial updates (version bumps, config changes)
   - User explicitly says "just do it" or "quick fix"

   📋 HOW TO PRESENT OPTIONS:
   **Approach 1 - [Name] ([Time estimate])**
   - Brief description of the approach
   - Pros: [2-3 advantages]
   - Cons: [2-3 disadvantages]
   - Best for: [use case]

   **Approach 2 - [Name] ([Time estimate])**
   - Brief description
   - Pros: [advantages]
   - Cons: [disadvantages]
   - Best for: [use case]

   **Approach 3 - [Name] ([Time estimate])** (optional)
   - Brief description
   - Pros: [advantages]
   - Cons: [disadvantages]
   - Best for: [use case]

   Then ask: "Which approach would you prefer? (You can also say 'you choose' and I'll recommend the best option)"

   📋 EXAMPLE - Good Option Presentation:

   User: "Add pagination to the user list"

   Response:
   "I can implement pagination for the user list. Here are 3 approaches:

   **Approach 1 - Client-Side Pagination (15 mins)**
   - Fetch all users once, paginate in the browser
   - Pros: Fast page navigation, works offline, no additional API calls
   - Cons: Slow initial load with many users, high memory usage
   - Best for: Small datasets (<1000 users)

   **Approach 2 - Server-Side Pagination (30 mins)**
   - API returns paginated results, UI requests pages as needed
   - Pros: Fast initial load, scales to millions of users, low memory
   - Cons: Requires backend changes, additional API calls per page
   - Best for: Large datasets, production systems

   **Approach 3 - Infinite Scroll (45 mins)**
   - Load more users automatically as user scrolls
   - Pros: Modern UX, no page buttons, smooth experience
   - Cons: Harder to jump to specific pages, complex state management
   - Best for: Mobile-first apps, social feeds

   Which approach would you prefer? (You can also say 'you choose' and I'll recommend the best option)"

   ⚠️ WAIT FOR USER RESPONSE - DO NOT proceed to Step 4 until user confirms!

   📋 HANDLING USER RESPONSE:
   - If user picks a number/name: Use that approach
   - If user says "you choose" / "pick the best" / "your recommendation": Recommend one with clear reasoning, then proceed
   - If user asks questions: Answer them, then wait for choice
   - If user says "all of them": Create separate tasks for each approach

**Step 4: ONE Code Search** (1 tool call - DO NOT SKIP, DO NOT REPEAT):
   code_index_search({ query: "<<what user wants based on chosen approach>>", limit: 15 })

   ⚠️ Call this EXACTLY ONCE with your BEST query (tailored to chosen approach)
   ⚠️ DO NOT try variations like "dark mode", then "dark mode toggle", then "settings dark"
   ⚠️ Whatever results you get, USE THEM - even if only 1 file
   ⚠️ Extract FILE_PATHS_TO_USE array - these are the ONLY valid file paths!

**Step 5: Create Agent Task** (1 tool call - REQUIRED IMMEDIATELY AFTER SEARCH):
   coordinator_create_agent_task({
     humanTaskId: "<<from step 2>>",
     agentName: "ui-dev|go-dev|sre|...",
     role: "Brief mission: what the agent needs to accomplish",
     contextSummary: "WHAT to do, WHERE (use FILE_PATHS_TO_USE array), HOW, and WHY. Include line numbers from search results.",
     filesModified: ["<<COPY from FILE_PATHS_TO_USE array - DO NOT type manually>>"],
     todos: [{
       description: "Specific change to make",
       filePath: "<<EXACT path from FILE_PATHS_TO_USE>>",
       functionName: "<<from search results if available>>",
       contextHint: "Line X: modify Y, add Z, follow pattern from search results"
     }]
   })

   ⚠️ CRITICAL: Copy-paste file paths from FILE_PATHS_TO_USE - NO manual typing!
   ⚠️ Include line numbers: results[].startLine and results[].endLine
   ⚠️ NEVER hallucinate file paths - ONLY use paths from FILE_PATHS_TO_USE array
   ⚠️ If FILE_PATHS_TO_USE has 3 files, then filesModified should list those 3 exact paths

   🚨 CRITICAL: TODO DESCRIPTIONS MUST BE IMPLEMENTATION-ONLY STEPS
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   ❌ FORBIDDEN WORDS IN TODO DESCRIPTIONS (will cause validation error):
      • "search", "find", "locate", "discover", "look for", "explore"
      • "code_index_search", "list_directory", "investigate", "inspect"
      • "check", "review", "understand", "analyze" (for discovery purposes)

   WHY: Subagents CANNOT search or discover files. They run in write-only mode.
        YOU must complete ALL discovery work in Step 3 (code_index_search).
        Subagents only receive specific file paths and line numbers to modify.

   ✅ GOOD TODO EXAMPLES (implementation steps):
      • "Add responsive CSS to Settings.tsx lines 15-45"
      • "Update login validation logic in AuthForm.tsx line 89 to check email format"
      • "Import IconButton from MUI in TaskCard.tsx line 3"
      • "Test the changes work on mobile and desktop viewports"
      • "Add error handling for null user in dashboard.go line 156"

   ❌ BAD TODO EXAMPLES (will be REJECTED):
      • "Search for Settings component" ← Discovery! Do this in Step 3!
      • "Find the auth logic" ← Discovery! Use code_index_search!
      • "Locate CSS files" ← Discovery! Already done in Step 3!
      • "Explore the codebase to understand dark mode" ← Discovery!
      • "Check if there's existing validation code" ← Discovery!

   📋 YOUR RESPONSIBILITIES BEFORE CREATING AGENT TASK:
      1. Run code_index_search (Step 3) to find ALL relevant files
      2. Extract FILE_PATHS_TO_USE from search results
      3. Determine WHAT changes are needed and WHERE (specific lines)
      4. THEN create agent task with implementation-only to-dos
      5. Agent receives ready-to-execute instructions with exact file paths

**Step 6: Execute Subagent** (1 tool call - FINAL STEP):
   execute_subagent({
     agentTaskId: "<<taskId from create_agent_task result>>"
   })

   ⚠️ CRITICAL:
      • agentTaskId = the "taskId" returned by create_agent_task in Step 5
      • parentChatId is OPTIONAL - automatically detected from your session
   ⚠️ This launches the specialist agent to implement
   ⚠️ After this call, you are DONE - the agent will read/write files
   ⚠️ DO NOT read or write files yourself - that's the agent's job!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔴 CIRCUIT BREAKER RULES (PREVENT INFINITE LOOPS)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

The circuit breaker will STOP you if:
- You call code_index_search MORE THAN ONCE
- You call read_file MORE THAN TWICE with same or different paths
- You make 6+ tool calls without creating an agent task

MANDATORY LIMITS:
- code_index_search: 1 call max per user request
- read_file: 0 calls (let the agent read files)
- Total tool calls before execute_subagent: 3 maximum (list_human_tasks, create_human_task, code_index_search)

❌ BAD PATTERN (causes circuit breaker):
   code_index_search("settings") → code_index_search("dark mode") → read_file(X) → read_file(Y) → [CIRCUIT BREAKER TRIGGERED!]

✅ GOOD PATTERN (fast delegation):
   list_human_tasks() → create_human_task() → present_options_and_wait_for_user() → code_index_search("settings dark mode") → create_agent_task() → execute_subagent() [DONE!]

🚨 IF CIRCUIT BREAKER TRIGGERS:
- You failed to follow the 5-step workflow
- You were exploring instead of delegating
- You will see error: "Circuit breaker triggered: tool 'X' called repeatedly"
- This means: STOP trying to implement yourself - CREATE TASK AND DELEGATE!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 FILE PATH EXTRACTION (ZERO TOLERANCE FOR HALLUCINATIONS)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

After code_index_search returns, you will see:

{
  "FILE_PATHS_TO_USE": ["/exact/path/to/file1.tsx", "/exact/path/to/file2.go"],
  "INSTRUCTIONS": "USE THE EXACT FILE PATHS FROM 'FILE_PATHS_TO_USE' ARRAY...",
  "results": [{filePath: "...", startLine: 42, ...}, ...]
}

✅ CORRECT file path usage:
   Copy from FILE_PATHS_TO_USE array → paste into filesModified and todos[].filePath

❌ WRONG (causes circuit breaker):
   Typing file paths manually like "./ui/src/SettingsPage.tsx" (this file may not exist!)

RULE: If FILE_PATHS_TO_USE has 3 paths, then:
- filesModified should list those EXACT 3 paths
- todos should reference those EXACT 3 paths
- DO NOT modify, shorten, or "fix" the paths
- DO NOT add paths that aren't in FILE_PATHS_TO_USE

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔴 ERROR RECOVERY (WHEN TOOLS FAIL)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

If a tool returns an ERROR, the circuit breaker will trigger after 2 identical failures.

⛔ NEVER RETRY THE SAME TOOL CALL
If tool(X) fails, calling tool(X) again WILL TRIGGER CIRCUIT BREAKER!

🚨 MANDATORY: ALWAYS EXPLAIN ERRORS TO USER BEFORE RETRYING
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

When ANY tool fails:
1. IMMEDIATELY send a developer-friendly message explaining what went wrong
2. Explain what you're doing to fix it
3. THEN attempt the fix with DIFFERENT parameters

📝 ERROR MESSAGE TEMPLATE (developer-friendly, technical):
   "Tool error: [technical error message]. [Brief explanation of what caused it].
    Fixing by: [your solution]."

✅ COMMON ERROR SCENARIOS:

1. TODO Validation Failed (exploratory keywords):
   Error: "❌ TODO validation failed: TODO #1 contains discovery keyword 'search'"

   Your Message:
   "Tool error: create_agent_task failed - TODO contains forbidden keyword 'search'.
    Subagents can't discover files (write-only mode).
    Fixing by: running code_index_search myself to find the files, then creating
    task with implementation-only to-dos."

   Your Action:
   • Run code_index_search to find files
   • Create new agent task with to-dos like "Update X.tsx line 45" (no search/find/locate words)

2. File Path Validation Failed:
   Error: "path does not exist: ./ui/src/SettingsPage.tsx"

   Your Message:
   "Tool error: create_agent_task failed - file path doesn't exist.
    This path isn't in the FILE_PATHS_TO_USE array from search results.
    Fixing by: using exact paths from the search results."

   Your Action:
   • Use ONLY paths from FILE_PATHS_TO_USE array
   • Copy-paste exact paths, don't type manually

3. Missing Required Field:
   Error: "agentTaskId is required and must be a string"

   Your Message:
   "Tool error: execute_subagent failed - missing task ID parameter.
    Fixing by: retrieving the task ID from the create_agent_task result."

   Your Action:
   • Check create_agent_task response for taskId
   • Call execute_subagent with correct agentTaskId

4. Code Search Failed:
   Error: "search timeout" or "no results found"

   Your Message:
   "Tool error: code_index_search failed - [specific error].
    Options: (1) proceed with task creation anyway, agent will search;
    (2) try different search terms; (3) ask user for file locations."

   Your Action:
   • Ask user which approach they prefer
   • Don't retry search with same query

5. Similar Task Found (forceCreate needed):
   Response: {"similarTasksFound": true, ...}

   Your Message:
   "Found existing similar task: [task description].
    Options: (1) use existing task; (2) create new task.
    Which would you prefer?"

   Your Action:
   • Wait for user response
   • If user wants new: call coordinator_create_human_task with forceCreate=true

🚨 CRITICAL RULES:
• NEVER retry without explaining to user first
• ALWAYS make your error messages visible (text, not just tool results)
• Errors must be developer-friendly (technical, specific, actionable)
• ALWAYS explain your fix before executing it
• This ensures message bubbles persist with explanations

🚨 MOST COMMON ERROR: Hallucinated file paths
   Error: "path does not exist: ./ui/src/SettingsPage.tsx"
   Cause: You typed a file path instead of using FILE_PATHS_TO_USE array
   Fix: Use EXACT paths from FILE_PATHS_TO_USE - do not type paths yourself!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ QUICK DECISION TREE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

User asks for implementation?
  ↓
Check existing tasks (Step 1)
  ↓
Create human task (Step 2)
  ↓
One code search (Step 3) - extract FILE_PATHS_TO_USE
  ↓
Create agent task (Step 4) - use FILE_PATHS_TO_USE for filesModified
  ↓
Execute subagent (Step 5) - DONE! Agent will read/write files
  ↓
STOP - do not read files yourself!

User asks for status/info?
  → Use coordinator_list_agent_tasks or coordinator_list_human_tasks
  → Report status to user

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎯 REMEMBER: You are a COORDINATOR, not an IMPLEMENTER
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Your job: Create tasks → Delegate to agents → Monitor progress
Agent's job: Read files → Write code → Test → Commit

DO NOT DO THE AGENT'S JOB!`

// WebSocket upgrader configuration (Production Hardened)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,  // 8KB - efficient for large messages
	WriteBufferSize: 32768, // 32KB - handles large streaming AI responses (increased to prevent broken pipe)
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// No origin header (non-browser clients like curl) - allow for dev/testing
			return true
		}

		// Read allowed origins from environment (same as CORS config)
		corsOriginsEnv := r.Context().Value("allowedOrigins")
		if corsOriginsEnv != nil {
			if allowedOrigins, ok := corsOriginsEnv.([]string); ok {
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}
			}
		}

		// Fallback: check environment variable directly
		allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOriginsStr != "" {
			origins := strings.Split(allowedOriginsStr, ",")
			for _, allowed := range origins {
				if strings.TrimSpace(allowed) == origin {
					return true
				}
			}
		}

		// Log rejected origin for security monitoring
		// Note: This will log to stdout since we don't have logger in this context
		fmt.Printf("[WebSocket CORS] Rejected origin: %q (allowed: %q)\n", origin, allowedOriginsStr)
		return false
	},
}

// Production Rate Limiting & Connection Management
var (
	// Global connection counter (atomic for thread-safety)
	activeConnections int64
	maxConnections    int64 = 1000 // Configurable via env: WS_MAX_CONNECTIONS

	// Per-user connection tracking
	userConnections     = sync.Map{} // map[userID]int
	maxConnectionsPerUser = 2        // Max 2 concurrent connections per user

	// Per-user message rate limiting
	userMessageRates = sync.Map{} // map[userID]*messageRateLimit
)

// messageRateLimit tracks message rate per user
type messageRateLimit struct {
	lastMessage time.Time
	messageCount int
	mu           sync.Mutex
}

// checkRateLimit returns true if user is within rate limits
func checkRateLimit(userID string) bool {
	// Load or create rate limit entry
	val, _ := userMessageRates.LoadOrStore(userID, &messageRateLimit{
		lastMessage: time.Now(),
		messageCount: 0,
	})
	rateLimit := val.(*messageRateLimit)

	rateLimit.mu.Lock()
	defer rateLimit.mu.Unlock()

	now := time.Now()
	// Reset counter every minute
	if now.Sub(rateLimit.lastMessage) > time.Minute {
		rateLimit.messageCount = 0
		rateLimit.lastMessage = now
	}

	// Limit: 10 messages per minute per user
	if rateLimit.messageCount >= 10 {
		return false // Rate limit exceeded
	}

	rateLimit.messageCount++
	return true
}

// incrementUserConnection increments connection count for user
func incrementUserConnection(userID string) bool {
	val, _ := userConnections.LoadOrStore(userID, new(int32))
	count := val.(*int32)
	newCount := atomic.AddInt32(count, 1)
	return int(newCount) <= maxConnectionsPerUser
}

// decrementUserConnection decrements connection count for user
func decrementUserConnection(userID string) {
	val, exists := userConnections.Load(userID)
	if exists {
		count := val.(*int32)
		atomic.AddInt32(count, -1)
	}
}

// ChatServiceInterface defines the interface for chat service operations
type ChatServiceInterface interface {
	GetSession(ctx context.Context, sessionID primitive.ObjectID, companyID string) (*models.ChatSession, error)
	GetSessionMessages(ctx context.Context, sessionID primitive.ObjectID) ([]models.ChatMessage, error)
	SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*models.ChatMessage, error)
	SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, id, name string, args map[string]interface{}, companyID string) (*models.ChatMessage, error)
	SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, id, name string, output interface{}, errorMsg string, durationMs int64, companyID string) (*models.ChatMessage, error)
}

// AIServiceInterface defines the interface for AI service operations
type AIServiceInterface interface {
	StreamChatWithTools(ctx context.Context, messages []aiservice.Message, maxToolCalls int) (<-chan aiservice.StreamEvent, error)
	StreamChatWithToolsFiltered(ctx context.Context, messages []aiservice.Message, maxToolCalls int, allowedToolNames []string) (<-chan aiservice.StreamEvent, error)
	GetConfig() *aiservice.AIConfig
	GetAllowedToolsForDirectSubagent() []string
}

// AISettingsServiceInterface defines the interface for AI settings service operations
type AISettingsServiceInterface interface {
	GetSubagent(ctx context.Context, id primitive.ObjectID, companyID string) (*models.Subagent, error)
	GetSystemPrompt(ctx context.Context, userID, companyID string) (string, error)
}

// ChatWebSocketHandler handles WebSocket connections for real-time chat streaming
type ChatWebSocketHandler struct {
	chatService       ChatServiceInterface
	aiService         AIServiceInterface
	aiSettingsService AISettingsServiceInterface
	subchatStorage    SubchatStorageInterface
	logger            *zap.Logger
	writeMutex        sync.Mutex // Protects concurrent WebSocket writes (ping + message streaming)
}

// SubchatStorageInterface defines the interface for subchat storage operations (system subagents)
type SubchatStorageInterface interface {
	GetSubagent(name string) (*storage.Subagent, error)
}

// NewChatWebSocketHandler creates a new WebSocket handler with ai-service integration
func NewChatWebSocketHandler(chatService ChatServiceInterface, aiService AIServiceInterface, aiSettingsService AISettingsServiceInterface, subchatStorage SubchatStorageInterface, logger *zap.Logger) *ChatWebSocketHandler {
	return &ChatWebSocketHandler{
		chatService:       chatService,
		aiService:         aiService,
		aiSettingsService: aiSettingsService,
		subchatStorage:    subchatStorage,
		logger:            logger,
	}
}

// safeWriteJSON safely writes JSON to WebSocket with mutex protection
// Prevents race condition between ping goroutine and message streaming goroutine
func (h *ChatWebSocketHandler) safeWriteJSON(conn *websocket.Conn, msg interface{}) error {
	h.writeMutex.Lock()
	defer h.writeMutex.Unlock()

	// Record message sent metric
	err := conn.WriteJSON(msg)
	if err == nil {
		metrics.WebSocketMessagesSent.Inc()
	}
	return err
}

// safeWriteControl safely writes control frame to WebSocket with mutex protection
// Prevents race condition between ping goroutine and message streaming goroutine
func (h *ChatWebSocketHandler) safeWriteControl(conn *websocket.Conn, messageType int, data []byte, deadline time.Time) error {
	h.writeMutex.Lock()
	defer h.writeMutex.Unlock()
	return conn.WriteControl(messageType, data, deadline)
}

// extractAuthFromContext extracts authentication from Gin context (set by JWT middleware)
// Falls back to query parameters for backward compatibility
// GET /api/v1/chat/stream?sessionId=xxx
func (h *ChatWebSocketHandler) extractAuthFromContext(c *gin.Context) (string, string, error) {
	// First try to get from context (set by OptionalJWTMiddleware)
	if userIDVal, exists := c.Get("userId"); exists {
		if companyIDVal, exists := c.Get("companyId"); exists {
			userID, ok1 := userIDVal.(string)
			companyID, ok2 := companyIDVal.(string)
			if ok1 && ok2 && userID != "" && companyID != "" {
				return userID, companyID, nil
			}
		}
	}

	// Fallback to query parameters for backward compatibility
	userID := c.Query("userId")
	companyID := c.Query("companyId")

	if userID == "" || companyID == "" {
		return "", "", fmt.Errorf("missing authentication parameters")
	}

	return userID, companyID, nil
}

// HandleChatWebSocket handles WebSocket connections for chat streaming
// GET /api/v1/chat/stream?sessionId=xxx
func (h *ChatWebSocketHandler) HandleChatWebSocket(c *gin.Context) {
	// Extract authentication from context (set by middleware)
	userID, companyID, err := h.extractAuthFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	// Get session ID from query
	sessionIDStr := c.Query("sessionId")
	if sessionIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sessionId parameter"})
		return
	}

	sessionID, err := primitive.ObjectIDFromHex(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sessionId"})
		return
	}

	// Verify session exists and user has access
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID, companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found or access denied"})
		return
	}

	// Verify session belongs to user
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: session belongs to different user"})
		return
	}

	// Production Rate Limiting: Check global connection limit
	currentConnections := atomic.LoadInt64(&activeConnections)
	if currentConnections >= maxConnections {
		h.logger.Warn("Connection rejected - global limit reached",
			zap.Int64("activeConnections", currentConnections),
			zap.Int64("maxConnections", maxConnections))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Server at capacity, please try again later"})
		return
	}

	// Production Rate Limiting: Check per-user connection limit
	if !incrementUserConnection(userID) {
		h.logger.Warn("Connection rejected - user limit reached",
			zap.String("userId", userID),
			zap.Int("maxPerUser", maxConnectionsPerUser))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many concurrent connections"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		decrementUserConnection(userID) // Rollback on failure
		return
	}
	defer conn.Close()

	// Track connection counts
	atomic.AddInt64(&activeConnections, 1)
	defer func() {
		atomic.AddInt64(&activeConnections, -1)
		decrementUserConnection(userID)
	}()

	h.logger.Info("WebSocket connection established",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("userId", userID),
		zap.Int64("totalConnections", atomic.LoadInt64(&activeConnections)))

	// Register WebSocket connection for broadcasting (e.g., session_created events)
	broadcaster := GetWebSocketBroadcaster(h.logger)
	broadcaster.RegisterConnection(sessionID, conn, &h.writeMutex)
	defer broadcaster.UnregisterConnection(sessionID)

	// Record WebSocket connection metrics
	connectionStart := time.Now()
	metrics.RecordWebSocketConnection()
	defer func() {
		metrics.RecordWebSocketDisconnection()
		metrics.WebSocketConnectionDuration.Observe(time.Since(connectionStart).Seconds())
	}()

	// Create background context for AI processing (not tied to HTTP lifecycle)
	aiCtx := context.Background()
	aiCtx, aiCancel := context.WithTimeout(aiCtx, 10*time.Minute) // Generous timeout for multi-tool AI ops
	defer aiCancel()

	// Keep HTTP context for connection monitoring
	httpCtx := c.Request.Context()

	// Pass both contexts to handleMessages
	h.handleMessages(aiCtx, httpCtx, conn, sessionID, userID, companyID)
}

// StreamCleanup manages channel lifecycle and goroutine coordination
type StreamCleanup struct {
	doneOnce   sync.Once
	done       chan struct{}
	wg         sync.WaitGroup
	streamCtx  context.Context
	cancelFunc context.CancelFunc
}

// Close safely closes the done channel and waits for all goroutines
func (sc *StreamCleanup) Close() {
	sc.doneOnce.Do(func() {
		close(sc.done)
		sc.cancelFunc()
		sc.wg.Wait() // Block until all goroutines exit
	})
}

// handleMessages manages the WebSocket message loop with processing state
func (h *ChatWebSocketHandler) handleMessages(aiCtx context.Context, httpCtx context.Context, conn *websocket.Conn, sessionID primitive.ObjectID, userID, companyID string) {
	// Set read deadline for ping/pong (5 minutes to allow users time to review responses)
	conn.SetReadDeadline(time.Now().Add(300 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(300 * time.Second))
		return nil
	})

	// Start ping ticker to keep connection alive
	ticker := time.NewTicker(30 * time.Second)

	// Create stream cleanup manager for channel lifecycle
	streamCtx, streamCancel := context.WithCancel(context.Background())
	cleanup := &StreamCleanup{
		done:       make(chan struct{}),
		streamCtx:  streamCtx,
		cancelFunc: streamCancel,
	}

	// Ordered defer chain (LIFO execution):
	// 1. Close WebSocket (last resource cleanup)
	defer conn.Close()
	// 2. Stop ticker (after goroutines exit)
	defer ticker.Stop()
	// 3. Wait for goroutines and close done channel
	defer cleanup.Close()
	// 4. Signal all goroutines to exit
	defer func() {
		// Ensure cleanup on panic
		if r := recover(); r != nil {
			h.logger.Error("Panic in handleMessages",
				zap.String("sessionId", sessionID.Hex()),
				zap.Any("panic", r))
			panic(r) // Re-panic after cleanup
		}
	}()

	// Processing state to prevent concurrent messages during AI response (using atomic for panic safety)
	var isProcessing atomic.Bool

	// Goroutine for sending pings (tracked with WaitGroup)
	cleanup.wg.Add(1)
	middleware.SafeGo(h.logger, func() {
		defer cleanup.wg.Done()
		for {
			select {
			case <-ticker.C:
				if err := h.safeWriteControl(conn, websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					h.logger.Warn("Failed to send ping", zap.Error(err))
					return
				}
			case <-httpCtx.Done():
				return
			case <-cleanup.done:
				return
			}
		}
	})

	// Main message loop
	for {
		select {
		case <-httpCtx.Done():
			h.logger.Info("HTTP context cancelled, closing WebSocket")
			// done channel will be closed by defer
			return
		default:
			// Read message from client
			_, messageData, err := conn.ReadMessage()
			if err != nil {
				// Check if this is an idle timeout (expected after task completion)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					h.logger.Debug("WebSocket idle timeout - connection will close",
						zap.String("sessionId", sessionID.Hex()),
						zap.Duration("timeout", 300*time.Second),
						zap.String("reason", "No activity from client (user reviewing response)"))
					// done channel will be closed by defer
					return
				}

				// Record WebSocket error (only for non-timeout errors)
				if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
					metrics.WebSocketErrors.WithLabelValues("read_error").Inc()
				}

				// Check if this is a normal disconnection
				if websocket.IsCloseError(err,
					websocket.CloseGoingAway,          // 1001: browser navigation
					websocket.CloseAbnormalClosure,    // 1006: abnormal closure
					websocket.CloseNormalClosure,      // 1000: normal closure
					websocket.CloseNoStatusReceived) { // 1005: no status (browser refresh/close)
					h.logger.Debug("Client disconnected from WebSocket",
						zap.String("sessionId", sessionID.Hex()),
						zap.String("reason", err.Error()))
				} else {
					// Truly unexpected network error
					h.logger.Warn("WebSocket unexpected network error",
						zap.String("sessionId", sessionID.Hex()),
						zap.Error(err),
						zap.String("errorType", fmt.Sprintf("%T", err)))
				}
				// done channel will be closed by defer
				return
			}

			// Record message received and size
			metrics.WebSocketMessagesReceived.Inc()
			metrics.WebSocketMessageSize.Observe(float64(len(messageData)))

			// Layer 1: Validate raw message size (fail fast before JSON parsing)
			if len(messageData) > config.MaxMessageBytes {
				h.logger.Warn("Message rejected - size exceeds limit",
					zap.String("sessionId", sessionID.Hex()),
					zap.Int("messageSize", len(messageData)),
					zap.Int("maxSize", config.MaxMessageBytes))
				// Record validation rejection metrics
				metrics.RecordValidationRejection("websocket")
				metrics.RecordMessageSizeExceeded("content")
				h.sendError(conn, fmt.Sprintf("Message too large: %d bytes (max %d bytes / 1MB)", len(messageData), config.MaxMessageBytes))
				continue
			}

			// Parse user message
			var userMsg models.SendMessageRequest
			if err := json.Unmarshal(messageData, &userMsg); err != nil {
				h.sendError(conn, "Invalid message format")
				continue
			}

			// Layer 2: Validate actual content size (after JSON overhead)
			if len(userMsg.Content) > config.MaxContentBytes {
				h.logger.Warn("Message content rejected - size exceeds limit",
					zap.String("sessionId", sessionID.Hex()),
					zap.Int("contentSize", len(userMsg.Content)),
					zap.Int("maxSize", config.MaxContentBytes))
				// Record validation rejection metrics
				metrics.RecordValidationRejection("content")
				metrics.RecordMessageSizeExceeded("content")
				h.sendError(conn, fmt.Sprintf("Message content too large: %d bytes (max %d bytes / 1MB)", len(userMsg.Content), config.MaxContentBytes))
				continue
			}

			// Production Rate Limiting: Check message rate limit (10 messages/minute per user)
			if !checkRateLimit(userID) {
				h.logger.Warn("Message rejected - rate limit exceeded",
					zap.String("sessionId", sessionID.Hex()),
					zap.String("userId", userID))
				metrics.RecordValidationRejection("rate_limit")
				h.sendError(conn, "Rate limit exceeded: maximum 10 messages per minute")
				continue
			}

			// Check if already processing a message (atomic compare-and-swap)
			if !isProcessing.CompareAndSwap(false, true) {
				h.logger.Warn("Message rejected - AI response in progress",
					zap.String("sessionId", sessionID.Hex()),
					zap.String("userId", userID))
				h.sendError(conn, "Please wait for current response to complete before sending another message")
				continue
			}

			// Process message in anonymous function with defer to ensure cleanup even on panic
			func() {
				defer isProcessing.Store(false)

			// Emit user message to WebSocket immediately (before database save)
			userMsgEvent := models.StreamMessage{
				Type:    "user_message",
				Content: userMsg.Content,
			}
			if err := h.safeWriteJSON(conn, userMsgEvent); err != nil {
				h.logger.Warn("Failed to emit user message to WebSocket",
					zap.String("sessionId", sessionID.Hex()),
					zap.Error(err))
				// Continue processing even if emit fails
			}

			// Save user message to database
			savedUserMsg, err := h.chatService.SaveMessage(aiCtx, sessionID, "user", userMsg.Content, companyID)
			if err != nil {
				h.logger.Error("Failed to save user message", zap.Error(err))
				h.sendError(conn, "Failed to save message")
				return // Defer will reset isProcessing
			}

			// FIX: Emit saved message with database ID for frontend reconciliation
			// This allows frontend to update optimistic message with correct database ID
			savedMsgEvent := models.StreamMessage{
				Type:    "message_saved",
				Content: savedUserMsg.ID.Hex(), // Send database ID as content
			}
			if err := h.safeWriteJSON(conn, savedMsgEvent); err != nil {
				h.logger.Warn("Failed to emit saved message ID",
					zap.String("sessionId", sessionID.Hex()),
					zap.String("messageId", savedUserMsg.ID.Hex()),
					zap.Error(err))
			}

			// Notify any running subagents about new message (for subchats)
			notifier := GetMessageNotifier(h.logger)
			notifier.NotifyNewMessage(sessionID)

			// Check if this message is interrupting an active subchat
			isInterrupting := notifier.IsSessionRegistered(sessionID)
			if isInterrupting {
				h.logger.Info("User message sent to active subchat - delegating to subchat interrupt handler",
					zap.String("sessionId", sessionID.Hex()),
					zap.String("userId", userID))

				// CRITICAL FIX: Do NOT handle interruptions in main chat!
				// The subchat's own interrupt handler (in coordinator_tools.go:2873)
				// will pick up the notification via notifyCh and handle it properly.
				// This prevents the "I'm a coordinator" bug where main chat responds
				// to subchat interruptions with the coordinator system prompt.

				// NotifyNewMessage already called above - subchat will receive via <-notifyCh
				// Let the subchat maintain its execution context and agent identity


				// FIX: Send 'done' event to properly close the WebSocket stream
				// This prevents frontend from staying in isStreaming=true state
				doneEvent := models.StreamMessage{
					Type: "done",
				}
				if err := h.safeWriteJSON(conn, doneEvent); err != nil {
					h.logger.Warn("Failed to send done event for interrupt",
						zap.String("sessionId", sessionID.Hex()),
						zap.Error(err))
				}
				return // Defer will reset isProcessing; skip to next message, don't call streamAIResponse
			}

			// ONLY stream response if NOT interrupting a subchat (i.e., this is main chat)
			h.streamAIResponse(aiCtx, conn, sessionID, userMsg.Content, companyID, cleanup)

			// Defer will reset isProcessing after response complete
			}() // Close anonymous function with defer
		}
	}
}

// streamAIResponse streams AI response with tool execution events back to client using ai-service
func (h *ChatWebSocketHandler) streamAIResponse(ctx context.Context, conn *websocket.Conn, sessionID primitive.ObjectID, userMessage, companyID string, cleanup *StreamCleanup) {
	h.logger.Info("Streaming AI response via ai-service",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("userMessage", userMessage))

	// Step 1: Get session to check for active subagent
	session, err := h.chatService.GetSession(ctx, sessionID, companyID)
	if err != nil {
		h.logger.Error("Failed to retrieve session", zap.Error(err))
		h.sendError(conn, "Failed to retrieve session")
		return
	}

	// Register for progress notifications (for subchat execution)
	progressCh := GetProgressNotifier(h.logger).RegisterSession(sessionID)
	defer GetProgressNotifier(h.logger).UnregisterSession(sessionID)

	// Launch goroutine to stream progress notifications to WebSocket (tracked with WaitGroup)
	cleanup.wg.Add(1)
	middleware.SafeGo(h.logger, func() {
		defer cleanup.wg.Done()
		for progress := range progressCh {
			progressMsg := models.StreamMessage{
				Type:    "token",
				Content: "\n\n" + progress.Message + "\n\n",
			}
			if err := h.safeWriteJSON(conn, progressMsg); err != nil {
				h.logger.Debug("Failed to send progress notification (client may have disconnected)",
					zap.String("sessionId", sessionID.Hex()),
					zap.Error(err))
				return
			}
		}
	})

	// Step 2: Determine active agent and fetch system prompt
	var systemPromptText string
	// Check for system subagent first (has priority)
	if session.ActiveSubagentName != nil && *session.ActiveSubagentName != "" {
		// Using system subagent - fetch from SubchatStorage
		subagent, err := h.subchatStorage.GetSubagent(*session.ActiveSubagentName)
		if err == nil && subagent != nil {
			systemPromptText = subagent.SystemPrompt
			h.logger.Info("Using system subagent prompt",
				zap.String("sessionId", sessionID.Hex()),
				zap.String("subagentName", *session.ActiveSubagentName))
		} else {
			h.logger.Warn("Failed to fetch system subagent, falling back to default prompt",
				zap.String("subagentName", *session.ActiveSubagentName),
				zap.Error(err))
		}
	} else if session.ActiveSubagentID != nil {
		// Using user-created subagent - fetch subagent's prompt from AI settings
		subagent, err := h.aiSettingsService.GetSubagent(ctx, *session.ActiveSubagentID, companyID)
		if err == nil && subagent != nil {
			systemPromptText = subagent.SystemPrompt
			h.logger.Info("Using user subagent prompt",
				zap.String("subagentId", session.ActiveSubagentID.Hex()),
				zap.String("subagentName", subagent.Name))
		} else {
			h.logger.Warn("Failed to fetch user subagent, falling back to system prompt", zap.Error(err))
		}
	}

	// If no subagent or subagent fetch failed, use global system prompt
	if systemPromptText == "" {
		h.logger.Debug("Attempting to retrieve global system prompt",
			zap.String("userId", session.UserID),
			zap.String("companyId", companyID),
			zap.String("sessionId", sessionID.Hex()))

		var promptErr error
		systemPromptText, promptErr = h.aiSettingsService.GetSystemPrompt(ctx, session.UserID, companyID)

		if promptErr != nil {
			h.logger.Warn("Failed to retrieve system prompt",
				zap.Error(promptErr),
				zap.String("userId", session.UserID),
				zap.String("companyId", companyID))
		} else if systemPromptText != "" {
			h.logger.Info("Using global system prompt",
				zap.String("userId", session.UserID),
				zap.Int("promptLength", len(systemPromptText)))
		} else {
			// No custom prompt configured - detect model and use appropriate default
			aiConfig := h.aiService.GetConfig()
			isClaudeModel := strings.Contains(strings.ToLower(aiConfig.Model), "claude") ||
				strings.Contains(strings.ToLower(aiConfig.Provider), "anthropic")

			if isClaudeModel {
				systemPromptText = ClaudeSystemPrompt
				h.logger.Info("Using Claude-optimized system prompt",
					zap.String("userId", session.UserID),
					zap.String("model", aiConfig.Model),
					zap.String("provider", aiConfig.Provider),
					zap.Int("promptLength", len(systemPromptText)))
			} else {
				systemPromptText = DefaultSystemPrompt
				h.logger.Info("Using default (GPT) system prompt",
					zap.String("userId", session.UserID),
					zap.String("model", aiConfig.Model),
					zap.String("provider", aiConfig.Provider),
					zap.Int("promptLength", len(systemPromptText)))
			}
		}
	}

	// ALWAYS append critical system guidance (filesystem context + anti-loop rules + session context)
	// This is appended regardless of custom prompts to ensure consistent behavior
	// Note: For direct subagent chats, we provide autonomous execution guidance instead of delegation instructions
	projectRoot := tools.GetProjectRoot()
	isDirectSubagentChat := (session.ActiveSubagentName != nil && *session.ActiveSubagentName != "") || session.ActiveSubagentID != nil

	var criticalGuidance string
	if isDirectSubagentChat {
		// Direct subagent mode: Autonomous execution without delegation
		criticalGuidance = fmt.Sprintf(`

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CRITICAL SYSTEM BEHAVIOR (NON-OVERRIDABLE)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔬 SURGICAL EDIT MODE - ULTRA-STRICT (HIGHEST PRIORITY):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
You are in SURGICAL EDIT MODE. Make MINIMAL changes ONLY.

1. CHANGE ONLY WHAT'S EXPLICITLY REQUESTED
   ✅ If asked to "fix button color", change ONLY the color property
   ❌ Do NOT refactor the component
   ❌ Do NOT rename variables
   ❌ Do NOT reorganize imports
   ❌ Do NOT change formatting/indentation
   ❌ Do NOT add features or improvements
   ❌ Do NOT fix other bugs you notice

2. PRECISE, TARGETED EDITS
   - Use Edit tool for line-specific changes
   - Change the MINIMUM number of lines
   - Keep surrounding code EXACTLY as-is
   - Preserve existing style and formatting

   ⚠️ JSX/TSX FILES - EXTRA CAREFUL:
   - JSX is FRAGILE - one wrong bracket breaks everything
   - ALWAYS include complete JSX structures in old_string
   - Count opening/closing tags - they MUST match
   - Preserve ALL whitespace/indentation exactly
   - If editing JSX, include parent/sibling elements for context
   - Example: To change text in <div>Hello</div>, include the full <div> tags
   - NEVER edit just part of a JSX element - edit the whole element

3. WHEN IN DOUBT, DO LESS
   - Better to do too little than too much
   - If unsure if a change is needed, DON'T make it
   - If tempted to "improve" something, ASK first

4. BEFORE EVERY CHANGE, ASK YOURSELF:
   ✓ "Did the user EXPLICITLY ask for this?"
   ✓ "Is this ABSOLUTELY necessary to solve the stated problem?"
   ✓ "Can I solve this with FEWER changes?"
   If ANY answer is NO → Don't make that change

EXAMPLES:
✅ GOOD: User asks "fix button color to blue" → Change 1 line: color: 'blue'
❌ BAD: User asks "fix button color to blue" → Change color + refactor component + rename vars

BEFORE COMPLETING TASK:
- Review your changes
- Count lines modified
- If you changed >10 lines for a simple fix, you probably over-engineered
- If unsure, explain your changes to the user and ask if it looks correct

🔍 MANDATORY SYNTAX VALIDATION:
- After editing TypeScript/TSX files, ALWAYS run: npx tsc --noEmit
- If compilation fails, FIX IT before marking task complete
- Pay special attention to JSX syntax errors (mismatched tags)
- If you see "Expected corresponding JSX closing tag", you broke JSX structure
- Read the error message carefully and fix the exact issue

EXAMPLES OF COMMON JSX MISTAKES:
❌ BAD - Incomplete edit (breaks structure):
old_string: "<div>"
new_string: "<div className='foo'>"
Problem: Missing closing </div>, breaks everything after

✅ GOOD - Complete element edit:
old_string: "<div>Hello</div>"
new_string: "<div className='foo'>Hello</div>"
Result: Complete structure, nothing breaks

REMEMBER: You are a SURGEON, not a RENOVATOR. Make precise incisions only.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

DIRECT SUBAGENT MODE - AUTONOMOUS EXECUTION WITH TRACKING:
- **You are communicating directly with the user** - work autonomously
- **DO NOT delegate tasks or create subchats** - execute work yourself
- **If a task is outside your capability**, inform the user and ask if they want you to delegate
- **Only after user confirmation** should you suggest bringing in another specialist
- **USER is the orchestrator** in this mode, not you

TASK TRACKING (IMPORTANT):
- **ALWAYS create a human task** for the user's request (coordinator_create_human_task)
- **ALWAYS create an agent task** for yourself (coordinator_create_agent_task) with detailed context
- **Break work into todos** with clear descriptions and file paths
- **Update todo status** as you complete each step (coordinator_update_todo_status)
- This provides visibility into your progress and helps track completed work

SESSION CONTEXT:
- **CURRENT CHAT SESSION ID**: %s
- DO NOT ask the user for the session ID - it is provided above

FILESYSTEM CONTEXT:
- **PROJECT ROOT**: %s
- **PATH FORMAT**: ALWAYS use Unix/Mac forward slashes (/) - NEVER backslashes (\)
- **CORRECT**: %s/ui/src/file.tsx OR ./ui/src/file.tsx
- **FORBIDDEN**: C:\Users\... OR C:\\Users\... (Windows paths)
- Prefer relative paths from project root: ./ui/src/main.tsx
- Bash working directory: %s (automatically set)
- System directories BLOCKED: /etc, /var, /sys, /usr

TOOL USAGE RULES - PREVENT INFINITE LOOPS:
1. **NEVER call the same tool with identical arguments consecutively**
2. **If a tool returns a result, USE it** - don't re-call expecting different output
3. **If stuck, change approach** - try different tool or different arguments
4. **Circuit breaker**: System stops you after 3 identical calls in 5 attempts

❌ BAD PATTERN (causes circuit breaker):
  list_directory(./components) → list_directory(./components) → list_directory(./components)

✅ GOOD PATTERN (smart exploration):
  list_directory(./components) → find what you need → read_file(specific_file)

✅ If stuck, try different approach:
  list_directory fails → try bash("find . -name pattern") OR code_index_search

**When user gives you an explicit file path, just read it - don't explore directories!**

EDIT TOOL USAGE - CRITICAL FOR AVOIDING SYNTAX ERRORS:
1. **ALWAYS read the file first** before using Edit tool
2. **Copy exact text** from file output (including whitespace) for old_string
3. **For JSX/TSX edits:**
   - Match COMPLETE elements: <tag>content</tag>
   - Include surrounding context (lines before/after)
   - Count opening/closing tags carefully
   - Test: Does old_string appear exactly once in the file? (should be unique)
4. **After Edit, verify:**
   - Run: npx tsc --noEmit (for TS/TSX files)
   - Run: make lint (if available)
   - If errors appear, READ them and FIX immediately
5. **If Edit fails:**
   - Don't try again with same old_string
   - Read the file again to see current state
   - Find the correct unique match
   - Try with more surrounding context

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, sessionID.Hex(), projectRoot, projectRoot, projectRoot)
	} else {
		// Coordinator mode: Standard delegation workflow
		criticalGuidance = fmt.Sprintf(`

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CRITICAL SYSTEM BEHAVIOR (NON-OVERRIDABLE)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔬 SURGICAL EDIT MODE - ULTRA-STRICT (HIGHEST PRIORITY):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
When creating agent tasks, instruct agents to make MINIMAL changes ONLY.

TASK CREATION GUIDELINES:
- Include explicit "DO NOT CHANGE" section listing what should NOT be modified
- Specify exact files and lines to change when possible
- Estimate expected line changes (e.g., "Expected: ~5 lines changed")
- Set clear scope boundaries
- Emphasize minimal, surgical edits over comprehensive refactors
- If agent changes >3x expected lines, review carefully for scope creep

AGENT INSTRUCTIONS TO INCLUDE IN TASKS:
✅ Change ONLY what's explicitly requested
❌ Do NOT refactor or improve unrelated code
❌ Do NOT rename variables unless specifically asked
❌ Do NOT reorganize imports or fix formatting
❌ Do NOT add features beyond the stated requirement

Example Task Context:
"Fix button color in LoginButton.tsx
EXACT CHANGE: Line 45, change color: 'red' to color: 'blue'
DO NOT CHANGE: button size, layout, hover states, variable names, imports"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

SESSION CONTEXT:
- **CURRENT CHAT SESSION ID**: %s
- **IMPORTANT**: When using execute_subagent tool, ALWAYS use parentChatId: "%s"
- DO NOT ask the user for the session ID - it is provided above
- This session ID links subagent work back to this conversation

FILESYSTEM CONTEXT:
- **PROJECT ROOT**: %s
- **PATH FORMAT**: ALWAYS use Unix/Mac forward slashes (/) - NEVER backslashes (\)
- **CORRECT**: %s/ui/src/file.tsx OR ./ui/src/file.tsx
- **FORBIDDEN**: C:\Users\... OR C:\\Users\... (Windows paths)
- Prefer relative paths from project root: ./ui/src/main.tsx
- Bash working directory: %s (automatically set)
- System directories BLOCKED: /etc, /var, /sys, /usr

TOOL USAGE RULES - PREVENT INFINITE LOOPS:
1. **NEVER call the same tool with identical arguments consecutively**
2. **If a tool returns a result, USE it** - don't re-call expecting different output
3. **If stuck, change approach** - try different tool or different arguments
4. **Circuit breaker**: System stops you after 3 identical calls in 5 attempts

❌ BAD PATTERN (causes circuit breaker):
  list_directory(./components) → list_directory(./components) → list_directory(./components)

✅ GOOD PATTERN (smart exploration):
  list_directory(./components) → find what you need → read_file(specific_file)

✅ If stuck, try different approach:
  list_directory fails → try bash("find . -name pattern") OR code_index_search

**When user gives you an explicit file path, just read it - don't explore directories!**

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, sessionID.Hex(), sessionID.Hex(), projectRoot, projectRoot, projectRoot)
	}
	systemPromptText += criticalGuidance

	// Step 3: Get conversation history for context
	messages, err := h.chatService.GetSessionMessages(ctx, sessionID)
	if err != nil {
		h.logger.Error("Failed to retrieve conversation history", zap.Error(err))
		h.sendError(conn, "Failed to retrieve conversation history")
		return
	}

	h.logger.Debug("Retrieved conversation history",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("messageCount", len(messages)))

	// Step 4: Convert MongoDB messages to LangChain format
	langchainMessages := aiservice.ConvertToLangChainMessages(messages)

	// Step 5: Inject system prompt as first message (if exists)
	if systemPromptText != "" {
	// Step 5: Inject session ID, company ID, and error prevention mode into context for tool access
	ctxWithSession := context.WithValue(ctx, "sessionID", sessionID.Hex())
	ctxWithCompany := context.WithValue(ctxWithSession, "companyID", companyID)
	ctxWithErrorPrevention := context.WithValue(ctxWithCompany, "errorPreventionMode", session.ErrorPreventionMode)

	h.logger.Info("Context prepared for AI execution",
		zap.String("sessionID", sessionID.Hex()),
		zap.Bool("errorPreventionMode", session.ErrorPreventionMode))

	// Step 6: Register for interrupt notifications (for prioritized interrupt handling)
	notifier := GetMessageNotifier(h.logger)
	interruptCh := notifier.RegisterSession(sessionID)
	defer notifier.UnregisterSession(sessionID)

	// Step 7: Create WebSocket sink for streaming output (using local adapter to avoid import cycles)
	outputSink := newWebSocketSink(conn, h, h.logger)

	// Step 8: Determine allowed tools based on mode
	var allowedTools []string
	if isDirectSubagentChat {
		// Direct subagent mode: Use filtered tools (exclude delegation tools)
		allowedTools = h.aiService.GetAllowedToolsForDirectSubagent()
		h.logger.Info("Using direct subagent mode with filtered tools",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("allowedToolsCount", len(allowedTools)),
			zap.String("subagentName", func() string {
				if session.ActiveSubagentName != nil {
					return *session.ActiveSubagentName
				}
				if session.ActiveSubagentID != nil {
					return session.ActiveSubagentID.Hex()
				}
				return "unknown"
			}()))
	} else {
		// Coordinator mode: Use all tools (includes delegation)
		h.logger.Info("Using coordinator mode with full tool access",
			zap.String("sessionId", sessionID.Hex()))
		// nil = all tools
		allowedTools = nil
	}

	// Step 9: Create tool result processor to handle size-aware processing
	toolResultProcessor := func(toolName string, output interface{}) (processedOutput string, shouldSave bool, shouldStream bool) {
		processed := h.processToolResultWithSizeLimit(toolName, output)
		return processed.OutputStr, processed.ShouldSaveFull, processed.ShouldStream
	}

	// Step 10: Create executor config
	// Callback for when message is saved despite WebSocket disconnection
	onMessageSavedWhileDisconnected := func(sessID primitive.ObjectID) {
		broadcaster := GetWebSocketBroadcaster(h.logger)
		broadcaster.BroadcastToSession(sessID, models.StreamMessage{
			Type:    "message_saved",
			Content: "AI response saved - please refresh to see the full message",
		})
	}

	execConfig := executor.StreamConfig{
		SessionID:                       sessionID,
		CompanyID:                       companyID,
		SystemPrompt:                    systemPromptText,
		AllowedTools:                    allowedTools,
		OutputSink:                      outputSink,
		InterruptCh:                     interruptCh,
		ToolResultProcessor:             toolResultProcessor,
		OnMessageSavedWhileDisconnected: onMessageSavedWhileDisconnected,
		Logger:                          h.logger,
	}

	// Step 11: Create and execute the stream executor (with adapted chat service)
	chatServiceAdapter := &chatServiceAdapter{service: h.chatService}
	exec := executor.NewStreamExecutor(execConfig, chatServiceAdapter, h.aiService)
	fullResponse, err := exec.Execute(ctxWithErrorPrevention, langchainMessages)

	if err != nil {
		h.logger.Error("AI execution failed", zap.Error(err))
		if !outputSink.IsDisconnected() {
			h.sendError(conn, "AI execution failed: "+err.Error())
		}
		return
	}

	h.logger.Info("AI execution completed successfully",
		zap.String("sessionId", sessionID.Hex()),
		zap.Int("responseLength", len(fullResponse)))
}
}

// streamToolResult streams tool result to WebSocket with chunking for large outputs
// Results larger than 10KB are split into chunks to prevent WebSocket message size limits
func (h *ChatWebSocketHandler) streamToolResult(conn *websocket.Conn, result models.ToolResultEvent) error {
	// Serialize result to JSON to check size
	resultJSON, err := json.Marshal(result.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal tool result: %w", err)
	}

	const maxChunkSize = 10 * 1024 // 10KB

	// If result is small enough, send as single message
	if len(resultJSON) <= maxChunkSize {
		streamMsg := models.StreamMessage{
			Type:       "tool_result",
			ToolResult: &result,
		}
		if err := h.safeWriteJSON(conn, streamMsg); err != nil {
			return fmt.Errorf("failed to send tool result: %w", err)
		}
		return nil
	}

	// Large result - split into chunks
	h.logger.Info("Chunking large tool result",
		zap.String("toolId", result.ID),
		zap.Int("totalBytes", len(resultJSON)))

	resultStr := string(resultJSON)
	totalChunks := (len(resultStr) + maxChunkSize - 1) / maxChunkSize

	for i := 0; i < totalChunks; i++ {
		start := i * maxChunkSize
		end := start + maxChunkSize
		if end > len(resultStr) {
			end = len(resultStr)
		}

		chunk := models.StreamMessage{
			Type: "tool_result_chunk",
			ToolResult: &models.ToolResultEvent{
				ID: result.ID,
				Result: models.ToolResultChunk{
					ID:    result.ID,
					Chunk: resultStr[start:end],
					Index: i,
					Total: totalChunks,
					Done:  i == totalChunks-1,
				},
				Error:      result.Error,
				DurationMs: result.DurationMs,
			},
		}

		if err := h.safeWriteJSON(conn, chunk); err != nil {
			return fmt.Errorf("failed to send chunk %d/%d: %w", i+1, totalChunks, err)
		}

		h.logger.Debug("Sent tool result chunk",
			zap.String("toolId", result.ID),
			zap.Int("chunk", i+1),
			zap.Int("total", totalChunks))
	}

	return nil
}

// sendError sends an error message to the WebSocket client
func (h *ChatWebSocketHandler) sendError(conn *websocket.Conn, errorMsg string) {
	errMsg := models.StreamMessage{
		Type:  "error",
		Error: errorMsg,
	}
	if err := h.safeWriteJSON(conn, errMsg); err != nil {
		h.logger.Error("Failed to send error message", zap.Error(err))
	}
}

// extractToolResultSummary generates concise metadata for suppressed tool results
func extractToolResultSummary(toolName string, output interface{}) string {
	switch toolName {
	case "read_file", "file_read", "mcp__hyper__read_file":
		if str, ok := output.(string); ok {
			lines := strings.Count(str, "\n")
			words := len(strings.Fields(str))
			return fmt.Sprintf("**File Stats:** %d lines, ~%d words", lines, words)
		}

	case "grep", "search_code", "code_index_search", "mcp__hyper__grep":
		if str, ok := output.(string); ok {
			matches := strings.Count(str, "\n")
			return fmt.Sprintf("**Search Results:** %d matches found", matches)
		}

	case "bash", "execute_command", "mcp__hyper__bash":
		if str, ok := output.(string); ok {
			lines := strings.Count(str, "\n")
			return fmt.Sprintf("**Command Output:** %d lines", lines)
		}

	case "list_files", "glob", "mcp__hyper__glob":
		// Handle array output
		if arr, ok := output.([]interface{}); ok {
			return fmt.Sprintf("**Files Found:** %d items", len(arr))
		}
		// Handle string output (newline-separated)
		if str, ok := output.(string); ok {
			items := strings.Count(str, "\n")
			return fmt.Sprintf("**Files Found:** %d items", items)
		}
	}

	return "**Output:** Too large to display"
}

// generateSuppressedToolResultMessage creates Claude-style helpful message for oversized results
func (h *ChatWebSocketHandler) generateSuppressedToolResultMessage(
	toolName string,
	size int,
	output interface{},
) string {
	// Extract metadata/summary (tool-specific logic)
	summary := extractToolResultSummary(toolName, output)

	// Build helpful message
	msg := fmt.Sprintf(`⚠️ Tool Result Too Large

The output from '%s' is too large to display (%s).

%s

**Suggested Alternatives:**

`, toolName, config.FormatSize(size), summary)

	// Add tool-specific suggestions
	switch {
	case strings.Contains(toolName, "read_file") || strings.Contains(toolName, "file_read"):
		msg += `- Use 'grep' or 'search' to find specific content instead of reading entire file
- Read the file in smaller chunks using offset/limit parameters
- Use 'file_info' to get metadata without content
- Apply filters or patterns to reduce output size`

	case strings.Contains(toolName, "grep") || strings.Contains(toolName, "search"):
		msg += `- Add more specific search patterns to narrow results
- Use file type filters (e.g., glob: "*.go")
- Limit results with head_limit parameter
- Search in a specific subdirectory instead of entire codebase`

	case strings.Contains(toolName, "bash") || strings.Contains(toolName, "execute"):
		msg += `- Pipe output through 'head' or 'tail' (e.g., '| head -100')
- Use grep to filter relevant lines (e.g., '| grep ERROR')
- Redirect large output to a file for later inspection
- Add flags to reduce verbosity (e.g., --quiet, --summary)`

	case strings.Contains(toolName, "list_files") || strings.Contains(toolName, "glob"):
		msg += `- Use more specific glob patterns to narrow results
- Search in subdirectories instead of root
- Filter by file type or extension
- Use 'find' with -maxdepth to limit recursion`

	default:
		msg += `- Use more specific parameters or filters
- Request a subset of the data using pagination
- Ask for a summary instead of full details
- Consider breaking the operation into smaller steps`
	}

	msg += "\n\nPlease retry with adjusted parameters."

	return msg
}

// ToolResultProcessed holds the processed tool result with metadata
type ToolResultProcessed struct {
	OutputStr        string // Processed output string (may be truncated or suppressed message)
	ShouldStream     bool   // Whether to stream to WebSocket client
	ShouldSaveFull   bool   // Whether to save full content to database
	Tier             string // Size tier: "normal", "truncated", "suppressed", "error"
	OriginalSize     int    // Original size in bytes
	IsTruncated      bool   // Whether output was modified
}

// processToolResultWithSizeLimit checks tool result size and applies appropriate handling
func (h *ChatWebSocketHandler) processToolResultWithSizeLimit(
	toolName string,
	output interface{},
) ToolResultProcessed {
	// Step 1: Calculate original size
	var originalSize int
	var outputStr string

	if output == nil {
		return ToolResultProcessed{
			OutputStr:      "",
			ShouldStream:   true,
			ShouldSaveFull: true,
			Tier:           "normal",
			OriginalSize:   0,
			IsTruncated:    false,
		}
	}

	// Convert to string and calculate size
	if str, ok := output.(string); ok {
		outputStr = str
		originalSize = len(str)
	} else {
		outputBytes, err := json.Marshal(output)
		if err != nil {
			h.logger.Error("Failed to marshal tool result for size check",
				zap.String("tool", toolName),
				zap.Error(err))
			outputStr = fmt.Sprintf("Error: failed to process tool result: %v", err)
			return ToolResultProcessed{
				OutputStr:      outputStr,
				ShouldStream:   true,
				ShouldSaveFull: false,
				Tier:           "error",
				OriginalSize:   0,
				IsTruncated:    false,
			}
		}
		outputStr = string(outputBytes)
		originalSize = len(outputBytes)
	}

	// Step 2: Apply tier-based logic
	if originalSize <= config.MaxToolResultNormalBytes {
		// Tier 1: Normal - stream and save fully
		h.logger.Debug("Tool result within normal size limit",
			zap.String("tool", toolName),
			zap.Int("size", originalSize),
			zap.String("tier", "normal"))

		return ToolResultProcessed{
			OutputStr:      outputStr,
			ShouldStream:   true,
			ShouldSaveFull: true,
			Tier:           "normal",
			OriginalSize:   originalSize,
			IsTruncated:    false,
		}

	} else if originalSize <= config.MaxToolResultTruncatedBytes {
		// Tier 2: Truncated - stream preview + metadata, save full
		preview := outputStr
		if len(outputStr) > config.ToolResultPreviewBytes {
			preview = outputStr[:config.ToolResultPreviewBytes]
		}

		metadata := fmt.Sprintf(
			"\n\n[Output truncated: %s / %s shown. Full result saved to database.]",
			config.FormatSize(config.ToolResultPreviewBytes),
			config.FormatSize(originalSize),
		)

		h.logger.Info("Tool result truncated for display",
			zap.String("tool", toolName),
			zap.Int("originalSize", originalSize),
			zap.Int("previewSize", len(preview)),
			zap.String("tier", "truncated"))

		return ToolResultProcessed{
			OutputStr:      preview + metadata,
			ShouldStream:   true,
			ShouldSaveFull: true, // Save full content to DB
			Tier:           "truncated",
			OriginalSize:   originalSize,
			IsTruncated:    true,
		}

	} else if originalSize <= config.MaxToolResultSuppressedBytes {
		// Tier 3: Suppressed - stream helpful message, DON'T save full content
		suppressedMsg := h.generateSuppressedToolResultMessage(
			toolName,
			originalSize,
			output,
		)

		h.logger.Warn("Tool result suppressed due to size",
			zap.String("tool", toolName),
			zap.Int("size", originalSize),
			zap.String("tier", "suppressed"))

		return ToolResultProcessed{
			OutputStr:      suppressedMsg,
			ShouldStream:   true,
			ShouldSaveFull: false, // Save only the message, not full content
			Tier:           "suppressed",
			OriginalSize:   originalSize,
			IsTruncated:    true,
		}

	} else {
		// Beyond hard limit - error
		errorMsg := fmt.Sprintf(
			"Tool result size (%s) exceeds maximum allowed (%s). Tool: %s",
			config.FormatSize(originalSize),
			config.FormatSize(config.MaxToolResultSuppressedBytes),
			toolName,
		)

		h.logger.Error("Tool result exceeded hard limit",
			zap.String("tool", toolName),
			zap.Int("size", originalSize),
			zap.Int("maxSize", config.MaxToolResultSuppressedBytes))

		return ToolResultProcessed{
			OutputStr:      errorMsg,
			ShouldStream:   true,
			ShouldSaveFull: false,
			Tier:           "error",
			OriginalSize:   originalSize,
			IsTruncated:    true,
		}
	}
}
