package config

import (
	"os"
	"testing"
)

func TestDefaultProviderCapabilities(t *testing.T) {
	caps := DefaultProviderCapabilities()

	// Verify default values
	if caps.MaxContextTokens != 100000 {
		t.Errorf("MaxContextTokens: got %d, want 100000", caps.MaxContextTokens)
	}

	if caps.CharsPerToken != 3.5 {
		t.Errorf("CharsPerToken: got %f, want 3.5", caps.CharsPerToken)
	}

	if caps.ReservedForResponse != 8000 {
		t.Errorf("ReservedForResponse: got %d, want 8000", caps.ReservedForResponse)
	}

	if caps.SafeToolResultBytes != 50*1024 {
		t.Errorf("SafeToolResultBytes: got %d, want %d", caps.SafeToolResultBytes, 50*1024)
	}
}

func TestGetProviderCapabilities_Anthropic(t *testing.T) {
	caps := GetProviderCapabilities("anthropic", "claude-3-opus")

	if caps.MaxContextTokens != 200000 {
		t.Errorf("Anthropic MaxContextTokens: got %d, want 200000", caps.MaxContextTokens)
	}

	if caps.CharsPerToken != 3.5 {
		t.Errorf("Anthropic CharsPerToken: got %f, want 3.5", caps.CharsPerToken)
	}
}

func TestGetProviderCapabilities_OpenAI(t *testing.T) {
	caps := GetProviderCapabilities("openai", "gpt-4")

	// Provider sets 128k, but model override sets 8192
	if caps.MaxContextTokens != 8192 {
		t.Errorf("OpenAI GPT-4 MaxContextTokens: got %d, want 8192", caps.MaxContextTokens)
	}

	if caps.CharsPerToken != 4.0 {
		t.Errorf("OpenAI CharsPerToken: got %f, want 4.0", caps.CharsPerToken)
	}
}

func TestGetProviderCapabilities_OpenAI_GPT4Turbo(t *testing.T) {
	caps := GetProviderCapabilities("openai", "gpt-4-turbo")

	if caps.MaxContextTokens != 128000 {
		t.Errorf("OpenAI GPT-4-Turbo MaxContextTokens: got %d, want 128000", caps.MaxContextTokens)
	}
}

func TestGetProviderCapabilities_Google(t *testing.T) {
	caps := GetProviderCapabilities("google", "gemini-pro")

	if caps.MaxContextTokens != 32000 {
		t.Errorf("Google Gemini Pro MaxContextTokens: got %d, want 32000", caps.MaxContextTokens)
	}
}

func TestGetProviderCapabilities_Llama(t *testing.T) {
	caps := GetProviderCapabilities("meta", "llama-2-70b")

	if caps.MaxContextTokens != 4096 {
		t.Errorf("Llama 2 70b MaxContextTokens: got %d, want 4096", caps.MaxContextTokens)
	}
}

func TestGetProviderCapabilities_EnvOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("MAX_CONTEXT_TOKENS", "150000")
	os.Setenv("SAFE_TOOL_RESULT_BYTES", "100000")
	defer os.Unsetenv("MAX_CONTEXT_TOKENS")
	defer os.Unsetenv("SAFE_TOOL_RESULT_BYTES")

	caps := GetProviderCapabilities("anthropic", "claude-3-opus")

	// Environment should override provider defaults
	if caps.MaxContextTokens != 150000 {
		t.Errorf("Env override MaxContextTokens: got %d, want 150000", caps.MaxContextTokens)
	}

	if caps.SafeToolResultBytes != 100000 {
		t.Errorf("Env override SafeToolResultBytes: got %d, want 100000", caps.SafeToolResultBytes)
	}
}

func TestGetProviderCapabilities_InvalidEnv(t *testing.T) {
	// Set invalid environment variables
	os.Setenv("MAX_CONTEXT_TOKENS", "invalid")
	os.Setenv("SAFE_TOOL_RESULT_BYTES", "-100")
	defer os.Unsetenv("MAX_CONTEXT_TOKENS")
	defer os.Unsetenv("SAFE_TOOL_RESULT_BYTES")

	caps := GetProviderCapabilities("anthropic", "claude-3-opus")

	// Should fall back to provider defaults
	if caps.MaxContextTokens != 200000 {
		t.Errorf("Invalid env should use provider default: got %d, want 200000", caps.MaxContextTokens)
	}

	// Negative value should be ignored
	if caps.SafeToolResultBytes != 50*1024 {
		t.Errorf("Negative env should use default: got %d, want %d", caps.SafeToolResultBytes, 50*1024)
	}
}

func TestEstimateTokens(t *testing.T) {
	caps := DefaultProviderCapabilities() // CharsPerToken = 3.5

	tests := []struct {
		bytes    int
		expected int
		name     string
	}{
		{350, 100, "350 bytes / 3.5 = 100 tokens"},
		{700, 200, "700 bytes / 3.5 = 200 tokens"},
		{35, 10, "35 bytes / 3.5 = 10 tokens"},
		{0, 0, "0 bytes = 0 tokens"},
	}

	for _, tt := range tests {
		result := caps.EstimateTokens(tt.bytes)
		if result != tt.expected {
			t.Errorf("%s: got %d, want %d", tt.name, result, tt.expected)
		}
	}
}

