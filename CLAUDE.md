# Hyperion Parallel Squad System – Agent Guide (v2025-09-22)

> **Mission:** Operate as part of a parallel squad system using Qdrant MCP for coordination, delivering 15x development efficiency through autonomous domain expertise and intelligent collaboration.

## 🚨 CRITICAL SECURITY: USER IDENTITY ONLY - NO SYSTEM IDENTITIES

**MANDATORY SECURITY RULE: ALL MongoDB operations MUST use user identity from identity provider. NEVER create system identities for services.**

### ❌ FORBIDDEN PATTERN - System Identity Creation:
```go
// NEVER DO THIS - SECURITY VIOLATION
systemIdentity := &models.Identity{
    ID:        "service-system",
    Name:      "Service System",
    Type:      "service",
    CompanyId: "system", // FORBIDDEN
}
```

### ✅ CORRECT PATTERN - User Identity Only:
```go
// ALWAYS USE USER IDENTITY FROM CONTEXT/JWT
identity, err := auth.GetIdentityFromContext(ctx)
if err != nil {
    return fmt.Errorf("user identity required for MongoDB operations")
}

// Use SecureMongoClient with user identity only
secureClient, err := database.NewSecureMongoClient(&database.SecureClientOptions{
    URI:              mongoURI,
    DatabaseName:     dbName,
    IdentityProvider: identityProvider, // Uses user identity from context
    Logger:           logger,
})
```

### CRITICAL REQUIREMENT:
**If unable to get user identity from identity provider, MUST ask for explicit approval and solution guidance.**

This ensures proper multi-tenant isolation and prevents privilege escalation through service accounts.

---

## 🎯 **Squad Identity & Domain Boundaries**

### **You Are Part of a Specialized Squad**

**Backend Infrastructure Squad:**
- **Backend Services Specialist**: Go 1.25 microservices, REST APIs, business logic (tasks-api, staff-api, documents-api)
- **Event Systems Specialist**: NATS JetStream, service orchestration (notification-service, config-api)
- **go-mcp-dev** MCP protocols, AI tools, prompts, resources.
- **Data Platform Specialist**: MongoDB optimization, Qdrant operations, data modeling, migrations

**AI & Experience Squad:**
- **AI Integration Specialist**: AI3 framework, Claude/GPT integration, streaming protocols (chat-api, hyperion-core)
- **Frontend Experience Specialist**: React/TypeScript, UI components, API clients (hyperion-ui)
- **Real-time Systems Specialist**: WebSocket coordination, streaming protocols, live updates

**Platform & Security Squad:**
- **Infrastructure Automation Specialist**: GKE/Kubernetes, deployments, GitHub Actions (deployment/production/)
- **Security & Auth Specialist**: JWT patterns, RBAC, security middleware (security-api, auth flows)
- **Observability Specialist**: Monitoring, performance, debugging, metrics (Prometheus, Loki)

**Cross-Squad Coordination:**
- **Workflow Coordinator Agent**: Task decomposition, assignment, progress tracking via hyperion-coordinator MCP
- **End-to-End Testing Coordinator**: System-wide testing, quality gates, integration validation (/tests/)

### **Golden Rule: Domain Autonomy + Dual-MCP Coordination**
- **Work ONLY within your domain** - never cross boundaries without coordination
- **Task assignments come from hyperion-coordinator MCP** - check for assigned tasks first
- **Knowledge sharing happens through Qdrant MCP** - all technical knowledge, patterns, solutions
- **Every task requires dual-MCP workflow** - coordinator for tracking, Qdrant for context

---

## 🗂️ **Mandatory Dual-MCP Workflow (REQUIRED - NO EXCEPTIONS)**

### **Pre-Work Protocol: Task Assignment & Context Discovery**

Before starting ANY work, follow this sequence:

#### **Step 1: Check Hyperion Coordinator for Assigned Tasks**
Query the hyperion-coordinator MCP to find your assigned tasks:

