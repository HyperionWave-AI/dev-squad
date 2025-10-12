# Complete MCP Tools Inventory for Chat Integration

## Total: 31 MCP Tools (from coordinator/mcp-server/main.go:447)

### ✅ Category 1: Coordinator Tools (17 tools)
**Handler:** `toolHandler.RegisterToolHandlers()`
**File:** `coordinator/mcp-server/handlers/tools.go`

1. `coordinator_upsert_knowledge` - Store knowledge entries
2. `coordinator_query_knowledge` - Search knowledge base
3. `coordinator_get_popular_collections` - Get top collections
4. `coordinator_create_human_task` - Create human task
5. `coordinator_create_agent_task` - Create agent task
6. `coordinator_update_task_status` - Update task status
7. `coordinator_update_todo_status` - Update TODO status
8. `coordinator_list_human_tasks` - List all human tasks
9. `coordinator_list_agent_tasks` - List agent tasks (paginated)
10. `coordinator_get_agent_task` - Get single agent task
11. `coordinator_clear_task_board` - Clear all tasks (DESTRUCTIVE)
12. `coordinator_add_task_prompt_notes` - Add guidance notes to task
13. `coordinator_update_task_prompt_notes` - Update task guidance
14. `coordinator_clear_task_prompt_notes` - Clear task guidance
15. `coordinator_add_todo_prompt_notes` - Add guidance to TODO
16. `coordinator_update_todo_prompt_notes` - Update TODO guidance
17. `coordinator_clear_todo_prompt_notes` - Clear TODO guidance

### ✅ Category 2: Qdrant Tools (2 tools)
**Handler:** `qdrantToolHandler.RegisterQdrantTools()`
**File:** `coordinator/mcp-server/handlers/qdrant_tools.go`

18. `knowledge_find` - Semantic search in knowledge base
19. `knowledge_store` - Store knowledge with embeddings

### ✅ Category 3: Code Index Tools (5 tools)
**Handler:** `codeToolsHandler.RegisterCodeIndexTools()`
**File:** `coordinator/mcp-server/handlers/code_tools.go`

20. `code_index_add_folder` - Add folder to code index
21. `code_index_scan` - Scan/rescan folder
22. `code_index_search` - Search indexed code
23. `code_index_status` - Get indexing status
24. `code_index_remove_folder` - Remove folder from index

### ✅ Category 4: Filesystem Tools (4 tools)
**Handler:** `filesystemToolsHandler.RegisterFilesystemTools()`
**File:** `coordinator/mcp-server/handlers/filesystem_tools.go`

25. `bash` - Execute shell commands
26. `file_read` - Read file contents
27. `file_write` - Write file contents
28. `apply_patch` - Apply unified diff patches

**Note:** Missing `list_directory` tool - only 4 filesystem tools registered instead of 5.

### ✅ Category 5: Tools Discovery (3 tools)
**Handler:** `toolsDiscoveryHandler.RegisterToolsDiscoveryTools()`
**File:** `coordinator/mcp-server/handlers/tools_discovery.go`

29. `discover_tools` - List all available tools
30. `get_tool_schema` - Get tool schema by name
31. `execute_tool` - Execute tool by name

---

## Current Chat Integration Status

### ✅ Already Wrapped (from task c858e560):
- `create_agent_task` (coordinator)
- `list_agent_tasks` (coordinator)
- `query_knowledge` (coordinator)
- `knowledge_find` (Qdrant)
- `knowledge_store` (Qdrant)
- `code_index_search` (code index)
- `code_index_add_folder` (code index)

**Total Wrapped:** 7 tools

### ❌ Missing Wrappers (24 tools):

**Coordinator (14 missing):**
1. coordinator_upsert_knowledge
2. coordinator_get_popular_collections
3. coordinator_create_human_task
4. coordinator_update_task_status
5. coordinator_update_todo_status
6. coordinator_list_human_tasks
7. coordinator_get_agent_task
8. coordinator_clear_task_board (⚠️ DESTRUCTIVE - exclude from chat?)
9. coordinator_add_task_prompt_notes
10. coordinator_update_task_prompt_notes
11. coordinator_clear_task_prompt_notes
12. coordinator_add_todo_prompt_notes
13. coordinator_update_todo_prompt_notes
14. coordinator_clear_todo_prompt_notes

**Code Index (3 missing):**
15. code_index_scan
16. code_index_status
17. code_index_remove_folder

**Filesystem (4 missing - ALREADY IMPLEMENTED in Phase 1):**
18. bash (✅ implemented in Phase 1)
19. file_read (✅ implemented as ReadFileTool in Phase 1)
20. file_write (✅ implemented as WriteFileTool in Phase 1)
21. apply_patch (✅ implemented as ApplyPatchTool in Phase 1)

**Tools Discovery (3 missing):**
22. discover_tools
23. get_tool_schema
24. execute_tool

---

## Implementation Strategy

### Phase 1: ✅ COMPLETE - Filesystem Tools
Already implemented in task e8195c80 with LangChain Go `tools.Tool` interface:
- BashTool, ReadFileTool, WriteFileTool, ListDirectoryTool, ApplyPatchTool

### Phase 2: ✅ PARTIAL - MCP Tool Wrappers
Task c858e560 implemented 7 wrappers. Need to complete remaining 17 coordinator wrappers + 3 code index + 3 tools discovery.

### Phase 3: NEW - Complete All MCP Wrappers
Create wrappers for ALL MCP tools by calling internal handlers directly.

---

## Recommendation

**For Chat Integration:**

1. **INCLUDE (28 tools):**
   - All 17 coordinator tools (except clear_task_board)
   - All 2 Qdrant tools
   - All 5 code index tools
   - All 4 filesystem tools (already implemented)
   - 0 tools discovery tools (meta-tools not needed in chat)

2. **EXCLUDE (3 tools):**
   - `coordinator_clear_task_board` - Destructive, requires explicit confirmation
   - `discover_tools` - Meta-tool, redundant in chat context
   - `get_tool_schema` - Meta-tool, not needed
   - `execute_tool` - Meta-tool, security risk (arbitrary tool execution)

3. **TOTAL TOOLS IN CHAT: 28 tools**

---

## Next Steps

1. Complete coordinator tool wrappers (14 remaining)
2. Complete code index tool wrappers (3 remaining)
3. Register all tools in ToolRegistry by default
4. Update ChatService initialization to register all tools automatically
5. Test tool availability in chat UI
