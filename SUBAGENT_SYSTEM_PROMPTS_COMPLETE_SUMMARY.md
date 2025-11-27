# Hyperion Subagent System Prompts - Complete Summary

## 📍 System Prompt Locations

### Core Files

| Component | Location | Purpose |
|-----------|----------|---------|
| **Base Prompt** | `/Users/meghaneelamana/dev-squad/hyper/internal/mcp/storage/subagent_prompts.go` | Shared base prompt (~3,500 lines) used by all 16 agents |
| **Agent Definitions** | `/Users/meghaneelamana/dev-squad/hyper/internal/mcp/storage/subchat_storage.go` | Defines all 16 agents with their specialization prompts |
| **API Handler** | `/Users/meghaneelamana/dev-squad/hyper/internal/handlers/subagent_handler.go` | REST API endpoints for subagent operations |
| **Frontend UI** | `/Users/meghaneelamana/dev-squad/ui/src/components/organisms/SessionList.tsx` | "View Agents" button and modal UI |
| **Frontend Service** | `/Users/meghaneelamana/dev-squad/ui/src/services/subagentsService.ts` | API client for subagent operations |
| **Coordinator Prompt** | `/Users/meghaneelamana/dev-squad/CLAUDE.md` | Main coordinator prompt (orchestrator) |

### Agent-Specific Prompts

Located in: `/Users/meghaneelamana/dev-squad/.claude/agents/`

```
.claude/agents/
├── go-dev.md                              (Go Backend Development)
├── go-mcp-dev.md                          (Go MCP Development)
├── backend-services-specialist.md         (Backend Services)
├── event-systems-specialist.md            (Event Systems)
├── data-platform-specialist.md            (Data Platform)
├── ui-dev.md                              (Frontend Development)
├── ui-tester.md                           (Frontend Testing)
├── frontend-experience-specialist.md      (Frontend UX)
├── ai-integration-specialist.md           (AI Integration)
├── real-time-systems-specialist.md        (Real-time Systems)
├── sre.md                                 (Site Reliability Engineering)
├── k8s-deployment-expert.md               (Kubernetes)
├── infrastructure-automation-specialist.md (Infrastructure)
├── observability-specialist.md            (Observability)
├── security-auth-specialist.md            (Security & Auth)
└── e2e-testing-coordinator.md             (E2E Testing)
```

---

## 🤖 The 16 Available Subagents

### Backend Specialists (5 agents)

1. **go-dev**
   - Focus: Go backend development, clean code, testing
   - Key Rules: 90%+ test coverage, no god classes (>800 lines), service container pattern
   - Tools: Full access to all MCP tools
   - File: `.claude/agents/go-dev.md`

2. **go-mcp-dev**
   - Focus: MCP server development in Go
   - Key Rules: Strict MCP protocol compliance, tool schema validation
   - Tools: Full access to all MCP tools
   - File: `.claude/agents/go-mcp-dev.md`

3. **Backend Services Specialist**
   - Focus: Microservices architecture, API design
   - Key Rules: RESTful principles, service isolation, API versioning
   - Tools: Full access
   - File: `.claude/agents/backend-services-specialist.md`

4. **Event Systems Specialist**
   - Focus: Event-driven architecture, message queues, event sourcing
   - Key Rules: Event schema validation, idempotency, ordering guarantees
   - Tools: Full access
   - File: `.claude/agents/event-systems-specialist.md`

5. **Data Platform Specialist**
   - Focus: MongoDB, data modeling, query optimization
   - Key Rules: Index optimization, aggregation pipelines, migration safety
   - Tools: Full access
   - File: `.claude/agents/data-platform-specialist.md`

### Frontend Specialists (3 agents)

6. **ui-dev**
   - Focus: React/TypeScript frontend development
   - Key Rules: Component composition, hooks best practices, performance optimization
   - Tools: Full access to all MCP tools
   - File: `.claude/agents/ui-dev.md`

7. **ui-tester**
   - Focus: Frontend testing, component testing, E2E testing
   - Key Rules: Test coverage requirements, accessibility testing
   - Tools: Full access
   - File: `.claude/agents/ui-tester.md`

