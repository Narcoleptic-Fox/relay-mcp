package providers

import (
	    "fmt"
	    "time"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
	)
	// CustomProvider implements Provider for local models (Ollama, vLLM, etc.)
type CustomProvider struct {
	*OpenAICompatProvider
}

// NewCustomProvider creates a new custom provider
func NewCustomProvider(cfg *config.Config) (*CustomProvider, error) {
	if cfg.CustomAPIURL == "" {
		return nil, fmt.Errorf("CUSTOM_API_URL not configured")
	}

	models := cfg.ModelRegistries[types.ProviderCustom]
	if len(models) == 0 {
		models = defaultCustomModels()
	}

	return &CustomProvider{
		OpenAICompatProvider: NewOpenAICompatProvider(
			types.ProviderCustom,
			"ollama", // Ollama doesn't need a real key
			cfg.CustomAPIURL,
			models,
			10*time.Minute, // Longer timeout for local inference
		),
	}, nil
}

func defaultCustomModels() []types.ModelCapabilities {
	return []types.ModelCapabilities{
		{
			Provider:                 types.ProviderCustom,
			ModelName:                "llama3.2",
			FriendlyName:             "Llama 3.2",
			IntelligenceScore:        50,
			Aliases:                  []string{"llama", "local-llama"},
			ContextWindow:            128000,
			MaxOutputTokens:          8192,
			SupportsExtendedThinking: false,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           false,
			AllowCodeGeneration:      true,
		},
		{
			Provider:                 types.ProviderCustom,
			ModelName:                "codellama",
			FriendlyName:             "Code Llama",
			IntelligenceScore:        45,
			Aliases:                  []string{"code"},
			ContextWindow:            16384,
			MaxOutputTokens:          4096,
			SupportsExtendedThinking: false,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           false,
			AllowCodeGeneration:      true,
		},
	}
}
