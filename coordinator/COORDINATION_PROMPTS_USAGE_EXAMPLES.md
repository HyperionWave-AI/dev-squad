# Coordination Prompts - Usage Examples

## Overview

This document provides practical examples of using the Phase 3 coordination prompts to detect cross-squad impacts and plan smooth handoffs.

## Example 1: Detecting Cross-Squad Impact for API Change

### Scenario
Backend team wants to add pagination to the task list endpoint, which will affect the frontend team.

### Usage

```javascript
// Workflow coordinator detects impact before task creation
const impactAnalysis = await mcp.getPrompt({
  name: "detect_cross_squad_impact",
  arguments: {
    taskDescription: "Add pagination support to GET /api/v1/tasks endpoint with limit and offset parameters",
    filesModified: "coordinator/handlers/tasks.go,coordinator/models/task.go,frontend/src/api/tasks.ts",
    activeSquads: "backend-services,ui-dev,ui-tester"
  }
});

// AI analyzes the prompt and returns:
{
  affectedSquads: [
    {
      squad: "backend-services",
      impactType: "API Contract",
      impactLevel: "Breaking",
      reason: "Changing response structure from array to paginated object"
    },
    {
      squad: "ui-dev",
      impactType: "API Contract",
      impactLevel: "Breaking",
      reason: "Frontend code expects direct array, needs update for pagination"
    },
    {
      squad: "ui-tester",
      impactType: "API Contract",
      impactLevel: "Non-Breaking",
      reason: "Tests may need updates but can maintain compatibility"
    }
  ],

  requiredCommunication: {
    what: "API response structure changing from direct array to {items: [], total: N, offset: X}",
    how: {
      qdrantCollection: "team-coordination",
      metadata: {type: "api-breaking-change", endpoint: "/api/v1/tasks"},
      notificationMethod: "blocking-task-creation"
    },
    when: {
      beforeImplementation: true,
      reason: "ui-dev has active sprint work on task list UI"
    }
  },

  coordinationActions: {
    immediate: [
      "Post API contract change to team-coordination collection",
      "Query if ui-dev has conflicting work on GET /api/v1/tasks",
      "Wait for ui-dev acknowledgment before proceeding"
    ],
    duringImplementation: [
      "Update TODO notes when final pagination schema is decided",
      "Store pagination pattern in code-patterns collection"
    ],
    postCompletion: [
      "Document migration guide for ui-dev",
      "Update API documentation with pagination examples"
    ]
  },

  riskAssessment: {
    conflictRisk: "High",
    reasoning: "ui-dev currently working on task list filtering feature",
    mitigation: "Coordinate with ui-dev to merge changes or sequence work"
  },

  recommendedQueries: [
    {
      query: "pagination pattern implementation API response",
      purpose: "Check if pagination pattern already exists",
      collection: "code-patterns"
    },
    {
      query: "task list endpoint ui-dev active work",
      purpose: "Verify if ui-dev has conflicting changes",
      collection: "team-coordination"
    }
  ]
}
```

### Actions Taken

1. **Coordinator posts to team-coordination**:
```javascript
await qdrant.store({
  collection_name: "team-coordination",
  information: `
    BREAKING CHANGE NOTICE
    Endpoint: GET /api/v1/tasks
    Change: Adding pagination (limit/offset parameters)
    Response: Array → {items: [], total: N, offset: X}
    Affected Squads: ui-dev, ui-tester
    Timeline: Implementation starting today

    ui-dev: Please acknowledge and coordinate timeline
  `,
  metadata: {
    type: "api-breaking-change",
    urgency: "blocking",
    endpoint: "/api/v1/tasks",
    affectedSquads: ["ui-dev", "ui-tester"],
    requestedBy: "backend-services"
  }
});
```

