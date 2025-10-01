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

**Mission:** Embed 80% of context IN the task to minimize agent exploration queries.

**Task Structure:**
1. **`role`** - Business context + what to build + constraints (100-200 words)
2. **`notes`** - Pattern to follow + files to modify + key decisions + gotchas
3. **`todos`** - Each with file/function/contextHint in notes
4. **`qdrantCollections`** - 1-2 collections IF patterns needed

**Example:** See full template in HYPERION_COORDINATOR_MCP_REFERENCE.md

**Checklist (embed ALL in task):**
- Business context, pattern reference, file paths, function names
- Technical constraints, key decisions, gotchas, test requirements

**Task Sizing:**
- **Small:** 1-3 files, 3-5 TODOs, <30 min
- **Medium:** 3-5 files, 5-7 TODOs, <60 min
- **Large:** SPLIT IT (never >7 TODOs, never multiple services, never >5 file reads)

**Progressive Handoff:** When Agent 2 depends on Agent 1:
1. Agent 1 completes with detailed notes (completed work, key decisions, API contract)
2. Workflow Coordinator creates Agent 2 task WITH Agent 1's context embedded in notes
3. Result: Agent 2 starts coding in <2 minutes without reading Agent 1's code

---

## ⚡ **Execution Workflow**

**Phase 1: Discovery (2-5 min max)**
1. Get task: `coordinator_list_agent_tasks({ agentName })`
2. Read task.role, task.todos, task.notes (80% of context)
3. Grep to locate files (don't read yet)
4. Check blockers ONLY if task mentions dependencies
5. START CODING within 5 minutes

**Phase 2: Implementation**
- Update status to "in_progress"
- Work in your domain with your MCP tools
- Update coordinator on blockers immediately
- Post Qdrant updates every 30-60 min (long tasks only)

**Phase 3: Completion (2-5 min)**
- Store knowledge in coordinator + Qdrant
- Mark task "completed"
- Document for future agent searchability

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