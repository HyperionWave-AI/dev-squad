# Hyperion Parallel Squad System – Agent Guide

> **Mission:** Deliver 15x development efficiency through autonomous domain expertise and dual-MCP coordination.

## 🚨 **CRITICAL RULES - MUST FOLLOW**

### **RULE 1: MANDATORY HYPERION MCP USAGE**
**ALL work MUST use Hyperion Coordinator MCP. Direct work is FORBIDDEN.**

❌ **FORBIDDEN:**
- Writing code directly in main conversation
- Making changes without creating tasks
- Bypassing task assignment system
- Working without agent coordination

✅ **REQUIRED:**
- Create human task via `mcp__hyper__coordinator_create_human_task`
- Create agent task via `mcp__hyper__coordinator_create_agent_task`
- Assign to appropriate specialist agent
- Track progress via coordinator

### **RULE 2: ALWAYS USE SUB-AGENTS - NEVER WORK DIRECTLY**
**You are a COORDINATOR, not an implementer. Delegate ALL work to specialist agents.**

❌ **FORBIDDEN:**
- Implementing code yourself
- Making file changes directly
- Running builds/tests yourself
- Writing documentation directly

✅ **REQUIRED:**
- Use `Task` tool to launch specialist agents
- Assign work based on agent expertise
- Monitor agent progress via coordinator
- Coordinate handoffs between agents

**Exception:** ONLY coordinator operations (creating tasks, updating status) allowed directly.

---

## 📚 **QUICK START**

**Essential Documents:**
1. **HYPERION_COORDINATOR_MCP_REFERENCE.md** - Complete MCP tool reference (READ FIRST)
2. **This document** - Squad coordination & workflows
3. **AI-BAND-MANAGER-SPEC.md** - Project specification

**Core MCP Tools (7 tools) - MANDATORY USAGE:**
- `mcp__hyper__coordinator_create_human_task({ prompt: "..." })` - **START HERE** - Create user request
- `mcp__hyper__coordinator_create_agent_task({ ... })` - **REQUIRED** - Assign to specialist
- `mcp__hyper__coordinator_list_agent_tasks({ agentName: "..." })` - Get assignments
- `mcp__hyper__coordinator_update_task_status(...)` - Update progress
- `mcp__hyper__coordinator_update_todo_status(...)` - Update TODOs (uses todoId UUID, not index)
- `mcp__hyper__coordinator_upsert_knowledge(...)` - Store task knowledge in coordinator
- `mcp__hyper__coordinator_query_knowledge(...)` - Query task context from coordinator

**Sub-Agent Delegation (MANDATORY):**
- `Task` tool with `subagent_type` parameter - **REQUIRED for ALL implementation work**
- Available agents: go-dev, ui-dev, ui-tester, k8s-deployment-expert, sre, etc.
- Launch agents for: code changes, builds, tests, deployments, documentation

**MCP Resources (12 resources) - NEW!:**

**Documentation Resources (instant access, no queries):**
- `mcp__hyper__resources_read({ uri: "hyperion://docs/standards" })` - Engineering standards
- `mcp__hyper__resources_read({ uri: "hyperion://docs/architecture" })` - System architecture
- `mcp__hyper__resources_read({ uri: "hyperion://docs/squad-guide" })` - Squad coordination guide

**Workflow Resources (real-time status):**
- `mcp__hyper__resources_read({ uri: "hyperion://workflow/active-agents" })` - Who's working on what
- `mcp__hyper__resources_read({ uri: "hyperion://workflow/task-queue" })` - Pending tasks
- `mcp__hyper__resources_read({ uri: "hyperion://workflow/dependencies" })` - Task dependencies

**Knowledge Resources (discovery):**
- `mcp__hyper__resources_read({ uri: "hyperion://knowledge/collections" })` - Available knowledge collections
- `mcp__hyper__resources_read({ uri: "hyperion://knowledge/recent-learnings" })` - Last 24h knowledge from coordinator

**Metrics Resources (performance):**
- `mcp__hyper__resources_read({ uri: "hyperion://metrics/squad-velocity" })` - Completion rates
- `mcp__hyper__resources_read({ uri: "hyperion://metrics/context-efficiency" })` - Efficiency stats

**MCP Prompts (7 prompts) - AI Assistance! NEW!:**

**For Workflow Coordinators:**
- `mcp__hyper__prompts_get({ name: "plan_task_breakdown", arguments: {...} })` - Break down tasks
- `mcp__hyper__prompts_get({ name: "detect_cross_squad_impact", arguments: {...} })` - Impact analysis
- `mcp__hyper__prompts_get({ name: "suggest_handoff_strategy", arguments: {...} })` - Handoff planning

**For Implementation Agents:**
- `mcp__hyper__prompts_get({ name: "recommend_knowledge_query", arguments: {...} })` - Optimize knowledge queries
- `mcp__hyper__prompts_get({ name: "diagnose_blocked_task", arguments: {...} })` - Unblock help
- `mcp__hyper__prompts_get({ name: "guide_knowledge_storage", arguments: {...} })` - **MANDATORY** - Get storage guidelines before storing knowledge
- `mcp__hyper__prompts_get({ name: "suggest_knowledge_structure", arguments: {...} })` - Structure learnings for storage

---

## 🚨 **WORKFLOW ENFORCEMENT**

### **MANDATORY WORKFLOW FOR ALL USER REQUESTS**

