package simple

import (
	    "context"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/memory"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/providers"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
	)
	// BaseTool provides common functionality for simple tools
type BaseTool struct {
	name        string
	description string
	schema      *tools.SchemaBuilder
	cfg         *config.Config
	registry    *providers.Registry
	memory      *memory.ConversationMemory
}

// NewBaseTool creates a new base tool
func NewBaseTool(
	name, description string,
	cfg *config.Config,
	registry *providers.Registry,
	mem *memory.ConversationMemory,
) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      tools.NewSchemaBuilder(),
		cfg:         cfg,
		registry:    registry,
		memory:      mem,
	}
}

func (t *BaseTool) Name() string           { return t.name }
func (t *BaseTool) Description() string    { return t.description }
func (t *BaseTool) Schema() map[string]any { return t.schema.Build() }

// GetProvider finds a provider for the given model
func (t *BaseTool) GetProvider(modelName string) (providers.Provider, error) {
	if modelName == "" || modelName == "auto" {
		// Use auto-selection based on requirements
		caps, provider, err := t.registry.SelectBestModel(providers.ModelRequirements{})
		if err != nil {
			return nil, err
		}
		_ = caps // Use caps if needed
		return provider, nil
	}
	return t.registry.GetProviderForModel(modelName)
}

// ResolveModel determines the actual model to use
func (t *BaseTool) ResolveModel(requestedModel string) (string, providers.Provider, error) {
	if requestedModel == "" || requestedModel == "auto" {
		caps, provider, err := t.registry.SelectBestModel(providers.ModelRequirements{})
		if err != nil {
			return "", nil, err
		}
		return caps.ModelName, provider, nil
	}

	provider, err := t.registry.GetProviderForModel(requestedModel)
	if err != nil {
		return "", nil, err
	}

	return requestedModel, provider, nil
}

// GetOrCreateThread gets or creates a conversation thread
func (t *BaseTool) GetOrCreateThread(continuationID string) (*types.ThreadContext, bool) {
	if continuationID == "" {
		return t.memory.CreateThread(t.name), false
	}

	thread := t.memory.GetThread(continuationID)
	if thread == nil {
		return t.memory.CreateThread(t.name), false
	}

	return thread, true
}

// AddTurn adds a conversation turn
func (t *BaseTool) AddTurn(threadID, role, content string, files, images []string) {
    _ = t.memory.AddTurn(threadID, types.ConversationTurn{
        Role:     role,
        Content:  content,
        Files:    files,
        Images:   images,
        ToolName: t.name,
    })
}
// GenerateContent calls the AI provider
func (t *BaseTool) GenerateContent(
	ctx context.Context,
	provider providers.Provider,
	req *providers.GenerateRequest,
) (*types.ModelResponse, error) {
	return provider.GenerateContent(ctx, req)
}
