package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DocResourceHandler manages documentation resources
type DocResourceHandler struct{}

// NewDocResourceHandler creates a new documentation resource handler
func NewDocResourceHandler() *DocResourceHandler {
	return &DocResourceHandler{}
}

// RegisterDocResources registers all documentation resources with the MCP server
func (h *DocResourceHandler) RegisterDocResources(server *mcp.Server) error {
	// Register engineering standards resource
	standardsResource := &mcp.Resource{
		URI:         "hyperion://docs/standards",
		Name:        "Engineering Standards",
		Description: "Hyperion code quality gates, DRY/SOLID principles, and file size limits",
		MIMEType:    "text/markdown",
	}
	server.AddResource(standardsResource, h.createStandardsHandler())

	// Register architecture resource
	architectureResource := &mcp.Resource{
		URI:         "hyperion://docs/architecture",
		Name:        "System Architecture",
		Description: "Dual-MCP architecture, squad structure, and coordination patterns",
		MIMEType:    "text/markdown",
	}
	server.AddResource(architectureResource, h.createArchitectureHandler())

	// Register squad guide resource
	squadGuideResource := &mcp.Resource{
		URI:         "hyperion://docs/squad-guide",
		Name:        "Squad Coordination Guide",
		Description: "Dual-MCP workflow, work protocols, and coordination best practices",
		MIMEType:    "text/markdown",
	}
	server.AddResource(squadGuideResource, h.createSquadGuideHandler())

	return nil
}

// createStandardsHandler creates the handler for engineering standards resource
func (h *DocResourceHandler) createStandardsHandler() mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		content := `# Hyperion Engineering Standards

## Fail-Fast Principle
**Never use silent fallbacks.** Always return real errors with context.

✅ **Correct:**
` + "```go" + `
return "", fmt.Errorf("server URL not found for %s", serverName)
` + "```" + `

❌ **Wrong:**
` + "```go" + `
return fmt.Sprintf("http://%s:8080/mcp", serverName) // Hides problem
` + "```" + `

## MCP Compliance
- Use official Go SDK only
- Tool names: snake_case
- Parameters/JSON: camelCase

## Security
- JWT required for ALL endpoints
- Use ` + "`./scripts/generate-test-jwt.js`" + ` for testing
- Never log secrets

## JSON Naming (MANDATORY)
ALL JSON/URL params MUST be camelCase (frontend contract):

✅ ` + "`json:\"userId\"`" + `, ` + "`c.Query(\"userId\")`" + `, ` + "`/api/v1/ws?userId=123`" + `
❌ ` + "`json:\"user_id\"`" + `, ` + "`c.Query(\"user_id\")`" + `, ` + "`/api/v1/ws?user_id=123`" + `

## Code Quality
- Go 1.25 only
- Handlers ≤300 lines
- Services ≤400 lines
- main.go ≤200 lines
- CLAUDE.md required per package before merge

## DRY/SOLID/YAGNI
- No code duplication
- Single responsibility
- Interfaces for extensibility
- Inject dependencies
- No speculative features

## Quality Gates by Squad

### Backend
- Handlers ≤300 lines
- Services ≤400 lines
- main.go ≤200 lines
- Complexity ≤10/function
- >500 lines = REFACTOR NOW

### Frontend Experience
- Patterns ≤250 lines
- 80% component reuse
- ≤5 props per component
- Document all patterns

### ui-dev
- Components ≤250 lines
- Hooks ≤150 lines
- API clients ≤300 lines
- Zero duplicate logic
- Optimistic UI for task board

### ui-tester
- ≥80% test coverage
- WCAG 2.1 AA compliance
- ≥95% non-flaky tests
- ≤300 lines per test suite
- ≤5 min runtime

### Platform
- Zero hardcoded values
- K8s configs ≤200 lines
- Security configs ≤300 lines
- Deployment scripts ≤250 lines

## Refactoring Rules
- **72-hour rule** for oversized files
- God files block squad merges
- Refactoring gets sprint priority
`

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "hyperion://docs/standards",
					MIMEType: "text/markdown",
					Text:     content,
				},
			},
		}, nil
	}
}

