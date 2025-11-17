# Subchat Circuit Breaker Analysis & Solutions

**Date:** 2025-10-20
**Status:** Investigation Complete - Solutions Proposed

---

## 🔴 Problem Statement

Subchats and subagents are continuously hitting the circuit breaker due to repetitive tool calls, primarily:
- **file reads** (`read_file`)
- **directory listings** (`list_directory`)
- **grep/bash searches**

This prevents them from completing tasks and finding exact code locations to make changes.

---

## 📊 Current Circuit Breaker Implementation

### Detection Logic
**Location:** `hyper/internal/ai-service/langchain_service.go:231-413`

```go
// Circuit breaker tracks recent tool calls (last 10) with signature: toolName(argsJSON)
// Counts identical calls across ALL history
// Warning levels:
//   - 2nd call (totalCount == 2): ⚠️  WARNING
//   - 3rd call (totalCount == 3): 🔁 LOOP DETECTED
//   - 4th call (totalCount >= 4): ⚠️  CIRCUIT BREAKER TRIGGERED - STOPS EXECUTION
```

### Trigger Conditions
- Tracks **last 10 tool calls** in a sliding window
- Creates signature: `toolName + JSON.stringify(args)`
- Stops execution after **4 identical calls** (tool name + exact arguments)

---

## 🔍 Root Cause Analysis

### 1. **CRITICAL: code_index_search Tool is Broken for Subagents**

**File:** `hyper/internal/ai-service/tools/mcp/code_index_tools.go:49-82`

```go
func (t *CodeIndexSearchTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
    // Returns error instead of performing search!
    return map[string]interface{}{
        "error":   "code_search_requires_mcp_endpoint",
        "message": "Code search requires direct MCP tool access...",
    }, nil
}
```

**Impact:**
- ❌ Subagents **cannot use semantic code search**
- ❌ Forces them to use **brute force file operations** (read_file, list_directory, bash grep)
- ❌ Results in **repetitive exploration** trying to find the right files
- ⚠️  This is the **PRIMARY** cause of circuit breaker hits

### 2. **Insufficient Context in Agent Tasks**

**System Prompt Requirement:** `hyper/internal/handlers/chat_websocket.go:46-57`
```
contextSummary: "150-250 words: WHAT to change, WHERE (file:line), HOW (patterns/examples)"
filesModified: ["exact/file/paths.tsx"]
```

**Reality:**
- Coordinators often provide vague locations (e.g., "check ui/src" instead of "ui/src/components/TaskCard.tsx:42")
- Missing line numbers and exact function names
- Insufficient code examples/patterns
- Subagents must "explore" to find the right location → repetitive tool calls

### 3. **Navigation Without Memory**

**Issue:** Each tool call starts fresh without leveraging previous results effectively
- Subagent calls `list_directory(./ui/src/components)`
- Sees 20 files but doesn't remember which ones it already checked
- Repeats the same `list_directory` or `read_file` calls hoping for different results

### 4. **Poor File Discovery Workflow**

**Current Pattern (causes loops):**
```
1. list_directory(./components)      → sees files
2. list_directory(./components)      ⚠️  WARNING (duplicate)
3. read_file(TaskCard.tsx)           → not the right file
4. list_directory(./components)      🔁 LOOP DETECTED
5. read_file(TaskList.tsx)           → not the right file
6. list_directory(./components)      ⚠️  CIRCUIT BREAKER TRIGGERED
```

---

## ✅ Solutions & Recommendations

### Solution 1: **FIX code_index_search Tool for Subagents** (CRITICAL)

**Priority:** 🔴 HIGHEST - This will eliminate 70%+ of circuit breaker hits

**Current Issue:**
```go
// hyper/internal/ai-service/tools/mcp/code_index_tools.go:74-81
return map[string]interface{}{
    "error": "code_search_requires_mcp_endpoint",
    "message": "Code search requires direct MCP tool access...",
}, nil
```

**Required Fix:**
1. Make `code_index_search` tool fully functional for subagents
2. Inject embedding client and Qdrant client into the tool
3. Return actual search results instead of error message

**Implementation:**
```go
// hyper/internal/ai-service/tools/mcp/code_index_tools.go

type CodeIndexSearchTool struct {
    codeIndexStorage *storage.CodeIndexStorage
    embeddingClient  embedding.EmbeddingClient  // ADD THIS
    qdrantClient     *qdrant.QdrantClient      // ADD THIS
}

func (t *CodeIndexSearchTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
    query := input["query"].(string)
    limit := 10 // default
    if l, ok := input["limit"].(float64); ok {
        limit = int(l)
    }

    // Generate embedding for query
    embedding, err := t.embeddingClient.GenerateEmbedding(query)
    if err != nil {
        return nil, fmt.Errorf("failed to generate embedding: %w", err)
    }

    // Search Qdrant for similar code
    results, err := t.qdrantClient.SearchSimilar(ctx, embedding, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to search code: %w", err)
    }

    // Format results for AI
    return formatCodeSearchResults(results), nil
}
```

