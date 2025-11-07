# MCP Tools Analysis - Unified Hyper Binary vs Old Coordinator

**Date:** 2025-10-12
**Status:** ⚠️ CRITICAL - 9 tools missing from unified hyper binary

---

## Executive Summary

The unified hyper binary (`bin/hyper`) is missing **9 MCP tools** that were present in the old coordinator implementation. Additionally, 2 handler registrations are missing from `main.go`.

**Impact:** AI agents using the unified hyper binary via MCP will not have access to filesystem operations, tools discovery, or subagent management.

---

## Missing Tools Breakdown

### 1. Filesystem Tools (4 tools) ❌ NOT REGISTERED

**Tools:**
- `file_read` - Read file contents
- `file_write` - Write file contents
- `bash` - Execute bash commands
- `apply_patch` - Apply unified diff patches

**Handler Status:**
- ✅ Handler file EXISTS: `hyper/internal/mcp/handlers/filesystem_tools.go`
- ✅ Registration method exists: `RegisterFilesystemTools(server *mcp.Server)`
- ❌ NOT registered in `hyper/cmd/coordinator/main.go:createMCPServer`

**Found in archived version:**
```go
// hyper/.archived/mcp-server/main.go:210
handlers.NewFilesystemToolHandler(logger).RegisterFilesystemTools(server)
```

**Missing from current version:**
```go
// hyper/cmd/coordinator/main.go:createMCPServer (lines 213-244)
// filesystemToolHandler registration is ABSENT
```

**Fix Required:**
```go
// Add to createMCPServer function after line 224
filesystemToolHandler := handlers.NewFilesystemToolHandler(logger)
must(filesystemToolHandler.RegisterFilesystemTools(server))
```

---

### 2. Tools Discovery (3 tools) ❌ NOT REGISTERED

**Tools:**
- `discover_tools` - Discover MCP tools from other servers
- `get_tool_schema` - Get JSON schema for a tool
- `execute_tool` - Execute a discovered tool

**Handler Status:**
- ✅ Handler file EXISTS: `hyper/internal/mcp/handlers/tools_discovery.go`
- ✅ Registration method exists: `RegisterToolsDiscoveryTools(server *mcp.Server)`
- ❌ NOT registered in `hyper/cmd/coordinator/main.go:createMCPServer`
- ❌ Requires `toolsStorage` which is NOT initialized in unified main.go

**Found in archived version:**
```go
// hyper/.archived/mcp-server/main.go:108
toolsStorage, err := storage.NewToolsStorage(db, qdrantClient)

// hyper/.archived/mcp-server/main.go:211
handlers.NewToolsDiscoveryHandler(toolsStorage).RegisterToolsDiscoveryTools(server)
```

**Missing from current version:**
```go
// hyper/cmd/coordinator/main.go:createMCPServer
// toolsDiscoveryHandler registration is ABSENT
// toolsStorage initialization is ABSENT
```

**Fix Required:**
```go
// 1. Add to main() function after codeIndexStorage initialization (line 228):
toolsStorage, err := storage.NewToolsStorage(db, qdrantClient)
if err != nil {
    logger.Fatal("Failed to initialize tools storage", zap.Error(err))
}
logger.Info("Tools storage initialized")

// 2. Add to createMCPServer function signature (line 189):
func createMCPServer(
    taskStorage storage.TaskStorage,
    knowledgeStorage storage.KnowledgeStorage,
    codeIndexStorage *storage.CodeIndexStorage,
    qdrantClient *storage.QdrantClient,
    embeddingClient embeddings.EmbeddingClient,
    fileWatcher *watcher.FileWatcher,
    mongoClient *mongo.Client,
    toolsStorage *storage.ToolsStorage, // ADD THIS
    logger *zap.Logger,
) *mcp.Server {

// 3. Add to createMCPServer function after line 224:
toolsDiscoveryHandler := handlers.NewToolsDiscoveryHandler(toolsStorage)
must(toolsDiscoveryHandler.RegisterToolsDiscoveryTools(server))

// 4. Update createMCPServer call in main() (line 395):
mcpServer := createMCPServer(taskStorage, knowledgeStorage, codeIndexStorage,
    qdrantClient, embeddingClient, fileWatcher, mongoClient, toolsStorage, logger)
```

---

