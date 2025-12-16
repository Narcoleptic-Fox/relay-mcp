# 08 - Configuration & Deployment

## Overview

This document covers configuration, building, testing, and deploying the Go PAL MCP Server.

## Configuration Sources

```
Priority (highest to lowest):
1. Environment variables
2. .env file in working directory
3. Config files in configs/
4. Built-in defaults
```

## Environment Variables

### API Keys

```bash
# At least one provider must be configured
GEMINI_API_KEY=           # Google Gemini
OPENAI_API_KEY=           # OpenAI
AZURE_OPENAI_API_KEY=     # Azure OpenAI
AZURE_OPENAI_ENDPOINT=    # Azure endpoint URL
XAI_API_KEY=              # X.AI Grok
DIAL_API_KEY=             # DIAL
DIAL_ENDPOINT=            # DIAL endpoint URL
OPENROUTER_API_KEY=       # OpenRouter (catch-all)
CUSTOM_API_URL=           # Local models (Ollama, vLLM)
```

### Server Settings

```bash
# Model selection
DEFAULT_MODEL=auto        # auto|pro|flash|gpt-5|o3|etc.
DEFAULT_THINKING_MODE=medium  # minimal|low|medium|high|max

# Logging
LOG_LEVEL=info            # debug|info|warn|error
LOG_FORMAT=json           # json|text

# Conversation limits
MAX_CONVERSATION_TURNS=50
CONVERSATION_TIMEOUT_HOURS=3
```

### Model Restrictions

```bash
# Restrict which models can be used (comma-separated)
GOOGLE_ALLOWED_MODELS=gemini-2.5-pro,gemini-2.5-flash
OPENAI_ALLOWED_MODELS=gpt-5,o3

# Disable specific tools (comma-separated)
DISABLED_TOOLS=analyze,refactor,testgen
```

### CLI Clients

```bash
# Override CLI executable paths
GEMINI_CLI_PATH=/usr/local/bin/gemini
CLAUDE_CLI_PATH=/usr/local/bin/claude
CODEX_CLI_PATH=/usr/local/bin/codex
```

## Configuration File Structure

```
configs/
├── models/
│   ├── gemini.json       # Gemini model definitions
│   ├── openai.json       # OpenAI model definitions
│   ├── azure.json        # Azure OpenAI deployments
│   ├── xai.json          # X.AI Grok models
│   ├── dial.json         # DIAL models
│   ├── openrouter.json   # OpenRouter catalog
│   └── custom.json       # Local model definitions
└── cli_clients/
    ├── gemini.json       # Gemini CLI config
    ├── claude.json       # Claude CLI config
    └── codex.json        # Codex CLI config
```

## Model Configuration Example

### configs/models/gemini.json

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

## CLI Client Configuration

### configs/cli_clients/gemini.json

```json
{
    "name": "gemini",
    "command": "gemini",
    "additional_args": ["--yolo"],
    "timeout": "5m",
    "roles": {
        "default": {
            "prompt_path": "clink/default.txt"
        },
        "planner": {
            "prompt_path": "clink/planner.txt"
        },
        "codereviewer": {
            "prompt_path": "clink/codereviewer.txt"
        }
    }
}
```

## Building

### Development Build

```bash
# Simple build
go build -o pal-mcp ./cmd/pal-mcp

# Build with race detection (for development)
go build -race -o pal-mcp ./cmd/pal-mcp
```

### Production Build

```bash
# Optimized build with version info
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-s -w \
    -X github.com/yourorg/pal-mcp/internal/config.Version=$VERSION \
    -X github.com/yourorg/pal-mcp/internal/config.Commit=$COMMIT \
    -X github.com/yourorg/pal-mcp/internal/config.BuildTime=$BUILD_TIME" \
    -o pal-mcp ./cmd/pal-mcp
```

### Cross-Platform Builds

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o pal-mcp-linux-amd64 ./cmd/pal-mcp

