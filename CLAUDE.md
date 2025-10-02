# Hyperion Parallel Squad System – Agent Guide

> **Mission:** Deliver 15x development efficiency through autonomous domain expertise and dual-MCP coordination.

## 📚 **QUICK START**

**Essential Documents:**
1. **HYPERION_COORDINATOR_MCP_REFERENCE.md** - Complete MCP tool reference (READ FIRST)
2. **This document** - Squad coordination & workflows
3. **AI-BAND-MANAGER-SPEC.md** - Project specification

**Core MCP Tools:**
- `mcp__hyperion-coordinator__coordinator_list_agent_tasks({ agentName: "..." })` - Get assignments
- `mcp__hyperion-coordinator__coordinator_update_task_status(...)` - Update progress
- `mcp__hyperion-coordinator__coordinator_update_todo_status(...)` - Update TODOs (uses todoId UUID, not index)
- `mcp__qdrant__qdrant-find({ collection_name: "...", query: "..." })` - Search knowledge
- `mcp__qdrant__qdrant-store({ collection_name: "...", information: "...", metadata: {...} })` - Store knowledge

---

## 🚨 **CRITICAL SECURITY**

**ALL MongoDB operations MUST use user identity from JWT context. NEVER create system service identities.**

```go
// ✅ CORRECT: Extract user identity from context
identity, err := auth.GetIdentityFromContext(ctx)
secureClient, err := database.NewSecureMongoClient(&database.SecureClientOptions{
    IdentityProvider: identityProvider, // Uses user identity from context
    ...
})

// ❌ FORBIDDEN: System service identities
systemIdentity := &models.Identity{Type: "service", CompanyId: "system"}
```

**If unable to get user identity, MUST request explicit approval before proceeding.**

---

## 🎯 **Squad Structure**

**Backend Infrastructure:** Backend Services (Go microservices), Event Systems (NATS), go-mcp-dev (MCP tools), Data Platform (MongoDB/Qdrant)

**Frontend & Experience:** Frontend Experience (architecture), ui-dev (implementation), ui-tester (Playwright), AI Integration (Claude/GPT), Real-time Systems (WebSocket)

**Platform & Security:** Infrastructure Automation (GKE), Security & Auth (JWT/RBAC), Observability (metrics/monitoring)

**Cross-Squad:** Workflow Coordinator (task orchestration), End-to-End Testing (system validation)

**Golden Rules:**
- Work ONLY within your domain
- Tasks assigned via hyperion-coordinator MCP
- Knowledge shared via Qdrant MCP
- Every task uses dual-MCP workflow (coordinator=tracking, Qdrant=context)

---

## 🚨 **Context Window Management**

**Problem:** Agents exhaust context during planning, stopping mid-implementation.

**Solution - Context Budget:**
- **Planning**: <20% (5-10 min max) - Task contains 80% of needed context
- **Implementation**: 60% (actual work)
- **Documentation**: 20% (post-work)

**Rules:**
1. Task context is FREE - read task.role, task.todos, task.notes first
2. Query ONLY when insufficient - not speculatively
3. Read files to MODIFY, not to understand - Grep first
4. Start coding within 5 minutes

**Warning Signs:**
- Planning >10 min → Start implementing NOW
- Made >2 Qdrant queries → Over-researching
- Read >5 files → Exploring, not executing

**Emergency Recovery (if context exhausted mid-task):**
```typescript
// 1. Update coordinator with progress
coordinator_update_task_status({ notes: "Completed: X,Y. In progress: Z. Next: [steps]" })

// 2. Store work + decisions in coordinator knowledge
coordinator_upsert_knowledge({ text: "[accomplishments, decisions, gotchas, what's left]" })

// 3. Mark TODO with handoff notes
coordinator_update_todo_status({ notes: "Progress: X. Context exhausted. Next agent: [location]" })
```

---

## 🗂️ **Dual-MCP Workflow**

**READ FIRST:** `/HYPERION_COORDINATOR_MCP_REFERENCE.md` for complete tool reference.

**Common Mistakes:**
- ❌ Using `todoIndex` → ✅ Use `todoId` (UUID)
- ❌ Using `taskId` in TODO updates → ✅ Use `agentTaskId`
- ❌ Missing `mcp__hyperion-coordinator__` prefix
- ❌ Wrong parameter types

