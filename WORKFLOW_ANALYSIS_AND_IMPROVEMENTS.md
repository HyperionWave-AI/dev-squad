# Workflow Logic Analysis & Improvement Recommendations

## Executive Summary

After analyzing the codebase, I've identified **7 critical gaps** that cause Anthropic models to get stuck, unable to find files, fail to delegate work, and loop endlessly. This document provides specific, actionable solutions to make the orchestration work reliably.

---

## 🔴 **CRITICAL GAPS IDENTIFIED**

### **Gap 1: System Prompt is Too Prescriptive & Rigid**

**Problem:**
- 198-line system prompt tries to micromanage every step
- Forces a strict 5-step workflow that doesn't fit all scenarios
- Contains abstract concepts like "FILE_PATHS_TO_USE array" that don't exist in tool responses
- Uses military-style warnings (⛔, 🚨, ❌) that don't change model behavior

**Evidence:**
```go
// chat_websocket.go:23-198
const DefaultSystemPrompt = `⛔ CRITICAL: YOU ARE A COORDINATOR - NOT AN IMPLEMENTER ⛔
...
**Step 1: Check Existing Tasks** (1 tool call - ALWAYS FIRST):
**Step 2: Create Human Task** (1 tool call - REQUIRED):
...
```

**Why it Fails with Anthropic:**
- Claude works best with **outcome-focused** prompts, not step-by-step scripts
- Strict ordering prevents Claude from adapting to context
- Too many rules create decision paralysis
- Abstract concepts like "FILE_PATHS_TO_USE" don't map to actual tool outputs

**Solution:**
```markdown
# RECOMMENDED SYSTEM PROMPT (50 lines max)

You are a Task Coordinator for a development platform. Your job:

## Core Responsibilities
1. **Understand** what the user wants to accomplish
2. **Create tasks** to track the work (human task + agent task)
3. **Find relevant files** using code search (1 search, accept first results)
4. **Delegate to specialists** by calling execute_subagent

## Key Principles
- **Trust first results**: Don't retry searches or reads hoping for better data
- **Delegate quickly**: After finding files, create agent task and delegate
- **Use exact file paths**: Copy file paths exactly from tool responses
- **One search only**: Do ONE code_index_search, use whatever it returns
- **No implementation**: You coordinate; subagents implement

## Workflow (flexible order based on context)
1. Check if similar task exists → coordinator_list_human_tasks
2. Create human task → coordinator_create_human_task
3. Find relevant files (optional) → code_index_search
4. Create agent task with context → create_agent_task
5. Launch specialist → execute_subagent

## Tool Response Format
- code_index_search returns: { results: [{filePath: "...", ...}] }
- Use results[].filePath directly in filesModified array
- If search returns 0 results, proceed without files or ask user

## Error Recovery
- Tool fails once: try different parameters
- Tool fails twice with same params: report to user and stop
- Can't find files: ask user for hints or proceed with best guess
```

---

### **Gap 2: FILE_PATHS_TO_USE Array Doesn't Exist**

**Problem:**
- System prompt mentions "FILE_PATHS_TO_USE array" 15+ times
- This array is NEVER returned by any tool
- Models hallucinate its existence and get confused
- Causes file path errors and blocks

**Evidence:**
```go
// chat_websocket.go:131-138
After code_index_search returns, you will see:
{
  "FILE_PATHS_TO_USE": ["/exact/path/to/file1.tsx", ...],  // ← THIS NEVER EXISTS
  "INSTRUCTIONS": "USE THE EXACT FILE PATHS...",
  "results": [{filePath: "...", startLine: 42, ...}, ...]
}
```

**Actual Tool Response:**
```json
{
  "results": [
    {"filePath": "/real/path.go", "content": "...", "score": 0.95}
  ],
  "query": "dark mode",
  "totalResults": 5
}
```

**Why it Fails:**
- Claude tries to find a non-existent field
- Falls back to hallucinating file paths
- Causes "path does not exist" errors
- Circuit breaker triggers on retries

**Solution:**
1. **Remove all mentions of FILE_PATHS_TO_USE from prompts**
2. **Add clear guidance on actual response structure:**

```markdown
## How to Extract File Paths from code_index_search

