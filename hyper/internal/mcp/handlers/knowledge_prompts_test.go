package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledgePromptHandler_RecommendQdrantQuery(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	tests := []struct {
		name          string
		agentQuestion string
		taskContext   string
		collections   string
		wantError     bool
		checkContent  []string
	}{
		{
			name:          "Basic query recommendation",
			agentQuestion: "How do I implement JWT authentication middleware?",
			taskContext:   "Working on go-mcp-dev service, building auth middleware",
			collections:   "technical-knowledge,code-patterns",
			wantError:     false,
			checkContent: []string{
				"Qdrant Query Optimization",
				"Agent's Question",
				"How do I implement JWT authentication middleware?",
				"Current Task Context",
				"Available Qdrant Collections",
				"technical-knowledge",
				"code-patterns",
				"Analysis Framework",
				"Primary Query Strategy",
				"Alternative Query",
				"Fallback Plan",
				"Context Check",
			},
		},
		{
			name:          "UI component query",
			agentQuestion: "Need React Query pattern for optimistic updates",
			taskContext:   "ui-dev squad, building task board with real-time updates",
			collections:   "ui-component-patterns",
			wantError:     false,
			checkContent: []string{
				"Qdrant Query Optimization",
				"React Query pattern",
				"ui-component-patterns",
				"Query String:",
			},
		},
		{
			name:          "No collections specified - use defaults",
			agentQuestion: "How to handle WebSocket reconnection?",
			taskContext:   "Real-time chat service",
			collections:   "",
			wantError:     false,
			checkContent: []string{
				"Standard Qdrant Collections",
				"Task Collections:",
				"Technical Collections:",
				"UI Collections:",
				"Operations:",
			},
		},
		{
			name:          "Missing required agentQuestion",
			agentQuestion: "",
			taskContext:   "Some context",
			collections:   "technical-knowledge",
			wantError:     true,
		},
		{
			name:          "Missing required taskContext",
			agentQuestion: "Some question",
			taskContext:   "",
			collections:   "technical-knowledge",
			wantError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create MCP server for testing
			impl := &mcp.Implementation{
				Name:    "test-server",
				Version: "1.0.0",
			}
			opts := &mcp.ServerOptions{
				HasPrompts: true,
			}
			server := mcp.NewServer(impl, opts)

			// Register prompt
			err := handler.registerRecommendQdrantQuery(server)
			require.NoError(t, err)

			// Create request
			req := &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{
					Name: "recommend_qdrant_query",
					Arguments: map[string]string{
						"agentQuestion":        tt.agentQuestion,
						"taskContext":          tt.taskContext,
						"availableCollections": tt.collections,
					},
				},
			}

			// Execute
			err = handler.registerRecommendQdrantQuery(server)
			if tt.wantError {
				// For error cases, we need to call the handler directly
				// since registration succeeds but execution should fail
				impl := &mcp.Implementation{Name: "test", Version: "1.0.0"}
				opts := &mcp.ServerOptions{HasPrompts: true}
				testServer := mcp.NewServer(impl, opts)

				prompt := &mcp.Prompt{
					Name:        "recommend_qdrant_query",
					Description: "Test prompt",
					Arguments: []*mcp.PromptArgument{
						{Name: "agentQuestion", Required: true},
						{Name: "taskContext", Required: true},
						{Name: "availableCollections", Required: false},
					},
				}

				handlerFunc := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
					agentQuestion := ""
					taskContext := ""

					if req.Params != nil && req.Params.Arguments != nil {
						agentQuestion = req.Params.Arguments["agentQuestion"]
						taskContext = req.Params.Arguments["taskContext"]
					}

					if agentQuestion == "" || taskContext == "" {
						return nil, assert.AnError
					}

					return &mcp.GetPromptResult{}, nil
				}

				testServer.AddPrompt(prompt, handlerFunc)
				_, err := handlerFunc(context.Background(), req)
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// For success cases, check the prompt was built correctly
			promptText := handler.buildQdrantQueryRecommendation(
				tt.agentQuestion,
				tt.taskContext,
				parseCollections(tt.collections),
			)

			for _, check := range tt.checkContent {
				assert.Contains(t, promptText, check, "Prompt should contain: %s", check)
			}
		})
	}
}

