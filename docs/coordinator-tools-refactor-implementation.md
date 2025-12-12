# Coordinator Tools Refactor - Implementation Plan

## Problem Statement

The `coordinator_tools.go` file has grown to **5,003 lines**, making it:
- **Hard to navigate**: Multiple tool categories mixed together
- **Difficult to test**: Large file = complex test setup
- **Maintenance burden**: Changes risk unintended side effects
- **Code review challenges**: Hard to review changes in a monolithic file

## Current File Structure Analysis

```
coordinator_tools.go (5,003 lines)
├── Lines 1-26: Imports
├── Lines 27-327: Path correction utilities (correctFilePaths, tryFixPath, extractPatternFiles)
├── Lines 329-341: CoordinatorTools struct & constructor
├── Lines 343-499: AnalyzeComplexityTool
├── Lines 501-1097: CreateAgentTaskTool (large - complex validation)
├── Lines 1099-1192: ListAgentTasksTool
├── Lines 1194-1287: QueryKnowledgeTool
├── Lines 1289-1371: UpsertKnowledgeTool
├── Lines 1373-1418: ListCollectionsTool
├── Lines 1420-1560: CreateHumanTaskTool
├── Lines 1562-1621: UpdateTaskStatusTool
├── Lines 1623-1725: UpdateTodoStatusTool
├── Lines 1727-1825: ListHumanTasksTool
├── Lines 1827-1867: GetAgentTaskTool
├── Lines 1869-1958: FindSimilarTasksTool
├── Lines 1960-2116: Prompt notes tools (Add/Update/Clear Task & Todo)
├── Lines 2118-2303: More prompt notes tools
├── Lines 2305-2425: Subagent list/set tools
├── Lines 2427-2627: MCP discovery/execute tools (Discover, GetSchema, Execute, Add/Remove Server)
├── Lines 2629-2823: ExecuteSubagentTool (struct + Execute method)
├── Lines 2825-3211: FileOperationTracker, helper functions
├── Lines 3213-4935: executeSubagentInBackground (massive - 1700+ lines with interrupt handling)
├── Lines 4937-5003: RegisterCoordinatorTools function
```

## Proposed Split

| New File | Approx Lines | Responsibility | Dependencies |
|----------|--------------|----------------|--------------|
| `task_tools.go` | ~1,200 | Human/Agent task CRUD operations | storage, models |
| `knowledge_tools.go` | ~250 | Knowledge base operations | storage |
| `prompt_notes_tools.go` | ~350 | Task/TODO prompt notes | storage |
| `subagent_tools.go` | ~2,400 | Subagent execution & tracking | storage, handlers, aiservice |
| `mcp_tools.go` | ~250 | MCP server discovery & execution | mcphandlers |
| `path_utils.go` | ~350 | Path correction utilities | tools |
| `coordinator_tools.go` | ~200 | Main struct, registration | All above |

## Phase-by-Phase Implementation

---

## Phase 1: Extract `path_utils.go`

### What to Extract
- `correctFilePaths` function (lines 29-118)
- `extractPatternFiles` function (lines 120-232)
- `tryFixPath` function (lines 235-327)
- `normalizePathForComparison` function (lines 2965-2988)

### File Structure
```go
// path_utils.go
package mcp

import (
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "hyper/internal/ai-service/tools"
    "go.uber.org/zap"
)

// correctFilePaths attempts to fix invalid file paths using common correction strategies
// Returns (correctedPaths, unfixablePaths, wasIndexingIssue)
func correctFilePaths(paths []string, logger *zap.Logger) ([]string, []string, bool) {...}

// extractPatternFiles scans contextSummary and todos for file references
func extractPatternFiles(contextSummary string, todos []string, projectRoot string, logger *zap.Logger) []string {...}

// tryFixPath attempts multiple strategies to correct an invalid file path
func tryFixPath(path string, projectRoot string, logger *zap.Logger) string {...}

// normalizePathForComparison cleans and normalizes a path for comparison
func normalizePathForComparison(path string, projectRoot string) []string {...}
```

### Verification
```bash
go build ./internal/ai-service/tools/mcp/...
```

---

## Phase 2: Extract `task_tools.go`

### What to Extract
- `CreateAgentTaskTool` struct and methods (lines 501-1097)
- `ListAgentTasksTool` struct and methods (lines 1099-1192)
- `CreateHumanTaskTool` struct and methods (lines 1420-1560)
- `UpdateTaskStatusTool` struct and methods (lines 1562-1621)
- `UpdateTodoStatusTool` struct and methods (lines 1623-1725)
- `ListHumanTasksTool` struct and methods (lines 1727-1825)
- `GetAgentTaskTool` struct and methods (lines 1827-1867)
- `FindSimilarTasksTool` struct and methods (lines 1869-1958)
- `AnalyzeComplexityTool` struct and methods (lines 343-499)