When you call code_index_search, you get:
```json
{
  "results": [
    {"filePath": "/absolute/path/file.go", "content": "...", "score": 0.9},
    {"filePath": "/absolute/path/other.ts", "content": "...", "score": 0.8}
  ]
}
```

To use in create_agent_task:
```json
{
  "filesModified": [
    "/absolute/path/file.go",
    "/absolute/path/other.ts"
  ]
}
```

**Copy the filePath values EXACTLY - don't modify, shorten, or "fix" them.**
```

3. **Post-process code_index_search responses to make paths obvious:**

```go
// Add to code_index_search tool response formatting:
func formatCodeSearchResponse(results []SearchResult) map[string]interface{} {
    // Extract just the file paths for easy copying
    filePaths := make([]string, 0, len(results))
    for _, r := range results {
        filePaths = append(filePaths, r.FilePath)
    }

    return map[string]interface{}{
        "results": results,
        "filePaths": filePaths,  // ← Add this for easy extraction
        "instructions": "Use the exact paths from 'filePaths' array in your filesModified field",
    }
}
```

---

### **Gap 3: Circuit Breaker Triggers Too Aggressively**

**Problem:**
- Circuit breaker stops execution after 2-4 duplicate calls
- Doesn't distinguish between legitimate refinement vs. actual loops
- Provides warnings but models don't always understand them
- Blocks valid workflows (e.g., reading multiple related files)

**Evidence:**
```go
// langchain_service.go:380-387
circuitBreakerThresholds := map[string]int{
    "read_file":         2, // Stop after 2 attempts
    "write_file":        1, // Never allow duplicates
    "list_directory":    2,
    "code_index_search": 3,
}
```

**Why it Fails:**
- Sometimes models need to read 2+ files legitimately
- Threshold of 2 means only 1 duplicate allowed (too strict)
- Error messages are technical, not actionable for models

**Solution:**

1. **Distinguish between identical calls vs. valid variations:**

```go
// Improved circuit breaker logic
type CircuitBreaker struct {
    identicalCalls map[string]int  // Exact signature
    sameTool      map[string]int  // Tool name only
}

func (cb *CircuitBreaker) ShouldBlock(toolName string, signature string) (bool, string) {
    cb.identicalCalls[signature]++
    cb.sameTool[toolName]++

    // Block IDENTICAL calls after 2 attempts
    if cb.identicalCalls[signature] >= 2 {
        return true, fmt.Sprintf("You called '%s' with IDENTICAL arguments twice. This will never give different results. Try different parameters or a different approach.", toolName)
    }

    // Warn if same tool used 5+ times (but allow it)
    if cb.sameTool[toolName] >= 5 {
        return false, fmt.Sprintf("⚠️ You've used '%s' %d times. Consider if you have enough information to proceed.", toolName, cb.sameTool[toolName])
    }

    return false, ""
}
```

2. **Make warnings more actionable:**

```go
// Instead of: "Circuit breaker triggered"
// Use: "You've already called read_file('/path/to/file.go'). The result is in your previous response. Use that information instead of calling again."
```

3. **Increase thresholds for coordinator chat:**

```go
circuitBreakerThresholds := map[string]int{
    "read_file":         4, // Allow reading multiple files
    "code_index_search": 2, // Strict: one search + one retry max
    "list_directory":    3, // Allow exploring a bit
    "create_agent_task": 1, // Should only create once
}
```

---

### **Gap 4: No Feedback Loop from Subagent to Main Chat**

**Problem:**
- execute_subagent launches goroutine and returns immediately
- Main chat has NO visibility into subagent progress
- User never sees updates from subagent
- Main chat can't track completion or failures

**Evidence:**
```go
// coordinator_tools.go:1753
func (t *ExecuteSubagentTool) executeSubagentInBackground(subchatID string, ...) {
    // Runs in background goroutine - no feedback to parent!
    go func() {
        // ... subagent execution ...
        // No way to notify parent chat
    }()
}
```

**Why it Fails:**
- User gets "Subagent launched" but never sees progress
- Main chat thinks job is done and stops
- Subagent errors are invisible to user
- No way to track task completion

**Solution:**

1. **Add WebSocket event streaming from subagent to parent:**

```go
// New: SubagentProgressEvent
type SubagentProgressEvent struct {
    SubchatID    string
    ParentChatID string
    EventType    string  // "started", "progress", "completed", "failed"
    Message      string
    ToolCall     *ToolCall
    ToolResult   *ToolResult
}

