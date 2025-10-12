package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// BenchmarkHighLoad tests the bridge's performance under high concurrent load
// Simulates the UI polling behavior (2 requests every 3 seconds per client)
func BenchmarkHighLoad(b *testing.B) {
	bridge, router := setupTestBridge(b)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/mcp/tools", nil)
		req.Header.Set("X-Request-ID", generateRequestID())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Verify no pending requests leaked
	bridge.pendingReqsMu.RLock()
	pendingCount := len(bridge.pendingReqs)
	bridge.pendingReqsMu.RUnlock()

	if pendingCount > 0 {
		b.Errorf("Memory leak detected: %d pending requests not cleaned up", pendingCount)
	}
}

// BenchmarkConcurrentToolCalls benchmarks concurrent tool execution
func BenchmarkConcurrentToolCalls(b *testing.B) {
	bridge, router := setupTestBridge(b)

	body := bytes.NewBufferString(`{"name":"test_tool","arguments":{}}`)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/mcp/tools/call", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", generateRequestID())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	bridge.pendingReqsMu.RLock()
	pendingCount := len(bridge.pendingReqs)
	bridge.pendingReqsMu.RUnlock()

	if pendingCount > 0 {
		b.Errorf("Memory leak detected: %d pending requests not cleaned up", pendingCount)
	}
}

// BenchmarkUIPollingSimulation simulates real UI behavior
// Multiple clients polling every 3 seconds with 2 concurrent requests each
func BenchmarkUIPollingSimulation(b *testing.B) {
	bridge, router := setupTestBridge(b)

	const numClients = 5
	const pollInterval = 100 * time.Millisecond // Faster for benchmark

	b.ResetTimer()

	done := make(chan bool)
	var wg sync.WaitGroup

	// Start client goroutines
	for c := 0; c < numClients; c++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			ticker := time.NewTicker(pollInterval)
			defer ticker.Stop()

			requestCount := 0
			for {
				select {
				case <-ticker.C:
					// Make 2 concurrent requests per poll (like UI does)
					var clientWg sync.WaitGroup
					for i := 0; i < 2; i++ {
						clientWg.Add(1)
						go func() {
							defer clientWg.Done()
							req, _ := http.NewRequest("GET", "/api/mcp/tools", nil)
							req.Header.Set("X-Request-ID", generateRequestID())
							w := httptest.NewRecorder()
							router.ServeHTTP(w, req)
						}()
					}
					clientWg.Wait()
					requestCount++

					// Run for N iterations
					if requestCount >= b.N/numClients {
						return
					}
				case <-done:
					return
				}
			}
		}(c)
	}

	wg.Wait()
	close(done)

	// Verify no memory leaks
	bridge.pendingReqsMu.RLock()
	pendingCount := len(bridge.pendingReqs)
	bridge.pendingReqsMu.RUnlock()

	if pendingCount > 0 {
		b.Errorf("Memory leak detected: %d pending requests not cleaned up", pendingCount)
	}
}

// BenchmarkResourceRead benchmarks resource reading performance
func BenchmarkResourceRead(b *testing.B) {
	_, router := setupTestBridge(b)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/mcp/resources/read?uri=test://resource", nil)
		req.Header.Set("X-Request-ID", generateRequestID())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkStressTest applies maximum concurrent load
func BenchmarkStressTest(b *testing.B) {
	bridge, router := setupTestBridge(b)

	const concurrency = 50

	b.ResetTimer()

	var wg sync.WaitGroup
	requestsPerGoroutine := b.N / concurrency

	for g := 0; g < concurrency; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < requestsPerGoroutine; i++ {
				req, _ := http.NewRequest("GET", "/api/mcp/tools", nil)
				req.Header.Set("X-Request-ID", generateRequestID())
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
		}()
	}

	wg.Wait()

	// Verify cleanup
	bridge.pendingReqsMu.RLock()
	pendingCount := len(bridge.pendingReqs)
	bridge.pendingReqsMu.RUnlock()

	if pendingCount > 0 {
		b.Errorf("Memory leak detected: %d pending requests not cleaned up", pendingCount)
	}
}

// BenchmarkRequestIDGeneration benchmarks the request ID generator
func BenchmarkRequestIDGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateRequestID()
	}
}

// BenchmarkParallelToolCalls benchmarks truly parallel tool execution
func BenchmarkParallelToolCalls(b *testing.B) {
	_, router := setupTestBridge(b)

	body := bytes.NewBufferString(`{"name":"test_tool","arguments":{}}`)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("POST", "/api/mcp/tools/call", body)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Request-ID", generateRequestID())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// Benchmark results documentation:
//
// Expected performance targets:
// - BenchmarkHighLoad: >1000 ops/sec on modest hardware
// - BenchmarkConcurrentToolCalls: >500 ops/sec
// - BenchmarkUIPollingSimulation: Should sustain 5+ clients polling every 3s
// - BenchmarkStressTest: Should handle 50 concurrent clients without errors
//
// Memory leak indicators:
// - Growing pendingReqs map after benchmark completion
// - Increasing goroutine count
// - Memory allocations growing unbounded
//
// Performance regression indicators:
// - >50% increase in ns/op
// - Failures under concurrent load
// - Timeout errors appearing in tests