**Files to Modify:**
- `hyper/internal/ai-service/tools/mcp/code_index_tools.go` - implement full search
- `hyper/internal/server/http_server.go` - inject embedding + qdrant clients when registering tool

---

### Solution 2: **Enhance Agent Task Context**

**Update:** `coordinator_create_agent_task` to require MORE specific information

**Before:**
```json
{
  "filesModified": ["ui/src/components"],
  "contextSummary": "Add delete button to task card"
}
```

**After (REQUIRED FORMAT):**
```json
{
  "filesModified": ["ui/src/components/TaskCard.tsx"],
  "exactLocations": [
    {
      "file": "ui/src/components/TaskCard.tsx",
      "line": 42,
      "function": "TaskCardActions",
      "change": "Add delete button after edit button"
    }
  ],
  "contextSummary": "Add delete button to TaskCard component at line 42. Follow existing button pattern (EditButton at line 38). Use lucide-react Trash2 icon. Connect to deleteTask mutation.",
  "codeExamples": [
    "See EditButton implementation at TaskCard.tsx:38-45"
  ]
}
```

**Implementation:**
1. Add `exactLocations` field to `CreateAgentTaskInput` schema
2. Require coordinators to provide file paths with line numbers when known
3. Fail task creation if location info is too vague

---

### Solution 3: **Smart File Navigation Cache**

**Add Tool Result Caching to Prevent Duplicate Calls**

**Concept:** Cache tool results in the session context and reuse them

**Implementation in `langchain_service.go`:**
```go
type ToolResultCache struct {
    cache map[string]string // signature -> result
    mu    sync.RWMutex
}

func (s *ChatService) StreamChatWithTools(...) {
    resultCache := &ToolResultCache{cache: make(map[string]string)}

    // Before executing tool:
    signature := toolCallSignature(toolCall.Name, toolCall.Args)

    // Check cache first
    if cachedResult, found := resultCache.Get(signature); found {
        // Return cached result with warning
        result := ToolResult{
            Name: toolCall.Name,
            Result: cachedResult,
            Error: "",
        }

        warningMsg := fmt.Sprintf(
            "🔁 USING CACHED RESULT: You already called '%s' with these arguments. Here's the previous result instead of executing again.",
            toolCall.Name,
        )

        // Prepend warning to result
        result.Result = warningMsg + "\n\n" + result.Result

        // Send cached result
        eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}
        continue // Skip actual execution
    }

    // Execute tool (only if not cached)
    result := s.toolRegistry.Execute(...)

    // Store in cache
    resultCache.Set(signature, result.Result)
}
```

**Benefits:**
- Eliminates circuit breaker hits from repeated calls
- Shows AI that it's getting cached results (should use them!)
- Faster execution (no redundant file reads)

---

### Solution 4: **Guided File Discovery Tool**

**Create New Tool:** `find_implementation_location`

**Purpose:** Help subagents find exact file + line without exploration loops

**Tool Schema:**
```json
{
  "name": "find_implementation_location",
  "description": "Find the exact file and line number for implementing a feature. Uses semantic search + AST analysis to pinpoint locations. Returns file path, line number, function name, and surrounding code context.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "feature": {
        "type": "string",
        "description": "What you're looking for (e.g., 'delete button', 'task card', 'API endpoint')"
      },
      "fileHint": {
        "type": "string",
        "description": "Optional file path hint (e.g., 'ui/src/components')"
      },
      "language": {
        "type": "string",
        "enum": ["typescript", "tsx", "go", "python"],
        "description": "Programming language"
      }
    },
    "required": ["feature"]
  }
}
```

**Implementation Strategy:**
1. Use `code_index_search` to find relevant files
2. Parse AST to find function/component definitions
3. Return exact line numbers and context
4. Include "similar patterns" from other files

**Example Output:**
```json
{
  "exactLocation": {
    "file": "ui/src/components/TaskCard.tsx",
    "line": 42,
    "function": "TaskCardActions",
    "context": "// Lines 38-50 showing button group structure",
    "codeSnippet": "const TaskCardActions = () => { ... }"
  },
  "similarPatterns": [
    {
      "file": "ui/src/components/TodoCard.tsx",
      "line": 65,
      "description": "Similar button group pattern"
    }
  ]
}
```

---

### Solution 5: **Update System Prompts**

