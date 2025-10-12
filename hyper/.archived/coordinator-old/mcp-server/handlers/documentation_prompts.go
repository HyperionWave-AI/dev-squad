package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DocumentationPromptHandler manages documentation generation prompts for knowledge base building
type DocumentationPromptHandler struct{}

// NewDocumentationPromptHandler creates a new documentation prompt handler
func NewDocumentationPromptHandler() *DocumentationPromptHandler {
	return &DocumentationPromptHandler{}
}

// RegisterDocumentationPrompts registers all documentation prompts with the MCP server
func (h *DocumentationPromptHandler) RegisterDocumentationPrompts(server *mcp.Server) error {
	// Register build_knowledge_base prompt
	if err := h.registerBuildKnowledgeBase(server); err != nil {
		return fmt.Errorf("failed to register build_knowledge_base prompt: %w", err)
	}

	return nil
}

// registerBuildKnowledgeBase registers the build_knowledge_base prompt
func (h *DocumentationPromptHandler) registerBuildKnowledgeBase(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "build_knowledge_base",
		Description: "Analyze source code to identify components (ADRs, APIs, MCPs, Services, Components) and generate structured documentation templates. Can auto-create agent tasks for specialists to document their domains.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "projectPath",
				Description: "Absolute path to the project/codebase to analyze",
				Required:    true,
			},
			{
				Name:        "components",
				Description: "Comma-separated list of component types to document (optional). Options: adrs, apis, mcps, services, components, all. Default: all",
				Required:    false,
			},
			{
				Name:        "generateTasks",
				Description: "Whether to auto-create agent tasks for documentation work (true/false). Default: false",
				Required:    false,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments
		projectPath := ""
		componentsStr := "all"
		generateTasksStr := "false"

		if req.Params != nil && req.Params.Arguments != nil {
			projectPath = req.Params.Arguments["projectPath"]
			if c := req.Params.Arguments["components"]; c != "" {
				componentsStr = c
			}
			if g := req.Params.Arguments["generateTasks"]; g != "" {
				generateTasksStr = g
			}
		}

		if projectPath == "" {
			return nil, fmt.Errorf("projectPath is a required argument")
		}

		// Parse comma-separated components
		var components []string
		if componentsStr != "" {
			parts := strings.Split(componentsStr, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					components = append(components, trimmed)
				}
			}
		}

		generateTasks := strings.ToLower(generateTasksStr) == "true"

		promptText := h.buildKnowledgeBasePrompt(projectPath, components, generateTasks)

		return &mcp.GetPromptResult{
			Description: "Knowledge base building guide with component analysis and documentation templates",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: promptText,
					},
				},
			},
		}, nil
	}

	server.AddPrompt(prompt, handler)
	return nil
}

