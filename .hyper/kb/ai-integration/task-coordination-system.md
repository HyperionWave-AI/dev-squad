# Hyperion Task Coordination System - AI-Powered Workflow Orchestration

**Collection:** ai-integration
**Tags:** task-coordination, workflow, delegation, subagents
**File Reference:** handlers/chat_websocket.go:27-150
**Version:** 1.0

---

HYPERION TASK COORDINATION SYSTEM - AI-POWERED WORKFLOW ORCHESTRATION

COORDINATOR WORKFLOW (default system prompt):
Chat interface routes tasks through coordinator:

6-STEP MANDATORY WORKFLOW (chat_websocket.go:48-150):

1. CHECK EXISTING TASKS (1 tool call):
   - List pending tasks to prevent duplicates
   - Present similar task if found

2. CREATE HUMAN TASK (1 tool call):
   - Record user request verbatim
   - Returns humanTaskId for traceability

3. PRESENT IMPLEMENTATION OPTIONS (NO TOOL CALLS):
   - Show 2-3 approaches when appropriate
   - Skip for trivial fixes
   - Wait for user choice

4. CODE SEARCH (1 tool call):
   - code_index_search with best query
   - Extract FILE_PATHS_TO_USE (only valid paths)
   - Use first results without variations

5. CREATE AGENT TASK (1 tool call):
   - Assign to specialist subagent (ui-dev, go-dev, sre, etc.)
   - Include context summary and TODO items
   - Link to human task for traceability

6. DELEGATE TO SUBAGENT:
   - Agent implements via MCP tools
   - Updates task status and TODOs
   - Records decisions and outcomes

SUBAGENT TYPES:
- ui-dev: React/TypeScript frontend
- go-dev: Go backend services
- sre: Infrastructure/DevOps
- qa-tester: Quality assurance
- coordinator: Task orchestration

TASK HIERARCHY:
- HumanTask: User request (1:many with AgentTasks)
- AgentTask: Specialized work (contains TodoItems)
- TodoItem: Individual work unit (has status, notes)

TRACEABILITY:
- humanTaskId → agentTaskIds (bidirectional)
- Agent task → filesModified, qdrantCollections
- TODO → filePath, functionName for targeted changes

AI COORDINATION BENEFITS:
- Context routing: Right specialist for each task
- Parallel execution: Multiple agents simultaneously
- Knowledge capture: Lessons extracted from decisions
- Reflection: System learns from outcomes
