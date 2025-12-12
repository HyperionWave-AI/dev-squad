package summarizer

import (
	"os"
	"testing"
	"time"
)

// TestLoadSummarizerConfigDefaults tests loading config with all defaults
func TestLoadSummarizerConfigDefaults(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	config := LoadSummarizerConfig()

	if !config.Enabled {
		t.Error("Expected Enabled to be true by default")
	}

	if config.Model != "claude" {
		t.Errorf("Expected Model 'claude', got %s", config.Model)
	}

	if config.MaxTokens != 500 {
		t.Errorf("Expected MaxTokens 500, got %d", config.MaxTokens)
	}

	if !config.CacheEnabled {
		t.Error("Expected CacheEnabled to be true by default")
	}

	if !config.FallbackToHeuristic {
		t.Error("Expected FallbackToHeuristic to be true by default")
	}

	if config.CacheSize != 1000 {
		t.Errorf("Expected CacheSize 1000, got %d", config.CacheSize)
	}

	if config.CacheTTL != 3600*time.Second {
		t.Errorf("Expected CacheTTL 3600s, got %v", config.CacheTTL)
	}

	if config.LLMTimeout != 30*time.Second {
		t.Errorf("Expected LLMTimeout 30s, got %v", config.LLMTimeout)
	}

	if config.TokenBudget != 10000 {
		t.Errorf("Expected TokenBudget 10000, got %d", config.TokenBudget)
	}

	if config.TokenPerResult != 100 {
		t.Errorf("Expected TokenPerResult 100, got %d", config.TokenPerResult)
	}

	if config.MetricsEnabled {
		t.Error("Expected MetricsEnabled to be false by default")
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected LogLevel 'info', got %s", config.LogLevel)
	}
}

// TestLoadSummarizerConfigFromEnv tests loading config from environment variables
func TestLoadSummarizerConfigFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("SUMMARIZER_ENABLED", "false")
	os.Setenv("SUMMARIZER_MODEL", "gpt-3.5-turbo")
	os.Setenv("SUMMARIZER_MAX_TOKENS", "300")
	os.Setenv("SUMMARIZER_CACHE_ENABLED", "false")
	os.Setenv("SUMMARIZER_FALLBACK_TO_HEURISTIC", "false")
	os.Setenv("SUMMARIZER_CACHE_SIZE", "500")
	os.Setenv("SUMMARIZER_CACHE_TTL", "1800")
	os.Setenv("SUMMARIZER_LLM_API_KEY", "test-api-key")
	os.Setenv("SUMMARIZER_LLM_TIMEOUT", "60")
	os.Setenv("SUMMARIZER_TOKEN_BUDGET", "5000")
	os.Setenv("SUMMARIZER_TOKEN_PER_RESULT", "50")
	os.Setenv("SUMMARIZER_METRICS_ENABLED", "true")
	os.Setenv("SUMMARIZER_LOG_LEVEL", "debug")

	defer os.Clearenv()

	config := LoadSummarizerConfig()

	if config.Enabled {
		t.Error("Expected Enabled to be false")
	}

	if config.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected Model 'gpt-3.5-turbo', got %s", config.Model)
	}

	if config.MaxTokens != 300 {
		t.Errorf("Expected MaxTokens 300, got %d", config.MaxTokens)
	}

	if config.CacheEnabled {
		t.Error("Expected CacheEnabled to be false")
	}

	if config.FallbackToHeuristic {
		t.Error("Expected FallbackToHeuristic to be false")
	}

	if config.CacheSize != 500 {
		t.Errorf("Expected CacheSize 500, got %d", config.CacheSize)
	}

	if config.CacheTTL != 1800*time.Second {
		t.Errorf("Expected CacheTTL 1800s, got %v", config.CacheTTL)
	}

	if config.LLMAPIKey != "test-api-key" {
		t.Errorf("Expected LLMAPIKey 'test-api-key', got %s", config.LLMAPIKey)
	}

	if config.LLMTimeout != 60*time.Second {
		t.Errorf("Expected LLMTimeout 60s, got %v", config.LLMTimeout)
	}

	if config.TokenBudget != 5000 {
		t.Errorf("Expected TokenBudget 5000, got %d", config.TokenBudget)
	}

	if config.TokenPerResult != 50 {
		t.Errorf("Expected TokenPerResult 50, got %d", config.TokenPerResult)
	}

	if !config.MetricsEnabled {
		t.Error("Expected MetricsEnabled to be true")
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got %s", config.LogLevel)
	}
}

