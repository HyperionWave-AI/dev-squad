package aiservice

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// AIConfig holds AI provider configuration from .env.hyper
type AIConfig struct {
	Provider          string  // "openai", "anthropic", "custom", or "litellm"
	ProviderURL       string  // Custom endpoint URL (for custom provider)
	APIKey            string  // API key for the provider
	MaxIterations     int     // Maximum iteration count (default: 100)
	MaxToolCalls      int     // Maximum tool calls per session (default: 50)
	MaxOutputTokens   int     // Maximum output tokens
	Temperature       float64 // Temperature for generation (default: 0.7)
	ReasoningMode     string  // Reasoning mode (e.g., "o1", "o3" for OpenAI)
	Model             string  // Model name as configured in environment
	FallbackModel     string  // Fallback model when rate limited (e.g., "qwen2.5-coder:7b")
	ContextWindowSize int     // Context window size in tokens (optional, defaults based on provider)
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
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(os.Getenv("PROVIDER"))) // fallback to PROVIDER for compatibility
	}
	if provider == "" {
		return nil, fmt.Errorf("AI_PROVIDER or PROVIDER environment variable is required")
	}

	// Validate provider
	if provider != "openai" && provider != "anthropic" && provider != "custom" && provider != "litellm" {
		return nil, fmt.Errorf("provider must be 'openai', 'anthropic', 'custom', or 'litellm', got: %s", provider)
	}

	// Parse provider URL.
	// Priority: PROVIDER_URL > OPENAI_BASE_URL > LITELLM_BASE_URL
	// This allows a single OpenAI-compatible path (OpenAI, LiteLLM, Ollama, etc.).
	providerURL := strings.TrimSpace(os.Getenv("PROVIDER_URL"))
	if providerURL == "" {
		providerURL = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	}
	if providerURL == "" {
		providerURL = strings.TrimSpace(os.Getenv("LITELLM_BASE_URL"))
	}
	if provider == "litellm" && providerURL == "" {
		providerURL = "http://localhost:4000/v1"
	}
	if provider == "custom" && providerURL == "" {
		return nil, fmt.Errorf("PROVIDER_URL is required for custom provider")
	}

	// Parse API key (required for openai/anthropic)
	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))
	if apiKey == "" {
		// Try provider-specific keys
		switch provider {
		case "openai":
			apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		case "anthropic":
			apiKey = strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
		case "litellm":
			apiKey = strings.TrimSpace(os.Getenv("LITELLM_API_KEY"))
			if apiKey == "" {
				apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
			}
		}
	}
	// API key validation (allow dummy key for OpenAI-compatible local/proxy endpoints)
	if (provider == "openai" || provider == "anthropic" || provider == "litellm" || provider == "custom") && apiKey == "" {
		if provider == "anthropic" {
			return nil, fmt.Errorf("API_KEY or ANTHROPIC_API_KEY environment variable is required for anthropic provider")
		}
		if providerURL != "" {
			// OpenAI SDK requires a key even when upstream doesn't enforce auth.
			apiKey = "dummy-key"
		} else {
			return nil, fmt.Errorf("API_KEY or provider-specific API key environment variable is required for %s provider", provider)
		}
	}

	// Parse model name (required)
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = os.Getenv("MODEL") // fallback
	}
	if model == "" {
		return nil, fmt.Errorf("AI_MODEL or MODEL environment variable is required")
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

	// Parse fallback model (optional, for rate limit handling)
	fallbackModel := os.Getenv("FALLBACK_MODEL")

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
		FallbackModel:   fallbackModel,
	}, nil
}

// Validate checks if the configuration is valid
func (c *AIConfig) Validate() error {
	if c.Provider != "openai" && c.Provider != "anthropic" && c.Provider != "custom" && c.Provider != "litellm" {
		return fmt.Errorf("invalid provider: %s", c.Provider)
	}

	if c.Provider == "custom" && c.ProviderURL == "" {
		return fmt.Errorf("PROVIDER_URL required for custom provider")
	}

	if c.APIKey == "" {
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