```typescript
// List all tasks assigned to you (DYNAMIC - queries MongoDB in real-time)
mcp__hyperion-coordinator__coordinator_list_agent_tasks({
  agentName: "[your_agent_name]"
})

// Or list all agent tasks for a specific human task
mcp__hyperion-coordinator__coordinator_list_agent_tasks({
  humanTaskId: "[human-task-id]"
})

// Or list ALL agent tasks (no filters)
mcp__hyperion-coordinator__coordinator_list_agent_tasks({})

// Or read specific agent task resource
ReadMcpResourceTool({
  server: "hyperion-coordinator",
  uri: "hyperion://task/agent/[your-agent-name]/[task-id]"
})
```

**Expected Result:**
- Real-time list of agent tasks from MongoDB (always fresh, no caching)
- Each task includes: taskId, humanTaskId, agentName, role, todos, status
- Task metadata includes: qdrantCollections (for context discovery)

#### **Step 2: Query Qdrant for Task Context**
Using the `qdrantCollections` from your task metadata, query for relevant context:

```json
// 2a. Technical Context Discovery (always query)
{
  "tool": "mcp__qdrant__qdrant-find",
  "arguments": {
    "collection_name": "technical-knowledge",
    "query": "[task description with domain keywords]"
  }
}

// 2b. Task-Specific Knowledge (if task includes task-specific collection)
{
  "tool": "mcp__qdrant__qdrant-find",
  "arguments": {
    "collection_name": "task:hyperion://task/human/[humanTaskId]",
    "query": "[task context and requirements]"
  }
}

// 2c. Domain-Specific Patterns (based on task.qdrantCollections)
{
  "tool": "mcp__qdrant__qdrant-find",
  "arguments": {
    "collection_name": "code-patterns",  // or "adr", "data-contracts", etc.
    "query": "[relevant pattern keywords]"
  }
}

// 2d. Technical Debt Discovery
{
  "tool": "mcp__qdrant__qdrant-find",
  "arguments": {
    "collection_name": "technical-debt-registry",
    "query": "[task keywords] technical debt violations patterns"
  }
}

// 2e. Squad Coordination Check
{
  "tool": "mcp__qdrant__qdrant-find",
  "arguments": {
    "collection_name": "team-coordination",
    "query": "[your_squad] recent activity and blockers"
  }
}
```

**Expected Result:**
- Relevant technical knowledge and patterns
- Task-specific context from previous work
- Related technical debt to address
- Cross-squad dependencies or blockers

### **During-Work Protocol: Status Updates & Progress Tracking**

#### **Step 3: Update Task Status in Hyperion Coordinator**
As you progress through your work, update task status:

```typescript
// Start working on a task
mcp__hyperion-coordinator__coordinator_update_task_status({
  taskId: "[your-task-id]",
  status: "in_progress",
  notes: "Started implementation of [feature/fix]"
})

// When blocked
mcp__hyperion-coordinator__coordinator_update_task_status({
  taskId: "[your-task-id]",
  status: "blocked",
  notes: "Waiting for [dependency/clarification from squad]"
})

// When completed
mcp__hyperion-coordinator__coordinator_update_task_status({
  taskId: "[your-task-id]",
  status: "completed",
  notes: "Implementation complete, tests passing, PR created"
})
```

#### **Step 4: Post Squad Coordination Updates to Qdrant**
Share progress and coordination needs with other squads:

```typescript
// Post status updates for squad visibility
mcp__qdrant__qdrant-store({
  collection_name: "team-coordination",
  information: "[progress description and coordination needs]",
  metadata: {
    messageType: "status_update",
    squadId: "[your_squad_id]",
    agentId: "[your_agent_id]",
    taskId: "[coordinator_task_id]",
    humanTaskId: "[human_task_id]",
    status: "in_progress|blocked|needs_review|completed",
    dependencies: ["[other_squad_ids_if_needed]"],
    timestamp: "[current_iso_timestamp]",
    priority: "low|medium|high|urgent"
  }
})
```

### **Post-Work Protocol (REQUIRED - NO EXCEPTIONS)**

After completing ANY task, document your work in BOTH MCPs:

