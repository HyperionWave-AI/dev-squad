# Planning Prompts - Phase 1

The Hyperion Coordinator MCP now includes two specialized prompts to help Workflow Coordinators create context-rich tasks that minimize agent exploration during implementation.

## Available Prompts

### 1. `plan_task_breakdown`

**Purpose:** Break down a complex task into detailed TODOs with embedded context hints, file paths, and function names.

**Arguments:**
- `taskDescription` (required): High-level description of what needs to be accomplished
- `targetSquad` (required): The squad/agent that will implement this task (e.g., 'go-mcp-dev', 'ui-dev', 'backend-services')

**Example Usage:**
```typescript
const prompt = await mcp__hyper__prompts_get({
  name: "plan_task_breakdown",
  arguments: {
    taskDescription: "Implement JWT authentication middleware for the API gateway",
    targetSquad: "go-mcp-dev"
  }
});
```

**Output:** A comprehensive prompt that guides you to create:
- **description** (10-20 words): Specific, actionable task
- **filePath**: Exact file location from project root
- **functionName**: Exact function/method name to create or modify
- **contextHint** (50-100 words): HOW to implement with specific patterns, functions to use, error handling, and example code

**Task Sizing Guidelines:**
- **Small:** 1-3 files, 3-5 TODOs, <30 min
- **Medium:** 3-5 files, 5-7 TODOs, <60 min
- **Large:** SPLIT IT (never >7 TODOs, never multiple services)

**Quality Checklist:**
- [ ] Every TODO has exact file path
- [ ] Every function-level TODO has function name
- [ ] Every contextHint explains HOW, not just WHAT
- [ ] Implementation agent can code immediately without searching
- [ ] No generic TODOs like "implement feature" or "add tests"
- [ ] Each TODO is independently actionable

---

### 2. `suggest_context_offload`

**Purpose:** Analyze task scope and recommend what context to embed in task fields vs what to store in Qdrant for semantic search.

**Arguments:**
- `taskScope` (required): Detailed scope of the task including requirements, constraints, and integration points
- `existingKnowledge` (optional): Comma-separated list of existing Qdrant collections or knowledge references

**Example Usage:**
```typescript
const prompt = await mcp__hyper__prompts_get({
  name: "suggest_context_offload",
  arguments: {
    taskScope: "Build a new MCP tool for task management with MongoDB storage, comprehensive error handling, and WebSocket notifications",
    existingKnowledge: "technical-knowledge,mcp-patterns,mongodb-best-practices"
  }
});
```

**Output:** A structured analysis that recommends:

#### 1. Task Field Content Recommendations
- **contextSummary** (150-250 words): Business context, technical approach, integration points, constraints, testing approach
- **filesModified**: Complete list of every file to create or modify
- **qdrantCollections** (1-3 max): Specific collections with search terms
- **notes** (50-100 words): Critical gotchas and shortcuts
- **priorWorkSummary** (100-150 words, multi-phase only): Previous agent's work summary

#### 2. Qdrant Storage Recommendations
- Collections to create/update
- Purpose and example entries
- Reusable patterns to document

#### 3. Context Efficiency Score
- Task-embedded: Target 80%+
- Qdrant-required: Target <20%

#### 4. Agent Work Estimate
- Time to read task context: Target <2 minutes
- Qdrant queries needed: Target ≤1 query
- Time to start coding: Target <2 minutes

---

## Context Distribution Framework

### 📋 Task Fields (Embed 80% of Context)
Use task fields for task-specific information that agents need immediately:

**When to Embed in Task:**
- ✅ Task-specific requirements
- ✅ Exact file locations
- ✅ Function signatures for this task
- ✅ Business logic for this feature
- ✅ Integration details for this component

**Fields to Populate:**
1. **contextSummary**: WHY, WHAT, HOW, TESTING
2. **filesModified**: EVERY file (create/modify/reference)
3. **qdrantCollections**: 1-3 specific collections if needed
4. **priorWorkSummary**: API contracts from previous phase
5. **notes**: Gotchas, non-obvious requirements, performance/security

### 🔍 Qdrant Collections (Store 20% - Reusable Knowledge)
Use Qdrant for patterns that apply across multiple tasks:

