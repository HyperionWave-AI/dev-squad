package review

import (
	"context"
	"testing"
	"time"

	"hyper/internal/mcp/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// MockQdrantClient for testing
type MockQdrantClient struct {
	searchResults []*storage.QdrantQueryResult
	searchError   error
}

func (m *MockQdrantClient) EnsureCollection(collectionName string, vectorSize int) error {
	return nil
}

func (m *MockQdrantClient) StorePoint(collectionName string, id string, text string, metadata map[string]interface{}) error {
	return nil
}

func (m *MockQdrantClient) SearchSimilar(collectionName string, query string, limit int, voteBoost ...float64) ([]*storage.QdrantQueryResult, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
}

func (m *MockQdrantClient) SearchWithVoteFilter(collectionName string, query string, limit int, minVoteScore int, voteBoost ...float64) ([]*storage.QdrantQueryResult, error) {
	return nil, nil
}

func (m *MockQdrantClient) DeletePoint(collectionName string, pointID string) error {
	return nil
}

func (m *MockQdrantClient) DeleteCollection(collectionName string) error {
	return nil
}

func (m *MockQdrantClient) UpdatePointPayload(collectionName string, pointID string, payload map[string]interface{}) error {
	return nil
}

func (m *MockQdrantClient) RecreateCollectionWithReindex(collectionName string, entries []*storage.KnowledgeEntry, dimensions int) (int, error) {
	return 0, nil
}

func (m *MockQdrantClient) GetDimensions() int {
	return 384
}

func (m *MockQdrantClient) SearchSimilarWithFilter(collectionName string, query string, limit int, filter map[string]interface{}, voteBoost ...float64) ([]*storage.QdrantQueryResult, error) {
	return nil, nil
}

func (m *MockQdrantClient) Ping(ctx context.Context) error {
	return nil
}

// Test Alignment Score Calculation
func TestCalculateAlignment(t *testing.T) {
	logger := zap.NewNop()
	engine := NewScoringEngine(nil, logger)

	tests := []struct {
		name     string
		result   *VerificationResult
		expected float64
	}{
		{
			name:     "Perfect alignment - all refs valid",
			result:   &VerificationResult{TotalReferences: 10, ValidReferences: 10},
			expected: 100.0,
		},
		{
			name:     "Half valid references",
			result:   &VerificationResult{TotalReferences: 10, ValidReferences: 5},
			expected: 50.0,
		},
		{
			name:     "No valid references",
			result:   &VerificationResult{TotalReferences: 10, ValidReferences: 0},
			expected: 0.0,
		},
		{
			name:     "No references at all - edge case",
			result:   &VerificationResult{TotalReferences: 0, ValidReferences: 0},
			expected: 100.0, // Perfect score when no refs
		},
		{
			name:     "75% alignment",
			result:   &VerificationResult{TotalReferences: 8, ValidReferences: 6},
			expected: 75.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateAlignment(tt.result)
			assert.Equal(t, tt.expected, score)
		})
	}
}

// Test Freshness Score Calculation
func TestCalculateFreshness(t *testing.T) {
	logger := zap.NewNop()
	engine := NewScoringEngine(nil, logger)

	tests := []struct {
		name     string
		ageDays  int
		expected float64
	}{
		{
			name:     "Brand new entry - 0 days",
			ageDays:  0,
			expected: 100.0,
		},
		{
			name:     "1 day old",
			ageDays:  1,
			expected: 99.5,
		},
		{
			name:     "10 days old",
			ageDays:  10,
			expected: 95.0,
		},
		{
			name:     "100 days old",
			ageDays:  100,
			expected: 50.0,
		},
		{
			name:     "200 days old - reaches zero",
			ageDays:  200,
			expected: 0.0,
		},
		{
			name:     "300 days old - stays at zero",
			ageDays:  300,
			expected: 0.0,
		},
		{
			name:     "Very old - 1 year",
			ageDays:  365,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateFreshness(tt.ageDays)
			assert.Equal(t, tt.expected, score)
		})
	}
}

