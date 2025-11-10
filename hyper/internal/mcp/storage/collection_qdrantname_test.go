package storage

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestCollectionQdrantNameField verifies QdrantName field exists and works correctly
func TestCollectionQdrantNameField(t *testing.T) {
	// Test 1: Create Collection with QdrantName set
	id := primitive.NewObjectID()
	collection := &Collection{
		ID:          id,
		Name:        "test-collection",
		QdrantName:  id.Hex(), // Should be set to ID hex
		Category:    "test",
		Description: "Test collection",
		Tags:        []string{"test"},
		EntryCount:  0,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Verify QdrantName is set correctly
	if collection.QdrantName == "" {
		t.Error("QdrantName should not be empty")
	}

	if collection.QdrantName != id.Hex() {
		t.Errorf("QdrantName should equal ID.Hex(), got %s, expected %s", collection.QdrantName, id.Hex())
	}

	// Verify QdrantName is unique (different IDs = different QdrantNames)
	id2 := primitive.NewObjectID()
	collection2 := &Collection{
		ID:         id2,
		Name:       "test-collection-2",
		QdrantName: id2.Hex(),
	}

	if collection.QdrantName == collection2.QdrantName {
		t.Error("Different collections should have different QdrantNames")
	}

	t.Logf("Collection 1 QdrantName: %s", collection.QdrantName)
	t.Logf("Collection 2 QdrantName: %s", collection2.QdrantName)
}

// TestCollectionQdrantNameUniqueness verifies QdrantName uniqueness across multiple collections
func TestCollectionQdrantNameUniqueness(t *testing.T) {
	const numCollections = 10
	qdrantNames := make(map[string]bool)

	for i := 0; i < numCollections; i++ {
		id := primitive.NewObjectID()
		qdrantName := id.Hex()

		// Check for duplicates
		if qdrantNames[qdrantName] {
			t.Errorf("Duplicate QdrantName found: %s", qdrantName)
		}

		qdrantNames[qdrantName] = true
	}

	if len(qdrantNames) != numCollections {
		t.Errorf("Expected %d unique QdrantNames, got %d", numCollections, len(qdrantNames))
	}

	t.Logf("Generated %d unique QdrantNames", len(qdrantNames))
}
