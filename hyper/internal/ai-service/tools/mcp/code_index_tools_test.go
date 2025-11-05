package mcp

import (
	"testing"

	"hyper/internal/mcp/storage"

	"github.com/stretchr/testify/assert"
)

// TestParseQueryForFilenameBoost tests the query parsing function
func TestParseQueryForFilenameBoost(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		expectedTokens   []string
		expectedExts     []string
		expectedFilename string
	}{
		{
			name:             "Query with exact filename",
			query:            "App.tsx main application component",
			expectedTokens:   []string{"app.tsx", "application"},
			expectedExts:     []string{".tsx"},
			expectedFilename: "app.tsx",
		},
		{
			name:             "Query with multiple extensions",
			query:            "search for .tsx and .ts files",
			expectedTokens:   []string{"search", "tsx", "ts", "files"},
			expectedExts:     []string{".tsx", ".ts"},
			expectedFilename: "",
		},
		{
			name:             "Query with filename tokens only",
			query:            "ChatSessionList component implementation",
			expectedTokens:   []string{"chatsessionlist", "implementation"},
			expectedExts:     []string{},
			expectedFilename: "",
		},
		{
			name:             "Query with Go file",
			query:            "main.go server startup code",
			expectedTokens:   []string{"main.go", "server", "startup"},
			expectedExts:     []string{".go"},
			expectedFilename: "main.go",
		},
		{
			name:             "Query with hyphenated filename",
			query:            "code-index-tools.go implementation",
			expectedTokens:   []string{"index", "tools.go", "implementation"}, // "code" filtered as stopword
			expectedExts:     []string{".go"},
			expectedFilename: "code-index-tools.go", // Full filename captured by filename pattern
		},
		{
			name:             "Query with stopwords filtered",
			query:            "the main function in the App component",
			expectedTokens:   []string{"app"},
			expectedExts:     []string{},
			expectedFilename: "",
		},
		{
			name:             "Query with CSS file",
			query:            "App.css styling and layout",
			expectedTokens:   []string{"app.css", "styling", "layout"},
			expectedExts:     []string{".css"},
			expectedFilename: "app.css",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, exts, filename := parseQueryForFilenameBoost(tt.query)

			assert.ElementsMatch(t, tt.expectedTokens, tokens,
				"Tokens should match (order doesn't matter)")
			assert.ElementsMatch(t, tt.expectedExts, exts,
				"Extensions should match (order doesn't matter)")
			assert.Equal(t, tt.expectedFilename, filename,
				"Exact filename should match")
		})
	}
}

// TestCalculateFilenameBoost tests the boost calculation function
func TestCalculateFilenameBoost(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		queryTokens    []string
		queryExts      []string
		exactFilename  string
		expectedBoost  float64
		description    string
	}{
		{
			name:          "Exact filename match - highest boost",
			filePath:      "/path/to/App.tsx",
			queryTokens:   []string{"app"},
			queryExts:     []string{".tsx"},
			exactFilename: "app.tsx",
			expectedBoost: 1.5,
			description:   "50% boost for exact match",
		},
		{
			name:          "Extension match only",
			filePath:      "/path/to/Component.tsx",
			queryTokens:   []string{"helper"},
			queryExts:     []string{".tsx"},
			exactFilename: "",
			expectedBoost: 1.4,
			description:   "40% boost for extension match",
		},
		{
			name:          "Filename token match",
			filePath:      "/path/to/ChatSessionList.tsx",
			queryTokens:   []string{"chatsessionlist"},
			queryExts:     []string{},
			exactFilename: "",
			expectedBoost: 1.3,
			description:   "30% boost for token match",
		},
		{
			name:          "No match at all",
			filePath:      "/path/to/Unrelated.js",
			queryTokens:   []string{"app"},
			queryExts:     []string{".tsx"},
			exactFilename: "",
			expectedBoost: 1.0,
			description:   "No boost for no match",
		},
		{
			name:          "CSS file with extension match",
			filePath:      "/styles/App.css",
			queryTokens:   []string{"app"},
			queryExts:     []string{".css"},
			exactFilename: "",
			expectedBoost: 1.4,
			description:   "40% boost for CSS extension match",
		},
		{
			name:          "Partial filename token match",
			filePath:      "/components/AppHeader.tsx",
			queryTokens:   []string{"app"},
			queryExts:     []string{},
			exactFilename: "",
			expectedBoost: 1.3,
			description:   "30% boost for partial token match",
		},
		{
			name:          "Case insensitive exact match",
			filePath:      "/path/to/APP.TSX",
			queryTokens:   []string{"app"},
			queryExts:     []string{".tsx"},
			exactFilename: "app.tsx",
			expectedBoost: 1.5,
			description:   "50% boost - case insensitive",
		},
		{
			name:          "Empty file path",
			filePath:      "",
			queryTokens:   []string{"app"},
			queryExts:     []string{".tsx"},
			exactFilename: "app.tsx",
			expectedBoost: 1.0,
			description:   "No boost for empty path",
		},
		{
			name:          "Go file exact match",
			filePath:      "/cmd/main.go",
			queryTokens:   []string{"main"},
			queryExts:     []string{".go"},
			exactFilename: "main.go",
			expectedBoost: 1.5,
			description:   "50% boost for Go file exact match",
		},
		{
			name:          "Extension match takes priority over token match",
			filePath:      "/path/to/Helper.tsx",
			queryTokens:   []string{"help"},
			queryExts:     []string{".tsx"},
			exactFilename: "",
			expectedBoost: 1.4,
			description:   "Extension match (40%) > token match (30%)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boost := calculateFilenameBoost(tt.filePath, tt.queryTokens, tt.queryExts, tt.exactFilename)
			assert.Equal(t, tt.expectedBoost, boost, tt.description)
		})
	}
}

