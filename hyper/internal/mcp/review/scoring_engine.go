package review

import (
	"fmt"
	"strings"
	"time"

	"hyper/internal/mcp/storage"
	"go.uber.org/zap"
)

// Scores represents all calculated scores for a knowledge entry
type Scores struct {
	Alignment  float64 `json:"alignment"`
	Freshness  float64 `json:"freshness"`
	Verbosity  float64 `json:"verbosity"`
	Uniqueness float64 `json:"uniqueness"`
	Health     float64 `json:"health"`
	WordCount  int     `json:"wordCount"`
	AgeDays    int     `json:"ageDays"`
}

// ScoringEngine calculates health scores for knowledge entries
type ScoringEngine struct {
	qdrantClient storage.QdrantClientInterface
	logger       *zap.Logger
}

// NewScoringEngine creates a new scoring engine
func NewScoringEngine(qdrantClient storage.QdrantClientInterface, logger *zap.Logger) *ScoringEngine {
	return &ScoringEngine{
		qdrantClient: qdrantClient,
		logger:       logger,
	}
}

// CalculateScores is the main entry point that calculates all scores
func (s *ScoringEngine) CalculateScores(entry *storage.KnowledgeEntry, verificationResult *VerificationResult) (*Scores, error) {
	scores := &Scores{}

	// Calculate word count
	scores.WordCount = s.countWords(entry.Text)

	// Calculate age in days
	scores.AgeDays = int(time.Since(entry.CreatedAt).Hours() / 24)

	// Calculate individual scores
	scores.Alignment = s.calculateAlignment(verificationResult)
	scores.Freshness = s.calculateFreshness(scores.AgeDays)
	scores.Verbosity = s.calculateVerbosity(scores.WordCount)

	// Calculate uniqueness (requires Qdrant query)
	uniqueness, err := s.calculateUniqueness(entry)
	if err != nil {
		s.logger.Warn("Failed to calculate uniqueness score, using default",
			zap.String("entryId", entry.ID),
			zap.Error(err))
		scores.Uniqueness = 50.0 // Default to neutral score on error
	} else {
		scores.Uniqueness = uniqueness
	}

	// Calculate overall health score
	scores.Health = s.calculateHealthScore(scores)

	return scores, nil
}

// calculateAlignment calculates alignment score: (validRefs / totalRefs) × 100
func (s *ScoringEngine) calculateAlignment(verificationResult *VerificationResult) float64 {
	// Handle edge case: no references means perfect alignment
	if verificationResult.TotalReferences == 0 {
		return 100.0
	}

	return (float64(verificationResult.ValidReferences) / float64(verificationResult.TotalReferences)) * 100.0
}

// calculateFreshness calculates freshness score: max(0, 100 - (ageDays × 0.5))
func (s *ScoringEngine) calculateFreshness(ageDays int) float64 {
	score := 100.0 - (float64(ageDays) * 0.5)

	// Ensure score doesn't go below 0
	if score < 0 {
		return 0.0
	}

	return score
}

// calculateVerbosity calculates verbosity score based on word count
// Multi-tier formula:
// - < 100 words: wordCount (too short, penalize)
// - 100-200 words: 100 (acceptable)
// - 200-400 words: 100 (optimal)
// - 400-1000 words: 100 - ((wordCount - 400) × 0.05) (gradual penalty)
// - > 1000 words: max(20, 70 - ((wordCount - 1000) × 0.01)) (steep penalty)
func (s *ScoringEngine) calculateVerbosity(wordCount int) float64 {
	switch {
	case wordCount < 100:
		// Too short - score equals word count (penalizes heavily)
		return float64(wordCount)

	case wordCount >= 100 && wordCount <= 400:
		// Optimal range (100-400 words)
		return 100.0

	case wordCount > 400 && wordCount <= 1000:
		// Gradual penalty: 0.05 points per word over 400
		score := 100.0 - (float64(wordCount-400) * 0.05)
		return score

	default: // wordCount > 1000
		// Steep penalty: 0.01 points per word over 1000, starting from 70
		score := 70.0 - (float64(wordCount-1000) * 0.01)
		// Floor at 20
		if score < 20.0 {
			return 20.0
		}
		return score
	}
}

// calculateUniqueness calculates uniqueness score: (1 - avgSimilarity) × 100
// Queries Qdrant for top 5 similar entries and calculates average similarity
func (s *ScoringEngine) calculateUniqueness(entry *storage.KnowledgeEntry) (float64, error) {
	if s.qdrantClient == nil {
		return 50.0, fmt.Errorf("qdrant client not available")
	}

	// Query for top 5 similar entries (excluding self)
	// We query for 6 to account for the entry itself being in results
	results, err := s.qdrantClient.SearchSimilar(entry.Collection, entry.Text, 6)
	if err != nil {
		return 0, fmt.Errorf("failed to search similar entries: %w", err)
	}

	// Filter out the entry itself and keep only top 5
	var similarScores []float64
	for _, result := range results {
		if result.Entry.ID != entry.ID {
			similarScores = append(similarScores, result.Score)
			if len(similarScores) >= 5 {
				break
			}
		}
	}

	// If no similar entries found, it's 100% unique
	if len(similarScores) == 0 {
		return 100.0, nil
	}

	// Calculate average similarity
	totalSimilarity := 0.0
	for _, score := range similarScores {
		totalSimilarity += score
	}
	avgSimilarity := totalSimilarity / float64(len(similarScores))

	// Convert to uniqueness score: (1 - avgSimilarity) × 100
	uniqueness := (1.0 - avgSimilarity) * 100.0

	// Ensure score is between 0 and 100
	if uniqueness < 0 {
		uniqueness = 0
	}
	if uniqueness > 100 {
		uniqueness = 100
	}

	return uniqueness, nil
}

// calculateHealthScore calculates the weighted overall health score
// Formula: 0.40×Alignment + 0.25×Freshness + 0.20×Verbosity + 0.15×Uniqueness
func (s *ScoringEngine) calculateHealthScore(scores *Scores) float64 {
	health := (0.40 * scores.Alignment) +
		(0.25 * scores.Freshness) +
		(0.20 * scores.Verbosity) +
		(0.15 * scores.Uniqueness)

	return health
}

// countWords counts the number of words in a text string
func (s *ScoringEngine) countWords(text string) int {
	// Trim whitespace and split by whitespace
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}

	words := strings.Fields(trimmed)
	return len(words)
}

// Helper function for testing - allows calculating uniqueness without entry
func (s *ScoringEngine) CalculateUniquenessFromScores(similarityScores []float64) float64 {
	if len(similarityScores) == 0 {
		return 100.0
	}

	totalSimilarity := 0.0
	for _, score := range similarityScores {
		totalSimilarity += score
	}
	avgSimilarity := totalSimilarity / float64(len(similarityScores))

	uniqueness := (1.0 - avgSimilarity) * 100.0

	if uniqueness < 0 {
		uniqueness = 0
	}
	if uniqueness > 100 {
		uniqueness = 100
	}

	return uniqueness
}
