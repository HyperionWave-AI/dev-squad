package summarizer

import (
	"fmt"
	"testing"
	"time"
	"go.uber.org/zap"
)

// BenchmarkCacheGet benchmarks cache Get operations
func BenchmarkCacheGet(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(1000, 24*time.Hour, logger)
	defer cache.Clear()

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key_%d", i)
		summary := &CodeSummary{
			Text:       fmt.Sprintf("Summary %d", i),
			Type:       "cached",
			TokenCount: 100,
		}
		cache.Set(key, summary)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%100)
		cache.Get(key)
	}
}

// BenchmarkCacheSet benchmarks cache Set operations
func BenchmarkCacheSet(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(1000, 24*time.Hour, logger)
	defer cache.Clear()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		summary := &CodeSummary{
			Text:       fmt.Sprintf("Summary %d", i),
			Type:       "cached",
			TokenCount: 100,
		}
		cache.Set(key, summary)
	}
}

// BenchmarkCacheGetSet benchmarks mixed Get/Set operations
func BenchmarkCacheGetSet(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(1000, 24*time.Hour, logger)
	defer cache.Clear()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 70% Gets, 30% Sets
		if i%10 < 7 {
			key := fmt.Sprintf("key_%d", i%100)
			cache.Get(key)
		} else {
			key := fmt.Sprintf("key_%d", i)
			summary := &CodeSummary{
				Text:       fmt.Sprintf("Summary %d", i),
				Type:       "cached",
				TokenCount: 100,
			}
			cache.Set(key, summary)
		}
	}
}

// BenchmarkCacheStats benchmarks cache Stats operations
func BenchmarkCacheStats(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(1000, 24*time.Hour, logger)
	defer cache.Clear()

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key_%d", i)
		summary := &CodeSummary{
			Text:       fmt.Sprintf("Summary %d", i),
			Type:       "cached",
			TokenCount: 100,
		}
		cache.Set(key, summary)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Stats()
	}
}

// BenchmarkCacheEviction benchmarks cache eviction performance
func BenchmarkCacheEviction(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(100, 24*time.Hour, logger)
	defer cache.Clear()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		summary := &CodeSummary{
			Text:       fmt.Sprintf("Summary %d", i),
			Type:       "cached",
			TokenCount: 100,
		}
		cache.Set(key, summary)
	}
}

// BenchmarkTokenManagerCanSummarize benchmarks token manager CanSummarize
func BenchmarkTokenManagerCanSummarize(b *testing.B) {
	logger := zap.NewNop()
	estimator := &SimpleTokenEstimator{}
	tm := NewTokenManager(5000, 100, estimator, logger)

	code := `func main() {
		fmt.Println("Hello, World!")
		for i := 0; i < 10; i++ {
			fmt.Printf("Iteration %d\n", i)
		}
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.CanSummarize(code)
	}
}

// BenchmarkTokenManagerRecordUsage benchmarks token manager RecordUsage
func BenchmarkTokenManagerRecordUsage(b *testing.B) {
	logger := zap.NewNop()
	estimator := &SimpleTokenEstimator{}
	tm := NewTokenManager(50000, 100, estimator, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.RecordUsage(100)
	}
}

// BenchmarkTokenManagerGetMetrics benchmarks token manager GetMetrics
func BenchmarkTokenManagerGetMetrics(b *testing.B) {
	logger := zap.NewNop()
	estimator := &SimpleTokenEstimator{}
	tm := NewTokenManager(5000, 100, estimator, logger)

	// Record some usage
	for i := 0; i < 100; i++ {
		tm.RecordUsage(50)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.GetMetrics()
	}
}

// BenchmarkTokenEstimation benchmarks token estimation
func BenchmarkTokenEstimation(b *testing.B) {
	code := `func main() {
		fmt.Println("Hello, World!")
		for i := 0; i < 10; i++ {
			fmt.Printf("Iteration %d\n", i)
		}
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokensForCode(code)
	}
}

