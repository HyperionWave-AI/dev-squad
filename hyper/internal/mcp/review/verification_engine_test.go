package review

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseReferences tests the reference parser
func TestParseReferences(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedCount int
		expectedTypes map[ReferenceType]int
	}{
		{
			name: "file:line references",
			text: "See handlers/mcp.go:246 for implementation. Also check internal/storage/knowledge.go:50",
			expectedCount: 2,
			expectedTypes: map[ReferenceType]int{
				ReferenceTypeFileLine: 2,
			},
		},
		{
			name: "file (lines X-Y) references",
			text: "Implementation in handlers/mcp.go (lines 50-75) handles the logic",
			expectedCount: 1,
			expectedTypes: map[ReferenceType]int{
				ReferenceTypeFileLine: 1,
			},
		},
		{
			name: "function references",
			text: "Call validateKnowledgeCollectionDimensions() to verify, then use Storage.UpsertKnowledge() method",
			expectedCount: 2,
			expectedTypes: map[ReferenceType]int{
				ReferenceTypeFunction: 2,
			},
		},
		{
			name: "commit references",
			text: "Fixed in commit 9bd11c0 and also see SHA a5c6027",
			expectedCount: 2,
			expectedTypes: map[ReferenceType]int{
				ReferenceTypeCommit: 2,
			},
		},
		{
			name: "API endpoint references",
			text: "Use POST /api/v1/knowledge/entries to store and GET /mcp/tools to list",
			expectedCount: 2,
			expectedTypes: map[ReferenceType]int{
				ReferenceTypeAPI: 2,
			},
		},
		{
			name: "mixed references",
			text: "See handlers/mcp.go:246 for handleSearch() implementation. Use POST /api/v1/search endpoint. Fixed in commit 9bd11c0",
			expectedCount: 4,
			expectedTypes: map[ReferenceType]int{
				ReferenceTypeFileLine: 1,
				ReferenceTypeFunction: 1,
				ReferenceTypeAPI:      1,
				ReferenceTypeCommit:   1,
			},
		},
		{
			name: "no references",
			text: "This is plain text without any code references",
			expectedCount: 0,
			expectedTypes: map[ReferenceType]int{},
		},
		{
			name: "deduplicate references",
			text: "See handlers/mcp.go:246 and also handlers/mcp.go:246 again",
			expectedCount: 1,
			expectedTypes: map[ReferenceType]int{
				ReferenceTypeFileLine: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := parseReferences(tt.text)
			assert.Equal(t, tt.expectedCount, len(refs), "Reference count mismatch")

			// Count by type
			typeCounts := make(map[ReferenceType]int)
			for _, ref := range refs {
				typeCounts[ref.Type]++
			}

			for refType, expectedCount := range tt.expectedTypes {
				assert.Equal(t, expectedCount, typeCounts[refType], "Type %s count mismatch", refType)
			}
		})
	}
}

// TestExtractContext tests context extraction
func TestExtractContext(t *testing.T) {
	text := "This is a very long text that contains a reference to handlers/mcp.go:246 somewhere in the middle of it"
	start := 53 // Position of "handlers/mcp.go:246"
	end := 74

	context := extractContext(text, start, end)

	assert.Contains(t, context, "handlers/mcp.go:246")
	assert.True(t, len(context) <= 174) // Max 50 chars before + ref + 50 chars after
}