### 3. Subagent Management (2 tools) ❌ MISSING FROM HANDLER

**Tools:**
- `list_subagents` - List available specialist agents
- `set_current_subagent` - Associate subagent with chat session

**Handler Status:**
- ✅ EXISTS in old coordinator: `coordinator/mcp-server/handlers/tools.go:1517-1660`
- ❌ MISSING from unified hyper: `hyper/internal/mcp/handlers/tools.go` (not present)

**Found in old coordinator:**
```go
// coordinator/mcp-server/handlers/tools.go:119-126
// Register list_subagents
if err := h.registerListSubagents(server); err != nil {
    return fmt.Errorf("failed to register list_subagents tool: %w", err)
}
// Register set_current_subagent
if err := h.registerSetCurrentSubagent(server); err != nil {
    return fmt.Errorf("failed to register set_current_subagent tool: %w", err)
}
```

**Missing from unified hyper:**
- No `registerListSubagents` method
- No `registerSetCurrentSubagent` method
- No `handleListSubagents` function
- No `handleSetCurrentSubagent` function

**Fix Required:**
Copy the following functions from `coordinator/mcp-server/handlers/tools.go` to `hyper/internal/mcp/handlers/tools.go`:
- Lines 1517-1536: `registerListSubagents`
- Lines 1538-1600: `handleListSubagents`
- Lines 1602-1629: `registerSetCurrentSubagent`
- Lines 1631-1660: `handleSetCurrentSubagent`

Then add registrations to `RegisterToolHandlers` method.

---

## Complete Tool Inventory

### ✅ Tools Present in Unified Hyper (24 tools)

#### Coordinator Tools (17 tools)
- `coordinator_create_human_task`
- `coordinator_create_agent_task`
- `coordinator_list_human_tasks`
- `coordinator_list_agent_tasks`
- `coordinator_get_agent_task`
- `coordinator_update_task_status`
- `coordinator_update_todo_status`
- `coordinator_add_task_prompt_notes`
- `coordinator_update_task_prompt_notes`
- `coordinator_clear_task_prompt_notes`
- `coordinator_add_todo_prompt_notes`
- `coordinator_update_todo_prompt_notes`
- `coordinator_clear_todo_prompt_notes`
- `coordinator_upsert_knowledge`
- `coordinator_query_knowledge`
- `coordinator_get_popular_collections`
- `coordinator_clear_task_board`

#### Qdrant Tools (2 tools)
- `knowledge_find` (renamed from qdrant_find)
- `knowledge_store` (renamed from qdrant_store)

#### Code Indexing Tools (5 tools)
- `code_index_add_folder`
- `code_index_remove_folder`
- `code_index_scan`
- `code_index_search`
- `code_index_status`

**Total Present:** 24 tools

---

### ❌ Tools Missing from Unified Hyper (9 tools)

#### Filesystem Tools (4 tools) - Handler exists, not registered
- `file_read`
- `file_write`
- `bash`
- `apply_patch`

#### Tools Discovery (3 tools) - Handler exists, not registered
- `discover_tools`
- `get_tool_schema`
- `execute_tool`

#### Subagent Management (2 tools) - Handler missing
- `list_subagents`
- `set_current_subagent`

**Total Missing:** 9 tools

---

## Resources & Prompts Inventory

### ✅ MCP Resources (12 resources) - All Present

#### Documentation Resources (3)
- `hyperion://docs/standards`
- `hyperion://docs/architecture`
- `hyperion://docs/squad-guide`

#### Workflow Resources (3)
- `hyperion://workflow/active-agents`
- `hyperion://workflow/task-queue`
- `hyperion://workflow/dependencies`

#### Knowledge Resources (2)
- `hyperion://knowledge/collections`
- `hyperion://knowledge/recent-learnings`

#### Metrics Resources (2)
- `hyperion://metrics/squad-velocity`
- `hyperion://metrics/context-efficiency`

#### Dynamic Task Resources (2)
- `hyperion://task/human/{id}`
- `hyperion://task/agent/{agent}/{id}`

**All resources are properly registered in unified hyper.**

---

### ✅ MCP Prompts (7 prompts) - All Present

#### Planning Prompts (4)
- `plan_task_breakdown`
- `suggest_context_offload`
- `detect_cross_squad_impact`
- `suggest_handoff_strategy`