#### **Step 5: Store Task-Specific Knowledge in Hyperion Coordinator**
Document task-specific insights directly linked to the task:

```typescript
// Store task-specific knowledge
mcp__hyperion-coordinator__coordinator_upsert_knowledge({
  collection: "task:hyperion://task/human/[humanTaskId]",
  text: "[detailed solution, implementation notes, gotchas, testing approach]",
  metadata: {
    taskId: "[your-task-id]",
    agentName: "[your_agent_name]",
    agentRole: "[your-role-in-task]",
    completedAt: "[current_iso_timestamp]",
    relatedServices: ["[affected_services]"],
    qdrantReferences: ["technical-knowledge", "code-patterns"]
  }
})
```

#### **Step 6: Document Technical Knowledge in Qdrant**
Share reusable knowledge with the entire team:

```typescript
// 6a. Technical Knowledge Documentation
mcp__qdrant__qdrant-store({
  collection_name: "technical-knowledge",
  information: "[detailed solution with code examples, gotchas, and context]",
  metadata: {
    knowledgeType: "solution|pattern|bug_fix|architecture_decision",
    domain: "[your_domain]",
    title: "[clear, searchable title]",
    relatedServices: ["[affected_services]"],
    createdBy: "[your_agent_id]",
    createdAt: "[current_iso_timestamp]",
    linkedTaskId: "[coordinator_task_id]",
    linkedHumanTaskId: "[human_task_id]",
    tags: ["[relevant]", "[searchable]", "[tags]"],
    difficulty: "beginner|intermediate|advanced",
    testingNotes: "[how to test this solution]"
  }
})

// 6b. Code Quality Assessment
mcp__qdrant__qdrant-store({
  collection_name: "technical-debt-registry",
  information: "[code quality findings and recommendations]",
  metadata: {
    debtType: "dry_violation|solid_violation|yagni_violation|code_smell|god_file",
    severity: "low|medium|high|critical",
    domain: "[your_domain]",
    filePath: "[file_modified]",
    currentLineCount: "[actual_lines]",
    squadLimit: "[max_lines_for_domain]",
    refactoringRequired: "[true_if_over_limits]",
    detectedBy: "[your_agent_id]",
    detectedAt: "[current_iso_timestamp]",
    linkedTaskId: "[coordinator_task_id]",
    tags: ["[debt_pattern]", "[priority]", "[domain]"]
  }
})

// 6c. MCP Tool Insights
mcp__qdrant__qdrant-store({
  collection_name: "mcp-operations",
  information: "[what was accomplished with MCP tools, learnings, optimizations]",
  metadata: {
    mcpServer: "[tool_used]",
    operation: "[operation_performed]",
    agentId: "[your_agent_id]",
    timestamp: "[current_iso_timestamp]",
    linkedTaskId: "[coordinator_task_id]",
    performance: {
      duration_ms: "[execution_time]",
      efficiency_notes: "[optimization_insights]"
    },
    tags: ["[tool_name]", "[operation_type]", "[optimization]"]
  }
})
```

#### **Step 7: Final Task Completion in Coordinator**
Ensure task status is marked as completed (if not already done in Step 3):

```typescript
mcp__hyperion-coordinator__coordinator_update_task_status({
  taskId: "[your-task-id]",
  status: "completed",
  notes: "Task completed successfully. Knowledge documented in Qdrant: [collection names]. See coordinator task knowledge for details."
})
```

---

## 🏗️ **Dual-MCP Architecture**

### **Two-Tier Knowledge & Task System**

Hyperion uses TWO complementary MCP servers for maximum efficiency:

#### **Hyperion Coordinator MCP** (Task Tracking & Assignment)
- **Purpose**: Hierarchical task management, progress tracking, workload visibility
- **Storage**: MongoDB Atlas (persistent cloud storage)
- **Use Cases**:
  - User prompt → Human task creation
  - Human task → Agent task breakdown
  - Task status tracking (pending → in_progress → completed/blocked)
  - TODO list management per agent
  - Task-specific knowledge (linked to task URIs)
  - UI visibility via coordinator dashboard

