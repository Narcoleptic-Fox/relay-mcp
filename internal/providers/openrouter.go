package providers

import (
	    "fmt"
	    "time"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
	)
	const openRouterBaseURL = "https://openrouter.ai/api/v1"

// OpenRouterProvider implements Provider for OpenRouter (catch-all)
type OpenRouterProvider struct {
	*OpenAICompatProvider
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(cfg *config.Config) (*OpenRouterProvider, error) {
	if cfg.OpenRouterAPIKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	models := cfg.ModelRegistries[types.ProviderOpenRouter]
	if len(models) == 0 {
		models = defaultOpenRouterModels()
	}

	return &OpenRouterProvider{
		OpenAICompatProvider: NewOpenAICompatProvider(
			types.ProviderOpenRouter,
			cfg.OpenRouterAPIKey,
			openRouterBaseURL,
			models,
			5*time.Minute,
		),
	}, nil
}

func defaultOpenRouterModels() []types.ModelCapabilities {
	// OpenRouter provides access to many models
	return []types.ModelCapabilities{
		{
			Provider:                 types.ProviderOpenRouter,
			ModelName:                "anthropic/claude-3.5-sonnet",
			FriendlyName:             "Claude 3.5 Sonnet",
			IntelligenceScore:        90,
			Aliases:                  []string{"sonnet", "claude-sonnet"},
			ContextWindow:            200000,
			MaxOutputTokens:          8192,
			SupportsExtendedThinking: false,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           true,
			AllowCodeGeneration:      true,
		},
		{
			Provider:                 types.ProviderOpenRouter,
			ModelName:                "meta-llama/llama-3.3-70b-instruct",
			FriendlyName:             "Llama 3.3 70B",
			IntelligenceScore:        75,
			Aliases:                  []string{"llama-70b"},
			ContextWindow:            128000,
			MaxOutputTokens:          8192,
			SupportsExtendedThinking: false,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           false,
			AllowCodeGeneration:      true,
		},
	}
}
