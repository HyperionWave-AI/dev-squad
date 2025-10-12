package handlers

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentationPromptHandler(t *testing.T) {
	// Create handler
	handler := NewDocumentationPromptHandler()
	require.NotNil(t, handler)

	// Create mock server
	impl := &mcp.Implementation{
		Name:    "test-documentation-prompts",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasPrompts: true,
	}

	server := mcp.NewServer(impl, opts)

	// Register prompts
	err := handler.RegisterDocumentationPrompts(server)
	require.NoError(t, err)

	t.Run("build_knowledge_base prompt with all components", func(t *testing.T) {
		// Test the handler directly
		promptHandler := handler.buildKnowledgeBasePrompt(
			"/path/to/project",
			[]string{"all"},
			false,
		)

		assert.NotEmpty(t, promptHandler)
		assert.Contains(t, promptHandler, "Knowledge Base Builder")
		assert.Contains(t, promptHandler, "/path/to/project")
		assert.Contains(t, promptHandler, "ADR")
		assert.Contains(t, promptHandler, "API")
		assert.Contains(t, promptHandler, "MCP")
		assert.Contains(t, promptHandler, "Services")
		assert.Contains(t, promptHandler, "Components")
		assert.Contains(t, promptHandler, "Codebase Analysis")
		assert.Contains(t, promptHandler, "Knowledge Storage Strategy")
	})

	t.Run("build_knowledge_base with task generation enabled", func(t *testing.T) {
		promptHandler := handler.buildKnowledgeBasePrompt(
			"/path/to/project",
			[]string{"all"},
			true,
		)

		assert.NotEmpty(t, promptHandler)
		assert.Contains(t, promptHandler, "Agent Task Generation")
		assert.Contains(t, promptHandler, "mcp__hyper__coordinator_create_human_task")
		assert.Contains(t, promptHandler, "mcp__hyper__coordinator_create_agent_task")
		assert.Contains(t, promptHandler, "contextSummary")
		assert.Contains(t, promptHandler, "Agent Assignment Matrix")
	})

	t.Run("build_knowledge_base with specific components", func(t *testing.T) {
		promptHandler := handler.buildKnowledgeBasePrompt(
			"/path/to/project",
			[]string{"apis", "mcps"},
			false,
		)

		assert.NotEmpty(t, promptHandler)
		assert.Contains(t, promptHandler, "apis, mcps")
	})

	t.Run("build_knowledge_base without task generation", func(t *testing.T) {
		promptHandler := handler.buildKnowledgeBasePrompt(
			"/path/to/project",
			[]string{"all"},
			false,
		)

		assert.NotEmpty(t, promptHandler)
		assert.NotContains(t, promptHandler, "Agent Task Generation")
	})
}

