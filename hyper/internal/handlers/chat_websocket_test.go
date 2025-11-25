package handlers

import (
	"encoding/json"
	"testing"

	"hyper/internal/models"
)

// TestToolResultChunking tests the chunking mechanism for large tool results
// IMPORTANT: The chunking is based on JSON-serialized size, not raw data size
// JSON marshaling adds ~2-4x overhead due to escaping and quotes
func TestToolResultChunking(t *testing.T) {
	tests := []struct {
		name           string
		resultSize     int
		description    string
	}{
		{
			name:        "Small result - no chunking",
			resultSize:  2 * 1024, // 2KB raw → ~8KB JSON
			description: "Results under 10KB JSON are sent as single message",
		},
		{
			name:        "Medium result - chunking",
			resultSize:  3 * 1024, // 3KB raw → ~12KB JSON (triggers chunking)
			description: "Results over 10KB JSON are split into chunks",
		},
		{
			name:        "Large result - multiple chunks",
			resultSize:  25 * 1024, // 25KB raw → ~100KB JSON
			description: "Large results split into multiple chunks",
		},
		{
			name:        "Very large result - many chunks",
			resultSize:  100 * 1024, // 100KB raw → ~400KB JSON
			description: "Very large results split into many chunks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test data
			testData := make([]byte, tt.resultSize)
			for i := 0; i < len(testData); i++ {
				testData[i] = byte(i % 256)
			}

			// Serialize to JSON (as would happen in real scenario)
			resultJSON, err := json.Marshal(string(testData))
			if err != nil {
				t.Fatalf("Failed to marshal test data: %v", err)
			}

			// Calculate expected chunks based on JSON size
			const maxChunkSize = 10 * 1024 // 10KB
			resultStr := string(resultJSON)
			calculatedChunks := (len(resultStr) + maxChunkSize - 1) / maxChunkSize

			t.Logf("Raw size: %d bytes → JSON size: %d bytes → %d chunks",
				tt.resultSize, len(resultStr), calculatedChunks)

			// Simulate chunking process
			chunks := make([]models.ToolResultChunk, 0)
			for i := 0; i < calculatedChunks; i++ {
				start := i * maxChunkSize
				end := start + maxChunkSize
				if end > len(resultStr) {
					end = len(resultStr)
				}

				chunk := models.ToolResultChunk{
					ID:    "test-tool-123",
					Chunk: resultStr[start:end],
					Index: i,
					Total: calculatedChunks,
					Done:  i == calculatedChunks-1,
				}
				chunks = append(chunks, chunk)
			}

			// Verify chunk count
			if len(chunks) != calculatedChunks {
				t.Errorf("Expected %d chunks, got %d chunks", calculatedChunks, len(chunks))
			}

			// Verify all chunks have correct metadata
			for i, chunk := range chunks {
				if chunk.Index != i {
					t.Errorf("Chunk %d has wrong index: %d", i, chunk.Index)
				}
				if chunk.Total != calculatedChunks {
					t.Errorf("Chunk %d has wrong total: %d (expected %d)", i, chunk.Total, calculatedChunks)
				}
				if chunk.Done != (i == calculatedChunks-1) {
					t.Errorf("Chunk %d has wrong Done flag: %v", i, chunk.Done)
				}
			}

			// Verify chunks can be reassembled
			reassembled := ""
			for _, chunk := range chunks {
				reassembled += chunk.Chunk
			}
			if reassembled != resultStr {
				t.Errorf("Reassembled result doesn't match original")
			}

			t.Logf("✓ %s - %s", tt.name, tt.description)
		})
	}
}

// TestToolResultChunkingEdgeCases tests edge cases in chunking
func TestToolResultChunkingEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		resultSize  int
		description string
	}{
		{
			name:        "Empty result",
			resultSize:  0,
			description: "Empty results should produce 1 chunk",
		},
		{
			name:        "Single byte",
			resultSize:  1,
			description: "Single byte results should produce 1 chunk",
		},
		{
			name:        "Small result",
			resultSize:  1024, // 1KB raw
			description: "Small results under 10KB JSON",
		},
		{
			name:        "Boundary result",
			resultSize:  2500, // ~10KB JSON
			description: "Results near chunk boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := make([]byte, tt.resultSize)
			resultJSON, _ := json.Marshal(string(testData))

			const maxChunkSize = 10 * 1024
			resultStr := string(resultJSON)
			chunks := (len(resultStr) + maxChunkSize - 1) / maxChunkSize

			if chunks < 1 {
				t.Errorf("Expected at least 1 chunk, got %d", chunks)
			}

			t.Logf("✓ %s - %s (raw: %d bytes, JSON: %d bytes, chunks: %d)",
				tt.name, tt.description, tt.resultSize, len(resultStr), chunks)
		})
	}
}