**Correct Pattern:**
```typescript
const myTasks = await mcp__hyperion-coordinator__coordinator_list_agent_tasks({ agentName: "..." })
const agentTaskId = myTasks.tasks[0].id
const todoId = myTasks.tasks[0].todos[0].id  // UUID, not index

await mcp__hyperion-coordinator__coordinator_update_todo_status({
  agentTaskId: agentTaskId,  // Not "taskId"
  todoId: todoId,            // UUID from listing
  status: "completed",
  notes: "..."
})
```

---

### **Context Retrieval Strategy (Priority Order)**

**Task context contains 80% of what you need. Read FIRST before any queries.**

**Priority 1: Read Task Context (FREE - 0 queries)**
```typescript
const myTask = (await coordinator_list_agent_tasks({ agentName: "..." })).tasks[0]
// Read: task.role, task.todos, task.notes, task.qdrantCollections
// Provides: requirements, files to modify, patterns to follow, constraints
```

**Priority 2: Coordinator Knowledge (1 query MAX, only if incomplete)**
```typescript
coordinator_query_knowledge({
  collection: `task:hyperion://task/human/${myTask.humanTaskId}`,
  query: "specific question about [one thing]"
})
// Use ONLY if task notes reference specific coordinator knowledge
```

**Priority 3: Qdrant Technical Patterns (1 query MAX)**
```typescript
mcp__qdrant__qdrant-find({
  collection_name: myTask.qdrantCollections?.[0] || "technical-knowledge",
  query: "specific pattern [tech] [problem] [constraint]"
})
// Use ONLY if task suggests pattern AND you don't know it
```

**Priority 4: File Reading (Read what you'll MODIFY)**
```typescript
Grep({ pattern: "functionName", path: "service/", output_mode: "files_with_matches" })
Read({ file_path: "service/handlers/export.go" })
// Grep first, read only files you'll change, max 3 files
```

**Context Budget: <4000 tokens (15%) vs OLD 23000 tokens (80%)**

---

### **Work Protocol**

**During Work:**
```typescript
// Update task status as you progress
coordinator_update_task_status({ taskId, status: "in_progress|blocked|completed", notes: "..." })

// Preserve context in TODO updates for next step
coordinator_update_todo_status({
  agentTaskId, todoId,
  notes: "Completed: X at line 45. Key decision: Y. Gotcha: Z. NEXT TODO: use pattern A"
})

// Coordinate with other squads (ONLY if needed)
qdrant-store({ collection_name: "team-coordination", information: "...", metadata: {...} })
```

**Post-Work (REQUIRED):**
```typescript
// 1. Store task-specific knowledge in coordinator
coordinator_upsert_knowledge({
  collection: `task:hyperion://task/human/${humanTaskId}`,
  text: "[solution, gotchas, testing approach]",
  metadata: { taskId, agentName, completedAt, relatedServices }
})

// 2. Share reusable knowledge in Qdrant
qdrant-store({
  collection_name: "technical-knowledge",
  information: "[detailed solution with code examples]",
  metadata: { knowledgeType, domain, title, tags, linkedTaskId }
})

// 3. Document technical debt (if found)
qdrant-store({
  collection_name: "technical-debt-registry",
  metadata: { debtType, severity, filePath, currentLineCount, squadLimit }
})