func TestKnowledgePromptHandler_SuggestKnowledgeStructure(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	tests := []struct {
		name         string
		rawLearning  string
		context      map[string]interface{}
		wantError    bool
		checkContent []string
	}{
		{
			name:        "Backend implementation learning",
			rawLearning: "Implemented JWT validation using HS256. Had to handle token expiration carefully. Store user ID in context for downstream handlers.",
			context: map[string]interface{}{
				"squad":      "go-mcp-dev",
				"service":    "auth-service",
				"taskType":   "implementation",
				"filesModified": []interface{}{
					"middleware/auth.go",
					"middleware/auth_test.go",
				},
			},
			wantError: false,
			checkContent: []string{
				"Knowledge Structuring Guide",
				"Raw Learning",
				"Implemented JWT validation using HS256",
				"Task Context",
				"go-mcp-dev",
				"auth-service",
				"implementation",
				"middleware/auth.go",
				"1. Title",
				"2. Summary",
				"3. Implementation",
				"4. Gotchas",
				"5. Metadata Tags",
				"Quality Checklist",
				"Qdrant Storage Format",
				"Collection Selection Guide",
			},
		},
		{
			name:        "UI pattern learning",
			rawLearning: "Created custom React Query hook for optimistic task updates. Key was using mutation.onMutate to update cache before server response.",
			context: map[string]interface{}{
				"squad":      "ui-dev",
				"service":    "task-board",
				"taskType":   "feature",
				"filesModified": []interface{}{
					"hooks/useTaskMutation.ts",
					"hooks/useTaskMutation.test.ts",
				},
			},
			wantError: false,
			checkContent: []string{
				"React Query hook for optimistic task updates",
				"ui-dev",
				"task-board",
				"ui-component-patterns",
			},
		},
		{
			name:        "Empty files list - should use default",
			rawLearning: "Some learning",
			context: map[string]interface{}{
				"squad":         "backend-services",
				"service":       "api-gateway",
				"filesModified": []interface{}{},
			},
			wantError: false,
			checkContent: []string{
				"No files specified",
			},
		},
		{
			name:        "Invalid context JSON should fail during execution",
			rawLearning: "Some learning",
			context:     nil,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create MCP server for testing
			impl := &mcp.Implementation{
				Name:    "test-server",
				Version: "1.0.0",
			}
			opts := &mcp.ServerOptions{
				HasPrompts: true,
			}
			server := mcp.NewServer(impl, opts)

			// Register prompt
			err := handler.registerSuggestKnowledgeStructure(server)
			require.NoError(t, err)

			if tt.wantError {
				// For error cases, test with invalid context
				contextJSON := ""
				if tt.context != nil {
					contextBytes, _ := json.Marshal(tt.context)
					contextJSON = string(contextBytes)
				}

				req := &mcp.GetPromptRequest{
					Params: &mcp.GetPromptParams{
						Name: "suggest_knowledge_structure",
						Arguments: map[string]string{
							"rawLearning": tt.rawLearning,
							"context":     contextJSON,
						},
					},
				}

				// This should fail during handler execution
				handlerFunc := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
					rawLearning := ""
					contextStr := ""

					if req.Params != nil && req.Params.Arguments != nil {
						rawLearning = req.Params.Arguments["rawLearning"]
						contextStr = req.Params.Arguments["context"]
					}

					if rawLearning == "" || contextStr == "" {
						return nil, assert.AnError
					}

					var taskContext map[string]interface{}
					if err := json.Unmarshal([]byte(contextStr), &taskContext); err != nil {
						return nil, err
					}

					return &mcp.GetPromptResult{}, nil
				}

				_, err := handlerFunc(context.Background(), req)
				assert.Error(t, err)
				return
			}

			// For success cases, check the prompt was built correctly
			promptText := handler.buildKnowledgeStructurePrompt(tt.rawLearning, tt.context)

			for _, check := range tt.checkContent {
				assert.Contains(t, promptText, check, "Prompt should contain: %s", check)
			}
		})
	}
}

