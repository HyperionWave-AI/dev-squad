package aiservice

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// AIConfig holds AI provider configuration from .env.hyper
type AIConfig struct {
	Provider        string  // "openai", "anthropic", or "custom"
	ProviderURL     string  // Custom endpoint URL (for custom provider)
	APIKey          string  // API key for the provider
	MaxIterations   int     // Maximum iteration count (default: 100)
	MaxToolCalls    int     // Maximum tool calls per session (default: 50)
	MaxOutputTokens int     // Maximum output tokens
	Temperature     float64 // Temperature for generation (default: 0.7)
	ReasoningMode   string  // Reasoning mode (e.g., "o1", "o3" for OpenAI)
	Model           string  // Model name (e.g., "gpt-4", "claude-3-sonnet")
}

// LoadAIConfig loads AI configuration from .env.hyper file
// Returns *AIConfig and error if configuration is invalid
func LoadAIConfig(envFilePath string) (*AIConfig, error) {
	// Load .env.hyper file if path provided
	if envFilePath != "" {
		if err := godotenv.Load(envFilePath); err != nil {
			return nil, fmt.Errorf("failed to load .env.hyper: %w", err)
		}
	}

	// Parse provider (required)
	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = os.Getenv("PROVIDER") // fallback to PROVIDER for compatibility
	}
	if provider == "" {
		return nil, fmt.Errorf("AI_PROVIDER or PROVIDER environment variable is required")
	}

	// Validate provider
	if provider != "openai" && provider != "anthropic" && provider != "custom" {
		return nil, fmt.Errorf("provider must be 'openai', 'anthropic', or 'custom', got: %s", provider)
	}

	// Parse provider URL (required for custom provider, optional for openai to support Ollama)
	providerURL := os.Getenv("PROVIDER_URL")
	if providerURL == "" {
		// Try OpenAI-specific base URL for Ollama support
		providerURL = os.Getenv("OPENAI_BASE_URL")
	}
	if provider == "custom" && providerURL == "" {
		return nil, fmt.Errorf("PROVIDER_URL is required for custom provider")
	}

	// Parse API key (required for openai/anthropic)
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		// Try provider-specific keys
		switch provider {
		case "openai":
			apiKey = os.Getenv("OPENAI_API_KEY")
		case "anthropic":
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}
	// API key validation (allow dummy key for Ollama)
	if (provider == "openai" || provider == "anthropic") && apiKey == "" {
		// For Ollama or other local providers, allow default dummy key
		if providerURL != "" && (provider == "openai") {
			apiKey = "ollama" // Ollama doesn't validate keys, but library requires one
		} else {
			return nil, fmt.Errorf("API_KEY or %s_API_KEY environment variable is required for %s provider",
				provider, provider)
		}
	}

	// Parse model name (required)
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = os.Getenv("MODEL") // fallback
	}
	if model == "" {
		// Set defaults based on provider
		switch provider {
		case "openai":
			model = "gpt-4-turbo-preview"
		case "anthropic":
			model = "claude-3-sonnet-20240229"
		default:
			return nil, fmt.Errorf("AI_MODEL or MODEL environment variable is required")
		}
	}

	// Parse max iterations with default (try MAX_ITERATIONS first, fall back to MAX_ITERATION)
	maxIterations := 100
	maxIterStr := os.Getenv("MAX_ITERATIONS")
	if maxIterStr == "" {
		maxIterStr = os.Getenv("MAX_ITERATION") // backwards compatibility
	}
	if maxIterStr != "" {
		if val, err := strconv.Atoi(maxIterStr); err == nil && val > 0 {
			maxIterations = val
		}
	}

	// Parse max tool calls with default
	maxToolCalls := 50
	if maxToolCallsStr := os.Getenv("MAX_TOOL_CALLS"); maxToolCallsStr != "" {
		if val, err := strconv.Atoi(maxToolCallsStr); err == nil && val > 0 {
			maxToolCalls = val
		}
	}

	// Parse max output tokens (optional)
	maxOutputTokens := 0
	if maxTokensStr := os.Getenv("MAX_OUT_TOKENS"); maxTokensStr != "" {
		if val, err := strconv.Atoi(maxTokensStr); err == nil && val > 0 {
			maxOutputTokens = val
		}
	}

	// Parse temperature with default
	temperature := 0.7
	if tempStr := os.Getenv("TEMPERATURE"); tempStr != "" {
		if val, err := strconv.ParseFloat(tempStr, 64); err == nil && val >= 0 && val <= 2.0 {
			temperature = val
		}
	}

	// Parse reasoning mode (optional, for OpenAI o1/o3)
	reasoningMode := os.Getenv("REASONING")

	return &AIConfig{
		Provider:        provider,
		ProviderURL:     providerURL,
		APIKey:          apiKey,
		MaxIterations:   maxIterations,
		MaxToolCalls:    maxToolCalls,
		MaxOutputTokens: maxOutputTokens,
		Temperature:     temperature,
		ReasoningMode:   reasoningMode,
		Model:           model,
	}, nil
}

// Validate checks if the configuration is valid
func (c *AIConfig) Validate() error {
	if c.Provider != "openai" && c.Provider != "anthropic" && c.Provider != "custom" {
		return fmt.Errorf("invalid provider: %s", c.Provider)
	}

	if c.Provider == "custom" && c.ProviderURL == "" {
		return fmt.Errorf("PROVIDER_URL required for custom provider")
	}

	if (c.Provider == "openai" || c.Provider == "anthropic") && c.APIKey == "" {
		return fmt.Errorf("API key required for %s provider", c.Provider)
	}

	if c.Model == "" {
		return fmt.Errorf("model name is required")
	}

	if c.MaxIterations <= 0 {
		return fmt.Errorf("max iterations must be positive")
	}

	if c.Temperature < 0 || c.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0 and 2.0")
	}

	return nil
}
