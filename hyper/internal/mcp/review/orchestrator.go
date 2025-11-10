package review

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hyper/internal/mcp/storage"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// ReviewMode defines how the review is conducted
type ReviewMode string

const (
	ReviewModeInteractive ReviewMode = "interactive"
	ReviewModeAutomatic   ReviewMode = "automatic"
)

// ReviewOrchestrator coordinates all review engines
type ReviewOrchestrator struct {
	scoringEngine     *ScoringEngine
	actionEngine      *ActionEngine
	compactionEngine  *CompactionEngine
	knowledgeStorage  storage.KnowledgeStorage
	reviewStorage     ReviewStorage
	logger            *zap.Logger
}

// NewReviewOrchestrator creates a new review orchestrator
func NewReviewOrchestrator(
	scoringEngine *ScoringEngine,
	actionEngine *ActionEngine,
	compactionEngine *CompactionEngine,
	knowledgeStorage storage.KnowledgeStorage,
	reviewStorage ReviewStorage,
	logger *zap.Logger,
) *ReviewOrchestrator {
	return &ReviewOrchestrator{
		scoringEngine:    scoringEngine,
		actionEngine:     actionEngine,
		compactionEngine: compactionEngine,
		knowledgeStorage: knowledgeStorage,
		reviewStorage:    reviewStorage,
		logger:           logger,
	}
}

// ReviewEntry performs a complete review of a knowledge entry
// Main orchestration flow: verify → score → determine actions → apply actions → store
func (o *ReviewOrchestrator) ReviewEntry(
	ctx context.Context,
	entryID string,
	mode ReviewMode,
	dryRun bool,
) (*ReviewResult, error) {
	o.logger.Info("Starting entry review",
		zap.String("entryId", entryID),
		zap.String("mode", string(mode)),
		zap.Bool("dryRun", dryRun))

	// Step 1: Fetch the entry
	entry, err := o.getEntry(entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entry: %w", err)
	}

	// Step 2: Perform verification (simple schema validation)
	verificationResult := o.verifyEntry(entry)

	// Step 3: Calculate scores
	scores, err := o.scoringEngine.CalculateScores(entry, verificationResult)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate scores: %w", err)
	}

	// Step 4: Determine actions based on scores
	entryAge := time.Since(entry.CreatedAt)
	actions := o.actionEngine.DetermineActions(
		entryID,
		entry.Collection,
		scores.Alignment,
		scores.Health,
		scores.WordCount,
		entryAge,
	)

	// Step 5: Apply actions (automatic actions + create suggestions)
	actionsTaken, suggestedActions, err := o.actionEngine.ApplyActions(
		entryID,
		entry.Collection,
		actions,
		dryRun,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to apply actions: %w", err)
	}

	// Step 6: Create review result
	result := &ReviewResult{
		ID:               primitive.NewObjectID(),
		EntryID:          entryID,
		CollectionName:   entry.Collection,
		ReviewedAt:       time.Now().UTC(),
		SchemaValid:      verificationResult.TotalReferences == verificationResult.ValidReferences,
		MinWordCount:     100,
		ActualWordCount:  scores.WordCount,
		AlignmentScore:   scores.Alignment,
		FreshnessScore:   scores.Freshness,
		VerbosityScore:   scores.Verbosity,
		UniquenessScore:  scores.Uniqueness,
		HealthScore:      scores.Health,
		TotalReferences:  verificationResult.TotalReferences,
		ValidReferences:  verificationResult.ValidReferences,
		BrokenReferences: verificationResult.BrokenReferences,
		ActionsTaken:     actionsTaken,
		SuggestedActions: suggestedActions,
		ReviewMode:       string(mode),
		DryRun:           dryRun,
	}

	// Step 7: Store review result
	if !dryRun {
		if err := o.reviewStorage.StoreReview(result); err != nil {
			return nil, fmt.Errorf("failed to store review result: %w", err)
		}
	}

	o.logger.Info("Review completed",
		zap.String("entryId", entryID),
		zap.Float64("alignmentScore", scores.Alignment),
		zap.Float64("healthScore", scores.Health),
		zap.Int("actionsTaken", len(actionsTaken)),
		zap.Int("suggestedActions", len(suggestedActions)))

	return result, nil
}

