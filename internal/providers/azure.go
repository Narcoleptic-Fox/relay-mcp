package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	"github.com/Narcoleptic-Fox/relay-mcp/internal/types"
)

const (
	// Azure API version - using a stable, recent version
	azureAPIVersion = "2024-02-15-preview"
)

// AzureProvider implements Provider for Azure OpenAI
type AzureProvider struct {
	*BaseProvider
	apiKey     string
	endpoint   string
	httpClient *http.Client
}

// NewAzureProvider creates a new Azure provider
func NewAzureProvider(cfg *config.Config) (*AzureProvider, error) {
	if cfg.AzureAPIKey == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_API_KEY not configured")
	}
	if cfg.AzureEndpoint == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_ENDPOINT not configured")
	}

	models := cfg.ModelRegistries[types.ProviderAzure]
	if len(models) == 0 {
		models = defaultAzureModels()
	}

	// Normalize endpoint - remove trailing slash
	endpoint := strings.TrimSuffix(cfg.AzureEndpoint, "/")

	return &AzureProvider{
		BaseProvider: NewBaseProvider(types.ProviderAzure, models),
		apiKey:       cfg.AzureAPIKey,
		endpoint:     endpoint,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

func (p *AzureProvider) IsConfigured() bool {
	return p.apiKey != "" && p.endpoint != ""
}

func (p *AzureProvider) CountTokens(text string, modelName string) (int, error) {
	// Rough estimate: 4 chars per token
	return len(text) / 4, nil
}

// GenerateContent calls the Azure OpenAI API with proper authentication
func (p *AzureProvider) GenerateContent(ctx context.Context, req *GenerateRequest) (*types.ModelResponse, error) {
	modelName := p.ResolveModelName(req.Model)

	// Build messages
	messages := p.buildMessages(req)

	// Build request body - Azure doesn't need "model" in body, it's in the URL
	body := map[string]any{
		"messages": messages,
	}

	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxOutputTokens > 0 {
		body["max_tokens"] = req.MaxOutputTokens
	}

	// Azure-specific URL format: /openai/deployments/{deployment-name}/chat/completions?api-version={version}
	// The deployment name in Azure typically matches the model name
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.endpoint, modelName, azureAPIVersion)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", p.apiKey) // Azure uses api-key header, NOT Authorization: Bearer

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Azure API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response - Azure uses same format as OpenAI
	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return p.parseResponse(modelName, &oaiResp)
}

func (p *AzureProvider) buildMessages(req *GenerateRequest) []map[string]any {
	var messages []map[string]any

	// Add system prompt
	if req.SystemPrompt != "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}

	// Add conversation history
	for _, turn := range req.ConversationHistory {
		messages = append(messages, map[string]any{
			"role":    turn.Role,
			"content": turn.Content,
		})
	}

	// Add current prompt
	messages = append(messages, map[string]any{
		"role":    "user",
		"content": req.Prompt,
	})

	return messages
}

func (p *AzureProvider) parseResponse(model string, resp *openAIResponse) (*types.ModelResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]

	return &types.ModelResponse{
		Content:      choice.Message.Content,
		Model:        model,
		Provider:     types.ProviderAzure,
		FinishReason: choice.FinishReason,
		TokensUsed: types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func defaultAzureModels() []types.ModelCapabilities {
	return []types.ModelCapabilities{
		{
			Provider:          types.ProviderAzure,
			ModelName:         "gpt-4o",
			FriendlyName:      "Azure GPT-4o",
			IntelligenceScore: 90,
			ContextWindow:     128000,
			MaxOutputTokens:   4096,
			SupportsStreaming: true,
		},
		{
			Provider:          types.ProviderAzure,
			ModelName:         "gpt-4o-mini",
			FriendlyName:      "Azure GPT-4o Mini",
			IntelligenceScore: 75,
			ContextWindow:     128000,
			MaxOutputTokens:   4096,
			SupportsStreaming: true,
		},
	}
}
