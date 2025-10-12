package handlers

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoordinationPromptHandler(t *testing.T) {
	// Create handler
	handler := NewCoordinationPromptHandler()
	require.NotNil(t, handler)

	// Create mock server
	impl := &mcp.Implementation{
		Name:    "test-coordination-prompts",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasPrompts: true,
	}

	server := mcp.NewServer(impl, opts)

	// Register prompts
	err := handler.RegisterCoordinationPrompts(server)
	require.NoError(t, err)

	t.Run("detect_cross_squad_impact prompt", func(t *testing.T) {
		taskDescription := "Modify authentication middleware to add OAuth2 support"
		filesModified := []string{
			"coordinator/middleware/auth.go",
			"coordinator/models/user.go",
			"frontend/src/auth/AuthContext.tsx",
		}

		promptText := handler.buildCrossSquadImpactPrompt(
			taskDescription,
			filesModified,
			[]string{"backend-services", "ui-dev", "go-mcp-dev"},
		)

		assert.NotEmpty(t, promptText)
		assert.Contains(t, promptText, "Cross-Squad Impact Detection")
		assert.Contains(t, promptText, taskDescription)
		assert.Contains(t, promptText, "coordinator/middleware/auth.go")
		assert.Contains(t, promptText, "Active Squads")
		assert.Contains(t, promptText, "backend-services")
		assert.Contains(t, promptText, "Impact Analysis Framework")
		assert.Contains(t, promptText, "API Contracts")
		assert.Contains(t, promptText, "Breaking vs Non-Breaking Changes")
		assert.Contains(t, promptText, "Urgency Classification")
		assert.Contains(t, promptText, "Affected Squads")
		assert.Contains(t, promptText, "Required Communication")
	})

	t.Run("detect_cross_squad_impact without active squads", func(t *testing.T) {
		taskDescription := "Add new logging utility"
		filesModified := []string{"coordinator/utils/logger.go"}

		promptText := handler.buildCrossSquadImpactPrompt(
			taskDescription,
			filesModified,
			nil,
		)

		assert.NotEmpty(t, promptText)
		assert.Contains(t, promptText, "Cross-Squad Impact Detection")
		assert.Contains(t, promptText, taskDescription)
		assert.NotContains(t, promptText, "Active Squads")
	})

	t.Run("suggest_handoff_strategy prompt", func(t *testing.T) {
		phase1Work := `{
			"completed": "Built authentication middleware with JWT validation",
			"files": ["coordinator/middleware/auth.go", "coordinator/models/token.go"],
			"apiContracts": {
				"ValidateToken": "func(token string) (*User, error)",
				"RefreshToken": "func(refreshToken string) (string, error)"
			}
		}`

		phase2Scope := "Implement frontend authentication hooks using the backend JWT API"
		knowledgeGap := "How to call the authentication API, what headers are required, error handling patterns"

		promptText := handler.buildHandoffStrategyPrompt(phase1Work, phase2Scope, knowledgeGap)

		assert.NotEmpty(t, promptText)
		assert.Contains(t, promptText, "Multi-Phase Task Handoff Strategy")
		assert.Contains(t, promptText, phase1Work)
		assert.Contains(t, promptText, phase2Scope)
		assert.Contains(t, promptText, knowledgeGap)
		assert.Contains(t, promptText, "Coordinator Knowledge (Task-Specific Context)")
		assert.Contains(t, promptText, "Qdrant Knowledge (Reusable Patterns)")
		assert.Contains(t, promptText, "priorWorkSummary Field")
		assert.Contains(t, promptText, "Handoff Quality Criteria")
		assert.Contains(t, promptText, "Phase 2 Agent Should Be Able To")
		assert.Contains(t, promptText, "Start coding in <2 minutes")
	})
}