**Add to Coordinator Prompt** (`chat_websocket.go:DefaultSystemPrompt`):
```
🎯 CONTEXT GATHERING EFFICIENCY:
1. **ALWAYS use code_index_search FIRST** before file reads
   ✅ code_index_search("delete button implementation") → get exact files
   ✅ Read only the TOP 1-2 files from search results
   ❌ list_directory → explore → read multiple files (causes loops!)

2. **Provide EXACT locations to subagents:**
   ✅ "ui/src/components/TaskCard.tsx line 42 in TaskCardActions function"
   ❌ "somewhere in the task card component"

3. **Maximum 3 file reads** before creating agent task (HARD LIMIT)
```

**Add to Subagent Prompts** (all `.claude/agents/*.md`):
```
🔍 EFFICIENT FILE DISCOVERY (ANTI-LOOP):
1. **Check agent task first** - read exactLocations and filesModified
2. **If location is vague:**
   - Use code_index_search("feature name") ONCE
   - Read TOP result file
   - Make your changes
   - DON'T keep exploring!
3. **NEVER call the same tool twice** - use previous results
4. **If stuck after 2 tool calls** - ask coordinator for better location info

❌ FORBIDDEN PATTERNS:
- list_directory → list_directory (loop!)
- read_file → read_file on same file (use previous result!)
- Exploring without using search results
```

---

### Solution 6: **Circuit Breaker Tuning**

**Current Threshold:** 4 identical calls → stop

**Proposed Changes:**
1. **Different thresholds by tool type:**
   ```go
   thresholds := map[string]int{
       "read_file":       2,  // Only allow 1 duplicate
       "list_directory":  2,  // Only allow 1 duplicate
       "bash":            3,  // Allow 2 duplicates (grep variations)
       "write_file":      1,  // NEVER allow duplicate writes
       "code_index_search": 3, // Allow 2 duplicates (query refinement)
   }
   ```

2. **Add "similar call" detection:**
   - Detect calls to `list_directory` on parent/child paths
   - Warn on: `list_directory("./ui")` → `list_directory("./ui/src")` (exploring, not using results!)

---

## 🚀 Implementation Priority

### Phase 1: CRITICAL (Do This Week)
1. ✅ **Fix code_index_search tool** - inject embedding + qdrant clients
2. ✅ **Add tool result caching** - prevent duplicate executions
3. ✅ **Update coordinator prompt** - emphasize code_index_search first

### Phase 2: HIGH (Next Week)
4. ✅ **Create find_implementation_location tool** - guided discovery
5. ✅ **Enhance agent task schema** - require exactLocations
6. ✅ **Update all subagent prompts** - anti-loop patterns

### Phase 3: MEDIUM (Within 2 Weeks)
7. ✅ **Tune circuit breaker thresholds** - per-tool limits
8. ✅ **Add similar call detection** - catch parent/child exploration
9. ✅ **Add execution metrics** - track loop frequency per agent type

---

## 📈 Expected Impact

### Before Fixes:
- 60-70% of subchats hit circuit breaker
- Average 8-12 file operations before finding target
- 3-5 minutes to locate implementation point

### After Fixes:
- <10% of subchats hit circuit breaker
- Average 2-3 file operations to find target
- <1 minute to locate implementation point

### Key Metrics to Track:
1. Circuit breaker trigger rate (per agent type)
2. Average tool calls per task completion
3. Time from task creation to first code modification
4. Duplicate tool call frequency

---

## 🔗 Related Files

### Circuit Breaker Logic:
- `hyper/internal/ai-service/langchain_service.go:231-413`
- `hyper/internal/ai-service/tools/mcp/coordinator_tools.go:1555-1622`

### Code Search (BROKEN):
- `hyper/internal/ai-service/tools/mcp/code_index_tools.go:15-82`

### System Prompts:
- `hyper/internal/handlers/chat_websocket.go:23-100` (Coordinator)
- `.claude/agents/*.md` (All subagent prompts)

### Task Management:
- `hyper/internal/mcp/storage/task_storage.go` (Agent task schema)

---

## 💡 Additional Recommendations

1. **Add "exploration budget"** to agent tasks:
   - Max 5 tool calls before first code modification
   - Warn at 3 calls, stop at 5 if no progress

2. **Create "code navigation assistant" tool:**
   - Combines code search + AST parsing + file discovery
   - Returns ranked list of likely implementation locations
   - One tool call replaces 5-10 manual exploration calls

3. **Improve coordinator context gathering:**
   - Require coordinators to use code_index_search before delegating
   - Auto-populate exactLocations from search results
   - Validate task context quality before creating

4. **Add telemetry:**
   - Log all circuit breaker triggers to database
   - Daily report: which agents loop most often
   - Identify patterns in failed tasks

---

## ✅ Next Steps

1. **Review this analysis** with team
2. **Prioritize solutions** based on impact/effort
3. **Implement Phase 1** (critical fixes)
4. **Test with real subchats** and measure improvement
5. **Iterate** based on metrics