func TestEstimateTokensFromString(t *testing.T) {
	caps := DefaultProviderCapabilities()

	// "Hello, World!" is 13 characters / 3.5 = 3 tokens (truncated)
	result := caps.EstimateTokensFromString("Hello, World!")
	if result != 3 {
		t.Errorf("EstimateTokensFromString: got %d, want 3", result)
	}
}

func TestMaxSafeResultTokens(t *testing.T) {
	caps := DefaultProviderCapabilities()

	// 50KB / 3.5 chars per token = ~14628 tokens
	safeBytes := float64(50 * 1024)
	expected := int(safeBytes / 3.5)
	result := caps.MaxSafeResultTokens()

	if result != expected {
		t.Errorf("MaxSafeResultTokens: got %d, want %d", result, expected)
	}
}

func TestCalculateRemainingContext(t *testing.T) {
	caps := DefaultProviderCapabilities() // 100k context, 8k reserved

	tests := []struct {
		systemBytes  int
		messageBytes int
		expected     int
		name         string
	}{
		{
			systemBytes:  3500,  // ~1000 tokens
			messageBytes: 35000, // ~10000 tokens
			// 100000 - 1000 - 10000 - 8000 = 81000
			expected: 81000,
			name:     "Normal conversation",
		},
		{
			systemBytes:  3500,   // ~1000 tokens
			messageBytes: 315000, // ~90000 tokens
			// 100000 - 1000 - 90000 - 8000 = 1000
			expected: 1000,
			name:     "Near capacity",
		},
		{
			systemBytes:  3500,   // ~1000 tokens
			messageBytes: 350000, // ~100000 tokens
			// Would be negative, should return 0
			expected: 0,
			name:     "Over capacity",
		},
	}

	for _, tt := range tests {
		result := caps.CalculateRemainingContext(tt.systemBytes, tt.messageBytes)
		if result != tt.expected {
			t.Errorf("%s: got %d, want %d", tt.name, result, tt.expected)
		}
	}
}

func TestIsResultTooLarge(t *testing.T) {
	caps := DefaultProviderCapabilities() // 50KB limit

	tests := []struct {
		bytes    int
		expected bool
		name     string
	}{
		{10 * 1024, false, "10KB under limit"},
		{50 * 1024, false, "50KB at limit"},
		{50*1024 + 1, true, "50KB+1 over limit"},
		{100 * 1024, true, "100KB over limit"},
	}

	for _, tt := range tests {
		result := caps.IsResultTooLarge(tt.bytes)
		if result != tt.expected {
			t.Errorf("%s: got %v, want %v", tt.name, result, tt.expected)
		}
	}
}

func TestProviderCapabilities_ProviderAgnostic(t *testing.T) {
	// Verify defaults work for any provider
	providers := []struct {
		name  string
		model string
	}{
		{"anthropic", "claude-3-opus"},
		{"openai", "gpt-4-turbo"},
		{"google", "gemini-pro"},
		{"meta", "llama-2-70b"},
		{"unknown", "unknown-model"},
	}

	for _, p := range providers {
		caps := GetProviderCapabilities(p.name, p.model)

		// All providers should have valid positive values
		if caps.MaxContextTokens <= 0 {
			t.Errorf("%s/%s: MaxContextTokens should be positive, got %d", p.name, p.model, caps.MaxContextTokens)
		}

		if caps.CharsPerToken <= 0 {
			t.Errorf("%s/%s: CharsPerToken should be positive, got %f", p.name, p.model, caps.CharsPerToken)
		}

		if caps.SafeToolResultBytes <= 0 {
			t.Errorf("%s/%s: SafeToolResultBytes should be positive, got %d", p.name, p.model, caps.SafeToolResultBytes)
		}
	}
}

func TestGetProviderCapabilities_ModelOverridesCoverage(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		wantMax  int
	}{
		{name: "gpt4 turbo preview", provider: "openai", model: "gpt-4-turbo-preview", wantMax: 128000},
		{name: "gpt4 32k", provider: "openai", model: "gpt-4-32k", wantMax: 32768},
		{name: "gpt3.5 turbo", provider: "openai", model: "gpt-3.5-turbo", wantMax: 16385},
		{name: "gpt3.5 turbo 16k", provider: "openai", model: "gpt-3.5-turbo-16k", wantMax: 16385},
		{name: "claude 3 sonnet", provider: "anthropic", model: "claude-3-sonnet", wantMax: 200000},
		{name: "claude 3 haiku", provider: "anthropic", model: "claude-3-haiku", wantMax: 200000},
		{name: "claude 2", provider: "anthropic", model: "claude-2", wantMax: 100000},
		{name: "claude 2.1", provider: "anthropic", model: "claude-2.1", wantMax: 100000},
		{name: "gemini 1.5 pro", provider: "google", model: "gemini-1.5-pro", wantMax: 1000000},
		{name: "llama 3 70b", provider: "llama", model: "llama-3-70b", wantMax: 8192},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			caps := GetProviderCapabilities(tc.provider, tc.model)
			if caps.MaxContextTokens != tc.wantMax {
				t.Fatalf("GetProviderCapabilities(%q, %q).MaxContextTokens = %d, want %d",
					tc.provider, tc.model, caps.MaxContextTokens, tc.wantMax)
			}
		})
	}
}