// Test Verbosity Score Calculation
func TestCalculateVerbosity(t *testing.T) {
	logger := zap.NewNop()
	engine := NewScoringEngine(nil, logger)

	tests := []struct {
		name      string
		wordCount int
		expected  float64
	}{
		// Too short - penalized
		{
			name:      "Very short - 10 words",
			wordCount: 10,
			expected:  10.0,
		},
		{
			name:      "Still too short - 50 words",
			wordCount: 50,
			expected:  50.0,
		},
		{
			name:      "Just under acceptable - 99 words",
			wordCount: 99,
			expected:  99.0,
		},

		// Acceptable range (100-200)
		{
			name:      "Minimum acceptable - 100 words",
			wordCount: 100,
			expected:  100.0,
		},
		{
			name:      "Mid acceptable - 150 words",
			wordCount: 150,
			expected:  100.0,
		},
		{
			name:      "Top acceptable - 200 words",
			wordCount: 200,
			expected:  100.0,
		},

		// Optimal range (200-400)
		{
			name:      "Optimal start - 250 words",
			wordCount: 250,
			expected:  100.0,
		},
		{
			name:      "Optimal end - 400 words",
			wordCount: 400,
			expected:  100.0,
		},

		// Gradual penalty (400-1000)
		{
			name:      "Just over optimal - 401 words",
			wordCount: 401,
			expected:  99.95, // 100 - (1 * 0.05)
		},
		{
			name:      "Mid penalty range - 500 words",
			wordCount: 500,
			expected:  95.0, // 100 - (100 * 0.05)
		},
		{
			name:      "Near steep penalty - 1000 words",
			wordCount: 1000,
			expected:  70.0, // 100 - (600 * 0.05)
		},

		// Steep penalty (>1000)
		{
			name:      "Just over 1000 - 1001 words",
			wordCount: 1001,
			expected:  69.99, // 70 - (1 * 0.01)
		},
		{
			name:      "Very verbose - 1500 words",
			wordCount: 1500,
			expected:  65.0, // 70 - (500 * 0.01)
		},
		{
			name:      "Extremely verbose - 2000 words",
			wordCount: 2000,
			expected:  60.0, // 70 - (1000 * 0.01)
		},
		{
			name:      "Approaching floor - 5000 words",
			wordCount: 5000,
			expected:  30.0, // 70 - (4000 * 0.01) = 30
		},
		{
			name:      "Reaches floor - 6000 words",
			wordCount: 6000,
			expected:  20.0, // 70 - (5000 * 0.01) = 20 (floor)
		},
		{
			name:      "Beyond floor - 10000 words",
			wordCount: 10000,
			expected:  20.0, // Stays at floor
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateVerbosity(tt.wordCount)
			assert.InDelta(t, tt.expected, score, 0.01, "Score should match expected value")
		})
	}
}

// Test Uniqueness Score Calculation
func TestCalculateUniqueness(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name          string
		mockResults   []*storage.QdrantQueryResult
		mockError     error
		entryID       string
		expected      float64
		expectDefault bool
	}{
		{
			name: "100% unique - no similar entries",
			mockResults: []*storage.QdrantQueryResult{
				{Entry: &storage.KnowledgeEntry{ID: "test-entry"}, Score: 1.0}, // Self
			},
			entryID:  "test-entry",
			expected: 100.0,
		},
		{
			name: "90% unique - low similarity",
			mockResults: []*storage.QdrantQueryResult{
				{Entry: &storage.KnowledgeEntry{ID: "test-entry"}, Score: 1.0},   // Self
				{Entry: &storage.KnowledgeEntry{ID: "other-1"}, Score: 0.05},
				{Entry: &storage.KnowledgeEntry{ID: "other-2"}, Score: 0.08},
				{Entry: &storage.KnowledgeEntry{ID: "other-3"}, Score: 0.12},
			},
			entryID:  "test-entry",
			expected: 91.67, // (1 - (0.05+0.08+0.12)/3) * 100 = 91.67
		},
		{
			name: "50% unique - medium similarity",
			mockResults: []*storage.QdrantQueryResult{
				{Entry: &storage.KnowledgeEntry{ID: "test-entry"}, Score: 1.0}, // Self
				{Entry: &storage.KnowledgeEntry{ID: "other-1"}, Score: 0.4},
				{Entry: &storage.KnowledgeEntry{ID: "other-2"}, Score: 0.5},
				{Entry: &storage.KnowledgeEntry{ID: "other-3"}, Score: 0.6},
			},
			entryID:  "test-entry",
			expected: 50.0, // (1 - (0.4+0.5+0.6)/3) * 100 = 50
		},
		{
			name: "Not unique - high similarity",
			mockResults: []*storage.QdrantQueryResult{
				{Entry: &storage.KnowledgeEntry{ID: "test-entry"}, Score: 1.0}, // Self
				{Entry: &storage.KnowledgeEntry{ID: "other-1"}, Score: 0.9},
				{Entry: &storage.KnowledgeEntry{ID: "other-2"}, Score: 0.85},
				{Entry: &storage.KnowledgeEntry{ID: "other-3"}, Score: 0.88},
			},
			entryID:  "test-entry",
			expected: 12.33, // (1 - (0.9+0.85+0.88)/3) * 100 = 12.33
		},
		{
			name: "Top 5 only - ignores beyond 5",
			mockResults: []*storage.QdrantQueryResult{
				{Entry: &storage.KnowledgeEntry{ID: "test-entry"}, Score: 1.0}, // Self
				{Entry: &storage.KnowledgeEntry{ID: "other-1"}, Score: 0.8},
				{Entry: &storage.KnowledgeEntry{ID: "other-2"}, Score: 0.7},
				{Entry: &storage.KnowledgeEntry{ID: "other-3"}, Score: 0.6},
				{Entry: &storage.KnowledgeEntry{ID: "other-4"}, Score: 0.5},
				{Entry: &storage.KnowledgeEntry{ID: "other-5"}, Score: 0.4},
				{Entry: &storage.KnowledgeEntry{ID: "other-6"}, Score: 0.1}, // Ignored
			},
			entryID:  "test-entry",
			expected: 40.0, // (1 - (0.8+0.7+0.6+0.5+0.4)/5) * 100 = 40
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockQdrantClient{
				searchResults: tt.mockResults,
				searchError:   tt.mockError,
			}
			engine := NewScoringEngine(mockClient, logger)

			entry := &storage.KnowledgeEntry{
				ID:         tt.entryID,
				Collection: "test-collection",
				Text:       "Test text",
				CreatedAt:  time.Now(),
			}

			score, err := engine.calculateUniqueness(entry)

			if tt.expectDefault {
				assert.Error(t, err)
				assert.Equal(t, 0.0, score)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expected, score, 0.01, "Score should match expected value")
			}
		})
	}
}

