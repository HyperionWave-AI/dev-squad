package main

import (
	"context"
	"fmt"
	"os"
	"time"

	aiservice "hyper/internal/ai-service"
)

func main() {
	fmt.Println("=== OpenAI Provider Test with DeepInfra ===")
	fmt.Println()

	// Configure for DeepInfra
	// DeepInfra API endpoint (OpenAI-compatible)
	os.Setenv("AI_PROVIDER", "openai")
	os.Setenv("PROVIDER_URL", "https://api.deepinfra.com/v1/openai")
	os.Setenv("API_KEY", "CbceTXwStWFjr4V8qFdVEjmuqcjdnwf0")
	// Best models on DeepInfra for tool use:
	// - meta-llama/Llama-3.3-70B-Instruct (excellent, fast)
	// - Qwen/Qwen2.5-72B-Instruct (great for coding)
	// - deepseek-ai/DeepSeek-V3 (very capable)
	os.Setenv("AI_MODEL", "meta-llama/Llama-3.3-70B-Instruct")
	os.Setenv("TEMPERATURE", "0.7")
	os.Setenv("MAX_OUT_TOKENS", "1024")

	// Load config
	config, err := aiservice.LoadAIConfig("")
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Config loaded:\n")
	fmt.Printf("  Provider: %s\n", config.Provider)
	fmt.Printf("  Model: %s\n", config.Model)
	fmt.Printf("  URL: %s\n", config.ProviderURL)
	fmt.Printf("  Temperature: %.1f\n", config.Temperature)
	fmt.Println()

	// Create provider
	provider, err := aiservice.NewChatProvider(config, nil)
	if err != nil {
		fmt.Printf("❌ Failed to create provider: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Provider created")

	// Check tool support
	if toolProvider, ok := provider.(aiservice.ToolCapableProvider); ok {
		fmt.Printf("✓ Tool support: %v\n", toolProvider.SupportsTools())
	}
	fmt.Println()

	// Test 1: Simple streaming chat
	fmt.Println("=== Test 1: Simple Streaming Chat ===")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []aiservice.Message{
		{Role: "system", Content: "You are a helpful assistant. Be concise."},
		{Role: "user", Content: "What is 2+2? Answer in one word."},
	}

	fmt.Print("Response: ")
	textChan, err := provider.StreamChat(ctx, messages)
	if err != nil {
		fmt.Printf("\n❌ StreamChat error: %v\n", err)
		os.Exit(1)
	}

	for chunk := range textChan {
		fmt.Print(chunk)
	}
	fmt.Println()
	fmt.Println("✓ Streaming chat completed")
	fmt.Println()

	// Test 2: Tool calling
	fmt.Println("=== Test 2: Tool Calling ===")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel2()

	toolProvider, ok := provider.(aiservice.ToolCapableProvider)
	if !ok {
		fmt.Println("❌ Provider does not support tools")
		os.Exit(1)
	}

	// Define a test tool
	tools := []aiservice.Tool{
		{
			Type: "function",
			Function: &aiservice.FunctionDefinition{
				Name:        "get_weather",
				Description: "Get the current weather for a location",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The city name, e.g. San Francisco",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	toolMessages := []aiservice.Message{
		{Role: "system", Content: "You are a helpful assistant. Use tools when needed."},
		{Role: "user", Content: "What's the weather in Tokyo?"},
	}

	fmt.Println("Sending request with tools...")
	response, err := toolProvider.StreamChatWithTools(ctx2, toolMessages, tools)
	if err != nil {
		fmt.Printf("❌ StreamChatWithTools error: %v\n", err)
		os.Exit(1)
	}

	// Drain text channel
	var textContent string
	for chunk := range response.TextChannel {
		textContent += chunk
	}

	fmt.Printf("Text Response: %s\n", textContent)
	fmt.Printf("Stop Reason: %s\n", response.StopReason)
	fmt.Printf("Tool Calls: %d\n", len(response.ToolCalls))

	for i, tc := range response.ToolCalls {
		fmt.Printf("  Tool %d: %s (ID: %s)\n", i+1, tc.Name, tc.ID)
		fmt.Printf("    Args: %v\n", tc.Args)
	}

	if response.TokenUsage != nil {
		fmt.Printf("Token Usage:\n")
		fmt.Printf("  Prompt: %d\n", response.TokenUsage.PromptTokens)
		fmt.Printf("  Completion: %d\n", response.TokenUsage.CompletionTokens)
		fmt.Printf("  Total: %d\n", response.TokenUsage.TotalTokens)
	}

	fmt.Println()
	fmt.Println("=== All Tests Completed ===")
}
