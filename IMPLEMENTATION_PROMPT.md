# Human Prompt Notes - Implementation Prompt

**Copy and paste this into a NEW Claude Code session:**

---

I want to implement the Human Prompt Notes feature for the Hyperion Coordinator MCP system.

## Context

The complete specification is in `HUMAN_PROMPT_NOTES_SPEC.md` - **please read it thoroughly before proceeding**.

This feature enables humans to add guidance notes to agent tasks and TODOs after AI planning completes. Agents must read and prioritize these notes during implementation.

## Architecture Overview

This system has **THREE separate projects** that need coordination:

### 1. MCP Server (`coordinator/mcp-server/`)
- Core MCP tool implementation using Go
- MongoDB storage layer
- Handles all business logic
- **Primary work location**

### 2. HTTP Bridge (`coordinator/mcp-http-bridge/`)
- Stateless HTTP ↔ MCP stdio translator
- **Should require NO code changes** (verifies dynamically)
- Only needs testing/verification

### 3. UI (`coordinator/ui/`)
- React + TypeScript frontend
- Calls HTTP bridge endpoints
- Renders task management UI

## Your Task

Act as the **Workflow Coordinator** and create properly structured agent tasks for **Phase 1: MCP Server Implementation**.

### Phase 1 Focus: MCP Server Only

Create agent tasks for the `coordinator/mcp-server/` project:

**Task 1: Data Model Updates** (Backend Services Specialist)
- **File:** `coordinator/mcp-server/storage/tasks.go`
- Update `AgentTask` struct: Add `humanPromptNotes`, `humanPromptNotesAddedAt`, `humanPromptNotesUpdatedAt`
- Update `TodoItem` struct: Add same 3 fields
- Update MongoDB bson tags (camelCase in JSON, same in bson)
- Ensure backward compatibility (all fields optional with omitempty)

**Task 2: Storage Interface & Methods** (Backend Services Specialist)
- **File:** `coordinator/mcp-server/storage/tasks.go`
- Add 6 new methods to `TaskStorage` interface:
  - `AddTaskPromptNotes(agentTaskID string, notes string) error`
  - `UpdateTaskPromptNotes(agentTaskID string, notes string) error`
  - `ClearTaskPromptNotes(agentTaskID string) error`
  - `AddTodoPromptNotes(agentTaskID string, todoID string, notes string) error`
  - `UpdateTodoPromptNotes(agentTaskID string, todoID string, notes string) error`
  - `ClearTodoPromptNotes(agentTaskID string, todoID string) error`
- Implement all 6 methods in `MongoTaskStorage`
- Use MongoDB `$set` for updates, array filters for TODO updates
- Set timestamps (`time.Now()`) for AddedAt and UpdatedAt fields

**Task 3: Input Validation** (Backend Services Specialist)
- **File:** `coordinator/mcp-server/storage/validation.go` (new file)
- Create `ValidatePromptNotes(notes string) (string, error)` function
- Max length: 5000 characters
- Sanitize markdown using `github.com/microcosm-cc/bluemonday`
- Allow safe markdown: headers, lists, code blocks, bold, italic, links
- Strip dangerous HTML/scripts

**Task 4: MCP Tool Registration** (go-mcp-dev)
- **File:** `coordinator/mcp-server/handlers/tools.go`
- Add 6 new tools to `GetToolsList()` method:
  - `coordinator_add_task_prompt_notes`
  - `coordinator_update_task_prompt_notes`
  - `coordinator_clear_task_prompt_notes`
  - `coordinator_add_todo_prompt_notes`
  - `coordinator_update_todo_prompt_notes`
  - `coordinator_clear_todo_prompt_notes`
- Define InputSchema for each (agentTaskId, todoId, promptNotes)
- Follow existing tool patterns in the file

**Task 5: MCP Tool Handlers** (go-mcp-dev)
- **File:** `coordinator/mcp-server/handlers/tools.go`
- Implement 6 handler functions (e.g., `handleAddTaskPromptNotes`)
- Validate parameters (agentTaskId required, promptNotes required)
- Call validation function before storage
- Call storage methods
- Return success/error in MCP format
- Add to tool routing in `HandleToolCall()` method