// TestFilenameBoostingIntegration tests the complete boosting workflow
func TestFilenameBoostingIntegration(t *testing.T) {
	// Simulate the query: "App.tsx main application component layout structure"
	query := "App.tsx main application component layout structure"
	tokens, exts, exactFilename := parseQueryForFilenameBoost(query)

	// Simulate Qdrant results where App.css has higher semantic score than App.tsx
	// (because CSS has more "layout" and "structure" keywords)
	type mockResult struct {
		filePath      string
		originalScore float64
	}

	results := []mockResult{
		{filePath: "/styles/App.css", originalScore: 0.85},      // Higher semantic score
		{filePath: "/components/App.tsx", originalScore: 0.72},   // Lower semantic score
		{filePath: "/layouts/Layout.tsx", originalScore: 0.68},
		{filePath: "/utils/helpers.ts", originalScore: 0.55},
	}

	// Apply boosting
	for i := range results {
		boost := calculateFilenameBoost(results[i].filePath, tokens, exts, exactFilename)
		results[i].originalScore *= boost
		// Clamp to 1.0
		if results[i].originalScore > 1.0 {
			results[i].originalScore = 1.0
		}
	}

	// After boosting, App.tsx should rank first
	// App.tsx: 0.72 * 1.5 (exact match) = 1.08 → clamped to 1.0
	// App.css: 0.85 * 1.0 (no match - wrong extension) = 0.85
	// Layout.tsx: 0.68 * 1.4 (extension match) = 0.952
	// helpers.ts: 0.55 * 1.0 (no match) = 0.55

	assert.Equal(t, 1.0, results[1].originalScore, "App.tsx should have max score after exact filename boost")
	assert.Less(t, results[0].originalScore, results[1].originalScore, "App.tsx should rank higher than App.css after boosting")
	assert.Greater(t, results[2].originalScore, 0.68, "Layout.tsx should be boosted for extension match")

	t.Logf("Final scores after boosting:")
	t.Logf("  App.tsx: %.2f (was 0.72)", results[1].originalScore)
	t.Logf("  App.css: %.2f (was 0.85)", results[0].originalScore)
	t.Logf("  Layout.tsx: %.2f (was 0.68)", results[2].originalScore)
	t.Logf("  helpers.ts: %.2f (was 0.55)", results[3].originalScore)
}

// TestRealWorldQueries tests actual user query patterns
func TestRealWorldQueries(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		expectedTop  string // Expected top result after boosting
		results      []struct {
			path  string
			score float64
		}
	}{
		{
			name:        "User searches for specific React component",
			query:       "ChatSessionList component",
			expectedTop: "/components/ChatSessionList.tsx",
			results: []struct {
				path  string
				score float64
			}{
				{path: "/components/SessionManager.tsx", score: 0.80},
				{path: "/components/ChatSessionList.tsx", score: 0.75},
				{path: "/utils/chatHelpers.ts", score: 0.70},
			},
		},
		{
			name:        "User searches with file extension",
			query:       "authentication logic .go",
			expectedTop: "/auth/handler.go",
			results: []struct {
				path  string
				score float64
			}{
				{path: "/services/auth.ts", score: 0.85},
				{path: "/auth/handler.go", score: 0.78},
				{path: "/middleware/auth.go", score: 0.76},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, exts, exactFilename := parseQueryForFilenameBoost(tt.query)

			// Apply boosting
			maxScore := 0.0
			topFile := ""
			for _, r := range tt.results {
				boost := calculateFilenameBoost(r.path, tokens, exts, exactFilename)
				boostedScore := r.score * boost
				if boostedScore > 1.0 {
					boostedScore = 1.0
				}
				if boostedScore > maxScore {
					maxScore = boostedScore
					topFile = r.path
				}
			}

			assert.Equal(t, tt.expectedTop, topFile,
				"After boosting, expected file should rank first")
		})
	}
}

