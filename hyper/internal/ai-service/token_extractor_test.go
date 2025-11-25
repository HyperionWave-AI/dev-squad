package aiservice

import (
	"testing"
)

func TestExtractOpenAITokens(t *testing.T) {
	extractor := NewTokenUsageExtractor("openai")

	// Mock OpenAI response
	responseBody := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"usage": {
			"prompt_tokens": 100,
			"completion_tokens": 50,
			"total_tokens": 150
		}
	}`)

	inputTokens, outputTokens, err := extractor.ExtractOpenAITokens(responseBody)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if inputTokens != 100 {
		t.Errorf("Expected prompt_tokens=100, got %d", inputTokens)
	}

	if outputTokens != 50 {
		t.Errorf("Expected completion_tokens=50, got %d", outputTokens)
	}
}

func TestExtractClaudeTokens(t *testing.T) {
	extractor := NewTokenUsageExtractor("anthropic")

	// Mock Anthropic/Claude response
	responseBody := []byte(`{
		"id": "msg_123",
		"type": "message",
		"role": "assistant",
		"content": [{"type": "text", "text": "Hello"}],
		"model": "claude-3-opus-20240229",
		"stop_reason": "end_turn",
		"usage": {
			"input_tokens": 75,
			"output_tokens": 25,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens": 0
		}
	}`)

	inputTokens, outputTokens, cacheCreation, cacheRead, err := extractor.ExtractClaudeTokens(responseBody)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if inputTokens != 75 {
		t.Errorf("Expected input_tokens=75, got %d", inputTokens)
	}

	if outputTokens != 25 {
		t.Errorf("Expected output_tokens=25, got %d", outputTokens)
	}

	if cacheCreation != 0 {
		t.Errorf("Expected cache_creation_input_tokens=0, got %d", cacheCreation)
	}

	if cacheRead != 0 {
		t.Errorf("Expected cache_read_input_tokens=0, got %d", cacheRead)
	}
}

func TestExtractGroqTokens(t *testing.T) {
	extractor := NewTokenUsageExtractor("groq")

	// Mock Groq response (same format as OpenAI)
	responseBody := []byte(`{
		"id": "cmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "mixtral-8x7b-32768",
		"usage": {
			"prompt_tokens": 50,
			"completion_tokens": 30,
			"total_tokens": 80
		}
	}`)

	inputTokens, outputTokens, err := extractor.ExtractGroqTokens(responseBody)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if inputTokens != 50 {
		t.Errorf("Expected prompt_tokens=50, got %d", inputTokens)
	}

	if outputTokens != 30 {
		t.Errorf("Expected completion_tokens=30, got %d", outputTokens)
	}
}

func TestExtractTokensByProvider(t *testing.T) {
	tests := []struct {
		name             string
		provider         string
		responseBody     []byte
		expectedInput    int
		expectedOutput   int
		shouldError      bool
	}{
		{
			name:     "OpenAI",
			provider: "openai",
			responseBody: []byte(`{
				"usage": {
					"prompt_tokens": 100,
					"completion_tokens": 50,
					"total_tokens": 150
				}
			}`),
			expectedInput:  100,
			expectedOutput: 50,
			shouldError:    false,
		},
		{
			name:     "Anthropic",
			provider: "anthropic",
			responseBody: []byte(`{
				"usage": {
					"input_tokens": 75,
					"output_tokens": 25,
					"cache_creation_input_tokens": 0,
					"cache_read_input_tokens": 0
				}
			}`),
			expectedInput:  75,
			expectedOutput: 25,
			shouldError:    false,
		},
		{
			name:     "Groq",
			provider: "groq",
			responseBody: []byte(`{
				"usage": {
					"prompt_tokens": 50,
					"completion_tokens": 30,
					"total_tokens": 80
				}
			}`),
			expectedInput:  50,
			expectedOutput: 30,
			shouldError:    false,
		},
		{
			name:             "Unsupported Provider",
			provider:         "unsupported",
			responseBody:     []byte(`{}`),
			expectedInput:    0,
			expectedOutput:   0,
			shouldError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewTokenUsageExtractor(tt.provider)
			inputTokens, outputTokens, err := extractor.ExtractTokensByProvider(tt.responseBody)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error, got nil")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if !tt.shouldError {
				if inputTokens != tt.expectedInput {
					t.Errorf("Expected input_tokens=%d, got %d", tt.expectedInput, inputTokens)
				}

				if outputTokens != tt.expectedOutput {
					t.Errorf("Expected output_tokens=%d, got %d", tt.expectedOutput, outputTokens)
				}
			}
		})
	}
}

func TestTokenUsageLogger(t *testing.T) {
	logger := NewTokenUsageLogger()

	// Log some usage
	usage1 := &TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		Provider:         "openai",
		Model:            "gpt-4",
	}

	usage2 := &TokenUsage{
		PromptTokens:     75,
		CompletionTokens: 25,
		TotalTokens:      100,
		Provider:         "anthropic",
		Model:            "claude-3-opus",
	}

	logger.LogUsage(usage1)
	logger.LogUsage(usage2)

	// Get total usage
	stats := logger.GetTotalUsage()

	// Verify totals
	if stats["total_prompt_tokens"] != 175 {
		t.Errorf("Expected total_prompt_tokens=175, got %v", stats["total_prompt_tokens"])
	}

	if stats["total_completion_tokens"] != 75 {
		t.Errorf("Expected total_completion_tokens=75, got %v", stats["total_completion_tokens"])
	}

	if stats["total_tokens"] != 250 {
		t.Errorf("Expected total_tokens=250, got %v", stats["total_tokens"])
	}

	if stats["total_calls"] != 2 {
		t.Errorf("Expected total_calls=2, got %v", stats["total_calls"])
	}

	// Verify provider stats
	providerStats := stats["provider_stats"].(map[string]map[string]int)

	if providerStats["openai"]["total_tokens"] != 150 {
		t.Errorf("Expected openai total_tokens=150, got %v", providerStats["openai"]["total_tokens"])
	}

	if providerStats["anthropic"]["total_tokens"] != 100 {
		t.Errorf("Expected anthropic total_tokens=100, got %v", providerStats["anthropic"]["total_tokens"])
	}

	// Verify history
	history := logger.GetUsageHistory()
	if len(history) != 2 {
		t.Errorf("Expected history length=2, got %d", len(history))
	}
}

func TestTokenUsageLoggerWithNilUsage(t *testing.T) {
	logger := NewTokenUsageLogger()

	// Log nil usage - should not panic
	logger.LogUsage(nil)

	stats := logger.GetTotalUsage()
	if stats["total_calls"] != 0 {
		t.Errorf("Expected total_calls=0 after logging nil, got %v", stats["total_calls"])
	}
}
