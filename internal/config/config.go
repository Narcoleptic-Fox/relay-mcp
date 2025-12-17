package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	    "strconv"
	    "strings"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
	)
	// Build-time variables (set via ldflags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Config holds all configuration
type Config struct {
	// Version info
	Version   string
	Commit    string
	BuildTime string

	// API Keys
	GeminiAPIKey     string
	OpenAIAPIKey     string
	AzureAPIKey      string
	AzureEndpoint    string
	XAIAPIKey        string
	DIALAPIKey       string
	DIALEndpoint     string
	OpenRouterAPIKey string
	CustomAPIURL     string

	// Defaults
	DefaultModel        string
	DefaultThinkingMode types.ThinkingMode
	LogLevel            string

	// Model restrictions
	GoogleAllowedModels []string
	OpenAIAllowedModels []string

	// Conversation settings
	MaxConversationTurns     int
	ConversationTimeoutHours int

	// Disabled tools
	DisabledTools []string

	// CLI paths
	GeminiCLIPath string
	ClaudeCLIPath string
	CodexCLIPath  string

	// Model registries (loaded from JSON files)
	ModelRegistries map[types.ProviderType][]types.ModelCapabilities

	// CLI client configs
	CLIClients map[string]CLIClientConfig
}

// CLIClientConfig defines a CLI client
type CLIClientConfig struct {
	Name           string             `json:"name"`
	Command        string             `json:"command"`
	AdditionalArgs []string           `json:"additional_args"`
	Roles          map[string]CLIRole `json:"roles"`
	Timeout        string             `json:"timeout,omitempty"`
}

// CLIRole defines a role for a CLI client
type CLIRole struct {
	PromptPath   string   `json:"prompt_path"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	Args         []string `json:"args,omitempty"`
}

// Load reads configuration from environment and files
func Load() (*Config, error) {
	cfg := &Config{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,

		// Environment variables
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		OpenAIAPIKey:     os.Getenv("OPENAI_API_KEY"),
		AzureAPIKey:      os.Getenv("AZURE_OPENAI_API_KEY"),
		AzureEndpoint:    os.Getenv("AZURE_OPENAI_ENDPOINT"),
		XAIAPIKey:        os.Getenv("XAI_API_KEY"),
		DIALAPIKey:       os.Getenv("DIAL_API_KEY"),
		DIALEndpoint:     os.Getenv("DIAL_ENDPOINT"),
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		CustomAPIURL:     os.Getenv("CUSTOM_API_URL"),

		DefaultModel:        getEnvOrDefault("DEFAULT_MODEL", "auto"),
		DefaultThinkingMode: types.ThinkingMode(getEnvOrDefault("DEFAULT_THINKING_MODE", "medium")),
		LogLevel:            getEnvOrDefault("LOG_LEVEL", "info"),

		MaxConversationTurns:     getEnvInt("MAX_CONVERSATION_TURNS", 50),
		ConversationTimeoutHours: getEnvInt("CONVERSATION_TIMEOUT_HOURS", 3),

		GeminiCLIPath: getEnvOrDefault("GEMINI_CLI_PATH", "gemini"),
		ClaudeCLIPath: getEnvOrDefault("CLAUDE_CLI_PATH", "claude"),
		CodexCLIPath:  getEnvOrDefault("CODEX_CLI_PATH", "codex"),

		ModelRegistries: make(map[types.ProviderType][]types.ModelCapabilities),
		CLIClients:      make(map[string]CLIClientConfig),
	}

	// Parse allowed models
	if v := os.Getenv("GOOGLE_ALLOWED_MODELS"); v != "" {
		cfg.GoogleAllowedModels = strings.Split(v, ",")
	}
	if v := os.Getenv("OPENAI_ALLOWED_MODELS"); v != "" {
		cfg.OpenAIAllowedModels = strings.Split(v, ",")
	}

	// Parse disabled tools
	if v := os.Getenv("DISABLED_TOOLS"); v != "" {
		cfg.DisabledTools = strings.Split(v, ",")
	}

	// Load model registries
	if err := cfg.loadModelRegistries(); err != nil {
		return nil, fmt.Errorf("loading model registries: %w", err)
	}

	// Load CLI client configs
	if err := cfg.loadCLIClients(); err != nil {
		return nil, fmt.Errorf("loading CLI clients: %w", err)
	}

	return cfg, nil
}

func (c *Config) loadModelRegistries() error {
	configDir := "configs/models"

	files := map[types.ProviderType]string{
		types.ProviderGemini:     "gemini.json",
		types.ProviderOpenAI:     "openai.json",
		types.ProviderAzure:      "azure.json",
		types.ProviderXAI:        "xai.json",
		types.ProviderDIAL:       "dial.json",
		types.ProviderOpenRouter: "openrouter.json",
		types.ProviderCustom:     "custom.json",
	}

	for provider, filename := range files {
		path := filepath.Join(configDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip if file doesn't exist
			}
			return fmt.Errorf("reading %s: %w", path, err)
		}

		var models []types.ModelCapabilities
		if err := json.Unmarshal(data, &models); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		c.ModelRegistries[provider] = models
	}

	return nil
}

func (c *Config) loadCLIClients() error {
	configDir := "configs/cli_clients"

	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No CLI clients configured
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(configDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		var client CLIClientConfig
		if err := json.Unmarshal(data, &client); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		c.CLIClients[client.Name] = client
	}

	return nil
}

// IsToolDisabled checks if a tool is disabled
func (c *Config) IsToolDisabled(name string) bool {
	for _, t := range c.DisabledTools {
		if strings.TrimSpace(t) == name {
			return true
		}
	}
	return false
}

// HasProvider checks if a provider is configured
func (c *Config) HasProvider(p types.ProviderType) bool {
	switch p {
	case types.ProviderGemini:
		return c.GeminiAPIKey != ""
	case types.ProviderOpenAI:
		return c.OpenAIAPIKey != ""
	case types.ProviderAzure:
		return c.AzureAPIKey != "" && c.AzureEndpoint != ""
	case types.ProviderXAI:
		return c.XAIAPIKey != ""
	case types.ProviderDIAL:
		return c.DIALAPIKey != "" && c.DIALEndpoint != ""
	case types.ProviderOpenRouter:
		return c.OpenRouterAPIKey != ""
	case types.ProviderCustom:
		return c.CustomAPIURL != ""
	default:
		return false
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
