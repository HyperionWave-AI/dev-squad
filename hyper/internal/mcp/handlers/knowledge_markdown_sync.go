package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hyper/internal/mcp/storage"

	"go.uber.org/zap"
)

// MarkdownSyncService handles syncing markdown files to knowledge base
type MarkdownSyncService struct {
	knowledgeStorage storage.KnowledgeStorage
	logger           *zap.Logger
}

// NewMarkdownSyncService creates a new markdown sync service
func NewMarkdownSyncService(knowledgeStorage storage.KnowledgeStorage, logger *zap.Logger) *MarkdownSyncService {
	return &MarkdownSyncService{
		knowledgeStorage: knowledgeStorage,
		logger:           logger,
	}
}

// ParsedMarkdown represents a parsed markdown file
type ParsedMarkdown struct {
	FilePath   string
	Title      string
	Collection string
	Tags       []string
	Version    string
	Content    string
	Technology string
}

// SyncReport contains results of a sync operation
type SyncReport struct {
	FilesProcessed  int                    `json:"filesProcessed"`
	EntriesCreated  int                    `json:"entriesCreated"`
	EntriesUpdated  int                    `json:"entriesUpdated"`
	Errors          []string               `json:"errors"`
	Collections     []string               `json:"collections"`
	Details         []map[string]string    `json:"details"`
}

// SyncDocsKB scans directory and syncs all markdown files to knowledge base
func (s *MarkdownSyncService) SyncDocsKB(ctx context.Context, docsPath string) (*SyncReport, error) {
	s.logger.Info("Starting markdown sync", zap.String("path", docsPath))

	report := &SyncReport{
		Errors:      []string{},
		Collections: []string{},
		Details:     []map[string]string{},
	}

	// Track unique collections
	collectionsMap := make(map[string]bool)

	// Walk directory tree
	err := filepath.Walk(docsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Error accessing path %s: %v", path, err))
			return nil // Continue processing other files
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip README.md (catalog file)
		if filepath.Base(path) == "README.md" {
			s.logger.Debug("Skipping README.md", zap.String("path", path))
			return nil
		}

		// Only process .md files
		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Process markdown file
		report.FilesProcessed++

		parsed, err := s.parseMarkdownFile(path)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to parse %s: %v", path, err)
			report.Errors = append(report.Errors, errMsg)
			s.logger.Warn("Failed to parse markdown file",
				zap.String("path", path),
				zap.Error(err))
			return nil // Continue processing other files
		}

		// Upsert entry to knowledge base
		created, err := s.upsertEntry(ctx, parsed)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to upsert %s: %v", path, err)
			report.Errors = append(report.Errors, errMsg)
			s.logger.Warn("Failed to upsert entry",
				zap.String("path", path),
				zap.String("collection", parsed.Collection),
				zap.Error(err))
			return nil // Continue processing other files
		}

		if created {
			report.EntriesCreated++
		} else {
			report.EntriesUpdated++
		}

		// Track collection
		if !collectionsMap[parsed.Collection] {
			collectionsMap[parsed.Collection] = true
			report.Collections = append(report.Collections, parsed.Collection)
		}

		// Add details
		report.Details = append(report.Details, map[string]string{
			"file":       filepath.Base(path),
			"collection": parsed.Collection,
			"title":      parsed.Title,
			"action":     map[bool]string{true: "created", false: "updated"}[created],
		})

		s.logger.Debug("Successfully synced markdown file",
			zap.String("path", path),
			zap.String("collection", parsed.Collection),
			zap.String("title", parsed.Title),
			zap.Bool("created", created))

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	s.logger.Info("Markdown sync completed",
		zap.Int("filesProcessed", report.FilesProcessed),
		zap.Int("entriesCreated", report.EntriesCreated),
		zap.Int("entriesUpdated", report.EntriesUpdated),
		zap.Int("errors", len(report.Errors)),
		zap.Strings("collections", report.Collections))

	return report, nil
}

