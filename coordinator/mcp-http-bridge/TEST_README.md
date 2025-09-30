# MCP HTTP Bridge Test Suite

## Overview

Comprehensive test suite for the MCP HTTP Bridge to prevent regressions, especially around concurrent request handling and the "broken pipe" bug that was fixed.

## Test Coverage

**Current Coverage: 60.3%** (target: >80%)

### Covered Components

- ✅ Concurrent request handling (20+ simultaneous requests)
- ✅ Request/response ID matching with out-of-order responses
- ✅ Timeout handling and resource cleanup
- ✅ Error propagation from MCP server to HTTP clients
- ✅ stdin write locking (no interleaved writes)
- ✅ Pending request map cleanup
- ✅ All HTTP endpoints (tools, resources)
- ✅ Invalid request handling

### Test Files

1. **main_test.go** - Core functionality tests
   - Mock MCP server using bash script (simulates real MCP server behavior)
   - Concurrent request tests
   - Request/response matching verification
   - Resource cleanup tests
   - HTTP endpoint validation

2. **benchmark_test.go** - Performance and load tests
   - High load simulation (>100 req/sec capability)
   - UI polling simulation (2 concurrent requests per client every 3s)
   - Stress testing with 50 concurrent goroutines
   - Memory leak detection

## Running Tests

### Unit Tests

```bash
# Run all tests
go test -v -timeout=120s

# Run specific test
go test -v -timeout=120s -run TestConcurrentRequests

# Run tests with coverage
go test -cover -coverprofile=coverage.out -timeout=120s

# View coverage report
go tool cover -html=coverage.out
```

### Benchmark Tests

```bash
# Run all benchmarks
go test -bench=. -benchtime=10s

# Run specific benchmark
go test -bench=BenchmarkHighLoad -benchtime=10s

# Run with memory profiling
go test -bench=. -benchmem
```

## Test Architecture

### Mock MCP Server

The test suite uses a bash script mock server instead of compiling Go code during tests. This approach:

- ✅ Avoids compilation overhead in tests
- ✅ Works across different Go versions
- ✅ Simulates real MCP protocol behavior
- ✅ Supports all standard MCP methods (initialize, tools/list, tools/call, resources/*, etc.)

### Testing Helper Interface

Uses a `testingHelper` interface to support both `*testing.T` and `*testing.B`, enabling shared test infrastructure between unit tests and benchmarks.

## Critical Test Cases

### 1. TestConcurrentRequests
**Purpose:** Verify no broken pipe errors with 20+ simultaneous requests
**Why:** The original bug occurred when UI polled with 2 concurrent requests every 3 seconds
**Validates:**
- All requests complete successfully
- Responses are received and valid
- No goroutine leaks

### 2. TestRequestResponseMatching
**Purpose:** Verify responses are matched to correct requests even when out of order
**Why:** Background response handler routes responses by ID
**Validates:**
- Correct request ID → response ID mapping
- No response cross-contamination
- All responses received

### 3. TestStdinWriteLocking
**Purpose:** Verify concurrent stdin writes don't corrupt JSON
**Why:** Multiple goroutines writing to stdin simultaneously can interleave JSON
**Validates:**
- Writes are serialized with mutex
- No broken pipe errors
- At least 90% success rate (accounting for bash mock performance)

### 4. TestPendingRequestCleanup
**Purpose:** Verify no memory leaks in pendingReqs map
**Why:** Long-running bridge must clean up completed requests
**Validates:**
- Map returns to initial size after requests
- Channels are closed properly
- No goroutine leaks

### 5. TestTimeoutHandling
**Purpose:** Verify requests timeout appropriately
**Why:** Prevent indefinite blocking on MCP server failures
**Validates:**
- Cleanup happens even on timeout
- Resources are released

### 6. TestErrorPropagation
**Purpose:** Verify MCP errors reach HTTP clients
**Why:** Clients need to see real errors from MCP server
**Validates:**
- Error responses are not masked
- HTTP status codes are appropriate

## Benchmark Tests

### BenchmarkHighLoad
**Purpose:** Measure throughput under sequential load
**Target:** >1000 ops/sec on modest hardware
**Validates:** No performance regressions

### BenchmarkUIPollingSimulation
**Purpose:** Simulate real UI behavior (5 clients, 2 reqs every 3s)
**Target:** Sustain load without errors or timeouts
**Validates:** Production-like workload handling

### BenchmarkStressTest
**Purpose:** Maximum concurrent load (50 goroutines)
**Target:** No crashes, panics, or memory leaks
**Validates:** System stability under extreme load

## Edge Cases Covered

1. **Missing Request ID** - Handled gracefully
2. **Invalid JSON in request** - Returns 400 Bad Request
3. **Missing URI parameter** - Returns 400 Bad Request
4. **Unknown MCP method** - Propagates MCP error to client
5. **Concurrent same-ID requests** - First response wins
6. **Very slow MCP server** - Request times out appropriately
7. **Bash mock limitations** - Tests tolerate 90% success rate

## Known Limitations

### Bash Mock Server Performance
The bash script mock is slower than a real Go MCP server:
- **Real server:** ~1000 req/sec
- **Bash mock:** ~100 req/sec

This is acceptable for testing concurrent logic but means:
- Stress tests use lower request counts
- Success rate thresholds allow for timeouts (90% vs 100%)

### BenchmarkBrokenPipeRecovery (Skipped)
Testing MCP server crash recovery requires advanced subprocess control. Test documents expected behavior but is skipped in CI.

## Future Improvements

1. **Increase coverage to 80%+**
   - Add tests for main() startup logic (integration tests)
   - Test CORS configuration
   - Test health check endpoint

2. **Add integration tests with real MCP server**
   - Use coordinator/mcp-server/hyperion-coordinator-mcp
   - Test against real MongoDB
   - Validate end-to-end flows

3. **Add chaos engineering tests**
   - MCP server crashes mid-request
   - Network delays and timeouts
   - Memory pressure scenarios

4. **Performance regression detection**
   - Automated benchmark comparison
   - Alert on >10% performance degradation

## Regression Prevention

### The Broken Pipe Bug

**Original Issue:**
- UI polling with 2 concurrent requests every 3 seconds
- Race condition in response handling
- Broken pipe errors when responses arrived out of order

**Fix:**
- Added background goroutine `handleResponses()`
- Added `pendingReqs` map with mutex protection
- Request/response matching by ID

**Prevention:**
- TestConcurrentRequests validates 20+ simultaneous requests
- TestRequestResponseMatching validates out-of-order responses
- TestStdinWriteLocking validates no interleaved writes
- BenchmarkUIPollingSimulation validates real UI behavior

### Monitoring Test Health

```bash
# Ensure all tests pass before commits
go test -timeout=120s ./...

# Check coverage hasn't regressed
go test -cover ./... | grep coverage

# Run benchmarks to detect performance regressions
go test -bench=. -benchmem
```

## Continuous Integration

Recommended CI pipeline:

```yaml
- name: Run Unit Tests
  run: go test -v -timeout=120s -cover ./...

- name: Run Benchmarks
  run: go test -bench=. -benchtime=5s

- name: Check Coverage
  run: |
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' | awk '{if ($1 < 60) exit 1}'
```

## Contact

For questions or issues with the test suite, refer to:
- Main implementation: `main.go`
- Original bug fix: Background response handler pattern
- System documentation: `coordinator/SPECIFICATION.md`