package review

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// ReviewResult represents the complete review result for a knowledge entry
type ReviewResult struct {
	ID                  primitive.ObjectID `json:"id" bson:"_id"`
	EntryID             string             `json:"entryId" bson:"entryId"`
	CollectionName      string             `json:"collectionName" bson:"collectionName"`
	ReviewedAt          time.Time          `json:"reviewedAt" bson:"reviewedAt"`

	// Verification scores
	SchemaValid         bool               `json:"schemaValid" bson:"schemaValid"`
	MinWordCount        int                `json:"minWordCount" bson:"minWordCount"`
	ActualWordCount     int                `json:"actualWordCount" bson:"actualWordCount"`

	// Quality scores
	AlignmentScore      float64            `json:"alignmentScore" bson:"alignmentScore"`
	FreshnessScore      float64            `json:"freshnessScore" bson:"freshnessScore"`
	VerbosityScore      float64            `json:"verbosityScore" bson:"verbosityScore"`
	UniquenessScore     float64            `json:"uniquenessScore" bson:"uniquenessScore"`
	HealthScore         float64            `json:"healthScore" bson:"healthScore"`

	// Verification details
	TotalReferences     int                `json:"totalReferences" bson:"totalReferences"`
	ValidReferences     int                `json:"validReferences" bson:"validReferences"`
	BrokenReferences    []Reference        `json:"brokenReferences" bson:"brokenReferences"`

	// Actions taken
	ActionsTaken        []string           `json:"actionsTaken" bson:"actionsTaken"`
	SuggestedActions    []string           `json:"suggestedActions" bson:"suggestedActions"`

	// Review metadata
	ReviewMode          string             `json:"reviewMode" bson:"reviewMode"` // "interactive" or "automatic"
	DryRun              bool               `json:"dryRun" bson:"dryRun"`
}

// ReviewSuggestion represents a pending action that requires approval
type ReviewSuggestion struct {
	ID                  primitive.ObjectID `json:"id" bson:"_id"`
	EntryID             string             `json:"entryId" bson:"entryId"`
	CollectionName      string             `json:"collectionName" bson:"collectionName"`
	SuggestionType      string             `json:"suggestionType" bson:"suggestionType"` // "delete", "compact"
	Reason              string             `json:"reason" bson:"reason"`
	CreatedAt           time.Time          `json:"createdAt" bson:"createdAt"`
	ApprovedAt          *time.Time         `json:"approvedAt,omitempty" bson:"approvedAt,omitempty"`
	RejectedAt          *time.Time         `json:"rejectedAt,omitempty" bson:"rejectedAt,omitempty"`
	Status              string             `json:"status" bson:"status"` // "pending", "approved", "rejected"

	// For compaction suggestions
	OriginalText        string             `json:"originalText,omitempty" bson:"originalText,omitempty"`
	CompactedText       string             `json:"compactedText,omitempty" bson:"compactedText,omitempty"`
	TargetWordCount     int                `json:"targetWordCount,omitempty" bson:"targetWordCount,omitempty"`
}

// ReviewStorage provides storage interface for review results and suggestions
type ReviewStorage interface {
	StoreReview(result *ReviewResult) error
	GetReview(entryID string) (*ReviewResult, error)
	ListReviews(collectionName string, limit int) ([]*ReviewResult, error)

	StoreSuggestion(suggestion *ReviewSuggestion) error
	GetSuggestion(suggestionID primitive.ObjectID) (*ReviewSuggestion, error)
	ListPendingSuggestions(collectionName string, limit int) ([]*ReviewSuggestion, error)
	UpdateSuggestionStatus(suggestionID primitive.ObjectID, status string) error
}

// MongoReviewStorage implements ReviewStorage using MongoDB
type MongoReviewStorage struct {
	reviewsCollection     *mongo.Collection
	suggestionsCollection *mongo.Collection
	logger                *zap.Logger
}

// NewMongoReviewStorage creates a new MongoDB review storage
func NewMongoReviewStorage(db *mongo.Database, logger *zap.Logger) (*MongoReviewStorage, error) {
	storage := &MongoReviewStorage{
		reviewsCollection:     db.Collection("knowledge_reviews"),
		suggestionsCollection: db.Collection("review_suggestions"),
		logger:                logger,
	}

	// Create indexes
	ctx := context.Background()

	// Index on entryId for reviews (unique to store latest review per entry)
	_, err := storage.reviewsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "entryId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create entryId index: %w", err)
	}

	// Index on collectionName for reviews
	_, err = storage.reviewsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "collectionName", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create collectionName index: %w", err)
	}

	// Index on reviewedAt for sorting
	_, err = storage.reviewsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "reviewedAt", Value: -1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create reviewedAt index: %w", err)
	}

	// Index on entryId for suggestions
	_, err = storage.suggestionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "entryId", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create suggestions entryId index: %w", err)
	}

	// Index on status for suggestions
	_, err = storage.suggestionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create suggestions status index: %w", err)
	}

	return storage, nil
}

