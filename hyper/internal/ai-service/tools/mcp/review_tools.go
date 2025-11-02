package mcp

import (
	"context"
	"fmt"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/mcp/review"
)

// KnowledgeReviewTool implements single entry review
type KnowledgeReviewTool struct {
	orchestrator *review.ReviewOrchestrator
}

func (t *KnowledgeReviewTool) Name() string {
	return "knowledge_review"
}

func (t *KnowledgeReviewTool) Description() string {
	return "Review a single knowledge entry for quality, alignment, and health. Returns scores, actions taken, and suggestions. Use mode='interactive' for manual review or 'automatic' for batch processing."
}

func (t *KnowledgeReviewTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"entryId": map[string]interface{}{
				"type":        "string",
				"description": "ID of the knowledge entry to review",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "Review mode: 'interactive' (default) or 'automatic'",
				"enum":        []string{"interactive", "automatic"},
			},
			"dryRun": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, performs review without applying actions (default: false)",
			},
		},
		"required": []string{"entryId"},
	}
}

func (t *KnowledgeReviewTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract entryId
	entryID, ok := input["entryId"].(string)
	if !ok || entryID == "" {
		return nil, fmt.Errorf("entryId is required and must be a string")
	}

	// Extract mode (default: interactive)
	mode := "interactive"
	if m, ok := input["mode"].(string); ok && m != "" {
		mode = m
	}

	// Extract dryRun (default: false)
	dryRun := false
	if dr, ok := input["dryRun"].(bool); ok {
		dryRun = dr
	}

	// Convert mode string to ReviewMode
	reviewMode := review.ReviewModeInteractive
	if mode == "automatic" {
		reviewMode = review.ReviewModeAutomatic
	}

	// Perform review
	result, err := t.orchestrator.ReviewEntry(ctx, entryID, reviewMode, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to review entry: %w", err)
	}

	// Format response
	return map[string]interface{}{
		"entryId":         result.EntryID,
		"collectionName":  result.CollectionName,
		"reviewedAt":      result.ReviewedAt,
		"schemaValid":     result.SchemaValid,
		"wordCount":       result.ActualWordCount,
		"alignmentScore":  result.AlignmentScore,
		"healthScore":     result.HealthScore,
		"actionsTaken":    result.ActionsTaken,
		"suggestedActions": result.SuggestedActions,
		"reviewMode":      result.ReviewMode,
		"dryRun":          result.DryRun,
	}, nil
}

// KnowledgeReviewCollectionTool implements batch collection review
type KnowledgeReviewCollectionTool struct {
	orchestrator *review.ReviewOrchestrator
}

func (t *KnowledgeReviewCollectionTool) Name() string {
	return "knowledge_review_collection"
}

func (t *KnowledgeReviewCollectionTool) Description() string {
	return "Review all entries in a collection for quality assessment. Returns review results for entries below the health threshold. Use for bulk quality audits."
}

func (t *KnowledgeReviewCollectionTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{
				"type":        "string",
				"description": "Name of the collection to review",
			},
			"minHealthScore": map[string]interface{}{
				"type":        "number",
				"description": "Minimum health score threshold (default: 100, show all)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of entries to review (default: 50)",
			},
		},
		"required": []string{"collection"},
	}
}

func (t *KnowledgeReviewCollectionTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract collection
	collection, ok := input["collection"].(string)
	if !ok || collection == "" {
		return nil, fmt.Errorf("collection is required and must be a string")
	}

	// Extract minHealthScore (default: 100)
	minHealthScore := 100.0
	if mhs, ok := input["minHealthScore"].(float64); ok {
		minHealthScore = mhs
	}

	// Extract limit (default: 50)
	limit := 50
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	// Perform batch review
	results, err := t.orchestrator.ReviewCollection(ctx, collection, minHealthScore, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to review collection: %w", err)
	}

	// Format response
	formattedResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		formattedResults[i] = map[string]interface{}{
			"entryId":         result.EntryID,
			"collectionName":  result.CollectionName,
			"reviewedAt":      result.ReviewedAt,
			"wordCount":       result.ActualWordCount,
			"alignmentScore":  result.AlignmentScore,
			"healthScore":     result.HealthScore,
			"actionsTaken":    result.ActionsTaken,
			"suggestedActions": result.SuggestedActions,
		}
	}

	return map[string]interface{}{
		"collection":    collection,
		"reviewedCount": len(results),
		"results":       formattedResults,
	}, nil
}