**Step 1: Create Human Task (REQUIRED)**
```typescript
const humanTask = await mcp__hyper__coordinator_create_human_task({
  prompt: "[User's original request verbatim]"
})
// Returns: { taskId: "uuid", status: "pending" }
```

**Step 2: Create Agent Task with Context (REQUIRED)**
```typescript
const agentTask = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTask.taskId,
  agentName: "[Specialist Agent Name]", // See squad list below
  role: "[What agent will accomplish]",
  contextSummary: "[WHY/WHAT/HOW/CONSTRAINTS/TESTING - 150-250 words]",
  filesModified: ["exact/file/paths.ts"],
  knowledgeCollections: ["relevant-collection"], // Optional - coordinator knowledge collections
  todos: [
    {
      description: "Specific task",
      filePath: "exact/path.ts",
      functionName: "FunctionName",
      contextHint: "50-100 word implementation guide"
    }
  ]
})
```

**Step 3: Launch Sub-Agent (REQUIRED for implementation)**
```typescript
// For code/build/test/deploy work - MANDATORY
await Task({
  subagent_type: "go-dev", // or ui-dev, ui-tester, sre, etc.
  description: "Brief task description",
  prompt: `
    You have been assigned agent task: ${agentTask.taskId}

    Retrieve your assignment:
    - Use mcp__hyper__coordinator_list_agent_tasks({ agentName: "[Agent Name]" })
    - Read task.contextSummary, task.filesModified, task.todos
    - Each TODO has contextHint with implementation guidance

    Complete the work:
    - Update status: mcp__hyper__coordinator_update_task_status()
    - Mark TODOs complete: mcp__hyper__coordinator_update_todo_status()
    - Store knowledge: mcp__hyper__coordinator_upsert_knowledge()

    Context is embedded in the task - start coding within 2 minutes.
  `
})
```

**Step 4: Monitor & Coordinate (Your Role)**
- Check agent progress via `coordinator_list_agent_tasks`
- Handle blockers via `coordinator_update_task_status({ status: "blocked" })`
- Coordinate handoffs between agents
- Update human task status when complete

**YOU ARE A COORDINATOR - NEVER THE IMPLEMENTER**

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

## 🎯 **Squad Structure & Agent Assignment**

### **Available Sub-Agents (Use Task tool with subagent_type)**

**Backend Infrastructure:**
- `go-dev` - Go microservices, REST APIs, business logic
- `go-mcp-dev` - MCP tools and integrations (Model Context Protocol)
- `Backend Services Specialist` - Service architecture (via coordinator)
- `Event Systems Specialist` - NATS JetStream (via coordinator)
- `Data Platform Specialist` - MongoDB optimization and data modeling (via coordinator)

**Frontend & Experience:**
- `ui-dev` - React/TypeScript implementation, components
- `ui-tester` - Playwright E2E tests, accessibility validation
- `Frontend Experience Specialist` - UI architecture (via coordinator)
- `AI Integration Specialist` - Claude/GPT integration (via coordinator)
- `Real-time Systems Specialist` - WebSocket, streaming (via coordinator)

**Platform & Operations:**
- `sre` - Deployment to dev/prod environments
- `k8s-deployment-expert` - Kubernetes manifests, rollouts, scaling
- `Infrastructure Automation Specialist` - GKE, GitHub Actions (via coordinator)
- `Security & Auth Specialist` - JWT, RBAC, security policies (via coordinator)
- `Observability Specialist` - Metrics, monitoring (via coordinator)

**Testing & Quality:**
- `ui-tester` - Frontend E2E tests
- `End-to-End Testing Coordinator` - Cross-squad testing (via coordinator)

### **Assignment Rules (CRITICAL)**

**Use Task tool for:**
- All code implementation (go-dev, ui-dev)
- All testing (ui-tester)
- All deployments (sre, k8s-deployment-expert)
- All MCP development (go-mcp-dev)

**Use coordinator_create_agent_task for:**
- Architecture design (Frontend Experience, Backend Services)
- Cross-squad coordination (Event Systems, AI Integration)
- Security reviews (Security & Auth Specialist)
- Planning and design (all *Specialist agents)

**Golden Rules:**
- **NEVER implement directly** - always delegate to agents
- Work ONLY within your coordinator role
- Tasks assigned via hyper MCP (MANDATORY)
- Knowledge shared via coordinator MCP
- Every task uses coordinator MCP (tracking + knowledge storage)

---

## 🚨 **Context Window Management**

**Problem:** Agents exhaust context during planning, stopping mid-implementation.

**Solution - Context Budget:**
- **Coordinator (YOU)**: <10% (task creation only) - Embed context IN tasks
- **Sub-Agent Planning**: <20% (5-10 min max) - Task contains 80% of context
- **Sub-Agent Implementation**: 60% (actual work)
- **Sub-Agent Documentation**: 20% (post-work)

**YOUR Role (Coordinator):**
- Create context-rich tasks (5-10 min)
- Launch sub-agents via Task tool
- Monitor progress via coordinator MCP
- **NEVER implement yourself** - wastes your context window

**Rules for Sub-Agents (include in Task prompt):**
1. Task context is FREE - read task.role, task.todos, task.notes first
2. Query ONLY when insufficient - not speculatively
3. Read files to MODIFY, not to understand - Grep first
4. Start coding within 5 minutes

**Warning Signs (Sub-Agent behavior):**
- Planning >10 min → Agent over-researching (improve task context)
- Made >2 knowledge queries → Insufficient context in task
- Read >5 files → Exploring, not executing (add filesModified to task)