func TestBuildCrossSquadImpactPrompt(t *testing.T) {
	handler := NewCoordinationPromptHandler()

	tests := []struct {
		name            string
		taskDescription string
		filesModified   []string
		activeSquads    []string
		expectedContent []string
		notExpected     []string
	}{
		{
			name:            "API contract changes affecting multiple squads",
			taskDescription: "Add pagination to task list endpoint",
			filesModified: []string{
				"coordinator/handlers/tasks.go",
				"coordinator/models/task.go",
				"frontend/src/api/tasks.ts",
			},
			activeSquads: []string{"backend-services", "ui-dev"},
			expectedContent: []string{
				"Cross-Squad Impact Detection",
				"Add pagination to task list endpoint",
				"coordinator/handlers/tasks.go",
				"Active Squads",
				"backend-services",
				"API Contracts (High Impact)",
				"Breaking vs Non-Breaking Changes",
				"Urgency Classification",
				"BLOCKING (Must coordinate before proceeding)",
			},
		},
		{
			name:            "Shared code modification",
			taskDescription: "Update shared validation utility",
			filesModified: []string{
				"coordinator/utils/validation.go",
			},
			activeSquads: []string{"backend-services", "go-mcp-dev"},
			expectedContent: []string{
				"Shared Code (Medium-High Impact)",
				"All squads using those packages",
				"Required Communication",
				"Coordination Actions",
			},
		},
		{
			name:            "Internal service change",
			taskDescription: "Refactor internal cache implementation",
			filesModified: []string{
				"coordinator/internal/cache/redis.go",
			},
			activeSquads: []string{"backend-services"},
			expectedContent: []string{
				"Domain-Specific (Low-Medium Impact)",
				"Owning squad only",
			},
		},
		{
			name:            "Without active squads",
			taskDescription: "Add health check endpoint",
			filesModified:   []string{"coordinator/handlers/health.go"},
			activeSquads:    nil,
			expectedContent: []string{
				"Cross-Squad Impact Detection",
				"Add health check endpoint",
			},
			notExpected: []string{
				"Active Squads",
			},
		},
		{
			name:            "Database schema change",
			taskDescription: "Add new field to tasks collection",
			filesModified: []string{
				"coordinator/storage/mongo_tasks.go",
				"coordinator/models/task.go",
			},
			activeSquads: []string{"backend-services", "data-platform"},
			expectedContent: []string{
				"Database schemas → Affects: Data Platform",
				"Breaking Changes (Require Immediate Coordination)",
				"Database schema changes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := handler.buildCrossSquadImpactPrompt(
				tt.taskDescription,
				tt.filesModified,
				tt.activeSquads,
			)

			for _, content := range tt.expectedContent {
				assert.Contains(t, prompt, content, "Prompt should contain: %s", content)
			}

			for _, content := range tt.notExpected {
				assert.NotContains(t, prompt, content, "Prompt should NOT contain: %s", content)
			}

			// Verify structure
			assert.Contains(t, prompt, "## Your Mission")
			assert.Contains(t, prompt, "## Squad Domain Mapping")
			assert.Contains(t, prompt, "## Impact Analysis Framework")
			assert.Contains(t, prompt, "## Output Format")
		})
	}
}

func TestBuildHandoffStrategyPrompt(t *testing.T) {
	handler := NewCoordinationPromptHandler()

	tests := []struct {
		name            string
		phase1Work      string
		phase2Scope     string
		knowledgeGap    string
		expectedContent []string
	}{
		{
			name: "Backend to Frontend handoff",
			phase1Work: `{
				"completed": "REST API with JWT auth",
				"endpoints": ["/api/v1/tasks", "/api/v1/auth"],
				"authentication": "Bearer token in Authorization header"
			}`,
			phase2Scope:  "Build React dashboard consuming the tasks API",
			knowledgeGap: "API request/response formats, authentication flow, error handling",
			expectedContent: []string{
				"Multi-Phase Task Handoff Strategy",
				"REST API with JWT auth",
				"Build React dashboard",
				"API request/response formats",
				"Knowledge Distribution Strategy",
				"Coordinator Knowledge (Task-Specific Context)",
				"API contracts established",
				"priorWorkSummary Field (Phase 2 Task Context)",
				"API contracts to consume",
				"Phase 2 Agent Should Be Able To",
				"Start coding in <2 minutes",
			},
		},
		{
			name: "Service layer to MCP tools handoff",
			phase1Work: `{
				"service": "TaskService with CRUD operations",
				"methods": ["CreateTask", "UpdateTask", "DeleteTask"],
				"database": "MongoDB with tasks collection"
			}`,
			phase2Scope:  "Create MCP tools that wrap TaskService methods",
			knowledgeGap: "Service method signatures, error handling patterns, validation requirements",
			expectedContent: []string{
				"TaskService with CRUD operations",
				"Create MCP tools",
				"Service method signatures",
				"Handoff Quality Criteria",
				"Integration instructions",
				"Phase 2 Context Efficiency",
				"Estimated Context Budget",
			},
		},
		{
			name: "Infrastructure to deployment handoff",
			phase1Work: `{
				"infrastructure": "GKE cluster configured",
				"namespaces": ["production", "staging"],
				"secrets": "Configured via kubectl"
			}`,
			phase2Scope:  "Create CI/CD pipeline for automated deployments",
			knowledgeGap: "Cluster access credentials, deployment process, rollback procedures",
			expectedContent: []string{
				"GKE cluster configured",
				"CI/CD pipeline",
				"Cluster access credentials",
				"What Phase 2 should NOT waste time on",
				"Validation Checklist",
				"priorWorkSummary is complete",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := handler.buildHandoffStrategyPrompt(
				tt.phase1Work,
				tt.phase2Scope,
				tt.knowledgeGap,
			)

			for _, content := range tt.expectedContent {
				assert.Contains(t, prompt, content, "Prompt should contain: %s", content)
			}

			// Verify structure
			assert.Contains(t, prompt, "## Your Mission")
			assert.Contains(t, prompt, "## Handoff Architecture")
			assert.Contains(t, prompt, "### 1. Coordinator Knowledge Entry")
			assert.Contains(t, prompt, "### 2. Qdrant Knowledge Entries")
			assert.Contains(t, prompt, "### 3. priorWorkSummary Content")
			assert.Contains(t, prompt, "### 4. Phase 2 Context Efficiency")
			assert.Contains(t, prompt, "### 5. Phase 2 Agent Instructions")
			assert.Contains(t, prompt, "### 6. Validation Checklist")
		})
	}
}

