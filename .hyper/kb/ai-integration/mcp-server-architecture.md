# Hyperion MCP Server Architecture - Model Context Protocol

**Collection:** ai-integration
**Tags:** MCP, Model Context Protocol, tool-registration, stdio
**File Reference:** cmd/coordinator/main.go:660-729
**Version:** 1.0

---

HYPERION MCP SERVER ARCHITECTURE - MODEL CONTEXT PROTOCOL

MCP SERVER INTEGRATION (cmd/coordinator/main.go:660-729):

MCP SERVER INITIALIZATION:
Implementation:
- Name: "hyper-coordinator"
- Version: "1.0.0"
- SDK: github.com/modelcontextprotocol/go-sdk/mcp
- Transport: Stdio (line 785) for Claude integration

DUAL SERVER MODES (line 779-792):
- HTTP mode: REST API + UI on port 8080 (HTTP_PORT env)
- MCP mode: Protocol over stdio for Claude Code/API
- Both mode: Run both servers concurrently
- Configuration: --mode http|mcp|both flag

TOOL REGISTRATION (lines 698-729):
Five tool categories registered:

1. COORDINATOR TOOLS (19 tools) - lines 698-704:
   - Task management: create_human_task, create_agent_task, list_agent_tasks
   - Task updates: update_task_status, update_todo_status
   - Knowledge: upsert_knowledge, query_knowledge
   - Guidance: add/update/clear prompt notes
   - Reflection: decision tracking, lesson extraction

2. QDRANT TOOLS (2 tools) - lines 706-713:
   - knowledge_store: Store semantic knowledge
   - knowledge_find: Search similar entries
   - Integrated with MongoDB for persistence

3. REFLECTION TOOLS (3 tools) - lines 715-721:
   - record_decision: Metacognitive decision logging
   - record_outcome: Outcome tracking for calibration
   - query_relevant_lessons: Learn from past experience

4. FILESYSTEM TOOLS (5 tools) - lines 723-729:
   - bash: Execute shell commands
   - file_read: Read files (with streaming)
   - file_write: Write files (with streaming)
   - apply_patch: Apply unified diffs
   - List directories

5. MCP HUB TOOLS (optional) - lines 673-696:
   - discover_tools: Tool discovery across MCP network
   - get_tool_schema: Tool schema retrieval
   - execute_tool: Execute external MCP tools
   - mcp_add_server: Register new MCP servers
   - Controlled by MCP_HUB env (default: true)

METADATA REGISTRY:
- Tracks all registered tools
- Enables semantic indexing for discovery
- Supports tool versioning and categorization
