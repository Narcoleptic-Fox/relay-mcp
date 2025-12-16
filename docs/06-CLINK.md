# 06 - CLI Linking (Clink)

## Overview

The CLI Linking system (clink) allows Claude Code to spawn and communicate with other AI CLIs like Gemini CLI, Codex CLI, etc. This enables multi-model collaboration through subprocess management.

## Architecture

```
┌─────────────────┐
│  Claude Code    │
│  (MCP Client)   │
└────────┬────────┘
         │ MCP Protocol
         ▼
┌─────────────────┐
│   PAL Server    │
│  (clink tool)   │
└────────┬────────┘
         │ Subprocess
         ├──────────────┬──────────────┐
         ▼              ▼              ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│ Gemini CLI  │ │ Claude CLI  │ │ Codex CLI   │
│  (gemini)   │ │  (claude)   │ │  (codex)    │
└─────────────┘ └─────────────┘ └─────────────┘
```

## Agent Interface (internal/clink/agent.go)

```go
package clink

import (
    "context"
    "time"
)

// Agent is the interface for CLI agents
type Agent interface {
    // Name returns the agent name
    Name() string

    // Run executes the agent with the given request
    Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error)

    // IsAvailable checks if the CLI is installed and accessible
    IsAvailable() bool
}

// AgentRequest contains the input for an agent
type AgentRequest struct {
    Role         string            // Role preset (default, planner, codereviewer)
    Prompt       string            // The prompt to send
    SystemPrompt string            // System prompt for the role
    Files        []string          // File paths to include
    Images       []string          // Image paths to include
    WorkDir      string            // Working directory
    Timeout      time.Duration     // Execution timeout
    Env          map[string]string // Environment variables
}

// AgentOutput contains the result from an agent
type AgentOutput struct {
    Content      string        // The agent's response
    ExitCode     int           // Process exit code
    Duration     time.Duration // Execution time
    TokensUsed   int           // Estimated tokens (if available)
    ErrorMessage string        // Error message if failed
}

// AgentConfig defines a CLI agent configuration
type AgentConfig struct {
    Name           string            `json:"name"`
    Command        string            `json:"command"`
    AdditionalArgs []string          `json:"additional_args"`
    Roles          map[string]Role   `json:"roles"`
    Env            map[string]string `json:"env,omitempty"`
    Timeout        string            `json:"timeout,omitempty"` // e.g., "5m"
}

// Role defines a role preset for an agent
type Role struct {
    PromptPath   string   `json:"prompt_path"`
    SystemPrompt string   `json:"system_prompt,omitempty"`
    Args         []string `json:"args,omitempty"`
}
```

## Agent Registry (internal/clink/registry.go)

```go
package clink

import (
    "fmt"
    "sync"

    "github.com/yourorg/pal-mcp/internal/config"
)

// Registry manages CLI agents
type Registry struct {
    agents map[string]Agent
    mu     sync.RWMutex
}

// NewRegistry creates a new agent registry
func NewRegistry(cfg *config.Config) (*Registry, error) {
    r := &Registry{
        agents: make(map[string]Agent),
    }

    // Register configured CLI clients
    for name, clientCfg := range cfg.CLIClients {
        agent, err := r.createAgent(name, clientCfg, cfg)
        if err != nil {
            // Log warning but continue
            continue
        }

        if agent.IsAvailable() {
            r.agents[name] = agent
        }
    }

    return r, nil
}

func (r *Registry) createAgent(name string, clientCfg config.CLIClientConfig, cfg *config.Config) (Agent, error) {
    switch name {
    case "gemini":
        return NewGeminiAgent(clientCfg, cfg), nil
    case "claude":
        return NewClaudeAgent(clientCfg, cfg), nil
    case "codex":
        return NewCodexAgent(clientCfg, cfg), nil
    default:
        return NewGenericAgent(clientCfg, cfg), nil
    }
}

// Get returns an agent by name
func (r *Registry) Get(name string) (Agent, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    agent, ok := r.agents[name]
    return agent, ok
}

// List returns all available agents
func (r *Registry) List() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()

    names := make([]string, 0, len(r.agents))
    for name := range r.agents {
        names = append(names, name)
    }
    return names
}
```

## Base Agent (internal/clink/base.go)

