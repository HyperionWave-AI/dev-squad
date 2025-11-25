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

// TokenMetrics represents a single token usage event with cost calculation
type TokenMetrics struct {
	ID                  string                 `json:"id" bson:"_id"`
	RequestID           string                 `json:"requestId" bson:"requestId"`           // Unique request identifier
	Provider            string                 `json:"provider" bson:"provider"`             // "openai", "anthropic", "groq", etc.
	Model               string                 `json:"model" bson:"model"`                   // "gpt-4", "claude-3-opus", etc.
	InputTokens         int                    `json:"inputTokens" bson:"inputTokens"`       // Tokens in the prompt
	OutputTokens        int                    `json:"outputTokens" bson:"outputTokens"`     // Tokens in the response
	TotalTokens         int                    `json:"totalTokens" bson:"totalTokens"`       // Sum of input + output
	InputCost           float64                `json:"inputCost" bson:"inputCost"`           // Cost for input tokens (USD)
	OutputCost          float64                `json:"outputCost" bson:"outputCost"`         // Cost for output tokens (USD)
	TotalCost           float64                `json:"totalCost" bson:"totalCost"`           // Sum of input + output cost
	CacheHits           int                    `json:"cacheHits" bson:"cacheHits"`           // Number of cache hits
	CacheMisses         int                    `json:"cacheMisses" bson:"cacheMisses"`       // Number of cache misses
	CacheHitRate        float64                `json:"cacheHitRate" bson:"cacheHitRate"`     // Percentage of cache hits
	ToolName            string                 `json:"toolName,omitempty" bson:"toolName,omitempty"` // Tool that was executed
	Duration            int64                  `json:"duration" bson:"duration"`             // Request duration in milliseconds
	Status              string                 `json:"status" bson:"status"`                 // "success", "error", "partial"
	ErrorMessage        string                 `json:"errorMessage,omitempty" bson:"errorMessage,omitempty"`
	CreatedAt           time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt           time.Time              `json:"updatedAt" bson:"updatedAt"`
	Metadata            map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"` // Custom metadata
}

// MetricsStats aggregates metrics statistics
type MetricsStats struct {
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
	AverageCacheHitRate float64   `json:"averageCacheHitRate" bson:"averageCacheHitRate"`
	PeriodStart         time.Time `json:"periodStart" bson:"periodStart"`
	PeriodEnd           time.Time `json:"periodEnd" bson:"periodEnd"`
}

// CostBreakdown provides cost analysis by provider/model
type CostBreakdown struct {
	Provider    string  `json:"provider" bson:"provider"`
	Model       string  `json:"model" bson:"model"`
	TotalCost   float64 `json:"totalCost" bson:"totalCost"`
	Percentage  float64 `json:"percentage" bson:"percentage"`
	RequestCount int    `json:"requestCount" bson:"requestCount"`
}

// MetricsStorage defines the interface for metrics persistence
type MetricsStorage interface {
	// Record a new metrics entry
	RecordMetrics(ctx context.Context, metrics *TokenMetrics) error

	// Get metrics by request ID
	GetMetricsByRequestID(ctx context.Context, requestID string) (*TokenMetrics, error)

	// List metrics records with filters
	ListMetrics(ctx context.Context, filter bson.M, limit int, offset int) ([]*TokenMetrics, int, error)

	// Get metrics statistics for a time period
	GetMetricsStats(ctx context.Context, provider string, model string, startTime time.Time, endTime time.Time) (*MetricsStats, error)

	// Get metrics statistics by provider
	GetMetricsByProvider(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]*MetricsStats, error)

	// Get total cost for a time period
	GetTotalCost(ctx context.Context, startTime time.Time, endTime time.Time) (float64, error)

	// Get cost breakdown by model
	GetCostByModel(ctx context.Context, startTime time.Time, endTime time.Time) ([]CostBreakdown, error)

	// Get cache hit rate statistics
	GetCacheHitRateStats(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]float64, error)

	// Delete old metrics records (for cleanup)
	DeleteOldRecords(ctx context.Context, olderThan time.Time) (int64, error)
}

// MongoMetricsStorage implements MetricsStorage using MongoDB
type MongoMetricsStorage struct {
	collection *mongo.Collection
	logger     *zap.Logger
}

// NewMongoMetricsStorage creates a new MongoDB metrics storage
func NewMongoMetricsStorage(db *mongo.Database, logger *zap.Logger) (*MongoMetricsStorage, error) {
	collection := db.Collection("metrics")
	storage := &MongoMetricsStorage{
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

	// Index on toolName for tool-specific analysis
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "toolName", Value: 1}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create toolName index: %w", err)
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

// RecordMetrics stores a metrics record
func (s *MongoMetricsStorage) RecordMetrics(ctx context.Context, metrics *TokenMetrics) error {
	if metrics.ID == "" {
		return fmt.Errorf("metrics ID is required")
	}

	now := time.Now().UTC()
	metrics.CreatedAt = now
	metrics.UpdatedAt = now

	// Calculate total tokens if not already set
	if metrics.TotalTokens == 0 {
		metrics.TotalTokens = metrics.InputTokens + metrics.OutputTokens
	}

	// Calculate total cost if not already set
	if metrics.TotalCost == 0 {
		metrics.TotalCost = metrics.InputCost + metrics.OutputCost
	}

	// Calculate cache hit rate
	totalCacheOps := metrics.CacheHits + metrics.CacheMisses
	if totalCacheOps > 0 {
		metrics.CacheHitRate = float64(metrics.CacheHits) / float64(totalCacheOps)
	}

	_, err := s.collection.InsertOne(ctx, metrics)
	if err != nil {
		return fmt.Errorf("failed to insert metrics: %w", err)
	}

	s.logger.Debug("recorded metrics",
		zap.String("requestId", metrics.RequestID),
		zap.String("provider", metrics.Provider),
		zap.String("model", metrics.Model),
		zap.Int("totalTokens", metrics.TotalTokens),
		zap.Float64("totalCost", metrics.TotalCost),
		zap.Float64("cacheHitRate", metrics.CacheHitRate))

	return nil
}

// GetMetricsByRequestID retrieves metrics by request ID
func (s *MongoMetricsStorage) GetMetricsByRequestID(ctx context.Context, requestID string) (*TokenMetrics, error) {
	var metrics TokenMetrics
	err := s.collection.FindOne(ctx, bson.M{"requestId": requestID}).Decode(&metrics)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find metrics: %w", err)
	}
	return &metrics, nil
}

