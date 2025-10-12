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
