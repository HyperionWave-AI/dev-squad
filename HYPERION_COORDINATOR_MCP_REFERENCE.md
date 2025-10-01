# Hyperion Coordinator MCP - Complete Tool Reference

**Version:** 1.0
**Last Updated:** 2025-10-01
**Purpose:** Authoritative reference for all agents on how to interact with hyperion-coordinator MCP

---

## 🚨 CRITICAL: Tool Naming Convention

**ALL coordinator MCP tools use the prefix:** `mcp__hyperion-coordinator__`

**Example:**
```typescript
// ✅ CORRECT
mcp__hyperion-coordinator__coordinator_list_human_tasks({})

// ❌ WRONG - Missing mcp__ prefix
coordinator_list_human_tasks({})

// ❌ WRONG - Wrong server name
mcp__coordinator__coordinator_list_human_tasks({})
```

---

## 📋 Available Tools

### 1. List Human Tasks

**Tool Name:** `mcp__hyperion-coordinator__coordinator_list_human_tasks`

**Description:** Retrieve all human tasks from MongoDB (real-time, always fresh data)

**Parameters:** NONE (empty object)

**Example:**
```typescript
mcp__hyperion-coordinator__coordinator_list_human_tasks({})
```

**Returns:**
```json
{
  "tasks": [
    {
      "id": "uuid",
      "prompt": "User's request",
      "status": "pending|in_progress|completed|blocked",
      "createdAt": "2025-10-01T...",
      "updatedAt": "2025-10-01T..."
    }
  ]
}
```

---

### 2. List Agent Tasks

**Tool Name:** `mcp__hyperion-coordinator__coordinator_list_agent_tasks`

**Description:** Retrieve agent tasks, optionally filtered by agent name or human task

**Parameters:**
- `agentName` (string, optional): Filter by specific agent
- `humanTaskId` (string, optional): Filter by parent human task

**Examples:**
```typescript
// List all agent tasks
mcp__hyperion-coordinator__coordinator_list_agent_tasks({})

// List tasks for specific agent
mcp__hyperion-coordinator__coordinator_list_agent_tasks({
  agentName: "Backend Services Specialist"
})

// List tasks for specific human task
mcp__hyperion-coordinator__coordinator_list_agent_tasks({
  humanTaskId: "4361dcdb-3781-4686-88d3-3feb20c6948e"
})
```

**Returns:**
```json
{
  "tasks": [
    {
      "id": "uuid",
      "humanTaskId": "uuid",
      "agentName": "Backend Services Specialist",
      "role": "Build Go backend...",
      "status": "pending|in_progress|completed|blocked",
      "todos": [
        {
          "id": "uuid",
          "description": "Set up project structure",
          "status": "pending|in_progress|completed",
          "createdAt": "2025-10-01T...",
          "completedAt": "2025-10-01T..." // if completed
        }
      ],
      "createdAt": "2025-10-01T...",
      "updatedAt": "2025-10-01T..."
    }
  ]
}
```

---

### 3. Create Human Task

**Tool Name:** `mcp__hyperion-coordinator__coordinator_create_human_task`

**Description:** Create a new human task (user request/prompt)

**Parameters:**
- `prompt` (string, REQUIRED): The user's request or task description

**Example:**
```typescript
mcp__hyperion-coordinator__coordinator_create_human_task({
  prompt: "Build a feature to export data to CSV format"
})
```

**Returns:**
```json
{
  "taskId": "uuid",
  "status": "pending",
  "createdAt": "2025-10-01T..."
}
```

---

### 4. Create Agent Task

**Tool Name:** `mcp__hyperion-coordinator__coordinator_create_agent_task`

**Description:** Create a new agent task with TODOs under a human task

**Parameters:**
- `humanTaskId` (string, REQUIRED): Parent human task UUID
- `agentName` (string, REQUIRED): Name of the agent (must match your agent identity)
- `role` (string, REQUIRED): Description of agent's role/responsibility
- `todos` (array of strings, REQUIRED): List of TODO descriptions