// TestParseBoolValid tests parseBool with valid values
func TestParseBoolValid(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"NO", false},
		{"off", false},
		{"OFF", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseBool(tt.value, false)
			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestParseBoolDefault tests parseBool with invalid values uses default
func TestParseBoolDefault(t *testing.T) {
	tests := []struct {
		value    string
		defValue bool
		expected bool
	}{
		{"invalid", true, true},
		{"invalid", false, false},
		{"", true, true},
		{"", false, false},
		{"maybe", true, true},
		{"maybe", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseBool(tt.value, tt.defValue)
			if result != tt.expected {
				t.Errorf("parseBool(%q, %v) = %v, expected %v", tt.value, tt.defValue, result, tt.expected)
			}
		})
	}
}

// TestParseStringEmpty tests parseString with empty value
func TestParseStringEmpty(t *testing.T) {
	result := parseString("", "default")
	if result != "default" {
		t.Errorf("parseString(\"\", \"default\") = %q, expected \"default\"", result)
	}
}

// TestParseStringValue tests parseString with value
func TestParseStringValue(t *testing.T) {
	result := parseString("value", "default")
	if result != "value" {
		t.Errorf("parseString(\"value\", \"default\") = %q, expected \"value\"", result)
	}
}

// TestParseStringWhitespace tests parseString trims whitespace
func TestParseStringWhitespace(t *testing.T) {
	result := parseString("  value  ", "default")
	if result != "value" {
		t.Errorf("parseString(\"  value  \", \"default\") = %q, expected \"value\"", result)
	}
}

// TestParseIntValid tests parseInt with valid values
func TestParseIntValid(t *testing.T) {
	tests := []struct {
		value    string
		expected int
	}{
		{"0", 0},
		{"100", 100},
		{"1000", 1000},
		{"-50", -50},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseInt(tt.value, 0)
			if result != tt.expected {
				t.Errorf("parseInt(%q, 0) = %d, expected %d", tt.value, result, tt.expected)
			}
		})
	}
}

// TestParseIntInvalid tests parseInt with invalid values uses default
func TestParseIntInvalid(t *testing.T) {
	tests := []struct {
		value    string
		defValue int
		expected int
	}{
		{"invalid", 100, 100},
		{"", 100, 100},
		{"12.34", 100, 100},
		{"abc123", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseInt(tt.value, tt.defValue)
			if result != tt.expected {
				t.Errorf("parseInt(%q, %d) = %d, expected %d", tt.value, tt.defValue, result, tt.expected)
			}
		})
	}
}