// 4. Final status update
coordinator_update_task_status({ taskId, status: "completed", notes: "..." })
```

---

## 🏗️ **Dual-MCP Architecture**

**Hyperion Coordinator MCP** (MongoDB) - Task tracking, assignments, progress, TODO management, UI visibility

**Qdrant MCP** (Vector DB) - Technical knowledge, patterns, architecture decisions, coordination, semantic search

**Why Both?**
- Separation: Tasks vs Knowledge
- Optimized: Relational (MongoDB) vs Semantic (Qdrant)
- Visibility: Real-time UI task board
- Reuse: Discover existing solutions
- Parallel: Independent squad workflows

---

## 🛠️ **MCP Tools by Squad**

**ALL AGENTS:** hyperion-coordinator + qdrant-mcp (MANDATORY)

**Backend Infrastructure:** + filesystem, github, fetch, mongodb
**Frontend & Experience:** + filesystem, github, fetch, playwright-mcp
**Platform & Security:** + kubernetes, github, filesystem, fetch
**Workflow Coordinator:** Primarily hyperion-coordinator for task orchestration, qdrant-mcp for context

---

### **🎯 Workflow Coordinator: Context-Rich Task Creation**

**Mission:** Embed 90%+ of context IN the task during planning to eliminate agent exploration during implementation.

---

### **PLANNING STAGE: Maximum Context Offloading**

**Goal:** Store ALL context so implementation agents can code immediately without research.

**Required Task Fields (MUST all be populated):**

1. **`contextSummary`** (150-250 words) - The agent's complete briefing:
   - Business context: WHY this task exists, user impact, success criteria
   - Technical approach: WHAT solution pattern to use (be specific)
   - Integration points: HOW this connects to other components
   - Constraints: Resource limits, performance targets, security requirements
   - Testing approach: Unit tests, integration tests, edge cases to cover

2. **`role`** (50-100 words) - High-level mission statement:
   - One-sentence objective
   - Primary deliverable
   - Acceptance criteria

3. **`filesModified`** (COMPLETE list) - EVERY file the agent will touch:
   - New files to create with full paths
   - Existing files to modify with full paths
   - Related files to read for context (mark as "reference only")
   - Test files to create/update

4. **`qdrantCollections`** (1-3 collections) - Where to find patterns:
   - Name specific collections with relevant examples
   - Indicate what to search for ("search for JWT middleware pattern")
   - Only include if genuinely needed

5. **`priorWorkSummary`** (100-150 words, multi-phase tasks only):
   - What previous agent accomplished
   - API contracts/interfaces established
   - Key decisions made that affect this task
   - Gotchas discovered
   - Where to find the completed code

6. **`notes`** (50-100 words) - Critical gotchas and shortcuts:
   - Common pitfalls specific to this task
   - Non-obvious requirements
   - Performance considerations
   - Security notes

---

### **TODO Item Structure (Each must have):**

**MANDATORY fields for EVERY TODO:**

1. **`description`** (10-20 words) - Specific, actionable task
   - ✅ "Create JWT middleware with token validation and error handling"
   - ❌ "Add authentication"

2. **`contextHint`** (50-100 words) - HOW to implement:
   - Code structure/pattern to follow
   - Key functions/methods to use
   - Error handling approach
   - Example code snippets if helpful
   - What to return/output

3. **`filePath`** - EXACT file location:
   - Full path from project root
   - Create new vs modify existing (be explicit)

4. **`functionName`** (if applicable) - Specific function/method:
   - Exact name to use
   - Function signature if creating new
   - Where to add if modifying existing file

**Example TODO (GOOD):**
```json
{
  "description": "Implement JWT token validation middleware",
  "filePath": "backend/middleware/auth.go",
  "functionName": "ValidateJWT",
  "contextHint": "Extract token from Authorization header (Bearer scheme). Use jwt.Parse() with HS256. Validate exp, iss, aud claims. Store user ID in gin.Context. Return 401 with {\"error\": \"invalid_token\"} on failure. See code-patterns collection for example."
}
```

**Example TODO (BAD - missing context):**
```json
{
  "description": "Add authentication",
  "filePath": "backend/auth.go"
}
```

---

### **Context Quality Checklist (Workflow Coordinator)**

Before creating a task, verify:

- [ ] `contextSummary` answers: WHY, WHAT, HOW, and TESTING
- [ ] `filesModified` is COMPLETE (agent won't need to search for files)
- [ ] `role` clearly states the deliverable
- [ ] EVERY TODO has `contextHint` with implementation guidance
- [ ] EVERY TODO has `filePath` (exact location)
- [ ] Function-level TODOs have `functionName`
- [ ] `qdrantCollections` specifies WHAT to search for
- [ ] Multi-phase tasks have `priorWorkSummary` with API contracts
- [ ] Agent can start coding in <2 minutes by reading task only

---

### **Task Sizing:**
- **Small:** 1-3 files, 3-5 TODOs, <30 min, single responsibility
- **Medium:** 3-5 files, 5-7 TODOs, <60 min, one feature/fix
- **Large:** SPLIT IT (never >7 TODOs, never multiple services, never >5 file reads)

---

### **Progressive Handoff (Multi-Phase Tasks):**

**Phase 1 Agent completes:**
1. Update all TODOs with detailed notes (what was done, line numbers, key decisions)
2. Store final state in coordinator knowledge with API contracts

**Workflow Coordinator creates Phase 2 task:**
1. Copies Phase 1 completion notes into `priorWorkSummary`
2. Specifies exact API endpoints/functions Phase 2 will call
3. Includes sample request/response if applicable
4. Lists files Phase 1 created (for reference only, don't modify)

**Result:** Phase 2 agent starts coding in <2 minutes without reading Phase 1 code

---

## ⚡ **IMPLEMENTATION STAGE: Execution Workflow**

**Goal:** Use stored context efficiently, code immediately, minimize exploration.

---

### **Phase 1: Context Discovery (2 minutes max)**

**Step 1: Retrieve Task (30 seconds)**
```typescript
const tasks = await coordinator_list_agent_tasks({ agentName: "your-name" });
const task = tasks.tasks[0]; // Your assigned task
```

**Step 2: Read ALL Context Fields (60 seconds)**

Read in this order:
1. **`contextSummary`** - Your complete briefing (WHY, WHAT, HOW, TESTING)
2. **`role`** - Your specific objective
3. **`filesModified`** - EXACT files to create/modify (no searching needed)
4. **`todos`** - Each has `description`, `contextHint`, `filePath`, `functionName`
5. **`priorWorkSummary`** - What previous agent did (if multi-phase)
6. **`qdrantCollections`** - Where to find patterns (if you need them)
7. **`notes`** - Gotchas and shortcuts

**Step 3: Validate Context Sufficiency (30 seconds)**

Ask yourself:
- [ ] Do I know WHY I'm doing this? → Check `contextSummary`
- [ ] Do I know WHAT to build? → Check `role` and TODOs
- [ ] Do I know WHERE to write code? → Check `filesModified` and `filePath`
- [ ] Do I know HOW to implement? → Check `contextHint` in each TODO
- [ ] Do I have integration details? → Check `priorWorkSummary`

**If ANY answer is NO:** STOP and ask Workflow Coordinator for clarification. DO NOT proceed with incomplete context.

**If ALL answers are YES:** START CODING immediately (skip to Phase 2).

---

### **Context Usage Rules (CRITICAL)**

**✅ DO:**
- Trust the context provided in task fields
- Use `filesModified` as your complete file list
- Use `contextHint` as your implementation guide
- Use `priorWorkSummary` for API contracts (don't read other agent's code)
- Query Qdrant ONLY for patterns mentioned in `qdrantCollections`
- Start with first TODO's `filePath` and `functionName`

**❌ DON'T:**
- Search for files (you already have `filesModified`)
- Read files speculatively (only read files you'll modify)
- Query Qdrant without checking `qdrantCollections` first
- Ignore `contextHint` and figure it out yourself
- Read previous agent's code (use `priorWorkSummary` instead)
- Spend >2 minutes planning (context already has the plan)

**🚫 FORBIDDEN:**
- Starting implementation without reading `contextSummary`
- Skipping `contextHint` in TODOs
- Searching for "similar code" when pattern is in `qdrantCollections`
- Making >2 Qdrant queries (context should have everything)
- Reading >3 files before writing first line of code

---

### **Phase 2: Implementation (80% of time)**

**Before starting:**
```typescript
coordinator_update_task_status({
  taskId: task.id,
  status: "in_progress",
  notes: "Starting implementation with context from planning phase"
});
```

**For each TODO:**

1. **Read the TODO context:**
   - `description` - What to do
   - `contextHint` - HOW to do it (use this as your guide!)
   - `filePath` - Where to write code
   - `functionName` - What to name it

2. **Implement exactly as described:**
   - Follow the pattern in `contextHint`
   - Use the exact file path and function name
   - Don't deviate unless you find a critical issue

3. **Update TODO status immediately:**
   ```typescript
   coordinator_update_todo_status({
     agentTaskId: task.id,
     todoId: todo.id,
     status: "completed",
     notes: "Implemented at lines 45-78. Used pattern from contextHint. Returns JSON as specified."
   });
   ```

4. **Store implementation details for next agent:**
   - What you actually built (line numbers, file paths)
   - Any deviations from `contextHint` (with reasons)
   - API contracts created (request/response formats)
   - Gotchas discovered

**Query Qdrant ONLY if:**
- Task explicitly mentions a pattern in `qdrantCollections`
- You need a specific example after reading `contextHint`
- **Limit: 1 query max per task**

**Example:**
```typescript
// Task says: qdrantCollections: ["jwt-middleware-patterns"]
// contextHint says: "Use jwt.Parse() with HS256. See jwt-middleware-patterns for example."