// buildKnowledgeBasePrompt builds the comprehensive knowledge base building prompt
func (h *DocumentationPromptHandler) buildKnowledgeBasePrompt(projectPath string, components []string, generateTasks bool) string {
	// Determine which components to document
	componentFilter := "all"
	if len(components) > 0 && components[0] != "all" {
		componentFilter = strings.Join(components, ", ")
	}

	taskGenerationSection := ""
	if generateTasks {
		taskGenerationSection = `
## 🤖 Agent Task Generation

You will auto-create agent tasks for specialists to document their domains.

### Task Creation Strategy

For each component type identified:

1. **Create Human Task** (if not exists)
` + "```typescript" + `
const humanTask = await mcp__hyper__coordinator_create_human_task({
  prompt: "Build comprehensive knowledge base for [project-name]"
})
` + "```" + `

2. **Create Agent Tasks** for each domain
` + "```typescript" + `
// Example: Backend Services Documentation
const backendTask = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTask.taskId,
  agentName: "go-dev",
  role: "Document all backend services with API contracts, dependencies, and patterns",
  contextSummary: ` + "`" + `
    WHY: Knowledge base needs comprehensive backend service documentation
    WHAT: Document each Go microservice (staff-api, tasks-api, security-api, etc.)
    HOW: Use service templates with API schemas, dependencies, deployment configs
    CONSTRAINTS: Follow Hyperion documentation standards
    TESTING: Verify docs include all endpoints and auth requirements
  ` + "`" + `,
  filesModified: ["SERVICES.md", "service-name/CLAUDE.md"],
  todos: [
    {
      description: "Document staff-api service",
      filePath: "staff-api/CLAUDE.md",
      contextHint: "Use service template. Include: purpose, API endpoints, MongoDB schemas, dependencies, deployment config, testing approach"
    }
  ]
})
` + "```" + `

**Agent Assignment Matrix:**
- **Backend Services/APIs:** → go-dev
- **MCP Tools:** → go-mcp-dev
- **React Components:** → ui-dev
- **Architecture Decisions (ADRs):** → Backend Services Specialist
- **Data Contracts:** → Data Platform Specialist

### Task Structure Requirements

Each agent task MUST include:
- **contextSummary:** WHY, WHAT, HOW, CONSTRAINTS, TESTING (150-250 words)
- **filesModified:** Exact paths to documentation files
- **todos:** Specific components to document with contextHints
- **knowledgeCollections:** ["technical-knowledge"] for referencing existing patterns
`
	}

	return fmt.Sprintf(`# Knowledge Base Builder

## Project to Analyze
**Path:** %s
**Component Filter:** %s

## Your Mission

Analyze the codebase structure and generate a comprehensive documentation plan that covers:
1. **Architecture Decision Records (ADRs)** - Why certain technical choices were made
2. **API Interfaces** - REST endpoints, request/response schemas, authentication
3. **MCP Interfaces** - Model Context Protocol tools, parameters, examples
4. **Services** - Microservices, their purpose, dependencies, and APIs
5. **Components** - React components, hooks, utilities, patterns

## Phase 1: Codebase Analysis (10-15 minutes)

### Step 1: Identify Services and APIs

Scan for:
- **Go Services:** Directories with ` + "`cmd/server/main.go`" + ` or ` + "`internal/`" + ` structure
  - Look for: ` + "`handlers/`" + `, ` + "`services/`" + `, ` + "`models/`" + `, ` + "`repositories/`" + `
  - Identify: Service name, port, database, dependencies

- **REST APIs:** Look for HTTP handlers
  - Gin routes: ` + "`router.GET()`" + `, ` + "`router.POST()`" + `, etc.
  - Handler patterns: ` + "`*_handler.go`" + ` files
  - Extract: Endpoints, methods, auth requirements

- **MCP Tools:** Search for MCP implementations
  - Files: ` + "`*_mcp_handler.go`" + `, ` + "`mcp/handlers/`" + `
  - Pattern: ` + "`mcp.AddTool()`" + ` calls
  - Extract: Tool names, parameters, descriptions

### Step 2: Identify Components

Scan for:
- **React Components:** ` + "`.tsx`" + ` files in ` + "`ui/src/components/`" + `
  - Extract: Component name, props, usage patterns

- **React Hooks:** ` + "`use*.ts`" + ` files
  - Extract: Hook name, parameters, return value

- **Utilities:** ` + "`utils/`" + `, ` + "`lib/`" + ` directories
  - Extract: Function purposes, use cases

### Step 3: Identify Data Models

Scan for:
- **MongoDB Schemas:** Go structs with ` + "`bson:`" + ` tags
- **TypeScript Types:** ` + "`.ts`" + ` files with ` + "`interface`" + ` or ` + "`type`" + `
- **API Contracts:** Request/response types

### Step 4: Identify Architecture Decisions

Look for:
- Existing ADR documents (often in ` + "`/docs/adr/`" + `)
- Comments with "DECISION:", "WHY:", "PATTERN:"
- Configuration files showing architectural choices

## Phase 2: Documentation Structure Generation

%s

### Documentation Templates by Component Type

%s

## Phase 3: Knowledge Storage Strategy

### Collection Organization

**technical-knowledge** ← Reusable patterns
- MCP tool patterns
- API middleware patterns
- Error handling strategies
- Authentication flows

**code-patterns** ← Specific implementations
- React component examples
- Go service patterns
- Database query patterns

**adr** ← Architecture decisions
- Why MongoDB over PostgreSQL
- Why MCP over REST for certain operations
- Microservices communication patterns

**data-contracts** ← API schemas
- REST endpoint request/response formats
- Event schemas
- Database models

**ui-component-patterns** ← Frontend patterns
- Radix UI component usage
- React Query patterns
- Form validation approaches

### Storage Format

For each documented component, store in Qdrant:

` + "```typescript" + `
await mcp__hyper__coordinator_upsert_knowledge({
  collection: "[appropriate-collection]",
  text: ` + "`" + `
## [Component Title]

### Summary
[2-3 sentence overview with business context]

### Implementation
[Detailed implementation with code examples]

### Dependencies
[What this depends on]

### API/Interface
[Endpoints, methods, parameters]

### Testing
[How to test, edge cases]

### Gotchas
[Common pitfalls and solutions]
  ` + "`" + `,
  metadata: {
    knowledgeType: "service|api|component|adr",
    domain: "backend|frontend|infrastructure",
    title: "[Component Title]",
    tags: ["tag1", "tag2", ...],
    service: "[service-name]",
    filePath: "[file-path]"
  }
})
` + "```" + `

## Output Format

Provide a structured analysis in this format:

### 1. Services Identified
` + "```json" + `
{
  "services": [
    {
      "name": "staff-api",
      "path": "staff-api/",
      "port": 8081,
      "database": "staff_db",
      "apis": ["REST", "MCP"],
      "dependencies": ["MongoDB", "NATS"]
    }
  ]
}
` + "```" + `

### 2. API Endpoints Identified
` + "```json" + `
{
  "endpoints": [
    {
      "service": "staff-api",
      "method": "GET",
      "path": "/api/v1/persons",
      "auth": "JWT required",
      "description": "List all persons with pagination"
    }
  ]
}
` + "```" + `

### 3. MCP Tools Identified
` + "```json" + `
{
  "tools": [
    {
      "service": "tasks-api",
      "name": "task_create",
      "description": "Create a new task",
      "parameters": ["title", "description", "assignedTo"]
    }
  ]
}
` + "```" + `

### 4. Components Identified
` + "```json" + `
{
  "components": [
    {
      "name": "TaskBoard",
      "path": "ui/src/components/TaskBoard.tsx",
      "type": "React component",
      "props": ["tasks", "onTaskUpdate"]
    }
  ]
}
` + "```" + `

### 5. Documentation Plan

For each component type, provide:
- **Priority:** High/Medium/Low (based on usage/complexity)
- **Template:** Which documentation template to use
- **Assignee:** Which agent should document it
- **Estimated Time:** How long it will take
- **Dependencies:** What needs to be documented first

### 6. Knowledge Base Structure

Recommended Qdrant collections and what goes in each:
- **Collection Name:** [collection]
  - **Contents:** [what to store]
  - **Example Entries:** [sample titles]

## Quality Checklist

Before finalizing, verify:

- [ ] **Completeness:** All services, APIs, and components identified
- [ ] **Accuracy:** Endpoints, parameters, and schemas are correct
- [ ] **Prioritization:** Critical components marked as high priority
- [ ] **Templates:** Appropriate template selected for each component type
- [ ] **Assignments:** Right specialist agent assigned to each task
- [ ] **Dependencies:** Documentation order accounts for dependencies
- [ ] **Searchability:** Knowledge will be discoverable via semantic search

## Example Analysis

**Service:** tasks-api
**Location:** ` + "`/tasks-api`" + `
**Type:** Go microservice with REST + MCP
**Priority:** HIGH (core user-facing service)

**APIs Identified:**
- REST: ` + "`/api/v1/tasks`" + ` (CRUD operations)
- MCP: ` + "`task_create`" + `, ` + "`task_get`" + `, ` + "`task_list`" + `, ` + "`task_update`" + `, ` + "`task_delete`" + `

**Documentation Needed:**
1. Service overview (purpose, architecture)
2. REST API contracts (endpoints, auth, schemas)
3. MCP tool documentation (parameters, examples)
4. MongoDB schemas (Task, Comment models)
5. Integration points (NATS events, WebSocket)

**Template:** Use Service + API + MCP templates
**Assignee:** go-dev + go-mcp-dev
**Storage Collections:** technical-knowledge, code-patterns, data-contracts
**Estimated Time:** 2-3 hours

---

Now, analyze the project at **%s** and provide your comprehensive documentation plan:`, projectPath, componentFilter, taskGenerationSection, h.buildComponentTemplates(components), projectPath)
}