8. **Frontend Experience Specialist**
   - Focus: UX/UI design implementation, accessibility, performance
   - Key Rules: WCAG compliance, performance budgets, responsive design
   - Tools: Full access
   - File: `.claude/agents/frontend-experience-specialist.md`

### AI Integration Specialist (1 agent)

9. **AI Integration Specialist**
   - Focus: LLM integration, prompt engineering, AI workflows
   - Key Rules: Prompt optimization, token efficiency, safety guardrails
   - Tools: Full access to AI/LLM tools
   - File: `.claude/agents/ai-integration-specialist.md`

### Infrastructure & DevOps Specialists (5 agents)

10. **Real-time Systems Specialist**
    - Focus: WebSockets, real-time communication, low-latency systems
    - Key Rules: Connection pooling, message ordering, backpressure handling
    - Tools: Full access
    - File: `.claude/agents/real-time-systems-specialist.md`

11. **sre** (Site Reliability Engineering)
    - Focus: Deployment, monitoring, reliability, operational excellence
    - Key Rules: Canary deployments, health checks, incident response
    - Tools: Full access to deployment and monitoring tools
    - File: `.claude/agents/sre.md`

12. **k8s-deployment-expert**
    - Focus: Kubernetes deployment, orchestration, scaling
    - Key Rules: Resource limits, health probes, rolling updates
    - Tools: Full access to K8s tools
    - File: `.claude/agents/k8s-deployment-expert.md`

13. **Infrastructure Automation Specialist**
    - Focus: Infrastructure as Code, automation, provisioning
    - Key Rules: Idempotency, state management, disaster recovery
    - Tools: Full access to infrastructure tools
    - File: `.claude/agents/infrastructure-automation-specialist.md`

14. **Observability Specialist**
    - Focus: Logging, metrics, tracing, monitoring
    - Key Rules: Structured logging, metric cardinality, trace sampling
    - Tools: Full access to observability tools
    - File: `.claude/agents/observability-specialist.md`

### Security Specialist (1 agent)

15. **Security & Auth Specialist**
    - Focus: Authentication, authorization, security best practices
    - Key Rules: JWT validation, RBAC implementation, secret management
    - Tools: Full access to security tools
    - File: `.claude/agents/security-auth-specialist.md`

### Testing Specialist (1 agent)

16. **End-to-End Testing Coordinator**
    - Focus: E2E testing, test automation, test infrastructure
    - Key Rules: Test isolation, flakiness prevention, test data management
    - Tools: Full access to testing tools
    - File: `.claude/agents/e2e-testing-coordinator.md`

---

## 📝 System Prompt Structure

### How Prompts Are Composed

Each agent's complete system prompt is built from:

```
Agent System Prompt = BaseSubagentPrompt + SpecializationPrompt
```

### BaseSubagentPrompt (~3,500 lines)

**Location**: `hyper/internal/mcp/storage/subagent_prompts.go`

**Contains**:

1. **Tool Usage Guidance** (500 lines)
   - Tool selection matrix
   - When to use which tool
   - Tool parameter validation
   - Error handling patterns

2. **Async Operations & Polling** (400 lines)
   - How to handle async tasks
   - Polling patterns
   - Timeout handling
   - Progress tracking

3. **File Path Guidance** (200 lines)
   - Unix paths only (no Windows paths)
   - Relative paths from project root
   - Path validation
   - Storage API integration

4. **Error Handling & Retry Logic** (300 lines)
   - Circuit breaker pattern
   - Exponential backoff
   - Idempotency
   - Fail-fast principle

5. **Code Modification Rules** (400 lines)
   - Surgical edit mode (minimal changes)
   - JSX/TSX safety
   - Syntax validation
   - Testing requirements

6. **Task Management** (300 lines)
   - Creating human tasks
   - Creating agent tasks
   - Updating task status
   - TODO tracking

7. **Storage Operations** (200 lines)
   - Hyperion file operations
   - Upload/download patterns
   - Metadata handling
   - URI formats

8. **Workflow Patterns** (300 lines)
   - Multi-step workflows
   - Dependency management
   - State tracking
   - Rollback procedures

### SpecializationPrompts (50-200 lines each)

Each agent has a specialized prompt that adds domain-specific guidance:

**Example: go-dev.md**
```
- 90%+ test coverage requirement
- No god classes (>800 lines)
- Service container pattern mandatory
- Interface-based design
- Dependency injection
- Error wrapping with context
- Structured logging
- Graceful shutdown
```

**Example: ui-dev.md**
```
- React hooks best practices
- Component composition patterns
- Performance optimization (memoization, lazy loading)
- TypeScript strict mode
- Accessibility (WCAG 2.1 AA)
- Responsive design
- CSS-in-JS or Tailwind
- Storybook documentation
```

**Example: sre.md**
```
- Canary deployments
- Blue-green deployments
- Health checks and readiness probes
- Monitoring and alerting
- Incident response procedures
- Runbooks
- Disaster recovery
- Capacity planning
```

---

## 🔄 How the "View Agents" Flow Works

### Step-by-Step Flow

1. **User Interface**
   - User opens chat session list page
   - Clicks "View Agents" button
   - Modal displays all 16 available agents

2. **Frontend Service Call**
   - `subagentsService.ts` calls `GET /api/v1/subagents`
   - Backend returns list of all 16 agents with metadata

3. **User Selection**
   - User clicks on an agent (e.g., "go-dev")
   - Frontend calls `POST /api/v1/subagents/go-dev/sessions`

4. **Backend Processing**
   - `subagent_handler.go` receives request
   - Creates new chat session in MongoDB
   - Sets `activeSubagentName: "go-dev"`
   - Loads agent's system prompt from `subchat_storage.go`

5. **Prompt Loading**
   - Backend retrieves BaseSubagentPrompt from `subagent_prompts.go`
   - Appends go-dev specialization prompt from `.claude/agents/go-dev.md`
   - Stores complete prompt in session context

6. **Chat Session**
   - Frontend switches to new session
   - User sends message
   - Backend uses agent's complete system prompt
   - Agent responds with specialized knowledge

7. **Session Persistence**
   - Session stored in MongoDB with:
     - `activeSubagentName`: "go-dev"
     - `systemPrompt`: Complete combined prompt
     - `createdAt`: Timestamp
     - `messages`: Chat history

---

## 🎯 Key Features & Behaviors

### Automatic Seeding

On startup, `EnsureSystemSubagents()` in `subchat_storage.go`:
- Creates all 16 agents if they don't exist
- Loads specialization prompts from `.claude/agents/` directory
- Stores in MongoDB for persistence
- Enables dynamic agent management

### Tool Access Control

Different agents have different tool access:

```
Backend Agents (go-dev, go-mcp-dev, etc.):
  - Full access to all MCP tools
  - Code modification tools
  - Testing tools
  - Storage tools

Frontend Agents (ui-dev, ui-tester, etc.):
  - Full access to all MCP tools
  - Browser automation tools
  - Component testing tools
  - Screenshot/recording tools

Infrastructure Agents (sre, k8s-deployment-expert, etc.):
  - Full access to deployment tools
  - Monitoring tools
  - Infrastructure tools
  - Limited code modification
```

### Dedicated Sessions

Each subagent chat:
- Gets its own MongoDB session
- Separate from coordinator session
- Maintains independent chat history
- Uses agent-specific system prompt
- Can be switched between agents

### Surgical Edit Mode

All agents follow strict rules:
- Make ONLY requested changes
- Preserve existing code style
- No refactoring unless asked
- No feature additions
- Minimal line changes
- JSX/TSX safety checks

### Circuit Breaker Protection

Prevents infinite loops:
- Tracks identical tool calls
- Stops after 3 identical calls in 5 attempts
- Forces approach change
- Prevents resource exhaustion

### Mandatory Testing

Go agents enforce:
- 90%+ code coverage
- No test stubs
- No god classes (>800 lines)
- Interface-based design
- Fail-fast principle

---