#### Knowledge Prompts (3)
- `recommend_knowledge_query`
- `guide_knowledge_storage`
- `suggest_knowledge_structure`

**All prompts are properly registered in unified hyper.**

---

## Handler Files Comparison

### Identical Files (No Differences)
- `knowledge_resources.go` ✅
- `knowledge_prompts.go` ✅
- `documentation_prompts.go` ✅
- `qdrant_tools.go` ✅
- `code_tools.go` ✅
- `planning_prompts.go` ✅
- `metrics_resources.go` ✅
- `health.go` ✅
- `resources.go` ✅
- `doc_resources.go` ✅
- `workflow_resources.go` ✅
- `coordination_prompts.go` ✅

### Files with Differences
- `tools.go` - Missing 2 subagent tools in hyper version
- `filesystem_tools.go` - Exists in hyper but not registered
- `tools_discovery.go` - Exists in hyper but not registered

### Files Only in Hyper
- `filesystem_tools_patch_test.go` - Additional test file (not critical)

---

## Registration Status in main.go

### ✅ Handlers Registered in Unified Hyper (createMCPServer:213-244)

```go
must(resourceHandler.RegisterResourceHandlers(server))
must(docResourceHandler.RegisterDocResources(server))
must(workflowResourceHandler.RegisterWorkflowResources(server))
must(knowledgeResourceHandler.RegisterKnowledgeResources(server))
must(metricsResourceHandler.RegisterMetricsResources(server))
must(toolHandler.RegisterToolHandlers(server))
must(qdrantToolHandler.RegisterQdrantTools(server))
must(codeToolsHandler.RegisterCodeIndexTools(server))
must(planningPromptHandler.RegisterPlanningPrompts(server))
must(knowledgePromptHandler.RegisterKnowledgePrompts(server))
must(coordinationPromptHandler.RegisterCoordinationPrompts(server))
must(documentationPromptHandler.RegisterDocumentationPrompts(server))
```

**Total: 12 handlers registered**

---

### ❌ Handlers Missing from Unified Hyper

```go
// MISSING - Handler file exists
filesystemToolHandler := handlers.NewFilesystemToolHandler(logger)
must(filesystemToolHandler.RegisterFilesystemTools(server))

// MISSING - Handler file exists, but requires toolsStorage initialization
toolsDiscoveryHandler := handlers.NewToolsDiscoveryHandler(toolsStorage)
must(toolsDiscoveryHandler.RegisterToolsDiscoveryTools(server))
```

**These registrations are present in `.archived/mcp-server/main.go:210-211` but missing from current `cmd/coordinator/main.go`**

---

## Impact Assessment

### Critical Impact (HIGH)
1. **Filesystem Tools Missing:**
   - AI agents cannot read/write files via MCP
   - No bash command execution
   - No patch application support
   - **Workaround:** Agents must use native filesystem tools (not MCP)

2. **Subagent Management Missing:**
   - Cannot list available specialist agents
   - Cannot associate subagent with chat
   - **Workaround:** Manual agent selection, no dynamic discovery

### Medium Impact
3. **Tools Discovery Missing:**
   - Cannot dynamically discover tools from other MCP servers
   - No runtime tool schema introspection
   - **Workaround:** Static tool configuration only

---

## Recommended Fixes (Priority Order)

### Priority 1: Register Filesystem Tools (HIGH)
**Time:** 5 minutes
**Risk:** Low (handler already exists and tested)

1. Edit `hyper/cmd/coordinator/main.go:createMCPServer`
2. Add after line 224:
   ```go
   filesystemToolHandler := handlers.NewFilesystemToolHandler(logger)
   must(filesystemToolHandler.RegisterFilesystemTools(server))
   ```
3. Test with `make build && bin/hyper --mode=mcp`

---

### Priority 2: Add Subagent Tools (HIGH)
**Time:** 20 minutes
**Risk:** Medium (requires copying functions)

1. Copy from `coordinator/mcp-server/handlers/tools.go` to `hyper/internal/mcp/handlers/tools.go`:
   - Lines 1517-1660 (4 functions)
2. Add registrations to `RegisterToolHandlers` method
3. Test subagent listing and association

---

### Priority 3: Register Tools Discovery (MEDIUM)
**Time:** 10 minutes
**Risk:** Medium (requires new storage initialization)

