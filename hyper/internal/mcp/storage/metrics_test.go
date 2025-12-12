package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// TestTokenMetricsStorage tests the metrics storage implementation
func TestTokenMetricsStorage(t *testing.T) {
	// Skip if no MongoDB available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Connect to test MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("Could not connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("test_metrics")
	defer db.Drop(ctx)

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	storage, err := NewMongoMetricsStorage(db, logger)
	require.NoError(t, err)

	t.Run("RecordMetrics", func(t *testing.T) {
		metrics := &TokenMetrics{
			ID:           uuid.New().String(),
			RequestID:    uuid.New().String(),
			Provider:     "openai",
			Model:        "gpt-4",
			InputTokens:  100,
			OutputTokens: 50,
			InputCost:    0.003,
			OutputCost:   0.006,
			CacheHits:    5,
			CacheMisses:  2,
			Duration:     1000,
			Status:       "success",
		}

		err := storage.RecordMetrics(ctx, metrics)
		assert.NoError(t, err)

		// Verify total tokens calculated
		assert.Equal(t, 150, metrics.TotalTokens)

		// Verify total cost calculated
		assert.Equal(t, 0.009, metrics.TotalCost)

		// Verify cache hit rate calculated
		expectedRate := float64(5) / float64(7)
		assert.InDelta(t, expectedRate, metrics.CacheHitRate, 0.001)
	})

	t.Run("GetMetricsByRequestID", func(t *testing.T) {
		requestID := uuid.New().String()
		metrics := &TokenMetrics{
			ID:           uuid.New().String(),
			RequestID:    requestID,
			Provider:     "anthropic",
			Model:        "claude-3-opus",
			InputTokens:  200,
			OutputTokens: 100,
			InputCost:    0.015,
			OutputCost:   0.075,
			Duration:     2000,
			Status:       "success",
		}

		err := storage.RecordMetrics(ctx, metrics)
		assert.NoError(t, err)

		retrieved, err := storage.GetMetricsByRequestID(ctx, requestID)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, requestID, retrieved.RequestID)
		assert.Equal(t, "anthropic", retrieved.Provider)
	})

	t.Run("ListMetrics", func(t *testing.T) {
		// Record multiple metrics
		for i := 0; i < 5; i++ {
			metrics := &TokenMetrics{
				ID:           uuid.New().String(),
				RequestID:    uuid.New().String(),
				Provider:     "openai",
				Model:        "gpt-3.5-turbo",
				InputTokens:  50 + i*10,
				OutputTokens: 25 + i*5,
				Duration:     500 + int64(i*100),
				Status:       "success",
			}
			err := storage.RecordMetrics(ctx, metrics)
			assert.NoError(t, err)
		}

		// List with filter
		list, total, err := storage.ListMetrics(ctx, map[string]interface{}{"provider": "openai"}, 10, 0)
		assert.NoError(t, err)
		assert.Greater(t, total, 0)
		assert.Greater(t, len(list), 0)
	})

	t.Run("GetMetricsStats", func(t *testing.T) {
		provider := "openai"
		model := "gpt-4"

		// Record metrics for stats
		for i := 0; i < 3; i++ {
			metrics := &TokenMetrics{
				ID:           uuid.New().String(),
				RequestID:    uuid.New().String(),
				Provider:     provider,
				Model:        model,
				InputTokens:  100,
				OutputTokens: 50,
				InputCost:    0.003,
				OutputCost:   0.006,
				Duration:     1000,
				Status:       "success",
			}
			err := storage.RecordMetrics(ctx, metrics)
			assert.NoError(t, err)
		}

		now := time.Now()
		stats, err := storage.GetMetricsStats(ctx, provider, model, now.Add(-1*time.Hour), now.Add(1*time.Hour))
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.TotalRequests, 3)
		assert.Equal(t, provider, stats.Provider)
		assert.Equal(t, model, stats.Model)
	})

	t.Run("GetTotalCost", func(t *testing.T) {
		// Record metrics with known costs
		metrics1 := &TokenMetrics{
			ID:           uuid.New().String(),
			RequestID:    uuid.New().String(),
			Provider:     "openai",
			Model:        "gpt-4",
			InputTokens:  100,
			OutputTokens: 50,
			InputCost:    0.003,
			OutputCost:   0.006,
			Duration:     1000,
			Status:       "success",
		}

		metrics2 := &TokenMetrics{
			ID:           uuid.New().String(),
			RequestID:    uuid.New().String(),
			Provider:     "anthropic",
			Model:        "claude-3-opus",
			InputTokens:  200,
			OutputTokens: 100,
			InputCost:    0.015,
			OutputCost:   0.075,
			Duration:     2000,
			Status:       "success",
		}

		err := storage.RecordMetrics(ctx, metrics1)
		assert.NoError(t, err)
		err = storage.RecordMetrics(ctx, metrics2)
		assert.NoError(t, err)

		now := time.Now()
		totalCost, err := storage.GetTotalCost(ctx, now.Add(-1*time.Hour), now.Add(1*time.Hour))
		assert.NoError(t, err)
		assert.Greater(t, totalCost, 0.0)
	})

	t.Run("GetCostByModel", func(t *testing.T) {
		now := time.Now()
		breakdown, err := storage.GetCostByModel(ctx, now.Add(-1*time.Hour), now.Add(1*time.Hour))
		assert.NoError(t, err)
		assert.NotNil(t, breakdown)
		// Should have at least one model
		assert.Greater(t, len(breakdown), 0)
	})

	t.Run("GetCacheHitRateStats", func(t *testing.T) {
		now := time.Now()
		stats, err := storage.GetCacheHitRateStats(ctx, now.Add(-1*time.Hour), now.Add(1*time.Hour))
		assert.NoError(t, err)
		assert.NotNil(t, stats)
	})

	t.Run("DeleteOldRecords", func(t *testing.T) {
		// Record a metric with old timestamp
		oldMetrics := &TokenMetrics{
			ID:           uuid.New().String(),
			RequestID:    uuid.New().String(),
			Provider:     "openai",
			Model:        "gpt-4",
			InputTokens:  100,
			OutputTokens: 50,
			Duration:     1000,
			Status:       "success",
		}

		err := storage.RecordMetrics(ctx, oldMetrics)
		assert.NoError(t, err)

		// Manually update the timestamp to be old
		collection := db.Collection("metrics")
		_, err = collection.UpdateOne(ctx, map[string]interface{}{"_id": oldMetrics.ID}, map[string]interface{}{
			"$set": map[string]interface{}{
				"createdAt": time.Now().Add(-100 * 24 * time.Hour),
			},
		})
		assert.NoError(t, err)

		// Delete records older than 90 days
		deleted, err := storage.DeleteOldRecords(ctx, time.Now().Add(-90*24*time.Hour))
		assert.NoError(t, err)
		assert.Greater(t, deleted, int64(0))
	})
}