// ListMetrics lists metrics records with filters
func (s *MongoMetricsStorage) ListMetrics(ctx context.Context, filter bson.M, limit int, offset int) ([]*TokenMetrics, int, error) {
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
		return nil, 0, fmt.Errorf("failed to find metrics records: %w", err)
	}
	defer cursor.Close(ctx)

	var metrics []*TokenMetrics
	if err = cursor.All(ctx, &metrics); err != nil {
		return nil, 0, fmt.Errorf("failed to decode metrics records: %w", err)
	}

	return metrics, int(totalCount), nil
}

// GetMetricsStats gets statistics for a specific provider/model combination
func (s *MongoMetricsStorage) GetMetricsStats(ctx context.Context, provider string, model string, startTime time.Time, endTime time.Time) (*MetricsStats, error) {
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
			{Key: "averageCacheHitRate", Value: bson.D{{Key: "$avg", Value: "$cacheHitRate"}}},
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
		return nil, fmt.Errorf("failed to aggregate metrics stats: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	if len(results) == 0 {
		return &MetricsStats{
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

	averageCacheHitRate := 0.0
	if cacheHitRate, ok := result["averageCacheHitRate"].(float64); ok {
		averageCacheHitRate = cacheHitRate
	}

	return &MetricsStats{
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
		AverageCacheHitRate: averageCacheHitRate,
		PeriodStart:         startTime,
		PeriodEnd:           endTime,
	}, nil
}

// GetMetricsByProvider gets statistics for all models of a provider
func (s *MongoMetricsStorage) GetMetricsByProvider(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]*MetricsStats, error) {
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
			{Key: "averageCacheHitRate", Value: bson.D{{Key: "$avg", Value: "$cacheHitRate"}}},
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

	stats := make(map[string]*MetricsStats)
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

		averageCacheHitRate := 0.0
		if cacheHitRate, ok := result["averageCacheHitRate"].(float64); ok {
			averageCacheHitRate = cacheHitRate
		}

		stats[key] = &MetricsStats{
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
			AverageCacheHitRate: averageCacheHitRate,
			PeriodStart:         startTime,
			PeriodEnd:           endTime,
		}
	}

	return stats, nil
}

// GetTotalCost gets the total cost for a time period
func (s *MongoMetricsStorage) GetTotalCost(ctx context.Context, startTime time.Time, endTime time.Time) (float64, error) {
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
func (s *MongoMetricsStorage) GetCostByModel(ctx context.Context, startTime time.Time, endTime time.Time) ([]CostBreakdown, error) {
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
			{Key: "requestCount", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "totalCost", Value: -1}}}},
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

	// Calculate total cost for percentage calculation
	totalCost := 0.0
	for _, result := range results {
		totalCost += result["totalCost"].(float64)
	}

	// Build breakdown
	var breakdown []CostBreakdown
	for _, result := range results {
		id := result["_id"].(bson.M)
		provider := id["provider"].(string)
		model := id["model"].(string)
		cost := result["totalCost"].(float64)
		requestCount := int(result["requestCount"].(int32))

		percentage := 0.0
		if totalCost > 0 {
			percentage = (cost / totalCost) * 100
		}

		breakdown = append(breakdown, CostBreakdown{
			Provider:     provider,
			Model:        model,
			TotalCost:    cost,
			Percentage:   percentage,
			RequestCount: requestCount,
		})
	}

	return breakdown, nil
}

// GetCacheHitRateStats gets cache hit rate statistics
func (s *MongoMetricsStorage) GetCacheHitRateStats(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]float64, error) {
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
			{Key: "averageCacheHitRate", Value: bson.D{{Key: "$avg", Value: "$cacheHitRate"}}},
		}}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate cache hit rates: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode cache hit rates: %w", err)
	}

	stats := make(map[string]float64)
	for _, result := range results {
		id := result["_id"].(bson.M)
		provider := id["provider"].(string)
		model := id["model"].(string)
		key := provider + ":" + model
		stats[key] = result["averageCacheHitRate"].(float64)
	}

	return stats, nil
}

// DeleteOldRecords deletes metrics records older than the specified time
func (s *MongoMetricsStorage) DeleteOldRecords(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := s.collection.DeleteMany(ctx, bson.M{
		"createdAt": bson.M{"$lt": olderThan},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to delete old records: %w", err)
	}
	return result.DeletedCount, nil
}
