# 01 - Project Structure & Setup

## Go Module Initialization

```bash
# Create project directory
mkdir pal-mcp && cd pal-mcp

# Initialize Go module
go mod init github.com/yourorg/pal-mcp

# Create directory structure
mkdir -p cmd/pal-mcp
mkdir -p internal/{server,providers,tools/simple,tools/workflow,clink/parsers,memory,config}
mkdir -p configs/{models,cli_clients}
mkdir -p prompts
```

## Complete Directory Structure

```
pal-mcp/
├── cmd/
│   └── pal-mcp/
│       └── main.go                 # Application entry point
│
├── internal/                       # Private application code
│   ├── server/
│   │   ├── server.go              # MCP server implementation
│   │   ├── handlers.go            # Request handlers
│   │   └── errors.go              # Error types
│   │
│   ├── providers/
│   │   ├── provider.go            # Provider interface
│   │   ├── registry.go            # Provider registry (singleton)
│   │   ├── capabilities.go        # Model capabilities
│   │   ├── gemini.go              # Google Gemini
│   │   ├── openai.go              # OpenAI
│   │   ├── openai_compat.go       # OpenAI-compatible base
│   │   ├── azure.go               # Azure OpenAI
│   │   ├── xai.go                 # X.AI Grok
│   │   ├── dial.go                # DIAL
│   │   ├── openrouter.go          # OpenRouter
│   │   └── custom.go              # Local models (Ollama)
│   │
│   ├── tools/
│   │   ├── tool.go                # Tool interface
│   │   ├── schema.go              # JSON schema generation
│   │   ├── simple/
│   │   │   ├── base.go            # SimpleTool base
│   │   │   ├── chat.go            # Chat tool
│   │   │   ├── apilookup.go       # API lookup
│   │   │   ├── challenge.go       # Challenge tool
│   │   │   ├── listmodels.go      # List models
│   │   │   └── version.go         # Version info
│   │   └── workflow/
│   │       ├── base.go            # WorkflowTool base
│   │       ├── thinkdeep.go       # Extended reasoning
│   │       ├── debug.go           # Debugging
│   │       ├── codereview.go      # Code review
│   │       ├── precommit.go       # Pre-commit
│   │       ├── planner.go         # Planning
│   │       ├── consensus.go       # Multi-model
│   │       ├── analyze.go         # Analysis
│   │       ├── refactor.go        # Refactoring
│   │       └── testgen.go         # Test generation
│   │
│   ├── clink/
│   │   ├── agent.go               # CLI agent interface
│   │   ├── registry.go            # Agent registry
│   │   ├── gemini.go              # Gemini CLI agent
│   │   ├── claude.go              # Claude Code agent
│   │   ├── codex.go               # Codex CLI agent
│   │   └── parsers/
│   │       ├── parser.go          # Parser interface
│   │       ├── gemini.go          # Gemini output parser
│   │       ├── claude.go          # Claude output parser
│   │       └── codex.go           # Codex output parser
│   │
│   ├── memory/
│   │   ├── conversation.go        # Conversation memory
│   │   ├── thread.go              # Thread context
│   │   └── turn.go                # Conversation turn
│   │
│   ├── config/
│   │   ├── config.go              # Configuration loading
│   │   ├── env.go                 # Environment variables
│   │   └── models.go              # Model registry loading
│   │
│   └── utils/
│       ├── files.go               # File utilities
│       ├── security.go            # Path validation
│       └── tokens.go              # Token counting
│
├── configs/                        # Configuration files
│   ├── models/
│   │   ├── gemini.json
│   │   ├── openai.json
│   │   ├── azure.json
│   │   ├── xai.json
│   │   ├── dial.json
│   │   ├── openrouter.json
│   │   └── custom.json
│   └── cli_clients/
│       ├── gemini.json
│       ├── claude.json
│       └── codex.json
│
├── prompts/                        # System prompts
│   ├── chat.txt
│   ├── thinkdeep.txt
│   ├── debug.txt
│   ├── codereview.txt
│   ├── precommit.txt
│   ├── planner.txt
│   ├── consensus.txt
│   ├── analyze.txt
│   ├── refactor.txt
│   ├── testgen.txt
│   └── clink/
│       ├── default.txt
│       ├── planner.txt
│       └── codereviewer.txt
│
├── scripts/
│   ├── build.sh                   # Build script
│   └── release.sh                 # Release script
│
├── .env.example                   # Example environment
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## go.mod

```go
module github.com/yourorg/pal-mcp

