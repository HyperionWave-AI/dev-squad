package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/tools"
	"hyper/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

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
🚨 MANDATORY 5-STEP WORKFLOW (NO DEVIATIONS ALLOWED)
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

   ⚠️ SAVE the returned humanTaskId - you need it for Step 4!

**Step 3: ONE Code Search** (1 tool call - DO NOT SKIP, DO NOT REPEAT):
   code_index_search({ query: "<<what user wants>>", limit: 15 })

   ⚠️ Call this EXACTLY ONCE with your BEST query
   ⚠️ DO NOT try variations like "dark mode", then "dark mode toggle", then "settings dark"
   ⚠️ Whatever results you get, USE THEM - even if only 1 file
   ⚠️ Extract FILE_PATHS_TO_USE array - these are the ONLY valid file paths!

**Step 4: Create Agent Task** (1 tool call - REQUIRED IMMEDIATELY AFTER SEARCH):
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

**Step 5: Execute Subagent** (1 tool call - FINAL STEP):
   execute_subagent({
     agentTaskId: "<<taskId from create_agent_task result>>"
   })

   ⚠️ CRITICAL:
      • agentTaskId = the "taskId" returned by create_agent_task in Step 4
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
- Total tool calls before execute_subagent: 5 maximum

❌ BAD PATTERN (causes circuit breaker):
   code_index_search("settings") → code_index_search("dark mode") → read_file(X) → read_file(Y) → [CIRCUIT BREAKER TRIGGERED!]

✅ GOOD PATTERN (fast delegation):
   list_human_tasks() → create_human_task() → code_index_search("settings dark mode") → create_agent_task() → execute_subagent() [DONE!]

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
🚨 MANDATORY 5-STEP WORKFLOW (NO DEVIATIONS ALLOWED)
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

   ⚠️ SAVE the returned humanTaskId - you need it for Step 4!

**Step 3: ONE Code Search** (1 tool call - DO NOT SKIP, DO NOT REPEAT):
   code_index_search({ query: "<<what user wants>>", limit: 15 })

   ⚠️ Call this EXACTLY ONCE with your BEST query
   ⚠️ DO NOT try variations like "dark mode", then "dark mode toggle", then "settings dark"
   ⚠️ Whatever results you get, USE THEM - even if only 1 file
   ⚠️ Extract FILE_PATHS_TO_USE array - these are the ONLY valid file paths!

**Step 4: Create Agent Task** (1 tool call - REQUIRED IMMEDIATELY AFTER SEARCH):
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

**Step 5: Execute Subagent** (1 tool call - FINAL STEP):
   execute_subagent({
     agentTaskId: "<<taskId from create_agent_task result>>"
   })

   ⚠️ CRITICAL:
      • agentTaskId = the "taskId" returned by create_agent_task in Step 4
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
- Total tool calls before execute_subagent: 5 maximum

❌ BAD PATTERN (causes circuit breaker):
   code_index_search("settings") → code_index_search("dark mode") → read_file(X) → read_file(Y) → [CIRCUIT BREAKER TRIGGERED!]

✅ GOOD PATTERN (fast delegation):
   list_human_tasks() → create_human_task() → code_index_search("settings dark mode") → create_agent_task() → execute_subagent() [DONE!]

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

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// TODO: Restrict in production based on allowed origins
		return true
	},
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
	logger            *zap.Logger
}

// NewChatWebSocketHandler creates a new WebSocket handler with ai-service integration
func NewChatWebSocketHandler(chatService ChatServiceInterface, aiService AIServiceInterface, aiSettingsService AISettingsServiceInterface, logger *zap.Logger) *ChatWebSocketHandler {
	return &ChatWebSocketHandler{
		chatService:       chatService,
		aiService:         aiService,
		aiSettingsService: aiSettingsService,
		logger:            logger,
	}
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

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}
	defer conn.Close()

	h.logger.Info("WebSocket connection established",
		zap.String("sessionId", sessionID.Hex()),
		zap.String("userId", userID))

	// Create background context for AI processing (not tied to HTTP lifecycle)
	aiCtx := context.Background()
	aiCtx, aiCancel := context.WithTimeout(aiCtx, 10*time.Minute) // Generous timeout for multi-tool AI ops
	defer aiCancel()

	// Keep HTTP context for connection monitoring
	httpCtx := c.Request.Context()

	// Pass both contexts to handleMessages
	h.handleMessages(aiCtx, httpCtx, conn, sessionID, userID, companyID)
}

