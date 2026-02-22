# MCP Tool Cheat Sheet - Quick Reference

## ⚡ Most Used Tools

### 1. Get My Tasks
```typescript
mcp__hyper__coordinator_list_agent_tasks({
  agentName: "YOUR_AGENT_NAME"  // Exact match required
})
```

### 2. Start Working on Task
```typescript
mcp__hyper__coordinator_update_task_status({
  taskId: "AGENT_TASK_UUID",
  status: "in_progress",
  notes: "Started implementation"
})
```

### 3. Update a TODO
```typescript
// Get TODO IDs first!
const tasks = await mcp__hyper__coordinator_list_agent_tasks({ agentName: "..." })
const agentTaskId = tasks.tasks[0].id
const todoId = tasks.tasks[0].todos[0].id

// Then update
mcp__hyper__coordinator_update_todo_status({
  agentTaskId: agentTaskId,  // UUID from above
  todoId: todoId,            // UUID from above
  status: "completed",
  notes: "Feature implemented"
})
```

### 4. Mark TODO In Progress
```typescript
mcp__hyper__coordinator_update_todo_status({
  agentTaskId: "AGENT_TASK_UUID",
  todoId: "TODO_UUID",
  status: "in_progress",
  notes: "Working on this now"
})
```

### 5. Complete a TODO
```typescript
mcp__hyper__coordinator_update_todo_status({
  agentTaskId: "AGENT_TASK_UUID",
  todoId: "TODO_UUID",
  status: "completed",
  notes: "Finished and tested"
})
```

### 6. Search Knowledge
```typescript
mcp__hyper__knowledge_find({
  collectionName: "technical-knowledge",
  query: "how to implement CSV export"
})
```

### 7. Store Knowledge
```typescript
mcp__hyper__knowledge_store({
  collectionName: "technical-knowledge",
  information: "Detailed implementation notes...",
  metadata: {
    taskId: "...",
    agentName: "...",
    tags: ["backend", "api"]
  }
})
```

---

## 🚨 Common Mistakes

### ❌ WRONG: Using todoIndex
```typescript
// DON'T DO THIS
mcp__hyper__coordinator_update_todo_status({
  agentTaskId: "...",
  todoIndex: 0,  // ❌ WRONG - doesn't exist
  status: "completed"
})
```

### ✅ CORRECT: Using todoId
```typescript
// Get the UUID first
const tasks = await mcp__hyper__coordinator_list_agent_tasks({ agentName: "..." })
const todoId = tasks.tasks[0].todos[0].id  // This is a UUID

// Then use it
mcp__hyper__coordinator_update_todo_status({
  agentTaskId: "...",
  todoId: todoId,  // ✅ CORRECT - UUID
  status: "completed"
})
```

---

## 📋 Parameter Quick Lookup

| Tool | Parameter 1 | Parameter 2 | Parameter 3 |
|------|-------------|-------------|-------------|
| `coordinator_list_agent_tasks` | `agentName` (string, optional) | `humanTaskId` (string, optional) | - |
| `coordinator_update_task_status` | `taskId` (string, required) | `status` (string, required) | `notes` (string, optional) |
| `coordinator_update_todo_status` | `agentTaskId` (string, required) | `todoId` (string, required) | `status` (string, required) |
| `knowledge_find` | `collectionName` (string, required) | `query` (string, required) | `limit` (number, optional) |
| `knowledge_store` | `collectionName` (string, required) | `information` (string, required) | `metadata` (object, optional) |

---

## 🎯 Status Values

### Task Status (for tasks)
- `pending` - Not started
- `in_progress` - Currently working
- `completed` - Finished
- `blocked` - Waiting on dependency

### TODO Status (for individual todos)
- `pending` - Not started
- `in_progress` - Currently working
- `completed` - Finished

⚠️ Note: TODOs cannot be `blocked` - only tasks can!

---

## 🔄 Typical Workflow

```typescript
// 1. Get your tasks
const tasks = await mcp__hyper__coordinator_list_agent_tasks({
  agentName: "Backend Services Specialist"
})

const myTask = tasks.tasks[0]
const agentTaskId = myTask.id

// 2. Mark task as in progress
await mcp__hyper__coordinator_update_task_status({
  taskId: agentTaskId,
  status: "in_progress",
  notes: "Starting work"
})

// 3. Work through each TODO
for (const todo of myTask.todos) {
  // Mark TODO as in progress
  await mcp__hyper__coordinator_update_todo_status({
    agentTaskId: agentTaskId,
    todoId: todo.id,
    status: "in_progress"
  })

  // Do the work...

  // Mark TODO as completed
  await mcp__hyper__coordinator_update_todo_status({
    agentTaskId: agentTaskId,
    todoId: todo.id,
    status: "completed",
    notes: "Implementation notes..."
  })
}

// 4. Task automatically marked completed when all TODOs done!

// 5. Store knowledge
await mcp__hyper__knowledge_store({
  collectionName: "technical-knowledge",
  information: "Learned that...",
  metadata: {
    taskId: agentTaskId,
    agentName: "Backend Services Specialist",
    completedAt: new Date().toISOString()
  }
})
```

---

## 📖 Full Documentation

For complete details, see: `HYPERION_COORDINATOR_MCP_REFERENCE.md`