```go
package clink

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/utils"
)

// BaseAgent provides common functionality for CLI agents
type BaseAgent struct {
    name    string
    command string
    args    []string
    roles   map[string]Role
    env     map[string]string
    timeout time.Duration
    cfg     *config.Config
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(clientCfg config.CLIClientConfig, cfg *config.Config) *BaseAgent {
    timeout := 5 * time.Minute
    if clientCfg.Timeout != "" {
        if d, err := time.ParseDuration(clientCfg.Timeout); err == nil {
            timeout = d
        }
    }

    roles := make(map[string]Role)
    for name, roleCfg := range clientCfg.Roles {
        roles[name] = Role{
            PromptPath:   roleCfg.PromptPath,
            SystemPrompt: loadPromptFile(roleCfg.PromptPath),
        }
    }

    return &BaseAgent{
        name:    clientCfg.Name,
        command: clientCfg.Command,
        args:    clientCfg.AdditionalArgs,
        roles:   roles,
        timeout: timeout,
        cfg:     cfg,
    }
}

func (a *BaseAgent) Name() string {
    return a.name
}

// IsAvailable checks if the CLI executable exists
func (a *BaseAgent) IsAvailable() bool {
    _, err := exec.LookPath(a.command)
    return err == nil
}

// Run executes the CLI agent
func (a *BaseAgent) Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error) {
    start := time.Now()

    // Build the full prompt
    fullPrompt := a.buildPrompt(req)

    // Set timeout
    timeout := a.timeout
    if req.Timeout > 0 {
        timeout = req.Timeout
    }
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Build command
    args := append([]string{}, a.args...)
    if roleArgs := a.getRoleArgs(req.Role); len(roleArgs) > 0 {
        args = append(args, roleArgs...)
    }

    cmd := exec.CommandContext(ctx, a.command, args...)

    // Set working directory
    if req.WorkDir != "" {
        cmd.Dir = req.WorkDir
    }

    // Set environment
    cmd.Env = os.Environ()
    for k, v := range a.env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }
    for k, v := range req.Env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }

    // Set up stdin/stdout/stderr
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, fmt.Errorf("creating stdin pipe: %w", err)
    }

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    slog.Info("starting CLI agent",
        "name", a.name,
        "command", a.command,
        "args", args,
        "workdir", req.WorkDir,
    )

    // Start the process
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("starting process: %w", err)
    }

    // Write prompt to stdin
    go func() {
        defer stdin.Close()
        io.WriteString(stdin, fullPrompt)
    }()

    // Wait for completion
    err = cmd.Wait()
    duration := time.Since(start)

    output := &AgentOutput{
        Content:  stdout.String(),
        ExitCode: cmd.ProcessState.ExitCode(),
        Duration: duration,
    }

    if err != nil {
        output.ErrorMessage = fmt.Sprintf("process error: %v\nstderr: %s", err, stderr.String())
        slog.Warn("CLI agent error",
            "name", a.name,
            "error", err,
            "stderr", stderr.String(),
            "duration", duration,
        )
    } else {
        slog.Info("CLI agent completed",
            "name", a.name,
            "duration", duration,
            "output_length", len(output.Content),
        )
    }

    return output, nil
}

// buildPrompt constructs the full prompt with files and context
func (a *BaseAgent) buildPrompt(req *AgentRequest) string {
    var sb strings.Builder

    // Add system prompt from role
    if role, ok := a.roles[req.Role]; ok && role.SystemPrompt != "" {
        sb.WriteString(role.SystemPrompt)
        sb.WriteString("\n\n")
    } else if req.SystemPrompt != "" {
        sb.WriteString(req.SystemPrompt)
        sb.WriteString("\n\n")
    }

    // Add file contents
    if len(req.Files) > 0 {
        sb.WriteString("## Files\n\n")
        for _, path := range req.Files {
            content, err := os.ReadFile(path)
            if err != nil {
                continue
            }
            sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", path, string(content)))
        }
    }

    // Add the main prompt
    sb.WriteString("## Request\n\n")
    sb.WriteString(req.Prompt)

    return sb.String()
}

func (a *BaseAgent) getRoleArgs(role string) []string {
    if r, ok := a.roles[role]; ok {
        return r.Args
    }
    return nil
}

func loadPromptFile(path string) string {
    if path == "" {
        return ""
    }

    // Try relative to prompts directory
    fullPath := filepath.Join("prompts", path)
    content, err := os.ReadFile(fullPath)
    if err != nil {
        // Try absolute path
        content, err = os.ReadFile(path)
        if err != nil {
            return ""
        }
    }
    return string(content)
}
```

