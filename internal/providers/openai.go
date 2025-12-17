package providers

import (
	    "fmt"
	    "time"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
	)
	const openAIBaseURL = "https://api.openai.com/v1"

// OpenAIProvider implements Provider for OpenAI
type OpenAIProvider struct {
	*OpenAICompatProvider
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(cfg *config.Config) (*OpenAIProvider, error) {
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not configured")
	}

	models := cfg.ModelRegistries[types.ProviderOpenAI]
	if len(models) == 0 {
		models = defaultOpenAIModels()
	}

	return &OpenAIProvider{
		OpenAICompatProvider: NewOpenAICompatProvider(
			types.ProviderOpenAI,
			cfg.OpenAIAPIKey,
			openAIBaseURL,
			models,
			5*time.Minute,
		),
	}, nil
}

func defaultOpenAIModels() []types.ModelCapabilities {
	return []types.ModelCapabilities{
		{
			Provider:                 types.ProviderOpenAI,
			ModelName:                "gpt-5",
			FriendlyName:             "GPT-5",
			IntelligenceScore:        95,
			Aliases:                  []string{"gpt5"},
			ContextWindow:            128000,
			MaxOutputTokens:          16384,
			SupportsExtendedThinking: false,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           true,
			AllowCodeGeneration:      true,
		},
		{
			Provider:                 types.ProviderOpenAI,
			ModelName:                "o3",
			FriendlyName:             "O3",
			IntelligenceScore:        98,
			Aliases:                  []string{},
			ContextWindow:            200000,
			MaxOutputTokens:          100000,
			SupportsExtendedThinking: true,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           true,
			AllowCodeGeneration:      true,
		},
		{
			Provider:                 types.ProviderOpenAI,
			ModelName:                "o4-mini",
			FriendlyName:             "O4 Mini",
			IntelligenceScore:        70,
			Aliases:                  []string{"o4"},
			ContextWindow:            128000,
			MaxOutputTokens:          65536,
			SupportsExtendedThinking: true,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           true,
			AllowCodeGeneration:      true,
		},
	}
}