func TestBuildComponentTemplates(t *testing.T) {
	handler := NewDocumentationPromptHandler()

	t.Run("all components template", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"all"})

		assert.NotEmpty(t, templates)
		assert.Contains(t, templates, "ADR")
		assert.Contains(t, templates, "Architecture Decision Record")
		assert.Contains(t, templates, "API Documentation Template")
		assert.Contains(t, templates, "MCP Tool Documentation Template")
		assert.Contains(t, templates, "Service Documentation Template")
		assert.Contains(t, templates, "React Component Documentation Template")
	})

	t.Run("ADR template only", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"adrs"})

		assert.NotEmpty(t, templates)
		assert.Contains(t, templates, "Architecture Decision Record")
		assert.Contains(t, templates, "Context")
		assert.Contains(t, templates, "Decision")
		assert.Contains(t, templates, "Rationale")
		assert.Contains(t, templates, "Consequences")
		assert.Contains(t, templates, "Implementation")
		assert.Contains(t, templates, "Validation")
		assert.NotContains(t, templates, "API Documentation Template")
	})

	t.Run("API template only", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"apis"})

		assert.NotEmpty(t, templates)
		assert.Contains(t, templates, "API Documentation Template")
		assert.Contains(t, templates, "Base URL")
		assert.Contains(t, templates, "Authentication")
		assert.Contains(t, templates, "Endpoints")
		assert.Contains(t, templates, "Rate Limiting")
		assert.Contains(t, templates, "Error Handling")
		assert.NotContains(t, templates, "Architecture Decision Record")
	})

	t.Run("MCP template only", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"mcps"})

		assert.NotEmpty(t, templates)
		assert.Contains(t, templates, "MCP Tool Documentation Template")
		assert.Contains(t, templates, "tool_name")
		assert.Contains(t, templates, "Parameters")
		assert.Contains(t, templates, "Usage Example")
		assert.Contains(t, templates, "Response")
		assert.Contains(t, templates, "Error Handling")
		assert.Contains(t, templates, "Common Gotchas")
		assert.NotContains(t, templates, "Service Documentation Template")
	})

	t.Run("Service template only", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"services"})

		assert.NotEmpty(t, templates)
		assert.Contains(t, templates, "Service Documentation Template")
		assert.Contains(t, templates, "Overview")
		assert.Contains(t, templates, "Architecture")
		assert.Contains(t, templates, "Data Model")
		assert.Contains(t, templates, "MongoDB Collections")
		assert.Contains(t, templates, "Published Events")
		assert.Contains(t, templates, "Development")
		assert.Contains(t, templates, "Deployment")
		assert.NotContains(t, templates, "React Component")
	})

	t.Run("Component template only", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"components"})

		assert.NotEmpty(t, templates)
		assert.Contains(t, templates, "React Component Documentation Template")
		assert.Contains(t, templates, "Props")
		assert.Contains(t, templates, "Usage")
		assert.Contains(t, templates, "Dependencies")
		assert.Contains(t, templates, "Patterns")
		assert.Contains(t, templates, "Optimistic Updates")
		assert.Contains(t, templates, "Accessibility")
		assert.Contains(t, templates, "Testing")
		assert.NotContains(t, templates, "Service Documentation Template")
	})

	t.Run("multiple components", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"apis", "mcps", "services"})

		assert.NotEmpty(t, templates)
		assert.Contains(t, templates, "API Documentation Template")
		assert.Contains(t, templates, "MCP Tool Documentation Template")
		assert.Contains(t, templates, "Service Documentation Template")
		assert.NotContains(t, templates, "Architecture Decision Record")
		assert.NotContains(t, templates, "React Component")
	})

	t.Run("empty components defaults to all", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{})

		assert.NotEmpty(t, templates)
		assert.Contains(t, templates, "ADR")
		assert.Contains(t, templates, "API")
		assert.Contains(t, templates, "MCP")
		assert.Contains(t, templates, "Service")
		assert.Contains(t, templates, "Component")
	})
}