## Gemini Agent (internal/clink/gemini.go)

```go
package clink

import (
    "context"
    "encoding/json"
    "strings"

    "github.com/yourorg/pal-mcp/internal/config"
)

// GeminiAgent is a CLI agent for Gemini CLI
type GeminiAgent struct {
    *BaseAgent
    parser *GeminiParser
}

// NewGeminiAgent creates a new Gemini CLI agent
func NewGeminiAgent(clientCfg config.CLIClientConfig, cfg *config.Config) *GeminiAgent {
    return &GeminiAgent{
        BaseAgent: NewBaseAgent(clientCfg, cfg),
        parser:    &GeminiParser{},
    }
}

// Run executes the Gemini CLI and parses the output
func (a *GeminiAgent) Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error) {
    output, err := a.BaseAgent.Run(ctx, req)
    if err != nil {
        return output, err
    }

    // Parse Gemini-specific output format
    parsed := a.parser.Parse(output.Content)
    output.Content = parsed

    return output, nil
}

// GeminiParser parses Gemini CLI output
type GeminiParser struct{}

// Parse extracts content from Gemini CLI output
func (p *GeminiParser) Parse(raw string) string {
    // Gemini CLI may output JSON or plain text
    // Try to parse as JSON first
    raw = strings.TrimSpace(raw)

    if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
        // Try JSON parsing
        var result struct {
            Response string `json:"response"`
            Content  string `json:"content"`
            Text     string `json:"text"`
        }

        if err := json.Unmarshal([]byte(raw), &result); err == nil {
            if result.Response != "" {
                return result.Response
            }
            if result.Content != "" {
                return result.Content
            }
            if result.Text != "" {
                return result.Text
            }
        }

        // Try as array
        var responses []struct {
            Content string `json:"content"`
        }
        if err := json.Unmarshal([]byte(raw), &responses); err == nil && len(responses) > 0 {
            var parts []string
            for _, r := range responses {
                if r.Content != "" {
                    parts = append(parts, r.Content)
                }
            }
            if len(parts) > 0 {
                return strings.Join(parts, "\n")
            }
        }
    }

    // Return raw output if no special parsing needed
    return raw
}
```

## Claude Agent (internal/clink/claude.go)

```go
package clink

import (
    "context"
    "strings"

    "github.com/yourorg/pal-mcp/internal/config"
)

// ClaudeAgent is a CLI agent for Claude Code
type ClaudeAgent struct {
    *BaseAgent
    parser *ClaudeParser
}

// NewClaudeAgent creates a new Claude CLI agent
func NewClaudeAgent(clientCfg config.CLIClientConfig, cfg *config.Config) *ClaudeAgent {
    return &ClaudeAgent{
        BaseAgent: NewBaseAgent(clientCfg, cfg),
        parser:    &ClaudeParser{},
    }
}

// Run executes Claude CLI and parses the output
func (a *ClaudeAgent) Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error) {
    output, err := a.BaseAgent.Run(ctx, req)
    if err != nil {
        return output, err
    }

    // Parse Claude-specific output
    parsed := a.parser.Parse(output.Content)
    output.Content = parsed

    return output, nil
}

// ClaudeParser parses Claude CLI output
type ClaudeParser struct{}

// Parse extracts content from Claude CLI output
func (p *ClaudeParser) Parse(raw string) string {
    // Claude Code outputs markdown-formatted responses
    // Strip any ANSI codes and clean up
    raw = stripANSI(raw)
    return strings.TrimSpace(raw)
}

func stripANSI(s string) string {
    // Simple ANSI escape code removal
    result := strings.Builder{}
    inEscape := false

    for _, r := range s {
        if r == '\x1b' {
            inEscape = true
            continue
        }
        if inEscape {
            if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
                inEscape = false
            }
            continue
        }
        result.WriteRune(r)
    }

    return result.String()
}
```

## Generic Agent (internal/clink/generic.go)