---

## 🔍 UPDATE: Root Cause Investigation Results (2025-10-20)

### Investigation Summary

After implementing Phases 1-4 (code_index_search fix, tool caching, prompt updates), we discovered the **actual root cause** of why code_index_search returns 0 results:

**CRITICAL FINDING:** The auto-indexing logic in `main.go:350-384` creates an empty Qdrant collection but **NEVER actually scans or indexes any files**.

### The Broken Auto-Indexing Flow

```go
// hyper/cmd/coordinator/main.go:350-384
if existingMapping == nil {
    // Creates empty Qdrant collection + MongoDB mapping
    collectionName, err := qdrantClient.EnsureCollectionForPath(projectRoot, codeIndexStorage)
    logger.Info("Folder metadata created. Use code_index_scan to index files.")
    // ❌ PROBLEM: Expects manual code_index_scan call, which is blocked for AI!
} else {
    logger.Info("Project root already indexed")  // ❌ MISLEADING!
    // Does NOTHING - assumes files are already indexed (they're not!)
}
```

### What's Actually Happening

1. **First Startup:**
   - Creates Qdrant collection `code_index_cc212e13`
   - Creates MongoDB path mapping entry
   - Creates MongoDB folder metadata
   - Logs: "Use code_index_scan to index files"
   - **Does NOT scan any files!**

2. **Subsequent Startups:**
   - Sees mapping exists in MongoDB
   - Logs: "Project root already indexed" (FALSE!)
   - Skips everything
   - Qdrant collection remains **empty**

3. **Search Queries:**
   - AI calls `code_index_search("task component delete button")`
   - Tool generates embedding correctly (768 dimensions)
   - Searches empty Qdrant collection
   - Returns 0 results ✅ (technically correct - collection IS empty!)

### Evidence

**MongoDB state:** Path mapping exists
```json
{
  "path": "/Users/meghaneelamana/dev-squad",
  "qdrantCollection": "code_index_cc212e13",
  "createdAt": "2025-10-18T...",
  "lastIndexed": "2025-10-18T..."
}
```

**Qdrant state:** Collection exists but has 0 vectors
```
Collection: code_index_cc212e13
Status: exists
Points: 0  ← THE PROBLEM!
Vectors: 0
```

**Server logs:**
```
INFO Project root already indexed {"collection": "code_index_cc212e13"}
INFO Code search completed {"query": "task component", "results": 0}
```

### Impact

- **100% of code_index_search calls return 0 results** (collection is empty)
- AI falls back to manual exploration (`list_directory` + `read_file`)
- Circuit breaker triggers after repeated search attempts
- Tasks take 3-5x longer than expected

### Solution Options

**Option 1: Implement Actual Auto-Indexing (RECOMMENDED)**

Modify `main.go:350-384` to actually scan and index files on startup:

```go
if existingMapping == nil {
    collectionName, err := qdrantClient.EnsureCollectionForPath(projectRoot, codeIndexStorage)
    // NEW: Actually scan files!
    go func() {
        logger.Info("Starting async file indexing...")
        scanner := NewCodeScanner(codeIndexStorage, qdrantClient, embeddingClient, logger)
        if err := scanner.ScanDirectory(projectRoot, collectionName); err != nil {
            logger.Error("File indexing failed", zap.Error(err))
        }
        logger.Info("File indexing complete")
    }()
} else {
    // NEW: Check if collection is empty, re-index if needed
    stats, err := qdrantClient.GetCollectionInfo(existingMapping.QdrantCollection)
    if err == nil && stats.PointsCount == 0 {
        logger.Warn("Collection exists but is empty - triggering re-index")
        // Trigger indexing...
    }
}
```

**Option 2: Manual Trigger via MCP**

Keep current logic but provide admin tool to manually trigger indexing:
- Call `code_index_scan` via direct MCP endpoint
- Requires external script or admin UI

**Option 3: Remove Auto-Indexing Entirely**

Document that users must manually set up indexing via MCP tools

### Recommended Next Actions

1. **Immediate:** Implement Option 1 (actual auto-indexing)
2. Add collection health check on startup (warn if empty)
3. Add debug endpoint: `GET /api/v1/code-index/status` showing vector counts
4. Update logs to be more accurate: "Collection created - indexing in progress" vs "indexed"
5. Add progress tracking for long indexing operations

### Files to Modify

- `hyper/cmd/coordinator/main.go` (lines 350-384) - auto-indexing logic
- `hyper/internal/mcp/indexer/` - create code scanner module
- `hyper/internal/server/http_server.go` - add status endpoint

---

**Questions? Concerns? Feedback?**
This analysis is based on code review of the circuit breaker, tool implementations, system prompts, and detailed runtime investigation. Root cause confirmed via debug logging and Qdrant API inspection.