// In executeSubagentInBackground:
func (t *ExecuteSubagentTool) executeSubagentInBackground(...) {
    // Notify parent chat of progress
    progressChan := t.getParentChatChannel(parentChatID)

    progressChan <- SubagentProgressEvent{
        EventType: "started",
        Message: fmt.Sprintf("✅ Subagent %s started working on task", agentTask.AgentName),
    }

    // Stream tool calls to parent
    for event := range aiStream {
        if event.Type == StreamEventToolCall {
            progressChan <- SubagentProgressEvent{
                EventType: "progress",
                ToolCall: event.ToolCall,
            }
        }
    }

    progressChan <- SubagentProgressEvent{
        EventType: "completed",
        Message: fmt.Sprintf("🎉 Subagent completed task successfully"),
    }
}
```

2. **Store subagent messages in parent chat history:**

```go
// When subagent makes progress, save to parent session
func (h *ChatWebSocketHandler) handleSubagentProgress(event SubagentProgressEvent) {
    h.chatService.SaveMessage(ctx, parentSessionID, "system",
        fmt.Sprintf("[%s]: %s", event.SubchatID, event.Message), companyID)
}
```

3. **Add subagent status checking tool:**

```go
// New coordinator tool
func (t *CheckSubagentStatusTool) Execute(ctx context.Context, input map[string]interface{}) {
    subchatID := input["subchatID"].(string)

    // Get current status
    subchat := t.storage.GetSubchat(subchatID)
    agentTask := t.storage.GetAgentTask(subchat.AssignedTaskID)

    completedTodos := 0
    for _, todo := range agentTask.Todos {
        if todo.Status == "completed" {
            completedTodos++
        }
    }

    return map[string]interface{}{
        "status": subchat.Status,
        "progress": fmt.Sprintf("%d/%d TODOs completed", completedTodos, len(agentTask.Todos)),
        "lastUpdate": subchat.UpdatedAt,
    }
}
```

---

### **Gap 5: Context Window Management Too Aggressive**

**Problem:**
- Sliding window kicks in at 40KB (≈10K tokens)
- Aggressive truncation loses important task context
- Models lose track of file paths, task IDs, previous decisions
- Keeps only 6 messages total (too few)

**Evidence:**
```go
// langchain_service.go:402-407
const maxContextSize = 40000 // 40KB threshold
if contextSize > maxContextSize {
    currentMessages = applySlidingWindow(currentMessages, 6) // max 6 messages total
}
```

**Why it Fails:**
- Modern models support 200K tokens (≈800KB text)
- 40KB limit wastes 95% of context capacity
- Important information (task IDs, file paths) gets lost
- Models have to re-discover information

**Solution:**

1. **Increase context threshold dramatically:**

```go
// For Claude Sonnet (200K context):
const maxContextSize = 150000 // 150KB ≈ 37K tokens, leaves room for output

// For GPT-4 (128K context):
const maxContextSize = 100000 // 100KB ≈ 25K tokens
```

2. **Implement smart context retention:**

```go
func applySlidingWindowSmart(messages []Message, maxSize int) []Message {
    // ALWAYS keep:
    // 1. System prompt (first message)
    // 2. Task IDs and file paths (extract and preserve)
    // 3. Most recent 10 messages
    // 4. Tool results from last 5 tool calls

    systemPrompt := messages[0]
    recentMessages := messages[max(0, len(messages)-10):]

    // Extract key information from middle messages
    keyInfo := extractKeyInformation(messages)

    preserved := []Message{systemPrompt}
    preserved = append(preserved, Message{
        Role: "system",
        Content: fmt.Sprintf("CONTEXT SUMMARY:\n%s", keyInfo),
    })
    preserved = append(preserved, recentMessages...)

    return preserved
}

func extractKeyInformation(messages []Message) string {
    var info strings.Builder

    // Extract task IDs
    taskIDs := extractPatterns(messages, `humanTaskId: ([a-f0-9-]+)`)
    if len(taskIDs) > 0 {
        info.WriteString(fmt.Sprintf("Human Task ID: %s\n", taskIDs[0]))
    }

    // Extract file paths
    filePaths := extractPatterns(messages, `filePath": "([^"]+)"`)
    if len(filePaths) > 0 {
        info.WriteString("Files Identified:\n")
        for _, path := range filePaths {
            info.WriteString(fmt.Sprintf("  - %s\n", path))
        }
    }

    return info.String()
}
```

3. **Add context usage monitoring:**

```go
// Log context usage to identify waste
log.Printf("[Context] Using %d/%d tokens (%.1f%%), %d messages",
    usedTokens, maxTokens, float64(usedTokens)/float64(maxTokens)*100, len(messages))