// GOOD: One targeted query
qdrant_find({
  collection_name: "jwt-middleware-patterns",
  query: "HS256 token validation error handling"
});

// BAD: Speculative exploration
qdrant_find({ collection_name: "backend-patterns", query: "authentication" });
qdrant_find({ collection_name: "go-examples", query: "middleware" });
```

---

### **Phase 3: Completion (2-5 min)**

**Step 1: Update all TODOs with implementation notes**

For EACH completed TODO:
```typescript
coordinator_update_todo_status({
  agentTaskId: task.id,
  todoId: todo.id,
  status: "completed",
  notes: "Created ValidateJWT() in backend/middleware/auth.go:15-45. Uses HS256. Returns 401 on invalid token. Stores userID in context. Tested with expired/invalid tokens."
});
```

**Step 2: Store task completion in coordinator knowledge**

```typescript
coordinator_upsert_knowledge({
  collection: `task:hyperion://task/human/${task.humanTaskId}`,
  text: `
## Implementation Summary
Agent: ${task.agentName}
Files Created: ${filesCreated}
API Contracts: ${apiContracts}
Key Decisions: ${decisions}
Gotchas: ${gotchas}
Next Agent Should Know: ${handoffInfo}
  `,
  metadata: {
    taskId: task.id,
    agentName: task.agentName,
    completedAt: new Date().toISOString()
  }
});
```

**Step 3: Store reusable patterns in Qdrant (if created new pattern)**

Only if you created a NEW pattern not in `qdrantCollections`:
```typescript
qdrant_store({
  collection_name: "technical-knowledge",
  information: "Detailed implementation of ${pattern} with code examples",
  metadata: {
    knowledgeType: "pattern",
    domain: "backend",
    tags: ["jwt", "middleware", "authentication"]
  }
});
```

**Step 4: Mark task complete**

```typescript
coordinator_update_task_status({
  taskId: task.id,
  status: "completed",
  notes: "All TODOs completed. Created JWT middleware at backend/middleware/auth.go. Exports ValidateJWT() function. See task knowledge for API contract and testing details."
});
```

---

### **Context Efficiency Metrics (Self-Check)**

After completing a task, verify you used context efficiently:

- [ ] Started coding within 2 minutes of reading task
- [ ] Made ≤1 Qdrant query (only if `qdrantCollections` specified)
- [ ] Read ≤3 files (only files in `filesModified`)
- [ ] Used `contextHint` for every TODO (didn't reinvent approach)
- [ ] Used `priorWorkSummary` instead of reading other agent's code
- [ ] Updated all TODOs with implementation notes
- [ ] Stored completion summary for next agent

**If any checkbox is unchecked:** Review why and improve next time.

---

## 🚨 **Engineering Standards**

**Fail-Fast:** Never silent fallbacks. Return real errors with context.
```go
// ✅ return "", fmt.Errorf("server URL not found for %s", serverName)
// ❌ return fmt.Sprintf("http://%s:8080/mcp", serverName) // Hides problem
```

**MCP Compliance:** Official Go SDK only, tool names=snake_case, params/JSON=camelCase

**Security:** JWT required for ALL endpoints, use `./scripts/generate-test-jwt.js`, never log secrets

**JSON Naming (MANDATORY):** ALL JSON/URL params MUST be camelCase (frontend contract)
```go
// ✅ json:"userId", c.Query("userId"), /api/v1/ws?userId=123
// ❌ json:"user_id", c.Query("user_id"), /api/v1/ws?user_id=123
```

**Code Quality:**
- Go 1.25 only, Handlers ≤300 lines, Services ≤400, main.go ≤200
- CLAUDE.md required per package before merge

**DRY/SOLID/YAGNI:** No code duplication, single responsibility, interfaces for extensibility, inject dependencies, no speculative features

**Quality Gates by Squad:**
- **Backend:** Handlers ≤300, Services ≤400, main.go ≤200, complexity ≤10/function, >500 lines=REFACTOR NOW
- **Frontend Experience:** Patterns ≤250 lines, 80% component reuse, ≤5 props, doc all patterns
- **ui-dev:** Components ≤250, Hooks ≤150, API clients ≤300, zero duplicate logic, **optimistic UI for task board**
- **ui-tester:** ≥80% coverage, WCAG 2.1 AA, ≥95% non-flaky, ≤300 lines/suite, ≤5 min runtime
- **Platform:** Zero hardcoded values, K8s ≤200, Security ≤300, Deployment ≤250

**Refactoring:** 72-hour rule for oversized files, god files block squad merges, refactoring gets sprint priority

---

## 🎯 **Coordination Patterns**

**DO:** Query Qdrant first, document immediately, design for parallel work, update status frequently
**DON'T:** Work outside domain, skip protocols, create hidden dependencies, bypass Qdrant, ignore existing knowledge

**Cross-Squad:** API changes/security updates/performance issues posted to team-coordination, relevant squads discover and handle

**UI Workflows:**
- **Feature:** Frontend Experience (architecture) || Backend (API) → ui-dev (implementation) → ui-tester (validation)
- **Bug Fix:** ui-dev (fix) || ui-tester (regression test)
- **Test Coverage:** ui-tester queries Qdrant → implement tests → document strategies

**Handoffs via Qdrant Collections:** ui-component-patterns, ui-test-strategies, ui-accessibility-standards, team-coordination

**Technical Debt:** Monthly review of technical-debt-registry, 20% sprint allocation, cross-squad debt gets priority

---

## 📊 **Success Metrics**

**Platform Targets:** 15x efficiency, 90% knowledge reuse, <5% conflicts, <24h delivery
**Code Quality:** 90% DRY, 2:1 debt reduction ratio, 95% SOLID adherence, zero god files
**Agent:** Fast context discovery, high-quality contributions, minimal blocking
**Squad:** High parallel ratio, frequent knowledge reuse, low conflicts, fast velocity

---

## 🚀 **Deployment**

**GCP Production:** project=production-471918, cluster=hyperion-production (europe-west2), namespace=hyperion-prod
**Coordination:** Infrastructure squad manages GKE via GitHub Actions, other squads request via team-coordination
**NEVER:** Run kubectl directly on production - use GitHub Actions only

---

## 📚 **Qdrant Collections**

**Task:** task:hyperion://task/human/{taskId}, team-coordination, agent-coordination
**Tech:** technical-knowledge, code-patterns, adr, data-contracts, technical-debt-registry
**UI:** ui-component-patterns, ui-test-strategies, ui-accessibility-standards, ui-visual-regression-baseline
**Ops:** mcp-operations, code-quality-violations

**Documentation Templates:** Problem, Solution, Implementation, Testing, Dependencies, Performance (see existing Qdrant entries for examples)

---

## 🔄 **Emergencies**

**Blocked >2h:** Post to team-coordination → Query Qdrant → Coordinate → Escalate after 4h
**Conflict:** Document in team-coordination → Search solutions → Propose fixes → Escalate after 2h
**Qdrant Failure:** Use local CLAUDE.md → Document offline → Alert squads → Resume when restored
**Production:** Infrastructure leads → Squads provide expertise → Document response → Post-incident review

## 📝 **Daily Operations**

**Start:** Get tasks, review team-coordination, plan parallel work
**During:** Use MCP tools, update status (30-60 min for long tasks), document immediately
**End:** Store knowledge (coordinator + Qdrant), update status, flag oversized files
**Emergency:** Post to team-coordination → Search solutions → Coordinate → Escalate last

---

**Version:** v1.2 Context-in-Tasks Architecture | **Updated:** 2025-10-01
**Mantra:** *Context First, Domain Focus, Parallel Always, Knowledge Shared, Quality Enforced*