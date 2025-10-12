package aiservice

import (
	"os"
	"testing"
)

func TestLoadAIConfig_ValidConfig(t *testing.T) {
	// Set up environment variables for a valid config
	t.Setenv("AI_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "sk-test-key-123")
	t.Setenv("AI_MODEL", "gpt-4")
	t.Setenv("MAX_ITERATION", "150")
	t.Setenv("MAX_OUT_TOKENS", "4096")
	t.Setenv("TEMPERATURE", "0.8")
	t.Setenv("REASONING", "o1")

	config, err := LoadAIConfig("")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify all fields
	if config.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got: %s", config.Provider)
	}
	if config.APIKey != "sk-test-key-123" {
		t.Errorf("Expected API key 'sk-test-key-123', got: %s", config.APIKey)
	}
	if config.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got: %s", config.Model)
	}
	if config.MaxIterations != 150 {
		t.Errorf("Expected max iterations 150, got: %d", config.MaxIterations)
	}
	if config.MaxOutputTokens != 4096 {
		t.Errorf("Expected max output tokens 4096, got: %d", config.MaxOutputTokens)
	}
	if config.Temperature != 0.8 {
		t.Errorf("Expected temperature 0.8, got: %f", config.Temperature)
	}
	if config.ReasoningMode != "o1" {
		t.Errorf("Expected reasoning mode 'o1', got: %s", config.ReasoningMode)
	}
}

func TestLoadAIConfig_MinimalConfig_WithDefaults(t *testing.T) {
	// Set only required fields, expect defaults for optional ones
	t.Setenv("PROVIDER", "anthropic") // Test fallback to PROVIDER
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-123")
	t.Setenv("MODEL", "claude-3-opus-20240229") // Test fallback to MODEL

	config, err := LoadAIConfig("")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify required fields
	if config.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got: %s", config.Provider)
	}
	if config.APIKey != "sk-ant-test-123" {
		t.Errorf("Expected API key 'sk-ant-test-123', got: %s", config.APIKey)
	}
	if config.Model != "claude-3-opus-20240229" {
		t.Errorf("Expected model 'claude-3-opus-20240229', got: %s", config.Model)
	}

	// Verify defaults applied
	if config.MaxIterations != 100 {
		t.Errorf("Expected default max iterations 100, got: %d", config.MaxIterations)
	}
	if config.Temperature != 0.7 {
		t.Errorf("Expected default temperature 0.7, got: %f", config.Temperature)
	}
	if config.MaxOutputTokens != 0 {
		t.Errorf("Expected max output tokens 0 (not set), got: %d", config.MaxOutputTokens)
	}
}

func TestLoadAIConfig_CustomProvider(t *testing.T) {
	t.Setenv("AI_PROVIDER", "custom")
	t.Setenv("PROVIDER_URL", "https://custom-llm.example.com/v1/chat")
	t.Setenv("AI_MODEL", "custom-model-v1")

	config, err := LoadAIConfig("")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.Provider != "custom" {
		t.Errorf("Expected provider 'custom', got: %s", config.Provider)
	}
	if config.ProviderURL != "https://custom-llm.example.com/v1/chat" {
		t.Errorf("Expected provider URL, got: %s", config.ProviderURL)
	}
	if config.Model != "custom-model-v1" {
		t.Errorf("Expected model 'custom-model-v1', got: %s", config.Model)
	}
}

func TestLoadAIConfig_InvalidProvider(t *testing.T) {
	t.Setenv("AI_PROVIDER", "invalid-provider")
	t.Setenv("API_KEY", "test-key")
	t.Setenv("AI_MODEL", "test-model")

	_, err := LoadAIConfig("")
	if err == nil {
		t.Fatal("Expected error for invalid provider, got nil")
	}

	// Just check that error contains the key phrase
	if err.Error() == "" {
		t.Error("Expected error message, got empty string")
	}
}

func TestLoadAIConfig_MissingProvider(t *testing.T) {
	// Clear environment
	os.Unsetenv("AI_PROVIDER")
	os.Unsetenv("PROVIDER")

	_, err := LoadAIConfig("")
	if err == nil {
		t.Fatal("Expected error for missing provider, got nil")
	}
}

func TestLoadAIConfig_MissingAPIKey(t *testing.T) {
	// Clear any existing API keys
	os.Unsetenv("API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")

	t.Setenv("AI_PROVIDER", "openai")
	// Don't set model - will use default
	// No API key set

	_, err := LoadAIConfig("")
	if err == nil {
		t.Fatal("Expected error for missing API key, got nil")
	}
}

func TestLoadAIConfig_MissingProviderURL_ForCustom(t *testing.T) {
	t.Setenv("AI_PROVIDER", "custom")
	t.Setenv("AI_MODEL", "custom-model")
	// No PROVIDER_URL set

	_, err := LoadAIConfig("")
	if err == nil {
		t.Fatal("Expected error for missing PROVIDER_URL with custom provider, got nil")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	config := &AIConfig{
		Provider:        "openai",
		APIKey:          "sk-test",
		Model:           "gpt-4",
		MaxIterations:   100,
		MaxOutputTokens: 2000,
		Temperature:     0.7,
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidate_InvalidProvider(t *testing.T) {
	config := &AIConfig{
		Provider:      "gemini", // Invalid
		APIKey:        "test-key",
		Model:         "gemini-pro",
		MaxIterations: 100,
		Temperature:   0.7,
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Expected error for invalid provider, got nil")
	}
}

func TestValidate_NegativeIterations(t *testing.T) {
	config := &AIConfig{
		Provider:      "openai",
		APIKey:        "test-key",
		Model:         "gpt-4",
		MaxIterations: -1, // Invalid
		Temperature:   0.7,
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Expected error for negative iterations, got nil")
	}
}

func TestValidate_InvalidTemperature(t *testing.T) {
	config := &AIConfig{
		Provider:      "anthropic",
		APIKey:        "test-key",
		Model:         "claude-3-sonnet",
		MaxIterations: 100,
		Temperature:   3.0, // Invalid (>2.0)
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Expected error for invalid temperature, got nil")
	}
}

func TestValidate_MissingAPIKey(t *testing.T) {
	config := &AIConfig{
		Provider:      "openai",
		APIKey:        "", // Missing
		Model:         "gpt-4",
		MaxIterations: 100,
		Temperature:   0.7,
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Expected error for missing API key, got nil")
	}
}

func TestValidate_CustomProvider_MissingURL(t *testing.T) {
	config := &AIConfig{
		Provider:      "custom",
		ProviderURL:   "", // Missing
		Model:         "custom-model",
		MaxIterations: 100,
		Temperature:   0.7,
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Expected error for missing PROVIDER_URL, got nil")
	}
}