// Test Health Score Calculation
func TestCalculateHealthScore(t *testing.T) {
	logger := zap.NewNop()
	engine := NewScoringEngine(nil, logger)

	tests := []struct {
		name     string
		scores   *Scores
		expected float64
	}{
		{
			name: "Perfect scores",
			scores: &Scores{
				Alignment:  100.0,
				Freshness:  100.0,
				Verbosity:  100.0,
				Uniqueness: 100.0,
			},
			expected: 100.0, // 0.40*100 + 0.25*100 + 0.20*100 + 0.15*100
		},
		{
			name: "All zeros",
			scores: &Scores{
				Alignment:  0.0,
				Freshness:  0.0,
				Verbosity:  0.0,
				Uniqueness: 0.0,
			},
			expected: 0.0,
		},
		{
			name: "Mixed scores - example from architecture",
			scores: &Scores{
				Alignment:  85.0,
				Freshness:  92.0,
				Verbosity:  100.0,
				Uniqueness: 75.0,
			},
			expected: 88.25, // 0.40*85 + 0.25*92 + 0.20*100 + 0.15*75 = 34 + 23 + 20 + 11.25
		},
		{
			name: "Poor alignment dominates (40% weight)",
			scores: &Scores{
				Alignment:  20.0, // Poor
				Freshness:  100.0,
				Verbosity:  100.0,
				Uniqueness: 100.0,
			},
			expected: 68.0, // 0.40*20 + 0.25*100 + 0.20*100 + 0.15*100 = 8 + 25 + 20 + 15 = 68
		},
		{
			name: "Realistic degraded entry",
			scores: &Scores{
				Alignment:  60.0,  // Some broken refs
				Freshness:  30.0,  // ~140 days old
				Verbosity:  95.0,  // Slightly verbose
				Uniqueness: 40.0,  // Some duplication
			},
			expected: 56.5, // 0.40*60 + 0.25*30 + 0.20*95 + 0.15*40 = 24 + 7.5 + 19 + 6 = 56.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateHealthScore(tt.scores)
			assert.InDelta(t, tt.expected, score, 0.01, "Health score should match expected value")
		})
	}
}