// BenchmarkTokenEstimationLargeCode benchmarks token estimation on large code
func BenchmarkTokenEstimationLargeCode(b *testing.B) {
	// Generate large code
	code := ""
	for i := 0; i < 100; i++ {
		code += fmt.Sprintf(`func function%d() {
			fmt.Println("Function %d")
			for j := 0; j < 10; j++ {
				fmt.Printf("Iteration %%d\n", j)
			}
		}
		`, i, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokensForCode(code)
	}
}

// BenchmarkMetricsCollectorRecordSummarization benchmarks metrics recording
func BenchmarkMetricsCollectorRecordSummarization(b *testing.B) {
	logger := zap.NewNop()
	collector := NewMetricsCollector(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordSummarization("llm", 150, 250, false)
	}
}

// BenchmarkMetricsCollectorUpdateCacheStats benchmarks cache stats update
func BenchmarkMetricsCollectorUpdateCacheStats(b *testing.B) {
	logger := zap.NewNop()
	collector := NewMetricsCollector(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.UpdateCacheStats(0.85, 1024)
	}
}

// BenchmarkMetricsCollectorGetMetrics benchmarks metrics retrieval
func BenchmarkMetricsCollectorGetMetrics(b *testing.B) {
	logger := zap.NewNop()
	collector := NewMetricsCollector(logger)

	// Record some metrics
	for i := 0; i < 1000; i++ {
		collector.RecordSummarization("llm", int64(100+i%100), 250, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetMetrics()
	}
}

// BenchmarkMetricsCollectorGetSummaryBreakdown benchmarks summary breakdown
func BenchmarkMetricsCollectorGetSummaryBreakdown(b *testing.B) {
	logger := zap.NewNop()
	collector := NewMetricsCollector(logger)

	// Record various types of summaries
	for i := 0; i < 1000; i++ {
		summaryType := "llm"
		if i%3 == 0 {
			summaryType = "heuristic"
		} else if i%3 == 1 {
			summaryType = "cached"
		}
		collector.RecordSummarization(summaryType, 150, 250, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetSummaryBreakdown()
	}
}

// BenchmarkMetricsCollectorPrometheusMetrics benchmarks Prometheus export
func BenchmarkMetricsCollectorPrometheusMetrics(b *testing.B) {
	logger := zap.NewNop()
	collector := NewMetricsCollector(logger)

	// Record metrics with latencies
	for i := 0; i < 1000; i++ {
		collector.RecordSummarization("llm", int64(100+i%200), 250, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.PrometheusMetrics()
	}
}

// BenchmarkCacheKeyGeneration benchmarks cache key generation
func BenchmarkCacheKeyGeneration(b *testing.B) {
	code := `func main() {
		fmt.Println("Hello, World!")
	}`

	metadata := CodeMetadata{
		FilePath:  "main.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "main",
		LineStart: 1,
		LineEnd:   3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateCacheKey(code, metadata)
	}
}

// BenchmarkIntegrationCacheAndTokenManager benchmarks cache + token manager
func BenchmarkIntegrationCacheAndTokenManager(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(1000, 24*time.Hour, logger)
	defer cache.Clear()

	estimator := &SimpleTokenEstimator{}
	tm := NewTokenManager(50000, 100, estimator, logger)

	code := `func main() {
		fmt.Println("Hello, World!")
	}`

	metadata := CodeMetadata{
		FilePath:  "main.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "main",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := GenerateCacheKey(code, metadata)

		// Try cache first
		if _, ok := cache.Get(key); !ok {
			// Check token budget
			if tm.CanSummarize(code) {
				summary := &CodeSummary{
					Text:       "Summary",
					Type:       "llm",
					TokenCount: 100,
				}
				cache.Set(key, summary)
				tm.RecordUsage(100)
			}
		}
	}
}

// BenchmarkIntegrationFullPipeline benchmarks full summarization pipeline
func BenchmarkIntegrationFullPipeline(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(1000, 24*time.Hour, logger)
	defer cache.Clear()

	estimator := &SimpleTokenEstimator{}
	tm := NewTokenManager(50000, 100, estimator, logger)
	collector := NewMetricsCollector(logger)

	code := `func main() {
		fmt.Println("Hello, World!")
	}`

	metadata := CodeMetadata{
		FilePath:  "main.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "main",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		startTime := time.Now()
		key := GenerateCacheKey(code, metadata)

		// Try cache first
		if summary, ok := cache.Get(key); ok {
			latencyMs := time.Since(startTime).Milliseconds()
			collector.RecordSummarization("cached", latencyMs, summary.TokenCount, true)
		} else if tm.CanSummarize(code) {
			// Simulate LLM call
			summary := &CodeSummary{
				Text:       "Summary",
				Type:       "llm",
				TokenCount: 100,
			}
			cache.Set(key, summary)
			tm.RecordUsage(100)

			latencyMs := time.Since(startTime).Milliseconds()
			collector.RecordSummarization("llm", latencyMs, summary.TokenCount, false)
		}
	}
}

// BenchmarkConcurrentCacheAccess benchmarks concurrent cache access
func BenchmarkConcurrentCacheAccess(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(1000, 24*time.Hour, logger)
	defer cache.Clear()

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key_%d", i)
		summary := &CodeSummary{
			Text:       fmt.Sprintf("Summary %d", i),
			Type:       "cached",
			TokenCount: 100,
		}
		cache.Set(key, summary)
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i%100)
			cache.Get(key)
			i++
		}
	})
}