go 1.22

require (
    // MCP SDK - primary dependency for MCP protocol
    github.com/mark3labs/mcp-go v0.17.0

    // UUID generation for thread IDs
    github.com/google/uuid v1.6.0

    // Environment variable loading
    github.com/joho/godotenv v1.5.1
)
```

## Entry Point (cmd/pal-mcp/main.go)

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/joho/godotenv"
    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/providers"
    "github.com/yourorg/pal-mcp/internal/server"
)

func main() {
    // Load .env file if present
    _ = godotenv.Load()

    // Initialize structured logging
    logLevel := slog.LevelInfo
    if os.Getenv("LOG_LEVEL") == "debug" {
        logLevel = slog.LevelDebug
    }

    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: logLevel,
    }))
    slog.SetDefault(logger)

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load configuration", "error", err)
        os.Exit(1)
    }

    // Initialize provider registry
    registry := providers.NewRegistry(cfg)
    if err := registry.Initialize(); err != nil {
        slog.Error("failed to initialize providers", "error", err)
        os.Exit(1)
    }

    // Create MCP server
    srv := server.New(cfg, registry)

    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        slog.Info("shutting down...")
        cancel()
    }()

    // Run server (blocks on stdio)
    slog.Info("starting PAL MCP server", "version", cfg.Version)
    if err := srv.Run(ctx); err != nil {
        slog.Error("server error", "error", err)
        os.Exit(1)
    }
}
```

## Makefile

```makefile
.PHONY: build run test clean lint install

# Binary name
BINARY=pal-mcp

# Build info
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-s -w \
    -X github.com/yourorg/pal-mcp/internal/config.Version=$(VERSION) \
    -X github.com/yourorg/pal-mcp/internal/config.Commit=$(COMMIT) \
    -X github.com/yourorg/pal-mcp/internal/config.BuildTime=$(BUILD_TIME)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/pal-mcp

# Build for all platforms
build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 ./cmd/pal-mcp
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64 ./cmd/pal-mcp
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-darwin-amd64 ./cmd/pal-mcp
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-darwin-arm64 ./cmd/pal-mcp
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-windows-amd64.exe ./cmd/pal-mcp

# Run the server
run: build
	./$(BINARY)

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-*
	rm -f coverage.out coverage.html

# Install to GOPATH/bin
install: build
	cp $(BINARY) $(GOPATH)/bin/

# Download dependencies
deps:
	go mod download
	go mod tidy

# Update dependencies
deps-update:
	go get -u ./...
	go mod tidy
```

## .gitignore

```gitignore
# Binaries
pal-mcp
pal-mcp-*
*.exe

# Environment
.env
.env.local

# IDE
.idea/
.vscode/
*.swp
*.swo

# Build
/dist/
/coverage.out
/coverage.html

# OS
.DS_Store
Thumbs.db

# Logs
logs/
*.log
```

## .env.example