```go
package clink

import (
    "context"

    "github.com/yourorg/pal-mcp/internal/config"
)

// GenericAgent is a generic CLI agent for custom CLIs
type GenericAgent struct {
    *BaseAgent
}

// NewGenericAgent creates a new generic CLI agent
func NewGenericAgent(clientCfg config.CLIClientConfig, cfg *config.Config) *GenericAgent {
    return &GenericAgent{
        BaseAgent: NewBaseAgent(clientCfg, cfg),
    }
}

// Run executes the CLI (uses base implementation)
func (a *GenericAgent) Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error) {
    return a.BaseAgent.Run(ctx, req)
}
```

## Clink Tool (internal/tools/simple/clink.go)

```go
package simple

import (
    "context"
    "fmt"
    "strings"

    "github.com/yourorg/pal-mcp/internal/clink"
    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/memory"
    "github.com/yourorg/pal-mcp/internal/tools"
)

// ClinkTool bridges to external AI CLIs
type ClinkTool struct {
    name        string
    description string
    cfg         *config.Config
    memory      *memory.ConversationMemory
    registry    *clink.Registry
    schema      *tools.SchemaBuilder
}

// NewClinkTool creates a new clink tool
func NewClinkTool(cfg *config.Config, mem *memory.ConversationMemory) *ClinkTool {
    registry, _ := clink.NewRegistry(cfg)

    tool := &ClinkTool{
        name: "clink",
        description: "Link to an external AI CLI (Gemini CLI, Claude CLI, Codex CLI) " +
            "to reuse their capabilities inside existing workflows.",
        cfg:      cfg,
        memory:   mem,
        registry: registry,
        schema:   tools.NewSchemaBuilder(),
    }

    // Define schema
    availableCLIs := registry.List()
    tool.schema.
        AddString("prompt", "User request to forward to the CLI", true).
        AddStringEnum("cli_name", "CLI client name", availableCLIs, true).
        AddStringEnum("role", "Role preset for the CLI", []string{"default", "planner", "codereviewer"}, false).
        AddStringArray("absolute_file_paths", "File paths to share with the CLI", false).
        AddStringArray("images", "Image paths for visual context", false).
        AddString("continuation_id", "Thread ID for conversation continuation", false)

    return tool
}

func (t *ClinkTool) Name() string        { return t.name }
func (t *ClinkTool) Description() string { return t.description }
func (t *ClinkTool) Schema() map[string]any { return t.schema.Build() }

func (t *ClinkTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
    parser := tools.NewArgumentParser(args)

    prompt, err := parser.GetStringRequired("prompt")
    if err != nil {
        return nil, err
    }

    cliName, err := parser.GetStringRequired("cli_name")
    if err != nil {
        return nil, err
    }

    role := parser.GetString("role")
    if role == "" {
        role = "default"
    }

    files := parser.GetStringArray("absolute_file_paths")
    images := parser.GetStringArray("images")
    continuationID := parser.GetString("continuation_id")

    // Get the agent
    agent, ok := t.registry.Get(cliName)
    if !ok {
        return nil, fmt.Errorf("CLI agent not found: %s (available: %s)",
            cliName, strings.Join(t.registry.List(), ", "))
    }

    // Build request
    req := &clink.AgentRequest{
        Role:   role,
        Prompt: prompt,
        Files:  files,
        Images: images,
    }

    // Add conversation context if continuing
    if continuationID != "" {
        thread := t.memory.GetThread(continuationID)
        if thread != nil {
            req.Prompt = t.buildContextualPrompt(thread, prompt)
        }
    }

    // Execute
    output, err := agent.Run(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("CLI execution failed: %w", err)
    }

    if output.ExitCode != 0 {
        return tools.NewToolError(fmt.Sprintf(
            "CLI exited with code %d: %s", output.ExitCode, output.ErrorMessage)), nil
    }

    // Build response
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("## Response from %s (%s role)\n\n", cliName, role))
    sb.WriteString(output.Content)
    sb.WriteString(fmt.Sprintf("\n\n---\n*Execution time: %s*", output.Duration))

    if continuationID != "" {
        sb.WriteString(fmt.Sprintf("\ncontinuation_id: %s", continuationID))
    }

    return tools.NewToolResult(sb.String()), nil
}

func (t *ClinkTool) buildContextualPrompt(thread *memory.ThreadContext, currentPrompt string) string {
    var sb strings.Builder

    sb.WriteString("## Previous Context\n\n")

    // Include recent turns (last 5)
    turns := thread.Turns
    if len(turns) > 5 {
        turns = turns[len(turns)-5:]
    }

    for _, turn := range turns {
        sb.WriteString(fmt.Sprintf("**%s**: %s\n\n", turn.Role, truncate(turn.Content, 500)))
    }

    sb.WriteString("## Current Request\n\n")
    sb.WriteString(currentPrompt)

    return sb.String()
}

func truncate(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}
```