// BenchmarkConcurrentTokenManagerAccess benchmarks concurrent token manager access
func BenchmarkConcurrentTokenManagerAccess(b *testing.B) {
	logger := zap.NewNop()
	estimator := &SimpleTokenEstimator{}
	tm := NewTokenManager(50000, 100, estimator, logger)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tm.RecordUsage(50)
		}
	})
}

// BenchmarkConcurrentMetricsCollection benchmarks concurrent metrics collection
func BenchmarkConcurrentMetricsCollection(b *testing.B) {
	logger := zap.NewNop()
	collector := NewMetricsCollector(logger)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			summaryType := "llm"
			if i%3 == 0 {
				summaryType = "heuristic"
			} else if i%3 == 1 {
				summaryType = "cached"
			}
			collector.RecordSummarization(summaryType, 150, 250, false)
			i++
		}
	})
}

// TestBenchmarkResults runs benchmarks and prints results
func TestBenchmarkResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark results in short mode")
	}

	fmt.Println("\n=== Cache Benchmarks ===")
	fmt.Println("BenchmarkCacheGet: Get operations from cache")
	fmt.Println("BenchmarkCacheSet: Set operations to cache")
	fmt.Println("BenchmarkCacheGetSet: Mixed Get/Set operations (70/30)")
	fmt.Println("BenchmarkCacheStats: Cache statistics retrieval")
	fmt.Println("BenchmarkCacheEviction: Cache eviction performance")

	fmt.Println("\n=== Token Manager Benchmarks ===")
	fmt.Println("BenchmarkTokenManagerCanSummarize: Check if summarization is possible")
	fmt.Println("BenchmarkTokenManagerRecordUsage: Record token usage")
	fmt.Println("BenchmarkTokenManagerGetMetrics: Retrieve token metrics")

	fmt.Println("\n=== Token Estimation Benchmarks ===")
	fmt.Println("BenchmarkTokenEstimation: Estimate tokens for small code")
	fmt.Println("BenchmarkTokenEstimationLargeCode: Estimate tokens for large code")

	fmt.Println("\n=== Metrics Benchmarks ===")
	fmt.Println("BenchmarkMetricsCollectorRecordSummarization: Record summarization event")
	fmt.Println("BenchmarkMetricsCollectorUpdateCacheStats: Update cache statistics")
	fmt.Println("BenchmarkMetricsCollectorGetMetrics: Retrieve metrics")
	fmt.Println("BenchmarkMetricsCollectorGetSummaryBreakdown: Get summary breakdown")
	fmt.Println("BenchmarkMetricsCollectorPrometheusMetrics: Export Prometheus metrics")

	fmt.Println("\n=== Integration Benchmarks ===")
	fmt.Println("BenchmarkCacheKeyGeneration: Generate cache keys")
	fmt.Println("BenchmarkIntegrationCacheAndTokenManager: Cache + Token Manager")
	fmt.Println("BenchmarkIntegrationFullPipeline: Full summarization pipeline")

	fmt.Println("\n=== Concurrent Benchmarks ===")
	fmt.Println("BenchmarkConcurrentCacheAccess: Concurrent cache access")
	fmt.Println("BenchmarkConcurrentTokenManagerAccess: Concurrent token manager access")
	fmt.Println("BenchmarkConcurrentMetricsCollection: Concurrent metrics collection")

	fmt.Println("\nRun with: go test -bench=. -benchmem ./hyper/internal/mcp/summarizer")
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	logger := zap.NewNop()
	cache := NewLRUCache(10000, 24*time.Hour, logger)
	defer cache.Clear()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		summary := &CodeSummary{
			Text:       fmt.Sprintf("Summary %d with some content", i),
			Type:       "cached",
			TokenCount: 100,
		}
		cache.Set(key, summary)
	}
}

// BenchmarkLatencyDistribution benchmarks latency distribution
func BenchmarkLatencyDistribution(b *testing.B) {
	logger := zap.NewNop()
	collector := NewMetricsCollector(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate varying latencies
		latency := int64(50 + (i % 200))
		collector.RecordSummarization("llm", latency, 250, false)
	}

	metrics := collector.GetMetrics()
	fmt.Printf("\nLatency Distribution:\n")
	fmt.Printf("Average: %.2f ms\n", metrics.AverageLatency)
	fmt.Printf("P95: %d ms\n", metrics.P95Latency)
	fmt.Printf("P99: %d ms\n", metrics.P99Latency)
}