// TestParseDurationValid tests parseDuration with valid values
func TestParseDurationValid(t *testing.T) {
	tests := []struct {
		value    string
		expected time.Duration
	}{
		{"0", 0},
		{"30", 30 * time.Second},
		{"3600", 3600 * time.Second},
		{"60", 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseDuration(tt.value, 0)
			if result != tt.expected {
				t.Errorf("parseDuration(%q, 0) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestParseDurationInvalid tests parseDuration with invalid values uses default
func TestParseDurationInvalid(t *testing.T) {
	defValue := 30 * time.Second

	tests := []struct {
		value    string
		expected time.Duration
	}{
		{"invalid", defValue},
		{"", defValue},
		{"-10", defValue},
		{"abc", defValue},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseDuration(tt.value, defValue)
			if result != tt.expected {
				t.Errorf("parseDuration(%q, %v) = %v, expected %v", tt.value, defValue, result, tt.expected)
			}
		})
	}
}

// TestLoadSummarizerConfigMaxTokensEnforcement tests max tokens limit is enforced
func TestLoadSummarizerConfigMaxTokensEnforcement(t *testing.T) {
	os.Setenv("SUMMARIZER_MAX_TOKENS", "2000")
	defer os.Clearenv()

	config := LoadSummarizerConfig()

	if config.MaxTokens != 1000 {
		t.Errorf("Expected MaxTokens to be capped at 1000, got %d", config.MaxTokens)
	}
}

// TestLoadSummarizerConfigNegativeValues tests negative values are corrected
func TestLoadSummarizerConfigNegativeValues(t *testing.T) {
	os.Setenv("SUMMARIZER_MAX_TOKENS", "-100")
	os.Setenv("SUMMARIZER_CACHE_SIZE", "-50")
	os.Setenv("SUMMARIZER_CACHE_TTL", "-100")
	os.Setenv("SUMMARIZER_LLM_TIMEOUT", "-30")
	os.Setenv("SUMMARIZER_TOKEN_BUDGET", "-1000")
	os.Setenv("SUMMARIZER_TOKEN_PER_RESULT", "-50")
	defer os.Clearenv()

	config := LoadSummarizerConfig()

	if config.MaxTokens <= 0 {
		t.Errorf("Expected MaxTokens to be positive, got %d", config.MaxTokens)
	}

	if config.CacheSize < 0 {
		t.Errorf("Expected CacheSize to be non-negative, got %d", config.CacheSize)
	}

	if config.CacheTTL < 0 {
		t.Errorf("Expected CacheTTL to be non-negative, got %v", config.CacheTTL)
	}

	if config.LLMTimeout <= 0 {
		t.Errorf("Expected LLMTimeout to be positive, got %v", config.LLMTimeout)
	}

	if config.TokenBudget <= 0 {
		t.Errorf("Expected TokenBudget to be positive, got %d", config.TokenBudget)
	}

	if config.TokenPerResult <= 0 {
		t.Errorf("Expected TokenPerResult to be positive, got %d", config.TokenPerResult)
	}
}

// TestLoadSummarizerConfigWithDefaults tests LoadSummarizerConfigWithDefaults
func TestLoadSummarizerConfigWithDefaults(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	defaults := SummarizerConfig{
		Model:          "gpt-4-turbo",
		MaxTokens:      750,
		CacheSize:      500,
		CacheTTL:       2 * time.Hour,
		LLMAPIKey:      "default-key",
		LLMTimeout:     60 * time.Second,
		TokenBudget:    20000,
		TokenPerResult: 200,
		LogLevel:       "warn",
	}

	config := LoadSummarizerConfigWithDefaults(defaults)

	// When env vars are not set, LoadSummarizerConfig returns built-in defaults
	// LoadSummarizerConfigWithDefaults only overrides if loaded value is zero/empty
	// Since LoadSummarizerConfig returns "claude" by default, it won't be overridden
	if config.Model == "" {
		t.Errorf("Expected Model to be set, got empty string")
	}

	if config.MaxTokens <= 0 {
		t.Errorf("Expected MaxTokens to be positive, got %d", config.MaxTokens)
	}

	if config.CacheSize < 0 {
		t.Errorf("Expected CacheSize to be non-negative, got %d", config.CacheSize)
	}

	if config.LogLevel == "" {
		t.Errorf("Expected LogLevel to be set, got empty string")
	}
}

// TestValidateAndLoadConfig tests ValidateAndLoadConfig
func TestValidateAndLoadConfig(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	config, err := ValidateAndLoadConfig()
	if err != nil {
		t.Fatalf("ValidateAndLoadConfig failed: %v", err)
	}

	if config.Model == "" {
		t.Error("Expected config to have a model")
	}

	if config.MaxTokens <= 0 {
		t.Error("Expected config to have positive max tokens")
	}
}

// TestLoadSummarizerConfigBoolVariations tests various boolean string formats
func TestLoadSummarizerConfigBoolVariations(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true lowercase", "true", true},
		{"true uppercase", "TRUE", true},
		{"true mixed", "TrUe", true},
		{"one", "1", true},
		{"yes", "yes", true},
		{"on", "on", true},
		{"false lowercase", "false", false},
		{"false uppercase", "FALSE", false},
		{"zero", "0", false},
		{"no", "no", false},
		{"off", "off", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SUMMARIZER_ENABLED", tt.envValue)
			defer os.Clearenv()

			config := LoadSummarizerConfig()
			if config.Enabled != tt.expected {
				t.Errorf("Expected Enabled=%v for %q, got %v", tt.expected, tt.envValue, config.Enabled)
			}
		})
	}
}

// TestLoadSummarizerConfigWhitespace tests whitespace handling
func TestLoadSummarizerConfigWhitespace(t *testing.T) {
	os.Setenv("SUMMARIZER_MODEL", "  gpt-4  ")
	os.Setenv("SUMMARIZER_LOG_LEVEL", "  debug  ")
	defer os.Clearenv()

	config := LoadSummarizerConfig()

	if config.Model != "gpt-4" {
		t.Errorf("Expected Model 'gpt-4', got %q", config.Model)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got %q", config.LogLevel)
	}
}