// ReviewCollection performs batch review of all entries in a collection
func (o *ReviewOrchestrator) ReviewCollection(
	ctx context.Context,
	collectionName string,
	minHealthScore float64,
	limit int,
) ([]*ReviewResult, error) {
	o.logger.Info("Starting collection review",
		zap.String("collection", collectionName),
		zap.Float64("minHealthScore", minHealthScore),
		zap.Int("limit", limit))

	// Fetch all entries in the collection
	entries, err := o.knowledgeStorage.ListKnowledge(collectionName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list knowledge entries: %w", err)
	}

	results := make([]*ReviewResult, 0, len(entries))
	reviewedCount := 0
	skippedCount := 0

	for _, entry := range entries {
		// Review the entry
		result, err := o.ReviewEntry(ctx, entry.ID, ReviewModeAutomatic, false)
		if err != nil {
			o.logger.Warn("Failed to review entry",
				zap.String("entryId", entry.ID),
				zap.Error(err))
			continue
		}

		// Filter by health score if specified
		if minHealthScore > 0 && result.HealthScore < minHealthScore {
			skippedCount++
			continue
		}

		results = append(results, result)
		reviewedCount++
	}

	o.logger.Info("Collection review completed",
		zap.String("collection", collectionName),
		zap.Int("totalEntries", len(entries)),
		zap.Int("reviewed", reviewedCount),
		zap.Int("skipped", skippedCount))

	return results, nil
}

// CompactEntry performs intelligent compaction of a knowledge entry
func (o *ReviewOrchestrator) CompactEntry(
	ctx context.Context,
	entryID string,
	targetWordCount int,
	dryRun bool,
) (*CompactionResult, error) {
	o.logger.Info("Starting entry compaction",
		zap.String("entryId", entryID),
		zap.Int("targetWords", targetWordCount),
		zap.Bool("dryRun", dryRun))

	// Check if compaction engine is available
	if o.compactionEngine == nil {
		return nil, fmt.Errorf("compaction engine not initialized (CLAUDE_API_KEY may not be set)")
	}

	// Fetch the entry
	entry, err := o.getEntry(entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entry: %w", err)
	}

	// Set target word count in compaction engine if specified
	if targetWordCount > 0 {
		o.compactionEngine.targetWords = targetWordCount
	}

	// Compact the entry
	result, err := o.compactionEngine.CompactEntry(ctx, entry.Text, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to compact entry: %w", err)
	}

	// If not dry run, update the entry
	if !dryRun && result.CompactedText != result.OriginalText {
		_, err := o.knowledgeStorage.UpdateEntry(entryID, result.CompactedText, entry.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to update entry with compacted text: %w", err)
		}

		o.logger.Info("Entry compacted and updated",
			zap.String("entryId", entryID),
			zap.Int("originalWords", result.OriginalWords),
			zap.Int("compactedWords", result.CompactedWords))
	}

	return result, nil
}

// GetCompactionEngine returns the compaction engine for direct text compaction
func (o *ReviewOrchestrator) GetCompactionEngine() *CompactionEngine {
	return o.compactionEngine
}

