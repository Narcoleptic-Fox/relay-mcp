package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/types"
)

// OpenAICompatProvider is a base for OpenAI-compatible APIs
type OpenAICompatProvider struct {
	*BaseProvider
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAICompatProvider creates a new OpenAI-compatible provider
func NewOpenAICompatProvider(
	pt types.ProviderType,
	apiKey string,
	baseURL string,
	models []types.ModelCapabilities,
	timeout time.Duration,
) *OpenAICompatProvider {
	return &OpenAICompatProvider{
		BaseProvider: NewBaseProvider(pt, models),
		apiKey:       apiKey,
		baseURL:      baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *OpenAICompatProvider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *OpenAICompatProvider) CountTokens(text string, modelName string) (int, error) {
	// Rough estimate: 4 chars per token
	return len(text) / 4, nil
}

// GenerateContent calls an OpenAI-compatible API
func (p *OpenAICompatProvider) GenerateContent(ctx context.Context, req *GenerateRequest) (*types.ModelResponse, error) {
	modelName := p.ResolveModelName(req.Model)

	// Build messages
	messages := p.buildMessages(req)

	// Build request body
	body := map[string]any{
		"model":    modelName,
		"messages": messages,
	}

	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxOutputTokens > 0 {
		body["max_tokens"] = req.MaxOutputTokens
	}

	// Make request
	url := p.baseURL + "/chat/completions"

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return p.parseResponse(modelName, &oaiResp)
}

func (p *OpenAICompatProvider) buildMessages(req *GenerateRequest) []map[string]any {
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

func (p *OpenAICompatProvider) parseResponse(model string, resp *openAIResponse) (*types.ModelResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]

	return &types.ModelResponse{
		Content:      choice.Message.Content,
		Model:        model,
		Provider:     p.providerType,
		FinishReason: choice.FinishReason,
		TokensUsed: types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// OpenAI API response types
type openAIResponse struct {
	ID      string         `json:"id"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
