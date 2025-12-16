package types

import "time"

// ProviderType represents an AI provider
type ProviderType string

const (
	ProviderGemini     ProviderType = "gemini"
	ProviderOpenAI     ProviderType = "openai"
	ProviderAzure      ProviderType = "azure"
	ProviderXAI        ProviderType = "xai"
	ProviderDIAL       ProviderType = "dial"
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderCustom     ProviderType = "custom"
)

// ModelCapabilities defines what a model can do
type ModelCapabilities struct {
	Provider          ProviderType `json:"provider"`
	ModelName         string       `json:"model_name"`
	FriendlyName      string       `json:"friendly_name"`
	IntelligenceScore int          `json:"intelligence_score"` // 1-100
	Aliases           []string     `json:"aliases"`

	// Limits
	ContextWindow     int `json:"context_window"`
	MaxOutputTokens   int `json:"max_output_tokens"`
	MaxThinkingTokens int `json:"max_thinking_tokens"`

	// Features
	SupportsExtendedThinking bool `json:"supports_extended_thinking"`
	SupportsSystemPrompts    bool `json:"supports_system_prompts"`
	SupportsStreaming        bool `json:"supports_streaming"`
	SupportsVision           bool `json:"supports_vision"`
	AllowCodeGeneration      bool `json:"allow_code_generation"`

	// Temperature constraints
	MinTemperature *float64 `json:"min_temperature,omitempty"`
	MaxTemperature *float64 `json:"max_temperature,omitempty"`
}

// ModelResponse is the unified response from any provider
type ModelResponse struct {
	Content      string         `json:"content"`
	Model        string         `json:"model"`
	Provider     ProviderType   `json:"provider"`
	TokensUsed   TokenUsage     `json:"tokens_used"`
	FinishReason string         `json:"finish_reason,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	ThinkingTokens   int `json:"thinking_tokens,omitempty"`
}

// ConversationTurn represents a single turn in a conversation
type ConversationTurn struct {
	Role          string    `json:"role"` // "user" or "assistant"
	Content       string    `json:"content"`
	Timestamp     time.Time `json:"timestamp"`
	Files         []string  `json:"files,omitempty"`
	Images        []string  `json:"images,omitempty"`
	ToolName      string    `json:"tool_name,omitempty"`
	ModelProvider string    `json:"model_provider,omitempty"`
	ModelName     string    `json:"model_name,omitempty"`
}

// ThreadContext holds conversation state
type ThreadContext struct {
	ThreadID       string             `json:"thread_id"`
	ParentThreadID string             `json:"parent_thread_id,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	LastUpdatedAt  time.Time          `json:"last_updated_at"`
	ToolName       string             `json:"tool_name"`
	Turns          []ConversationTurn `json:"turns"`
}

// ThinkingMode for extended reasoning
type ThinkingMode string

const (
	ThinkingMinimal ThinkingMode = "minimal"
	ThinkingLow     ThinkingMode = "low"
	ThinkingMedium  ThinkingMode = "medium"
	ThinkingHigh    ThinkingMode = "high"
	ThinkingMax     ThinkingMode = "max"
)

// ConfidenceLevel for workflow tools
type ConfidenceLevel string

const (
	ConfidenceExploring     ConfidenceLevel = "exploring"
	ConfidenceLow           ConfidenceLevel = "low"
	ConfidenceMedium        ConfidenceLevel = "medium"
	ConfidenceHigh          ConfidenceLevel = "high"
	ConfidenceVeryHigh      ConfidenceLevel = "very_high"
	ConfidenceAlmostCertain ConfidenceLevel = "almost_certain"
	ConfidenceCertain       ConfidenceLevel = "certain"
)

// Stance for consensus tool
type Stance string

const (
	StanceFor     Stance = "for"
	StanceAgainst Stance = "against"
	StanceNeutral Stance = "neutral"
)