**When to Store in Qdrant:**
- ✅ Reusable technical patterns (JWT auth, error handling)
- ✅ Architectural decisions (ADRs)
- ✅ Cross-service contracts (API schemas)
- ✅ Best practices and gotchas
- ✅ Testing strategies

---

## Workflow Coordinator Usage

### Phase 1: Planning (Use These Prompts)

1. **Receive human task:**
   ```typescript
   const humanTask = await coordinator_create_human_task({
     prompt: "Original user request..."
   });
   ```

2. **Break down the task:**
   ```typescript
   const breakdownPrompt = await mcp__hyper__prompts_get({
     name: "plan_task_breakdown",
     arguments: {
       taskDescription: humanTask.prompt,
       targetSquad: "go-mcp-dev"
     }
   });

   // Use the prompt to plan detailed TODOs
   ```

3. **Determine context strategy:**
   ```typescript
   const contextPrompt = await mcp__hyper__prompts_get({
     name: "suggest_context_offload",
     arguments: {
       taskScope: humanTask.prompt,
       existingKnowledge: "technical-knowledge,mcp-patterns"
     }
   });

   // Use the analysis to populate task fields
   ```

4. **Create agent task with context:**
   ```typescript
   const agentTask = await coordinator_create_agent_task({
     humanTaskId: humanTask.id,
     agentName: "go-mcp-dev",
     role: "Implement MCP tools for task management",
     contextSummary: "...", // From context analysis
     filesModified: [...], // From breakdown
     qdrantCollections: [...], // From context analysis
     todos: [
       {
         description: "Create task storage interface",
         filePath: "coordinator/storage/tasks.go",
         functionName: "TaskStorage",
         contextHint: "Define interface with Create, Update, Get, List methods. Use MongoDB bson tags. Return *Task and error."
       }
     ]
   });
   ```

### Phase 2: Implementation (Agent Uses Context)

The implementation agent receives a task with 80%+ context embedded:

```typescript
const myTasks = await coordinator_list_agent_tasks({ agentName: "go-mcp-dev" });
const task = myTasks.tasks[0];

// Read embedded context (FREE - no queries needed)
console.log(task.contextSummary);    // WHY, WHAT, HOW, TESTING
console.log(task.filesModified);     // Exact files to modify
console.log(task.todos[0].contextHint); // HOW to implement

// Query Qdrant ONLY if task suggests it
if (task.qdrantCollections?.length > 0) {
  const pattern = await qdrant_find({
    collection_name: task.qdrantCollections[0],
    query: "specific pattern mentioned in contextHint"
  });
}

// Start coding immediately (<2 minutes)
```

---

## Benefits

### For Workflow Coordinators:
- ✅ Structured guidance for task planning
- ✅ Clear framework for context distribution
- ✅ Quality checklist to ensure completeness
- ✅ Efficiency scoring to optimize tasks

### For Implementation Agents:
- ✅ 80%+ context embedded in task (no exploration)
- ✅ Start coding within 2 minutes
- ✅ ≤1 Qdrant query needed (if any)
- ✅ Clear implementation guidance in every TODO

### For the System:
- ✅ Reduced context window usage (15% vs 80%)
- ✅ Faster task completion
- ✅ Better knowledge reuse
- ✅ Higher quality deliverables

---

## Next Steps (Phase 2 & 3)

**Phase 2:** Implementation validation prompts
- `validate_implementation_quality`: Check completed work against task requirements
- `suggest_test_coverage`: Recommend test cases based on implementation

**Phase 3:** Multi-agent coordination prompts
- `suggest_task_dependencies`: Identify cross-squad dependencies
- `recommend_parallel_work`: Suggest tasks that can run in parallel

---

## Testing

Run tests:
```bash
cd coordinator/mcp-server
go test ./handlers -v -run TestPlanning
```

Start server with prompts:
```bash
cd coordinator/mcp-server
go run main.go
```

Verify prompts registered:
```
INFO: All handlers registered successfully {"tools": 9, "resources": 5, "prompts": 2}
```

---

**Version:** v1.0 Planning Prompts
**Updated:** 2025-10-04
**Status:** ✅ Implemented and Tested