# Linux ARM64 (Raspberry Pi, AWS Graviton)
GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o pal-mcp-linux-arm64 ./cmd/pal-mcp

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o pal-mcp-darwin-amd64 ./cmd/pal-mcp

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o pal-mcp-darwin-arm64 ./cmd/pal-mcp

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o pal-mcp-windows-amd64.exe ./cmd/pal-mcp
```

### Build Script (scripts/build.sh)

```bash
#!/bin/bash
set -e

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS="-s -w \
    -X github.com/yourorg/pal-mcp/internal/config.Version=$VERSION \
    -X github.com/yourorg/pal-mcp/internal/config.Commit=$COMMIT \
    -X github.com/yourorg/pal-mcp/internal/config.BuildTime=$BUILD_TIME"

echo "Building PAL MCP Server $VERSION ($COMMIT)"

# Build for current platform
go build -ldflags "$LDFLAGS" -o pal-mcp ./cmd/pal-mcp

echo "Build complete: pal-mcp"
```

## Testing

### Run All Tests

```bash
go test -v ./...
```

### Run Tests with Coverage

```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Run Specific Tests

```bash
# Run tests for a specific package
go test -v ./internal/providers/...

# Run a specific test
go test -v -run TestGeminiProvider_GenerateContent ./internal/providers/

# Run tests matching a pattern
go test -v -run ".*Consensus.*" ./...
```

### Integration Tests

```bash
# Set up test environment
export GEMINI_API_KEY="test-key"
export TEST_MODE=integration

# Run integration tests
go test -v -tags=integration ./...
```

## Running

### Direct Execution

```bash
# With environment variables
GEMINI_API_KEY=your-key ./pal-mcp

# With .env file
./pal-mcp
```

### With Claude Code

```bash
# Configure MCP server
claude config set mcpServers.pal.command /path/to/pal-mcp

# Or use command line
claude --mcp-server /path/to/pal-mcp
```

### Claude Code Configuration File

**Linux/macOS:** `~/.config/claude/mcp_servers.json`
**Windows:** `%APPDATA%\Claude\mcp_servers.json`

```json
{
    "pal": {
        "command": "/path/to/pal-mcp",
        "args": [],
        "env": {
            "GEMINI_API_KEY": "your-gemini-key",
            "OPENAI_API_KEY": "your-openai-key",
            "LOG_LEVEL": "info"
        }
    }
}
```

## Docker Deployment

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o pal-mcp ./cmd/pal-mcp

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary
COPY --from=builder /app/pal-mcp .

# Copy config files
COPY configs/ ./configs/
COPY prompts/ ./prompts/

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

ENTRYPOINT ["./pal-mcp"]
```

### docker-compose.yml

```yaml
version: '3.8'