**Task 6: Unit Tests** (Backend Services Specialist)
- **Files:**
  - `coordinator/mcp-server/storage/tasks_test.go`
  - `coordinator/mcp-server/handlers/tools_test.go`
- Test all 6 storage methods
- Test all 6 tool handlers
- Test validation (max length, sanitization)
- Test error cases (invalid IDs, missing params)
- Test MongoDB operations (add, update, clear)
- Test TODO array updates with array filters
- Target: 90% coverage for new code

**Task 7: Documentation** (Backend Services Specialist)
- **File:** `HYPERION_COORDINATOR_MCP_REFERENCE.md`
- Add documentation for all 6 new MCP tools
- Include parameter descriptions
- Include example usage
- Include response formats

## Critical Requirements

**Use the Context-Rich Task Format from CLAUDE.md:**

Each task MUST include:

1. **`contextSummary`** (150-250 words):
   - WHY: This feature exists (human-agent collaboration)
   - WHAT: Specific deliverables for this task
   - HOW: Technical approach (MongoDB updates, MCP patterns, etc.)
   - FILES: Exact list of files to modify
   - CONSTRAINTS: Backward compatibility, security, validation
   - TESTING: How to verify it works

2. **`filesModified`** (complete list):
   - Exact file paths from project root
   - Mark new files vs existing files
   - Include test files

3. **Detailed `contextHint` for EVERY TODO** (50-100 words each):
   - Specific code pattern to follow
   - Function signatures
   - Error handling approach
   - Example code snippets if helpful
   - References to existing code (line numbers)

4. **`qdrantCollections`** (ONLY if needed):
   - Specify WHAT to search for
   - Limit to 1-2 collections max
   - Example: "Search 'go-mongodb-patterns' for array filter examples"

## Example Context-Rich TODO

**Bad (too vague):**
```json
{
  "description": "Add MongoDB method",
  "filePath": "storage/tasks.go"
}
```

**Good (context-rich):**
```json
{
  "description": "Implement AddTaskPromptNotes MongoDB method",
  "filePath": "coordinator/mcp-server/storage/tasks.go",
  "functionName": "AddTaskPromptNotes",
  "contextHint": "Use MongoDB UpdateOne with $set operator. Update fields: humanPromptNotes (string), humanPromptNotesAddedAt (time.Now()), humanPromptNotesUpdatedAt (time.Now()). Filter by taskId field. Return error if update fails. Pattern: See UpdateTaskStatus() method at line 245 for similar update pattern."
}
```

## Security Requirements (from CLAUDE.md)

**CRITICAL:** ALL MongoDB operations MUST use JWT identity from context.

```go
// ✅ CORRECT
identity, err := auth.GetIdentityFromContext(ctx)
// Use identity for authorization checks

// ❌ FORBIDDEN
systemIdentity := &models.Identity{Type: "service"}
```

For this feature:
- Verify user has permission to add notes (task creator or admin)
- Extract identity from context
- Log access attempts (optional in v1)

## What NOT to Do

❌ Don't create tasks for HTTP Bridge yet (Phase 2)
❌ Don't create tasks for UI yet (Phase 3)
❌ Don't create generic "implement feature" tasks
❌ Don't skip contextHint fields in TODOs
❌ Don't create tasks without filesModified lists
❌ Don't reference non-existent patterns

## Expected Output

Create **7 well-structured agent tasks** with:
- Clear, specific roles
- 150-250 word contextSummary for each
- Complete filesModified lists
- Detailed TODOs with contextHints
- Testing requirements included
- References to existing code patterns

## After Task Creation

Once tasks are created, **I will review them** before implementation begins.

If tasks lack detail or context, I'll ask you to expand them before proceeding.

## Questions?

If you need clarification on:
- Existing codebase patterns
- MongoDB schemas
- MCP tool patterns
- Security requirements

**Ask BEFORE creating tasks** - don't guess or make assumptions.

---

**Ready? Start by reading `HUMAN_PROMPT_NOTES_SPEC.md` and examining the existing `coordinator/mcp-server/` codebase.**