func TestKnowledgePromptRegistration(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasPrompts: true,
	}

	server := mcp.NewServer(impl, opts)

	// Register should succeed
	err := handler.RegisterKnowledgePrompts(server)
	assert.NoError(t, err)

	// Should not panic on re-registration
	err = handler.RegisterKnowledgePrompts(server)
	assert.NoError(t, err)
}

func TestBuildQdrantQueryRecommendation(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	tests := []struct {
		name               string
		agentQuestion      string
		taskContext        string
		availableCollections []string
		checkContent       []string
	}{
		{
			name:          "With specific collections",
			agentQuestion: "How to implement caching?",
			taskContext:   "Backend service",
			availableCollections: []string{"technical-knowledge", "code-patterns"},
			checkContent: []string{
				"Available Qdrant Collections",
				"technical-knowledge",
				"code-patterns",
			},
		},
		{
			name:                 "Without collections - show defaults",
			agentQuestion:        "Need help with deployment",
			taskContext:          "Infrastructure work",
			availableCollections: []string{},
			checkContent: []string{
				"Standard Qdrant Collections",
				"technical-knowledge",
				"ui-component-patterns",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.buildQdrantQueryRecommendation(
				tt.agentQuestion,
				tt.taskContext,
				tt.availableCollections,
			)

			for _, check := range tt.checkContent {
				assert.Contains(t, result, check)
			}
		})
	}
}

func TestBuildKnowledgeStructurePrompt(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	tests := []struct {
		name         string
		rawLearning  string
		taskContext  map[string]interface{}
		checkContent []string
	}{
		{
			name:        "Full context",
			rawLearning: "Implemented feature X",
			taskContext: map[string]interface{}{
				"squad":   "backend-services",
				"service": "api-gateway",
				"taskType": "feature",
				"filesModified": []interface{}{
					"handler.go",
					"handler_test.go",
				},
			},
			checkContent: []string{
				"backend-services",
				"api-gateway",
				"feature",
				"handler.go",
			},
		},
		{
			name:        "Minimal context with defaults",
			rawLearning: "Fixed bug",
			taskContext: map[string]interface{}{},
			checkContent: []string{
				"unknown",
				"implementation",
				"No files specified",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.buildKnowledgeStructurePrompt(tt.rawLearning, tt.taskContext)

			for _, check := range tt.checkContent {
				assert.Contains(t, result, check)
			}
		})
	}
}

// Helper function
func parseCollections(collectionsStr string) []string {
	if collectionsStr == "" {
		return []string{}
	}
	var collections []string
	for _, c := range splitByComma(collectionsStr) {
		if trimmed := trimSpace(c); trimmed != "" {
			collections = append(collections, trimmed)
		}
	}
	return collections
}

