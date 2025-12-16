# 03 - Provider System

## Overview

The provider system abstracts AI model access behind a unified interface. This allows tools to work with any supported model without knowing provider-specific details.

## Provider Interface (internal/providers/provider.go)

```go
package providers

import (
    "context"

    "github.com/yourorg/pal-mcp/internal/types"
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
    ThinkingMode    types.ThinkingMode
    ThinkingBudget  int

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
```

## Provider Registry (internal/providers/registry.go)

```go
package providers

import (
    "fmt"
    "log/slog"
    "sort"
    "sync"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/types"
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
        return fmt.Errorf("no providers configured")
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
```

## Gemini Provider (internal/providers/gemini.go)

```go
package providers

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/types"
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

    // Add images if present
    for _, img := range req.Images {
        parts = append(parts, map[string]any{
            "inlineData": map[string]any{
                "mimeType": "image/jpeg",
                "data":     img,
            },
        })
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
```

## OpenAI-Compatible Base (internal/providers/openai_compat.go)

```go
package providers

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/yourorg/pal-mcp/internal/types"
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
    Index        int            `json:"index"`
    Message      openAIMessage  `json:"message"`
    FinishReason string         `json:"finish_reason"`
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
```

## OpenAI Provider (internal/providers/openai.go)

```go
package providers

import (
    "fmt"
    "time"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/types"
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
```

## Custom/Local Provider (internal/providers/custom.go)

```go
package providers

import (
    "fmt"
    "time"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/types"
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
```

## OpenRouter Provider (internal/providers/openrouter.go)

```go
package providers

import (
    "fmt"
    "time"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/types"
)

const openRouterBaseURL = "https://openrouter.ai/api/v1"

// OpenRouterProvider implements Provider for OpenRouter (catch-all)
type OpenRouterProvider struct {
    *OpenAICompatProvider
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(cfg *config.Config) (*OpenRouterProvider, error) {
    if cfg.OpenRouterAPIKey == "" {
        return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
    }

    models := cfg.ModelRegistries[types.ProviderOpenRouter]
    if len(models) == 0 {
        models = defaultOpenRouterModels()
    }

    return &OpenRouterProvider{
        OpenAICompatProvider: NewOpenAICompatProvider(
            types.ProviderOpenRouter,
            cfg.OpenRouterAPIKey,
            openRouterBaseURL,
            models,
            5*time.Minute,
        ),
    }, nil
}

func defaultOpenRouterModels() []types.ModelCapabilities {
    // OpenRouter provides access to many models
    return []types.ModelCapabilities{
        {
            Provider:                 types.ProviderOpenRouter,
            ModelName:                "anthropic/claude-3.5-sonnet",
            FriendlyName:             "Claude 3.5 Sonnet",
            IntelligenceScore:        90,
            Aliases:                  []string{"sonnet", "claude-sonnet"},
            ContextWindow:            200000,
            MaxOutputTokens:          8192,
            SupportsExtendedThinking: false,
            SupportsSystemPrompts:    true,
            SupportsStreaming:        true,
            SupportsVision:           true,
            AllowCodeGeneration:      true,
        },
        {
            Provider:                 types.ProviderOpenRouter,
            ModelName:                "meta-llama/llama-3.3-70b-instruct",
            FriendlyName:             "Llama 3.3 70B",
            IntelligenceScore:        75,
            Aliases:                  []string{"llama-70b"},
            ContextWindow:            128000,
            MaxOutputTokens:          8192,
            SupportsExtendedThinking: false,
            SupportsSystemPrompts:    true,
            SupportsStreaming:        true,
            SupportsVision:           false,
            AllowCodeGeneration:      true,
        },
    }
}
```

## Error Types (internal/providers/errors.go)

```go
package providers

import (
    "fmt"

    "github.com/yourorg/pal-mcp/internal/types"
)

// ErrModelNotFound indicates a model wasn't found
type ErrModelNotFound struct {
    Model    string
    Provider types.ProviderType
}

func (e ErrModelNotFound) Error() string {
    if e.Provider != "" {
        return fmt.Sprintf("model %q not found in provider %s", e.Model, e.Provider)
    }
    return fmt.Sprintf("model %q not found in any provider", e.Model)
}

// ErrProviderNotConfigured indicates a provider isn't configured
type ErrProviderNotConfigured struct {
    Provider types.ProviderType
}

func (e ErrProviderNotConfigured) Error() string {
    return fmt.Sprintf("provider %s not configured", e.Provider)
}

// ErrAPIError indicates an API error
type ErrAPIError struct {
    Provider   types.ProviderType
    StatusCode int
    Message    string
}

func (e ErrAPIError) Error() string {
    return fmt.Sprintf("%s API error (%d): %s", e.Provider, e.StatusCode, e.Message)
}
```

## Model Registry JSON (configs/models/gemini.json)

```json
[
    {
        "provider": "gemini",
        "model_name": "gemini-2.5-pro",
        "friendly_name": "Gemini 2.5 Pro",
        "intelligence_score": 100,
        "aliases": ["pro", "gemini-pro"],
        "context_window": 1000000,
        "max_output_tokens": 65536,
        "max_thinking_tokens": 32768,
        "supports_extended_thinking": true,
        "supports_system_prompts": true,
        "supports_streaming": true,
        "supports_vision": true,
        "allow_code_generation": true
    },
    {
        "provider": "gemini",
        "model_name": "gemini-2.5-flash",
        "friendly_name": "Gemini 2.5 Flash",
        "intelligence_score": 61,
        "aliases": ["flash", "gemini-flash"],
        "context_window": 1000000,
        "max_output_tokens": 65536,
        "max_thinking_tokens": 24576,
        "supports_extended_thinking": true,
        "supports_system_prompts": true,
        "supports_streaming": true,
        "supports_vision": true,
        "allow_code_generation": true
    }
]
```

## Next Steps

Continue to [04-TOOLS.md](./04-TOOLS.md) for the tool system implementation.