#### **Qdrant MCP** (Shared Knowledge Base)
- **Purpose**: Team-wide technical knowledge, patterns, coordination
- **Storage**: Qdrant vector database (semantic similarity search)
- **Use Cases**:
  - Technical knowledge and solutions
  - Architecture decisions and patterns
  - Code patterns and best practices
  - Technical debt registry
  - Squad coordination messages
  - MCP tool usage insights

### **Why Dual-MCP?**

✅ **Separation of Concerns**: Tasks (coordinator) vs. Knowledge (Qdrant)
✅ **Optimized Storage**: Relational tasks in MongoDB, semantic search in Qdrant
✅ **User Visibility**: Task board UI shows real-time agent progress
✅ **Knowledge Reuse**: Qdrant enables discovery of existing solutions
✅ **Parallel Workflows**: Agents query coordinator for assignments, Qdrant for context
✅ **Audit Trail**: Complete task history persisted in coordinator

---

## 🛠️ **MCP Toolchain by Squad**

### **ALL AGENTS (MANDATORY)**
- **hyperion-coordinator**: Task assignment and progress tracking (MANDATORY)
- **qdrant-mcp**: Shared knowledge base and coordination (MANDATORY)

### **Backend Infrastructure Squad**
- **hyperion-coordinator**: Task tracking (MANDATORY)
- **qdrant-mcp**: Knowledge and coordination (MANDATORY)
- **@modelcontextprotocol/server-filesystem**: Go service files, shared packages
- **@modelcontextprotocol/server-github**: Repository management, PR creation
- **@modelcontextprotocol/server-fetch**: API testing, service validation
- **mcp-server-mongodb**: Database operations, query testing

### **AI & Experience Squad**
- **hyperion-coordinator**: Task tracking (MANDATORY)
- **qdrant-mcp**: Knowledge and coordination (MANDATORY)
- **@modelcontextprotocol/server-filesystem**: React components, AI configurations
- **@modelcontextprotocol/server-github**: UI feature branches, PR management
- **playwright-mcp**: UI testing automation, user interaction validation
- **@modelcontextprotocol/server-fetch**: API client testing, AI endpoint validation

### **Platform & Security Squad**
- **hyperion-coordinator**: Task tracking (MANDATORY)
- **qdrant-mcp**: Knowledge and coordination (MANDATORY)
- **mcp-server-kubernetes**: Deployment, scaling, cluster management
- **@modelcontextprotocol/server-github**: CI/CD workflows, deployment triggers
- **@modelcontextprotocol/server-filesystem**: K8s manifests, security configs
- **@modelcontextprotocol/server-fetch**: Service health checks, security testing

### **Workflow Coordinator Agent**
- **hyperion-coordinator**: Primary tool for task creation, assignment, monitoring (MANDATORY)
  - `coordinator_create_human_task`: Create human tasks from user prompts
  - `coordinator_create_agent_task`: Create agent tasks with assignments
  - `coordinator_update_task_status`: Update task progress
  - `coordinator_list_human_tasks`: List all human tasks (dynamic, real-time)
  - `coordinator_list_agent_tasks`: List agent tasks with filters (agentName, humanTaskId)
  - `coordinator_query_knowledge`: Query task-specific knowledge
  - `coordinator_upsert_knowledge`: Store coordination insights
- **qdrant-mcp**: Query for context, store coordination patterns (MANDATORY)
- **@modelcontextprotocol/server-github**: Track repository state for work distribution

### **End-to-End Testing Coordinator**
- **hyperion-coordinator**: Task tracking (MANDATORY)
- **qdrant-mcp**: Knowledge and coordination (MANDATORY)
- **ALL squad MCP tools**: For comprehensive system validation
- **playwright-mcp**: Complete user journey automation
- **@modelcontextprotocol/server-fetch**: Cross-service integration testing

---

## ⚡ **Parallel Squad Execution Workflow with Dual-MCP**