func TestBuildKnowledgeBasePrompt(t *testing.T) {
	handler := NewDocumentationPromptHandler()

	tests := []struct {
		name            string
		projectPath     string
		components      []string
		generateTasks   bool
		expectedContent []string
		notExpected     []string
	}{
		{
			name:          "Full documentation with task generation",
			projectPath:   "/Users/dev/hyperion",
			components:    []string{"all"},
			generateTasks: true,
			expectedContent: []string{
				"Knowledge Base Builder",
				"/Users/dev/hyperion",
				"Codebase Analysis",
				"Identify Services and APIs",
				"Go Services",
				"REST APIs",
				"MCP Tools",
				"React Components",
				"Data Models",
				"Architecture Decisions",
				"Agent Task Generation",
				"mcp__hyper__coordinator_create_human_task",
				"mcp__hyper__coordinator_create_agent_task",
				"Documentation Templates by Component Type",
				"Knowledge Storage Strategy",
				"technical-knowledge",
				"code-patterns",
				"adr",
				"data-contracts",
				"ui-component-patterns",
				"Output Format",
				"Quality Checklist",
			},
		},
		{
			name:          "Documentation without task generation",
			projectPath:   "/path/to/project",
			components:    []string{"all"},
			generateTasks: false,
			expectedContent: []string{
				"Knowledge Base Builder",
				"Codebase Analysis",
				"Documentation Templates",
				"Knowledge Storage Strategy",
			},
			notExpected: []string{
				"Agent Task Generation",
				"mcp__hyper__coordinator_create_human_task",
			},
		},
		{
			name:          "Specific components only",
			projectPath:   "/project",
			components:    []string{"apis", "services"},
			generateTasks: false,
			expectedContent: []string{
				"Knowledge Base Builder",
				"apis, services",
				"Codebase Analysis",
				"REST APIs",
				"Go Services",
			},
		},
		{
			name:          "MCP and components focus",
			projectPath:   "/project",
			components:    []string{"mcps", "components"},
			generateTasks: true,
			expectedContent: []string{
				"mcps, components",
				"MCP Tools",
				"React Components",
				"Agent Task Generation",
				"go-mcp-dev",
				"ui-dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := handler.buildKnowledgeBasePrompt(tt.projectPath, tt.components, tt.generateTasks)

			for _, content := range tt.expectedContent {
				assert.Contains(t, prompt, content, "Prompt should contain: %s", content)
			}

			for _, content := range tt.notExpected {
				assert.NotContains(t, prompt, content, "Prompt should NOT contain: %s", content)
			}

			// Verify structure
			assert.Contains(t, prompt, "## Your Mission")
			assert.Contains(t, prompt, "## Phase 1: Codebase Analysis")
			assert.Contains(t, prompt, "## Phase 2: Documentation Structure Generation")
			assert.Contains(t, prompt, "## Phase 3: Knowledge Storage Strategy")
			assert.Contains(t, prompt, "## Output Format")
			assert.Contains(t, prompt, "## Quality Checklist")
		})
	}
}

func TestTemplateStorage(t *testing.T) {
	handler := NewDocumentationPromptHandler()

	// Test that all templates include storage instructions
	allTemplates := handler.buildComponentTemplates([]string{"all"})

	assert.Contains(t, allTemplates, "mcp__hyper__coordinator_upsert_knowledge")
	assert.Contains(t, allTemplates, "collection:")
	assert.Contains(t, allTemplates, "metadata:")
	assert.Contains(t, allTemplates, "knowledgeType:")
	assert.Contains(t, allTemplates, "title:")
	assert.Contains(t, allTemplates, "tags:")
}

func TestComponentTemplateStructure(t *testing.T) {
	handler := NewDocumentationPromptHandler()

	t.Run("ADR template structure", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"adrs"})

		assert.Contains(t, templates, "# ADR-NNN:")
		assert.Contains(t, templates, "**Date:**")
		assert.Contains(t, templates, "**Status:**")
		assert.Contains(t, templates, "## Context")
		assert.Contains(t, templates, "## Decision")
		assert.Contains(t, templates, "## Rationale")
		assert.Contains(t, templates, "## Consequences")
		assert.Contains(t, templates, "### Positive")
		assert.Contains(t, templates, "### Negative")
	})

	t.Run("API template structure", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"apis"})

		assert.Contains(t, templates, "**Base URL:**")
		assert.Contains(t, templates, "**Authentication:**")
		assert.Contains(t, templates, "## Endpoints")
		assert.Contains(t, templates, "### List Resources")
		assert.Contains(t, templates, "**Endpoint:**")
		assert.Contains(t, templates, "**Query Parameters:**")
		assert.Contains(t, templates, "**Response (200 OK):**")
	})

	t.Run("MCP template structure", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"mcps"})

		assert.Contains(t, templates, "## Overview")
		assert.Contains(t, templates, "**Service:**")
		assert.Contains(t, templates, "**MCP Endpoint:**")
		assert.Contains(t, templates, "## Tools")
		assert.Contains(t, templates, "**Parameters:**")
		assert.Contains(t, templates, "**Usage Example:**")
		assert.Contains(t, templates, "**Common Gotchas:**")
	})

	t.Run("Service template structure", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"services"})

		assert.Contains(t, templates, "## Overview")
		assert.Contains(t, templates, "**Purpose:**")
		assert.Contains(t, templates, "**Technology Stack:**")
		assert.Contains(t, templates, "## Architecture")
		assert.Contains(t, templates, "### Service Boundaries")
		assert.Contains(t, templates, "### Data Model")
		assert.Contains(t, templates, "### Events")
		assert.Contains(t, templates, "## Development")
		assert.Contains(t, templates, "## Deployment")
		assert.Contains(t, templates, "## Monitoring")
	})

	t.Run("Component template structure", func(t *testing.T) {
		templates := handler.buildComponentTemplates([]string{"components"})

		assert.Contains(t, templates, "## Overview")
		assert.Contains(t, templates, "## Props")
		assert.Contains(t, templates, "interface ComponentNameProps")
		assert.Contains(t, templates, "## Usage")
		assert.Contains(t, templates, "## Dependencies")
		assert.Contains(t, templates, "## Patterns")
		assert.Contains(t, templates, "### Optimistic Updates")
		assert.Contains(t, templates, "### Accessibility")
		assert.Contains(t, templates, "## Testing")
		assert.Contains(t, templates, "## Gotchas")
	})
}

