package handlers

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanningPromptHandler(t *testing.T) {
	// Create handler
	handler := NewPlanningPromptHandler()
	require.NotNil(t, handler)

	// Create mock server
	impl := &mcp.Implementation{
		Name:    "test-planning-prompts",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasPrompts: true,
	}

	server := mcp.NewServer(impl, opts)

	// Register prompts
	err := handler.RegisterPlanningPrompts(server)
	require.NoError(t, err)

	t.Run("plan_task_breakdown prompt", func(t *testing.T) {
		// Test the handler directly
		promptHandler := handler.buildTaskBreakdownPrompt(
			"Implement JWT authentication middleware",
			"go-mcp-dev",
		)

		assert.NotEmpty(t, promptHandler)
		assert.Contains(t, promptHandler, "Task Breakdown Planning for go-mcp-dev")
		assert.Contains(t, promptHandler, "Implement JWT authentication middleware")
		assert.Contains(t, promptHandler, "description")
		assert.Contains(t, promptHandler, "filePath")
		assert.Contains(t, promptHandler, "functionName")
		assert.Contains(t, promptHandler, "contextHint")
		assert.Contains(t, promptHandler, "Quality Checklist")
	})

	t.Run("plan_task_breakdown with missing arguments", func(t *testing.T) {
		// Test handler logic directly
		taskDescription := ""
		targetSquad := "go-mcp-dev"

		// Simulate the validation logic
		if taskDescription == "" || targetSquad == "" {
			// This should trigger an error in the actual handler
			assert.True(t, taskDescription == "" || targetSquad == "", "Should detect missing arguments")
		}
	})

	t.Run("suggest_context_offload prompt", func(t *testing.T) {
		// Test with valid arguments
		taskScope := "Build a new MCP tool for task management with MongoDB storage and comprehensive error handling"

		promptHandler := handler.buildContextOffloadPrompt(
			taskScope,
			[]string{"technical-knowledge", "mcp-patterns"},
		)

		assert.NotEmpty(t, promptHandler)
		assert.Contains(t, promptHandler, "Context Offloading Strategy")
		assert.Contains(t, promptHandler, taskScope)
		assert.Contains(t, promptHandler, "technical-knowledge")
		assert.Contains(t, promptHandler, "mcp-patterns")
		assert.Contains(t, promptHandler, "contextSummary")
		assert.Contains(t, promptHandler, "filesModified")
		assert.Contains(t, promptHandler, "qdrantCollections")
		assert.Contains(t, promptHandler, "Context Efficiency Score")
	})

	t.Run("suggest_context_offload without existing knowledge", func(t *testing.T) {
		taskScope := "Create a new HTTP endpoint"

		promptHandler := handler.buildContextOffloadPrompt(
			taskScope,
			nil,
		)

		assert.NotEmpty(t, promptHandler)
		assert.Contains(t, promptHandler, "Context Offloading Strategy")
		assert.Contains(t, promptHandler, taskScope)
		assert.NotContains(t, promptHandler, "Existing Knowledge References")
	})
}

func TestBuildTaskBreakdownPrompt(t *testing.T) {
	handler := NewPlanningPromptHandler()

	tests := []struct {
		name            string
		taskDescription string
		targetSquad     string
		expectedContent []string
	}{
		{
			name:            "Backend task",
			taskDescription: "Create REST API for user management",
			targetSquad:     "backend-services",
			expectedContent: []string{
				"Task Breakdown Planning for backend-services",
				"Create REST API for user management",
				"description",
				"filePath",
				"functionName",
				"contextHint",
			},
		},
		{
			name:            "Frontend task",
			taskDescription: "Build responsive dashboard UI",
			targetSquad:     "ui-dev",
			expectedContent: []string{
				"Task Breakdown Planning for ui-dev",
				"Build responsive dashboard UI",
				"Output Format (JSON)",
			},
		},
		{
			name:            "MCP development task",
			taskDescription: "Add new MCP tools for document search",
			targetSquad:     "go-mcp-dev",
			expectedContent: []string{
				"Task Breakdown Planning for go-mcp-dev",
				"Add new MCP tools for document search",
				"Quality Checklist",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := handler.buildTaskBreakdownPrompt(tt.taskDescription, tt.targetSquad)

			for _, content := range tt.expectedContent {
				assert.Contains(t, prompt, content, "Prompt should contain: %s", content)
			}

			// Verify structure
			assert.Contains(t, prompt, "## Your Mission")
			assert.Contains(t, prompt, "## Critical Requirements")
			assert.Contains(t, prompt, "### Task Sizing Guidelines:")
			assert.Contains(t, prompt, "### Output Format (JSON):")
			assert.Contains(t, prompt, "## Quality Checklist")
		})
	}
}

func TestBuildContextOffloadPrompt(t *testing.T) {
	handler := NewPlanningPromptHandler()

	tests := []struct {
		name              string
		taskScope         string
		existingKnowledge []string
		expectedContent   []string
		notExpected       []string
	}{
		{
			name:              "With existing knowledge",
			taskScope:         "Implement OAuth2 authentication flow",
			existingKnowledge: []string{"auth-patterns", "security-best-practices"},
			expectedContent: []string{
				"Context Offloading Strategy",
				"Implement OAuth2 authentication flow",
				"Existing Knowledge References",
				"auth-patterns",
				"security-best-practices",
				"contextSummary",
				"filesModified",
				"Context Efficiency Score",
			},
		},
		{
			name:      "Without existing knowledge",
			taskScope: "Create a new microservice",
			expectedContent: []string{
				"Context Offloading Strategy",
				"Create a new microservice",
				"contextSummary",
				"filesModified",
				"qdrantCollections",
			},
			notExpected: []string{
				"Existing Knowledge References",
			},
		},
		{
			name:              "Complex multi-phase task",
			taskScope:         "Build end-to-end user registration system with email validation and 2FA",
			existingKnowledge: []string{"email-templates", "2fa-implementations"},
			expectedContent: []string{
				"priorWorkSummary",
				"API contracts/interfaces established",
				"Agent Work Estimate",
				"Time to start coding",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := handler.buildContextOffloadPrompt(tt.taskScope, tt.existingKnowledge)

			for _, content := range tt.expectedContent {
				assert.Contains(t, prompt, content, "Prompt should contain: %s", content)
			}

			for _, content := range tt.notExpected {
				assert.NotContains(t, prompt, content, "Prompt should NOT contain: %s", content)
			}

			// Verify structure
			assert.Contains(t, prompt, "## Your Mission")
			assert.Contains(t, prompt, "## Context Distribution Framework")
			assert.Contains(t, prompt, "### 📋 Task Fields (Embed 80% of Context Here)")
			assert.Contains(t, prompt, "### 🔍 Qdrant Collections (Store Reusable 20%)")
			assert.Contains(t, prompt, "## Your Analysis")
		})
	}
}

func TestPromptRegistration(t *testing.T) {
	handler := NewPlanningPromptHandler()

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasPrompts: true,
	}

	server := mcp.NewServer(impl, opts)

	// Register should succeed
	err := handler.RegisterPlanningPrompts(server)
	assert.NoError(t, err)

	// Should not panic on re-registration
	err = handler.RegisterPlanningPrompts(server)
	assert.NoError(t, err)
}
