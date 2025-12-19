package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	"github.com/Narcoleptic-Fox/relay-mcp/internal/types"
	"github.com/Narcoleptic-Fox/relay-mcp/internal/utils"
)
	const (
	geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// GeminiProvider implements Provider for Google Gemini
type GeminiProvider struct {
	*BaseProvider
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(cfg *config.Config) (*GeminiProvider, error) {
	if cfg.GeminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not configured")
	}

	models := cfg.ModelRegistries[types.ProviderGemini]
	if len(models) == 0 {
		models = defaultGeminiModels()
	}

	return &GeminiProvider{
		BaseProvider: NewBaseProvider(types.ProviderGemini, models),
		apiKey:       cfg.GeminiAPIKey,
		baseURL:      geminiBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

func (p *GeminiProvider) IsConfigured() bool {
	return p.apiKey != ""
}

func (p *GeminiProvider) CountTokens(text string, modelName string) (int, error) {
	// Rough estimate: 4 chars per token
	return len(text) / 4, nil
}

// GenerateContent calls the Gemini API
func (p *GeminiProvider) GenerateContent(ctx context.Context, req *GenerateRequest) (*types.ModelResponse, error) {
	modelName := p.ResolveModelName(req.Model)

	// Build request body
	body := map[string]any{
		"contents": p.buildContents(req),
	}

	// Add generation config
	genConfig := map[string]any{}
	if req.Temperature > 0 {
		genConfig["temperature"] = req.Temperature
	}
	if req.MaxOutputTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxOutputTokens
	}

	// Add thinking config for supported models
	caps, _ := p.GetCapabilities(modelName)
	if caps != nil && caps.SupportsExtendedThinking && req.ThinkingMode != "" {
		genConfig["thinkingConfig"] = map[string]any{
			"thinkingBudget": p.getThinkingBudget(req.ThinkingMode, req.ThinkingBudget),
		}
	}

	if len(genConfig) > 0 {
		body["generationConfig"] = genConfig
	}

	// Add system instruction
	if req.SystemPrompt != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]any{
				{"text": req.SystemPrompt},
			},
		}
	}

	// Make request
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, modelName, p.apiKey)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

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
	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return p.parseResponse(modelName, &geminiResp)
}

// buildContents converts conversation history to Gemini format
func (p *GeminiProvider) buildContents(req *GenerateRequest) []map[string]any {
	var contents []map[string]any

	// Add conversation history
	for _, turn := range req.ConversationHistory {
		role := "user"
		if turn.Role == "assistant" {
			role = "model"
		}

		parts := []map[string]any{
			{"text": turn.Content},
		}

		contents = append(contents, map[string]any{
			"role":  role,
			"parts": parts,
		})
	}

	// Add current prompt
	parts := []map[string]any{
		{"text": req.Prompt},
	}

	// Add images if present (supports file paths, data URIs, and base64)
	if len(req.Images) > 0 {
		processedImages := utils.ProcessImages(req.Images)
		for _, imgData := range processedImages {
			parts = append(parts, map[string]any{
				"inlineData": map[string]any{
					"mimeType": imgData.MimeType,
					"data":     imgData.Base64,
				},
			})
		}
	}

	contents = append(contents, map[string]any{
		"role":  "user",
		"parts": parts,
	})

	return contents
}

func (p *GeminiProvider) getThinkingBudget(mode types.ThinkingMode, custom int) int {
	if custom > 0 {
		return custom
	}

	switch mode {
	case types.ThinkingMinimal:
		return 1024
	case types.ThinkingLow:
		return 4096
	case types.ThinkingMedium:
		return 8192
	case types.ThinkingHigh:
		return 16384
	case types.ThinkingMax:
		return 32768
	default:
		return 8192
	}
}

func (p *GeminiProvider) parseResponse(model string, resp *geminiResponse) (*types.ModelResponse, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	var content string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			content += part.Text
		}
	}

	return &types.ModelResponse{
		Content:      content,
		Model:        model,
		Provider:     types.ProviderGemini,
		FinishReason: candidate.FinishReason,
		TokensUsed: types.TokenUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

// Gemini API response types
type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata geminiUsage       `json:"usageMetadata"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func defaultGeminiModels() []types.ModelCapabilities {
	return []types.ModelCapabilities{
		{
			Provider:                 types.ProviderGemini,
			ModelName:                "gemini-2.5-pro",
			FriendlyName:             "Gemini 2.5 Pro",
			IntelligenceScore:        100,
			Aliases:                  []string{"pro", "gemini-pro"},
			ContextWindow:            1000000,
			MaxOutputTokens:          65536,
			MaxThinkingTokens:        32768,
			SupportsExtendedThinking: true,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           true,
			AllowCodeGeneration:      true,
		},
		{
			Provider:                 types.ProviderGemini,
			ModelName:                "gemini-2.5-flash",
			FriendlyName:             "Gemini 2.5 Flash",
			IntelligenceScore:        61,
			Aliases:                  []string{"flash", "gemini-flash"},
			ContextWindow:            1000000,
			MaxOutputTokens:          65536,
			MaxThinkingTokens:        24576,
			SupportsExtendedThinking: true,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           true,
			AllowCodeGeneration:      true,
		},
		{
			Provider:                 types.ProviderGemini,
			ModelName:                "gemini-2.0-flash",
			FriendlyName:             "Gemini 2.0 Flash",
			IntelligenceScore:        56,
			Aliases:                  []string{"flash-2.0"},
			ContextWindow:            1000000,
			MaxOutputTokens:          8192,
			SupportsExtendedThinking: true,
			SupportsSystemPrompts:    true,
			SupportsStreaming:        true,
			SupportsVision:           true,
			AllowCodeGeneration:      true,
		},
	}
}