// createArchitectureHandler creates the handler for architecture resource
func (h *DocResourceHandler) createArchitectureHandler() mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		content := `# Hyperion System Architecture

## Dual-MCP Architecture

### Hyperion Coordinator MCP (MongoDB)
- Task tracking and assignments
- Progress monitoring
- TODO management
- UI visibility (real-time task board)

### Qdrant MCP (Vector DB)
- Technical knowledge storage
- Architecture patterns
- Architecture decisions
- Cross-team coordination
- Semantic search

### Why Both?
- **Separation:** Tasks vs Knowledge
- **Optimized:** Relational (MongoDB) vs Semantic (Qdrant)
- **Visibility:** Real-time UI task board
- **Reuse:** Discover existing solutions
- **Parallel:** Independent squad workflows

## Squad Structure

### Backend Infrastructure
- Backend Services (Go microservices)
- Event Systems (NATS)
- go-mcp-dev (MCP tools)
- Data Platform (MongoDB/Qdrant)

### Frontend & Experience
- Frontend Experience (architecture)
- ui-dev (implementation)
- ui-tester (Playwright)
- AI Integration (Claude/GPT)
- Real-time Systems (WebSocket)

### Platform & Security
- Infrastructure Automation (GKE)
- Security & Auth (JWT/RBAC)
- Observability (metrics/monitoring)

### Cross-Squad
- Workflow Coordinator (task orchestration)
- End-to-End Testing (system validation)

## Golden Rules
1. Work ONLY within your domain
2. Tasks assigned via hyperion-coordinator MCP
3. Knowledge shared via Qdrant MCP
4. Every task uses dual-MCP workflow

## MCP Tools by Squad

### ALL AGENTS (MANDATORY)
- hyperion-coordinator
- qdrant-mcp

### Additional by Squad
- **Backend Infrastructure:** filesystem, github, fetch, mongodb
- **Frontend & Experience:** filesystem, github, fetch, playwright-mcp
- **Platform & Security:** kubernetes, github, filesystem, fetch
- **Workflow Coordinator:** Primarily hyperion-coordinator for task orchestration

## Qdrant Collections

### Task Collections
- ` + "`task:hyperion://task/human/{taskId}`" + `
- team-coordination
- agent-coordination

### Technical Collections
- technical-knowledge
- code-patterns
- adr
- data-contracts
- technical-debt-registry

### UI Collections
- ui-component-patterns
- ui-test-strategies
- ui-accessibility-standards
- ui-visual-regression-baseline

### Operations
- mcp-operations
- code-quality-violations
`

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "hyperion://docs/architecture",
					MIMEType: "text/markdown",
					Text:     content,
				},
			},
		}, nil
	}
}

// createSquadGuideHandler creates the handler for squad coordination guide resource
func (h *DocResourceHandler) createSquadGuideHandler() mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		content := `# Squad Coordination Guide

## Dual-MCP Workflow

### Core MCP Tools
` + "```typescript" + `
// Get assignments
mcp__hyperion-coordinator__coordinator_list_agent_tasks({ agentName: "..." })

// Update progress
mcp__hyperion-coordinator__coordinator_update_task_status(...)

// Update TODOs (uses todoId UUID, not index)
mcp__hyperion-coordinator__coordinator_update_todo_status(...)

// Search knowledge
mcp__qdrant__qdrant-find({ collection_name: "...", query: "..." })

// Store knowledge
mcp__qdrant__qdrant-store({ collection_name: "...", information: "...", metadata: {...} })
` + "```" + `

### Common Mistakes
❌ Using ` + "`todoIndex`" + ` → ✅ Use ` + "`todoId`" + ` (UUID)
❌ Using ` + "`taskId`" + ` in TODO updates → ✅ Use ` + "`agentTaskId`" + `
❌ Missing ` + "`mcp__hyperion-coordinator__`" + ` prefix
❌ Wrong parameter types

## Work Protocol

### During Work
` + "```typescript" + `
// Update task status as you progress
coordinator_update_task_status({
  taskId,
  status: "in_progress|blocked|completed",
  notes: "..."
})

// Preserve context in TODO updates
coordinator_update_todo_status({
  agentTaskId, todoId,
  notes: "Completed: X at line 45. Key decision: Y. Gotcha: Z. NEXT TODO: use pattern A"
})

// Coordinate with other squads (ONLY if needed)
qdrant-store({
  collection_name: "team-coordination",
  information: "...",
  metadata: {...}
})
` + "```" + `

### Post-Work (REQUIRED)
1. **Store task-specific knowledge in coordinator**
2. **Share reusable knowledge in Qdrant**
3. **Document technical debt** (if found)
4. **Final status update**

## Coordination Patterns

### DO
- Query Qdrant first before duplicating work
- Document immediately after discoveries
- Design for parallel work
- Update status frequently

### DON'T
- Work outside your domain
- Skip coordination protocols
- Create hidden dependencies
- Bypass Qdrant knowledge sharing
- Ignore existing knowledge

### Cross-Squad Coordination
Post to ` + "`team-coordination`" + ` collection for:
- API changes
- Security updates
- Performance issues

Relevant squads discover and handle via Qdrant queries.

## Context Window Management

### Context Budget
- **Planning:** <20% (5-10 min max)
- **Implementation:** 60% (actual work)
- **Documentation:** 20% (post-work)

### Rules
1. Task context is FREE - read task fields first
2. Query ONLY when insufficient - not speculatively
3. Read files to MODIFY, not to understand
4. Start coding within 5 minutes

### Warning Signs
- Planning >10 min → Start implementing NOW
- Made >2 Qdrant queries → Over-researching
- Read >5 files → Exploring, not executing

## Emergency Recovery
If context exhausted mid-task:
1. Update coordinator with progress
2. Store work + decisions in coordinator knowledge
3. Mark TODO with handoff notes for next agent
`

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "hyperion://docs/squad-guide",
					MIMEType: "text/markdown",
					Text:     content,
				},
			},
		}, nil
	}
}