2. **ui-dev queries and responds**:
```javascript
// ui-dev finds the notice
const notices = await qdrant.find({
  collection_name: "team-coordination",
  query: "task list API changes blocking"
});

// ui-dev responds
await qdrant.store({
  collection_name: "team-coordination",
  information: `
    ACKNOWLEDGMENT: GET /api/v1/tasks pagination
    Status: Can accommodate change
    Action: Will update TaskListComponent after backend deploys
    Timeline: Ready to start when API is deployed
    Testing: Will verify with new response format
  `,
  metadata: {
    type: "coordination-response",
    relatedTo: "api-breaking-change-tasks-endpoint",
    squad: "ui-dev"
  }
});
```

3. **backend-services proceeds** with task creation, knowing coordination is complete.

---

## Example 2: Multi-Phase Handoff (Backend → Frontend)

### Scenario
Phase 1: Backend builds authentication API
Phase 2: Frontend builds login UI using the API

### Phase 1 Completion

Backend agent completes work and reports:

```javascript
const phase1Summary = {
  completed: "JWT-based authentication API with login, refresh, and logout endpoints",
  files: [
    "coordinator/handlers/auth.go (lines 1-250): Authentication handlers",
    "coordinator/middleware/jwt.go (lines 1-180): JWT validation middleware",
    "coordinator/models/user.go (lines 1-100): User and token models"
  ],
  apiContracts: {
    login: {
      endpoint: "POST /api/v1/auth/login",
      request: {email: "string", password: "string"},
      response: {accessToken: "string", refreshToken: "string", expiresIn: 3600},
      errors: {401: "Invalid credentials", 422: "Validation failed"}
    },
    refresh: {
      endpoint: "POST /api/v1/auth/refresh",
      request: {refreshToken: "string"},
      response: {accessToken: "string", expiresIn: 3600},
      errors: {401: "Invalid or expired refresh token"}
    },
    logout: {
      endpoint: "POST /api/v1/auth/logout",
      headers: {Authorization: "Bearer {accessToken}"},
      response: {success: true},
      errors: {401: "Unauthorized"}
    }
  },
  keyDecisions: [
    "JWT tokens stored in HTTP-only cookies for security",
    "Access token expiry: 1 hour, refresh token: 7 days",
    "Password hashing: bcrypt with cost factor 12"
  ],
  gotchas: [
    "Refresh endpoint requires old refresh token, not access token",
    "Logout must clear both access and refresh cookies",
    "Email validation is case-insensitive"
  ],
  testingApproach: "Integration tests with real JWT validation, unit tests for handlers"
};
```

### Planning Phase 2 Handoff

Coordinator uses handoff strategy prompt:

```javascript
const handoffStrategy = await mcp.getPrompt({
  name: "suggest_handoff_strategy",
  arguments: {
    phase1Work: JSON.stringify(phase1Summary),
    phase2Scope: "Build React login/logout UI with JWT token management and session handling",
    knowledgeGap: "How to call auth API, where to store tokens, how to handle token refresh, error handling patterns"
  }
});

// AI returns handoff strategy:
{
  coordinatorKnowledge: {
    collection: "task:hyperion://task/human/AUTH-123",
    text: `
## Phase 1 Completion: Authentication API

### API Contracts for Phase 2

**Login Flow:**
1. POST /api/v1/auth/login
   Request: {email: "user@example.com", password: "secret"}
   Response: {accessToken: "jwt...", refreshToken: "refresh...", expiresIn: 3600}

**Token Refresh:**
2. POST /api/v1/auth/refresh
   Request: {refreshToken: "refresh..."}
   Response: {accessToken: "jwt...", expiresIn: 3600}

**Logout:**
3. POST /api/v1/auth/logout
   Headers: {Authorization: "Bearer {accessToken}"}
   Response: {success: true}

### Files Created (DO NOT MODIFY)
- coordinator/handlers/auth.go: Exports LoginHandler, RefreshHandler, LogoutHandler
- coordinator/middleware/jwt.go: Exports JWTMiddleware for protected routes
- coordinator/models/user.go: User and AuthResponse structs

### Integration Instructions for Phase 2
1. Import fetch or axios in React
2. Store tokens in HTTP-only cookies (backend sets them)
3. Include Authorization header for protected routes
4. Implement auto-refresh before token expires