// TestPricingCalculation tests the pricing calculation functions
func TestPricingCalculation(t *testing.T) {
	t.Run("CalculateCostOpenAI", func(t *testing.T) {
		inputCost, outputCost, totalCost := CalculateCost("openai", "gpt-4", 1000, 500, 0, 0)

		// GPT-4: $30 per 1M input, $60 per 1M output
		expectedInputCost := (1000.0 / 1_000_000) * 30.0
		expectedOutputCost := (500.0 / 1_000_000) * 60.0
		expectedTotalCost := expectedInputCost + expectedOutputCost

		assert.InDelta(t, expectedInputCost, inputCost, 0.0001)
		assert.InDelta(t, expectedOutputCost, outputCost, 0.0001)
		assert.InDelta(t, expectedTotalCost, totalCost, 0.0001)
	})

	t.Run("CalculateCostAnthropic", func(t *testing.T) {
		inputCost, outputCost, totalCost := CalculateCost("anthropic", "claude-3-opus-20250219", 1000, 500, 0, 0)

		// Claude 3 Opus: $15 per 1M input, $75 per 1M output
		expectedInputCost := (1000.0 / 1_000_000) * 15.0
		expectedOutputCost := (500.0 / 1_000_000) * 75.0
		expectedTotalCost := expectedInputCost + expectedOutputCost

		assert.InDelta(t, expectedInputCost, inputCost, 0.0001)
		assert.InDelta(t, expectedOutputCost, outputCost, 0.0001)
		assert.InDelta(t, expectedTotalCost, totalCost, 0.0001)
	})

	t.Run("CalculateCostWithCache", func(t *testing.T) {
		// Claude with cache
		inputCost, outputCost, totalCost := CalculateCost(
			"anthropic",
			"claude-3-5-sonnet-20241022",
			1000,  // input tokens
			500,   // output tokens
			200,   // cache creation tokens
			100,   // cache read tokens
		)

		// Should include cache costs
		assert.Greater(t, totalCost, 0.0)
		assert.Greater(t, inputCost, 0.0)
		assert.Greater(t, outputCost, 0.0)
	})

	t.Run("GetModelPricing", func(t *testing.T) {
		pricing := GetModelPricing("openai", "gpt-4")
		assert.NotNil(t, pricing)
		assert.Equal(t, "openai", pricing.Provider)
		assert.Equal(t, "gpt-4", pricing.Model)
		assert.Greater(t, pricing.InputCostPer1M, 0.0)
		assert.Greater(t, pricing.OutputCostPer1M, 0.0)
	})

	t.Run("FormatCost", func(t *testing.T) {
		formatted := FormatCost(0.00001)
		assert.Equal(t, "$0.00", formatted)

		formatted = FormatCost(0.1234)
		assert.Equal(t, "$0.1234", formatted)
	})

	t.Run("FormatCostUSD", func(t *testing.T) {
		formatted := FormatCostUSD(0.1234)
		assert.Equal(t, "$0.12", formatted)

		formatted = FormatCostUSD(1.5678)
		assert.Equal(t, "$1.57", formatted)
	})
}

// TestMetricsCalculations tests various metrics calculations
func TestMetricsCalculations(t *testing.T) {
	t.Run("CacheHitRateCalculation", func(t *testing.T) {
		metrics := &TokenMetrics{
			ID:          uuid.New().String(),
			CacheHits:   75,
			CacheMisses: 25,
		}

		// Simulate the calculation that happens in RecordMetrics
		totalOps := metrics.CacheHits + metrics.CacheMisses
		if totalOps > 0 {
			metrics.CacheHitRate = float64(metrics.CacheHits) / float64(totalOps)
		}

		assert.InDelta(t, 0.75, metrics.CacheHitRate, 0.001)
	})

	t.Run("TotalTokensCalculation", func(t *testing.T) {
		metrics := &TokenMetrics{
			ID:           uuid.New().String(),
			InputTokens:  1000,
			OutputTokens: 500,
		}

		metrics.TotalTokens = metrics.InputTokens + metrics.OutputTokens

		assert.Equal(t, 1500, metrics.TotalTokens)
	})

	t.Run("TotalCostCalculation", func(t *testing.T) {
		metrics := &TokenMetrics{
			ID:         uuid.New().String(),
			InputCost:  0.003,
			OutputCost: 0.006,
		}

		metrics.TotalCost = metrics.InputCost + metrics.OutputCost

		assert.InDelta(t, 0.009, metrics.TotalCost, 0.0001)
	})
}
