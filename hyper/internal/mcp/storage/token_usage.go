package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// TokenUsage represents a single API call's token usage and cost
type TokenUsage struct {
	ID              string    `json:"id" bson:"_id"`
	RequestID       string    `json:"requestId" bson:"requestId"`           // Unique request identifier
	Provider        string    `json:"provider" bson:"provider"`             // "claude", "openai", "groq", etc.
	Model           string    `json:"model" bson:"model"`                   // "claude-3-5-sonnet", "gpt-4", etc.
	InputTokens     int       `json:"inputTokens" bson:"inputTokens"`       // Tokens in the prompt
	OutputTokens    int       `json:"outputTokens" bson:"outputTokens"`     // Tokens in the response
	TotalTokens     int       `json:"totalTokens" bson:"totalTokens"`       // Sum of input + output
	InputCost       float64   `json:"inputCost" bson:"inputCost"`           // Cost for input tokens (in USD)
	OutputCost      float64   `json:"outputCost" bson:"outputCost"`         // Cost for output tokens (in USD)
	TotalCost       float64   `json:"totalCost" bson:"totalCost"`           // Sum of input + output cost
	CacheReadTokens int       `json:"cacheReadTokens,omitempty" bson:"cacheReadTokens,omitempty"` // Claude cache reads
	CacheCreationTokens int   `json:"cacheCreationTokens,omitempty" bson:"cacheCreationTokens,omitempty"` // Claude cache writes
	Duration        int64     `json:"duration" bson:"duration"`             // Request duration in milliseconds
	Status          string    `json:"status" bson:"status"`                 // "success", "error", "partial"
	ErrorMessage    string    `json:"errorMessage,omitempty" bson:"errorMessage,omitempty"`
	CreatedAt       time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt" bson:"updatedAt"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"` // Custom metadata
}

// TokenUsageStats aggregates token usage statistics
type TokenUsageStats struct {
	Provider            string    `json:"provider" bson:"provider"`
	Model               string    `json:"model" bson:"model"`
	TotalRequests       int       `json:"totalRequests" bson:"totalRequests"`
	TotalInputTokens    int       `json:"totalInputTokens" bson:"totalInputTokens"`
	TotalOutputTokens   int       `json:"totalOutputTokens" bson:"totalOutputTokens"`
	TotalTokens         int       `json:"totalTokens" bson:"totalTokens"`
	TotalCost           float64   `json:"totalCost" bson:"totalCost"`
	AverageCostPerRequest float64 `json:"averageCostPerRequest" bson:"averageCostPerRequest"`
	AverageDuration     int64     `json:"averageDuration" bson:"averageDuration"`
	SuccessRate         float64   `json:"successRate" bson:"successRate"`
	PeriodStart         time.Time `json:"periodStart" bson:"periodStart"`
	PeriodEnd           time.Time `json:"periodEnd" bson:"periodEnd"`
}

// TokenUsageStorage defines the interface for token usage persistence
type TokenUsageStorage interface {
	// Record a new token usage entry
	RecordTokenUsage(ctx context.Context, usage *TokenUsage) error

	// Get token usage by request ID
	GetTokenUsageByRequestID(ctx context.Context, requestID string) (*TokenUsage, error)

	// List token usage records with filters
	ListTokenUsage(ctx context.Context, filter bson.M, limit int, offset int) ([]*TokenUsage, int, error)

	// Get token usage statistics for a time period
	GetTokenUsageStats(ctx context.Context, provider string, model string, startTime time.Time, endTime time.Time) (*TokenUsageStats, error)

	// Get token usage statistics by provider
	GetTokenUsageByProvider(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]*TokenUsageStats, error)

	// Get total cost for a time period
	GetTotalCost(ctx context.Context, startTime time.Time, endTime time.Time) (float64, error)

	// Get cost breakdown by model
	GetCostByModel(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]float64, error)

	// Delete old token usage records (for cleanup)
	DeleteOldRecords(ctx context.Context, olderThan time.Time) (int64, error)
}

// MongoTokenUsageStorage implements TokenUsageStorage using MongoDB
type MongoTokenUsageStorage struct {
	collection *mongo.Collection
	logger     *zap.Logger
}

// NewMongoTokenUsageStorage creates a new MongoDB token usage storage
func NewMongoTokenUsageStorage(db *mongo.Database, logger *zap.Logger) (*MongoTokenUsageStorage, error) {
	collection := db.Collection("token_usage")
	storage := &MongoTokenUsageStorage{
		collection: collection,
		logger:     logger,
	}

	// Create indexes
	ctx := context.Background()

	// Index on requestId (unique)
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "requestId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create requestId index: %w", err)
	}

	// Index on provider and model for stats queries
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "provider", Value: 1},
			{Key: "model", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create provider/model index: %w", err)
	}

	// Index on createdAt for time-based queries
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "createdAt", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create createdAt index: %w", err)
	}

	// Compound index for efficient stats queries
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "provider", Value: 1},
			{Key: "model", Value: 1},
			{Key: "createdAt", Value: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stats query index: %w", err)
	}

	// TTL index to auto-delete records older than 90 days
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "createdAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(90 * 24 * 60 * 60), // 90 days
	})
	if err != nil {
		logger.Warn("failed to create TTL index (non-critical)", zap.Error(err))
	}

	return storage, nil
}

// RecordTokenUsage stores a token usage record
func (s *MongoTokenUsageStorage) RecordTokenUsage(ctx context.Context, usage *TokenUsage) error {
	if usage.ID == "" {
		return fmt.Errorf("token usage ID is required")
	}

	now := time.Now().UTC()
	usage.CreatedAt = now
	usage.UpdatedAt = now

	// Calculate total tokens if not already set
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}

	// Calculate total cost if not already set
	if usage.TotalCost == 0 {
		usage.TotalCost = usage.InputCost + usage.OutputCost
	}

	_, err := s.collection.InsertOne(ctx, usage)
	if err != nil {
		return fmt.Errorf("failed to insert token usage: %w", err)
	}

	s.logger.Debug("recorded token usage",
		zap.String("requestId", usage.RequestID),
		zap.String("provider", usage.Provider),
		zap.String("model", usage.Model),
		zap.Int("totalTokens", usage.TotalTokens),
		zap.Float64("totalCost", usage.TotalCost))

	return nil
}

// GetTokenUsageByRequestID retrieves token usage by request ID
func (s *MongoTokenUsageStorage) GetTokenUsageByRequestID(ctx context.Context, requestID string) (*TokenUsage, error) {
	var usage TokenUsage
	err := s.collection.FindOne(ctx, bson.M{"requestId": requestID}).Decode(&usage)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find token usage: %w", err)
	}
	return &usage, nil
}

// ListTokenUsage lists token usage records with filters
func (s *MongoTokenUsageStorage) ListTokenUsage(ctx context.Context, filter bson.M, limit int, offset int) ([]*TokenUsage, int, error) {
	// Get total count
	totalCount, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %w", err)
	}

	// Set defaults
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	// Query with pagination
	opts := options.Find().
		SetSort(bson.M{"createdAt": -1}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find token usage records: %w", err)
	}
	defer cursor.Close(ctx)

	var usages []*TokenUsage
	if err = cursor.All(ctx, &usages); err != nil {
		return nil, 0, fmt.Errorf("failed to decode token usage records: %w", err)
	}

	return usages, int(totalCount), nil
}

// GetTokenUsageStats gets statistics for a specific provider/model combination
func (s *MongoTokenUsageStorage) GetTokenUsageStats(ctx context.Context, provider string, model string, startTime time.Time, endTime time.Time) (*TokenUsageStats, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "provider", Value: provider},
			{Key: "model", Value: model},
			{Key: "createdAt", Value: bson.D{
				{Key: "$gte", Value: startTime},
				{Key: "$lte", Value: endTime},
			}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "totalRequests", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "totalInputTokens", Value: bson.D{{Key: "$sum", Value: "$inputTokens"}}},
			{Key: "totalOutputTokens", Value: bson.D{{Key: "$sum", Value: "$outputTokens"}}},
			{Key: "totalTokens", Value: bson.D{{Key: "$sum", Value: "$totalTokens"}}},
			{Key: "totalCost", Value: bson.D{{Key: "$sum", Value: "$totalCost"}}},
			{Key: "averageDuration", Value: bson.D{{Key: "$avg", Value: "$duration"}}},
			{Key: "successCount", Value: bson.D{{Key: "$sum", Value: bson.D{
				{Key: "$cond", Value: bson.A{
					bson.D{{Key: "$eq", Value: bson.A{"$status", "success"}}},
					1,
					0,
				}},
			}}}},
		}}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate token usage stats: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	if len(results) == 0 {
		return &TokenUsageStats{
			Provider:    provider,
			Model:       model,
			PeriodStart: startTime,
			PeriodEnd:   endTime,
		}, nil
	}

	result := results[0]
	totalRequests := int(result["totalRequests"].(int32))
	successCount := int(result["successCount"].(int32))
	successRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
	}

	averageCostPerRequest := 0.0
	totalCost := result["totalCost"].(float64)
	if totalRequests > 0 {
		averageCostPerRequest = totalCost / float64(totalRequests)
	}

	return &TokenUsageStats{
		Provider:            provider,
		Model:               model,
		TotalRequests:       totalRequests,
		TotalInputTokens:    int(result["totalInputTokens"].(int32)),
		TotalOutputTokens:   int(result["totalOutputTokens"].(int32)),
		TotalTokens:         int(result["totalTokens"].(int32)),
		TotalCost:           totalCost,
		AverageCostPerRequest: averageCostPerRequest,
		AverageDuration:     int64(result["averageDuration"].(float64)),
		SuccessRate:         successRate,
		PeriodStart:         startTime,
		PeriodEnd:           endTime,
	}, nil
}

// GetTokenUsageByProvider gets statistics for all models of a provider
func (s *MongoTokenUsageStorage) GetTokenUsageByProvider(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]*TokenUsageStats, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "createdAt", Value: bson.D{
				{Key: "$gte", Value: startTime},
				{Key: "$lte", Value: endTime},
			}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "provider", Value: "$provider"},
				{Key: "model", Value: "$model"},
			}},
			{Key: "totalRequests", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "totalInputTokens", Value: bson.D{{Key: "$sum", Value: "$inputTokens"}}},
			{Key: "totalOutputTokens", Value: bson.D{{Key: "$sum", Value: "$outputTokens"}}},
			{Key: "totalTokens", Value: bson.D{{Key: "$sum", Value: "$totalTokens"}}},
			{Key: "totalCost", Value: bson.D{{Key: "$sum", Value: "$totalCost"}}},
			{Key: "averageDuration", Value: bson.D{{Key: "$avg", Value: "$duration"}}},
			{Key: "successCount", Value: bson.D{{Key: "$sum", Value: bson.D{
				{Key: "$cond", Value: bson.A{
					bson.D{{Key: "$eq", Value: bson.A{"$status", "success"}}},
					1,
					0,
				}},
			}}}},
		}}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate by provider: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode provider stats: %w", err)
	}

	stats := make(map[string]*TokenUsageStats)
	for _, result := range results {
		id := result["_id"].(bson.M)
		provider := id["provider"].(string)
		model := id["model"].(string)
		key := provider + ":" + model

		totalRequests := int(result["totalRequests"].(int32))
		successCount := int(result["successCount"].(int32))
		successRate := 0.0
		if totalRequests > 0 {
			successRate = float64(successCount) / float64(totalRequests)
		}

		totalCost := result["totalCost"].(float64)
		averageCostPerRequest := 0.0
		if totalRequests > 0 {
			averageCostPerRequest = totalCost / float64(totalRequests)
		}

		stats[key] = &TokenUsageStats{
			Provider:            provider,
			Model:               model,
			TotalRequests:       totalRequests,
			TotalInputTokens:    int(result["totalInputTokens"].(int32)),
			TotalOutputTokens:   int(result["totalOutputTokens"].(int32)),
			TotalTokens:         int(result["totalTokens"].(int32)),
			TotalCost:           totalCost,
			AverageCostPerRequest: averageCostPerRequest,
			AverageDuration:     int64(result["averageDuration"].(float64)),
			SuccessRate:         successRate,
			PeriodStart:         startTime,
			PeriodEnd:           endTime,
		}
	}

	return stats, nil
}

// GetTotalCost gets the total cost for a time period
func (s *MongoTokenUsageStorage) GetTotalCost(ctx context.Context, startTime time.Time, endTime time.Time) (float64, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "createdAt", Value: bson.D{
				{Key: "$gte", Value: startTime},
				{Key: "$lte", Value: endTime},
			}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "totalCost", Value: bson.D{{Key: "$sum", Value: "$totalCost"}}},
		}}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate total cost: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return 0, fmt.Errorf("failed to decode total cost: %w", err)
	}

	if len(results) == 0 {
		return 0, nil
	}

	return results[0]["totalCost"].(float64), nil
}

// GetCostByModel gets cost breakdown by model
func (s *MongoTokenUsageStorage) GetCostByModel(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]float64, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "createdAt", Value: bson.D{
				{Key: "$gte", Value: startTime},
				{Key: "$lte", Value: endTime},
			}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "provider", Value: "$provider"},
				{Key: "model", Value: "$model"},
			}},
			{Key: "totalCost", Value: bson.D{{Key: "$sum", Value: "$totalCost"}}},
		}}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate cost by model: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode cost by model: %w", err)
	}

	costByModel := make(map[string]float64)
	for _, result := range results {
		id := result["_id"].(bson.M)
		provider := id["provider"].(string)
		model := id["model"].(string)
		key := provider + ":" + model
		costByModel[key] = result["totalCost"].(float64)
	}

	return costByModel, nil
}

// DeleteOldRecords deletes token usage records older than the specified time
func (s *MongoTokenUsageStorage) DeleteOldRecords(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := s.collection.DeleteMany(ctx, bson.M{
		"createdAt": bson.M{"$lt": olderThan},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to delete old records: %w", err)
	}
	return result.DeletedCount, nil
}