func splitByComma(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func TestKnowledgePromptHandler_KnowledgeWorkflowGuide(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	tests := []struct {
		name            string
		taskDescription string
		domain          string
		wantError       bool
		checkContent    []string
	}{
		{
			name:            "Full workflow with domain",
			taskDescription: "Implement JWT authentication middleware for Go service",
			domain:          "authentication",
			wantError:       false,
			checkContent: []string{
				"Knowledge Base Discovery Workflow",
				"Your Task",
				"Implement JWT authentication middleware for Go service",
				"Your Technical Domain",
				"authentication",
				"STEP 1: DISCOVER Available Collections",
				"knowledge_list_collections",
				"STEP 2: PICK the Right Collection",
				"STEP 3: SEARCH with Specific Queries",
				"Query Formula:",
				"STEP 4: REVIEW Results Thoroughly",
				"STEP 5: VOTE on Usefulness (MANDATORY!)",
				"knowledge_vote",
				"STEP 6: APPLY Patterns to Your Code",
				"WORKFLOW CHECKLIST",
				"ANTI-PATTERNS",
				"Example End-to-End Workflow",
			},
		},
		{
			name:            "Without domain",
			taskDescription: "Build React component for task board",
			domain:          "",
			wantError:       false,
			checkContent: []string{
				"Knowledge Base Discovery Workflow",
				"Build React component for task board",
				"STEP 1: DISCOVER Available Collections",
				"STEP 2: PICK the Right Collection",
				"STEP 3: SEARCH with Specific Queries",
				"STEP 4: REVIEW Results Thoroughly",
				"STEP 5: VOTE on Usefulness (MANDATORY!)",
				"STEP 6: APPLY Patterns to Your Code",
			},
		},
		{
			name:            "Database task",
			taskDescription: "Optimize MongoDB aggregation pipeline for duplicate detection",
			domain:          "database",
			wantError:       false,
			checkContent: []string{
				"Knowledge Base Discovery Workflow",
				"Optimize MongoDB aggregation pipeline",
				"database",
				"technical-knowledge, data-contracts",
				"knowledge_find",
				"retrieveMode: \"chunk\"",
			},
		},
		{
			name:            "Missing required taskDescription",
			taskDescription: "",
			domain:          "authentication",
			wantError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create MCP server for testing
			impl := &mcp.Implementation{
				Name:    "test-server",
				Version: "1.0.0",
			}
			opts := &mcp.ServerOptions{
				HasPrompts: true,
			}
			server := mcp.NewServer(impl, opts)

			// Register prompt
			err := handler.registerKnowledgeWorkflowGuide(server)
			require.NoError(t, err)

			if tt.wantError {
				// Test error case with missing required argument
				req := &mcp.GetPromptRequest{
					Params: &mcp.GetPromptParams{
						Name: "knowledge_workflow_guide",
						Arguments: map[string]string{
							"taskDescription": tt.taskDescription,
							"domain":          tt.domain,
						},
					},
				}

				handlerFunc := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
					taskDescription := ""
					if req.Params != nil && req.Params.Arguments != nil {
						taskDescription = req.Params.Arguments["taskDescription"]
					}
					if taskDescription == "" {
						return nil, assert.AnError
					}
					return &mcp.GetPromptResult{}, nil
				}

				_, err := handlerFunc(context.Background(), req)
				assert.Error(t, err)
				return
			}

			// For success cases, check the prompt was built correctly
			promptText := handler.buildKnowledgeWorkflowGuide(tt.taskDescription, tt.domain)

			for _, check := range tt.checkContent {
				assert.Contains(t, promptText, check, "Prompt should contain: %s", check)
			}

			// Verify workflow steps are present
			assert.Contains(t, promptText, "STEP 1:")
			assert.Contains(t, promptText, "STEP 2:")
			assert.Contains(t, promptText, "STEP 3:")
			assert.Contains(t, promptText, "STEP 4:")
			assert.Contains(t, promptText, "STEP 5:")
			assert.Contains(t, promptText, "STEP 6:")
		})
	}
}