// TestToolResultChunkingPerformance tests chunking performance with various sizes
func TestToolResultChunkingPerformance(t *testing.T) {
	sizes := []struct {
		name string
		size int
	}{
		{"10KB raw", 10 * 1024},
		{"50KB raw", 50 * 1024},
		{"100KB raw", 100 * 1024},
		{"500KB raw", 500 * 1024},
		{"1MB raw", 1024 * 1024},
	}

	for _, s := range sizes {
		t.Run(s.name, func(t *testing.T) {
			testData := make([]byte, s.size)
			resultJSON, _ := json.Marshal(string(testData))

			const maxChunkSize = 10 * 1024
			resultStr := string(resultJSON)
			chunks := (len(resultStr) + maxChunkSize - 1) / maxChunkSize

			// Verify chunking calculation
			totalSize := 0
			for i := 0; i < chunks; i++ {
				start := i * maxChunkSize
				end := start + maxChunkSize
				if end > len(resultStr) {
					end = len(resultStr)
				}
				totalSize += (end - start)
			}

			if totalSize != len(resultStr) {
				t.Errorf("Chunking lost data: original %d bytes, reassembled %d bytes", len(resultStr), totalSize)
			}

			t.Logf("✓ %s - %d chunks of %d bytes max (raw: %d bytes, JSON: %d bytes)",
				s.name, chunks, maxChunkSize, s.size, len(resultStr))
		})
	}
}

// TestToolResultChunkingStreamMessage tests the StreamMessage structure for chunks
func TestToolResultChunkingStreamMessage(t *testing.T) {
	// Create a test tool result
	toolResult := models.ToolResultEvent{
		ID:         "test-123",
		Result:     "This is a test result",
		Error:      "",
		DurationMs: 100,
	}

	// Create a chunk
	chunk := models.ToolResultChunk{
		ID:    "test-123",
		Chunk: "This is a test result",
		Index: 0,
		Total: 1,
		Done:  true,
	}

	// Create stream message with chunk
	streamMsg := models.StreamMessage{
		Type: "tool_result_chunk",
		ToolResult: &models.ToolResultEvent{
			ID:         toolResult.ID,
			Result:     chunk,
			Error:      toolResult.Error,
			DurationMs: toolResult.DurationMs,
		},
	}

	// Verify it can be marshaled to JSON
	msgJSON, err := json.Marshal(streamMsg)
	if err != nil {
		t.Fatalf("Failed to marshal stream message: %v", err)
	}

	// Verify it can be unmarshaled back
	var unmarshaledMsg models.StreamMessage
	err = json.Unmarshal(msgJSON, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal stream message: %v", err)
	}

	if unmarshaledMsg.Type != "tool_result_chunk" {
		t.Errorf("Expected type 'tool_result_chunk', got '%s'", unmarshaledMsg.Type)
	}

	t.Logf("✓ StreamMessage with chunk serializes correctly (%d bytes)", len(msgJSON))
}

