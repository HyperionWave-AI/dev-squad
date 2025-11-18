package handlers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"hyper/internal/mcp/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Test extractFrontmatter
func TestExtractFrontmatter(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewMarkdownSyncService(nil, logger)

	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "Complete frontmatter",
			content: `# Title
**Collection:** test-collection
**Tags:** tag1, tag2, tag3
**Version:** 1.0
**Technology:** golang
---
Content here`,
			expected: map[string]string{
				"collection": "test-collection",
				"tags":       "tag1, tag2, tag3",
				"version":    "1.0",
				"technology": "golang",
			},
		},
		{
			name: "Partial frontmatter",
			content: `# Title
**Collection:** another-collection
**Tags:** single-tag
---
Content`,
			expected: map[string]string{
				"collection": "another-collection",
				"tags":       "single-tag",
			},
		},
		{
			name: "No frontmatter",
			content: `# Title
Just content without frontmatter`,
			expected: map[string]string{},
		},
		{
			name: "Frontmatter with extra spaces",
			content: `# Title
**Collection:**   spaced-collection
**Version:**  2.0
---`,
			expected: map[string]string{
				"collection": "spaced-collection",
				"version":    "2.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractFrontmatter(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test parseMarkdownFile
func TestParseMarkdownFile(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewMarkdownSyncService(nil, logger)

	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	testContent := `# Test Article
**Collection:** test-collection
**Tags:** golang, testing, markdown
**Version:** 1.0
---
This is the main content of the article.
It can span multiple lines.`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	// Parse the file
	parsed, err := service.parseMarkdownFile(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, parsed)

	// Verify parsed data
	assert.Equal(t, "Test Article", parsed.Title)
	assert.Equal(t, "test-collection", parsed.Collection)
	assert.Equal(t, []string{"golang", "testing", "markdown"}, parsed.Tags)
	assert.Equal(t, "1.0", parsed.Version)
	assert.Contains(t, parsed.Content, "This is the main content")
	assert.Equal(t, testFile, parsed.FilePath)
}

// Test parseMarkdownFile with missing collection
func TestParseMarkdownFile_MissingCollection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewMarkdownSyncService(nil, logger)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.md")

	testContent := `# Test Article
**Tags:** tag1, tag2
---
Content without collection`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	// Should fail due to missing collection
	_, err = service.parseMarkdownFile(testFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field: Collection")
}

// Test parseMarkdownFile with title fallback
func TestParseMarkdownFile_TitleFallback(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewMarkdownSyncService(nil, logger)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "my-article.md")

	testContent := `**Collection:** test-collection
---
Content without title heading`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	parsed, err := service.parseMarkdownFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "my-article", parsed.Title) // Should use filename without extension
}

// Test generateEntryID consistency
func TestGenerateEntryID(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewMarkdownSyncService(nil, logger)

	// Same path should always generate same ID
	path1 := "/path/to/file.md"
	id1 := service.generateEntryID(path1)
	id2 := service.generateEntryID(path1)
	assert.Equal(t, id1, id2, "Same path should generate consistent ID")

	// Different paths should generate different IDs
	path2 := "/path/to/different.md"
	id3 := service.generateEntryID(path2)
	assert.NotEqual(t, id1, id3, "Different paths should generate different IDs")

	// ID should be valid hex string (SHA256 = 64 hex chars)
	assert.Len(t, id1, 64, "SHA256 hash should be 64 hex characters")
}