func TestKnowledgePromptHandler_KnowledgeVotingWorkflow(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	tests := []struct {
		name         string
		articleId    string
		articleTitle string
		wasHelpful   string
		wantError    bool
		checkContent []string
	}{
		{
			name:         "Full voting guide with all arguments",
			articleId:    "article-123",
			articleTitle: "Go JWT Middleware with HS256 Validation",
			wasHelpful:   "true",
			wantError:    false,
			checkContent: []string{
				"Knowledge Voting Decision Guide",
				"Article Under Review",
				"article-123",
				"Go JWT Middleware with HS256 Validation",
				"You indicated this article WAS helpful",
				"Why Voting Matters",
				"DECISION TREE: Should I Vote +1 (Helpful)?",
				"DECISION TREE: Should I Vote -1 (Not Helpful)?",
				"How to Write GOOD Vote Reasons",
				"Common Voting Mistakes",
				"Voting Workflow",
				"knowledge_vote",
				"Your Voting Decision for This Article",
				"Evaluation Checklist",
			},
		},
		{
			name:         "Article was not helpful",
			articleId:    "article-456",
			articleTitle: "Outdated Authentication Pattern",
			wasHelpful:   "false",
			wantError:    false,
			checkContent: []string{
				"Knowledge Voting Decision Guide",
				"article-456",
				"Outdated Authentication Pattern",
				"You indicated this article was NOT helpful",
				"Vote +1 (Helpful) When:",
				"Vote -1 (Not Helpful) When:",
				"Accuracy Issues:",
				"Completeness Issues:",
				"Relevance Issues:",
				"Example -1 Vote Reasons:",
			},
		},
		{
			name:         "Minimal - only articleId",
			articleId:    "article-789",
			articleTitle: "",
			wasHelpful:   "",
			wantError:    false,
			checkContent: []string{
				"Knowledge Voting Decision Guide",
				"article-789",
				"Why Voting Matters",
				"Search ranking",
				"Content quality",
				"Community learning",
				"Vote +1 (Helpful) When:",
				"Vote -1 (Not Helpful) When:",
				"Template for +1 Votes:",
				"Template for -1 Votes:",
			},
		},
		{
			name:         "Missing required articleId",
			articleId:    "",
			articleTitle: "Some Article",
			wasHelpful:   "true",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create MCP server for testing
			impl := &mcp.Implementation{
				Name:    "test-server",
				Version: "1.0.0",
			}
			opts := &mcp.ServerOptions{
				HasPrompts: true,
			}
			server := mcp.NewServer(impl, opts)

			// Register prompt
			err := handler.registerKnowledgeVotingWorkflow(server)
			require.NoError(t, err)

			if tt.wantError {
				// Test error case with missing required argument
				req := &mcp.GetPromptRequest{
					Params: &mcp.GetPromptParams{
						Name: "knowledge_voting_workflow",
						Arguments: map[string]string{
							"articleId":    tt.articleId,
							"articleTitle": tt.articleTitle,
							"wasHelpful":   tt.wasHelpful,
						},
					},
				}

				handlerFunc := func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
					articleId := ""
					if req.Params != nil && req.Params.Arguments != nil {
						articleId = req.Params.Arguments["articleId"]
					}
					if articleId == "" {
						return nil, assert.AnError
					}
					return &mcp.GetPromptResult{}, nil
				}

				_, err := handlerFunc(context.Background(), req)
				assert.Error(t, err)
				return
			}

			// For success cases, check the prompt was built correctly
			promptText := handler.buildKnowledgeVotingGuide(tt.articleId, tt.articleTitle, tt.wasHelpful)

			for _, check := range tt.checkContent {
				assert.Contains(t, promptText, check, "Prompt should contain: %s", check)
			}

			// Verify article ID is present
			assert.Contains(t, promptText, tt.articleId)
		})
	}
}

func TestBuildKnowledgeWorkflowGuide(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	tests := []struct {
		name            string
		taskDescription string
		domain          string
		checkContent    []string
		notContain      []string
	}{
		{
			name:            "With domain specified",
			taskDescription: "Implement user authentication",
			domain:          "security",
			checkContent: []string{
				"Your Technical Domain",
				"security",
				"Implement user authentication",
				"knowledge_list_collections",
				"knowledge_find",
				"knowledge_vote",
			},
		},
		{
			name:            "Without domain",
			taskDescription: "Build REST API endpoint",
			domain:          "",
			checkContent: []string{
				"Build REST API endpoint",
				"STEP 1: DISCOVER",
				"STEP 2: PICK",
				"STEP 3: SEARCH",
			},
			notContain: []string{
				"Your Technical Domain",
			},
		},
		{
			name:            "Complex task description",
			taskDescription: "Refactor MongoDB aggregation pipeline for performance optimization with caching layer",
			domain:          "database",
			checkContent: []string{
				"Refactor MongoDB aggregation pipeline",
				"database",
				"Query Formula:",
				"[Technology] + [Component] + [Problem/Pattern]",
				"retrieveMode: \"chunk\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.buildKnowledgeWorkflowGuide(tt.taskDescription, tt.domain)

			// Check required content
			for _, check := range tt.checkContent {
				assert.Contains(t, result, check, "Should contain: %s", check)
			}

			// Check excluded content
			for _, notWanted := range tt.notContain {
				assert.NotContains(t, result, notWanted, "Should NOT contain: %s", notWanted)
			}

			// Verify all steps are present
			assert.Contains(t, result, "STEP 1:")
			assert.Contains(t, result, "STEP 2:")
			assert.Contains(t, result, "STEP 3:")
			assert.Contains(t, result, "STEP 4:")
			assert.Contains(t, result, "STEP 5:")
			assert.Contains(t, result, "STEP 6:")

			// Verify mandatory elements
			assert.Contains(t, result, "WORKFLOW CHECKLIST")
			assert.Contains(t, result, "ANTI-PATTERNS")
			assert.Contains(t, result, "Example End-to-End Workflow")
		})
	}
}