// parseMarkdownFile parses a markdown file and extracts metadata and content
func (s *MarkdownSyncService) parseMarkdownFile(path string) (*ParsedMarkdown, error) {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	text := string(content)

	// Extract title from first # heading
	title := ""
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			break
		}
	}

	if title == "" {
		// Fallback to filename without extension
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	// Extract frontmatter
	frontmatter := s.extractFrontmatter(text)

	// Extract content after --- separator
	parts := strings.Split(text, "---")
	mainContent := text
	if len(parts) >= 2 {
		// Content is after the --- separator
		mainContent = strings.TrimSpace(strings.Join(parts[1:], "---"))
	}

	parsed := &ParsedMarkdown{
		FilePath:   path,
		Title:      title,
		Collection: frontmatter["collection"],
		Tags:       parseTags(frontmatter["tags"]),
		Version:    frontmatter["version"],
		Technology: frontmatter["technology"],
		Content:    mainContent,
	}

	// Validate required fields
	if parsed.Collection == "" {
		return nil, fmt.Errorf("missing required field: Collection")
	}

	return parsed, nil
}

// extractFrontmatter extracts **Key:** value format from markdown
func (s *MarkdownSyncService) extractFrontmatter(content string) map[string]string {
	frontmatter := make(map[string]string)

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Stop at --- separator
		if trimmed == "---" {
			break
		}

		// Look for **Key:** value pattern
		if strings.HasPrefix(trimmed, "**") && strings.Contains(trimmed, ":**") {
			// Extract key and value
			parts := strings.SplitN(trimmed, ":**", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(strings.TrimPrefix(parts[0], "**"))
				value := strings.TrimSpace(parts[1])

				// Store as lowercase key
				frontmatter[strings.ToLower(key)] = value
			}
		}
	}

	return frontmatter
}

// parseTags splits comma-separated tags
func parseTags(tagString string) []string {
	if tagString == "" {
		return []string{}
	}

	tags := strings.Split(tagString, ",")
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// generateEntryID creates a stable ID from file path using SHA256
func (s *MarkdownSyncService) generateEntryID(filePath string) string {
	hash := sha256.Sum256([]byte(filePath))
	return hex.EncodeToString(hash[:])
}

// upsertEntry stores or updates a markdown entry in knowledge base
// Returns true if created, false if updated
func (s *MarkdownSyncService) upsertEntry(ctx context.Context, parsed *ParsedMarkdown) (bool, error) {
	// Generate stable entry ID from file path
	entryID := s.generateEntryID(parsed.FilePath)

	// Build metadata
	metadata := map[string]interface{}{
		"sourceFile": parsed.FilePath,
		"title":      parsed.Title,
		"tags":       parsed.Tags,
		"version":    parsed.Version,
		"syncedAt":   time.Now().UTC().Format(time.RFC3339),
	}

	if parsed.Technology != "" {
		metadata["technology"] = parsed.Technology
	}

	// Check if entry already exists
	existingEntry, err := s.knowledgeStorage.GetEntryByID(entryID)

	if err != nil {
		// Entry doesn't exist, create new one
		if strings.Contains(err.Error(), "not found") {

			// For new entries, we need to use the storage's Upsert which generates its own ID
			// We'll store our stable ID in metadata for idempotency tracking
			metadata["stableEntryId"] = entryID

			_, err := s.knowledgeStorage.Upsert(parsed.Collection, parsed.Content, metadata, nil)
			if err != nil {
				return false, fmt.Errorf("failed to create entry: %w", err)
			}

			s.logger.Debug("Created new knowledge entry",
				zap.String("entryId", entryID),
				zap.String("collection", parsed.Collection),
				zap.String("title", parsed.Title))

			return true, nil
		}

		// Real error
		return false, fmt.Errorf("failed to check existing entry: %w", err)
	}

	// Entry exists, update it
	metadata["stableEntryId"] = entryID

	_, err = s.knowledgeStorage.UpdateEntry(existingEntry.ID, parsed.Content, metadata)
	if err != nil {
		return false, fmt.Errorf("failed to update entry: %w", err)
	}

	s.logger.Debug("Updated existing knowledge entry",
		zap.String("entryId", entryID),
		zap.String("collection", parsed.Collection),
		zap.String("title", parsed.Title))

	return false, nil
}