## CLI Client Configuration (configs/cli_clients/gemini.json)

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

## Role Prompts

### prompts/clink/default.txt

```text
You are an AI assistant helping with software development tasks.
Provide clear, actionable responses.
When discussing code, be specific about file locations and line numbers.
```

### prompts/clink/planner.txt

```text
You are a software architect and planning specialist.
Your role is to:
1. Analyze requirements thoroughly
2. Break down complex tasks into clear steps
3. Identify potential risks and dependencies
4. Suggest implementation approaches

Provide structured, step-by-step plans.
Do not include time estimates - focus on what needs to be done.
```

### prompts/clink/codereviewer.txt

```text
You are an expert code reviewer.
Your role is to:
1. Analyze code for bugs, security issues, and performance problems
2. Check adherence to best practices
3. Suggest improvements
4. Identify potential edge cases

Be thorough but constructive. Reference specific line numbers.
Categorize issues by severity (critical, high, medium, low).
```

## Usage Example

From Claude Code:
```
Use the clink tool to ask Gemini CLI to review this function
```

PAL Server executes:
```bash
gemini --yolo << EOF
[System prompt from codereviewer role]

## Files

### /path/to/function.go
```go
func doSomething() {
    // code here
}
```

## Request

Review this function for bugs and improvements.
EOF
```

## Testing

```go
// internal/clink/agent_test.go
package clink

import (
    "context"
    "testing"
    "time"

    "github.com/yourorg/pal-mcp/internal/config"
)

func TestGeminiParser(t *testing.T) {
    parser := &GeminiParser{}

    tests := []struct {
        input    string
        expected string
    }{
        {
            input:    `{"response": "Hello, world!"}`,
            expected: "Hello, world!",
        },
        {
            input:    "Plain text response",
            expected: "Plain text response",
        },
        {
            input:    `[{"content": "Part 1"}, {"content": "Part 2"}]`,
            expected: "Part 1\nPart 2",
        },
    }

    for _, tt := range tests {
        result := parser.Parse(tt.input)
        if result != tt.expected {
            t.Errorf("Parse(%q) = %q, want %q", tt.input, result, tt.expected)
        }
    }
}

func TestBaseAgent_IsAvailable(t *testing.T) {
    cfg := &config.Config{}
    clientCfg := config.CLIClientConfig{
        Name:    "test",
        Command: "echo", // echo is available on all systems
    }

    agent := NewBaseAgent(clientCfg, cfg)

    if !agent.IsAvailable() {
        t.Error("expected echo to be available")
    }

    // Test unavailable command
    clientCfg.Command = "nonexistent_command_12345"
    agent = NewBaseAgent(clientCfg, cfg)

    if agent.IsAvailable() {
        t.Error("expected nonexistent command to be unavailable")
    }
}

func TestBaseAgent_Run(t *testing.T) {
    cfg := &config.Config{}
    clientCfg := config.CLIClientConfig{
        Name:    "echo",
        Command: "cat", // cat echoes stdin
    }

    agent := NewBaseAgent(clientCfg, cfg)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    output, err := agent.Run(ctx, &AgentRequest{
        Prompt: "Hello from test",
    })

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if output.ExitCode != 0 {
        t.Errorf("expected exit code 0, got %d", output.ExitCode)
    }

    if !strings.Contains(output.Content, "Hello from test") {
        t.Errorf("expected output to contain prompt, got: %s", output.Content)
    }
}
```

## Next Steps

Continue to [07-CONSENSUS-WORKFLOWS.md](./07-CONSENSUS-WORKFLOWS.md) for multi-model orchestration.
