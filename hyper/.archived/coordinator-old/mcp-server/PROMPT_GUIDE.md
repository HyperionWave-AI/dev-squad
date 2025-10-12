# Hyperion MCP Prompts Guide

**Version:** 1.0.0
**Last Updated:** 2025-10-11
**Total Prompts:** 7 prompts across 4 categories

## Overview

MCP Prompts provide **AI-assisted guidance** for complex coordination tasks. Unlike tools that execute actions, prompts generate **structured recommendations** to help coordinators and agents make better decisions.

**Key Benefits:**
- Reduce cognitive load during planning
- Standardize task breakdown patterns
- Optimize knowledge queries
- Detect cross-squad conflicts early
- Streamline multi-phase handoffs

---

## Table of Contents

1. [Planning Prompts](#planning-prompts) (2 prompts)
   - [plan_task_breakdown](#1-plan_task_breakdown)
   - [suggest_context_offload](#2-suggest_context_offload)
2. [Knowledge Management Prompts](#knowledge-management-prompts) (2 prompts)
   - [recommend_qdrant_query](#3-recommend_qdrant_query)
   - [suggest_knowledge_structure](#4-suggest_knowledge_structure)
3. [Coordination Prompts](#coordination-prompts) (2 prompts)
   - [detect_cross_squad_impact](#5-detect_cross_squad_impact)
   - [suggest_handoff_strategy](#6-suggest_handoff_strategy)
4. [Documentation Prompts](#documentation-prompts) (1 prompt)
   - [build_knowledge_base](#7-build_knowledge_base)

---

## Planning Prompts

### 1. plan_task_breakdown

**Purpose:** Break down complex tasks into 3-7 detailed TODO items with exact file paths, function names, and implementation hints to enable agents to start coding within 2 minutes.

**When to Use:**
- Creating agent tasks for implementation work
- Need to embed 80%+ context directly in task
- Want to minimize agent exploration time
- Breaking down multi-step features

**Required Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `taskDescription` | string | High-level description of what needs to be accomplished |
| `targetSquad` | string | The squad/agent that will implement (e.g., `go-mcp-dev`, `ui-dev`, `backend-services`) |

**Example Invocation:**

```typescript
const prompt = await mcp.prompts.get({
  name: "plan_task_breakdown",
  arguments: {
    taskDescription: "Add JWT authentication middleware to staff-api service with token validation, error handling, and integration with existing Gin routes",
    targetSquad: "go-dev"
  }
});

// Send prompt.messages to AI for task breakdown
```

**Example Response Structure:**

The AI will provide a detailed planning prompt that guides you to create:

```json
{
  "todos": [
    {
      "description": "Create JWT validation middleware function",
      "filePath": "staff-api/middleware/auth.go",
      "functionName": "ValidateJWT",
      "contextHint": "Extract token from Authorization header (Bearer scheme). Use jwt.Parse() with HS256 algorithm. Validate exp, iss, aud claims. Store user ID in gin.Context with key 'userID'. Return 401 with {\"error\": \"invalid_token\"} on failure. See technical-knowledge collection for JWT middleware pattern."
    },
    {
      "description": "Integrate middleware with Gin router",
      "filePath": "staff-api/cmd/server/main.go",
      "functionName": "setupRoutes",
      "contextHint": "Add middleware.ValidateJWT() to protected route groups. Exclude /health and /metrics endpoints. Use router.Use() for global application, router.Group().Use() for specific routes. Protected routes: /api/v1/persons/*, /api/v1/companies/*."
    },
    {
      "description": "Add middleware tests",
      "filePath": "staff-api/middleware/auth_test.go",
      "functionName": "TestValidateJWT",
      "contextHint": "Test cases: valid token (200), expired token (401), invalid signature (401), missing token (401), malformed header (401). Use httptest.NewRecorder() and gin.CreateTestContext(). Generate test tokens with scripts/generate-test-jwt.js."
    }
  ],
  "filesModified": [
    "staff-api/middleware/auth.go",
    "staff-api/cmd/server/main.go",
    "staff-api/middleware/auth_test.go"
  ],
  "estimatedDuration": "45-60 minutes"
}
```

**Common Pitfalls:**

- ❌ **Too Generic:** "Add authentication" → ✅ "Create JWT middleware with token validation"
- ❌ **Missing Context:** No file paths or function names → ✅ Include exact locations
- ❌ **Too Many TODOs:** >7 TODOs means task is too large → ✅ Split into multiple tasks
- ❌ **No Implementation Hints:** Agent has to explore → ✅ Embed HOW in contextHint

---

### 2. suggest_context_offload

**Purpose:** Analyze task scope and recommend what context to embed directly in task fields (contextSummary, filesModified, etc.) versus what to store in Qdrant for semantic search.

**When to Use:**
- Planning complex multi-phase tasks
- Deciding what goes in task vs knowledge base
- Optimizing context efficiency (target: 80% in task, 20% in Qdrant)
- Balancing immediate context vs reusable patterns

**Required Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `taskScope` | string | Detailed scope including requirements, constraints, integration points |

**Optional Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `existingKnowledge` | string | Comma-separated list of Qdrant collections or knowledge references (e.g., `technical-knowledge,code-patterns`) |

**Example Invocation:**

```typescript
const prompt = await mcp.prompts.get({
  name: "suggest_context_offload",
  arguments: {
    taskScope: `
      Implement real-time task updates using WebSocket.
      Requirements:
      - Use existing NATS JetStream for pub/sub
      - Frontend uses React Query for state
      - Need optimistic UI updates
      - Support task assignment, status changes, comment additions
      Constraints:
      - <5ms latency for UI updates
      - Handle connection drops gracefully
      - Company-level data isolation
    `,
    existingKnowledge: "technical-knowledge,ui-component-patterns,real-time-systems"
  }
});
```

**Example Response Structure:**

The AI will provide recommendations like:

```markdown
### 1. Task Field Content Recommendations

**contextSummary:**
"Implement WebSocket real-time task updates. WHY: Users need instant feedback on task changes across browsers. WHAT: NATS JetStream pub/sub with WebSocket frontend connection. HOW: Subscribe to task.* events in NATS, broadcast via WebSocket to matching company. CONSTRAINTS: <5ms latency, graceful reconnection, company isolation via JWT. TESTING: Test with concurrent users, network interruptions, 100+ simultaneous connections."

**filesModified:**
- backend/websocket/handler.go
- backend/websocket/nats_subscriber.go
- ui/src/hooks/useRealtimeUpdates.ts
- ui/src/providers/WebSocketProvider.tsx

**qdrantCollections:**
- technical-knowledge: "NATS JetStream subscription pattern"
- ui-component-patterns: "React Query optimistic update pattern"

**notes:**
- WebSocket connection must validate JWT on connect
- Use gorilla/websocket library (already in codebase)
- Frontend reconnects exponentially (1s, 2s, 4s, 8s max)

### 2. Qdrant Storage Recommendations

**Collections to Update:**
- Collection: technical-knowledge
  Purpose: NATS JetStream subscription pattern (reusable)
  Example entry: "NATS JetStream durable consumer setup with error handling"

**3. Context Efficiency Score**
- Task-embedded: 85%
- Qdrant-required: 15%

**4. Agent Work Estimate**
- Time to read task context: 90 seconds
- Qdrant queries needed: 2 queries (NATS pattern, React Query pattern)
- Time to start coding: <2 minutes
```

**Common Pitfalls:**

- ❌ **Over-reliance on Qdrant:** Agent spends 10 min searching → ✅ Embed task-specific context
- ❌ **Duplicate Storage:** Same info in task AND Qdrant → ✅ Task-specific in task, reusable in Qdrant
- ❌ **No Knowledge Collections:** Agent doesn't know where to search → ✅ Specify 1-3 collections

---

## Knowledge Management Prompts

### 3. recommend_qdrant_query

**Purpose:** Analyze what an agent needs to know and recommend the optimal Qdrant query strategy (collection, query string, fallback) to find it efficiently with minimal context usage.

**When to Use:**
- Agent needs to find existing patterns/solutions
- Unsure which Qdrant collection to search
- Need to optimize query strings for semantic search
- Want to minimize query attempts (target: 1 query max)

**Required Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `agentQuestion` | string | What the agent wants to know or problem to solve |
| `taskContext` | string | Current task context (squad, service, feature being worked on) |

**Optional Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `availableCollections` | string | Comma-separated list of Qdrant collections (e.g., `technical-knowledge,code-patterns`) |

**Example Invocation:**

```typescript
const prompt = await mcp.prompts.get({
  name: "recommend_qdrant_query",
  arguments: {
    agentQuestion: "How do I implement JWT token validation in Go Gin middleware with proper error handling?",
    taskContext: "Working on staff-api authentication. Need to secure REST endpoints with JWT. Using Gin framework.",
    availableCollections: "technical-knowledge,code-patterns"
  }
});
```

**Example Response Structure:**

```markdown
### Primary Query Strategy

**Collection:** technical-knowledge
**Reason:** Technical patterns for authentication are stored here with code examples

**Query String:**
```
Go Gin JWT middleware HS256 token validation error handling
```

**Expected Results:**
- Code example showing jwt.Parse() usage with Gin
- Error handling pattern for 401 responses
- Integration with Gin router middleware chain
- Confidence level: High

### Alternative Query (if primary fails)

**Collection:** code-patterns
**Query String:**
```
Go JWT authentication middleware pattern
```

### Fallback Plan

If both queries return no results:
1. Check task's contextHint field for inline guidance
2. Search broader: "Go middleware authentication pattern"
3. Remember to DOCUMENT solution in technical-knowledge after implementing

### Context Check
⚠️ Before querying, verify task context doesn't already contain:
- [x] JWT validation pattern in contextHint
- [ ] File locations in filesModified
- [ ] Similar solutions in priorWorkSummary

**If task context has it → DON'T QUERY!**

## Example Query

```typescript
const results = await mcp__hyper__coordinator_query_knowledge({
  collection: "technical-knowledge",
  query: "Go Gin JWT middleware HS256 token validation error handling",
  limit: 3
});
```
```

**Common Pitfalls:**

- ❌ **Too Generic:** "authentication" → ✅ "Go Gin JWT middleware HS256 validation"
- ❌ **Multiple Queries:** Try 5 different collections → ✅ 1-2 targeted queries max
- ❌ **Skip Task Context:** Query before reading task → ✅ Check task.contextHint first
- ❌ **No Fallback:** Give up after first try → ✅ Have alternative query ready

---

### 4. suggest_knowledge_structure

**Purpose:** Help agents transform raw learnings and solutions into structured, searchable knowledge for optimal Qdrant storage and future reuse.

**When to Use:**
- After completing implementation work
- Discovered non-obvious gotchas or patterns
- Created reusable solution worth sharing
- Need to document for future agents

**Required Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `rawLearning` | string | What the agent learned/discovered/implemented (unstructured) |
| `context` | string | JSON task context (squad, service, files modified, task type) |

**Example Invocation:**

```typescript
const prompt = await mcp.prompts.get({
  name: "suggest_knowledge_structure",
  arguments: {
    rawLearning: `
      Implemented JWT validation middleware in Go Gin.
      Used jwt.Parse() with HS256.
      Had issues with token expiration - needed to check exp claim.
      Also discovered that without checking iss and aud claims, tokens from other services were accepted.
      Added proper error responses with specific messages for expired vs invalid signatures.
      Integration with Gin was tricky - had to exclude health endpoints.
    `,
    context: JSON.stringify({
      squad: "go-dev",
      service: "staff-api",
      taskType: "security-feature",
      filesModified: ["staff-api/middleware/auth.go", "staff-api/cmd/server/main.go"]
    })
  }
});
```

**Example Response Structure:**

The AI will guide you to structure as:

```markdown
### Title
"Go Gin JWT Middleware HS256 Token Validation with Error Handling"

### Summary
Implemented JWT validation middleware for staff-api to secure REST endpoints. The HS256 algorithm was chosen for compatibility with existing frontend auth flow. This prevents unauthorized access while maintaining <5ms validation latency per request.

### Implementation

**1. Setup/Prerequisites**
- Dependencies: `github.com/golang-jwt/jwt/v5`
- Configuration: JWT_SECRET environment variable
- Gin framework already in place

**2. Core Implementation**
```go
func ValidateJWT(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract token from Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "missing_token"})
            c.Abort()
            return
        }

        // Remove "Bearer " prefix
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        // Parse and validate token
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            // Verify algorithm
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return []byte(secret), nil
        })

        if err != nil || !token.Valid {
            c.JSON(401, gin.H{"error": "invalid_token", "message": err.Error()})
            c.Abort()
            return
        }

        // Validate claims (exp, iss, aud)
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.JSON(401, gin.H{"error": "invalid_claims"})
            c.Abort()
            return
        }

        // Check expiration (exp claim)
        if exp, ok := claims["exp"].(float64); ok {
            if time.Now().Unix() > int64(exp) {
                c.JSON(401, gin.H{"error": "token_expired"})
                c.Abort()
                return
            }
        }

        // Validate issuer and audience
        if iss, ok := claims["iss"].(string); !ok || iss != "hyperion-auth" {
            c.JSON(401, gin.H{"error": "invalid_issuer"})
            c.Abort()
            return
        }

        // Store user ID in context
        if userID, ok := claims["sub"].(string); ok {
            c.Set("userID", userID)
        }

        c.Next()
    }
}
```

**3. Integration Steps**
```go
// In main.go
router := gin.Default()

// Exclude health endpoints
router.GET("/health", healthHandler)
router.GET("/metrics", metricsHandler)

// Protected API routes
api := router.Group("/api/v1")
api.Use(middleware.ValidateJWT(os.Getenv("JWT_SECRET")))
{
    api.GET("/persons", listPersons)
    api.POST("/persons", createPerson)
}
```

### Gotchas

⚠️ **Gotcha:** JWT validation accepts tokens from other services
- **Why:** Missing issuer (iss) and audience (aud) validation
- **Solution:** Always validate iss and aud claims to ensure token is for your service
- **Detection:** Check token payload: `jwt.decode(token, verify=False)` and inspect iss/aud

⚠️ **Gotcha:** Health check fails during deployment readiness probe
- **Why:** Middleware runs on /health endpoint which has no auth token
- **Solution:** Register health/metrics endpoints BEFORE applying middleware
- **Detection:** Service fails Kubernetes readiness probe

⚠️ **Gotcha:** Token expiration causes cascading user logouts
- **Why:** exp claim not checked, or checked incorrectly (using wrong time format)
- **Solution:** Parse exp as Unix timestamp and compare with time.Now().Unix()
- **Detection:** Users stay logged in past token expiration time

### Metadata Tags
```json
["go", "jwt", "middleware", "authentication", "gin", "hs256", "security", "api"]
```

**Recommended Collection:** technical-knowledge
**Reason:** Reusable authentication pattern applicable to all Go microservices
```

**Common Pitfalls:**

- ❌ **No Code Examples:** Just descriptions → ✅ Include working code snippets
- ❌ **Generic Title:** "Authentication" → ✅ "Go Gin JWT Middleware HS256 Validation"
- ❌ **Missing Gotchas:** Don't document pitfalls → ✅ Include non-obvious edge cases
- ❌ **Wrong Collection:** Task-specific in technical-knowledge → ✅ Use task: collection for one-offs

---

## Coordination Prompts

### 5. detect_cross_squad_impact

**Purpose:** Analyze a task to detect which squads are affected by changes and what coordination is needed to prevent conflicts and ensure smooth integration.

**When to Use:**
- Before starting tasks that modify shared code
- Planning API changes or schema updates
- Working on cross-service features
- Need to know who to coordinate with

**Required Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `taskDescription` | string | Description of what's being changed or implemented |
| `filesModified` | string | Comma-separated list of file paths being modified/created |

**Optional Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `activeSquads` | string | Comma-separated list of active squad names (e.g., `backend-services,ui-dev,go-mcp-dev`) |

**Example Invocation:**

```typescript
const prompt = await mcp.prompts.get({
  name: "detect_cross_squad_impact",
  arguments: {
    taskDescription: "Add 'priority' field to Task model. Update MongoDB schema, REST API responses, MCP tools, and UI task board to display priority badges.",
    filesModified: "tasks-api/models/task.go,tasks-api/handlers/task_handler.go,tasks-api/internal/interfaces/mcp/task_mcp_handler.go,ui/src/types/task.ts,ui/src/components/TaskCard.tsx",
    activeSquads: "go-dev,go-mcp-dev,ui-dev,ui-tester"
  }
});
```

**Example Response Structure:**

```markdown
### Affected Squads

**Squad: go-dev**
- **Impact Type:** API Contract (Breaking Change)
- **Impact Level:** Non-Breaking (additive field)
- **Reason:** Adding new field to Task schema affects all consumers of task data

**Squad: go-mcp-dev**
- **Impact Type:** MCP Tool Schema
- **Impact Level:** Non-Breaking
- **Reason:** MCP tools (task_create, task_update, task_list) need to support priority parameter

**Squad: ui-dev**
- **Impact Type:** Data Model + Component
- **Impact Level:** Non-Breaking
- **Reason:** Task TypeScript interface needs priority field, TaskCard component needs update

**Squad: ui-tester**
- **Impact Type:** Test Coverage
- **Impact Level:** Medium
- **Reason:** E2E tests need to cover priority selection and display

### Required Communication

**What to Communicate:**
- New priority field: enum values ("low", "medium", "high", "urgent")
- MongoDB schema change (backward compatible - optional field)
- REST API response includes priority (optional, defaults to "medium")
- MCP tools support priority parameter (optional)
- UI displays priority badge with color coding

**How to Communicate:**
- Qdrant Collection: team-coordination
- Metadata: tags=["api-change", "schema-update", "priority-feature"]
- Notification Method: Post to team-coordination BEFORE implementation

**When to Communicate:**
- Before implementation: YES - share schema and API contract
- During implementation: YES - notify when backend complete (ui-dev can start)
- After completion: YES - document priority values and color mapping

### Coordination Actions

**Immediate Actions (Before Task Starts):**
1. Post priority field specification to team-coordination:
   - Enum values: low, medium, high, urgent
   - Default value: medium
   - MongoDB field: optional (backward compatible)
2. Query if ui-dev or go-mcp-dev have conflicting work on Task model

**During Implementation:**
1. Update task notes when MongoDB schema finalized (so ui-dev knows structure)
2. Store priority enum values in data-contracts collection

**Post-Completion:**
1. Document priority color mapping in ui-component-patterns:
   - low: gray
   - medium: blue
   - high: orange
   - urgent: red
2. Update API documentation with priority field

### Risk Assessment

**Conflict Risk:** Low
**Reasoning:** Additive change (new optional field) with no existing dependencies. All squads can work in parallel.

**Mitigation Strategy:**
1. Ensure priority field is optional in all layers (DB, API, UI)
2. Default to "medium" if not specified
3. ui-dev can mock priority data before backend ready

### Recommended Qdrant Queries

Before starting, recommend agents run:

1. **Query:** "Task model schema changes priority field"
   **Purpose:** Check if similar work was done before
   **Collection:** technical-knowledge

2. **Query:** "REST API enum field validation pattern"
   **Purpose:** Find pattern for enum validation in handlers
   **Collection:** code-patterns
```

**Common Pitfalls:**

- ❌ **Start Without Checking:** Implement then discover conflicts → ✅ Analyze impact first
- ❌ **Miss Squads:** Only notify obvious squads → ✅ Check all files modified
- ❌ **No Communication:** Assume others will discover changes → ✅ Proactive coordination
- ❌ **Unclear Contracts:** Vague API changes → ✅ Specify exact schema and examples

---

### 6. suggest_handoff_strategy

**Purpose:** Recommend optimal handoff strategy for multi-phase tasks to ensure Phase 2 agent can start coding in <2 minutes without reading Phase 1 code.

**When to Use:**
- Multi-phase task implementation (backend → frontend, or Phase 1 → Phase 2)
- Phase 1 complete, need to hand off to Phase 2
- Want to minimize Phase 2 onboarding time
- Ensure Phase 2 has API contracts without reading code

**Required Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `phase1Work` | string | JSON summary of Phase 1 completion (files, functions, API contracts, decisions) |
| `phase2Scope` | string | Description of what Phase 2 agent needs to accomplish |
| `knowledgeGap` | string | What information Phase 2 needs that isn't obvious from Phase 1 deliverables |

**Example Invocation:**

```typescript
const prompt = await mcp.prompts.get({
  name: "suggest_handoff_strategy",
  arguments: {
    phase1Work: JSON.stringify({
      completed: "JWT authentication middleware for staff-api",
      filesCreated: [
        "staff-api/middleware/auth.go",
        "staff-api/middleware/auth_test.go"
      ],
      apiContracts: {
        middleware: "ValidateJWT(secret string) gin.HandlerFunc",
        contextKey: "userID stored in gin.Context"
      },
      decisions: [
        "Used HS256 algorithm (compatible with frontend)",
        "Excluded /health and /metrics from auth",
        "Token in Authorization: Bearer <token> header"
      ],
      gotchas: [
        "Must validate iss and aud claims to prevent cross-service token acceptance",
        "Health endpoints must be registered before middleware applied"
      ]
    }),
    phase2Scope: "Frontend login form and token management. Store JWT in localStorage, add to all API requests, handle token expiration with refresh flow.",
    knowledgeGap: "Frontend doesn't know: 1) Exact Authorization header format, 2) Error response structure for expired tokens, 3) Which endpoints are protected vs public"
  }
});
```

**Example Response Structure:**

```markdown
### 1. Coordinator Knowledge Entry

**Collection:** `task:hyperion://task/human/{taskId}`

**Text Content:**
```markdown
## Phase 1 Completion Summary

### API Contracts Established

**Middleware Function:**
- Function: `ValidateJWT(secret string) gin.HandlerFunc`
- Location: `staff-api/middleware/auth.go`
- Usage: Apply to Gin router groups that need protection

**Authentication Flow:**
1. Client sends request with header: `Authorization: Bearer <jwt-token>`
2. Middleware validates token (signature, exp, iss, aud)
3. On success: User ID stored in `gin.Context` with key `"userID"`, request proceeds
4. On failure: Returns 401 with JSON error

**Error Response Format:**
```json
// Expired token
{"error": "token_expired"}

// Invalid signature
{"error": "invalid_token", "message": "signature is invalid"}

// Missing token
{"error": "missing_token"}

// Invalid issuer
{"error": "invalid_issuer"}
```

**Protected Endpoints:**
- `/api/v1/persons/*` - All person CRUD operations
- `/api/v1/companies/*` - All company CRUD operations

**Public Endpoints:**
- `/health` - Health check (no auth)
- `/metrics` - Metrics (no auth)
- `/api/v1/auth/login` - Login (Phase 2 will create)

### Key Decisions
- **Algorithm:** HS256 (shared secret with frontend)
- **Token Claims Required:** sub (user ID), exp (expiration), iss (issuer="hyperion-auth"), aud (audience="staff-api")
- **Context Storage:** User ID stored in gin.Context with key "userID"

### Files Created
- `staff-api/middleware/auth.go`: ValidateJWT function (60 lines)
- `staff-api/middleware/auth_test.go`: Unit tests (120 lines)
- `staff-api/cmd/server/main.go`: Modified to apply middleware (added 5 lines)

### Integration Points
Phase 2 needs to:
1. Call login endpoint to get JWT token
2. Store token in localStorage
3. Add `Authorization: Bearer <token>` header to all protected API calls
4. Handle 401 errors by redirecting to login
5. Implement token refresh before expiration

### Gotchas Discovered
- ⚠️ Must validate iss ("hyperion-auth") and aud claims - without these, tokens from other services are accepted
- ⚠️ Health endpoints must be registered BEFORE middleware - otherwise readiness probes fail
- ⚠️ exp claim is Unix timestamp (float64) - compare with time.Now().Unix()

### Testing Approach
- Unit tests cover: valid token, expired, invalid signature, missing token, wrong issuer
- Use `scripts/generate-test-jwt.js` to create test tokens
- httptest.NewRecorder() + gin.CreateTestContext() for testing
```

**Metadata:**
```json
{
  "taskId": "agent-task-uuid",
  "agentName": "go-dev",
  "phase": 1,
  "completedAt": "2025-10-11T22:30:00Z",
  "relatedServices": ["staff-api"],
  "handoffTo": "ui-dev"
}
```

### 2. Qdrant Knowledge Entries

**Collection:** technical-knowledge

**Information:**
```markdown
## Go Gin JWT Middleware HS256 Token Validation

[Full pattern documentation with code examples]
```

**Metadata:**
```json
{
  "knowledgeType": "pattern",
  "domain": "backend",
  "title": "Go Gin JWT Middleware HS256 Validation",
  "tags": ["go", "jwt", "middleware", "authentication", "gin"],
  "linkedTaskId": "human-task-uuid"
}
```

### 3. priorWorkSummary Content

This goes directly into Phase 2 task's `priorWorkSummary` field:

```markdown
## Phase 1 Delivered: JWT Authentication Middleware

**Built:** JWT validation middleware for staff-api with HS256 algorithm

**API Contracts for Phase 2:**

**Authentication Header Format:**
```
Authorization: Bearer <jwt-token>
```

**Error Responses (Handle These):**
```json
// 401 - Expired token
{"error": "token_expired"}

// 401 - Invalid token
{"error": "invalid_token", "message": "signature is invalid"}

// 401 - Missing token
{"error": "missing_token"}
```

**Protected Endpoints (Require JWT):**
- GET/POST/PUT/DELETE `/api/v1/persons/*`
- GET/POST/PUT/DELETE `/api/v1/companies/*`

**Public Endpoints (No JWT Required):**
- GET `/health`
- GET `/metrics`
- POST `/api/v1/auth/login` (Phase 2 will create this)

**Files Created (DO NOT MODIFY - use only):**
- `staff-api/middleware/auth.go`: Exports `ValidateJWT(secret string) gin.HandlerFunc`
- Middleware automatically stores user ID in context
- Already integrated with Gin router for protected routes

**How Phase 2 Integrates:**
1. Create login endpoint: POST `/api/v1/auth/login` (returns JWT token)
2. Frontend stores token in localStorage
3. Frontend adds `Authorization: Bearer <token>` header to ALL protected API requests
4. Frontend catches 401 errors and redirects to login
5. Implement token refresh before expiration (token expires in 24h)

**Key Decisions Affecting Phase 2:**
- **Token Claims:** JWT contains `sub` (user ID), `exp` (expiration), `iss` (issuer), `aud` (audience)
- **Token Lifetime:** 24 hours (frontend should refresh at 23h)
- **Storage:** Use localStorage (not sessionStorage - persist across tabs)
- **Error Handling:** 401 = token invalid/expired, redirect to login

**Gotchas to Avoid:**
- Don't forget `Bearer ` prefix in Authorization header
- Handle 401 globally (axios interceptor) not per-request
- Clear localStorage on logout AND on repeated 401 errors

**Phase 2 Should NOT:**
- Modify Phase 1 middleware code (it's tested and working)
- Create different auth header format
- Store tokens in cookies (use localStorage)
- Implement custom JWT parsing (backend handles all validation)
```

### 4. Phase 2 Context Efficiency

**Estimated Context Budget:**
- Task context reading: 60 seconds (read priorWorkSummary)
- Coordinator knowledge query: NOT NEEDED (all info in priorWorkSummary)
- Qdrant pattern search: OPTIONAL (only if implementing custom login UI pattern)
- File reading before coding: 0 files (don't need to read Phase 1 code)
- **Total time to start coding:** 60-90 seconds ✅

**Efficiency Score:** High
**Reasoning:** All API contracts, error formats, and integration steps are in priorWorkSummary. Phase 2 agent can start implementing login form immediately without reading backend code.

### 5. Phase 2 Agent Instructions

**First Steps for Phase 2 Agent:**
1. Read task's `priorWorkSummary` field (contains all API contracts and error formats)
2. NO coordinator knowledge query needed (all info embedded)
3. NO Qdrant query needed unless implementing custom UI pattern
4. Start coding at: `ui/src/pages/LoginPage.tsx` (create login form)
5. DO NOT: Read or modify Phase 1 middleware code

**Files Phase 2 Will Create:**
- `ui/src/pages/LoginPage.tsx` (login form)
- `ui/src/hooks/useAuth.ts` (token management)
- `ui/src/utils/api.ts` (axios interceptor for auth header)
- `ui/src/contexts/AuthContext.tsx` (auth state management)

**Files Phase 2 Will NOT Touch:**
- `staff-api/middleware/auth.go` (Phase 1 code - working, don't modify)
- `staff-api/cmd/server/main.go` (Phase 1 integration - complete)

### 6. Validation Checklist

- [x] Phase 2 has exact Authorization header format (Bearer <token>)
- [x] Phase 2 knows all error response formats (401 errors with specific codes)
- [x] Integration instructions are explicit (5-step process with exact endpoints)
- [x] Gotchas documented (Bearer prefix, global 401 handling, localStorage)
- [x] Reusable pattern stored in Qdrant (JWT middleware pattern)
- [x] priorWorkSummary is complete (no need to read Phase 1 code)
- [x] Phase 2 can start in <2 minutes (60-90 seconds estimated)
```

**Common Pitfalls:**

- ❌ **Vague API Contracts:** "Middleware validates tokens" → ✅ Exact function signatures and error formats
- ❌ **Force Code Reading:** Phase 2 must read Phase 1 code → ✅ All contracts in priorWorkSummary
- ❌ **Missing Gotchas:** Don't share pitfalls discovered → ✅ Document non-obvious issues
- ❌ **No Integration Steps:** "Use the middleware" → ✅ Step-by-step how to integrate

---

## Documentation Prompts

### 7. build_knowledge_base

**Purpose:** Analyze source code to identify components (ADRs, APIs, MCPs, Services, Components) and generate structured documentation templates. Can auto-create agent tasks for specialists to document their domains.

**When to Use:**
- Starting knowledge base for new project
- Need comprehensive documentation across codebase
- Want to auto-generate documentation tasks
- Identifying what needs to be documented

**Required Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `projectPath` | string | Absolute path to the project/codebase to analyze |

**Optional Arguments:**

| Argument | Type | Description |
|----------|------|-------------|
| `components` | string | Comma-separated component types (options: `adrs`, `apis`, `mcps`, `services`, `components`, `all`). Default: `all` |
| `generateTasks` | string | Whether to auto-create agent tasks (`true`/`false`). Default: `false` |

**Example Invocation:**

```typescript
const prompt = await mcp.prompts.get({
  name: "build_knowledge_base",
  arguments: {
    projectPath: "/Users/maxmednikov/MaxSpace/dev-squad",
    components: "services,apis,mcps",
    generateTasks: "true"
  }
});
```

**Example Response Structure:**

The AI will provide comprehensive codebase analysis:

```markdown
### 1. Services Identified

```json
{
  "services": [
    {
      "name": "staff-api",
      "path": "staff-api/",
      "port": 8081,
      "database": "staff_db",
      "apis": ["REST", "MCP"],
      "dependencies": ["MongoDB", "NATS"]
    },
    {
      "name": "tasks-api",
      "path": "tasks-api/",
      "port": 8082,
      "database": "tasks_db",
      "apis": ["REST", "MCP"],
      "dependencies": ["MongoDB", "NATS", "WebSocket"]
    }
  ]
}
```

### 2. API Endpoints Identified

```json
{
  "endpoints": [
    {
      "service": "staff-api",
      "method": "GET",
      "path": "/api/v1/persons",
      "auth": "JWT required",
      "description": "List all persons with pagination"
    },
    {
      "service": "tasks-api",
      "method": "POST",
      "path": "/api/v1/tasks",
      "auth": "JWT required",
      "description": "Create new task"
    }
  ]
}
```

### 3. MCP Tools Identified

```json
{
  "tools": [
    {
      "service": "tasks-api",
      "name": "task_create",
      "description": "Create a new task",
      "parameters": ["title", "description", "assignedTo", "priority"]
    },
    {
      "service": "staff-api",
      "name": "person_list",
      "description": "List all persons with filters",
      "parameters": ["companyId", "limit", "offset"]
    }
  ]
}
```

### 5. Documentation Plan

| Component | Priority | Template | Assignee | Est. Time | Dependencies |
|-----------|----------|----------|----------|-----------|--------------|
| staff-api service | HIGH | Service + API + MCP | go-dev + go-mcp-dev | 2-3h | None |
| tasks-api service | HIGH | Service + API + MCP | go-dev + go-mcp-dev | 3-4h | None |
| TaskBoard component | MEDIUM | React Component | ui-dev | 1h | tasks-api docs |
| JWT Auth ADR | HIGH | ADR | Backend Services Specialist | 1h | None |

### 6. Knowledge Base Structure

**technical-knowledge** ← Reusable patterns
- Entry: "Go Gin JWT Middleware Pattern"
- Entry: "NATS JetStream Subscription Pattern"
- Entry: "React Query Optimistic Updates"

**code-patterns** ← Specific implementations
- Entry: "MongoDB Aggregation for Task Search"
- Entry: "WebSocket Real-time Updates Pattern"

**adr** ← Architecture decisions
- Entry: "ADR-001: MongoDB vs PostgreSQL"
- Entry: "ADR-002: MCP for Agent Communication"

**data-contracts** ← API schemas
- Entry: "Staff API REST Contracts"
- Entry: "Tasks API MCP Tool Schemas"

**ui-component-patterns** ← Frontend patterns
- Entry: "Task Board Component with Radix UI"
- Entry: "Form Validation with React Hook Form"

---

**If generateTasks=true, AI will also provide:**

### Agent Tasks to Create

**Task 1: Document staff-api Service**
- Agent: go-dev
- Priority: HIGH
- Files: staff-api/CLAUDE.md, SERVICES.md
- TODOs: [Service overview, REST API docs, MongoDB schemas, NATS events, deployment config]

**Task 2: Document staff-api MCP Tools**
- Agent: go-mcp-dev
- Priority: HIGH
- Files: staff-api/MCP_TOOLS.md
- TODOs: [person_list tool, person_get tool, person_create tool, error handling, examples]

[Additional tasks for other services and components...]
```

**Common Pitfalls:**

- ❌ **Analyze Without Plan:** Generate docs randomly → ✅ Use AI guidance for structure
- ❌ **Skip Dependencies:** Document in wrong order → ✅ Follow dependency graph
- ❌ **Wrong Agent:** Assign all to one agent → ✅ Match expertise to component type
- ❌ **No Templates:** Freeform docs → ✅ Use standardized templates

---

## General Usage Guidelines

### When to Use Prompts vs Tools

**Use Prompts When:**
- Need AI-assisted decision making or planning
- Want structured recommendations
- Optimizing context usage
- Analyzing complex scenarios (cross-squad impact, handoffs)

**Use Tools When:**
- Executing concrete actions (create task, query knowledge, etc.)
- Storing or retrieving data
- Making changes to task status
- Searching knowledge base

### Prompt Response Handling

Prompts return `GetPromptResult` with `messages`:

```typescript
interface GetPromptResult {
  description?: string;
  messages: PromptMessage[];
}

interface PromptMessage {
  role: "user" | "assistant";
  content: TextContent | ImageContent | ResourceContent;
}
```

**Typical Workflow:**
1. Call prompt with arguments
2. Receive structured guidance in `messages[0].content.text`
3. Use AI (Claude/GPT) to process the guidance
4. Execute recommendations via tools

### Argument Validation

All prompts validate required arguments and return descriptive errors:

```
Error: "taskDescription and targetSquad are required arguments"
Error: "agentQuestion and taskContext are required arguments"
Error: "projectPath is a required argument"
```

**Best Practice:** Always provide required arguments to avoid validation errors.

### Error Handling

Prompts are **static template generators** - they don't query databases or external services. Errors are limited to:

- Missing required arguments
- Invalid JSON in context arguments
- Malformed comma-separated lists

**No Qdrant dependency:** Prompts work even if Qdrant is unavailable (they don't query it).

---

## Integration Examples

### Example 1: Using Prompts in Task Creation Workflow

```typescript
// Step 1: Use prompt to get guidance
const breakdownPrompt = await mcp.prompts.get({
  name: "plan_task_breakdown",
  arguments: {
    taskDescription: "Add CSV export to staff management",
    targetSquad: "go-dev"
  }
});

// Step 2: Send prompt to AI for task breakdown
const aiResponse = await ai.complete(breakdownPrompt.messages[0].content.text);

// Step 3: Parse AI response and create task
const taskPlan = JSON.parse(aiResponse);

const agentTask = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTask.taskId,
  agentName: "go-dev",
  role: "Implement CSV export with streaming",
  contextSummary: "...", // From AI response
  filesModified: taskPlan.filesModified,
  todos: taskPlan.todos
});
```

### Example 2: Optimizing Knowledge Queries

```typescript
// Step 1: Get query recommendations
const queryPrompt = await mcp.prompts.get({
  name: "recommend_qdrant_query",
  arguments: {
    agentQuestion: "How to implement streaming CSV export in Go?",
    taskContext: "Working on staff-api export feature using Gin framework",
    availableCollections: "technical-knowledge,code-patterns"
  }
});

// Step 2: Send to AI for recommendation
const recommendation = await ai.complete(queryPrompt.messages[0].content.text);

// Step 3: Execute recommended query
const results = await mcp__hyper__coordinator_query_knowledge({
  collection: recommendation.collection,
  query: recommendation.queryString,
  limit: 3
});
```

### Example 3: Multi-Phase Handoff

```typescript
// Phase 1 complete - prepare handoff
const handoffPrompt = await mcp.prompts.get({
  name: "suggest_handoff_strategy",
  arguments: {
    phase1Work: JSON.stringify({
      completed: "Backend CSV export API",
      filesCreated: ["staff-api/handlers/export_handler.go"],
      apiContracts: {
        endpoint: "GET /api/v1/export/persons",
        response: "text/csv stream"
      }
    }),
    phase2Scope: "Frontend export button with progress indicator",
    knowledgeGap: "Frontend doesn't know response format and headers"
  }
});

// Get handoff strategy from AI
const strategy = await ai.complete(handoffPrompt.messages[0].content.text);

// Store in coordinator knowledge
await mcp__hyper__coordinator_upsert_knowledge({
  collection: `task:hyperion://task/human/${humanTaskId}`,
  text: strategy.coordinatorKnowledge,
  metadata: strategy.metadata
});

// Create Phase 2 task with priorWorkSummary
const phase2Task = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTaskId,
  agentName: "ui-dev",
  role: "Add export button to persons page",
  priorWorkSummary: strategy.priorWorkSummary, // From prompt
  contextSummary: "...",
  filesModified: ["ui/src/pages/PersonsPage.tsx"],
  todos: [...]
});
```

---

## Performance Characteristics

| Prompt | Arguments Complexity | Response Size | Use Frequency |
|--------|---------------------|---------------|---------------|
| plan_task_breakdown | Low | Large (2-3 KB) | High (every task) |
| suggest_context_offload | Medium | Large (3-4 KB) | Medium (complex tasks) |
| recommend_qdrant_query | Low | Medium (1-2 KB) | High (when querying) |
| suggest_knowledge_structure | Medium (JSON) | Large (2-3 KB) | High (post-work) |
| detect_cross_squad_impact | Medium | Large (2-4 KB) | Medium (shared code) |
| suggest_handoff_strategy | High (JSON) | Very Large (4-6 KB) | Low (multi-phase) |
| build_knowledge_base | Low | Very Large (5-10 KB) | Very Low (once per project) |

**Best Practice:** Use prompts during planning/coordination phases, not during hot loops or frequent operations.

---

## Troubleshooting

### Common Issues

**Issue:** "Required arguments are missing"
- **Cause:** Missing required argument in prompt call
- **Solution:** Check prompt documentation for required arguments
- **Example:** `recommend_qdrant_query` requires both `agentQuestion` AND `taskContext`

**Issue:** "Invalid context JSON"
- **Cause:** `suggest_knowledge_structure` received malformed JSON in `context` argument
- **Solution:** Ensure JSON.stringify() when passing objects
- **Example:** `context: JSON.stringify({ squad: "go-dev", service: "staff-api" })`

**Issue:** Prompt response is too generic
- **Cause:** Insufficient context in arguments
- **Solution:** Provide detailed, specific information in arguments
- **Example:** Instead of "add auth", use "implement JWT middleware with HS256 validation and error handling"

**Issue:** AI doesn't follow prompt structure
- **Cause:** Prompt template may need tuning, or AI model limitations
- **Solution:** Re-run with more specific instructions, or manually structure the response
- **Example:** Explicitly ask for JSON output format in follow-up

---

## Future Enhancements

**Potential Improvements:**
- Add dynamic Qdrant queries to prompts (e.g., auto-search for similar tasks)
- Caching frequently used prompt responses
- Prompt composition (chain multiple prompts)
- Custom prompt templates (user-defined)
- Metrics on prompt effectiveness (how often recommendations followed)

**Feedback:**
If prompts aren't providing helpful guidance, document issues in `technical-debt-registry` collection with tag `mcp-prompts-improvement`.

---

## Conclusion

MCP Prompts are **AI assistants for coordination tasks**, not automation tools. They provide structured guidance to:

- **Reduce planning time** (task breakdowns in minutes, not hours)
- **Improve handoff quality** (Phase 2 starts in <2 min)
- **Prevent conflicts** (detect cross-squad impact early)
- **Optimize queries** (1 targeted query vs 5 exploratory)
- **Standardize documentation** (consistent knowledge structure)

**Remember:** Prompts generate recommendations. Tools execute actions. Use both together for maximum efficiency.

**Version:** 1.0.0 | **Last Updated:** 2025-10-11 | **Maintained by:** go-mcp-dev squad