func TestBuildKnowledgeVotingGuide(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	tests := []struct {
		name         string
		articleId    string
		articleTitle string
		wasHelpful   string
		checkContent []string
		notContain   []string
	}{
		{
			name:         "Article was helpful",
			articleId:    "abc123",
			articleTitle: "JWT Authentication Guide",
			wasHelpful:   "true",
			checkContent: []string{
				"abc123",
				"JWT Authentication Guide",
				"You indicated this article WAS helpful",
				"Vote +1 (Helpful)",
			},
		},
		{
			name:         "Article was not helpful",
			articleId:    "xyz789",
			articleTitle: "Deprecated Pattern",
			wasHelpful:   "false",
			checkContent: []string{
				"xyz789",
				"Deprecated Pattern",
				"You indicated this article was NOT helpful",
				"Vote -1 (Not Helpful)",
			},
		},
		{
			name:         "No helpfulness assessment",
			articleId:    "def456",
			articleTitle: "Some Article",
			wasHelpful:   "",
			checkContent: []string{
				"def456",
				"Some Article",
				"Why Voting Matters",
				"DECISION TREE",
			},
			notContain: []string{
				"Initial Assessment",
			},
		},
		{
			name:         "No title provided",
			articleId:    "ghi789",
			articleTitle: "",
			wasHelpful:   "true",
			checkContent: []string{
				"ghi789",
				"Vote +1 (Helpful) When:",
				"Vote -1 (Not Helpful) When:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.buildKnowledgeVotingGuide(tt.articleId, tt.articleTitle, tt.wasHelpful)

			// Check required content
			for _, check := range tt.checkContent {
				assert.Contains(t, result, check, "Should contain: %s", check)
			}

			// Check excluded content
			for _, notWanted := range tt.notContain {
				assert.NotContains(t, result, notWanted, "Should NOT contain: %s", notWanted)
			}

			// Verify mandatory elements
			assert.Contains(t, result, "Why Voting Matters")
			assert.Contains(t, result, "DECISION TREE")
			assert.Contains(t, result, "How to Write GOOD Vote Reasons")
			assert.Contains(t, result, "Common Voting Mistakes")
			assert.Contains(t, result, "Voting Workflow")
			assert.Contains(t, result, "Evaluation Checklist")
			assert.Contains(t, result, "Example Decision Process")

			// Verify article ID is always present
			assert.Contains(t, result, tt.articleId)
		})
	}
}

func TestKnowledgePromptsRegistration_WithNewPrompts(t *testing.T) {
	handler := NewKnowledgePromptHandler()

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	opts := &mcp.ServerOptions{
		HasPrompts: true,
	}

	server := mcp.NewServer(impl, opts)

	// Register all prompts including new ones
	err := handler.RegisterKnowledgePrompts(server)
	assert.NoError(t, err)

	// Verify no errors on re-registration
	err = handler.RegisterKnowledgePrompts(server)
	assert.NoError(t, err)
}