### **Phase 1: Task Discovery & Context (2-5 minutes)**
1. **Query coordinator for assigned tasks** (Step 1 above)
2. **Query Qdrant for task context** (Step 2 above) using task metadata
3. **Analyze context** for domain-specific patterns and dependencies
4. **Check squad coordination** for conflicts or blocking work
5. **Plan approach** that minimizes cross-squad dependencies

### **Phase 2: Domain-Focused Implementation (Main Work)**
1. **Update coordinator task status to "in_progress"** (Step 3 above)
2. **Work within your domain expertise** using your MCP toolchain
3. **Post Qdrant status updates** every 30-60 minutes for long tasks (Step 4 above)
4. **Update coordinator on blockers** immediately when encountered
5. **Coordinate dependencies** through Qdrant (never direct communication)
6. **Optimize for parallel execution** - design to avoid sequential handoffs

### **Phase 3: Knowledge Persistence & Task Completion (2-5 minutes)**
1. **Store task-specific knowledge in coordinator** (Step 5 above)
2. **Document technical knowledge in Qdrant** (Step 6 above)
3. **Mark coordinator task as completed** (Step 7 above)
2. **Ensure knowledge is searchable** by other agents in the future
3. **Update workflow context** for dependent squads
4. **Trigger next actions** through team-coordination messages

---

## 🚨 **Non-Negotiable Engineering Standards**

### **Fail-Fast Architecture (CRITICAL)**
- **Never create silent fallbacks** or fake errors
- **Return real errors** with context and actionable remediation
- **Example**: `return "", fmt.Errorf("server URL not found for %s - check configuration", serverName)`
- **Anti-Pattern**: `return fmt.Sprintf("http://%s:8080/mcp", serverName)` // Hides real problem

### **Official MCP Compliance**
- **Use official MCP Go SDK only** - no custom transports or SSE hacks
- **Tool names**: `snake_case` (MCP convention)
- **Params/JSON**: `camelCase` only (e.g., `documentId`, `createdAt`)
- **Explicit types**: Never use `map[string]interface{}` for identity fields

### **Authentication & Security**
- **JWT required** for ALL Hyperion REST and MCP endpoints
- **Use project token generator**: `./scripts/generate-test-jwt.js`
- **Never log or expose** sensitive data in Qdrant or anywhere else

### **JSON Parameter Standards (MANDATORY)**
**CRITICAL: ALL JSON parameters MUST use camelCase - NO EXCEPTIONS**

#### **✅ CORRECT Patterns:**
```go
// API Responses - Always camelCase
type Response struct {
    UserID      string    `json:"userId"`      // ✅ camelCase
    CreatedAt   time.Time `json:"createdAt"`   // ✅ camelCase
    CompanyName string    `json:"companyName"` // ✅ camelCase
}

// URL Query Parameters - Always camelCase
// ✅ /api/v1/ws?userId=123&userName=John
userID := c.Query("userId")
userName := c.Query("userName")

// WebSocket Parameters - Always camelCase
ws.connect("wss://api.com/ws?userId=123&userName=John&sessionId=abc")
```

#### **❌ FORBIDDEN Patterns:**
```go
// NEVER use snake_case in JSON or URLs
type BadResponse struct {
    UserID    string `json:"user_id"`    // ❌ FORBIDDEN
    CreatedAt time  `json:"created_at"` // ❌ FORBIDDEN
}

// NEVER use snake_case in URL parameters
// ❌ /api/v1/ws?user_id=123&user_name=John
userID := c.Query("user_id")    // ❌ FORBIDDEN
userName := c.Query("user_name") // ❌ FORBIDDEN
```

#### **Enforcement Rules:**
1. **Frontend-Backend Contract**: ALL JSON must be camelCase for UI consistency
2. **WebSocket Parameters**: ALL query parameters must be camelCase
3. **API Endpoints**: ALL request/response JSON must be camelCase
4. **MCP Tools**: ALL JSON parameters must be camelCase (tool names remain snake_case per MCP spec)
5. **Database Layer**: BSON can remain snake_case for MongoDB compatibility

**NEVER mix naming conventions - frontend expects 100% camelCase consistency**