**YOUR Warning Signs (Coordinator):**
- Writing code directly → STOP - launch sub-agent instead
- Making file changes → FORBIDDEN - use Task tool
- Implementing features → Delegate to specialist agent
- Skipping task creation → MANDATORY - always use MCP workflow

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
- ❌ Missing `mcp__hyper__` prefix
- ❌ Wrong parameter types

**Correct Pattern:**
```typescript
const myTasks = await mcp__hyper__coordinator_list_agent_tasks({ agentName: "..." })
const agentTaskId = myTasks.tasks[0].id
const todoId = myTasks.tasks[0].todos[0].id  // UUID, not index

await mcp__hyper__coordinator_update_todo_status({
  agentTaskId: agentTaskId,  // Not "taskId"
  todoId: todoId,            // UUID from listing
  status: "completed",
  notes: "..."
})
```

---

### **Context Retrieval Strategy (Priority Order)**

**Task context contains 80% of what you need. Resources are FREE. Read FIRST before any queries.**

**Priority 0: MCP Resources (FREE - instant access, no context cost)**
```typescript
// Check what others are working on (avoid duplicate work)
const activeAgents = await mcp__hyper__resources_read({
  uri: "hyperion://workflow/active-agents"
})

// Get engineering standards (quality gates, file size limits)
const standards = await mcp__hyper__resources_read({
  uri: "hyperion://docs/standards"
})

// See what was learned recently (check before implementing)
const recentLearnings = await mcp__hyper__resources_read({
  uri: "hyperion://knowledge/recent-learnings"
})

// Find which knowledge collections are available
const collections = await mcp__hyper__resources_read({
  uri: "hyperion://knowledge/collections"
})
```

**Priority 1: Read Task Context (FREE - 0 queries)**
```typescript
const myTask = (await coordinator_list_agent_tasks({ agentName: "..." })).tasks[0]
// Read: task.role, task.todos, task.notes, task.knowledgeCollections, task.filesModified
// Provides: requirements, files to modify, patterns to follow, constraints
```

**Priority 2: Use MCP Prompts for Guidance (FREE - AI assistance)**
```typescript
// If you need help with knowledge queries
const queryHelp = await mcp__hyper__prompts_get({
  name: "recommend_knowledge_query",
  arguments: {
    agentQuestion: "How to implement JWT validation?",
    taskContext: JSON.stringify(myTask),
    availableCollections: "technical-knowledge,code-patterns"
  }
})

// If you're blocked
const unblockHelp = await mcp__hyper__prompts_get({
  name: "diagnose_blocked_task",
  arguments: {
    taskId: myTask.id,
    blockReason: "Missing API documentation",
    attemptedSteps: JSON.stringify(["searched knowledge base", "read existing code"])
  }
})
```