1. Initialize `toolsStorage` in `main()` after line 228
2. Add `toolsStorage` parameter to `createMCPServer`
3. Register handler in `createMCPServer`
4. Update all `createMCPServer` calls to pass `toolsStorage`
5. Test tools discovery functionality

---

## Verification Steps

After implementing fixes:

1. **Build unified binary:**
   ```bash
   make clean
   make build
   ls -lh bin/hyper  # Should show ~17MB
   ```

2. **Start in MCP mode:**
   ```bash
   bin/hyper --mode=mcp
   ```

3. **List tools via MCP client:**
   ```json
   {"jsonrpc":"2.0","id":1,"method":"tools/list"}
   ```

4. **Expected tool count:**
   - Before fixes: 24 tools
   - After Priority 1: 28 tools (+4 filesystem)
   - After Priority 2: 30 tools (+2 subagent)
   - After Priority 3: 33 tools (+3 discovery)

5. **Test each new tool:**
   ```bash
   # Test file_read
   echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"file_read","arguments":{"path":"/tmp/test.txt"}}}' | bin/hyper --mode=mcp

   # Test list_subagents
   echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_subagents","arguments":{}}}' | bin/hyper --mode=mcp

   # Test discover_tools
   echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"discover_tools","arguments":{}}}' | bin/hyper --mode=mcp
   ```

---

## Documentation Updates Needed

After fixes are implemented, update:

1. **CLAUDE.md** - Update tool counts (31 → 33 tools)
2. **HYPERION_COORDINATOR_MCP_REFERENCE.md** - Add missing tools documentation
3. **coordinator/mcp-server/CLAUDE.md** - Mark as deprecated after verification
4. **README.md** - Update feature list with all 33 tools

---

## Architecture Notes

### Why Tools Were Missing

1. **Filesystem & Tools Discovery:**
   - Handler files were migrated from old coordinator
   - But registration calls were not added to new `main.go`
   - Likely oversight during refactoring

2. **Subagent Tools:**
   - Functions exist in old `coordinator/mcp-server/handlers/tools.go`
   - But NOT copied to new `hyper/internal/mcp/handlers/tools.go`
   - Intentional omission or forgot to migrate?

3. **No Test Coverage:**
   - Integration tests don't verify full tool count
   - No E2E test for MCP tool listing
   - Recommendation: Add test that counts all registered tools

---

## Summary

| Category | Old Coordinator | Unified Hyper | Status |
|----------|----------------|---------------|--------|
| Coordinator Tools | 17 | 17 | ✅ Complete |
| Qdrant Tools | 2 | 2 | ✅ Complete |
| Code Indexing Tools | 5 | 5 | ✅ Complete |
| **Filesystem Tools** | **4** | **0** | ❌ **Missing** |
| **Tools Discovery** | **3** | **0** | ❌ **Missing** |
| **Subagent Management** | **2** | **0** | ❌ **Missing** |
| **TOTAL** | **33** | **24** | **❌ 9 Missing (27%)** |

| Component | Count | Status |
|-----------|-------|--------|
| Resources | 12 | ✅ Complete |
| Prompts | 7 | ✅ Complete |
| Handlers Registered | 12 | ✅ Complete |

---

## Related Files

- `hyper/cmd/coordinator/main.go` - Main entry point (needs updates)
- `hyper/internal/mcp/handlers/tools.go` - Coordinator tools (needs subagent functions)
- `hyper/internal/mcp/handlers/filesystem_tools.go` - Filesystem tools (exists, not registered)
- `hyper/internal/mcp/handlers/tools_discovery.go` - Tools discovery (exists, not registered)
- `coordinator/mcp-server/handlers/tools.go` - Old coordinator (reference implementation)
- `hyper/.archived/mcp-server/main.go` - Archived version (shows what was registered)

---

**Next Steps:**
1. Review this analysis with team
2. Prioritize fixes (recommend Priority 1 & 2 first)
3. Implement fixes with tests
4. Update documentation
5. Deprecate old coordinator after verification

---

**Analysis Complete:** 2025-10-12
**Analyzed By:** Claude Code AI Assistant
**Total Tools Found:** 24/33 (9 missing)
**Criticality:** HIGH - Core functionality missing