func TestCoordinationPromptRegistration(t *testing.T) {
	handler := NewCoordinationPromptHandler()

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasPrompts: true,
	}

	server := mcp.NewServer(impl, opts)

	// Register should succeed
	err := handler.RegisterCoordinationPrompts(server)
	assert.NoError(t, err)

	// Should not panic on re-registration
	err = handler.RegisterCoordinationPrompts(server)
	assert.NoError(t, err)
}

func TestCrossSquadImpactEdgeCases(t *testing.T) {
	handler := NewCoordinationPromptHandler()

	t.Run("Empty files list", func(t *testing.T) {
		prompt := handler.buildCrossSquadImpactPrompt(
			"Some task",
			[]string{},
			nil,
		)
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "Cross-Squad Impact Detection")
	})

	t.Run("Many files modified", func(t *testing.T) {
		files := []string{
			"file1.go", "file2.go", "file3.go", "file4.go", "file5.go",
			"file6.go", "file7.go", "file8.go", "file9.go", "file10.go",
		}
		prompt := handler.buildCrossSquadImpactPrompt(
			"Large refactoring",
			files,
			[]string{"backend-services"},
		)
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "file1.go")
		assert.Contains(t, prompt, "file10.go")
	})

	t.Run("Special characters in task description", func(t *testing.T) {
		description := "Update API to support \"quoted strings\" and <special> chars & symbols"
		prompt := handler.buildCrossSquadImpactPrompt(
			description,
			[]string{"api/handler.go"},
			nil,
		)
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, description)
	})
}

func TestHandoffStrategyEdgeCases(t *testing.T) {
	handler := NewCoordinationPromptHandler()

	t.Run("Minimal phase 1 work", func(t *testing.T) {
		prompt := handler.buildHandoffStrategyPrompt(
			"Created basic structure",
			"Complete implementation",
			"Everything",
		)
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "Multi-Phase Task Handoff Strategy")
	})

	t.Run("JSON formatted phase 1 work", func(t *testing.T) {
		work := `{
			"files": ["a.go", "b.go"],
			"api": {
				"endpoint": "/api/v1/test",
				"method": "POST"
			}
		}`
		prompt := handler.buildHandoffStrategyPrompt(
			work,
			"Phase 2",
			"API details",
		)
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "files")
		assert.Contains(t, prompt, "/api/v1/test")
	})

	t.Run("Long knowledge gap description", func(t *testing.T) {
		gap := "Need to understand authentication flow, authorization patterns, " +
			"error handling strategies, logging requirements, monitoring setup, " +
			"deployment procedures, rollback mechanisms, and testing approaches"
		prompt := handler.buildHandoffStrategyPrompt(
			"Backend complete",
			"Frontend integration",
			gap,
		)
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, gap)
	})
}