services:
  pal-mcp:
    build: .
    environment:
      - GEMINI_API_KEY=${GEMINI_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - LOG_LEVEL=info
    volumes:
      - ./configs:/app/configs:ro
      - ./prompts:/app/prompts:ro
    stdin_open: true
    tty: true
```

## GitHub Actions CI/CD

### .github/workflows/ci.yml

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Download dependencies
        run: go mod download

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  build:
    runs-on: ubuntu-latest
    needs: [test, lint]
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build
        run: go build -v ./...
```

### .github/workflows/release.yml

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### .goreleaser.yml

```yaml
version: 1

project_name: pal-mcp

before:
  hooks:
    - go mod tidy

builds:
  - main: ./cmd/pal-mcp
    binary: pal-mcp
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/yourorg/pal-mcp/internal/config.Version={{.Version}}
      - -X github.com/yourorg/pal-mcp/internal/config.Commit={{.Commit}}
      - -X github.com/yourorg/pal-mcp/internal/config.BuildTime={{.Date}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE
      - configs/**/*
      - prompts/**/*

checksum:
  name_template: 'checksums.txt'

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
```

## Logging

### Structured Logging

```go
// Logs are output to stderr in JSON format
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"starting PAL MCP server","version":"1.0.0"}
{"time":"2024-01-15T10:30:01Z","level":"INFO","msg":"tool call","name":"chat","duration_ms":1234}
```

### Log Levels

| Level | Description |
|-------|-------------|
| debug | Detailed debugging information |
| info | Normal operational messages |
| warn | Warning conditions |
| error | Error conditions |

### Viewing Logs

```bash
# Follow logs (when running directly)
./pal-mcp 2>&1 | jq .

# Pretty print logs
./pal-mcp 2>&1 | jq -r '[.time, .level, .msg] | @tsv'
```

## Monitoring

### Health Check

The server responds to `ping` MCP method:

```json
{"jsonrpc":"2.0","id":1,"method":"ping"}
→ {"jsonrpc":"2.0","id":1,"result":{}}
```

### Metrics (Future)

Consider adding Prometheus metrics:

```go
// internal/metrics/metrics.go
var (
    ToolCallsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "pal_tool_calls_total",
            Help: "Total number of tool calls",
        },
        []string{"tool", "status"},
    )

    ToolCallDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "pal_tool_call_duration_seconds",
            Help:    "Tool call duration in seconds",
            Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
        },
        []string{"tool"},
    )
)
```

## Security Considerations

### API Key Management

1. **Never commit API keys** to version control
2. Use environment variables or secret management
3. Consider using a secrets manager (HashiCorp Vault, AWS Secrets Manager)

### File Access

1. All file paths are validated
2. Path traversal is blocked
3. Symlinks are resolved and validated
4. Binary files are excluded

### Network Security

1. Use HTTPS for all API calls
2. Validate SSL certificates
3. Set appropriate timeouts
4. Implement rate limiting for providers

## Troubleshooting

### Common Issues

**Server won't start:**
```bash
# Check if required environment variables are set
env | grep -E "(GEMINI|OPENAI|AZURE)"

# Verify config files exist
ls -la configs/
```

**Tool execution fails:**
```bash
# Enable debug logging
LOG_LEVEL=debug ./pal-mcp

# Check specific provider
# Look for "provider error" in logs
```

**CLI agent not found:**
```bash
# Verify CLI is installed
which gemini
which claude
which codex

# Check PATH
echo $PATH
```

**Memory issues:**
```bash
# Monitor memory usage
top -p $(pgrep pal-mcp)

# Check thread count
MAX_CONVERSATION_TURNS=25 ./pal-mcp
```

## Quick Reference

### Build Commands

```bash
make build          # Build for current platform
make build-all      # Build for all platforms
make test           # Run tests
make lint           # Run linter
make clean          # Clean build artifacts
```

### Run Commands

```bash
./pal-mcp                           # Run with .env file
GEMINI_API_KEY=x ./pal-mcp         # Run with env var
LOG_LEVEL=debug ./pal-mcp          # Debug mode
```

### Test Commands

```bash
# Quick test
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./pal-mcp

# Test specific tool
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"version","arguments":{}}}' | ./pal-mcp
```

---

## Summary

This completes the Go rewrite architecture documentation. The documents cover:

1. **00-INDEX.md** - Overview and navigation
2. **01-PROJECT-SETUP.md** - Project structure and dependencies
3. **02-MCP-SERVER.md** - MCP protocol implementation
4. **03-PROVIDERS.md** - AI provider abstraction
5. **04-TOOLS.md** - Tool system architecture
6. **05-CONVERSATION-MEMORY.md** - Thread-based memory
7. **06-CLINK.md** - CLI linking for multi-CLI orchestration
8. **07-CONSENSUS-WORKFLOWS.md** - Multi-model consensus
9. **08-CONFIG-DEPLOY.md** - Configuration and deployment

**Estimated Implementation Time:** 6-8 weeks for full feature parity
**Recommended Starting Point:** Follow documents in order, starting with project setup