// Test Complete Score Calculation (Integration Test)
func TestCalculateScores(t *testing.T) {
	logger := zap.NewNop()

	// Mock Qdrant client with medium similarity
	mockClient := &MockQdrantClient{
		searchResults: []*storage.QdrantQueryResult{
			{Entry: &storage.KnowledgeEntry{ID: "test-entry"}, Score: 1.0}, // Self
			{Entry: &storage.KnowledgeEntry{ID: "other-1"}, Score: 0.3},
			{Entry: &storage.KnowledgeEntry{ID: "other-2"}, Score: 0.4},
		},
	}

	engine := NewScoringEngine(mockClient, logger)

	// Create test entry - 30 days old, ~250 words
	text := "This is a test knowledge entry. " +
		"It contains enough words to reach the optimal verbosity range. " +
		"We need to fill it with meaningful content that demonstrates " +
		"a proper knowledge base entry with technical details and context. " +
		"The entry should be comprehensive yet concise, providing value " +
		"without being overly verbose. This balance is crucial for maintaining " +
		"high verbosity scores in our scoring system. " +
		"Additional content helps us reach the target word count of around 250 words " +
		"which falls in the optimal range. More details about implementation, " +
		"architecture decisions, code patterns, and best practices should be included. " +
		"The system evaluates entries based on multiple dimensions including alignment, " +
		"freshness, verbosity, and uniqueness to provide a comprehensive health score. " +
		"This multi-dimensional approach ensures that knowledge entries maintain high quality " +
		"over time and remain valuable resources for the development team. " +
		"References to code, documentation, and other resources help validate the accuracy " +
		"and relevance of the information provided in each entry."

	createdAt := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
	entry := &storage.KnowledgeEntry{
		ID:         "test-entry",
		Collection: "test-collection",
		Text:       text,
		CreatedAt:  createdAt,
	}

	verificationResult := &VerificationResult{
		TotalReferences: 10,
		ValidReferences: 9, // 90% alignment
	}

	scores, err := engine.CalculateScores(entry, verificationResult)

	assert.NoError(t, err)
	assert.NotNil(t, scores)

	// Verify individual scores
	assert.Equal(t, 90.0, scores.Alignment, "Should have 90% alignment")
	assert.Equal(t, 85.0, scores.Freshness, "30 days old = 100 - 30*0.5 = 85")
	assert.Equal(t, 100.0, scores.Verbosity, "~150 words in optimal range")
	assert.InDelta(t, 65.0, scores.Uniqueness, 1.0, "Uniqueness from mock similarity")

	// Verify word count and age
	assert.Greater(t, scores.WordCount, 100, "Should have counted words")
	assert.Equal(t, 30, scores.AgeDays, "Should be 30 days old")

	// Verify health score is calculated
	expectedHealth := 0.40*scores.Alignment + 0.25*scores.Freshness + 0.20*scores.Verbosity + 0.15*scores.Uniqueness
	assert.InDelta(t, expectedHealth, scores.Health, 0.01, "Health score should match formula")
}

// Test Edge Cases
func TestEdgeCases(t *testing.T) {
	logger := zap.NewNop()

	t.Run("Nil Qdrant client - uses default uniqueness", func(t *testing.T) {
		engine := NewScoringEngine(nil, logger)

		entry := &storage.KnowledgeEntry{
			ID:         "test",
			Collection: "test",
			Text:       "Test entry with optimal word count for verbosity scoring system to work correctly",
			CreatedAt:  time.Now(),
		}

		verificationResult := &VerificationResult{
			TotalReferences: 5,
			ValidReferences: 5,
		}

		scores, err := engine.CalculateScores(entry, verificationResult)

		assert.NoError(t, err)
		assert.Equal(t, 50.0, scores.Uniqueness, "Should use default 50.0 when Qdrant unavailable")
	})

	t.Run("Very old entry", func(t *testing.T) {
		engine := NewScoringEngine(nil, logger)

		entry := &storage.KnowledgeEntry{
			ID:         "test",
			Collection: "test",
			Text:       "Old entry",
			CreatedAt:  time.Now().Add(-365 * 24 * time.Hour), // 1 year old
		}

		verificationResult := &VerificationResult{
			TotalReferences: 1,
			ValidReferences: 1,
		}

		scores, err := engine.CalculateScores(entry, verificationResult)

		assert.NoError(t, err)
		assert.Equal(t, 0.0, scores.Freshness, "Should be 0 for very old entries")
		assert.Equal(t, 365, scores.AgeDays)
	})

	t.Run("Empty text entry", func(t *testing.T) {
		engine := NewScoringEngine(nil, logger)

		entry := &storage.KnowledgeEntry{
			ID:         "test",
			Collection: "test",
			Text:       "",
			CreatedAt:  time.Now(),
		}

		verificationResult := &VerificationResult{
			TotalReferences: 0,
			ValidReferences: 0,
		}

		scores, err := engine.CalculateScores(entry, verificationResult)

		assert.NoError(t, err)
		assert.Equal(t, 0, scores.WordCount)
		assert.Equal(t, 0.0, scores.Verbosity, "Empty text = 0 words = 0 score")
	})
}
