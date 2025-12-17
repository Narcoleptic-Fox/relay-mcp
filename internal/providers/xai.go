package providers

import (
	    "fmt"
	    "time"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
	)
	const xaiBaseURL = "https://api.x.ai/v1"

// XAIProvider implements Provider for X.AI (Grok)
type XAIProvider struct {
	*OpenAICompatProvider
}

// NewXAIProvider creates a new X.AI provider
func NewXAIProvider(cfg *config.Config) (*XAIProvider, error) {
	if cfg.XAIAPIKey == "" {
		return nil, fmt.Errorf("XAI_API_KEY not configured")
	}

	models := cfg.ModelRegistries[types.ProviderXAI]
	if len(models) == 0 {
		models = defaultXAIModels()
	}

	return &XAIProvider{
		OpenAICompatProvider: NewOpenAICompatProvider(
			types.ProviderXAI,
			cfg.XAIAPIKey,
			xaiBaseURL,
			models,
			5*time.Minute,
		),
	}, nil
}

func defaultXAIModels() []types.ModelCapabilities {
	return []types.ModelCapabilities{
		{
			Provider:          types.ProviderXAI,
			ModelName:         "grok-beta",
			FriendlyName:      "Grok Beta",
			IntelligenceScore: 85,
			ContextWindow:     128000,
			MaxOutputTokens:   4096,
			SupportsStreaming: true,
		},
	}
}
