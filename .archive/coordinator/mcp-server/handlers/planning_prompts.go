package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PlanningPromptHandler manages planning-related prompts for task creation
type PlanningPromptHandler struct{}

// NewPlanningPromptHandler creates a new planning prompt handler
func NewPlanningPromptHandler() *PlanningPromptHandler {
	return &PlanningPromptHandler{}
}

// RegisterPlanningPrompts registers all planning prompts with the MCP server
func (h *PlanningPromptHandler) RegisterPlanningPrompts(server *mcp.Server) error {
	// Register plan_task_breakdown prompt
	if err := h.registerPlanTaskBreakdown(server); err != nil {
		return fmt.Errorf("failed to register plan_task_breakdown prompt: %w", err)
	}

	// Register suggest_context_offload prompt
	if err := h.registerSuggestContextOffload(server); err != nil {
		return fmt.Errorf("failed to register suggest_context_offload prompt: %w", err)
	}

	return nil
}

// registerPlanTaskBreakdown registers the plan_task_breakdown prompt
func (h *PlanningPromptHandler) registerPlanTaskBreakdown(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "plan_task_breakdown",
		Description: "Break down a complex task into detailed TODOs with context hints, file paths, and function names to minimize agent exploration during implementation.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "taskDescription",
				Description: "High-level description of what needs to be accomplished",
				Required:    true,
			},
			{
				Name:        "targetSquad",
				Description: "The squad/agent that will implement this task (e.g., 'go-mcp-dev', 'ui-dev', 'backend-services')",
				Required:    true,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments - Arguments is map[string]string in the SDK
		taskDescription := ""
		targetSquad := ""

		if req.Params != nil && req.Params.Arguments != nil {
			taskDescription = req.Params.Arguments["taskDescription"]
			targetSquad = req.Params.Arguments["targetSquad"]
		}

		if taskDescription == "" || targetSquad == "" {
			return nil, fmt.Errorf("taskDescription and targetSquad are required arguments")
		}

		promptText := h.buildTaskBreakdownPrompt(taskDescription, targetSquad)

		return &mcp.GetPromptResult{
			Description: "Context-rich task breakdown planning prompt",
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

// registerSuggestContextOffload registers the suggest_context_offload prompt
func (h *PlanningPromptHandler) registerSuggestContextOffload(server *mcp.Server) error {
	prompt := &mcp.Prompt{
		Name:        "suggest_context_offload",
		Description: "Analyze task scope and suggest what context to embed in task fields (contextSummary, filesModified, etc.) vs what to store in Qdrant for semantic search.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "taskScope",
				Description: "Detailed scope of the task including requirements, constraints, and integration points",
				Required:    true,
			},
			{
				Name:        "existingKnowledge",
				Description: "Comma-separated list of existing Qdrant collections or knowledge references that might be relevant (optional)",
				Required:    false,
			},
		},
	}

	handler := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments - Arguments is map[string]string in the SDK
		taskScope := ""
		existingKnowledgeStr := ""

		if req.Params != nil && req.Params.Arguments != nil {
			taskScope = req.Params.Arguments["taskScope"]
			existingKnowledgeStr = req.Params.Arguments["existingKnowledge"]
		}

		if taskScope == "" {
			return nil, fmt.Errorf("taskScope is a required argument")
		}

		// Parse comma-separated knowledge collections
		var existingKnowledge []string
		if existingKnowledgeStr != "" {
			parts := strings.Split(existingKnowledgeStr, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					existingKnowledge = append(existingKnowledge, trimmed)
				}
			}
		}

		promptText := h.buildContextOffloadPrompt(taskScope, existingKnowledge)

		return &mcp.GetPromptResult{
			Description: "Context offloading strategy guidance",
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

// buildTaskBreakdownPrompt builds the detailed task breakdown prompt
func (h *PlanningPromptHandler) buildTaskBreakdownPrompt(taskDescription, targetSquad string) string {
	return fmt.Sprintf(`# Task Breakdown Planning for %s

## Original Task Description
%s

## Your Mission
Break down this task into 3-7 detailed TODO items that will allow the implementation agent to START CODING within 2 minutes.

## Critical Requirements

### Each TODO MUST Include:

1. **description** (10-20 words)
   - Specific, actionable task
   - ✅ "Create JWT middleware with token validation and error handling"
   - ❌ "Add authentication"

2. **filePath** (exact location)
   - Full path from project root
   - Specify if creating new vs modifying existing
   - Example: "coordinator/mcp-server/middleware/auth.go"

3. **functionName** (if applicable)
   - Exact function/method name to create or modify
   - Include signature if creating new
   - Example: "ValidateJWT" or "NewAuthMiddleware(config *Config)"

4. **contextHint** (50-100 words)
   - HOW to implement (specific pattern/approach)
   - Key functions/packages to use
   - Error handling strategy
   - Return value/output format
   - Example code snippet if helpful
   - Example: "Extract token from Authorization header (Bearer scheme). Use jwt.Parse() with HS256. Validate exp, iss, aud claims. Store user ID in gin.Context. Return 401 with {\"error\": \"invalid_token\"} on failure."

### Task Sizing Guidelines:
- **Small:** 1-3 files, 3-5 TODOs, <30 min
- **Medium:** 3-5 files, 5-7 TODOs, <60 min
- **Large:** SPLIT IT (never >7 TODOs, never multiple services)

### Output Format (JSON):
Return a JSON object with this structure:

{
  "todos": [
    {
      "description": "Specific actionable task",
      "filePath": "exact/path/to/file.go",
      "functionName": "FunctionName",
      "contextHint": "Detailed implementation guidance with specific patterns, functions to use, error handling approach, and example code if helpful"
    }
  ],
  "filesModified": [
    "exact/path/to/file1.go",
    "exact/path/to/file2.go"
  ],
  "estimatedDuration": "30-45 minutes"
}

## Quality Checklist
Before finalizing, verify:
- [ ] Every TODO has exact file path
- [ ] Every function-level TODO has function name
- [ ] Every contextHint explains HOW, not just WHAT
- [ ] Implementation agent can code immediately without searching
- [ ] No generic TODOs like "implement feature" or "add tests"
- [ ] Each TODO is independently actionable

Now, break down the task:`, targetSquad, taskDescription)
}

// buildContextOffloadPrompt builds the context offloading strategy prompt
func (h *PlanningPromptHandler) buildContextOffloadPrompt(taskScope string, existingKnowledge []string) string {
	knowledgeSection := ""
	if len(existingKnowledge) > 0 {
		knowledgeSection = fmt.Sprintf(`
## Existing Knowledge References
The following Qdrant collections or knowledge bases are available:
%s

Consider whether these provide reusable patterns or if new task-specific context is needed.
`, "- "+strings.Join(existingKnowledge, "\n- "))
	}

	return fmt.Sprintf(`# Context Offloading Strategy

## Task Scope
%s
%s
## Your Mission
Analyze this task scope and recommend what context to embed DIRECTLY in the task vs what to reference from Qdrant.

## Context Distribution Framework

### 📋 Task Fields (Embed 80%% of Context Here)
These fields make implementation agents efficient by eliminating exploration:

**1. contextSummary** (150-250 words)
- Business context: WHY this task exists, user impact
- Technical approach: WHAT solution pattern to use (be specific)
- Integration points: HOW this connects to other components
- Constraints: Resource limits, performance targets, security
- Testing approach: Unit tests, edge cases to cover

**2. filesModified** (complete list)
- EVERY file the agent will create or modify
- Mark reference files as "reference only"
- Include test files

**3. qdrantCollections** (1-3 max)
- Name specific collections with relevant patterns
- Indicate WHAT to search for
- Only include if genuinely needed

**4. priorWorkSummary** (100-150 words, multi-phase only)
- What previous agent accomplished
- API contracts/interfaces established
- Key decisions affecting this task
- Gotchas discovered

**5. notes** (50-100 words)
- Critical gotchas specific to this task
- Non-obvious requirements
- Performance/security considerations

### 🔍 Qdrant Collections (Store Reusable 20%%)
Use Qdrant for patterns that apply across multiple tasks:

**When to Store in Qdrant:**
- ✅ Reusable technical patterns (JWT auth, error handling)
- ✅ Architectural decisions (ADRs)
- ✅ Cross-service contracts (API schemas)
- ✅ Best practices and gotchas
- ✅ Testing strategies

**When to Embed in Task:**
- ✅ Task-specific requirements
- ✅ Exact file locations
- ✅ Function signatures for this task
- ✅ Business logic for this feature
- ✅ Integration details for this component

## Your Analysis
Provide a structured recommendation:

### 1. Task Field Content Recommendations

**contextSummary:**
[Write the exact 150-250 word summary to embed]

**filesModified:**
[List exact file paths]

**qdrantCollections (if needed):**
[List 1-3 collections with specific search terms]

**notes:**
[List critical gotchas and shortcuts]

**priorWorkSummary (if multi-phase):**
[Summarize prior work with API contracts]

### 2. Qdrant Storage Recommendations

**Collections to Create/Update:**
- Collection: [name]
  Purpose: [what reusable knowledge to store]
  Example entry: [what to document]

### 3. Context Efficiency Score
Estimate what %% of context is embedded in task vs requires Qdrant lookup:
- Task-embedded: [X%%] ← Target 80%%+
- Qdrant-required: [Y%%] ← Target <20%%

### 4. Agent Work Estimate
- Time to read task context: [X minutes]
- Qdrant queries needed: [N queries]
- Time to start coding: [X minutes] ← Target <2 minutes

Now, analyze the task scope and provide your recommendations:`, taskScope, knowledgeSection)
}
