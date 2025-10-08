package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hyperion-coordinator-mcp/storage"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// KnowledgeResourceHandler manages knowledge-related MCP resources
type KnowledgeResourceHandler struct {
	knowledgeStorage storage.KnowledgeStorage
}

// NewKnowledgeResourceHandler creates a new knowledge resource handler
func NewKnowledgeResourceHandler(knowledgeStorage storage.KnowledgeStorage) *KnowledgeResourceHandler {
	return &KnowledgeResourceHandler{
		knowledgeStorage: knowledgeStorage,
	}
}

// CollectionInfo represents metadata about a Qdrant collection
type CollectionInfo struct {
	Name         string   `json:"name"`
	Purpose      string   `json:"purpose"`
	ExampleQuery string   `json:"exampleQuery"`
	UseCases     []string `json:"useCases"`
	Category     string   `json:"category"`
}

// RecentLearning represents a recently stored knowledge entry
type RecentLearning struct {
	ID         string                 `json:"id"`
	Collection string                 `json:"collection"`
	Topic      string                 `json:"topic"`
	Text       string                 `json:"text"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"createdAt"`
	AgentName  string                 `json:"agentName,omitempty"`
}

// RegisterKnowledgeResources registers all knowledge resources with the MCP server
func (h *KnowledgeResourceHandler) RegisterKnowledgeResources(server *mcp.Server) error {
	// Register collections resource
	collectionsResource := &mcp.Resource{
		URI:         "hyperion://knowledge/collections",
		Name:        "Qdrant Collections Directory",
		Description: "Complete list of all Qdrant knowledge collections with metadata, purpose, and example queries",
		MIMEType:    "application/json",
	}
	server.AddResource(collectionsResource, h.handleCollectionsResource)

	// Register recent learnings resource
	recentLearningsResource := &mcp.Resource{
		URI:         "hyperion://knowledge/recent-learnings",
		Name:        "Recent Knowledge Learnings",
		Description: "Knowledge entries stored in the last 24 hours, grouped by collection and source",
		MIMEType:    "application/json",
	}
	server.AddResource(recentLearningsResource, h.handleRecentLearningsResource)

	return nil
}