### Key Decisions
- Tokens in HTTP-only cookies (secure, no localStorage)
- Access token TTL: 1 hour (refresh before expiry)
- Refresh token TTL: 7 days (re-login after)

### Gotchas Discovered
- Refresh endpoint needs refreshToken from cookie, not header
- Logout clears both cookies server-side
- Email is case-insensitive (lowercase before sending)

### Testing Approach
- Mock API responses in frontend tests
- Test token expiry and refresh flow
- Test error states (invalid credentials, expired tokens)
    `,
    metadata: {
      taskId: "AUTH-123",
      agentName: "backend-services",
      phase: 1,
      completedAt: "2025-01-15T10:00:00Z"
    }
  },

  qdrantKnowledge: [
    {
      collection: "code-patterns",
      information: `
# JWT Authentication Pattern (React + Go)

## Backend Setup (coordinator/handlers/auth.go)
\`\`\`go
func LoginHandler(c *gin.Context) {
    var req LoginRequest
    c.BindJSON(&req)

    user, err := authenticateUser(req.Email, req.Password)
    if err != nil {
        c.JSON(401, gin.H{"error": "Invalid credentials"})
        return
    }

    accessToken := generateJWT(user.ID, 1*time.Hour)
    refreshToken := generateRefreshToken(user.ID, 7*24*time.Hour)

    c.SetCookie("access_token", accessToken, 3600, "/", "", true, true)
    c.SetCookie("refresh_token", refreshToken, 604800, "/", "", true, true)

    c.JSON(200, AuthResponse{AccessToken: accessToken, ExpiresIn: 3600})
}
\`\`\`

## Frontend Usage (React)
\`\`\`typescript
// Login
const login = async (email: string, password: string) => {
    const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({email: email.toLowerCase(), password}),
        credentials: 'include' // Important: include cookies
    });

    if (!response.ok) {
        throw new Error('Invalid credentials');
    }

    const {accessToken, expiresIn} = await response.json();
    // Token is in HTTP-only cookie, also returned for client state
    scheduleTokenRefresh(expiresIn);
};

// Auto-refresh before expiry
const scheduleTokenRefresh = (expiresIn: number) => {
    setTimeout(async () => {
        await fetch('/api/v1/auth/refresh', {
            method: 'POST',
            credentials: 'include'
        });
    }, (expiresIn - 60) * 1000); // Refresh 1 min before expiry
};

// Protected API calls
const fetchProtectedData = async () => {
    const response = await fetch('/api/v1/protected', {
        credentials: 'include', // Sends cookies automatically
        headers: {
            'Authorization': \`Bearer \${accessToken}\` // If token also in state
        }
    });
};
\`\`\`

## Key Points
- Always use credentials: 'include' for cookie-based auth
- Backend sets HTTP-only cookies (secure, no XSS)
- Client schedules refresh before expiry
- Email lowercase before sending (backend is case-insensitive)
      `,
      metadata: {
        knowledgeType: "pattern",
        domain: "authentication",
        tags: ["jwt", "react", "golang", "cookies"],
        linkedTaskId: "AUTH-123"
      }
    }
  ],

  priorWorkSummary: `
## Phase 1: Authentication API (COMPLETE)

**Built**: JWT-based auth with login, refresh, logout endpoints

**API Contracts for Phase 2**:
- Login: POST /api/v1/auth/login → {accessToken, refreshToken, expiresIn}
- Refresh: POST /api/v1/auth/refresh → {accessToken, expiresIn}
- Logout: POST /api/v1/auth/logout → {success: true}

**Files Created** (consume only, don't modify):
- coordinator/handlers/auth.go: LoginHandler, RefreshHandler, LogoutHandler
- coordinator/middleware/jwt.go: JWTMiddleware for protected routes

**How Phase 2 Integrates**:
1. Use fetch with credentials: 'include' (cookies)
2. Tokens auto-managed in HTTP-only cookies
3. Schedule refresh 1 min before token expiry
4. Include Authorization: Bearer header for protected routes

**Key Decisions**:
- Tokens in HTTP-only cookies (not localStorage - security)
- Access token: 1hr, Refresh token: 7 days
- Email is case-insensitive (lowercase before sending)

**Gotchas**:
- Refresh endpoint uses refreshToken from cookie, not Authorization header
- Logout clears both cookies server-side
- Always use credentials: 'include' in fetch

**Phase 2 Should NOT**:
- Re-implement authentication logic
- Store tokens in localStorage (insecure)
- Modify Phase 1 backend files
  `,

  phase2ContextEfficiency: {
    taskContextReading: "1 minute (read priorWorkSummary)",
    coordinatorKnowledgeQuery: "not needed (all in priorWorkSummary)",
    qdrantPatternSearch: "30 seconds (React JWT pattern from code-patterns)",
    fileReading: "0 files (all integration details in summary)",
    totalTimeToStartCoding: "1.5 minutes",
    efficiencyScore: "High"
  },

  phase2AgentInstructions: {
    firstSteps: [
      "Read task's priorWorkSummary (contains all API contracts)",
      "Query code-patterns for 'React JWT authentication pattern'",
      "Start at: frontend/src/auth/AuthContext.tsx",
      "Implement: login(), logout(), auto-refresh logic",
      "DO NOT: Read backend code, modify auth.go files"
    ],

    filesWillModify: [
      "frontend/src/auth/AuthContext.tsx (new, auth context provider)",
      "frontend/src/components/LoginForm.tsx (new, login UI)",
      "frontend/src/hooks/useAuth.ts (new, auth hook)",
      "frontend/src/api/authClient.ts (new, API client)"
    ],

    filesReferenceOnly: [
      "coordinator/handlers/auth.go (API implementation, don't modify)",
      "coordinator/models/user.go (Response types, don't modify)"
    ]
  },

  validationChecklist: {
    phase2HasAPIContracts: true,
    integrationInstructionsExplicit: true,
    gotchasDocumented: true,
    reusablePatternsInQdrant: true,
    priorWorkSummaryComplete: true,
    phase2CanStartIn2Minutes: true
  }
}
```

### Creating Phase 2 Task

Coordinator creates Phase 2 task with handoff context:

```javascript
const phase2Task = await coordinator.createAgentTask({
  agentName: "ui-dev",
  humanTaskId: "AUTH-123",

  role: "Build React login/logout UI with JWT token management",

  contextSummary: `
Build React authentication UI that integrates with the completed backend JWT API.

**Business Context**: Enable users to securely login/logout with JWT-based session management.

**Technical Approach**:
- React Context for auth state
- Fetch API with credentials: 'include' for cookie-based tokens
- Auto-refresh tokens 1 min before expiry
- Error handling for invalid credentials and expired sessions

**Integration Points**:
- Backend: coordinator/handlers/auth.go (LoginHandler, RefreshHandler, LogoutHandler)
- API: POST /login, POST /refresh, POST /logout (see priorWorkSummary)
- Token storage: HTTP-only cookies (backend manages)

**Constraints**:
- Must use cookies, not localStorage (security requirement)
- Must handle auto-refresh to prevent session interruption
- Email must be lowercase before API call

**Testing**:
- Mock API responses in component tests
- Test token refresh flow
- Test error states (401, network errors)
  `,

  filesModified: [
    "frontend/src/auth/AuthContext.tsx",
    "frontend/src/components/LoginForm.tsx",
    "frontend/src/hooks/useAuth.ts",
    "frontend/src/api/authClient.ts"
  ],

  qdrantCollections: ["code-patterns"],

  priorWorkSummary: `
## Phase 1: Authentication API (COMPLETE)

**API Contracts**:
- Login: POST /api/v1/auth/login
  Request: {email, password}
  Response: {accessToken, refreshToken, expiresIn: 3600}

- Refresh: POST /api/v1/auth/refresh
  Request: {refreshToken} (from cookie)
  Response: {accessToken, expiresIn: 3600}

- Logout: POST /api/v1/auth/logout
  Headers: Authorization: Bearer {token}
  Response: {success: true}

**Integration**:
- Tokens in HTTP-only cookies (backend sets them)
- Use fetch with credentials: 'include'
- Auto-refresh 1 min before token expiry
- Include Authorization header for protected routes

**Gotchas**:
- Email must be lowercase
- Refresh uses cookie, not Authorization header
- Logout clears both cookies server-side
  `,

  notes: `
**Critical**:
- All tokens in HTTP-only cookies (not localStorage)
- Always use credentials: 'include' in fetch
- Query code-patterns for React JWT pattern before starting
  `,

  todos: [
    {
      description: "Create AuthContext with login/logout/refresh logic",
      filePath: "frontend/src/auth/AuthContext.tsx",
      functionName: "AuthProvider",
      contextHint: `
Create React context with authentication state and methods.

State: {user: User|null, isAuthenticated: boolean, loading: boolean}

Methods:
- login(email, password): Calls POST /api/v1/auth/login with credentials: 'include'
- logout(): Calls POST /api/v1/auth/logout
- refresh(): Calls POST /api/v1/auth/refresh (scheduled auto-refresh)

Auto-refresh: Schedule refresh 60 seconds before token expiry using setTimeout.

Error handling: Catch 401 (invalid credentials), 422 (validation), network errors.

See code-patterns collection for React JWT authentication pattern.
      `
    },
    {
      description: "Build LoginForm component with email/password inputs",
      filePath: "frontend/src/components/LoginForm.tsx",
      functionName: "LoginForm",
      contextHint: `
Form with email (lowercase), password inputs, and submit button.

On submit:
1. Convert email to lowercase
2. Call authContext.login(email, password)
3. Show loading state during API call
4. Handle errors: display "Invalid credentials" for 401

Use controlled inputs with React state.
      `
    },
    {
      description: "Create useAuth hook for consuming auth context",
      filePath: "frontend/src/hooks/useAuth.ts",
      functionName: "useAuth",
      contextHint: `
Export hook that returns useContext(AuthContext).

Throw error if used outside AuthProvider.

Return type: {user, isAuthenticated, loading, login, logout, refresh}
      `
    },
    {
      description: "Implement authClient for API calls with auto token refresh",
      filePath: "frontend/src/api/authClient.ts",
      functionName: "authFetch",
      contextHint: `
Wrapper around fetch that:
1. Always includes credentials: 'include'
2. Adds Authorization: Bearer header if token in state
3. Retries with token refresh on 401
4. Returns typed responses

Pattern: Try request → If 401, refresh token → Retry request once
      `
    }
  ]
});
```

### Phase 2 Agent Starts Work

**ui-dev agent receives task and:**

1. **Reads priorWorkSummary** (1 min): Gets all API contracts, integration details
2. **Queries code-patterns** (30 sec): Finds React JWT authentication pattern
3. **Starts coding** (90 seconds from task receipt):
   - Opens `frontend/src/auth/AuthContext.tsx`
   - Implements based on contextHints and pattern from Qdrant
   - No need to read backend code - everything in summary

**Total time to first line of code: <2 minutes ✅**

---

## Benefits Demonstrated

### Example 1 (Cross-Squad Impact):
- **Prevented conflict**: Detected ui-dev's active work before backend changed API
- **Saved time**: 2-4 hours of rework avoided through upfront coordination
- **Clear communication**: Team-coordination post made impact visible

### Example 2 (Multi-Phase Handoff):
- **Preserved context**: Phase 2 agent had 100% of needed information
- **Fast startup**: <2 minute context discovery vs typical 15-20 minutes
- **No code reading**: API contracts in summary, no need to read Phase 1 files
- **Pattern reuse**: Qdrant pattern saved writing boilerplate

## Summary

The coordination prompts enable:
1. **Proactive conflict detection** before work starts
2. **Context-rich handoffs** that eliminate exploration
3. **Knowledge preservation** across phases
4. **Parallel work** with minimal blocking

Use these prompts in **every multi-squad task** and **every multi-phase task** to achieve the target efficiency gains.