// KnowledgeCompactTool implements entry compaction
type KnowledgeCompactTool struct {
	orchestrator *review.ReviewOrchestrator
}

func (t *KnowledgeCompactTool) Name() string {
	return "knowledge_compact"
}

func (t *KnowledgeCompactTool) Description() string {
	return "Compact a knowledge entry to reduce verbosity while preserving critical information. Uses LLM to intelligently summarize. Returns original and compacted text with word counts."
}

func (t *KnowledgeCompactTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"entryId": map[string]interface{}{
				"type":        "string",
				"description": "ID of the knowledge entry to compact",
			},
			"targetWordCount": map[string]interface{}{
				"type":        "integer",
				"description": "Target word count for compacted text (default: 300)",
			},
			"dryRun": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, shows what would be compacted without applying changes (default: true)",
			},
		},
		"required": []string{"entryId"},
	}
}

func (t *KnowledgeCompactTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract entryId
	entryID, ok := input["entryId"].(string)
	if !ok || entryID == "" {
		return nil, fmt.Errorf("entryId is required and must be a string")
	}

	// Extract targetWordCount (default: 300)
	targetWordCount := 300
	if twc, ok := input["targetWordCount"].(float64); ok {
		targetWordCount = int(twc)
	}

	// Extract dryRun (default: true for safety)
	dryRun := true
	if dr, ok := input["dryRun"].(bool); ok {
		dryRun = dr
	}

	// Perform compaction
	result, err := t.orchestrator.CompactEntry(ctx, entryID, targetWordCount, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to compact entry: %w", err)
	}

	// Format response
	return map[string]interface{}{
		"entryId":          entryID,
		"originalWords":    result.OriginalWords,
		"compactedWords":   result.CompactedWords,
		"preservedAll":     result.PreservedAll,
		"missingElements":  result.MissingElements,
		"dryRun":           result.DryRun,
		"originalText":     result.OriginalText,
		"compactedText":    result.CompactedText,
	}, nil
}

// KnowledgeAuditAllTool implements full knowledge base audit
type KnowledgeAuditAllTool struct {
	orchestrator *review.ReviewOrchestrator
}

func (t *KnowledgeAuditAllTool) Name() string {
	return "knowledge_audit_all"
}

func (t *KnowledgeAuditAllTool) Description() string {
	return "Audit all knowledge collections for quality metrics. Returns aggregated statistics including average health scores, low health counts, and collection breakdowns. Optionally generates a human-readable report."
}

func (t *KnowledgeAuditAllTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collections": map[string]interface{}{
				"type":        "array",
				"description": "List of collection names to audit (empty = all collections)",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"generateReport": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, generates a human-readable report (default: true)",
			},
		},
	}
}

func (t *KnowledgeAuditAllTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract collections (default: empty array means all)
	var collections []string
	if colsRaw, ok := input["collections"]; ok && colsRaw != nil {
		// Parse collections array
		if colsArray, ok := colsRaw.([]interface{}); ok {
			for _, item := range colsArray {
				if str, ok := item.(string); ok {
					collections = append(collections, str)
				}
			}
		}
	}

	// Extract generateReport (default: true)
	generateReport := true
	if gr, ok := input["generateReport"].(bool); ok {
		generateReport = gr
	}

	// If collections is empty, get all collections
	if len(collections) == 0 {
		// This would need access to knowledgeStorage, but we can return error for now
		return nil, fmt.Errorf("collections parameter is required (specify collections to audit)")
	}

	// Perform audit
	stats, err := t.orchestrator.AuditAllCollections(ctx, collections, generateReport)
	if err != nil {
		return nil, fmt.Errorf("failed to audit collections: %w", err)
	}

	return stats, nil
}

// RegisterReviewTools registers all review-related MCP tools
func RegisterReviewTools(
	registry *aiservice.ToolRegistry,
	orchestrator *review.ReviewOrchestrator,
) error {
	tools := []aiservice.ToolExecutor{
		&KnowledgeReviewTool{orchestrator: orchestrator},
		&KnowledgeReviewCollectionTool{orchestrator: orchestrator},
		&KnowledgeCompactTool{orchestrator: orchestrator},
		&KnowledgeAuditAllTool{orchestrator: orchestrator},
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