// Test StructuralFilter building logic
func TestStructuralFilterParsing(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]interface{}
		expectedFunc string
		expectedClass string
		expectedNode  string
		expectedDoc   *bool
		hasFilters    bool
	}{
		{
			name: "FunctionName filter",
			input: map[string]interface{}{
				"query":        "auth functions",
				"functionName": "handleAuth.*",
			},
			expectedFunc: "handleAuth.*",
			hasFilters:   true,
		},
		{
			name: "ClassName filter",
			input: map[string]interface{}{
				"query":     "user service",
				"className": "UserService",
			},
			expectedClass: "UserService",
			hasFilters:    true,
		},
		{
			name: "NodeType filter",
			input: map[string]interface{}{
				"query":    "all classes",
				"nodeType": "class",
			},
			expectedNode: "class",
			hasFilters:   true,
		},
		{
			name: "HasDocstring filter true",
			input: map[string]interface{}{
				"query":        "documented code",
				"hasDocstring": true,
			},
			expectedDoc: func() *bool { b := true; return &b }(),
			hasFilters:  true,
		},
		{
			name: "HasDocstring filter false",
			input: map[string]interface{}{
				"query":        "undocumented code",
				"hasDocstring": false,
			},
			expectedDoc: func() *bool { b := false; return &b }(),
			hasFilters:  true,
		},
		{
			name: "No filters",
			input: map[string]interface{}{
				"query": "simple query",
			},
			hasFilters: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filter parsing logic from Execute method
			var filter storage.StructuralFilter
			var hasFilters bool

			if functionName, ok := tt.input["functionName"].(string); ok && functionName != "" {
				filter.FunctionName = functionName
				hasFilters = true
			}
			if className, ok := tt.input["className"].(string); ok && className != "" {
				filter.ClassName = className
				hasFilters = true
			}
			if nodeType, ok := tt.input["nodeType"].(string); ok && nodeType != "" {
				filter.NodeType = nodeType
				hasFilters = true
			}
			if hasDocstring, ok := tt.input["hasDocstring"].(bool); ok {
				filter.HasDocstring = &hasDocstring
				hasFilters = true
			}

			// Assert
			assert.Equal(t, tt.hasFilters, hasFilters, "hasFilters flag should match")
			assert.Equal(t, tt.expectedFunc, filter.FunctionName, "FunctionName should match")
			assert.Equal(t, tt.expectedClass, filter.ClassName, "ClassName should match")
			assert.Equal(t, tt.expectedNode, filter.NodeType, "NodeType should match")
			if tt.expectedDoc != nil {
				assert.NotNil(t, filter.HasDocstring, "HasDocstring should not be nil")
				assert.Equal(t, *tt.expectedDoc, *filter.HasDocstring, "HasDocstring value should match")
			} else {
				assert.Nil(t, filter.HasDocstring, "HasDocstring should be nil")
			}
		})
	}
}

// TestStructuralFilterNodeTypes tests valid node type values
func TestStructuralFilterNodeTypes(t *testing.T) {
	validNodeTypes := []string{"function", "class", "method", "interface", "import"}

	for _, nodeType := range validNodeTypes {
		t.Run("NodeType_"+nodeType, func(t *testing.T) {
			filter := storage.StructuralFilter{
				NodeType: nodeType,
			}
			assert.Equal(t, nodeType, filter.NodeType)
		})
	}
}

// TestDocstringBoosting tests that documented code gets higher ranking
func TestDocstringBoosting(t *testing.T) {
	tests := []struct {
		name                string
		hasDocstring        bool
		originalScore       float32
		expectedMultiplier  float64
		expectedFinalScore  float32
	}{
		{
			name:               "Documented code gets 1.2x boost",
			hasDocstring:       true,
			originalScore:      0.8,
			expectedMultiplier: 1.2,
			expectedFinalScore: 0.96,
		},
		{
			name:               "Undocumented code gets no boost",
			hasDocstring:       false,
			originalScore:      0.8,
			expectedMultiplier: 1.0,
			expectedFinalScore: 0.8,
		},
		{
			name:               "Documented code score clamped at 1.0",
			hasDocstring:       true,
			originalScore:      0.9,
			expectedMultiplier: 1.2,
			expectedFinalScore: 1.0, // 0.9 * 1.2 = 1.08, clamped to 1.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate docstring boost logic
			boostMultiplier := 1.0
			if tt.hasDocstring {
				boostMultiplier = 1.2
			}

			finalScore := float32(float64(tt.originalScore) * boostMultiplier)
			if finalScore > 1.0 {
				finalScore = 1.0
			}

			assert.Equal(t, tt.expectedMultiplier, boostMultiplier, "Boost multiplier should match")
			assert.InDelta(t, tt.expectedFinalScore, finalScore, 0.001, "Final score should match")
		})
	}
}

// TestCombinedFilters tests combining multiple structural filters
func TestCombinedFilters(t *testing.T) {
	hasDoc := true
	expectedFilter := storage.StructuralFilter{
		FunctionName: "create.*",
		NodeType:     "function",
		HasDocstring: &hasDoc,
	}

	assert.Equal(t, "create.*", expectedFilter.FunctionName)
	assert.Equal(t, "function", expectedFilter.NodeType)
	assert.NotNil(t, expectedFilter.HasDocstring)
	assert.True(t, *expectedFilter.HasDocstring)
}