// buildComponentTemplates returns documentation templates for different component types
func (h *DocumentationPromptHandler) buildComponentTemplates(components []string) string {
	// If "all" or empty, include all templates
	includeAll := len(components) == 0 || (len(components) == 1 && components[0] == "all")

	componentSet := make(map[string]bool)
	for _, c := range components {
		componentSet[strings.ToLower(c)] = true
	}

	templates := ""

	// ADR Template
	if includeAll || componentSet["adrs"] {
		templates += `
### 📋 Architecture Decision Record (ADR) Template

Use this template for documenting architectural decisions:

` + "```markdown" + `
# ADR-NNN: [Decision Title]

**Date:** YYYY-MM-DD
**Status:** Proposed | Accepted | Deprecated | Superseded
**Deciders:** [List of people/teams involved]

## Context

What is the issue we're facing? What factors are driving this decision?
- Business requirement
- Technical constraint
- User need
- Performance requirement

## Decision

What is the change we're proposing/have agreed to?

Be specific and concrete. Include:
- Technology/pattern chosen
- Implementation approach
- Migration strategy (if applicable)

## Rationale

Why did we choose this option?
- Benefits gained
- Trade-offs accepted
- Alternatives considered (and why rejected)

## Consequences

### Positive
- What improvements does this bring?
- What problems does this solve?

### Negative
- What limitations does this introduce?
- What technical debt might this create?

### Neutral
- What changes in process/workflow?
- What new dependencies?

## Implementation

How will this be implemented?
- Services affected
- Code changes required
- Configuration changes
- Migration steps

## Validation

How do we know this decision is working?
- Success metrics
- Monitoring/alerting
- Review timeline
` + "```" + `

**Storage:**
` + "```typescript" + `
await mcp__hyper__coordinator_upsert_knowledge({
  collection: "adr",
  text: "[ADR content above]",
  metadata: {
    knowledgeType: "adr",
    title: "ADR-NNN: [Decision Title]",
    status: "accepted|proposed|deprecated",
    tags: ["architecture", "decision", "[domain]"],
    decidedAt: "YYYY-MM-DD"
  }
})
` + "```" + `
`
	}

	// API Template
	if includeAll || componentSet["apis"] {
		templates += `
### 🔌 API Documentation Template

Use this template for REST/GraphQL APIs:

` + "```markdown" + `
# [Service Name] API

## Overview

**Base URL:** ` + "`http://service:port/api/v1`" + `
**Authentication:** JWT Bearer token required
**Rate Limiting:** 1000 requests/hour per user

## Endpoints

### List Resources

**Endpoint:** ` + "`GET /resources`" + `
**Description:** Retrieve a paginated list of resources

**Query Parameters:**
- ` + "`limit`" + ` (number, optional): Max items to return (default: 100)
- ` + "`offset`" + ` (number, optional): Number of items to skip (default: 0)
- ` + "`status`" + ` (string, optional): Filter by status

**Request Example:**
` + "```bash" + `
GET /api/v1/resources?limit=10&status=active
Authorization: Bearer <jwt-token>
` + "```" + `

**Response (200 OK):**
` + "```json" + `
{
  "data": [
    {
      "id": "507f1f77bcf86cd799439011",
      "name": "Resource Name",
      "status": "active",
      "createdAt": "2025-01-15T10:30:00Z"
    }
  ],
  "total": 42,
  "limit": 10,
  "offset": 0
}
` + "```" + `

**Error Responses:**
- ` + "`401 Unauthorized`" + `: Invalid or missing JWT token
- ` + "`403 Forbidden`" + `: Insufficient permissions
- ` + "`500 Internal Server Error`" + `: Server error

### Create Resource

**Endpoint:** ` + "`POST /resources`" + `
**Description:** Create a new resource

**Request Body:**
` + "```json" + `
{
  "name": "Resource Name",
  "description": "Detailed description",
  "category": "category1"
}
` + "```" + `

**Response (201 Created):**
` + "```json" + `
{
  "id": "507f1f77bcf86cd799439011",
  "name": "Resource Name",
  "status": "active",
  "createdAt": "2025-01-15T10:30:00Z"
}
` + "```" + `

## Authentication

All endpoints require JWT authentication:

` + "```bash" + `
Authorization: Bearer <jwt-token>
` + "```" + `

**Token Payload:**
` + "```json" + `
{
  "sub": "user-id",
  "companyId": "company-id",
  "roles": ["admin", "user"]
}
` + "```" + `

## Error Handling

Standard error response format:

` + "```json" + `
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {
    "field": "validation error details"
  }
}
` + "```" + `

## Rate Limiting

- **Limit:** 1000 requests/hour per user
- **Headers:** ` + "`X-RateLimit-Remaining`" + `, ` + "`X-RateLimit-Reset`" + `
- **Response (429):** Too Many Requests
` + "```" + `

**Storage:**
` + "```typescript" + `
await mcp__hyper__coordinator_upsert_knowledge({
  collection: "data-contracts",
  text: "[API documentation above]",
  metadata: {
    knowledgeType: "api",
    service: "[service-name]",
    title: "[Service Name] API Documentation",
    tags: ["api", "rest", "[service-name]", "authentication"],
    endpoints: ["/resources", "/resources/:id"]
  }
})
` + "```" + `
`
	}

	// MCP Template
	if includeAll || componentSet["mcps"] {
		templates += `
### 🔧 MCP Tool Documentation Template

Use this template for Model Context Protocol tools:

` + "```markdown" + `
# [Service Name] MCP Tools

## Overview

**Service:** [service-name]
**MCP Endpoint:** ` + "`http://service:port/mcp`" + `
**Transport:** HTTP Streamable
**Authentication:** JWT required

## Tools

### tool_name

**Name:** ` + "`tool_name`" + ` (snake_case)
**Description:** Brief description of what this tool does

**Parameters:**
` + "```json" + `
{
  "parameterName": {
    "type": "string",
    "description": "Parameter description",
    "required": true
  },
  "optionalParam": {
    "type": "number",
    "description": "Optional parameter",
    "required": false,
    "default": 100
  }
}
` + "```" + `

**Usage Example:**
` + "```typescript" + `
const result = await mcpClient.callTool({
  name: "tool_name",
  arguments: {
    parameterName: "value",
    optionalParam: 50
  }
})
` + "```" + `

**Response:**
` + "```json" + `
{
  "content": [
    {
      "type": "text",
      "text": "Operation completed successfully"
    }
  ]
}
` + "```" + `

**Return Data Structure:**
` + "```json" + `
{
  "id": "507f1f77bcf86cd799439011",
  "name": "Result Name",
  "status": "completed"
}
` + "```" + `

**Error Handling:**
- Invalid parameters: Returns error with field-specific validation messages
- Not found: Returns error indicating resource not found
- Unauthorized: Returns error if JWT is invalid

**Common Gotchas:**
- ⚠️ **Parameter names are camelCase** (not snake_case)
- ⚠️ **MongoDB ObjectIDs must be valid** (24 hex characters)
- ⚠️ **Requires company-level isolation** (uses JWT companyId)

## Testing

**Generate test JWT:**
` + "```bash" + `
node ./scripts/generate-test-jwt.js
` + "```" + `

**Test tool call:**
` + "```bash" + `
curl -X POST http://localhost:port/mcp \\
  -H "Authorization: Bearer <jwt>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "method": "tools/call",
    "params": {
      "name": "tool_name",
      "arguments": {"parameterName": "value"}
    }
  }'
` + "```" + `
` + "```" + `

**Storage:**
` + "```typescript" + `
await mcp__hyper__coordinator_upsert_knowledge({
  collection: "technical-knowledge",
  text: "[MCP tool documentation above]",
  metadata: {
    knowledgeType: "mcp-tool",
    service: "[service-name]",
    title: "[Service Name] MCP Tools",
    tags: ["mcp", "tools", "[service-name]", "api"],
    toolNames: ["tool_name", "another_tool"]
  }
})
` + "```" + `
`
	}

	// Service Template
	if includeAll || componentSet["services"] {
		templates += `
### 🏗️ Service Documentation Template

Use this template for microservices:

` + "```markdown" + `
# [Service Name]

## Overview

**Purpose:** What does this service do? What business capability does it provide?

**Technology Stack:**
- Language: Go 1.25
- Framework: Gin (REST), Official MCP Go SDK (MCP)
- Database: MongoDB
- Message Bus: NATS JetStream
- Other: List any other key dependencies

**Deployment:**
- Port: 8081 (REST), 8083 (MCP)
- GKE Cluster: hyperion-production
- Namespace: hyperion-prod
- Replicas: 2

## Architecture

### Service Boundaries

**This service owns:**
- [Domain concept 1] (e.g., "Person records")
- [Domain concept 2] (e.g., "Company hierarchy")

**This service depends on:**
- [External service 1]: For [purpose]
- [External service 2]: For [purpose]

**This service provides to:**
- [Consumer service 1]: [What it provides]
- [Consumer service 2]: [What it provides]

### Data Model

**MongoDB Collections:**

**persons:**
` + "```go" + `
type Person struct {
    ID          primitive.ObjectID  ` + "`" + `bson:"_id,omitempty" json:"id"` + "`" + `
    CompanyID   primitive.ObjectID  ` + "`" + `bson:"companyId" json:"companyId"` + "`" + `
    Name        string              ` + "`" + `bson:"name" json:"name"` + "`" + `
    Email       string              ` + "`" + `bson:"email" json:"email"` + "`" + `
    CreatedAt   time.Time           ` + "`" + `bson:"createdAt" json:"createdAt"` + "`" + `
}
` + "```" + `

### APIs

**REST API:** See [Service Name API Documentation]
**MCP Tools:** See [Service Name MCP Tools Documentation]

### Events

**Published Events:**
- ` + "`person.created`" + `: Published when new person is created
  ` + "```json" + `
  {
    "personId": "507f1f77bcf86cd799439011",
    "companyId": "507f1f77bcf86cd799439012",
    "name": "John Doe",
    "timestamp": "2025-01-15T10:30:00Z"
  }
  ` + "```" + `

**Subscribed Events:**
- ` + "`company.deleted`" + `: Cleans up all persons for deleted company

## Development

### Prerequisites

- Go 1.25+
- MongoDB 8.0+
- NATS 2.10+

### Local Setup

` + "```bash" + `
# Install dependencies
go mod download

# Run locally
go run cmd/server/main.go

# Run tests
go test ./...
` + "```" + `

### Configuration

**Environment Variables:**
` + "```bash" + `
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=staff_db
NATS_URL=nats://localhost:4222
JWT_SECRET=your-secret-key
PORT=8081
MCP_PORT=8083
` + "```" + `

## Testing

### Unit Tests

` + "```bash" + `
go test ./internal/services/...
` + "```" + `

### Integration Tests

` + "```bash" + `
go test ./tests/integration/...
` + "```" + `

### E2E Tests

` + "```bash" + `
cd ui && npm run test:e2e
` + "```" + `

## Deployment

### Manual Deployment

` + "```bash" + `
kubectl apply -f k8s/deployment.yaml
` + "```" + `

### GitHub Actions

Automatically deploys on merge to main branch.

## Monitoring

**Health Check:** ` + "`http://service:port/health`" + `
**Metrics:** Prometheus metrics at ` + "`/metrics`" + `
**Logs:** Structured JSON logs via Zap logger

## Known Issues / Technical Debt

- [Issue 1]: [Description and impact]
- [Issue 2]: [Description and impact]
` + "```" + `

**Storage:**
` + "```typescript" + `
await mcp__hyper__coordinator_upsert_knowledge({
  collection: "technical-knowledge",
  text: "[Service documentation above]",
  metadata: {
    knowledgeType: "service",
    service: "[service-name]",
    title: "[Service Name] Documentation",
    tags: ["service", "go", "microservice", "[domain]"],
    endpoints: ["REST", "MCP"],
    dependencies: ["MongoDB", "NATS"]
  }
})
` + "```" + `
`
	}

	// Component Template
	if includeAll || componentSet["components"] {
		templates += `
### ⚛️ React Component Documentation Template

Use this template for React components:

` + "```markdown" + `
# Component: [ComponentName]

## Overview

**Purpose:** What does this component do? What UI capability does it provide?

**Location:** ` + "`ui/src/components/[ComponentName].tsx`" + `

**Type:**
- [ ] Layout Component
- [ ] Data Display Component
- [ ] Form Component
- [ ] Interactive Component
- [ ] Utility Component

## Props

` + "```typescript" + `
interface ComponentNameProps {
  /** Primary data to display */
  data: DataType;

  /** Callback when user interacts */
  onAction?: (id: string) => void;

  /** Optional styling */
  className?: string;

  /** Loading state */
  isLoading?: boolean;
}
` + "```" + `

## Usage

**Basic Example:**
` + "```tsx" + `
import { ComponentName } from '@/components/ComponentName';

function ParentComponent() {
  const [data, setData] = useState<DataType>([]);

  return (
    <ComponentName
      data={data}
      onAction={(id) => console.log('Action:', id)}
    />
  );
}
` + "```" + `

**With Loading State:**
` + "```tsx" + `
<ComponentName
  data={data}
  isLoading={isLoading}
/>
` + "```" + `

## Dependencies

**UI Library:** Radix UI
**State Management:** React Query
**Styling:** Tailwind CSS

**Key Hooks Used:**
- ` + "`useQuery`" + `: For data fetching
- ` + "`useMutation`" + `: For actions
- ` + "`useState`" + `: For local state

## Patterns

### Optimistic Updates

` + "```tsx" + `
const mutation = useMutation({
  mutationFn: updateData,
  onMutate: async (newData) => {
    // Cancel queries
    await queryClient.cancelQueries({ queryKey: ['data'] })

    // Snapshot previous value
    const previous = queryClient.getQueryData(['data'])

    // Optimistically update
    queryClient.setQueryData(['data'], (old) => [...old, newData])

    return { previous }
  },
  onError: (err, newData, context) => {
    // Rollback on error
    queryClient.setQueryData(['data'], context.previous)
  }
})
` + "```" + `

### Accessibility

- **Keyboard Navigation:** Supports Tab, Enter, Escape
- **Screen Reader:** Proper ARIA labels and roles
- **Focus Management:** Focus trap when modal open

` + "```tsx" + `
<button
  aria-label="Close dialog"
  onClick={onClose}
>
  <X className="h-4 w-4" />
</button>
` + "```" + `

## Testing

**Unit Tests:**
` + "```tsx" + `
describe('ComponentName', () => {
  it('renders data correctly', () => {
    render(<ComponentName data={mockData} />);
    expect(screen.getByText('Expected Text')).toBeInTheDocument();
  });

  it('calls onAction when user interacts', () => {
    const onAction = jest.fn();
    render(<ComponentName data={mockData} onAction={onAction} />);
    fireEvent.click(screen.getByRole('button'));
    expect(onAction).toHaveBeenCalledWith('id');
  });
});
` + "```" + `

**E2E Tests:**
` + "```typescript" + `
test('user can interact with component', async ({ page }) => {
  await page.goto('/page-with-component');
  await page.click('[data-testid="action-button"]');
  await expect(page.locator('.success-message')).toBeVisible();
});
` + "```" + `

## Gotchas

- ⚠️ **Props must be memoized** if passing objects/arrays to avoid re-renders
- ⚠️ **Use optimistic updates** for better UX in task board interactions
- ⚠️ **Handle loading/error states** explicitly
- ⚠️ **Accessibility:** Always include aria-labels for icon buttons

## Related Components

- ` + "`[RelatedComponent1]`" + `: Used for [purpose]
- ` + "`[RelatedComponent2]`" + `: Similar pattern for [use case]
` + "```" + `

**Storage:**
` + "```typescript" + `
await mcp__hyper__coordinator_upsert_knowledge({
  collection: "ui-component-patterns",
  text: "[Component documentation above]",
  metadata: {
    knowledgeType: "component",
    domain: "frontend",
    title: "[ComponentName] React Component",
    tags: ["react", "component", "ui", "[feature-area]"],
    filePath: "ui/src/components/[ComponentName].tsx",
    dependencies: ["Radix UI", "React Query"]
  }
})
` + "```" + `
`
	}

	if templates == "" {
		templates = "No templates selected. Use components parameter to specify which templates to include."
	}

	return templates
}
