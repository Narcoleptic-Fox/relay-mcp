package providers

import (
	"context"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/types"
)

// Provider is the interface for AI model providers
type Provider interface {
	// GetProviderType returns the provider type
	GetProviderType() types.ProviderType

	// GenerateContent generates a response from the model
	GenerateContent(ctx context.Context, req *GenerateRequest) (*types.ModelResponse, error)

	// ListModels returns available models
	ListModels() []types.ModelCapabilities

	// GetCapabilities returns capabilities for a specific model
	GetCapabilities(modelName string) (*types.ModelCapabilities, error)

	// CountTokens estimates token count for text
	CountTokens(text string, modelName string) (int, error)

	// SupportsModel checks if provider can handle this model
	SupportsModel(modelName string) bool

	// IsConfigured checks if the provider has valid credentials
	IsConfigured() bool
}

// GenerateRequest contains all parameters for generation
type GenerateRequest struct {
	Prompt          string
	SystemPrompt    string
	Model           string
	Temperature     float64
	MaxOutputTokens int

	// Extended thinking
	ThinkingMode   types.ThinkingMode
	ThinkingBudget int

	// Conversation context
	ConversationHistory []types.ConversationTurn

	// Vision
	Images []string
}

// BaseProvider provides common functionality
type BaseProvider struct {
	providerType types.ProviderType
	models       map[string]types.ModelCapabilities
	aliases      map[string]string // alias -> canonical name
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(pt types.ProviderType, models []types.ModelCapabilities) *BaseProvider {
	bp := &BaseProvider{
		providerType: pt,
		models:       make(map[string]types.ModelCapabilities),
		aliases:      make(map[string]string),
	}

	for _, m := range models {
		bp.models[m.ModelName] = m
		for _, alias := range m.Aliases {
			bp.aliases[alias] = m.ModelName
		}
	}

	return bp
}

func (p *BaseProvider) GetProviderType() types.ProviderType {
	return p.providerType
}

func (p *BaseProvider) ListModels() []types.ModelCapabilities {
	models := make([]types.ModelCapabilities, 0, len(p.models))
	for _, m := range p.models {
		models = append(models, m)
	}
	return models
}

func (p *BaseProvider) GetCapabilities(modelName string) (*types.ModelCapabilities, error) {
	// Check direct match
	if m, ok := p.models[modelName]; ok {
		return &m, nil
	}

	// Check alias
	if canonical, ok := p.aliases[modelName]; ok {
		if m, ok := p.models[canonical]; ok {
			return &m, nil
		}
	}

	return nil, ErrModelNotFound{Model: modelName, Provider: p.providerType}
}

func (p *BaseProvider) SupportsModel(modelName string) bool {
	_, err := p.GetCapabilities(modelName)
	return err == nil
}

func (p *BaseProvider) ResolveModelName(modelName string) string {
	if canonical, ok := p.aliases[modelName]; ok {
		return canonical
	}
	return modelName
}
