package summarizer

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadSummarizerConfig loads the summarizer configuration from environment variables
// with sensible defaults. Supports both SUMMARIZER_* and SUMMARY_* prefixes for flexibility.
//
// Environment variables (primary / alternative):
//
// SUMMARIZER_ENABLED / ENABLE_CODE_SUMMARIES - Enable/disable the summarizer (default: true)
// SUMMARIZER_MODEL / SUMMARY_MODEL - LLM model to use (default: "claude")
// SUMMARIZER_MAX_TOKENS / SUMMARY_MAX_TOKENS - Maximum tokens for summaries (default: 500, max: 1000)
// SUMMARIZER_CACHE_ENABLED / SUMMARY_CACHE_ENABLED - Enable/disable caching (default: true)
// SUMMARIZER_FALLBACK_TO_HEURISTIC / SUMMARY_FALLBACK_HEURISTIC - Fallback to heuristic if LLM fails (default: true)
// SUMMARIZER_CACHE_SIZE / SUMMARY_CACHE_SIZE - Maximum cache entries (default: 1000)
// SUMMARIZER_CACHE_TTL / SUMMARY_CACHE_TTL - Cache time-to-live in seconds (default: 3600)
// SUMMARIZER_LLM_API_KEY / ANTHROPIC_API_KEY - API key for LLM service (default: "")
// SUMMARIZER_LLM_TIMEOUT / SUMMARY_LLM_TIMEOUT - LLM request timeout in seconds (default: 30)
// SUMMARIZER_TOKEN_BUDGET - Total token budget per request (default: 10000)
// SUMMARIZER_TOKEN_PER_RESULT - Tokens allocated per result (default: 100)
// SUMMARIZER_METRICS_ENABLED / SUMMARY_METRICS_ENABLED - Enable/disable metrics collection (default: false)
// SUMMARIZER_LOG_LEVEL - Log level (default: "info")
func LoadSummarizerConfig() SummarizerConfig {
	config := SummarizerConfig{
		Enabled:              parseBool(getEnvWithFallback("SUMMARIZER_ENABLED", "ENABLE_CODE_SUMMARIES"), true),
		Model:                parseString(getEnvWithFallback("SUMMARIZER_MODEL", "SUMMARY_MODEL"), "claude"),
		MaxTokens:            parseInt(getEnvWithFallback("SUMMARIZER_MAX_TOKENS", "SUMMARY_MAX_TOKENS"), 500),
		CacheEnabled:         parseBool(getEnvWithFallback("SUMMARIZER_CACHE_ENABLED", "SUMMARY_CACHE_ENABLED"), true),
		FallbackToHeuristic:  parseBool(getEnvWithFallback("SUMMARIZER_FALLBACK_TO_HEURISTIC", "SUMMARY_FALLBACK_HEURISTIC"), true),
		CacheSize:            parseInt(getEnvWithFallback("SUMMARIZER_CACHE_SIZE", "SUMMARY_CACHE_SIZE"), 1000),
		CacheTTL:             parseDuration(getEnvWithFallback("SUMMARIZER_CACHE_TTL", "SUMMARY_CACHE_TTL"), 3600*time.Second),
		LLMAPIKey:            parseString(getEnvWithFallback("SUMMARIZER_LLM_API_KEY", "ANTHROPIC_API_KEY"), ""),
		LLMTimeout:           parseDuration(getEnvWithFallback("SUMMARIZER_LLM_TIMEOUT", "SUMMARY_LLM_TIMEOUT"), 30*time.Second),
		TokenBudget:          parseInt(os.Getenv("SUMMARIZER_TOKEN_BUDGET"), 10000),
		TokenPerResult:       parseInt(os.Getenv("SUMMARIZER_TOKEN_PER_RESULT"), 100),
		MetricsEnabled:       parseBool(getEnvWithFallback("SUMMARIZER_METRICS_ENABLED", "SUMMARY_METRICS_ENABLED"), false),
		LogLevel:             parseString(os.Getenv("SUMMARIZER_LOG_LEVEL"), "info"),
	}

	// Enforce maximum token limit
	if config.MaxTokens > 1000 {
		config.MaxTokens = 1000
	}

	// Ensure positive values
	if config.MaxTokens <= 0 {
		config.MaxTokens = 500
	}

	if config.CacheSize < 0 {
		config.CacheSize = 0
	}

	if config.CacheTTL < 0 {
		config.CacheTTL = 0
	}

	if config.LLMTimeout <= 0 {
		config.LLMTimeout = 30 * time.Second
	}

	if config.TokenBudget <= 0 {
		config.TokenBudget = 10000
	}

	if config.TokenPerResult <= 0 {
		config.TokenPerResult = 100
	}

	return config
}

// getEnvWithFallback tries the primary env var, then falls back to alternative
func getEnvWithFallback(primary, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return os.Getenv(fallback)
}

// parseBool parses a boolean string value with a default fallback
func parseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}

	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// parseString returns the string value or a default if empty
func parseString(value string, defaultValue string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultValue
	}
	return trimmed
}

// parseInt parses an integer string value with a default fallback
func parseInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return defaultValue
	}

	return parsed
}

// parseDuration parses a duration from seconds string with a default fallback
func parseDuration(value string, defaultValue time.Duration) time.Duration {
	if value == "" {
		return defaultValue
	}

	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return defaultValue
	}

	if seconds < 0 {
		return defaultValue
	}

	return time.Duration(seconds) * time.Second
}

// LoadSummarizerConfigWithDefaults loads configuration and applies custom defaults
// This is useful for testing or when you want to override specific defaults
func LoadSummarizerConfigWithDefaults(defaults SummarizerConfig) SummarizerConfig {
	config := LoadSummarizerConfig()

	// Only override with defaults if the loaded value is zero/empty
	if config.Model == "" {
		config.Model = defaults.Model
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = defaults.MaxTokens
	}

	if config.CacheSize == 0 && defaults.CacheSize > 0 {
		config.CacheSize = defaults.CacheSize
	}

	if config.CacheTTL == 0 && defaults.CacheTTL > 0 {
		config.CacheTTL = defaults.CacheTTL
	}

	if config.LLMAPIKey == "" {
		config.LLMAPIKey = defaults.LLMAPIKey
	}

	if config.LLMTimeout == 0 {
		config.LLMTimeout = defaults.LLMTimeout
	}

	if config.TokenBudget == 0 {
		config.TokenBudget = defaults.TokenBudget
	}

	if config.TokenPerResult == 0 {
		config.TokenPerResult = defaults.TokenPerResult
	}

	if config.LogLevel == "" {
		config.LogLevel = defaults.LogLevel
	}

	return config
}

// ValidateAndLoadConfig loads configuration and validates it
// Returns error if configuration is invalid
func ValidateAndLoadConfig() (SummarizerConfig, error) {
	config := LoadSummarizerConfig()

	if err := validateConfig(config); err != nil {
		return SummarizerConfig{}, fmt.Errorf("invalid summarizer configuration: %w", err)
	}

	return config, nil
}