// handleMessages manages the WebSocket message loop with processing state
func (h *ChatWebSocketHandler) handleMessages(aiCtx context.Context, httpCtx context.Context, conn *websocket.Conn, sessionID primitive.ObjectID, userID, companyID string) {
	// Set read deadline for ping/pong
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Channel for handling graceful shutdown
	done := make(chan struct{})

	// Processing state to prevent concurrent messages during AI response
	isProcessing := false
	var processingMutex sync.Mutex

	// Goroutine for sending pings
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					h.logger.Warn("Failed to send ping", zap.Error(err))
					return
				}
			case <-httpCtx.Done():
				return
			case <-done:
				return
			}
		}
	}()

	// Main message loop
	for {
		select {
		case <-httpCtx.Done():
			h.logger.Info("HTTP context cancelled, closing WebSocket")
			close(done)
			return
		default:
			// Read message from client
			_, messageData, err := conn.ReadMessage()
			if err != nil {
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
					// Truly unexpected error
					h.logger.Warn("WebSocket unexpected error",
						zap.String("sessionId", sessionID.Hex()),
						zap.Error(err))
				}
				close(done)
				return
			}

			// Parse user message
			var userMsg models.SendMessageRequest
			if err := json.Unmarshal(messageData, &userMsg); err != nil {
				h.sendError(conn, "Invalid message format")
				continue
			}

			// Check if already processing a message
			processingMutex.Lock()
			if isProcessing {
				processingMutex.Unlock()
				h.logger.Warn("Message rejected - AI response in progress",
					zap.String("sessionId", sessionID.Hex()),
					zap.String("userId", userID))
				h.sendError(conn, "Please wait for current response to complete before sending another message")
				continue
			}
			isProcessing = true
			processingMutex.Unlock()

			// Save user message to database
			_, err = h.chatService.SaveMessage(aiCtx, sessionID, "user", userMsg.Content, companyID)
			if err != nil {
				h.logger.Error("Failed to save user message", zap.Error(err))
				h.sendError(conn, "Failed to save message")
				processingMutex.Lock()
				isProcessing = false
				processingMutex.Unlock()
				continue
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

				// Reset processing state and wait for next message
				processingMutex.Lock()
				isProcessing = false
				processingMutex.Unlock()
				continue // Skip to next message, don't call streamAIResponse
			}

			// ONLY stream response if NOT interrupting a subchat (i.e., this is main chat)
			h.streamAIResponse(aiCtx, conn, sessionID, userMsg.Content, companyID)

			// Reset processing state after response complete
			processingMutex.Lock()
			isProcessing = false
			processingMutex.Unlock()
		}
	}
}

