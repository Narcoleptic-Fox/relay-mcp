package providers

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/config"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/types"
)

// Priority order for provider selection
var ProviderPriority = []types.ProviderType{
	types.ProviderGemini,
	types.ProviderOpenAI,
	types.ProviderAzure,
	types.ProviderXAI,
	types.ProviderDIAL,
	types.ProviderCustom,
	types.ProviderOpenRouter, // Catch-all last
}

// Registry manages all providers
type Registry struct {
	cfg       *config.Config
	providers map[types.ProviderType]Provider
	mu        sync.RWMutex
}

// NewRegistry creates a new provider registry
func NewRegistry(cfg *config.Config) *Registry {
	return &Registry{
		cfg:       cfg,
		providers: make(map[types.ProviderType]Provider),
	}
}

// Initialize sets up all configured providers
func (r *Registry) Initialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize each provider if configured
	if r.cfg.HasProvider(types.ProviderGemini) {
		p, err := NewGeminiProvider(r.cfg)
		if err != nil {
			slog.Warn("failed to initialize Gemini provider", "error", err)
		} else {
			r.providers[types.ProviderGemini] = p
			slog.Info("initialized provider", "type", types.ProviderGemini)
		}
	}

	if r.cfg.HasProvider(types.ProviderOpenAI) {
		p, err := NewOpenAIProvider(r.cfg)
		if err != nil {
			slog.Warn("failed to initialize OpenAI provider", "error", err)
		} else {
			r.providers[types.ProviderOpenAI] = p
			slog.Info("initialized provider", "type", types.ProviderOpenAI)
		}
	}

	if r.cfg.HasProvider(types.ProviderAzure) {
		p, err := NewAzureProvider(r.cfg)
		if err != nil {
			slog.Warn("failed to initialize Azure provider", "error", err)
		} else {
			r.providers[types.ProviderAzure] = p
			slog.Info("initialized provider", "type", types.ProviderAzure)
		}
	}

	if r.cfg.HasProvider(types.ProviderXAI) {
		p, err := NewXAIProvider(r.cfg)
		if err != nil {
			slog.Warn("failed to initialize XAI provider", "error", err)
		} else {
			r.providers[types.ProviderXAI] = p
			slog.Info("initialized provider", "type", types.ProviderXAI)
		}
	}

	if r.cfg.HasProvider(types.ProviderDIAL) {
		p, err := NewDIALProvider(r.cfg)
		if err != nil {
			slog.Warn("failed to initialize DIAL provider", "error", err)
		} else {
			r.providers[types.ProviderDIAL] = p
			slog.Info("initialized provider", "type", types.ProviderDIAL)
		}
	}

	if r.cfg.HasProvider(types.ProviderCustom) {
		p, err := NewCustomProvider(r.cfg)
		if err != nil {
			slog.Warn("failed to initialize Custom provider", "error", err)
		} else {
			r.providers[types.ProviderCustom] = p
			slog.Info("initialized provider", "type", types.ProviderCustom)
		}
	}

	if r.cfg.HasProvider(types.ProviderOpenRouter) {
		p, err := NewOpenRouterProvider(r.cfg)
		if err != nil {
			slog.Warn("failed to initialize OpenRouter provider", "error", err)
		} else {
			r.providers[types.ProviderOpenRouter] = p
			slog.Info("initialized provider", "type", types.ProviderOpenRouter)
		}
	}

	if len(r.providers) == 0 {
		// For initial testing, we might return nil if no providers are set,
		// but the main.go expects an error if initialization fails.
		// However, if we have NO providers implemented yet, this will always fail.
		// I will return nil for now to allow the server to start even without providers (it just won't have models).
		slog.Warn("no providers configured or initialized")
		return nil
	}

	return nil
}

// GetProvider returns a specific provider
func (r *Registry) GetProvider(pt types.ProviderType) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[pt]
	return p, ok
}

// GetProviderForModel finds the best provider for a model
func (r *Registry) GetProviderForModel(modelName string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check providers in priority order
	for _, pt := range ProviderPriority {
		if p, ok := r.providers[pt]; ok && p.SupportsModel(modelName) {
			return p, nil
		}
	}

	return nil, ErrModelNotFound{Model: modelName}
}

// GetAllModels returns all available models across providers
func (r *Registry) GetAllModels() []types.ModelCapabilities {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []types.ModelCapabilities
	seen := make(map[string]bool)

	for _, pt := range ProviderPriority {
		if p, ok := r.providers[pt]; ok {
			for _, m := range p.ListModels() {
				if !seen[m.ModelName] {
					models = append(models, m)
					seen[m.ModelName] = true
				}
			}
		}
	}

	// Sort by intelligence score (descending)
	sort.Slice(models, func(i, j int) bool {
		return models[i].IntelligenceScore > models[j].IntelligenceScore
	})

	return models
}

// SelectBestModel finds the best model for a task
func (r *Registry) SelectBestModel(requirements ModelRequirements) (*types.ModelCapabilities, Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bestModel *types.ModelCapabilities
	var bestProvider Provider

	for _, pt := range ProviderPriority {
		p, ok := r.providers[pt]
		if !ok {
			continue
		}

		for _, m := range p.ListModels() {
			if !meetsRequirements(m, requirements) {
				continue
			}

			if bestModel == nil || m.IntelligenceScore > bestModel.IntelligenceScore {
				mCopy := m
				bestModel = &mCopy
				bestProvider = p
			}
		}
	}

	if bestModel == nil {
		return nil, nil, fmt.Errorf("no model meets requirements")
	}

	return bestModel, bestProvider, nil
}

// ModelRequirements specifies what a model needs to support
type ModelRequirements struct {
	MinIntelligence     int
	NeedsThinking       bool
	NeedsVision         bool
	NeedsCodeGeneration bool
	MinContextWindow    int
}

func meetsRequirements(m types.ModelCapabilities, req ModelRequirements) bool {
	if m.IntelligenceScore < req.MinIntelligence {
		return false
	}
	if req.NeedsThinking && !m.SupportsExtendedThinking {
		return false
	}
	if req.NeedsVision && !m.SupportsVision {
		return false
	}
	if req.NeedsCodeGeneration && !m.AllowCodeGeneration {
		return false
	}
	if m.ContextWindow < req.MinContextWindow {
		return false
	}
	return true
}