### File Structure
```go
// task_tools.go
package mcp

import (...)

// AnalyzeComplexityTool implements complexity analysis for tasks
type AnalyzeComplexityTool struct {...}

// CreateAgentTaskTool implements the ToolExecutor interface
type CreateAgentTaskTool struct {...}

// ListAgentTasksTool implements the ToolExecutor interface
type ListAgentTasksTool struct {...}

// CreateHumanTaskTool implements the ToolExecutor interface
type CreateHumanTaskTool struct {...}

// UpdateTaskStatusTool implements the ToolExecutor interface
type UpdateTaskStatusTool struct {...}

// UpdateTodoStatusTool implements the ToolExecutor interface
type UpdateTodoStatusTool struct {...}

// ListHumanTasksTool implements the ToolExecutor interface
type ListHumanTasksTool struct {...}

// GetAgentTaskTool implements the ToolExecutor interface
type GetAgentTaskTool struct {...}

// FindSimilarTasksTool implements the ToolExecutor interface
type FindSimilarTasksTool struct {...}
```

### Verification
```bash
go build ./internal/ai-service/tools/mcp/...
```

---

## Phase 3: Extract `knowledge_tools.go`

### What to Extract
- `QueryKnowledgeTool` struct and methods (lines 1194-1287)
- `UpsertKnowledgeTool` struct and methods (lines 1289-1371)
- `ListCollectionsTool` struct and methods (lines 1373-1418)

### File Structure
```go
// knowledge_tools.go
package mcp

import (...)

// QueryKnowledgeTool implements the ToolExecutor interface
type QueryKnowledgeTool struct {...}

// UpsertKnowledgeTool implements the ToolExecutor interface
type UpsertKnowledgeTool struct {...}

// ListCollectionsTool implements the ToolExecutor interface
type ListCollectionsTool struct {...}
```

### Verification
```bash
go build ./internal/ai-service/tools/mcp/...
```

---

## Phase 4: Extract `prompt_notes_tools.go`

### What to Extract
- `AddTaskPromptNotesTool` struct and methods (lines 1960-2016)
- `UpdateTaskPromptNotesTool` struct and methods (lines 2018-2073)
- `ClearTaskPromptNotesTool` struct and methods (lines 2075-2116)
- `AddTodoPromptNotesTool` struct and methods (lines 2118-2183)
- `UpdateTodoPromptNotesTool` struct and methods (lines 2185-2250)
- `ClearTodoPromptNotesTool` struct and methods (lines 2252-2303)

### File Structure
```go
// prompt_notes_tools.go
package mcp

import (...)

// AddTaskPromptNotesTool implements the ToolExecutor interface
type AddTaskPromptNotesTool struct {...}

// UpdateTaskPromptNotesTool implements the ToolExecutor interface
type UpdateTaskPromptNotesTool struct {...}

// ClearTaskPromptNotesTool implements the ToolExecutor interface
type ClearTaskPromptNotesTool struct {...}

// AddTodoPromptNotesTool implements the ToolExecutor interface
type AddTodoPromptNotesTool struct {...}

// UpdateTodoPromptNotesTool implements the ToolExecutor interface
type UpdateTodoPromptNotesTool struct {...}

// ClearTodoPromptNotesTool implements the ToolExecutor interface
type ClearTodoPromptNotesTool struct {...}
```

### Verification
```bash
go build ./internal/ai-service/tools/mcp/...
```

---

## Phase 5: Extract `mcp_tools.go`

### What to Extract
- `DiscoverToolsExecutor` struct and methods (lines 2427-2460)
- `GetToolSchemaExecutor` struct and methods (lines 2462-2491)
- `ExecuteToolExecutor` struct and methods (lines 2493-2526)
- `McpAddServerExecutor` struct and methods (lines 2528-2565)
- `McpRediscoverServerExecutor` struct and methods (lines 2567-2596)
- `McpRemoveServerExecutor` struct and methods (lines 2598-2627)

### File Structure
```go
// mcp_tools.go
package mcp

import (...)

// DiscoverToolsExecutor implements the discover_tools tool executor
type DiscoverToolsExecutor struct {...}

// GetToolSchemaExecutor implements the get_tool_schema tool executor
type GetToolSchemaExecutor struct {...}

// ExecuteToolExecutor implements the execute_tool tool executor
type ExecuteToolExecutor struct {...}

// McpAddServerExecutor implements the mcp_add_server tool executor
type McpAddServerExecutor struct {...}

// McpRediscoverServerExecutor implements the mcp_rediscover_server tool executor
type McpRediscoverServerExecutor struct {...}

// McpRemoveServerExecutor implements the mcp_remove_server tool executor
type McpRemoveServerExecutor struct {...}
```

### Verification
```bash
go build ./internal/ai-service/tools/mcp/...
```

---

## Phase 6: Extract `subagent_tools.go`