## 📊 System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Chat UI (React)                          │
│  SessionList.tsx - "View Agents" Button                     │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Frontend Service Layer                         │
│  subagentsService.ts - API Client                           │
│  - GET /api/v1/subagents (list all agents)                 │
│  - POST /api/v1/subagents/{name}/sessions (create session) │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Backend API Layer                              │
│  subagent_handler.go - REST Endpoints                       │
│  - ListSubagents()                                          │
│  - CreateSubagentSession()                                  │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Storage & Prompt Layer                         │
│  subchat_storage.go - Agent Management                      │
│  - EnsureSystemSubagents() (seeding)                        │
│  - GetSubagent(name)                                        │
│  - LoadSpecializationPrompt()                               │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Prompt Composition                             │
│  subagent_prompts.go - BaseSubagentPrompt (~3,500 lines)   │
│  +                                                          │
│  .claude/agents/{name}.md - SpecializationPrompt           │
│  =                                                          │
│  Complete Agent System Prompt                              │
└─────────────────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              MongoDB Persistence                            │
│  - Sessions collection (with activeSubagentName)            │
│  - Agents collection (with prompts)                         │
│  - Chat history                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔍 File Reference Guide

### Backend Files

**`hyper/internal/mcp/storage/subagent_prompts.go`**
- Contains: BaseSubagentPrompt constant (~3,500 lines)
- Used by: All 16 agents
- Key sections:
  - Tool usage guidance
  - Async operations
  - File path rules
  - Error handling
  - Code modification rules
  - Task management
  - Storage operations
  - Workflow patterns

**`hyper/internal/mcp/storage/subchat_storage.go`**
- Contains: Agent definitions and seeding logic
- Key functions:
  - `EnsureSystemSubagents()` - Creates all 16 agents on startup
  - `GetSubagent(name)` - Retrieves agent by name
  - `LoadSpecializationPrompt()` - Loads agent-specific prompt
- Agents defined with:
  - Name
  - Description
  - Specialization prompt file path
  - Tool access configuration

**`hyper/internal/handlers/subagent_handler.go`**
- Contains: REST API endpoints
- Key endpoints:
  - `GET /api/v1/subagents` - List all agents
  - `POST /api/v1/subagents/{name}/sessions` - Create session
  - `GET /api/v1/subagents/{name}` - Get agent details

### Frontend Files

**`ui/src/components/organisms/SessionList.tsx`**
- Contains: Chat session list UI
- Key features:
  - "View Agents" button
  - Agent modal display
  - Session creation on agent selection
  - Session switching

**`ui/src/services/subagentsService.ts`**
- Contains: API client for subagent operations
- Key methods:
  - `listSubagents()` - Fetch all agents
  - `createSubagentSession(name)` - Create new session
  - `getSubagent(name)` - Get agent details

### Agent Prompt Files

**`.claude/agents/` directory**
- 16 files, one per agent
- Each file contains specialization prompt (50-200 lines)
- Loaded dynamically on startup
- Appended to BaseSubagentPrompt

---

## 💡 Common Use Cases

### Adding a New Subagent

1. Create new file: `.claude/agents/new-agent.md`
2. Write specialization prompt (50-200 lines)
3. Add agent definition to `subchat_storage.go`:
   ```go
   {
       Name: "new-agent",
       Description: "...",
       SpecializationPromptPath: ".claude/agents/new-agent.md",
       ToolAccess: "*", // or specific tools
   }
   ```
4. Restart backend (triggers `EnsureSystemSubagents()`)
5. New agent appears in "View Agents" modal

### Modifying an Agent's Prompt

1. Edit `.claude/agents/{agent-name}.md`
2. Restart backend to reload
3. New sessions use updated prompt
4. Existing sessions keep old prompt

### Modifying BaseSubagentPrompt

1. Edit `hyper/internal/mcp/storage/subagent_prompts.go`
2. Update the `BaseSubagentPrompt` constant
3. Restart backend
4. All agents get updated base prompt

---

## 📚 Additional Resources

- **Coordinator Prompt**: `/Users/meghaneelamana/dev-squad/CLAUDE.md`
- **Project Root**: `/Users/meghaneelamana/dev-squad/`
- **Agent Directory**: `/Users/meghaneelamana/dev-squad/.claude/agents/`
- **Backend Source**: `/Users/meghaneelamana/dev-squad/hyper/internal/`
- **Frontend Source**: `/Users/meghaneelamana/dev-squad/ui/src/`

---

**Last Updated**: 2024
**Total Agents**: 16
**Base Prompt Size**: ~3,500 lines
**Specialization Prompts**: 50-200 lines each