```

---

### **Gap 6: No Clear Task Existence Check**

**Problem:**
- System prompt says "check existing tasks" but doesn't explain HOW to determine similarity
- No semantic search for existing tasks
- Models don't know what "similar" means
- Often skip this step entirely

**Why it Fails:**
- coordinator_list_human_tasks returns raw task list
- No similarity scoring
- Models can't determine if tasks are duplicates

**Solution:**

1. **Add semantic task similarity check:**

```go
// New tool: coordinator_find_similar_tasks
func (t *FindSimilarTasksTool) Execute(ctx context.Context, input map[string]interface{}) {
    userRequest := input["userRequest"].(string)

    // Use embedding to find similar tasks
    embedding := t.embeddingClient.Embed(userRequest)
    similarTasks := t.vectorDB.SearchTasks(embedding, limit: 5, threshold: 0.7)

    if len(similarTasks) == 0 {
        return map[string]interface{}{
            "found": false,
            "message": "No similar tasks found. Proceed with creating new task.",
        }
    }

    return map[string]interface{}{
        "found": true,
        "similarTasks": similarTasks,
        "message": "Found similar tasks. Review them before creating new one.",
    }
}
```

2. **Simplify system prompt guidance:**

```markdown
## Check for Existing Tasks (Optional)
Before creating a new task, optionally check if similar work is already in progress:

coordinator_find_similar_tasks({ userRequest: "user's exact request" })

- If found=false: Proceed with new task
- If found=true: Review similarTasks, ask user if they want to continue or use existing
```

---

### **Gap 7: Subagent Can't Find Files Either**

**Problem:**
- Subagent enters "WRITE-ONLY MODE" with most discovery tools blocked
- If files don't exist or paths are wrong, subagent has no way to recover
- Blocking code_index_search and list_directory prevents error recovery

**Evidence:**
```go
// coordinator_tools.go:2313-2317
BLOCKED TOOLS (these will FAIL):
❌ code_index_search - Discovery disabled in WRITE-ONLY MODE
❌ list_directory    - Discovery disabled in WRITE-ONLY MODE
```

**Why it Fails:**
- Coordinator gives wrong file path → subagent blocked from finding correct path
- File moved/renamed → subagent can't adapt
- New files need to be created → subagent can't explore where to put them

**Solution:**

1. **Allow limited discovery in write mode:**

```go
// REVISED system prompt for subagents
ALLOWED TOOLS:
✅ read_file       - Read source files
✅ write_file      - Write/create files
✅ apply_patch     - Apply code changes
✅ bash            - Run commands/tests
✅ list_directory  - ⚠️ LIMITED: Check if file exists, find adjacent files (max 3 calls)
✅ code_index_search - ⚠️ EMERGENCY ONLY: If file not found after 2 tries (max 1 call)