// handleCollectionsResource returns the complete directory of Qdrant collections
func (h *KnowledgeResourceHandler) handleCollectionsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Define all known Qdrant collections with metadata
	collections := []CollectionInfo{
		// Task Collections
		{
			Name:         "task:hyperion://task/human/{taskId}",
			Category:     "Task",
			Purpose:      "Task-specific knowledge for multi-phase work and progressive handoffs",
			ExampleQuery: "coordinator_query_knowledge with task URI",
			UseCases: []string{
				"Store task-specific solutions and decisions",
				"Document API contracts for next agent",
				"Track phase completion and handoff notes",
			},
		},
		{
			Name:         "team-coordination",
			Category:     "Task",
			Purpose:      "Cross-squad coordination and communication",
			ExampleQuery: "qdrant-find: 'API contract change authentication'",
			UseCases: []string{
				"Announce breaking API changes",
				"Coordinate cross-service features",
				"Share critical bug discoveries",
			},
		},
		{
			Name:         "agent-coordination",
			Category:     "Task",
			Purpose:      "Agent-to-agent workflow coordination",
			ExampleQuery: "qdrant-find: 'handoff Phase 2 implementation'",
			UseCases: []string{
				"Document work handoffs between phases",
				"Share integration patterns",
				"Communicate blocking issues",
			},
		},

		// Technical Collections
		{
			Name:         "technical-knowledge",
			Category:     "Tech",
			Purpose:      "General technical patterns and solutions",
			ExampleQuery: "qdrant-find: 'JWT middleware implementation Go'",
			UseCases: []string{
				"Store reusable code patterns",
				"Document architectural decisions",
				"Share bug fixes and workarounds",
			},
		},
		{
			Name:         "code-patterns",
			Category:     "Tech",
			Purpose:      "Specific code implementation patterns",
			ExampleQuery: "qdrant-find: 'dependency injection pattern service container'",
			UseCases: []string{
				"Language-specific patterns",
				"Framework best practices",
				"Design pattern implementations",
			},
		},
		{
			Name:         "adr",
			Category:     "Tech",
			Purpose:      "Architecture Decision Records",
			ExampleQuery: "qdrant-find: 'ADR microservices communication NATS'",
			UseCases: []string{
				"Document why architectural choices were made",
				"Track technology selection rationale",
				"Reference historical context for decisions",
			},
		},
		{
			Name:         "data-contracts",
			Category:     "Tech",
			Purpose:      "API and data structure contracts",
			ExampleQuery: "qdrant-find: 'task API response schema'",
			UseCases: []string{
				"Define API request/response schemas",
				"Document service interfaces",
				"Track contract versioning",
			},
		},
		{
			Name:         "technical-debt-registry",
			Category:     "Tech",
			Purpose:      "Technical debt tracking and monitoring",
			ExampleQuery: "qdrant-find: 'god class refactoring backend'",
			UseCases: []string{
				"Track files exceeding size limits",
				"Document refactoring needs",
				"Monitor code quality violations",
			},
		},

		// UI Collections
		{
			Name:         "ui-component-patterns",
			Category:     "UI",
			Purpose:      "React component patterns and architecture",
			ExampleQuery: "qdrant-find: 'task board optimistic updates React'",
			UseCases: []string{
				"Reusable component implementations",
				"State management patterns",
				"Performance optimization techniques",
			},
		},
		{
			Name:         "ui-test-strategies",
			Category:     "UI",
			Purpose:      "Frontend testing patterns with Playwright",
			ExampleQuery: "qdrant-find: 'Playwright drag-drop testing'",
			UseCases: []string{
				"E2E test patterns",
				"Visual regression strategies",
				"Test data management",
			},
		},
		{
			Name:         "ui-accessibility-standards",
			Category:     "UI",
			Purpose:      "WCAG 2.1 AA compliance and accessibility patterns",
			ExampleQuery: "qdrant-find: 'screen reader modal dialog'",
			UseCases: []string{
				"ARIA attribute patterns",
				"Keyboard navigation implementations",
				"Screen reader compatibility fixes",
			},
		},
		{
			Name:         "ui-visual-regression-baseline",
			Category:     "UI",
			Purpose:      "Visual regression test baselines and snapshots",
			ExampleQuery: "qdrant-find: 'visual baseline task card'",
			UseCases: []string{
				"Store component visual baselines",
				"Track UI change history",
				"Document intentional visual changes",
			},
		},

		// Operations Collections
		{
			Name:         "mcp-operations",
			Category:     "Ops",
			Purpose:      "MCP server operations and troubleshooting",
			ExampleQuery: "qdrant-find: 'MCP tool registration error'",
			UseCases: []string{
				"MCP integration patterns",
				"Tool implementation examples",
				"Common MCP errors and fixes",
			},
		},
		{
			Name:         "code-quality-violations",
			Category:     "Ops",
			Purpose:      "Code quality issues and enforcement",
			ExampleQuery: "qdrant-find: 'god class violation handlers'",
			UseCases: []string{
				"Track quality gate violations",
				"Document refactoring needs",
				"Monitor code complexity trends",
			},
		},
	}

	// Get actual collections from storage and merge with metadata
	actualCollections := h.knowledgeStorage.ListCollections()
	collectionMap := make(map[string]bool)
	for _, c := range actualCollections {
		collectionMap[c] = true
	}

	// Add flag for collections that have actual data
	type CollectionWithStatus struct {
		CollectionInfo
		HasData bool `json:"hasData"`
	}

	collectionsWithStatus := make([]CollectionWithStatus, 0, len(collections))
	for _, c := range collections {
		// Check if collection has actual data (skip task URIs as they're dynamic)
		hasData := false
		if c.Name != "task:hyperion://task/human/{taskId}" {
			hasData = collectionMap[c.Name]
		}

		collectionsWithStatus = append(collectionsWithStatus, CollectionWithStatus{
			CollectionInfo: c,
			HasData:        hasData,
		})
	}

	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"collections": collectionsWithStatus,
		"totalDefined": len(collections),
		"totalWithData": len(actualCollections),
		"lastUpdated": time.Now().UTC(),
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collections data: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "hyperion://knowledge/collections",
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// handleRecentLearningsResource returns knowledge entries from the last 24 hours
func (h *KnowledgeResourceHandler) handleRecentLearningsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Query for entries created in last 24 hours
	twentyFourHoursAgo := time.Now().UTC().Add(-24 * time.Hour)

	// Get all collections and query each for recent entries
	collections := h.knowledgeStorage.ListCollections()
	var recentLearnings []RecentLearning

	for _, collection := range collections {
		// Query this collection with a broad search to get recent entries
		// We'll filter by time after retrieval
		results, err := h.knowledgeStorage.Query(collection, "recent", 50) // Get up to 50 per collection
		if err != nil {
			// If search fails, try empty query
			results, err = h.knowledgeStorage.Query(collection, "", 50)
			if err != nil {
				continue // Skip collections with errors
			}
		}

		for _, result := range results {
			// Filter by creation time
			if result.Entry.CreatedAt.After(twentyFourHoursAgo) {
				// Extract topic from metadata or first 100 chars of text
				topic := ""
				if title, ok := result.Entry.Metadata["title"].(string); ok {
					topic = title
				} else if len(result.Entry.Text) > 100 {
					topic = result.Entry.Text[:100] + "..."
				} else {
					topic = result.Entry.Text
				}

				// Extract agent name from metadata
				agentName := ""
				if agent, ok := result.Entry.Metadata["agentName"].(string); ok {
					agentName = agent
				}

				recentLearnings = append(recentLearnings, RecentLearning{
					ID:         result.Entry.ID,
					Collection: result.Entry.Collection,
					Topic:      topic,
					Text:       result.Entry.Text,
					Metadata:   result.Entry.Metadata,
					CreatedAt:  result.Entry.CreatedAt,
					AgentName:  agentName,
				})
			}
		}
	}

	// Group by collection
	byCollection := make(map[string][]RecentLearning)
	for _, learning := range recentLearnings {
		byCollection[learning.Collection] = append(byCollection[learning.Collection], learning)
	}

	jsonData, err := json.MarshalIndent(map[string]interface{}{
		"timeRange": map[string]string{
			"start": twentyFourHoursAgo.Format(time.RFC3339),
			"end":   time.Now().UTC().Format(time.RFC3339),
		},
		"totalEntries":    len(recentLearnings),
		"byCollection":    byCollection,
		"allEntries":      recentLearnings,
		"collectionsWithActivity": len(byCollection),
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal recent learnings data: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "hyperion://knowledge/recent-learnings",
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}