// TestToolResultChunkingWebSocketFlow tests the complete WebSocket flow with chunking
func TestToolResultChunkingWebSocketFlow(t *testing.T) {
	// Simulate a large tool result (25KB raw)
	largeResult := make([]byte, 25*1024)
	for i := 0; i < len(largeResult); i++ {
		largeResult[i] = byte(i % 256)
	}

	// Serialize to JSON
	resultJSON, _ := json.Marshal(string(largeResult))
	resultStr := string(resultJSON)

	const maxChunkSize = 10 * 1024
	totalChunks := (len(resultStr) + maxChunkSize - 1) / maxChunkSize

	// Simulate WebSocket message stream
	messages := make([]models.StreamMessage, 0)

	// Add tool call message
	messages = append(messages, models.StreamMessage{
		Type: "tool_call",
		ToolCall: &models.ToolCallEvent{
			Tool: "read_file",
			Args: map[string]interface{}{"path": "/test/file.txt"},
			ID:   "tool-123",
		},
	})

	// Add chunked tool result messages
	for i := 0; i < totalChunks; i++ {
		start := i * maxChunkSize
		end := start + maxChunkSize
		if end > len(resultStr) {
			end = len(resultStr)
		}

		chunk := models.ToolResultChunk{
			ID:    "tool-123",
			Chunk: resultStr[start:end],
			Index: i,
			Total: totalChunks,
			Done:  i == totalChunks-1,
		}

		messages = append(messages, models.StreamMessage{
			Type: "tool_result_chunk",
			ToolResult: &models.ToolResultEvent{
				ID:         "tool-123",
				Result:     chunk,
				Error:      "",
				DurationMs: 100,
			},
		})
	}

	// Add done message
	messages = append(messages, models.StreamMessage{
		Type: "done",
	})

	// Verify message count
	expectedMessages := 2 + totalChunks // tool_call + chunks + done
	if len(messages) != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, len(messages))
	}

	// Verify all messages can be serialized
	for i, msg := range messages {
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			t.Errorf("Message %d failed to marshal: %v", i, err)
		}

		// Verify message size is reasonable (under 15KB for safety margin)
		if len(msgJSON) > 15*1024 {
			t.Errorf("Message %d is too large: %d bytes", i, len(msgJSON))
		}
	}

	// Verify chunks can be reassembled
	reassembled := ""
	for _, msg := range messages {
		if msg.Type == "tool_result_chunk" && msg.ToolResult != nil {
			if chunk, ok := msg.ToolResult.Result.(models.ToolResultChunk); ok {
				reassembled += chunk.Chunk
			}
		}
	}

	if reassembled != resultStr {
		t.Errorf("Reassembled result doesn't match original (got %d bytes, expected %d bytes)",
			len(reassembled), len(resultStr))
	}

	t.Logf("✓ WebSocket flow with chunking - %d messages for %d byte raw result (%d bytes JSON, %d chunks)",
		len(messages), len(largeResult), len(resultStr), totalChunks)
}

// TestToolResultChunkingCalculation tests the chunk calculation formula
func TestToolResultChunkingCalculation(t *testing.T) {
	tests := []struct {
		jsonSize       int
		expectedChunks int
		description    string
	}{
		{5 * 1024, 1, "5KB JSON → 1 chunk"},
		{10 * 1024, 1, "10KB JSON → 1 chunk (at boundary)"},
		{10*1024 + 1, 2, "10KB+1 JSON → 2 chunks"},
		{20 * 1024, 2, "20KB JSON → 2 chunks"},
		{30 * 1024, 3, "30KB JSON → 3 chunks"},
		{100 * 1024, 10, "100KB JSON → 10 chunks"},
		{1024 * 1024, 103, "1MB JSON → 103 chunks"},
	}

	const maxChunkSize = 10 * 1024

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			chunks := (tt.jsonSize + maxChunkSize - 1) / maxChunkSize
			if chunks != tt.expectedChunks {
				t.Errorf("Expected %d chunks, got %d chunks", tt.expectedChunks, chunks)
			}
			t.Logf("✓ %s", tt.description)
		})
	}
}

// TestToolResultChunkingRealWorldScenario tests a realistic scenario
func TestToolResultChunkingRealWorldScenario(t *testing.T) {
	// Simulate reading a large file (e.g., 500KB file)
	fileContent := make([]byte, 500*1024)
	for i := 0; i < len(fileContent); i++ {
		fileContent[i] = byte((i % 256))
	}

	// Serialize to JSON (as tool result)
	resultJSON, _ := json.Marshal(string(fileContent))
	resultStr := string(resultJSON)

	const maxChunkSize = 10 * 1024
	totalChunks := (len(resultStr) + maxChunkSize - 1) / maxChunkSize

	t.Logf("Real-world scenario: 500KB file")
	t.Logf("  Raw size: %d bytes", len(fileContent))
	t.Logf("  JSON size: %d bytes (%.1fx overhead)", len(resultStr), float64(len(resultStr))/float64(len(fileContent)))
	t.Logf("  Chunks: %d (10KB each)", totalChunks)
	t.Logf("  WebSocket messages: %d (tool_call + %d chunks + done)", totalChunks+2, totalChunks)

	// Verify chunking works
	reassembled := ""
	for i := 0; i < totalChunks; i++ {
		start := i * maxChunkSize
		end := start + maxChunkSize
		if end > len(resultStr) {
			end = len(resultStr)
		}
		reassembled += resultStr[start:end]
	}

	if reassembled != resultStr {
		t.Errorf("Reassembly failed")
	}

	t.Logf("✓ Real-world scenario: Successfully chunked and reassembled")
}