// AuditAllCollections performs audit across all collections
// Returns aggregated statistics
func (o *ReviewOrchestrator) AuditAllCollections(
	ctx context.Context,
	collectionNames []string,
	generateReport bool,
) (map[string]interface{}, error) {
	o.logger.Info("Starting audit of all collections",
		zap.Int("collections", len(collectionNames)),
		zap.Bool("generateReport", generateReport))

	stats := map[string]interface{}{
		"totalCollections": len(collectionNames),
		"totalEntries":     0,
		"totalReviewed":    0,
		"avgHealthScore":   0.0,
		"lowHealthCount":   0,
		"collections":      []map[string]interface{}{},
	}

	totalHealth := 0.0
	totalReviewed := 0

	for _, collectionName := range collectionNames {
		// Get all entries in collection
		entries, err := o.knowledgeStorage.ListKnowledge(collectionName, 1000)
		if err != nil {
			o.logger.Warn("Failed to list entries for collection",
				zap.String("collection", collectionName),
				zap.Error(err))
			continue
		}

		collectionStats := map[string]interface{}{
			"name":           collectionName,
			"entryCount":     len(entries),
			"reviewedCount":  0,
			"avgHealth":      0.0,
			"lowHealthCount": 0,
		}

		collectionHealth := 0.0
		reviewedCount := 0

		// Review each entry
		for _, entry := range entries {
			result, err := o.ReviewEntry(ctx, entry.ID, ReviewModeAutomatic, false)
			if err != nil {
				continue
			}

			collectionHealth += result.HealthScore
			reviewedCount++

			if result.HealthScore < 40 {
				collectionStats["lowHealthCount"] = collectionStats["lowHealthCount"].(int) + 1
			}
		}

		if reviewedCount > 0 {
			collectionStats["avgHealth"] = collectionHealth / float64(reviewedCount)
			collectionStats["reviewedCount"] = reviewedCount
		}

		stats["collections"] = append(stats["collections"].([]map[string]interface{}), collectionStats)
		stats["totalEntries"] = stats["totalEntries"].(int) + len(entries)
		stats["totalReviewed"] = stats["totalReviewed"].(int) + reviewedCount

		totalHealth += collectionHealth
		totalReviewed += reviewedCount
	}

	if totalReviewed > 0 {
		stats["avgHealthScore"] = totalHealth / float64(totalReviewed)
	}

	// Generate report if requested
	if generateReport {
		report := o.generateAuditReport(stats)
		stats["report"] = report
	}

	o.logger.Info("Audit completed",
		zap.Int("totalEntries", stats["totalEntries"].(int)),
		zap.Int("totalReviewed", stats["totalReviewed"].(int)),
		zap.Float64("avgHealthScore", stats["avgHealthScore"].(float64)))

	return stats, nil
}

// getEntry retrieves a knowledge entry by ID
func (o *ReviewOrchestrator) getEntry(entryID string) (*storage.KnowledgeEntry, error) {
	// List all collections to find the entry
	collections := o.knowledgeStorage.ListCollections()

	for _, collection := range collections {
		entries, err := o.knowledgeStorage.ListKnowledge(collection, 10000)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.ID == entryID {
				return entry, nil
			}
		}
	}

	return nil, fmt.Errorf("entry not found: %s", entryID)
}

// verifyEntry performs simple schema validation
// For now, we assume all entries are valid (no reference verification implemented yet)
func (o *ReviewOrchestrator) verifyEntry(entry *storage.KnowledgeEntry) *VerificationResult {
	// Simple validation: check if entry has text
	totalRefs := 1
	validRefs := 1

	if entry.Text == "" {
		validRefs = 0
	}

	return &VerificationResult{
		TotalReferences: totalRefs,
		ValidReferences: validRefs,
	}
}

// generateAuditReport generates a human-readable audit report
func (o *ReviewOrchestrator) generateAuditReport(stats map[string]interface{}) string {
	var report strings.Builder

	report.WriteString("=== Knowledge Base Audit Report ===\n\n")
	report.WriteString(fmt.Sprintf("Total Collections: %d\n", stats["totalCollections"]))
	report.WriteString(fmt.Sprintf("Total Entries: %d\n", stats["totalEntries"]))
	report.WriteString(fmt.Sprintf("Total Reviewed: %d\n", stats["totalReviewed"]))
	report.WriteString(fmt.Sprintf("Average Health Score: %.2f\n\n", stats["avgHealthScore"]))

	collections := stats["collections"].([]map[string]interface{})
	report.WriteString("Collection Breakdown:\n")
	for _, col := range collections {
		report.WriteString(fmt.Sprintf("  - %s: %d entries, %.2f avg health, %d low health\n",
			col["name"],
			col["entryCount"],
			col["avgHealth"],
			col["lowHealthCount"]))
	}

	return report.String()
}
