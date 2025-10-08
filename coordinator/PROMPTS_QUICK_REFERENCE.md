# Planning Prompts - Quick Reference

## 🚀 Quick Start

### 1. Break Down Task into TODOs

```typescript
const breakdownPrompt = await mcp__hyper__prompts_get({
  name: "plan_task_breakdown",
  arguments: {
    taskDescription: "Your high-level task description",
    targetSquad: "go-mcp-dev" // or ui-dev, backend-services, etc.
  }
});

// Use the returned prompt to plan detailed TODOs
```

### 2. Determine Context Strategy

```typescript
const contextPrompt = await mcp__hyper__prompts_get({
  name: "suggest_context_offload",
  arguments: {
    taskScope: "Detailed task scope with requirements and constraints",
    existingKnowledge: "collection1,collection2,collection3" // optional, comma-separated
  }
});

// Use the analysis to populate task fields optimally
```

---

## 📋 `plan_task_breakdown`

**Purpose:** Generate detailed TODO items with embedded implementation guidance

**Required Arguments:**
| Argument | Type | Description |
|----------|------|-------------|
| `taskDescription` | string | High-level description of what needs to be accomplished |
| `targetSquad` | string | Target squad: `go-mcp-dev`, `ui-dev`, `backend-services`, etc. |

**Output Format:**
```json
{
  "todos": [
    {
      "description": "Create middleware function for JWT validation",
      "filePath": "coordinator/mcp-server/middleware/auth.go",
      "functionName": "ValidateJWT",
      "contextHint": "Extract token from Authorization header (Bearer). Use jwt.Parse() with HS256. Validate exp, iss, aud claims. Store user ID in context. Return 401 on failure."
    }
  ],
  "filesModified": [
    "coordinator/mcp-server/middleware/auth.go",
    "coordinator/mcp-server/main.go"
  ],
  "estimatedDuration": "30-45 minutes"
}
```

**Quality Checklist:**
- ✅ Each TODO has exact file path
- ✅ Function-level TODOs have function name
- ✅ Context hints explain HOW, not just WHAT
- ✅ Agent can start coding in <2 minutes

---

## 🧠 `suggest_context_offload`

**Purpose:** Optimize context distribution between task fields and Qdrant

**Required Arguments:**
| Argument | Type | Description |
|----------|------|-------------|
| `taskScope` | string | Detailed scope with requirements, constraints, integration points |
| `existingKnowledge` | string | Comma-separated Qdrant collections (optional) |

**Output Analysis:**

### 1. Task Field Recommendations
```
contextSummary: [150-250 word summary]
- Business context: WHY this task exists
- Technical approach: WHAT solution to use
- Integration points: HOW it connects
- Constraints: Performance, security limits
- Testing: Unit tests, edge cases

filesModified: [Complete file list]
- Every file to create or modify
- Mark reference files

qdrantCollections: [1-3 max]
- Specific collections with search terms
- Only if genuinely needed

notes: [50-100 words]
- Critical gotchas
- Non-obvious requirements
```

### 2. Qdrant Storage Recommendations
- Collections to create/update
- Reusable patterns to document

### 3. Efficiency Metrics
```
Task-embedded: 80%+  ← Target
Qdrant-required: <20% ← Target
Time to start coding: <2 min ← Target
```

---

## 🎯 Workflow Example

### Step 1: Create Human Task
```typescript
const humanTask = await coordinator_create_human_task({
  prompt: "Implement OAuth2 authentication with refresh tokens"
});
```

### Step 2: Plan Task Breakdown
```typescript
const breakdown = await mcp__hyper__prompts_get({
  name: "plan_task_breakdown",
  arguments: {
    taskDescription: humanTask.prompt,
    targetSquad: "go-mcp-dev"
  }
});

// AI responds with structured TODO guidance
```

### Step 3: Analyze Context Strategy
```typescript
const contextStrategy = await mcp__hyper__prompts_get({
  name: "suggest_context_offload",
  arguments: {
    taskScope: humanTask.prompt,
    existingKnowledge: "auth-patterns,oauth2-implementations"
  }
});

// AI responds with context distribution recommendations
```