**Example:**
```typescript
mcp__hyperion-coordinator__coordinator_create_agent_task({
  humanTaskId: "4361dcdb-3781-4686-88d3-3feb20c6948e",
  agentName: "Backend Services Specialist",
  role: "Build REST API endpoints for data export",
  todos: [
    "Create export service interface",
    "Implement CSV formatter",
    "Add export endpoint to API router",
    "Write unit tests"
  ]
})
```

**Returns:**
```json
{
  "taskId": "uuid",
  "agentName": "Backend Services Specialist",
  "status": "pending",
  "todos": [
    {"id": "uuid", "description": "...", "status": "pending"}
  ]
}
```

---

### 5. Update Task Status

**Tool Name:** `mcp__hyperion-coordinator__coordinator_update_task_status`

**Description:** Update the status of a human task or agent task

**Parameters:**
- `taskId` (string, REQUIRED): Task UUID (human or agent task)
- `status` (string, REQUIRED): New status - one of: `pending`, `in_progress`, `completed`, `blocked`
- `notes` (string, optional): Progress notes or context

**Example:**
```typescript
// Mark agent task as in progress
mcp__hyperion-coordinator__coordinator_update_task_status({
  taskId: "7b22374a-58a6-47fa-8790-978c6d2d4e5b",
  status: "in_progress",
  notes: "Started implementation of backend API"
})

// Mark as completed
mcp__hyperion-coordinator__coordinator_update_task_status({
  taskId: "7b22374a-58a6-47fa-8790-978c6d2d4e5b",
  status: "completed",
  notes: "All endpoints implemented and tested"
})

// Mark as blocked
mcp__hyperion-coordinator__coordinator_update_task_status({
  taskId: "7b22374a-58a6-47fa-8790-978c6d2d4e5b",
  status: "blocked",
  notes: "Waiting for database schema approval from Data Platform Specialist"
})
```

**Returns:**
```json
{
  "success": true,
  "taskId": "uuid",
  "newStatus": "in_progress"
}
```

---

### 6. Update TODO Status

**Tool Name:** `mcp__hyperion-coordinator__coordinator_update_todo_status`

**Description:** Update the status of an individual TODO item within an agent task

**⚠️ CRITICAL - Correct Parameters:**
- `agentTaskId` (string, REQUIRED): The agent task UUID (NOT `taskId`)
- `todoId` (string, REQUIRED): The TODO item UUID (NOT `todoIndex`)
- `status` (string, REQUIRED): New status - one of: `pending`, `in_progress`, `completed`
- `notes` (string, optional): Progress notes

**❌ COMMON MISTAKES:**
```typescript
// WRONG - using taskId instead of agentTaskId
mcp__hyperion-coordinator__coordinator_update_todo_status({
  taskId: "...",  // ❌ WRONG PARAMETER NAME
  todoId: "...",
  status: "completed"
})

// WRONG - using todoIndex instead of todoId
mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: "...",
  todoIndex: 0,  // ❌ WRONG - doesn't exist
  status: "completed"
})
```

**✅ CORRECT USAGE:**
```typescript
mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: "7b22374a-58a6-47fa-8790-978c6d2d4e5b",  // ✅ Agent task UUID
  todoId: "fab8f464-a42a-4fd4-adbe-ca30825e4440",      // ✅ TODO item UUID
  status: "completed",
  notes: "Project structure created successfully"
})
```

**How to get TODO IDs:**
```typescript
// First, list your agent tasks to see TODO items with their IDs
const result = await mcp__hyperion-coordinator__coordinator_list_agent_tasks({
  agentName: "Backend Services Specialist"
})

// Result includes todos array with IDs:
// {
//   "id": "7b22374a-58a6-47fa-8790-978c6d2d4e5b",  // <-- agentTaskId
//   "todos": [
//     {
//       "id": "fab8f464-a42a-4fd4-adbe-ca30825e4440",  // <-- todoId
//       "description": "Set up project structure",
//       "status": "pending"
//     }
//   ]
// }

// Then update specific TODO by its ID
mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: "7b22374a-58a6-47fa-8790-978c6d2d4e5b",
  todoId: "fab8f464-a42a-4fd4-adbe-ca30825e4440",
  status: "completed"
})
```