WORKFLOW:
1. Try to read files from task (filesModified array)
2. If read fails (file not found), call list_directory on parent directory
3. If still can't find, call code_index_search once
4. If still can't find, ask coordinator for clarification
```

2. **Add file existence validation before subagent launch:**

```go
// In execute_subagent tool, before launching:
func (t *ExecuteSubagentTool) Execute(ctx context.Context, input map[string]interface{}) {
    agentTask := t.taskStorage.GetAgentTask(agentTaskID)

    // Validate files exist
    var missingFiles []string
    for _, filePath := range agentTask.FilesModified {
        if !fileExists(filePath) {
            missingFiles = append(missingFiles, filePath)
        }
    }

    if len(missingFiles) > 0 {
        return nil, fmt.Errorf(
            "Cannot launch subagent: %d files don't exist:\n%s\n\n"+
            "Please verify file paths and update agent task before launching.",
            len(missingFiles), strings.Join(missingFiles, "\n"))
    }

    // Proceed with launch
    return t.launchSubagent(...)
}
```

---

## 🎯 **RECOMMENDED IMPLEMENTATION PRIORITY**

### **Phase 1: Critical Fixes (Do First)**
1. **Remove FILE_PATHS_TO_USE fiction** (Gap 2) - 1 hour
2. **Increase circuit breaker thresholds** (Gap 3) - 30 minutes
3. **Fix context window** (Gap 5) - 1 hour
4. **Validate files before subagent launch** (Gap 7) - 1 hour

### **Phase 2: Workflow Improvements**
1. **Simplify system prompt** (Gap 1) - 2 hours
2. **Add subagent progress feedback** (Gap 4) - 4 hours
3. **Add task similarity check** (Gap 6) - 2 hours

### **Phase 3: Enhanced Features**
1. Smart context retention
2. Subagent error recovery
3. Progress tracking UI

---

## 📋 **SUCCESS METRICS**

After implementing these fixes, measure:

1. **Task Success Rate**: % of tasks completed without coordinator intervention
2. **Circuit Breaker Hits**: Should drop from ~40% to <5%
3. **File Path Errors**: Should drop from ~30% to <2%
4. **Average Tool Calls to Completion**: Should drop from 15+ to 8-10
5. **User Intervention Rate**: Should drop from 60% to <20%

---

## 🔬 **TESTING PLAN**

### Test Case 1: Simple Feature Addition
```
User: "Add a dark mode toggle to the settings page"

Expected Flow:
1. coordinator_list_human_tasks → no similar tasks
2. coordinator_create_human_task → returns task ID
3. code_index_search("settings page dark mode") → returns 2-3 files
4. create_agent_task with exact file paths from search
5. execute_subagent → launches ui-dev
6. [Subagent reads files, writes changes, tests]
7. User sees progress updates in main chat
8. Task marked complete

Success: Completes without errors, no circuit breaker hits, files modified correctly
```

### Test Case 2: File Not Found Recovery
```
User: "Update the API endpoint in auth service"

Expected Flow:
1-4. [Normal task creation]
5. execute_subagent → launches go-dev
6. go-dev tries to read /path/to/auth.go → FILE NOT FOUND
7. go-dev calls list_directory("/path/to/") → finds auth_service.go
8. go-dev reads correct file, makes changes
9. Task completed

Success: Subagent recovers from wrong file path without coordinator help
```

### Test Case 3: Complex Multi-File Change
```
User: "Implement user profile caching with Redis"

Expected Flow:
1-3. [Normal task creation]
4. code_index_search returns 5 files (user model, cache layer, API routes)
5. create_agent_task with all 5 files
6. execute_subagent → launches go-dev
7. Subagent reads all 5 files (no circuit breaker)
8. Makes coordinated changes across files
9. Runs tests
10. Task completed

Success: Reads multiple files without triggering circuit breaker, completes successfully
```

---

## 💡 **KEY INSIGHTS FOR ANTHROPIC MODELS**

### What Works Well with Claude:
1. **Outcome-focused prompts** ("Achieve X") vs. step-by-step instructions
2. **Clear tool response formats** with explicit field names
3. **Flexible workflows** that adapt to context
4. **Concrete examples** over abstract concepts
5. **Error messages that suggest alternatives** ("Try X instead of Y")

### What Doesn't Work:
1. ❌ Micromanaging with strict step orders
2. ❌ Abstract concepts (FILE_PATHS_TO_USE array)
3. ❌ Emoji-heavy warnings (⛔🚨❌) without actionable guidance
4. ❌ Blocking tools without providing alternatives
5. ❌ Assuming models can "copy-paste" or "see" data structures

---

## 🚀 **QUICK WINS**

If you can only do 3 things, do these:

1. **Remove FILE_PATHS_TO_USE from all prompts** and add actual response structure examples
2. **Increase circuit breaker thresholds** to 4+ for most tools
3. **Add file existence check** before launching subagents

These alone will fix ~60% of the reported issues.

---

## 📞 **NEXT STEPS**

1. Review this analysis with the team
2. Prioritize fixes based on impact vs. effort
3. Implement Phase 1 (critical fixes) first
4. Test with real user scenarios
5. Measure success metrics
6. Iterate on Phase 2 based on results

The core insight: **Less prescription, more intelligence**. Let Claude figure out the best path given clear objectives and accurate information about tool capabilities.
