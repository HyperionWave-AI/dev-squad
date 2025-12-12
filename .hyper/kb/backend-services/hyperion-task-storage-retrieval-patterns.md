# HYPERION TASK STORAGE & RETRIEVAL PATTERNS

**Collection:** backend-services
**Created:** 2025-11-20

---

HYPERION TASK STORAGE & RETRIEVAL PATTERNS

MongoDB Collections for Task Management (storage/tasks.go):

HUMAN TASKS COLLECTION:
Schema (HumanTask struct, lines 67-77):
- taskId (unique): UUID identifier
- prompt: Original user request (verbatim)
- summary: AI-generated summary (≤100 tokens)
- agentTaskIds[]: References to delegated agent tasks
- status: pending | in_progress | completed | blocked
- notes: Human-added notes
- timestamps: createdAt, updatedAt

Indexes:
- taskId (unique primary)

Query Patterns:
- Get single: GetHumanTask(taskId)
- List all: ListAllHumanTasks() or ListHumanTasks(filter)
- Search: SearchSimilarHumanTasks(prompt, limit, minScore)

AGENT TASKS COLLECTION:
Schema (AgentTask struct, lines 80-98):
- taskId (unique): UUID identifier
- humanTaskId: Link to parent human task
- agentName: Specialist agent type (ui-dev, go-dev, sre)
- role: Mission description (100w)
- todos[]: Array of TodoItem (nested documents)
- contextSummary: Task context (150-250w)
- filesModified[]: File paths this task touches
- qdrantCollections[]: Relevant knowledge collections
- priorWorkSummary: Previous agent work (for multi-phase)
- status: pending | in_progress | completed | blocked
- humanPromptNotes: Human guidance text

Indexes:
- taskId (unique)
- agentName (for agent lookup)
- humanTaskId (for parent task queries)

TODO ITEMS (nested in agent_tasks):
Schema (TodoItem struct, lines 43-56):
- id: UUID (unique within task)
- description: What to do
- status: pending | in_progress | completed
- filePath: Optional file target
- functionName: Optional function target
- contextHint: Implementation guidance (50-100w)
- humanPromptNotes: Human guidance per TODO
- timestamps: createdAt, completedAt, notes

QUERY OPERATIONS:
```
ListAgentTasks(filter bson.M, offset int, limit int)
  → Returns paginated results with total count

UpdateTaskStatus(taskID string, status TaskStatus, notes string)
  → Updates task and timestamps atomically

UpdateTodoStatus(agentTaskID string, todoID string, status TodoStatus, notes string)
  → Updates nested TODO with atomic MongoDB operation
```

TRACEABILITY:
- HumanTask.agentTaskIds[]: One-to-many relationship
- AgentTask.humanTaskId: Back-reference to parent
- TodoItem in nested array: Direct parent reference
- Full audit trail: All timestamps recorded