**Returns:**
```json
{
  "success": true,
  "agentTaskId": "uuid",
  "todoId": "uuid",
  "newStatus": "completed"
}
```

**Auto-Completion:**
When ALL TODOs in an agent task are marked as `completed`, the agent task status is automatically updated to `completed`.

---

### 7. Clear Task Board

**Tool Name:** `mcp__hyperion-coordinator__coordinator_clear_task_board`

**Description:** Delete ALL human tasks and agent tasks (destructive operation)

**Parameters:**
- `confirm` (boolean, REQUIRED): Must be `true` to confirm deletion

**Example:**
```typescript
mcp__hyperion-coordinator__coordinator_clear_task_board({
  confirm: true
})
```

**⚠️ WARNING:** This deletes ALL tasks and cannot be undone!

**Returns:**
```json
{
  "humanTasksDeleted": 5,
  "agentTasksDeleted": 15,
  "clearedAt": "2025-10-01T..."
}
```

---

### 8. Upsert Knowledge

**Tool Name:** `mcp__hyperion-coordinator__coordinator_upsert_knowledge`

**Description:** Store knowledge in the coordinator's Qdrant knowledge base

**Parameters:**
- `collection` (string, REQUIRED): Collection name (e.g., `task:hyperion://task/human/{id}`)
- `text` (string, REQUIRED): Knowledge content to store
- `metadata` (object, optional): Additional metadata (taskId, agentName, etc.)

**Example:**
```typescript
mcp__hyperion-coordinator__coordinator_upsert_knowledge({
  collection: "task:hyperion://task/human/4361dcdb-3781-4686-88d3-3feb20c6948e",
  text: "Implemented CSV export feature using streaming to handle large datasets. Key gotcha: must set Content-Type header to text/csv and Content-Disposition to attachment.",
  metadata: {
    taskId: "7b22374a-58a6-47fa-8790-978c6d2d4e5b",
    agentName: "Backend Services Specialist",
    completedAt: "2025-10-01T10:00:00Z",
    tags: ["csv", "export", "streaming"]
  }
})
```

**Returns:**
```json
{
  "success": true,
  "knowledgeId": "uuid"
}
```

---

### 9. Query Knowledge

**Tool Name:** `mcp__hyperion-coordinator__coordinator_query_knowledge`

**Description:** Search the coordinator's knowledge base using semantic similarity

**Parameters:**
- `collection` (string, REQUIRED): Collection to search
- `query` (string, REQUIRED): Search query
- `limit` (number, optional): Max results (default: 5)

**Example:**
```typescript
mcp__hyperion-coordinator__coordinator_query_knowledge({
  collection: "task:hyperion://task/human/4361dcdb-3781-4686-88d3-3feb20c6948e",
  query: "How to implement CSV export with streaming",
  limit: 3
})
```

**Returns:**
```json
{
  "results": [
    {
      "text": "Knowledge entry content...",
      "metadata": {...},
      "score": 0.95
    }
  ]
}
```

---

## 🔄 Complete Workflow Example

### Scenario: Agent completes a task with 3 TODOs