// Test parseTags helper
func TestParseTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Multiple tags",
			input:    "tag1, tag2, tag3",
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "Single tag",
			input:    "single-tag",
			expected: []string{"single-tag"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Tags with extra spaces",
			input:    "  tag1  ,  tag2  ,  tag3  ",
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "Tags with empty entries",
			input:    "tag1, , tag2, ,tag3",
			expected: []string{"tag1", "tag2", "tag3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test SyncDocsKB integration
func TestSyncDocsKB_Integration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockStorage := new(CompleteKnowledgeStorageMock)
	service := NewMarkdownSyncService(mockStorage, logger)

	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create valid markdown file
	validFile := filepath.Join(tmpDir, "article1.md")
	validContent := `# Article One
**Collection:** test-collection
**Tags:** test, integration
**Version:** 1.0
---
Test content for article one`

	err := os.WriteFile(validFile, []byte(validContent), 0644)
	assert.NoError(t, err)

	// Create README.md (should be skipped)
	readmeFile := filepath.Join(tmpDir, "README.md")
	readmeContent := "# README\nThis should be skipped"
	err = os.WriteFile(readmeFile, []byte(readmeContent), 0644)
	assert.NoError(t, err)

	// Create non-markdown file (should be skipped)
	txtFile := filepath.Join(tmpDir, "notes.txt")
	err = os.WriteFile(txtFile, []byte("Not markdown"), 0644)
	assert.NoError(t, err)

	// Mock storage expectations - return "not found" error to simulate new entry
	notFoundErr := errors.New("knowledge entry not found: test-id")
	mockStorage.On("GetEntryByID", mock.Anything).Return(nil, notFoundErr)
	mockStorage.On("Upsert", "test-collection", mock.Anything, mock.Anything, (*string)(nil)).
		Return(&storage.KnowledgeEntry{ID: "entry1"}, nil)

	// Run sync
	ctx := context.Background()
	report, err := service.SyncDocsKB(ctx, tmpDir)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 1, report.FilesProcessed, "Should process 1 markdown file (excluding README.md and .txt)")
	assert.Equal(t, 1, report.EntriesCreated, "Should create 1 entry")
	assert.Equal(t, 0, report.EntriesUpdated, "Should not update any entries")
	assert.Contains(t, report.Collections, "test-collection")
	assert.Empty(t, report.Errors, "Should have no errors")

	mockStorage.AssertExpectations(t)
}

// Test SyncDocsKB with update scenario
func TestSyncDocsKB_Update(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockStorage := new(CompleteKnowledgeStorageMock)
	service := NewMarkdownSyncService(mockStorage, logger)

	// Create temporary directory with test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.md")
	testContent := `# Existing Article
**Collection:** update-test
**Version:** 2.0
---
Updated content`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	// Mock existing entry
	existingEntry := &storage.KnowledgeEntry{ID: "existing-id"}
	mockStorage.On("GetEntryByID", mock.Anything).Return(existingEntry, nil)
	mockStorage.On("UpdateEntry", "existing-id", mock.Anything, mock.Anything).
		Return(existingEntry, nil)

	// Run sync
	ctx := context.Background()
	report, err := service.SyncDocsKB(ctx, tmpDir)

	// Verify update occurred
	assert.NoError(t, err)
	assert.Equal(t, 1, report.FilesProcessed)
	assert.Equal(t, 0, report.EntriesCreated)
	assert.Equal(t, 1, report.EntriesUpdated)

	mockStorage.AssertExpectations(t)
}

// Test SyncDocsKB error handling
func TestSyncDocsKB_ErrorHandling(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockStorage := new(CompleteKnowledgeStorageMock)
	service := NewMarkdownSyncService(mockStorage, logger)

	// Create temporary directory with invalid file
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.md")
	invalidContent := `# No Collection
**Tags:** orphan
---
Missing required collection field`

	err := os.WriteFile(invalidFile, []byte(invalidContent), 0644)
	assert.NoError(t, err)

	// Run sync
	ctx := context.Background()
	report, err := service.SyncDocsKB(ctx, tmpDir)

	// Should continue despite error
	assert.NoError(t, err, "SyncDocsKB should not fail on individual file errors")
	assert.Equal(t, 1, report.FilesProcessed)
	assert.Equal(t, 0, report.EntriesCreated)
	assert.Len(t, report.Errors, 1, "Should record parsing error")
	assert.Contains(t, report.Errors[0], "missing required field: Collection")
}