**Priority 3: Coordinator Knowledge (1 query MAX, only if incomplete)**
```typescript
coordinator_query_knowledge({
  collection: `task:hyperion://task/human/${myTask.humanTaskId}`,
  query: "specific question about [one thing]"
})
// Use ONLY if task notes reference specific coordinator knowledge
```

**Priority 4: Knowledge Base Search (1 query MAX)**
```typescript
coordinator_query_knowledge({
  collection: myTask.knowledgeCollections?.[0] || "technical-knowledge",
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

### **Using MCP Resources & Prompts Effectively**

**🎯 When to Use Resources (FREE context):**

**BEFORE starting any task:**
```typescript
// 1. Check if someone else is already working on similar task
const activeAgents = await resources_read({ uri: "hyperion://workflow/active-agents" })
// Avoid duplicate work!

// 2. Check recent learnings (last 24h)
const recentLearnings = await resources_read({ uri: "hyperion://knowledge/recent-learnings" })
// Someone may have just solved your problem!

// 3. Review engineering standards
const standards = await resources_read({ uri: "hyperion://docs/standards" })
// Know the quality gates and file size limits
```

**IF you're blocked or inefficient:**
```typescript
// Get knowledge query suggestions
const queryHelp = await prompts_get({
  name: "recommend_knowledge_query",
  arguments: {
    agentQuestion: "your specific question",
    taskContext: JSON.stringify(myTask)
  }
})
// Follow the recommended query instead of guessing

// Diagnose why you're blocked
const unblockHelp = await prompts_get({
  name: "diagnose_blocked_task",
  arguments: {
    taskId: myTask.id,
    blockReason: "specific reason",
    attemptedSteps: JSON.stringify(["what you tried"])
  }
})
// Get specific unblocking actions
```

**AFTER completing work (MANDATORY SEQUENCE):**
```typescript
// 1. Get storage guidelines FIRST (MANDATORY)
const storageGuide = await prompts_get({
  name: "guide_knowledge_storage",
  arguments: {
    knowledgeType: "pattern|solution|gotcha|adr",
    domain: "backend|frontend|infrastructure"
  }
})
// Follow guide: Headline (max 100 words) + What + Why + How + Important

// 2. Structure your learnings per guide format
const structureHelp = await prompts_get({
  name: "suggest_knowledge_structure",
  arguments: {
    rawLearning: "what I learned",
    context: JSON.stringify({ squad: "...", files: [...] })
  }
})
// Store knowledge in the exact format from guide
```

**FOR Workflow Coordinators:**
```typescript
// Break down complex tasks
const breakdown = await prompts_get({
  name: "plan_task_breakdown",
  arguments: {
    taskDescription: "high-level task",
    targetSquad: "go-mcp-dev"
  }
})
// Create context-rich tasks with embedded guidance

// Detect cross-squad impacts
const impact = await prompts_get({
  name: "detect_cross_squad_impact",
  arguments: {
    taskDescription: "what's changing",
    filesModified: "file1.go,file2.ts"
  }
})
// Prevent conflicts before they happen

// Plan multi-phase handoffs
const handoff = await prompts_get({
  name: "suggest_handoff_strategy",
  arguments: {
    phase1Work: JSON.stringify({ completed: "..." }),
    phase2Scope: "what's next",
    knowledgeGap: "what phase2 needs"
  }
})
// Smooth handoffs in <2 minutes
```

**🚫 DON'T:**
- Skip resource checks (they're FREE context!)
- Ignore prompt suggestions (they're AI-optimized!)
- Query knowledge base blindly (use `recommend_knowledge_query` first)
- Implement without checking `recent-learnings` (reuse > rebuild)
- **Store knowledge without using `guide_knowledge_storage` prompt** (FORBIDDEN!)
- Write generic headlines or fluff content

**✅ DO:**
- Check `active-agents` before starting (avoid duplicates)
- Check `recent-learnings` before coding (reuse solutions)
- Use `recommend_knowledge_query` before querying (optimize queries)
- Use `diagnose_blocked_task` when stuck (unblock faster)
- **Use `guide_knowledge_storage` BEFORE storing any knowledge** (MANDATORY!)
- Use prompts for guidance (they know the patterns)

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
coordinator_upsert_knowledge({ collection_name: "team-coordination", information: "...", metadata: {...} })
```

**Post-Work (REQUIRED):**
```typescript
// 0. Get knowledge storage guidelines (MANDATORY BEFORE STORING)
const storageGuide = await mcp__hyper__prompts_get({
  name: "guide_knowledge_storage",
  arguments: {
    knowledgeType: "pattern|solution|gotcha|adr",
    domain: "backend|frontend|infrastructure"
  }
})
// Follow the guide: Headline (max 100 words), What, Why, How, Important
// NO FLUFF - only what matters for future AI agents

// 1. Store task-specific knowledge in coordinator
coordinator_upsert_knowledge({
  collection: `task:hyperion://task/human/${humanTaskId}`,
  text: `
## [Headline - WHAT + WHY in <100 words]

### What
[Core technical details with code]

### Why
[Business/technical importance]

### How
[Step-by-step implementation]

### Important
[Critical gotchas with Why/Solution/Detection]
  `,
  metadata: { taskId, agentName, completedAt, relatedServices }
})

// 2. Share reusable knowledge in knowledge base (FOLLOW STORAGE GUIDE)
coordinator_upsert_knowledge({
  collection_name: "technical-knowledge",
  information: "[Use format from guide_knowledge_storage prompt]",
  metadata: {
    knowledgeType: "pattern|solution|gotcha|adr",
    domain: "backend|frontend|infrastructure",
    title: "[Searchable title from headline]",
    tags: ["tag1", "tag2", "tag3", "tag4", "tag5"] // 5-8 specific tags
  }
})

// 3. Document technical debt (if found)
coordinator_upsert_knowledge({
  collection_name: "technical-debt-registry",
  metadata: { debtType, severity, filePath, currentLineCount, squadLimit }
})

// 4. Final status update
coordinator_update_task_status({ taskId, status: "completed", notes: "..." })
```

---

## 🚨 **KNOWLEDGE STORAGE ENFORCEMENT (MANDATORY)**

**CRITICAL RULE: NEVER store knowledge without following the storage guide format.**

### **Before Storing ANY Knowledge (REQUIRED)**

**Step 1: Retrieve the Storage Guide**
```typescript
const storageGuide = await mcp__hyper__prompts_get({
  name: "guide_knowledge_storage",
  arguments: {
    knowledgeType: "pattern|solution|gotcha|adr",
    domain: "backend|frontend|infrastructure"
  }
})
// This returns the EXACT format to follow
```

**Step 2: Follow the Format EXACTLY**

**EVERY knowledge entry MUST have:**

1. **Headline (Max 100 words)** - WHAT + WHY
   - ✅ "JWT HS256 Middleware for Go Gin - Validates Bearer tokens with 5ms latency. Critical for authenticated endpoints."
   - ❌ "Authentication middleware" (too generic)

2. **What** - Core technical details with tested code
   - NO fluff, NO filler words
   - Working code examples with comments
   - File locations, function names, exact details

3. **Why** - Business/technical importance
   - What problem does this solve?
   - Why this approach over alternatives?
   - When should agents use this?

4. **How** - Step-by-step implementation
   - Prerequisites (dependencies, env vars)
   - Integration steps
   - Testing approach

5. **Important** - Critical gotchas (≥2 required)
   - Format: ⚠️ **Gotcha:** + Why + Solution + Detection
   - Non-obvious issues that cause failures
   - Edge cases discovered

6. **Metadata** - 5-8 specific searchable tags
   - Technology + Domain + Pattern + Problem
   - ✅ ["go", "jwt", "middleware", "authentication", "hs256"]
   - ❌ ["backend", "code", "implementation"]

### **Quality Checklist (Before Storing)**

Verify EVERY knowledge entry has:
- [ ] Headline <100 words answering WHAT + WHY
- [ ] What section with tested code (NO FLUFF)
- [ ] Why section explaining value/importance
- [ ] How section with step-by-step implementation
- [ ] Important section with ≥2 gotchas (Why/Solution/Detection format)
- [ ] 5-8 specific, searchable tags
- [ ] Zero filler words ("This section explains...", "As you can see...")
- [ ] All code is tested and working

### **Enforcement Rules**

**✅ ALLOWED:**
- Storing knowledge that follows guide format
- Concise, actionable content
- Tested code examples
- Non-obvious gotchas with solutions

**❌ FORBIDDEN:**
- Storing knowledge without retrieving guide first
- Generic headlines that don't answer WHAT + WHY
- Filler content ("This explains...", "It's important to note...")
- Untested code
- Missing gotchas section
- Vague tags (["backend", "code"])
- Fluff or padding content

**💡 Remember:** Future AI agents rely on this knowledge. Every word matters. Make it count.

---

## 🏗️ **Dual-MCP Architecture**

**Hyperion Coordinator MCP** (MongoDB) - Task tracking, assignments, progress, TODO management, UI visibility

**Coordinator Knowledge** (Vector DB) - Technical knowledge, patterns, architecture decisions, coordination, semantic search

**Why Both?**
- Separation: Tasks vs Knowledge
- Optimized: Relational (MongoDB) vs Semantic (knowledge base)
- Visibility: Real-time UI task board
- Reuse: Discover existing solutions
- Parallel: Independent squad workflows

---

## 🛠️ **MCP Tools by Squad**

**ALL AGENTS:** hyper + coordinator knowledge (MANDATORY)

**Backend Infrastructure:** + filesystem, github, fetch, mongodb
**Frontend & Experience:** + filesystem, github, fetch, playwright-mcp
**Platform & Security:** + kubernetes, github, filesystem, fetch
**Workflow Coordinator:** Primarily hyper for task orchestration, coordinator knowledge for context

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

4. **`knowledgeCollections`** (1-3 collections) - Where to find patterns:
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
- [ ] `knowledgeCollections` specifies WHAT to search for
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
6. **`knowledgeCollections`** - Where to find patterns (if you need them)
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
- Query knowledge base ONLY for patterns mentioned in `knowledgeCollections`
- Start with first TODO's `filePath` and `functionName`

**❌ DON'T:**
- Search for files (you already have `filesModified`)
- Read files speculatively (only read files you'll modify)
- Query knowledge base without checking `knowledgeCollections` first
- Ignore `contextHint` and figure it out yourself
- Read previous agent's code (use `priorWorkSummary` instead)
- Spend >2 minutes planning (context already has the plan)

**🚫 FORBIDDEN:**
- Starting implementation without reading `contextSummary`
- Skipping `contextHint` in TODOs
- Searching for "similar code" when pattern is in `knowledgeCollections`
- Making >2 knowledge base queries (context should have everything)
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

**Query knowledge base ONLY if:**
- Task explicitly mentions a pattern in `knowledgeCollections`
- You need a specific example after reading `contextHint`
- **Limit: 1 query max per task**

**Example:**
```typescript
// Task says: knowledgeCollections: ["jwt-middleware-patterns"]
// contextHint says: "Use jwt.Parse() with HS256. See jwt-middleware-patterns for example."

// GOOD: One targeted query
coordinator_query_knowledge({
  collection_name: "jwt-middleware-patterns",
  query: "HS256 token validation error handling"
});

// BAD: Speculative exploration
coordinator_query_knowledge({ collection_name: "backend-patterns", query: "authentication" });
coordinator_query_knowledge({ collection_name: "go-examples", query: "middleware" });
```

---

### **Phase 3: Completion (2-5 min)**

**Step 0: Get Knowledge Storage Guidelines (MANDATORY)**

**BEFORE storing ANY knowledge, retrieve the storage guide:**
```typescript
const storageGuide = await mcp__hyper__prompts_get({
  name: "guide_knowledge_storage",
  arguments: {
    knowledgeType: "pattern", // or "solution", "gotcha", "adr"
    domain: "backend"          // or "frontend", "infrastructure"
  }
})
// Read the guide output - it provides the EXACT structure to follow
// Headline (max 100 words) + What + Why + How + Important
// NO FLUFF - only what matters for AI agents
```

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

**Step 2: Store task completion in coordinator knowledge (FOLLOW GUIDE FORMAT)**

```typescript
coordinator_upsert_knowledge({
  collection: `task:hyperion://task/human/${task.humanTaskId}`,
  text: `
## JWT HS256 Middleware for Go Gin - Validates Bearer tokens in Authorization headers. Uses jwt.Parse() with 5ms latency. Critical for authenticated endpoints. Handles exp/iss/aud claims, stores user context. [<100 words HEADLINE]

### What
[Core implementation with code examples - NO FLUFF]
` + "```go\n// Working, tested code\n```" + `

### Why
[Business/technical value - why this approach was chosen]

### How
[Prerequisites, integration steps, testing approach]

### Important
⚠️ **Gotcha:** [What can go wrong]
   - **Why:** [Root cause]
   - **Solution:** [How to fix]
   - **Detection:** [How to recognize]
  `,
  metadata: {
    taskId: task.id,
    agentName: task.agentName,
    completedAt: new Date().toISOString(),
    knowledgeType: "solution",
    tags: ["specific", "searchable", "tags"]
  }
});
```

**Step 3: Store reusable patterns in knowledge base (MANDATORY GUIDE FORMAT)**

Only if you created a NEW pattern not in `knowledgeCollections`:
```typescript
// CRITICAL: Follow guide_knowledge_storage format EXACTLY
coordinator_upsert_knowledge({
  collection_name: "technical-knowledge",
  information: `
## [Headline - WHAT + WHY in <100 words - make it count!]

### What
[Core technical details with tested code examples]

### Why
[Business/technical importance - when to use this]

### How
[Step-by-step implementation with prerequisites]

### Important
⚠️ **Gotcha:** [Non-obvious issue]
   - **Why:** [Root cause]
   - **Solution:** [How to fix]
   - **Detection:** [How to recognize]
  `,
  metadata: {
    knowledgeType: "pattern",
    domain: "backend",
    title: "[Searchable title matching headline]",
    tags: ["tag1", "tag2", "tag3", "tag4", "tag5"], // 5-8 specific tags
    linkedTaskId: task.id
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
- [ ] Made ≤1 knowledge query (only if `knowledgeCollections` specified)
- [ ] Read ≤3 files (only files in `filesModified`)
- [ ] Used `contextHint` for every TODO (didn't reinvent approach)
- [ ] Used `priorWorkSummary` instead of reading other agent's code
- [ ] Updated all TODOs with implementation notes
- [ ] **Retrieved `guide_knowledge_storage` prompt BEFORE storing knowledge** (MANDATORY!)
- [ ] **Followed guide format: Headline (max 100 words) + What + Why + How + Important**
- [ ] **Used 5-8 specific, searchable tags (not generic)**
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

**DO:** Query knowledge base first, document immediately, design for parallel work, update status frequently
**DON'T:** Work outside domain, skip protocols, create hidden dependencies, bypass knowledge base, ignore existing knowledge

**Cross-Squad:** API changes/security updates/performance issues posted to team-coordination, relevant squads discover and handle

**UI Workflows:**
- **Feature:** Frontend Experience (architecture) || Backend (API) → ui-dev (implementation) → ui-tester (validation)
- **Bug Fix:** ui-dev (fix) || ui-tester (regression test)
- **Test Coverage:** ui-tester queries knowledge base → implement tests → document strategies

**Handoffs via knowledge base Collections:** ui-component-patterns, ui-test-strategies, ui-accessibility-standards, team-coordination

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

## 📚 **knowledge base Collections**

**Task:** task:hyperion://task/human/{taskId}, team-coordination, agent-coordination
**Tech:** technical-knowledge, code-patterns, adr, data-contracts, technical-debt-registry
**UI:** ui-component-patterns, ui-test-strategies, ui-accessibility-standards, ui-visual-regression-baseline
**Ops:** mcp-operations, code-quality-violations

**Documentation Templates:** Problem, Solution, Implementation, Testing, Dependencies, Performance (see existing knowledge base entries for examples)

---

## 🔄 **Emergencies**

**Blocked >2h:** Post to team-coordination → Query knowledge base → Coordinate → Escalate after 4h
**Conflict:** Document in team-coordination → Search solutions → Propose fixes → Escalate after 2h
**knowledge base Failure:** Use local CLAUDE.md → Document offline → Alert squads → Resume when restored
**Production:** Infrastructure leads → Squads provide expertise → Document response → Post-incident review

## 📝 **Daily Operations**

**Start:** Get tasks, review team-coordination, plan parallel work
**During:** Use MCP tools, update status (30-60 min for long tasks), document immediately
**End:** Store knowledge (coordinator + knowledge base), update status, flag oversized files
**Emergency:** Post to team-coordination → Search solutions → Coordinate → Escalate last

---

## 📖 **MCP Quick Reference**

### **Resources (12 total - FREE context, instant access)**

| URI | Purpose | When to Use |
|-----|---------|-------------|
| `hyperion://docs/standards` | Engineering standards, quality gates | Before coding (know the rules) |
| `hyperion://docs/architecture` | System architecture, dual-MCP | Understanding system design |
| `hyperion://docs/squad-guide` | Squad coordination patterns | Learning workflows |
| `hyperion://workflow/active-agents` | Live agent status | Before starting (avoid duplicates) |
| `hyperion://workflow/task-queue` | Pending tasks with priority | Coordinators: task assignment |
| `hyperion://workflow/dependencies` | Task dependency graph | Understanding blocking relationships |
| `hyperion://knowledge/collections` | knowledge base collection directory | Finding where to search |
| `hyperion://knowledge/recent-learnings` | Last 24h knowledge | Before implementing (reuse!) |
| `hyperion://metrics/squad-velocity` | Task completion rates | Monitoring performance |
| `hyperion://metrics/context-efficiency` | Context usage stats | Optimizing efficiency |
| `hyperion://task/human/{id}` | Human task details | Dynamic (auto-registered) |
| `hyperion://task/agent/{agent}/{id}` | Agent task details | Dynamic (auto-registered) |

### **Prompts (7 total - AI assistance for complex decisions)**

| Prompt | Purpose | Required Arguments | Who Uses It |
|--------|---------|-------------------|-------------|
| `plan_task_breakdown` | Break down complex tasks into TODOs | `taskDescription`, `targetSquad` | Coordinators |
| `suggest_context_offload` | Determine what goes in task vs knowledge base | `taskScope`, `existingKnowledge` | Coordinators |
| `detect_cross_squad_impact` | Detect multi-squad impacts | `taskDescription`, `filesModified` | Coordinators |
| `suggest_handoff_strategy` | Plan multi-phase handoffs | `phase1Work`, `phase2Scope`, `knowledgeGap` | Coordinators |
| `recommend_knowledge_query` | Optimize knowledge base queries | `agentQuestion`, `taskContext` | Agents |
| `diagnose_blocked_task` | Unblock stuck agents | `taskId`, `blockReason`, `attemptedSteps` | Agents |
| `guide_knowledge_storage` | **MANDATORY** - Get storage format before storing | `knowledgeType`, `domain` (optional) | **ALL Agents** |
| `suggest_knowledge_structure` | Structure learnings for storage | `rawLearning`, `context` | Agents |

### **Tools (9 total - Core operations)**

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `coordinator_list_agent_tasks` | Get your assigned tasks | Start of work session |
| `coordinator_create_agent_task` | Create task for agent | Coordinators only |
| `coordinator_update_task_status` | Update task progress | During/after work |
| `coordinator_update_todo_status` | Mark TODO complete | After each TODO |
| `coordinator_upsert_knowledge` | Store task knowledge | After completing task |
| `coordinator_query_knowledge` | Query task context | Max 1 query per task |

---

**Version:** v1.5 Knowledge Storage Enforcement | **Updated:** 2025-10-12
**Capabilities:** 7 tools, 12 resources, 7 prompts | **Status:** Production Ready
**Mantra:** *Context First, Resources Free, Prompts Guide, Domain Focus, Parallel Always, Knowledge Shared, Quality Enforced*
**Key Addition:** Mandatory `guide_knowledge_storage` prompt for all knowledge storage operations
---

## 🚨 **ENFORCEMENT CHECKLIST**

### **Before Starting ANY User Request**

- [ ] Created human task via `mcp__hyper__coordinator_create_human_task`
- [ ] Created agent task via `mcp__hyper__coordinator_create_agent_task` with:
  - [ ] contextSummary (150-250 words with WHY/WHAT/HOW/CONSTRAINTS/TESTING)
  - [ ] filesModified (complete list of files to create/modify)
  - [ ] todos with contextHint (50-100 words per TODO)
  - [ ] knowledgeCollections (if patterns needed)
- [ ] Identified appropriate specialist agent (go-dev, ui-dev, sre, etc.)
- [ ] Launched sub-agent via Task tool with context-rich prompt
- [ ] **VERIFIED**: No direct implementation by coordinator

### **Forbidden Actions (Auto-Reject)**

❌ Writing code in main conversation
❌ Using Edit/Write tools directly (only sub-agents may use these)
❌ Running npm/go/docker commands yourself
❌ Making any file changes
❌ Implementing features directly
❌ Skipping task creation workflow

### **Required Actions (MANDATORY)**

✅ Create human task FIRST (always)
✅ Create agent task SECOND with full context
✅ Launch sub-agent via Task tool THIRD
✅ Monitor progress via coordinator MCP
✅ Coordinate handoffs between agents
✅ Update task status as agents progress

---

## 📊 **COORDINATOR RESPONSIBILITIES MATRIX**

| Responsibility | Coordinator (YOU) | Sub-Agent |
|----------------|-------------------|-----------|
| Create human tasks | ✅ REQUIRED | ❌ Never |
| Create agent tasks | ✅ REQUIRED | ❌ Never |
| Launch sub-agents | ✅ REQUIRED | ❌ Never |
| Write code | ❌ FORBIDDEN | ✅ Yes |
| Edit files | ❌ FORBIDDEN | ✅ Yes |
| Run builds/tests | ❌ FORBIDDEN | ✅ Yes |
| Deploy services | ❌ FORBIDDEN | ✅ Yes |
| Write docs | ❌ FORBIDDEN | ✅ Yes |
| Update task status | ✅ REQUIRED | ✅ Yes |
| Store knowledge | ✅ Can | ✅ Yes |
| Monitor progress | ✅ REQUIRED | ❌ Never |
| Coordinate handoffs | ✅ REQUIRED | ❌ Never |

---

## 🎯 **QUICK DECISION TREE**

```
User makes request
    ↓
Is this a task that requires implementation/code changes?
    ↓
YES → Use Hyperion MCP workflow:
    1. mcp__hyper__coordinator_create_human_task({ prompt })
    2. mcp__hyper__coordinator_create_agent_task({ ...context... })
    3. Task({ subagent_type: "specialist", prompt: "..." })
    4. Monitor via coordinator_list_agent_tasks
    ↓
NO → Is this a question/information request?
    ↓
    YES → Answer directly OR query knowledge base
    ↓
    NO → Is this coordination/planning?
        ↓
        YES → Create tasks, coordinate agents, monitor progress
```

**Simple Rule: If it changes files, launches sub-agent. If it requires knowledge, query MCP. If it needs coordination, use coordinator tools.**

---

## 🚀 **EXAMPLE WORKFLOWS**

### Example 1: User Requests Feature Implementation

```typescript
// ❌ WRONG - Don't do this
// You: "I'll implement the CSV export feature..."
// [Uses Edit tool directly] ← FORBIDDEN

// ✅ CORRECT - Do this instead
const humanTask = await mcp__hyper__coordinator_create_human_task({
  prompt: "Add CSV export feature to staff management API"
})

const agentTask = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTask.taskId,
  agentName: "go-dev",
  role: "Implement CSV export with streaming for large datasets",
  contextSummary: `
    User needs CSV export. PATTERN: Use io.Pipe() streaming.
    KEY DECISIONS: Stream with Flush() after each row.
    CONSTRAINTS: Company-level data isolation via JWT.
    TESTING: Test with >10K rows, verify no OOM.
  `,
  filesModified: [
    "staff-api/handlers/export_handler.go",
    "staff-api/services/export_service.go"
  ],
  todos: [{
    description: "Create streaming CSV export endpoint",
    filePath: "staff-api/handlers/export_handler.go",
    contextHint: "Use Gin c.Stream() with csv.Writer..."
  }]
})

await Task({
  subagent_type: "go-dev",
  description: "Implement CSV export",
  prompt: `
    Retrieve your task: mcp__hyper__coordinator_list_agent_tasks({ agentName: "go-dev" })
    Task ID: ${agentTask.taskId}
    All context is in the task - start coding in <2 minutes.
  `
})
```

### Example 2: User Requests UI Changes

```typescript
// ❌ WRONG
// You: "I'll update the KnowledgePage component..."
// [Uses Edit tool] ← FORBIDDEN

// ✅ CORRECT
const humanTask = await mcp__hyper__coordinator_create_human_task({
  prompt: "Add search filters to knowledge base UI"
})

const agentTask = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTask.taskId,
  agentName: "ui-dev",
  role: "Add filter chips for metadata tags in SearchResults",
  contextSummary: "...",
  filesModified: ["ui/src/components/SearchResults.tsx"],
  todos: [...]
})

await Task({
  subagent_type: "ui-dev",
  description: "Add search filters",
  prompt: `Task: ${agentTask.taskId}. Retrieve via MCP and implement.`
})
```

### Example 3: User Requests Deployment

```typescript
// ❌ WRONG
// You: "I'll deploy to production..."
// [Uses Bash kubectl commands] ← FORBIDDEN

// ✅ CORRECT
const humanTask = await mcp__hyper__coordinator_create_human_task({
  prompt: "Deploy OpenAI MCP service to production"
})

const agentTask = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTask.taskId,
  agentName: "sre",
  role: "Deploy openai-mcp service to production GKE cluster",
  contextSummary: "...",
  todos: [...]
})

await Task({
  subagent_type: "sre",
  description: "Deploy to production",
  prompt: `Deploy openai-mcp. Task: ${agentTask.taskId}`
})
```

---

## ⚠️ **VIOLATION EXAMPLES**

### ❌ VIOLATION 1: Direct Implementation
```typescript
// User: "Add a new endpoint"
// You: [Uses Edit tool to modify handler.go]
// WRONG! Must use Task tool to launch go-dev agent
```

### ❌ VIOLATION 2: Skipping Task Creation
```typescript
// User: "Fix the bug in auth middleware"
// You: [Launches Task tool without creating MCP tasks]
// WRONG! Must create human task + agent task first
```

### ❌ VIOLATION 3: Implementing in Main Conversation
```typescript
// User: "Update the React component"
// You: [Writes code in conversation]
// WRONG! Must delegate to ui-dev agent via Task tool
```

### ✅ CORRECT: Full Workflow
```typescript
// User: "Add feature X"
// You: 
// 1. Create human task
// 2. Create agent task with context
// 3. Launch specialist agent
// 4. Monitor progress
// 5. Update status when complete
```

---

## 📝 **MANDATORY WORKFLOW TEMPLATE**

**Copy this template for EVERY user request:**

```typescript
// Step 1: Create human task (REQUIRED)
const humanTask = await mcp__hyper__coordinator_create_human_task({
  prompt: "[User's request verbatim]"
})

// Step 2: Create agent task (REQUIRED)
const agentTask = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTask.taskId,
  agentName: "[go-dev|ui-dev|ui-tester|sre|etc]",
  role: "[Clear objective]",
  contextSummary: `
    WHY: [Business context]
    WHAT: [Solution approach]
    HOW: [Integration points]
    CONSTRAINTS: [Limits, security, performance]
    TESTING: [Test strategy]
  `,
  filesModified: ["exact/paths"],
  knowledgeCollections: ["relevant-collection"], // optional
  todos: [
    {
      description: "Specific task",
      filePath: "path/to/file",
      functionName: "FunctionName",
      contextHint: "Implementation guide 50-100 words"
    }
  ]
})

// Step 3: Launch sub-agent (REQUIRED)
await Task({
  subagent_type: "[agent-type]",
  description: "Brief description",
  prompt: `
    You are ${agentTask.agentName}.
    
    Retrieve task: mcp__hyper__coordinator_list_agent_tasks({ agentName: "${agentTask.agentName}" })
    Task ID: ${agentTask.taskId}
    
    Implementation context is embedded in task.contextSummary and todos[].contextHint.
    Start coding within 2 minutes.
    
    Update status: mcp__hyper__coordinator_update_task_status()
    Mark TODOs complete: mcp__hyper__coordinator_update_todo_status()
  `
})

// Step 4: Monitor and report
// Check progress via coordinator_list_agent_tasks
// Update human when complete
```

---

## 🔒 **ENFORCEMENT SUMMARY**

**YOU ARE A COORDINATOR, NOT AN IMPLEMENTER**

**Your ONLY allowed actions:**
1. Create human tasks (coordinator_create_human_task)
2. Create agent tasks (coordinator_create_agent_task)
3. Launch sub-agents (Task tool)
4. Monitor progress (coordinator_list_agent_tasks)
5. Update status (coordinator_update_task_status)
6. Store coordination knowledge (coordinator_upsert_knowledge)
7. Query knowledge base (coordinator_query_knowledge)

**FORBIDDEN actions (will be rejected):**
1. Writing code directly
2. Using Edit/Write tools
3. Running build/test commands
4. Making file changes
5. Deploying services
6. Implementing features
7. Skipping MCP task creation

**Remember: Delegate ALL implementation to specialist sub-agents via Task tool.**