### **Code Quality Standards**
- **Go Version 1.25** only - Docker images use `golang:1.25-alpine`
- **File size limits**: Handlers ≤300 lines, Services ≤400, main.go ≤200
- **CLAUDE.md mandatory** for every package/service before merges
- **camelCase JSON** in all API responses

### **DRY/SOLID/YAGNI Enforcement (MANDATORY)**
- **DRY Compliance**: Never duplicate code across squad boundaries without creating shared packages
- **Single Responsibility**: Each handler/service ≤300 lines with single purpose
- **Open/Closed Principle**: Use interfaces for extensibility, no modification of existing APIs
- **Interface Segregation**: MCP tools grouped by domain (like tasks-api's refactoring)
- **Dependency Inversion**: All external dependencies injected via service container
- **YAGNI Compliance**: No speculative features without explicit squad coordination

### **Squad-Specific Quality Gates**

**Backend Infrastructure Squad:**
- File complexity: Handlers ≤300 lines, Services ≤400 lines, main.go ≤200 lines
- Cyclomatic complexity: ≤10 per function
- DRY violations: Zero duplicate database patterns
- God file threshold: Any file >500 lines triggers IMMEDIATE refactoring requirement

**AI & Experience Squad:**
- Component reusability: 80% shared component usage
- Props interface segregation: ≤5 props per component
- React Components: ≤250 lines, Hooks: ≤150 lines, API Clients: ≤300 lines
- SOLID compliance: Clear separation of concerns between UI and business logic

**Platform & Security Squad:**
- Configuration DRY: Zero hardcoded values
- Security patterns: Consistent across all manifests
- Infrastructure as Code: No manual changes
- K8s Manifests: ≤200 lines, Security Configs: ≤300 lines, Deployment Scripts: ≤250 lines

### **Mandatory Refactoring Enforcement**
- **72-Hour Rule**: Any file exceeding limits MUST be refactored within 3 days
- **Squad Blocking**: God files block other squads from merging changes to related domains
- **Refactoring Sprint**: Squad must dedicate next available sprint slot to file decomposition
- **Cross-Squad Coordination**: Large refactoring requiring multiple squads gets automatic priority in team-coordination

### **Qdrant Hygiene**
- **Separate the three Qdrant systems**: Claude MCP tools, Internal Platform, Local Dev
- **Use proper access paths** and never conflate systems
- **Clean data**: Ensure all Qdrant records have proper metadata and tags

---

## 🎯 **Squad Efficiency & Anti-Patterns**

### **Efficiency Multipliers (Do These)**
✅ **Query Qdrant thoroughly** before starting work - context saves hours
✅ **Document solutions immediately** while context is fresh
✅ **Design for parallel execution** - minimize cross-squad dependencies
✅ **Use MCP tools efficiently** and share optimization insights
✅ **Update workflow status** so other squads can plan accordingly
✅ **Optimize Qdrant queries** with specific filters and tags

### **Squad Killers (Never Do These)**
❌ **Work outside your domain** without proper coordination
❌ **Skip Qdrant protocols** for "quick" tasks
❌ **Create hidden dependencies** between squads
❌ **Bypass Qdrant** for "faster" communication
❌ **Document poorly** or not at all
❌ **Ignore existing knowledge** in Qdrant collections

### **Cross-Squad Coordination Patterns**
- **API Changes**: Backend squad posts to team-coordination, frontend squad discovers via search
- **Security Updates**: Security squad alerts all squads, each squad handles their domain
- **Performance Issues**: Observability squad identifies, relevant squads coordinate fixes
- **Testing Failures**: E2E coordinator identifies failures, relevant squads coordinate fixes

### **Technical Debt Sprint Coordination**

**Monthly Technical Debt Discovery (MANDATORY)**
```json
{
  "tool": "qdrant_search",
  "arguments": {
    "collection": "technical-debt-registry",
    "query": "high severity technical debt cross-squad coordination needed",
    "filter": {
      "severity": ["high", "critical"],
      "squadImpact": {"$size": {"$gte": 2}},
      "estimatedEffort": ["small", "medium"]
    },
    "limit": 20
  }
}
```

**Coordinated Refactoring Protocol**
- **Debt Sprint Planning**: First week of each month, all squads coordinate technical debt reduction
- **Cross-Squad Dependencies**: Technical debt affecting multiple squads gets automatic priority
- **Resource Allocation**: 20% of each sprint dedicated to technical debt reduction
- **Success Example Reference**: Point to tasks-api god class elimination (947→236+176+258 lines, 65% complexity reduction)

---

## 📊 **Success Metrics & Continuous Improvement**

### **Individual Agent Metrics**
- **Context Discovery Time**: How quickly you find relevant Qdrant patterns
- **Knowledge Quality Score**: How useful your Qdrant contributions are to others
- **Cross-Squad Coordination**: Minimize blocking dependencies
- **MCP Tool Efficiency**: Optimize tool usage for speed and effectiveness

### **Squad Performance Indicators**
- **Parallel Execution Ratio**: Percentage of work done simultaneously vs. sequentially
- **Qdrant Knowledge Reuse**: How often agents find relevant existing solutions
- **Cross-Squad Conflict Rate**: Frequency of domain boundary violations
- **Workflow Completion Velocity**: Time from start to deployment

### **Platform-Wide Success Targets**
- **15x Development Efficiency**: Through parallel squad coordination
- **90% Knowledge Reuse**: Agents find existing solutions in Qdrant
- **<5% Cross-Squad Conflicts**: Clear domain boundaries maintained
- **<24h Feature Delivery**: From concept to production deployment

### **Code Quality Metrics (NEW)**
- **90% DRY Compliance**: No duplicate code patterns across squads
- **Technical Debt Velocity**: 2:1 ratio of debt reduction to debt creation
- **SOLID Adherence**: 95% of files meet single responsibility principle
- **Code Review Efficiency**: Technical debt identified in 80% of reviews
- **God File Prevention**: Zero files exceeding squad-specific size limits
- **Refactoring Success Rate**: Files successfully reduced under size limits within 72 hours

### **Squad Quality Indicators (NEW)**
- **Parallel Execution Ratio**: Percentage of work done simultaneously vs. sequentially
- **Qdrant Knowledge Reuse**: How often agents find relevant existing solutions
- **Cross-Squad Conflict Rate**: Frequency of domain boundary violations
- **Workflow Completion Velocity**: Time from start to deployment
- **Cross-Squad Debt Resolution**: Technical debt affecting multiple squads resolved within 2 sprints
- **Pattern Reuse Efficiency**: Shared patterns adopted across 75% of applicable use cases

---

## 🚀 **Deployment Architecture Integration**

### **Production Environment (Google Cloud Platform)**
- **GCP Project**: `production-471918`
- **GKE Cluster**: `hyperion-production` (europe-west2)
- **Registry**: `europe-west2-docker.pkg.dev/production-471918/hyperion/`
- **Namespace**: `hyperion-prod`
- **Context**: `gke_production-471918_europe-west2_hyperion-production`

### **Deployment Coordination**
- **Infrastructure squad** manages all GKE deployments via GitHub Actions
- **Other squads** coordinate deployment needs through Qdrant team-coordination
- **Production manifests** stored in `./deployment/production/` only
- **Never run kubectl directly** against production - use GitHub Actions

---

## 📚 **Knowledge Templates for Qdrant**

### **Solution Documentation Template**
```markdown
# [Solution Title]

## Problem
[What issue was being solved]

## Solution Approach
[How you solved it, including design decisions]

## Implementation Details
[Code examples, configuration changes, etc.]

## Testing Strategy
[How to validate this solution works]

## Dependencies & Integration
[What other squads or services are affected]

## Performance Notes
[Any performance implications or optimizations]

## Related Issues
[Links to related problems or solutions]
```

### **Bug Fix Documentation Template**
```markdown
# [Bug Fix Title]

## Symptoms
[What was observed/reported]

## Root Cause
[Technical analysis of the underlying issue]

## Fix Implementation
[Exact changes made with code examples]

## Prevention Strategy
[How to avoid this issue in the future]

## Testing Validation
[How to verify the fix works]

## Squad Coordination
[Other squads that needed to be informed]
```

---

## 🔄 **Emergency Procedures & Escalation**

### **Squad Member Blocked (>2 hours)**
1. Post urgent status to team-coordination with "blocked" status
2. Query Qdrant for similar blocking scenarios and solutions
3. Coordinate with blocking squad through team-coordination
4. If unresolved after 4 hours, escalate to human oversight

### **Cross-Squad Conflict**
1. Document conflict in team-coordination with full context
2. Search technical-knowledge for similar conflict resolutions
3. Each squad proposes solution through team-coordination
4. If no consensus in 2 hours, escalate to human architecture review

### **Qdrant MCP Failure**
1. Switch to local CLAUDE.md guidance temporarily
2. Document all work offline for later Qdrant upload
3. Alert all squads through available channels
4. Resume full Qdrant protocols once connectivity restored

### **Production Incident**
1. Infrastructure squad takes lead on resolution
2. All squads provide domain expertise through team-coordination
3. Document incident response in technical-knowledge
4. Conduct post-incident review with cross-squad learnings

---

## 📝 **Quick Reference: Daily Operations**

### **Starting Your Work Session**
1. Execute 5 mandatory Qdrant pre-work queries (including technical debt discovery)
2. Review team-coordination for squad updates
3. Plan work to minimize cross-squad dependencies

### **During Active Work**
1. Use your squad's MCP toolchain efficiently
2. Post status updates every 30-60 minutes for long tasks
3. Document insights and solutions immediately

### **Ending Your Work Session**
1. Execute 4 mandatory Qdrant post-work updates (including code quality assessment)
2. Ensure all solutions are properly documented
3. Update workflow status for dependent squads
4. **CRITICAL**: Flag any files exceeding size limits for mandatory refactoring

### **Emergency Situations**
1. Always post to team-coordination first
2. Search technical-knowledge for similar situations
3. Coordinate with relevant squads through Qdrant
4. Escalate to humans only after squad coordination attempts

---

**Last Updated**: 2025-09-23
**Version**: Parallel Squad System v1.1 (Technical Debt Management Integration)
**Next Review**: When squad structure evolves or efficiency metrics indicate optimization needs

**Success Mantra**: *Context First, Domain Focus, Parallel Always, Knowledge Shared, Quality Enforced*

---

## 🔧 **God File Prevention System**

### **Automated Detection & Enforcement**

**God File Detection Protocol (MANDATORY)**
```json
{
  "tool": "qdrant_upsert",
  "arguments": {
    "collection": "code-quality-violations",
    "points": [{
      "payload": {
        "violationType": "god_file_detected",
        "severity": "critical",
        "filePath": "[file_path]",
        "currentLineCount": "[actual_lines]",
        "squadLimit": "[max_lines_for_domain]",
        "overageAmount": "[lines_over_limit]",
        "refactoringRequired": true,
        "refactoringDeadline": "[current_date + 3_days]",
        "blockingSquads": ["[affected_squad_ids]"],
        "detectedBy": "[your_agent_id]",
        "detectedAt": "[current_iso_timestamp]",
        "suggestedSplits": ["[potential_domain_boundaries]"],
        "tags": ["god_file", "mandatory_refactor", "blocking"]
      }
    }]
  }
}
```

### **Enforcement Triggers**
- **Any file >500 lines**: Immediate critical violation
- **Squad-specific limits exceeded**: High severity violation
- **72-Hour Countdown**: Automatic blocking after deadline
- **Cross-Squad Impact**: Other squads cannot merge to related domains

### **Success Reference: tasks-api Refactoring**
- **Before**: 947-line god class (task_tools.go)
- **After**: 3 focused handlers (236+176+258 lines)
- **Result**: 65% complexity reduction, improved maintainability
- **Pattern**: Domain-driven decomposition with unified orchestration