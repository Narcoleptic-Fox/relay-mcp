package providers

import (
	"fmt"
	"time"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/config"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/types"
)

// DIALProvider implements Provider for DIAL
type DIALProvider struct {
	*OpenAICompatProvider
}

// NewDIALProvider creates a new DIAL provider
func NewDIALProvider(cfg *config.Config) (*DIALProvider, error) {
	if cfg.DIALAPIKey == "" {
		return nil, fmt.Errorf("DIAL_API_KEY not configured")
	}
	if cfg.DIALEndpoint == "" {
		return nil, fmt.Errorf("DIAL_ENDPOINT not configured")
	}

	models := cfg.ModelRegistries[types.ProviderDIAL]
	if len(models) == 0 {
		models = defaultDIALModels()
	}

	return &DIALProvider{
		OpenAICompatProvider: NewOpenAICompatProvider(
			types.ProviderDIAL,
			cfg.DIALAPIKey,
			cfg.DIALEndpoint,
			models,
			5*time.Minute,
		),
	}, nil
}

func defaultDIALModels() []types.ModelCapabilities {
	return []types.ModelCapabilities{
		{
			Provider:          types.ProviderDIAL,
			ModelName:         "gpt-4",
			FriendlyName:      "DIAL GPT-4",
			IntelligenceScore: 90,
			ContextWindow:     8192,
			MaxOutputTokens:   4096,
			SupportsStreaming: true,
		},
	}
}
