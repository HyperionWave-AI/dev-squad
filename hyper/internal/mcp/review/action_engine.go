package review

import (
	"fmt"
	"time"

	"hyper/internal/mcp/storage"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// Action represents an action to be taken on a knowledge entry
type Action struct {
	Type        string `json:"type"`        // "downvote", "flag", "suggest_delete", "suggest_compact"
	Description string `json:"description"` // Human-readable description
	Automatic   bool   `json:"automatic"`   // Whether action is auto-applied or requires approval
}

// ActionEngine determines and applies actions based on review scores
type ActionEngine struct {
	knowledgeStorage storage.KnowledgeStorage
	reviewStorage    ReviewStorage
	logger           *zap.Logger
}

// NewActionEngine creates a new action engine
func NewActionEngine(
	knowledgeStorage storage.KnowledgeStorage,
	reviewStorage ReviewStorage,
	logger *zap.Logger,
) *ActionEngine {
	return &ActionEngine{
		knowledgeStorage: knowledgeStorage,
		reviewStorage:    reviewStorage,
		logger:           logger,
	}
}

// DetermineActions analyzes scores and determines which actions to take
// Returns list of actions based on the decision rules
func (e *ActionEngine) DetermineActions(
	entryID string,
	collectionName string,
	alignmentScore float64,
	healthScore float64,
	wordCount int,
	entryAge time.Duration,
) []Action {
	actions := make([]Action, 0)

	// Rule 1: Alignment < 50 → Auto-downvote
	if alignmentScore < 50 {
		actions = append(actions, Action{
			Type:        "downvote",
			Description: fmt.Sprintf("Low alignment score: %.1f (threshold: 50)", alignmentScore),
			Automatic:   true,
		})
	}

	// Rule 2: Alignment 50-69 → Flag for review
	if alignmentScore >= 50 && alignmentScore < 70 {
		actions = append(actions, Action{
			Type:        "flag",
			Description: fmt.Sprintf("Moderate alignment score: %.1f (requires review)", alignmentScore),
			Automatic:   true,
		})
	}

	// Rule 3: Health < 40 → Suggest deletion
	if healthScore < 40 {
		actions = append(actions, Action{
			Type:        "suggest_delete",
			Description: fmt.Sprintf("Low health score: %.1f (threshold: 40)", healthScore),
			Automatic:   false,
		})
	}

	// Rule 4: WordCount > 1500 → Suggest compaction
	if wordCount > 1500 {
		actions = append(actions, Action{
			Type:        "suggest_compact",
			Description: fmt.Sprintf("High word count: %d (threshold: 1500)", wordCount),
			Automatic:   false,
		})
	}

	// Rule 5: Age > 90 days + Alignment < 70 → Suggest deletion
	if entryAge > 90*24*time.Hour && alignmentScore < 70 {
		actions = append(actions, Action{
			Type:        "suggest_delete",
			Description: fmt.Sprintf("Old entry (%.0f days) with low alignment: %.1f", entryAge.Hours()/24, alignmentScore),
			Automatic:   false,
		})
	}

	return actions
}

// ApplyActions executes the determined actions
// Returns (actionsTaken, suggestedActions, error)
func (e *ActionEngine) ApplyActions(
	entryID string,
	collectionName string,
	actions []Action,
	dryRun bool,
) ([]string, []string, error) {
	actionsTaken := make([]string, 0)
	suggestedActions := make([]string, 0)

	for _, action := range actions {
		if action.Automatic {
			// Apply automatic actions
			if !dryRun {
				if err := e.applyAutomaticAction(entryID, collectionName, action); err != nil {
					e.logger.Warn("Failed to apply automatic action",
						zap.String("entryId", entryID),
						zap.String("action", action.Type),
						zap.Error(err))
					continue
				}
			}
			actionsTaken = append(actionsTaken, fmt.Sprintf("%s: %s", action.Type, action.Description))
		} else {
			// Create suggestions for manual actions
			if !dryRun {
				if err := e.createSuggestion(entryID, collectionName, action); err != nil {
					e.logger.Warn("Failed to create suggestion",
						zap.String("entryId", entryID),
						zap.String("action", action.Type),
						zap.Error(err))
					continue
				}
			}
			suggestedActions = append(suggestedActions, fmt.Sprintf("%s: %s", action.Type, action.Description))
		}
	}

	return actionsTaken, suggestedActions, nil
}

// applyAutomaticAction executes automatic actions (downvote, flag)
func (e *ActionEngine) applyAutomaticAction(entryID string, collectionName string, action Action) error {
	switch action.Type {
	case "downvote":
		// Apply downvote using voting system
		// Use system user ID for automatic votes
		systemUserID := "system-reviewer"
		reason := "Low alignment score detected by automated review"
		_, err := e.knowledgeStorage.VoteOnEntry(entryID, systemUserID, "-", reason)
		if err != nil {
			return fmt.Errorf("failed to apply downvote: %w", err)
		}
		e.logger.Info("Applied automatic downvote",
			zap.String("entryId", entryID),
			zap.String("reason", action.Description))

	case "flag":
		// Flagging is informational - logged in review results
		e.logger.Info("Flagged entry for review",
			zap.String("entryId", entryID),
			zap.String("reason", action.Description))

	default:
		return fmt.Errorf("unknown automatic action type: %s", action.Type)
	}

	return nil
}

// createSuggestion creates a pending suggestion for manual actions
func (e *ActionEngine) createSuggestion(entryID string, collectionName string, action Action) error {
	suggestion := &ReviewSuggestion{
		ID:             primitive.NewObjectID(),
		EntryID:        entryID,
		CollectionName: collectionName,
		SuggestionType: action.Type,
		Reason:         action.Description,
		CreatedAt:      time.Now().UTC(),
		Status:         "pending",
	}

	// For compaction suggestions, we'll add original text
	// (compacted text will be generated when approved)
	if action.Type == "suggest_compact" {
		// Fetch entry to get original text
		entries, err := e.knowledgeStorage.ListKnowledge(collectionName, 1000)
		if err == nil {
			for _, entry := range entries {
				if entry.ID == entryID {
					suggestion.OriginalText = entry.Text
					suggestion.TargetWordCount = 300 // Default target
					break
				}
			}
		}
	}

	if err := e.reviewStorage.StoreSuggestion(suggestion); err != nil {
		return fmt.Errorf("failed to store suggestion: %w", err)
	}

	e.logger.Info("Created review suggestion",
		zap.String("entryId", entryID),
		zap.String("type", action.Type),
		zap.String("suggestionId", suggestion.ID.Hex()))

	return nil
}

// ApproveSuggestion approves and executes a pending suggestion
func (e *ActionEngine) ApproveSuggestion(suggestionID primitive.ObjectID) error {
	// Get suggestion
	suggestion, err := e.reviewStorage.GetSuggestion(suggestionID)
	if err != nil {
		return fmt.Errorf("failed to get suggestion: %w", err)
	}

	if suggestion.Status != "pending" {
		return fmt.Errorf("suggestion is not pending (status: %s)", suggestion.Status)
	}

	// Execute the suggested action
	switch suggestion.SuggestionType {
	case "suggest_delete":
		// Delete the knowledge entry
		if err := e.knowledgeStorage.DeleteEntry(suggestion.EntryID); err != nil {
			return fmt.Errorf("failed to delete entry: %w", err)
		}
		e.logger.Info("Deleted knowledge entry per approved suggestion",
			zap.String("entryId", suggestion.EntryID),
			zap.String("suggestionId", suggestionID.Hex()))

	case "suggest_compact":
		// For compaction, the compacted text should already be in the suggestion
		// (generated by CompactionEngine before approval)
		if suggestion.CompactedText == "" {
			return fmt.Errorf("compacted text not available for suggestion")
		}

		// Update the entry with compacted text
		_, err := e.knowledgeStorage.UpdateEntry(suggestion.EntryID, suggestion.CompactedText, nil)
		if err != nil {
			return fmt.Errorf("failed to update entry with compacted text: %w", err)
		}
		e.logger.Info("Compacted knowledge entry per approved suggestion",
			zap.String("entryId", suggestion.EntryID),
			zap.String("suggestionId", suggestionID.Hex()),
			zap.Int("originalWords", len(suggestion.OriginalText)),
			zap.Int("compactedWords", len(suggestion.CompactedText)))

	default:
		return fmt.Errorf("unknown suggestion type: %s", suggestion.SuggestionType)
	}

	// Update suggestion status
	if err := e.reviewStorage.UpdateSuggestionStatus(suggestionID, "approved"); err != nil {
		return fmt.Errorf("failed to update suggestion status: %w", err)
	}

	return nil
}

// RejectSuggestion rejects a pending suggestion
func (e *ActionEngine) RejectSuggestion(suggestionID primitive.ObjectID) error {
	// Get suggestion
	suggestion, err := e.reviewStorage.GetSuggestion(suggestionID)
	if err != nil {
		return fmt.Errorf("failed to get suggestion: %w", err)
	}

	if suggestion.Status != "pending" {
		return fmt.Errorf("suggestion is not pending (status: %s)", suggestion.Status)
	}

	// Update suggestion status
	if err := e.reviewStorage.UpdateSuggestionStatus(suggestionID, "rejected"); err != nil {
		return fmt.Errorf("failed to update suggestion status: %w", err)
	}

	e.logger.Info("Rejected suggestion",
		zap.String("entryId", suggestion.EntryID),
		zap.String("type", suggestion.SuggestionType),
		zap.String("suggestionId", suggestionID.Hex()))

	return nil
}
