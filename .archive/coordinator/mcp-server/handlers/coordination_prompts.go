package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CoordinationPromptHandler manages coordination-related prompts for cross-squad impact detection and handoffs
type CoordinationPromptHandler struct{}

// NewCoordinationPromptHandler creates a new coordination prompt handler
func NewCoordinationPromptHandler() *CoordinationPromptHandler {
	return &CoordinationPromptHandler{}
}

// RegisterCoordinationPrompts registers all coordination prompts with the MCP server
func (h *CoordinationPromptHandler) RegisterCoordinationPrompts(server *mcp.Server) error {
	// Register detect_cross_squad_impact prompt
	if err := h.registerDetectCrossSquadImpact(server); err != nil {
		return fmt.Errorf("failed to register detect_cross_squad_impact prompt: %w", err)
	}

	// Register suggest_handoff_strategy prompt
	if err := h.registerSuggestHandoffStrategy(server); err != nil {
		return fmt.Errorf("failed to register suggest_handoff_strategy prompt: %w", err)
	}

	return nil
}

// registerDetectCrossSquadImpact registers the detect_cross_squad_impact prompt
func (h *CoordinationPromptHandler) registerDetectCrossSquadImpact(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "detect_cross_squad_impact",
		Description: "Analyze a task to detect which squads are affected by the changes and what needs to be communicated to prevent conflicts and ensure smooth coordination.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "taskDescription",
				Description: "Description of what's being changed or implemented",
				Required:    true,
			},
			{
				Name:        "filesModified",
				Description: "Comma-separated list of file paths that will be modified or created",
				Required:    true,
			},
			{
				Name:        "activeSquads",
				Description: "Comma-separated list of currently active squad names (optional, e.g., 'backend-services,ui-dev,go-mcp-dev')",
				Required:    false,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments - Arguments is map[string]string in the SDK
		taskDescription := ""
		filesModifiedStr := ""
		activeSquadsStr := ""

		if req.Params != nil && req.Params.Arguments != nil {
			taskDescription = req.Params.Arguments["taskDescription"]
			filesModifiedStr = req.Params.Arguments["filesModified"]
			activeSquadsStr = req.Params.Arguments["activeSquads"]
		}

		if taskDescription == "" || filesModifiedStr == "" {
			return nil, fmt.Errorf("taskDescription and filesModified are required arguments")
		}

		// Parse comma-separated files
		var filesModified []string
		if filesModifiedStr != "" {
			parts := strings.Split(filesModifiedStr, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					filesModified = append(filesModified, trimmed)
				}
			}
		}

		// Parse comma-separated squads
		var activeSquads []string
		if activeSquadsStr != "" {
			parts := strings.Split(activeSquadsStr, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					activeSquads = append(activeSquads, trimmed)
				}
			}
		}

		promptText := h.buildCrossSquadImpactPrompt(taskDescription, filesModified, activeSquads)

		return &mcp.GetPromptResult{
			Description: "Cross-squad impact analysis and coordination recommendations",
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

// registerSuggestHandoffStrategy registers the suggest_handoff_strategy prompt
func (h *CoordinationPromptHandler) registerSuggestHandoffStrategy(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "suggest_handoff_strategy",
		Description: "Recommend optimal handoff strategy for multi-phase tasks to ensure Phase 2 agent can start coding in <2 minutes without reading Phase 1 code.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "phase1Work",
				Description: "JSON summary of what Phase 1 agent completed (files, functions, API contracts, decisions)",
				Required:    true,
			},
			{
				Name:        "phase2Scope",
				Description: "Description of what Phase 2 agent needs to accomplish",
				Required:    true,
			},
			{
				Name:        "knowledgeGap",
				Description: "What information Phase 2 agent needs that isn't obvious from Phase 1 deliverables",
				Required:    true,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments - Arguments is map[string]string in the SDK
		phase1Work := ""
		phase2Scope := ""
		knowledgeGap := ""

		if req.Params != nil && req.Params.Arguments != nil {
			phase1Work = req.Params.Arguments["phase1Work"]
			phase2Scope = req.Params.Arguments["phase2Scope"]
			knowledgeGap = req.Params.Arguments["knowledgeGap"]
		}

		if phase1Work == "" || phase2Scope == "" || knowledgeGap == "" {
			return nil, fmt.Errorf("phase1Work, phase2Scope, and knowledgeGap are required arguments")
		}

		promptText := h.buildHandoffStrategyPrompt(phase1Work, phase2Scope, knowledgeGap)

		return &mcp.GetPromptResult{
			Description: "Multi-phase task handoff strategy recommendations",
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

// buildCrossSquadImpactPrompt builds the cross-squad impact detection prompt
func (h *CoordinationPromptHandler) buildCrossSquadImpactPrompt(taskDescription string, filesModified []string, activeSquads []string) string {
	filesSection := "- " + strings.Join(filesModified, "\n- ")

	squadsSection := ""
	if len(activeSquads) > 0 {
		squadsSection = fmt.Sprintf(`
## Active Squads
The following squads are currently working in the codebase:
%s

Focus your analysis on these squads, but also consider other squads that might be affected.
`, "- "+strings.Join(activeSquads, "\n- "))
	}

	return fmt.Sprintf(`# Cross-Squad Impact Detection

## Task Description
%s

## Files Being Modified
%s
%s
## Your Mission
Analyze this task to detect which squads will be affected by these changes and what coordination is needed to prevent conflicts and ensure smooth integration.

## Squad Domain Mapping

### Backend Infrastructure
- **Backend Services**: Go microservices (handlers, services, repositories)
- **Event Systems**: NATS integration, event publishing/consumption
- **go-mcp-dev**: MCP tool development, tool handlers, schemas
- **Data Platform**: MongoDB schemas, Qdrant collections, database migrations

### Frontend & Experience
- **Frontend Experience**: React architecture, component patterns, state management
- **ui-dev**: React component implementation, hooks, API clients
- **ui-tester**: Playwright tests, accessibility validation, visual regression
- **AI Integration**: Claude/GPT integration, prompt engineering
- **Real-time Systems**: WebSocket connections, real-time updates

### Platform & Security
- **Infrastructure Automation**: GKE deployments, Kubernetes configs, CI/CD
- **Security & Auth**: JWT middleware, RBAC, authentication flows
- **Observability**: Metrics, monitoring, logging infrastructure

### Cross-Squad
- **Workflow Coordinator**: Task orchestration, multi-agent workflows
- **End-to-End Testing**: Full system integration tests

## Impact Analysis Framework

### 1. File Pattern Analysis
Analyze the modified files to detect:

**API Contracts (High Impact)**
- REST endpoint changes → Affects: Backend + Frontend + ui-dev
- MCP tool signatures → Affects: go-mcp-dev + all tool consumers
- WebSocket protocols → Affects: Real-time Systems + ui-dev
- Database schemas → Affects: Data Platform + all services querying those collections

**Shared Code (Medium-High Impact)**
- Shared packages/utilities → Affects: All squads using those packages
- Authentication/middleware → Affects: Security & Auth + all protected endpoints
- Event schemas → Affects: Event Systems + all event consumers

**Domain-Specific (Low-Medium Impact)**
- Service internals → Affects: Owning squad only
- UI components → Affects: Frontend squads only
- Infrastructure configs → Affects: Platform squad only

### 2. Breaking vs Non-Breaking Changes

**Breaking Changes (Require Immediate Coordination):**
- Removing/renaming API endpoints
- Changing request/response schemas
- Modifying authentication flows
- Database schema changes (removing fields)
- Event payload changes

**Non-Breaking (Informational):**
- Adding new endpoints
- Adding optional fields
- New features that don't affect existing functionality
- Performance optimizations
- Internal refactoring

### 3. Urgency Classification

**BLOCKING (Must coordinate before proceeding):**
- Changes to active sprint work of other squads
- Breaking changes to shared APIs/contracts
- Database migrations affecting live data
- Security/auth changes affecting all services

**HIGH (Coordinate within 24h):**
- API additions that other squads plan to use
- New patterns/standards affecting future work
- Performance changes affecting SLAs

**MEDIUM (Document in team-coordination):**
- New reusable patterns
- Gotchas discovered during implementation
- Best practice updates

**LOW (Optional sharing):**
- Internal squad optimizations
- Bug fixes with no cross-squad impact

## Output Format

Provide your analysis in this structure:

### Affected Squads
For each affected squad, specify:
- **Squad Name**: [name]
- **Impact Type**: [API Contract / Shared Code / Domain-Specific]
- **Impact Level**: [Breaking / Non-Breaking]
- **Reason**: [Why this squad is affected]

### Required Communication

**What to Communicate:**
[Specific details that need to be shared - API changes, new patterns, breaking changes, etc.]

**How to Communicate:**
- Qdrant Collection: [which collection to use - team-coordination, code-patterns, etc.]
- Metadata: [suggested metadata tags for discoverability]
- Notification Method: [blocking task creation, async team-coordination post, etc.]

**When to Communicate:**
- Before implementation: [yes/no and why]
- During implementation: [yes/no and why]
- After completion: [yes/no and why]

### Coordination Actions

**Immediate Actions (Before Task Starts):**
1. [Action 1 - e.g., "Post API contract changes to team-coordination"]
2. [Action 2 - e.g., "Query if ui-dev has conflicting work on same endpoints"]

**During Implementation:**
1. [Action 1 - e.g., "Update TODO notes when API contract finalized"]
2. [Action 2 - e.g., "Store new pattern in code-patterns collection"]

**Post-Completion:**
1. [Action 1 - e.g., "Document migration steps in team-coordination"]
2. [Action 2 - e.g., "Update API documentation"]

### Risk Assessment

**Conflict Risk**: [High/Medium/Low]
**Reasoning**: [Why this risk level]

**Mitigation Strategy**:
[Specific steps to reduce conflict risk]

### Recommended Qdrant Queries

Before starting this task, recommend agent to run:
1. Query: [specific Qdrant query]
   Purpose: [what to check for]
   Collection: [which collection]

2. Query: [specific Qdrant query]
   Purpose: [what to check for]
   Collection: [which collection]

Now, analyze the cross-squad impact:`, taskDescription, filesSection, squadsSection)
}

// buildHandoffStrategyPrompt builds the handoff strategy recommendation prompt
func (h *CoordinationPromptHandler) buildHandoffStrategyPrompt(phase1Work, phase2Scope, knowledgeGap string) string {
	return fmt.Sprintf(`# Multi-Phase Task Handoff Strategy

## Phase 1 Completion Summary
%s

## Phase 2 Scope
%s

## Knowledge Gap Analysis
%s

## Your Mission
Recommend the optimal handoff strategy to ensure the Phase 2 agent can start coding in <2 minutes WITHOUT reading Phase 1 code.

## Handoff Architecture

### Knowledge Distribution Strategy

**1. Coordinator Knowledge (Task-Specific Context)**
Store in collection: ` + "`task:hyperion://task/human/{taskId}`" + `

**What to Store:**
- ✅ API contracts established (endpoints, schemas, response formats)
- ✅ Key architectural decisions made during Phase 1
- ✅ Gotchas discovered and solutions found
- ✅ Testing patterns that worked
- ✅ Integration points created

**What NOT to Store:**
- ❌ Implementation details (how code works internally)
- ❌ Code snippets (unless demonstrating API usage)
- ❌ Generic patterns (those go to Qdrant)

**2. Qdrant Knowledge (Reusable Patterns)**
Store in collection: [technical-knowledge, code-patterns, etc.]

**What to Store:**
- ✅ Reusable patterns discovered during Phase 1
- ✅ New architectural approaches worth sharing
- ✅ Performance optimization techniques
- ✅ Security patterns implemented
- ✅ Testing strategies that apply broadly

**What NOT to Store:**
- ❌ Task-specific decisions (those go to coordinator)
- ❌ One-off solutions
- ❌ Temporary workarounds

**3. priorWorkSummary Field (Phase 2 Task Context)**
Embed directly in Phase 2 task

**What to Include:**
- ✅ Phase 1 deliverables summary (what was built)
- ✅ API contracts to consume (exact endpoints, request/response)
- ✅ Files created (list with purpose)
- ✅ Integration instructions (how to connect Phase 2 to Phase 1)
- ✅ What Phase 2 should NOT waste time on

**What NOT to Include:**
- ❌ Phase 1 implementation details
- ❌ How Phase 1 code works internally
- ❌ Historical context (why decisions were made)

## Handoff Quality Criteria

### Phase 2 Agent Should Be Able To:
- [ ] Understand Phase 1 deliverables without reading code (from priorWorkSummary)
- [ ] Know exact API endpoints/functions to call (from coordinator knowledge)
- [ ] Start implementing immediately (from task context + TODO contextHints)
- [ ] Find reusable patterns if needed (from Qdrant collections)
- [ ] Avoid re-solving Phase 1 problems (from coordinator gotchas)

### Time Budget for Phase 2 Agent:
- Read task context: <1 minute (role, priorWorkSummary, TODOs)
- Query coordinator knowledge (if needed): <30 seconds
- Query Qdrant (if needed): <30 seconds
- **Target: Start coding in <2 minutes total**

## Output Format

Provide your handoff strategy in this structure:

### 1. Coordinator Knowledge Entry

**Collection**: ` + "`task:hyperion://task/human/{taskId}`" + `

**Text Content**:
` + "```" + `
## Phase 1 Completion Summary

### API Contracts Established
[List exact endpoints, schemas, response formats that Phase 2 will consume]

### Key Decisions
[Architectural decisions that affect Phase 2 - e.g., "Used JWT in request headers, not cookies"]

### Files Created
[List files with their purpose and what Phase 2 needs to know about them]

### Integration Points
[How Phase 2 connects to Phase 1 - exact function calls, imports, endpoints]

### Gotchas Discovered
[Problems encountered and solutions found that Phase 2 should be aware of]

### Testing Approach
[What testing strategy worked and should be continued in Phase 2]
` + "```" + `

**Metadata**:
` + "```json" + `
{
  "taskId": "{agentTaskId}",
  "agentName": "phase1-agent-name",
  "phase": 1,
  "completedAt": "{timestamp}",
  "relatedServices": ["service1", "service2"]
}
` + "```" + `

### 2. Qdrant Knowledge Entries

For each reusable pattern discovered:

**Collection**: [collection-name]

**Information**:
` + "```" + `
[Detailed pattern description with code examples that apply beyond this task]
` + "```" + `

**Metadata**:
` + "```json" + `
{
  "knowledgeType": "pattern|architecture|best-practice",
  "domain": "backend|frontend|platform",
  "title": "descriptive-title",
  "tags": ["tag1", "tag2"],
  "linkedTaskId": "{humanTaskId}"
}
` + "```" + `

### 3. priorWorkSummary Content

This goes directly into Phase 2 task's priorWorkSummary field:

` + "```" + `
## Phase 1 Delivered

**Built**: [High-level summary of what was created]

**API Contracts for Phase 2**:
- Endpoint: POST /api/v1/example
  Request: { "field": "value" }
  Response: { "result": "data" }

**Files Created** (DO NOT MODIFY - consume only):
- coordinator/service/phase1.go: Exports ExampleFunction(param string) (*Result, error)
- coordinator/models/phase1.go: Contains Phase1Result struct

**How Phase 2 Integrates**:
1. Import: coordinator/service
2. Call: result, err := service.ExampleFunction(input)
3. Use: result.Data for your Phase 2 logic

**Key Decisions Affecting Phase 2**:
- [Decision 1 that Phase 2 must be aware of]
- [Decision 2 that constrains Phase 2 approach]

**Gotchas to Avoid**:
- [Problem 1 and how to avoid it]
- [Problem 2 and solution]

**Phase 2 Should NOT**:
- Re-implement Phase 1 functionality
- Modify Phase 1 files (extend, don't change)
- Duplicate Phase 1 logic
` + "```" + `

### 4. Phase 2 Context Efficiency

**Estimated Context Budget**:
- Task context reading: [X minutes]
- Coordinator knowledge query: [needed/not needed]
- Qdrant pattern search: [needed/not needed]
- File reading before coding: [N files]
- **Total time to start coding**: [X minutes] ← Target <2 minutes

**Efficiency Score**: [High/Medium/Low]
**Reasoning**: [Why this handoff is efficient or what could be improved]

### 5. Phase 2 Agent Instructions

**First Steps for Phase 2 Agent**:
1. Read task's priorWorkSummary field (contains Phase 1 API contracts)
2. [Query coordinator knowledge only if: specific condition]
3. [Query Qdrant only if: specific condition]
4. Start coding at: [specific file and function]
5. DO NOT: [specific things to avoid]

**Files Phase 2 Will Modify**:
[List exact files - these should be in Phase 2 task's filesModified]

**Files Phase 2 Will Reference Only**:
[List Phase 1 files that shouldn't be modified]

### 6. Validation Checklist

Before finalizing handoff, verify:
- [ ] Phase 2 agent has ALL API contracts (no need to read Phase 1 code)
- [ ] Integration instructions are explicit (exact function calls, imports)
- [ ] Gotchas are documented (prevents duplicate debugging)
- [ ] Reusable patterns stored in Qdrant (not buried in task context)
- [ ] priorWorkSummary is complete (answers all "how to use Phase 1" questions)
- [ ] Phase 2 can start in <2 minutes (no exploration needed)

Now, provide your handoff strategy:`, phase1Work, phase2Scope, knowledgeGap)
}