```typescript
// Step 1: List your assigned tasks
const myTasks = await mcp__hyperion-coordinator__coordinator_list_agent_tasks({
  agentName: "Backend Services Specialist"
})

// Get your task and TODO IDs
const myTask = myTasks.tasks[0]
const agentTaskId = myTask.id
const todos = myTask.todos

// Step 2: Mark task as in progress
await mcp__hyperion-coordinator__coordinator_update_task_status({
  taskId: agentTaskId,
  status: "in_progress",
  notes: "Starting backend implementation"
})

// Step 3: Work through TODOs one by one
// TODO 1: Start
await mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: agentTaskId,
  todoId: todos[0].id,
  status: "in_progress",
  notes: "Setting up project structure"
})

// TODO 1: Complete
await mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: agentTaskId,
  todoId: todos[0].id,
  status: "completed",
  notes: "Project structure created with cmd/ and internal/ layout"
})

// TODO 2: Start and complete
await mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: agentTaskId,
  todoId: todos[1].id,
  status: "in_progress"
})

await mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: agentTaskId,
  todoId: todos[1].id,
  status: "completed",
  notes: "CSV export endpoint implemented"
})

// TODO 3: Complete
await mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: agentTaskId,
  todoId: todos[2].id,
  status: "in_progress"
})

await mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: agentTaskId,
  todoId: todos[2].id,
  status: "completed",
  notes: "All tests passing"
})

// Step 4: Task automatically marked as completed when all TODOs done!
// No need to manually update task status

// Step 5: Store knowledge for future reference
await mcp__hyperion-coordinator__coordinator_upsert_knowledge({
  collection: "task:hyperion://task/human/" + myTask.humanTaskId,
  text: "CSV Export Implementation: Used streaming approach with io.Pipe() to handle large datasets efficiently. Set headers: Content-Type: text/csv, Content-Disposition: attachment.",
  metadata: {
    taskId: agentTaskId,
    agentName: "Backend Services Specialist",
    completedAt: new Date().toISOString(),
    tags: ["backend", "csv", "export", "streaming"]
  }
})
```

---

## 🐛 Common Errors and Solutions

### Error: "Parameter 'agentTaskId' is required"
**Cause:** Using `taskId` instead of `agentTaskId` in `coordinator_update_todo_status`
**Solution:** Use correct parameter name:
```typescript
// ❌ WRONG
coordinator_update_todo_status({ taskId: "...", todoId: "...", status: "completed" })

// ✅ CORRECT
mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: "...",
  todoId: "...",
  status: "completed"
})
```

### Error: "Tool not found: coordinator_list_human_tasks"
**Cause:** Missing `mcp__hyperion-coordinator__` prefix
**Solution:** Always include full tool name:
```typescript
// ❌ WRONG
coordinator_list_human_tasks({})

// ✅ CORRECT
mcp__hyperion-coordinator__coordinator_list_human_tasks({})
```

### Error: "Parameter 'todoId' must be a non-empty string"
**Cause:** Using `todoIndex` or integer index instead of UUID
**Solution:** Get TODO ID from agent task listing:
```typescript
// First get the TODO IDs
const tasks = await mcp__hyperion-coordinator__coordinator_list_agent_tasks({ agentName: "..." })
const todoId = tasks.tasks[0].todos[0].id  // Get UUID

// Then use the UUID
await mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: "...",
  todoId: todoId,  // ✅ Use UUID, not index
  status: "completed"
})
```

---

## ✅ Best Practices

1. **Always get fresh task data** - Call `coordinator_list_agent_tasks` to get current TODO IDs before updating
2. **Update TODO status progressively** - Mark as `in_progress` when starting, `completed` when done
3. **Include meaningful notes** - Helps other agents and humans understand progress
4. **Store knowledge after completion** - Document learnings for future tasks
5. **Use exact agent names** - Must match your identity exactly (case-sensitive)
6. **Check for blocking dependencies** - Use `blocked` status when waiting on other squads

---

## 📞 Support

If you encounter issues with coordinator MCP tools:
1. Verify tool name includes `mcp__hyperion-coordinator__` prefix
2. Check parameter names match exactly (camelCase, no typos)
3. Ensure you're using UUIDs, not indices or integers
4. Review this reference document for correct usage

**Last Updated:** 2025-10-01
**Maintained By:** Platform Team