```bash
# =============================================================================
# PAL MCP Server Configuration
# =============================================================================

# -----------------------------------------------------------------------------
# API Keys (at least one required)
# -----------------------------------------------------------------------------
GEMINI_API_KEY=
OPENAI_API_KEY=
AZURE_OPENAI_API_KEY=
AZURE_OPENAI_ENDPOINT=
XAI_API_KEY=
DIAL_API_KEY=
DIAL_ENDPOINT=
OPENROUTER_API_KEY=

# Custom/Local provider (Ollama, vLLM, LM Studio)
CUSTOM_API_URL=http://localhost:11434/v1

# -----------------------------------------------------------------------------
# Default Settings
# -----------------------------------------------------------------------------

# Default model for auto-selection (auto|pro|flash|gpt-5|o3|etc.)
DEFAULT_MODEL=auto

# Default thinking mode for thinkdeep tool (minimal|low|medium|high|max)
DEFAULT_THINKING_MODE=medium

# Log level (debug|info|warn|error)
LOG_LEVEL=info

# -----------------------------------------------------------------------------
# Model Restrictions (optional)
# -----------------------------------------------------------------------------

# Comma-separated list of allowed models per provider
# GOOGLE_ALLOWED_MODELS=gemini-2.5-pro,gemini-2.5-flash
# OPENAI_ALLOWED_MODELS=gpt-5,o3,o4-mini

# -----------------------------------------------------------------------------
# Conversation Settings
# -----------------------------------------------------------------------------

# Maximum turns per conversation thread
MAX_CONVERSATION_TURNS=50

# Thread TTL in hours
CONVERSATION_TIMEOUT_HOURS=3

# -----------------------------------------------------------------------------
# Disabled Tools (optional)
# -----------------------------------------------------------------------------

# Comma-separated list of tools to disable
# DISABLED_TOOLS=analyze,refactor,testgen

# -----------------------------------------------------------------------------
# CLI Clients (clink)
# -----------------------------------------------------------------------------

# Override CLI executable paths
# GEMINI_CLI_PATH=/usr/local/bin/gemini
# CLAUDE_CLI_PATH=/usr/local/bin/claude
# CODEX_CLI_PATH=/usr/local/bin/codex
```

## Core Type Definitions (internal/types/types.go)

```go
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
    Provider               ProviderType `json:"provider"`
    ModelName              string       `json:"model_name"`
    FriendlyName           string       `json:"friendly_name"`
    IntelligenceScore      int          `json:"intelligence_score"` // 1-100
    Aliases                []string     `json:"aliases"`

    // Limits
    ContextWindow          int          `json:"context_window"`
    MaxOutputTokens        int          `json:"max_output_tokens"`
    MaxThinkingTokens      int          `json:"max_thinking_tokens"`

    // Features
    SupportsExtendedThinking bool       `json:"supports_extended_thinking"`
    SupportsSystemPrompts    bool       `json:"supports_system_prompts"`
    SupportsStreaming        bool       `json:"supports_streaming"`
    SupportsVision           bool       `json:"supports_vision"`
    AllowCodeGeneration      bool       `json:"allow_code_generation"`

    // Temperature constraints
    MinTemperature         *float64     `json:"min_temperature,omitempty"`
    MaxTemperature         *float64     `json:"max_temperature,omitempty"`
}

// ModelResponse is the unified response from any provider
type ModelResponse struct {
    Content           string            `json:"content"`
    Model             string            `json:"model"`
    Provider          ProviderType      `json:"provider"`
    TokensUsed        TokenUsage        `json:"tokens_used"`
    FinishReason      string            `json:"finish_reason,omitempty"`
    Metadata          map[string]any    `json:"metadata,omitempty"`
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
```

## Configuration Loader (internal/config/config.go)

```go
package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"

    "github.com/yourorg/pal-mcp/internal/types"
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
    GoogleAllowedModels  []string
    OpenAIAllowedModels  []string

    // Conversation settings
    MaxConversationTurns    int
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
    Name           string            `json:"name"`
    Command        string            `json:"command"`
    AdditionalArgs []string          `json:"additional_args"`
    Roles          map[string]CLIRole `json:"roles"`
}

// CLIRole defines a role for a CLI client
type CLIRole struct {
    PromptPath string `json:"prompt_path"`
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
```

## Build & Test

```bash
# Download dependencies
make deps

# Build binary
make build

# Run tests
make test

# Build for all platforms
make build-all

# Run the server
make run
```

## IDE Setup (VS Code)

`.vscode/settings.json`:
```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "go.formatTool": "goimports",
    "editor.formatOnSave": true,
    "[go]": {
        "editor.codeActionsOnSave": {
            "source.organizeImports": true
        }
    }
}
```

## Next Steps

Continue to [02-MCP-SERVER.md](./02-MCP-SERVER.md) for MCP protocol implementation.