// StoreReview stores a review result (upserts by entryId to keep latest review)
func (s *MongoReviewStorage) StoreReview(result *ReviewResult) error {
	ctx := context.Background()

	// Set ID if not already set
	if result.ID.IsZero() {
		result.ID = primitive.NewObjectID()
	}

	// Upsert by entryId to keep only the latest review per entry
	filter := bson.M{"entryId": result.EntryID}
	update := bson.M{"$set": result}
	opts := options.Update().SetUpsert(true)

	_, err := s.reviewsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to store review: %w", err)
	}

	return nil
}

// GetReview retrieves the latest review for an entry
func (s *MongoReviewStorage) GetReview(entryID string) (*ReviewResult, error) {
	ctx := context.Background()

	var review ReviewResult
	err := s.reviewsCollection.FindOne(ctx, bson.M{"entryId": entryID}).Decode(&review)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("review not found for entry: %s", entryID)
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	return &review, nil
}

// ListReviews retrieves recent reviews for a collection
func (s *MongoReviewStorage) ListReviews(collectionName string, limit int) ([]*ReviewResult, error) {
	ctx := context.Background()

	filter := bson.M{}
	if collectionName != "" {
		filter["collectionName"] = collectionName
	}

	opts := options.Find().SetSort(bson.D{{Key: "reviewedAt", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := s.reviewsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list reviews: %w", err)
	}
	defer cursor.Close(ctx)

	var reviews []*ReviewResult
	if err := cursor.All(ctx, &reviews); err != nil {
		return nil, fmt.Errorf("failed to decode reviews: %w", err)
	}

	if reviews == nil {
		reviews = make([]*ReviewResult, 0)
	}

	return reviews, nil
}

// StoreSuggestion stores a review suggestion
func (s *MongoReviewStorage) StoreSuggestion(suggestion *ReviewSuggestion) error {
	ctx := context.Background()

	// Set ID if not already set
	if suggestion.ID.IsZero() {
		suggestion.ID = primitive.NewObjectID()
	}

	_, err := s.suggestionsCollection.InsertOne(ctx, suggestion)
	if err != nil {
		return fmt.Errorf("failed to store suggestion: %w", err)
	}

	return nil
}

// GetSuggestion retrieves a suggestion by ID
func (s *MongoReviewStorage) GetSuggestion(suggestionID primitive.ObjectID) (*ReviewSuggestion, error) {
	ctx := context.Background()

	var suggestion ReviewSuggestion
	err := s.suggestionsCollection.FindOne(ctx, bson.M{"_id": suggestionID}).Decode(&suggestion)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("suggestion not found")
		}
		return nil, fmt.Errorf("failed to get suggestion: %w", err)
	}

	return &suggestion, nil
}

// ListPendingSuggestions retrieves pending suggestions for a collection
func (s *MongoReviewStorage) ListPendingSuggestions(collectionName string, limit int) ([]*ReviewSuggestion, error) {
	ctx := context.Background()

	filter := bson.M{"status": "pending"}
	if collectionName != "" {
		filter["collectionName"] = collectionName
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := s.suggestionsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list suggestions: %w", err)
	}
	defer cursor.Close(ctx)

	var suggestions []*ReviewSuggestion
	if err := cursor.All(ctx, &suggestions); err != nil {
		return nil, fmt.Errorf("failed to decode suggestions: %w", err)
	}

	if suggestions == nil {
		suggestions = make([]*ReviewSuggestion, 0)
	}

	return suggestions, nil
}

// UpdateSuggestionStatus updates the status of a suggestion
func (s *MongoReviewStorage) UpdateSuggestionStatus(suggestionID primitive.ObjectID, status string) error {
	ctx := context.Background()

	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	// Set approved/rejected timestamp based on status
	if status == "approved" {
		update["$set"].(bson.M)["approvedAt"] = now
	} else if status == "rejected" {
		update["$set"].(bson.M)["rejectedAt"] = now
	}

	result, err := s.suggestionsCollection.UpdateOne(ctx, bson.M{"_id": suggestionID}, update)
	if err != nil {
		return fmt.Errorf("failed to update suggestion status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("suggestion not found")
	}

	return nil
}