// TestIsLikelyFunctionName tests function name filtering
func TestIsLikelyFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		expected bool
	}{
		{"valid function", "HandleSearch", true},
		{"valid method", "Storage.UpsertKnowledge", true},
		{"false positive If", "If", false},
		{"false positive Type", "Type", false},
		{"acronym HTTP", "HTTP", false},
		{"too short", "A", false},
		{"valid with lowercase", "CreateClient", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLikelyFunctionName(tt.funcName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidationCache tests the validation cache
func TestValidationCache(t *testing.T) {
	cache := NewValidationCache(100 * time.Millisecond)

	// Test Set and Get
	cache.Set("test-key", true)
	result, found := cache.Get("test-key")
	assert.True(t, found)
	assert.True(t, result)

	// Test non-existent key
	_, found = cache.Get("non-existent")
	assert.False(t, found)

	// Test expiration
	time.Sleep(150 * time.Millisecond)
	_, found = cache.Get("test-key")
	assert.False(t, found, "Cache entry should have expired")

	// Test Size
	cache.Set("key1", true)
	cache.Set("key2", false)
	assert.Equal(t, 2, cache.Size())

	// Test Clear
	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

// TestValidationCacheCleanExpired tests the cache cleanup
func TestValidationCacheCleanExpired(t *testing.T) {
	cache := NewValidationCache(50 * time.Millisecond)

	// Add multiple entries
	cache.Set("key1", true)
	cache.Set("key2", false)
	cache.Set("key3", true)

	assert.Equal(t, 3, cache.Size())

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Clean expired entries
	cache.CleanExpired()
	assert.Equal(t, 0, cache.Size())
}

// TestReferenceTypes tests reference type constants
func TestReferenceTypes(t *testing.T) {
	assert.Equal(t, ReferenceType("file_line"), ReferenceTypeFileLine)
	assert.Equal(t, ReferenceType("function"), ReferenceTypeFunction)
	assert.Equal(t, ReferenceType("commit"), ReferenceTypeCommit)
	assert.Equal(t, ReferenceType("api"), ReferenceTypeAPI)
	assert.Equal(t, ReferenceType("file"), ReferenceTypeFile)
}

// TestVerificationResult tests the VerificationResult struct
func TestVerificationResult(t *testing.T) {
	result := &VerificationResult{
		TotalReferences:   10,
		ValidReferences:   8,
		InvalidReferences: []Reference{
			{Type: ReferenceTypeFile, Value: "missing.go", Validated: false},
		},
		BrokenReferences: []Reference{
			{Type: ReferenceTypeCommit, Value: "abc1234", ErrorMessage: "not found"},
		},
		ValidationTime: 2 * time.Second,
	}

	assert.Equal(t, 10, result.TotalReferences)
	assert.Equal(t, 8, result.ValidReferences)
	assert.Equal(t, 1, len(result.InvalidReferences))
	assert.Equal(t, 1, len(result.BrokenReferences))
	assert.Equal(t, 2*time.Second, result.ValidationTime)
}

// TestParseFileLine tests parsing of file:line references
func TestParseFileLine(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedValue string
	}{
		{
			name:          "single line",
			text:          "See handlers/mcp.go:246 for details",
			expectedValue: "handlers/mcp.go:246",
		},
		{
			name:          "line range",
			text:          "Check handlers/mcp.go (lines 50-75)",
			expectedValue: "handlers/mcp.go:50-75",
		},
		{
			name:          "TypeScript file",
			text:          "See App.tsx:123",
			expectedValue: "App.tsx:123",
		},
		{
			name:          "Python file",
			text:          "Check main.py:45",
			expectedValue: "main.py:45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := parseReferences(tt.text)
			require.Greater(t, len(refs), 0, "Should find at least one reference")
			assert.Equal(t, ReferenceTypeFileLine, refs[0].Type)
			assert.Equal(t, tt.expectedValue, refs[0].Value)
		})
	}
}

// TestParseFunctionNames tests parsing of function references
func TestParseFunctionNames(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedValue string
	}{
		{
			name:          "simple function",
			text:          "Call HandleSearch() to execute",
			expectedValue: "HandleSearch",
		},
		{
			name:          "method call",
			text:          "Use Storage.UpsertKnowledge() method",
			expectedValue: "Storage.UpsertKnowledge",
		},
		{
			name:          "constructor",
			text:          "Create with NewClient() function",
			expectedValue: "NewClient",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := parseReferences(tt.text)
			require.Greater(t, len(refs), 0, "Should find at least one reference")

			// Find the function reference
			var found bool
			for _, ref := range refs {
				if ref.Type == ReferenceTypeFunction && ref.Value == tt.expectedValue {
					found = true
					break
				}
			}
			assert.True(t, found, "Should find function reference %s", tt.expectedValue)
		})
	}
}

// TestParseAPIEndpoints tests parsing of API endpoint references
func TestParseAPIEndpoints(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedValue string
	}{
		{
			name:          "POST endpoint",
			text:          "Use POST /api/v1/knowledge/entries to create",
			expectedValue: "POST /api/v1/knowledge/entries",
		},
		{
			name:          "GET endpoint",
			text:          "Call GET /mcp/tools to list all tools",
			expectedValue: "GET /mcp/tools",
		},
		{
			name:          "DELETE endpoint",
			text:          "Use DELETE /api/v1/entries/123 to remove",
			expectedValue: "DELETE /api/v1/entries/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := parseReferences(tt.text)
			require.Greater(t, len(refs), 0, "Should find at least one reference")

			// Find the API reference
			var found bool
			for _, ref := range refs {
				if ref.Type == ReferenceTypeAPI && ref.Value == tt.expectedValue {
					found = true
					break
				}
			}
			assert.True(t, found, "Should find API reference %s", tt.expectedValue)
		})
	}
}

// TestParseCommitHashes tests parsing of git commit references
func TestParseCommitHashes(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedValue string
		shouldFind    bool
	}{
		{
			name:          "7 char hash with commit keyword",
			text:          "Fixed in commit 9bd11c0",
			expectedValue: "9bd11c0",
			shouldFind:    true,
		},
		{
			name:          "40 char hash with SHA keyword",
			text:          "See SHA a5c60279bd11c09bd11c09bd11c09bd11c09bd11c",
			expectedValue: "a5c60279bd11c09bd11c09bd11c09bd11c09bd11c",
			shouldFind:    true,
		},
		{
			name:          "hash without context keyword",
			text:          "The value is 9bd11c0 but not a commit",
			expectedValue: "9bd11c0",
			shouldFind:    false, // No commit/SHA keyword nearby
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := parseReferences(tt.text)

			// Check if commit reference was found
			var found bool
			for _, ref := range refs {
				if ref.Type == ReferenceTypeCommit && ref.Value == tt.expectedValue {
					found = true
					break
				}
			}
			assert.Equal(t, tt.shouldFind, found, "Commit reference detection mismatch")
		})
	}
}