### Step 4: Create Context-Rich Agent Task
```typescript
const agentTask = await coordinator_create_agent_task({
  humanTaskId: humanTask.id,
  agentName: "go-mcp-dev",
  role: "Implement OAuth2 authentication server",

  // From context strategy
  contextSummary: "...",
  filesModified: [...],
  qdrantCollections: [...],
  notes: "...",

  // From task breakdown
  todos: [
    {
      description: "...",
      filePath: "...",
      functionName: "...",
      contextHint: "..."
    }
  ]
});
```

### Step 5: Agent Implements (Fast!)
```typescript
const myTasks = await coordinator_list_agent_tasks({ agentName: "go-mcp-dev" });
const task = myTasks.tasks[0];

// Read embedded context (FREE - no queries)
console.log(task.contextSummary);
console.log(task.filesModified);
console.log(task.todos[0].contextHint);

// Query Qdrant ONLY if needed (≤1 query)
if (task.qdrantCollections?.length > 0) {
  const pattern = await qdrant_find({
    collection_name: task.qdrantCollections[0],
    query: "specific pattern from contextHint"
  });
}

// Start coding (<2 minutes elapsed)
```

---

## 🏆 Best Practices

### Task Sizing
- **Small:** 1-3 files, 3-5 TODOs, <30 min
- **Medium:** 3-5 files, 5-7 TODOs, <60 min
- **Large:** SPLIT IT (never >7 TODOs)

### Context Distribution (80/20 Rule)
**Embed in Task (80%):**
- Task-specific requirements
- Exact file locations
- Function signatures
- Business logic
- Integration details

**Store in Qdrant (20%):**
- Reusable technical patterns
- Architectural decisions
- Cross-service contracts
- Best practices
- Testing strategies

### TODO Quality
**Good TODO:**
```json
{
  "description": "Create JWT middleware with token validation",
  "filePath": "coordinator/middleware/auth.go",
  "functionName": "ValidateJWT",
  "contextHint": "Extract token from Authorization header (Bearer scheme). Use jwt.Parse() with HS256. Validate exp, iss, aud claims. Store user ID in gin.Context. Return 401 with {\"error\": \"invalid_token\"} on failure."
}
```

**Bad TODO:**
```json
{
  "description": "Add authentication",
  "filePath": "coordinator/auth.go"
}
```

---

## ⚡ Performance Benefits

### Without Prompts
- Planning: 10-15 min (exploration)
- Implementation start: 5-10 min
- **Total: 15-25 min before coding**

### With Prompts
- Planning: 3-5 min (guided)
- Implementation start: <2 min
- **Total: 3-7 min before coding (70% faster)**

### Context Window Efficiency
- **Before:** ~25,500 tokens per task
- **After:** ~5,000 tokens per task (80% reduction)

---

## 🔍 Troubleshooting

### "Arguments must be strings"
**Problem:** Trying to pass arrays or objects
**Solution:** Use comma-separated strings for lists

```typescript
// ❌ Wrong
existingKnowledge: ["collection1", "collection2"]

// ✅ Correct
existingKnowledge: "collection1,collection2"
```

### "Prompt not found"
**Problem:** Server not registered with prompts
**Solution:** Verify server logs show: `"prompts": 2`

### "Context still not embedded"
**Problem:** Not using prompt guidance
**Solution:** Follow prompt structure exactly, populate ALL recommended fields

---

## 📚 Additional Resources

- **Full Documentation:** `/coordinator/PLANNING_PROMPTS.md`
- **Implementation Details:** `/coordinator/IMPLEMENTATION_SUMMARY.md`
- **MCP Reference:** `/coordinator/HYPERION_COORDINATOR_MCP_REFERENCE.md`

---

**Quick Tip:** Use both prompts together for maximum efficiency - `plan_task_breakdown` for TODO structure + `suggest_context_offload` for optimal context distribution.

**Version:** 1.0.0 | **Updated:** 2025-10-04
