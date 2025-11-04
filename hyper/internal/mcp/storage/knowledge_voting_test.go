package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// TestVoteWeightedRetrieval tests end-to-end vote-weighted search functionality
func TestVoteWeightedRetrieval(t *testing.T) {
	// Skip if MongoDB or Qdrant not available
	mongoURI := os.Getenv("MONGODB_URI")
	qdrantURL := os.Getenv("QDRANT_URL")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	// Connect to MongoDB
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	// Test database
	db := client.Database("hyper_test_voting")
	defer db.Drop(ctx)

	// Create Qdrant client with test embedding
	qdrantClient := NewQdrantClientWithEmbedding(qdrantURL, func(text string) ([]float64, error) {
		// Simple deterministic embedding for testing
		return generateSimpleEmbedding(text, 768), nil
	}, 768)

	// Ping Qdrant
	if err := qdrantClient.Ping(ctx); err != nil {
		t.Skipf("Qdrant not available: %v", err)
	}

	// Create knowledge storage
	logger := zap.NewNop()
	storage, err := NewMongoKnowledgeStorage(db, qdrantClient, logger)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	testCollection := "test_voting_collection"

	// Step 1: Create 3 knowledge entries
	entries := []struct {
		text        string
		upvotes     int
		downvotes   int
		expectedNet int
	}{
		{"Machine learning fundamentals and neural networks", 5, 0, 5},
		{"Basic programming concepts", 0, 0, 0},
		{"Outdated deprecated API documentation", 0, 3, -3},
	}

	entryIDs := make([]string, len(entries))
	for i, e := range entries {
		entry, err := storage.Upsert(testCollection, e.text, map[string]interface{}{
			"topic": fmt.Sprintf("topic_%d", i),
		}, nil)
		if err != nil {
			t.Fatalf("Failed to create entry %d: %v", i, err)
		}
		entryIDs[i] = entry.ID

		// Add votes
		for j := 0; j < e.upvotes; j++ {
			userID := fmt.Sprintf("user_upvote_%d_%d", i, j)
			_, err := storage.VoteOnEntry(entry.ID, userID, "+", "helpful")
			if err != nil {
				t.Fatalf("Failed to add upvote for entry %d: %v", i, err)
			}
		}

		for j := 0; j < e.downvotes; j++ {
			userID := fmt.Sprintf("user_downvote_%d_%d", i, j)
			_, err := storage.VoteOnEntry(entry.ID, userID, "-", "not helpful")
			if err != nil {
				t.Fatalf("Failed to add downvote for entry %d: %v", i, err)
			}
		}
	}

	// Wait for Qdrant indexing
	time.Sleep(500 * time.Millisecond)

	// Step 2: Verify votes are synced to Qdrant
	for i, entryID := range entryIDs {
		votes, err := storage.GetEntryVotes(entryID, "")
		if err != nil {
			t.Fatalf("Failed to get votes for entry %d: %v", i, err)
		}

		if votes.Upvotes != entries[i].upvotes {
			t.Errorf("Entry %d: expected %d upvotes, got %d", i, entries[i].upvotes, votes.Upvotes)
		}
		if votes.Downvotes != entries[i].downvotes {
			t.Errorf("Entry %d: expected %d downvotes, got %d", i, entries[i].downvotes, votes.Downvotes)
		}
		if votes.NetScore != entries[i].expectedNet {
			t.Errorf("Entry %d: expected net score %d, got %d", i, entries[i].expectedNet, votes.NetScore)
		}
	}

	// Step 3: Test query without vote boosting (baseline)
	t.Run("Query_NoVoteBoost", func(t *testing.T) {
		results, err := storage.Query(testCollection, "machine learning", 3, nil)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("Expected at least one result")
		}

		// First result should be the ML entry (semantic match)
		if results[0].Entry.ID != entryIDs[0] {
			t.Logf("Without vote boost, results order: %v", getResultIDs(results))
		}
	})

	// Step 4: Test query with vote boosting
	t.Run("Query_WithVoteBoost", func(t *testing.T) {
		// Use voteBoost=0.5 to significantly affect ranking
		results, err := storage.Query(testCollection, "machine learning", 3, nil, 0.5)
		if err != nil {
			t.Fatalf("Query with vote boost failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("Expected at least one result")
		}

		// First result should still be the ML entry (high semantic match + high votes)
		if results[0].Entry.ID != entryIDs[0] {
			t.Errorf("With vote boost, expected first result to be entry 0 (ML with +5 votes), got %s", results[0].Entry.ID)
		}

		// Entry with negative votes should rank lower
		hasNegativeEntry := false
		negativeEntryPos := -1
		for i, r := range results {
			if r.Entry.ID == entryIDs[2] {
				hasNegativeEntry = true
				negativeEntryPos = i
				break
			}
		}

		if hasNegativeEntry && negativeEntryPos == 0 {
			t.Error("Entry with -3 votes should not rank first with vote boosting enabled")
		}

		t.Logf("Vote-boosted results order: %v (scores: %v)", getResultIDs(results), getResultScores(results))
	})

	// Step 5: Test SearchWithVoteFilter
	t.Run("SearchWithVoteFilter", func(t *testing.T) {
		// Filter out entries with voteScore < 0 (downvoted content)
		results, err := qdrantClient.SearchWithVoteFilter(testCollection, "API documentation", 3, 0)
		if err != nil {
			t.Fatalf("SearchWithVoteFilter failed: %v", err)
		}

		// Should not include the downvoted entry (entry 2 with -3 votes)
		for _, r := range results {
			if r.Entry.ID == entryIDs[2] {
				t.Error("SearchWithVoteFilter should exclude entry with voteScore < 0")
			}
		}

		t.Logf("Filtered results (minVoteScore=0): %v", getQdrantResultIDs(results))
	})

	// Step 6: Test batch sync
	t.Run("BatchSyncVotes", func(t *testing.T) {
		count, err := storage.BatchSyncVotesToQdrant("")
		if err != nil {
			t.Fatalf("BatchSyncVotesToQdrant failed: %v", err)
		}

		// Should sync all 2 entries with votes (entry 0 has votes, entry 2 has votes, entry 1 has no votes)
		expectedCount := 2
		if count != expectedCount {
			t.Errorf("Expected to sync %d entries, got %d", expectedCount, count)
		}

		t.Logf("Batch synced %d entries", count)
	})

	// Step 7: Test batch sync with collection filter
	t.Run("BatchSyncVotes_WithFilter", func(t *testing.T) {
		count, err := storage.BatchSyncVotesToQdrant(testCollection)
		if err != nil {
			t.Fatalf("BatchSyncVotesToQdrant with filter failed: %v", err)
		}

		expectedCount := 2
		if count != expectedCount {
			t.Errorf("Expected to sync %d entries for collection %s, got %d", expectedCount, testCollection, count)
		}
	})

	// Step 8: Verify vote score normalization
	t.Run("VoteScoreNormalization", func(t *testing.T) {
		// Query with high vote boost
		results, err := storage.Query(testCollection, "programming", 3, nil, 1.0)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// All scores should be positive (normalization prevents negative scores)
		for _, r := range results {
			if r.Score < 0 {
				t.Errorf("Score should be >= 0 after normalization, got %f", r.Score)
			}
		}

		t.Logf("Normalized scores: %v", getResultScores(results))
	})
}

// Helper functions
func getResultIDs(results []*QueryResult) []string {
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.Entry.ID
	}
	return ids
}

func getResultScores(results []*QueryResult) []float64 {
	scores := make([]float64, len(results))
	for i, r := range results {
		scores[i] = r.Score
	}
	return scores
}

func getQdrantResultIDs(results []*QdrantQueryResult) []string {
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.Entry.ID
	}
	return ids
}