// streamAIResponse streams AI response with tool execution events back to client using ai-service
func (h *ChatWebSocketHandler) streamAIResponse(ctx context.Context, conn *websocket.Conn, sessionID primitive.ObjectID, userMessage, companyID string) {
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

	// Launch goroutine to stream progress notifications to WebSocket
	go func() {
		for progress := range progressCh {
			progressMsg := models.StreamMessage{
				Type:    "token",
				Content: "\n\n" + progress.Message + "\n\n",
			}
			if err := conn.WriteJSON(progressMsg); err != nil {
				h.logger.Debug("Failed to send progress notification (client may have disconnected)",
					zap.String("sessionId", sessionID.Hex()),
					zap.Error(err))
				return
			}
		}
	}()

	// Step 2: Determine active agent and fetch system prompt
	var systemPromptText string
	if session.ActiveSubagentID != nil {
		// Using custom subagent - fetch subagent's prompt
		subagent, err := h.aiSettingsService.GetSubagent(ctx, *session.ActiveSubagentID, companyID)
		if err == nil && subagent != nil {
			systemPromptText = subagent.SystemPrompt
			h.logger.Info("Using subagent prompt",
				zap.String("subagentId", session.ActiveSubagentID.Hex()),
				zap.String("subagentName", subagent.Name))
		} else {
			h.logger.Warn("Failed to fetch subagent, falling back to system prompt", zap.Error(err))
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
	projectRoot := tools.GetProjectRoot()
	criticalGuidance := fmt.Sprintf(`

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
CRITICAL SYSTEM BEHAVIOR (NON-OVERRIDABLE)
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
		// Prepend system message
		systemMessage := aiservice.Message{
			Role:    "system",
			Content: systemPromptText,
		}
		langchainMessages = append([]aiservice.Message{systemMessage}, langchainMessages...)

		h.logger.Debug("Injected system prompt",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("promptLength", len(systemPromptText)))
	}

	// Step 6: Stream AI response via ai-service with tool support
	// Inject session ID into context for tool access (e.g., execute_subagent)
	ctxWithSession := context.WithValue(ctx, "sessionID", sessionID.Hex())
	maxToolCalls := h.aiService.GetConfig().MaxToolCalls
	aiStream, err := h.aiService.StreamChatWithTools(ctxWithSession, langchainMessages, maxToolCalls)
	if err != nil {
		h.logger.Error("Failed to get AI response", zap.Error(err))
		h.sendError(conn, "Failed to get AI response: "+err.Error())
		return
	}

	// Step 7: Stream mixed content (tokens and tool events) to WebSocket client
	fullResponse := ""
	tokenCount := 0
	toolCallCount := 0
	clientDisconnected := false // Track client disconnect state

	for event := range aiStream {
		select {
		case <-ctx.Done():
			h.logger.Info("Context cancelled during streaming",
				zap.String("sessionId", sessionID.Hex()),
				zap.Int("tokensStreamed", tokenCount),
				zap.Int("toolCalls", toolCallCount))
			return
		default:
			// Handle different event types
			switch event.Type {
			case aiservice.StreamEventToken:
				// Accumulate response even if client disconnected
				fullResponse += event.Content
				tokenCount++

				// Try to send to WebSocket if client still connected
				if !clientDisconnected {
					streamMsg := models.StreamMessage{
						Type:    "token",
						Content: event.Content,
					}
					if err := conn.WriteJSON(streamMsg); err != nil {
						// Check if this is a normal disconnection (client closed browser/refreshed)
						if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
							h.logger.Debug("Client disconnected during streaming - continuing processing in background",
								zap.String("sessionId", sessionID.Hex()),
								zap.Int("tokensStreamed", tokenCount))
							clientDisconnected = true // Set flag and continue processing
						} else {
							h.logger.Warn("Failed to send token to WebSocket - continuing processing",
								zap.String("sessionId", sessionID.Hex()),
								zap.Error(err))
							clientDisconnected = true // Assume client is gone
						}
						// Don't return - continue processing to save to database
					}
				}

			case aiservice.StreamEventToolCall:
				// AI is requesting a tool execution
				toolCallCount++

				// Save tool call to database (always, even if client disconnected)
				_, err := h.chatService.SaveToolCall(ctx, sessionID, event.ToolCall.ID, event.ToolCall.Name, event.ToolCall.Args, companyID)
				if err != nil {
					h.logger.Error("Failed to save tool call to database", zap.Error(err))
					// Continue streaming even if save fails
				}

				// Send tool call to WebSocket client if still connected
				if !clientDisconnected {
					streamMsg := models.StreamMessage{
						Type: "tool_call",
						ToolCall: &models.ToolCallEvent{
							Tool: event.ToolCall.Name,
							Args: event.ToolCall.Args,
							ID:   event.ToolCall.ID,
						},
					}
					if err := conn.WriteJSON(streamMsg); err != nil {
						// Check if this is a normal disconnection
						if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
							h.logger.Debug("Client disconnected during tool call streaming - continuing processing",
								zap.String("sessionId", sessionID.Hex()))
							clientDisconnected = true
						} else {
							h.logger.Warn("Failed to send tool call to WebSocket - continuing processing",
								zap.String("sessionId", sessionID.Hex()),
								zap.Error(err))
							clientDisconnected = true
						}
						// Don't return - continue processing
					}
				}

			case aiservice.StreamEventToolResult:
				// Tool execution completed

				// Convert output to string for database storage
				outputStr := ""
				if event.ToolResult.Output != nil {
					if str, ok := event.ToolResult.Output.(string); ok {
						outputStr = str
					} else {
						// Marshal non-string outputs to JSON
						outputBytes, _ := json.Marshal(event.ToolResult.Output)
						outputStr = string(outputBytes)
					}
				}

				// Save tool result to database (always, even if client disconnected)
				_, err := h.chatService.SaveToolResult(ctx, sessionID, event.ToolResult.ID, event.ToolResult.Name, outputStr, event.ToolResult.Error, event.ToolResult.DurationMs, companyID)
				if err != nil {
					h.logger.Error("Failed to save tool result to database", zap.Error(err))
					// Continue streaming even if save fails
				}

				// Send tool result to WebSocket client if still connected
				if !clientDisconnected {
					streamMsg := models.StreamMessage{
						Type: "tool_result",
						ToolResult: &models.ToolResultEvent{
							ID:         event.ToolResult.ID,
							Result:     event.ToolResult.Output,
							Error:      event.ToolResult.Error,
							DurationMs: int(event.ToolResult.DurationMs),
						},
					}
					if err := conn.WriteJSON(streamMsg); err != nil {
						// Check if this is a normal disconnection
						if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
							h.logger.Debug("Client disconnected during tool result streaming - continuing processing",
								zap.String("sessionId", sessionID.Hex()))
							clientDisconnected = true
						} else {
							h.logger.Warn("Failed to send tool result to WebSocket - continuing processing",
								zap.String("sessionId", sessionID.Hex()),
								zap.Error(err))
							clientDisconnected = true
						}
						// Don't return - continue processing
					}
				}

			case aiservice.StreamEventError:
				// Error during processing
				h.logger.Error("AI service error during streaming", zap.String("error", event.Error))
				h.sendError(conn, "AI error: "+event.Error)
				return
			}
		}
	}

	// Step 8: Send completion message (if client still connected)
	if !clientDisconnected {
		doneMsg := models.StreamMessage{
			Type:    "done",
			Content: "",
		}
		if err := conn.WriteJSON(doneMsg); err != nil {
			// Check if this is a normal disconnection
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				h.logger.Debug("Client disconnected before completion message",
					zap.String("sessionId", sessionID.Hex()))
			} else {
				h.logger.Warn("Failed to send done message", zap.Error(err))
			}
			clientDisconnected = true
			// Don't return - continue to save response to database
		}
	}

	// Step 9: Save AI response to database (ALWAYS, even if client disconnected)
	_, err = h.chatService.SaveMessage(ctx, sessionID, "assistant", fullResponse, companyID)
	if err != nil {
		h.logger.Error("Failed to save AI response", zap.Error(err))
		// Only try to send error if client still connected
		if !clientDisconnected {
			h.sendError(conn, "Failed to save AI response")
		}
		return
	}

	if clientDisconnected {
		h.logger.Info("AI response completed in background after client disconnect",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("tokensStreamed", tokenCount),
			zap.Int("toolCalls", toolCallCount),
			zap.Int("responseLength", len(fullResponse)))
	} else {
		h.logger.Info("AI response streamed successfully",
			zap.String("sessionId", sessionID.Hex()),
			zap.Int("tokensStreamed", tokenCount),
			zap.Int("toolCalls", toolCallCount),
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
		if err := conn.WriteJSON(streamMsg); err != nil {
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

		if err := conn.WriteJSON(chunk); err != nil {
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
	if err := conn.WriteJSON(errMsg); err != nil {
		h.logger.Error("Failed to send error message", zap.Error(err))
	}
}