func TestDocumentationPromptRegistration(t *testing.T) {
	handler := NewDocumentationPromptHandler()

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasPrompts: true,
	}

	server := mcp.NewServer(impl, opts)

	// Register should succeed
	err := handler.RegisterDocumentationPrompts(server)
	assert.NoError(t, err)

	// Should not panic on re-registration
	err = handler.RegisterDocumentationPrompts(server)
	assert.NoError(t, err)
}

func TestAgentAssignmentMatrix(t *testing.T) {
	handler := NewDocumentationPromptHandler()

	prompt := handler.buildKnowledgeBasePrompt(
		"/path/to/project",
		[]string{"all"},
		true,
	)

	// Verify agent assignment matrix is present
	assert.Contains(t, prompt, "Agent Assignment Matrix")
	assert.Contains(t, prompt, "go-dev")
	assert.Contains(t, prompt, "go-mcp-dev")
	assert.Contains(t, prompt, "ui-dev")
	assert.Contains(t, prompt, "Backend Services Specialist")
	assert.Contains(t, prompt, "Data Platform Specialist")
}

func TestKnowledgeStorageGuidance(t *testing.T) {
	handler := NewDocumentationPromptHandler()

	prompt := handler.buildKnowledgeBasePrompt(
		"/path/to/project",
		[]string{"all"},
		false,
	)

	// Verify knowledge storage guidance
	assert.Contains(t, prompt, "Collection Organization")
	assert.Contains(t, prompt, "technical-knowledge")
	assert.Contains(t, prompt, "code-patterns")
	assert.Contains(t, prompt, "adr")
	assert.Contains(t, prompt, "data-contracts")
	assert.Contains(t, prompt, "ui-component-patterns")
	assert.Contains(t, prompt, "Storage Format")
	assert.Contains(t, prompt, "mcp__hyper__coordinator_upsert_knowledge")
}

func TestCodebaseAnalysisInstructions(t *testing.T) {
	handler := NewDocumentationPromptHandler()

	prompt := handler.buildKnowledgeBasePrompt(
		"/path/to/project",
		[]string{"all"},
		false,
	)

	// Verify codebase analysis instructions
	assert.Contains(t, prompt, "Identify Services and APIs")
	assert.Contains(t, prompt, "Go Services")
	assert.Contains(t, prompt, "cmd/server/main.go")
	assert.Contains(t, prompt, "REST APIs")
	assert.Contains(t, prompt, "MCP Tools")
	assert.Contains(t, prompt, "mcp.AddTool()")
	assert.Contains(t, prompt, "React Components")
	assert.Contains(t, prompt, ".tsx")
	assert.Contains(t, prompt, "Data Models")
	assert.Contains(t, prompt, "MongoDB Schemas")
	assert.Contains(t, prompt, "Architecture Decisions")
}