### What to Extract
- `ListSubagentsTool` struct and methods (lines 2305-2361)
- `SetCurrentSubagentTool` struct and methods (lines 2363-2425)
- `ExecuteSubagentTool` struct and methods (lines 2629-2823)
- Interface definitions: `AIServiceInterface`, `ChatServiceInterface`, `AISettingsServiceInterface` (lines 2641-2661)
- `FileOperationTracker` struct and methods (lines 2825-2963)
- `convertToolCallToPlainEnglish` function (lines 3075-3129)
- `convertToolResultToPlainEnglish` function (lines 3131-3211)
- `isSystemEnforcementMessage` function (lines 3213-3233)
- `executeSubagentInBackground` method (lines 3235-4935) - This is the largest function
- `validateFileModifications` method (lines 2991-3073)
- `handleExecutionFailure` method (lines 4839-4847)
- `categorizeInterrupt` method (lines 4855-4932)
- `InterruptCategorization` struct (lines 4849-4853)

### File Structure
```go
// subagent_tools.go
package mcp

import (...)

// Interface definitions
type AIServiceInterface interface {...}
type ChatServiceInterface interface {...}
type AISettingsServiceInterface interface {...}

// ListSubagentsTool implements the ToolExecutor interface
type ListSubagentsTool struct {...}

// SetCurrentSubagentTool implements the ToolExecutor interface
type SetCurrentSubagentTool struct {...}

// ExecuteSubagentTool implements the ToolExecutor interface
type ExecuteSubagentTool struct {...}

// FileOperationTracker tracks file operations during subagent execution
type FileOperationTracker struct {...}

// InterruptCategorization holds the result of interrupt analysis
type InterruptCategorization struct {...}

// Helper functions...
// executeSubagentInBackground runs the subagent AI streaming
func (t *ExecuteSubagentTool) executeSubagentInBackground(...) {...}
```

### Verification
```bash
go build ./internal/ai-service/tools/mcp/...
```

---

## Phase 7: Final Verification

### Build Check
```bash
make build
```

### Test Check
```bash
go test ./internal/ai-service/tools/mcp/... -v
```

### Line Count Verification
After all phases:
- `path_utils.go`: ~350 lines
- `task_tools.go`: ~1,200 lines
- `knowledge_tools.go`: ~250 lines
- `prompt_notes_tools.go`: ~350 lines
- `mcp_tools.go`: ~250 lines
- `subagent_tools.go`: ~2,400 lines
- `coordinator_tools.go`: ~200 lines (main struct + registration)

---

## Implementation Rules

### DO
1. Extract complete, self-contained units
2. Maintain all existing functionality
3. Keep imports minimal in extracted files
4. Preserve all comments and documentation
5. Run `go build` after each phase

### DO NOT
1. Change any logic or behavior
2. Rename functions or types
3. Modify method signatures
4. Add new features during refactoring
5. Remove any existing code (only move)

---

## Rollback Plan

Each phase creates a new file. If issues arise:
1. Delete the new file
2. Uncomment/restore the code in `coordinator_tools.go`
3. Rebuild and verify

---

## Benefits After Refactor

| Benefit | Before | After |
|---------|--------|-------|
| File size | 5,003 lines | ~200 lines main + 6 smaller files |
| Testability | Complex setup | Each file independently testable |
| Navigation | Scroll through 5K lines | Jump to specific file |
| Code review | Large diffs | Focused, smaller diffs |
| Onboarding | Overwhelming | Clear separation of concerns |

---

## File Dependency Graph (After Refactor)

```
coordinator_tools.go (main orchestrator + registration)
    ├── path_utils.go
    │   └── Used by: task_tools.go, subagent_tools.go
    │
    ├── task_tools.go
    │   └── Used by: coordinator_tools.go (registration)
    │
    ├── knowledge_tools.go
    │   └── Used by: coordinator_tools.go (registration)
    │
    ├── prompt_notes_tools.go
    │   └── Used by: coordinator_tools.go (registration)
    │
    ├── mcp_tools.go
    │   └── Used by: coordinator_tools.go (registration)
    │
    └── subagent_tools.go
        └── Used by: coordinator_tools.go (registration)
```

---

## Status Tracking

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: path_utils.go | COMPLETE | 340 lines - Path correction utilities |
| Phase 2: knowledge_tools.go | COMPLETE | 236 lines - Knowledge base tools |
| Phase 3: prompt_notes_tools.go | COMPLETE | 353 lines - Prompt notes tools |
| Phase 4: mcp_tools.go | COMPLETE | 209 lines - MCP discovery tools |
| Phase 5: subagent_tools.go | COMPLETE | 2,423 lines - Subagent execution |
| Phase 6: Final verification | COMPLETE | Build successful - `make build` passed |

## Final Results

| File | Lines | Description |
|------|-------|-------------|
| coordinator_tools.go | 1,493 | Main struct, registration, task CRUD |
| path_utils.go | 340 | Path correction utilities |
| knowledge_tools.go | 236 | Knowledge base tools |
| prompt_notes_tools.go | 353 | Prompt notes tools |
| mcp_tools.go | 209 | MCP discovery tools |
| subagent_tools.go | 2,423 | Subagent execution |
| **Total** | **5,054** | Split into 6 focused files |

**Note**: The original Phase 2 (task_tools.go) was kept in coordinator_tools.go since task management tools are tightly coupled with the main coordinator logic.
