# 04 - Tool System

## Overview

Tools are the core functionality exposed by the MCP server. They come in two types:
- **Simple Tools**: Single request/response (chat, apilookup, version)
- **Workflow Tools**: Multi-step investigations (debug, codereview, thinkdeep)

## Tool Interface (internal/tools/tool.go)

```go
package tools

import (
    "context"

    "github.com/yourorg/pal-mcp/internal/types"
)

// Tool is the interface all tools must implement
type Tool interface {
    // Name returns the tool name
    Name() string

    // Description returns the tool description
    Description() string

    // Schema returns the JSON schema for tool parameters
    Schema() map[string]any

    // Execute runs the tool with the given arguments
    Execute(ctx context.Context, args map[string]any) (*ToolResult, error)
}

// ToolResult is the result of tool execution
type ToolResult struct {
    Content  string            // Text content to return
    Metadata map[string]any    // Optional metadata
    IsError  bool              // Whether this is an error result
}

// NewToolResult creates a successful result
func NewToolResult(content string) *ToolResult {
    return &ToolResult{Content: content}
}

// NewToolError creates an error result
func NewToolError(message string) *ToolResult {
    return &ToolResult{
        Content: message,
        IsError: true,
    }
}

// ArgumentParser helps parse tool arguments
type ArgumentParser struct {
    args map[string]any
}

// NewArgumentParser creates a new parser
func NewArgumentParser(args map[string]any) *ArgumentParser {
    return &ArgumentParser{args: args}
}

// GetString returns a string argument
func (p *ArgumentParser) GetString(key string) string {
    if v, ok := p.args[key].(string); ok {
        return v
    }
    return ""
}

// GetStringRequired returns a required string argument
func (p *ArgumentParser) GetStringRequired(key string) (string, error) {
    v := p.GetString(key)
    if v == "" {
        return "", ErrMissingRequired{Field: key}
    }
    return v, nil
}

// GetInt returns an int argument
func (p *ArgumentParser) GetInt(key string, defaultVal int) int {
    switch v := p.args[key].(type) {
    case int:
        return v
    case float64:
        return int(v)
    case int64:
        return int(v)
    default:
        return defaultVal
    }
}

// GetFloat returns a float argument
func (p *ArgumentParser) GetFloat(key string, defaultVal float64) float64 {
    switch v := p.args[key].(type) {
    case float64:
        return v
    case int:
        return float64(v)
    default:
        return defaultVal
    }
}

// GetBool returns a bool argument
func (p *ArgumentParser) GetBool(key string, defaultVal bool) bool {
    if v, ok := p.args[key].(bool); ok {
        return v
    }
    return defaultVal
}

// GetStringArray returns a string array argument
func (p *ArgumentParser) GetStringArray(key string) []string {
    v, ok := p.args[key].([]any)
    if !ok {
        return nil
    }
    result := make([]string, 0, len(v))
    for _, item := range v {
        if s, ok := item.(string); ok {
            result = append(result, s)
        }
    }
    return result
}

// GetObjectArray returns an array of objects
func (p *ArgumentParser) GetObjectArray(key string) []map[string]any {
    v, ok := p.args[key].([]any)
    if !ok {
        return nil
    }
    result := make([]map[string]any, 0, len(v))
    for _, item := range v {
        if m, ok := item.(map[string]any); ok {
            result = append(result, m)
        }
    }
    return result
}

// Error types
type ErrMissingRequired struct {
    Field string
}

func (e ErrMissingRequired) Error() string {
    return "missing required field: " + e.Field
}

type ErrInvalidValue struct {
    Field   string
    Message string
}

func (e ErrInvalidValue) Error() string {
    return "invalid value for " + e.Field + ": " + e.Message
}
```

## Simple Tool Base (internal/tools/simple/base.go)

```go
package simple

import (
    "context"
    "fmt"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/memory"
    "github.com/yourorg/pal-mcp/internal/providers"
    "github.com/yourorg/pal-mcp/internal/tools"
    "github.com/yourorg/pal-mcp/internal/types"
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

func (t *BaseTool) Name() string        { return t.name }
func (t *BaseTool) Description() string { return t.description }
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
    t.memory.AddTurn(threadID, types.ConversationTurn{
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
```

## Version Tool (internal/tools/simple/version.go)

```go
package simple

import (
    "context"
    "fmt"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/tools"
)

// VersionTool returns server version information
type VersionTool struct {
    cfg *config.Config
}

// NewVersionTool creates a new version tool
func NewVersionTool(cfg *config.Config) *VersionTool {
    return &VersionTool{cfg: cfg}
}

func (t *VersionTool) Name() string {
    return "version"
}

func (t *VersionTool) Description() string {
    return "Get server version, configuration details, and list of available tools."
}

func (t *VersionTool) Schema() map[string]any {
    return tools.NewSchemaBuilder().Build()
}

func (t *VersionTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
    content := fmt.Sprintf(`PAL MCP Server
Version: %s
Commit: %s
Build Time: %s

This server provides AI-powered development tools including:
- chat: Multi-turn conversations with AI models
- thinkdeep: Extended reasoning and analysis
- debug: Root cause analysis for bugs
- codereview: Systematic code review
- consensus: Multi-model debate and synthesis
- clink: Bridge to external AI CLIs
- And more...

Use 'listmodels' to see available AI models.`,
        t.cfg.Version,
        t.cfg.Commit,
        t.cfg.BuildTime,
    )

    return tools.NewToolResult(content), nil
}
```

## ListModels Tool (internal/tools/simple/listmodels.go)

```go
package simple

import (
    "context"
    "fmt"
    "strings"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/providers"
    "github.com/yourorg/pal-mcp/internal/tools"
)

// ListModelsTool lists available models
type ListModelsTool struct {
    cfg      *config.Config
    registry *providers.Registry
}

// NewListModelsTool creates a new listmodels tool
func NewListModelsTool(cfg *config.Config, registry *providers.Registry) *ListModelsTool {
    return &ListModelsTool{
        cfg:      cfg,
        registry: registry,
    }
}

func (t *ListModelsTool) Name() string {
    return "listmodels"
}

func (t *ListModelsTool) Description() string {
    return "Shows which AI model providers are configured, available model names, their aliases and capabilities."
}

func (t *ListModelsTool) Schema() map[string]any {
    return tools.NewSchemaBuilder().Build()
}

func (t *ListModelsTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
    models := t.registry.GetAllModels()

    var sb strings.Builder
    sb.WriteString("# Available Models\n\n")

    currentProvider := ""
    for _, m := range models {
        if string(m.Provider) != currentProvider {
            currentProvider = string(m.Provider)
            sb.WriteString(fmt.Sprintf("\n## %s\n\n", strings.ToUpper(currentProvider)))
        }

        aliases := ""
        if len(m.Aliases) > 0 {
            aliases = fmt.Sprintf(" (aliases: %s)", strings.Join(m.Aliases, ", "))
        }

        features := []string{}
        if m.SupportsExtendedThinking {
            features = append(features, "thinking")
        }
        if m.SupportsVision {
            features = append(features, "vision")
        }
        if m.AllowCodeGeneration {
            features = append(features, "code-gen")
        }

        sb.WriteString(fmt.Sprintf("- **%s**%s\n", m.ModelName, aliases))
        sb.WriteString(fmt.Sprintf("  - Score: %d | Context: %dk | Features: %s\n",
            m.IntelligenceScore,
            m.ContextWindow/1000,
            strings.Join(features, ", "),
        ))
    }

    return tools.NewToolResult(sb.String()), nil
}
```

## Chat Tool (internal/tools/simple/chat.go)

```go
package simple

import (
    "context"
    "fmt"
    "log/slog"
    "strings"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/memory"
    "github.com/yourorg/pal-mcp/internal/providers"
    "github.com/yourorg/pal-mcp/internal/tools"
    "github.com/yourorg/pal-mcp/internal/types"
    "github.com/yourorg/pal-mcp/internal/utils"
)

// ChatTool handles multi-turn conversations
type ChatTool struct {
    *BaseTool
}

// NewChatTool creates a new chat tool
func NewChatTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *ChatTool {
    tool := &ChatTool{
        BaseTool: NewBaseTool("chat", "General chat and collaborative thinking partner", cfg, registry, mem),
    }

    // Define schema
    tool.schema.
        AddString("prompt", "Your question or idea for collaborative thinking", true).
        AddString("working_directory_absolute_path", "Absolute path to working directory", true).
        AddString("model", "Model to use (or 'auto' for automatic selection)", false).
        AddStringArray("absolute_file_paths", "Full paths to relevant code files", false).
        AddStringArray("images", "Image paths or base64 strings", false).
        AddString("continuation_id", "Thread ID for multi-turn conversations", false).
        AddNumber("temperature", "0 = deterministic, 1 = creative", false, ptr(0.0), ptr(1.0)).
        AddStringEnum("thinking_mode", "Reasoning depth", []string{"minimal", "low", "medium", "high", "max"}, false)

    return tool
}

func (t *ChatTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
    parser := tools.NewArgumentParser(args)

    prompt, err := parser.GetStringRequired("prompt")
    if err != nil {
        return nil, err
    }

    workDir, err := parser.GetStringRequired("working_directory_absolute_path")
    if err != nil {
        return nil, err
    }

    modelName := parser.GetString("model")
    filePaths := parser.GetStringArray("absolute_file_paths")
    images := parser.GetStringArray("images")
    continuationID := parser.GetString("continuation_id")
    temperature := parser.GetFloat("temperature", 0.7)
    thinkingMode := types.ThinkingMode(parser.GetString("thinking_mode"))

    // Get or create conversation thread
    thread, isExisting := t.GetOrCreateThread(continuationID)
    slog.Debug("chat thread", "id", thread.ThreadID, "existing", isExisting)

    // Resolve model
    resolvedModel, provider, err := t.ResolveModel(modelName)
    if err != nil {
        return nil, fmt.Errorf("resolving model: %w", err)
    }

    // Read files
    fileContents, err := utils.ReadFiles(filePaths, workDir)
    if err != nil {
        slog.Warn("error reading files", "error", err)
    }

    // Build prompt with file contents
    fullPrompt := t.buildPrompt(prompt, fileContents)

    // Get conversation history
    history := t.memory.GetHistory(thread.ThreadID)

    // Generate response
    resp, err := t.GenerateContent(ctx, provider, &providers.GenerateRequest{
        Prompt:              fullPrompt,
        SystemPrompt:        t.getSystemPrompt(),
        Model:               resolvedModel,
        Temperature:         temperature,
        ThinkingMode:        thinkingMode,
        ConversationHistory: history,
        Images:              images,
    })
    if err != nil {
        return nil, fmt.Errorf("generating content: %w", err)
    }

    // Save turns
    t.AddTurn(thread.ThreadID, "user", prompt, filePaths, images)
    t.AddTurn(thread.ThreadID, "assistant", resp.Content, nil, nil)

    // Build response with continuation ID
    result := fmt.Sprintf("%s\n\n---\ncontinuation_id: %s", resp.Content, thread.ThreadID)

    return tools.NewToolResult(result), nil
}

func (t *ChatTool) buildPrompt(prompt string, files []utils.FileContent) string {
    if len(files) == 0 {
        return prompt
    }

    var sb strings.Builder
    sb.WriteString(prompt)
    sb.WriteString("\n\n## Referenced Files\n\n")

    for _, f := range files {
        sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", f.Path, f.Content))
    }

    return sb.String()
}

func (t *ChatTool) getSystemPrompt() string {
    // Load from prompts/chat.txt or use default
    return `You are a helpful AI assistant for software development.
You help with coding questions, debugging, architecture decisions, and general development tasks.
Be concise but thorough. When discussing code, reference specific files and line numbers.
If you're unsure about something, say so rather than making assumptions.`
}

func ptr[T any](v T) *T {
    return &v
}
```

## Workflow Tool Base (internal/tools/workflow/base.go)

```go
package workflow

import (
    "context"
    "fmt"
    "log/slog"
    "strings"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/memory"
    "github.com/yourorg/pal-mcp/internal/providers"
    "github.com/yourorg/pal-mcp/internal/tools"
    "github.com/yourorg/pal-mcp/internal/types"
)

// WorkflowTool provides common functionality for multi-step workflow tools
type WorkflowTool struct {
    name        string
    description string
    schema      *tools.SchemaBuilder
    cfg         *config.Config
    registry    *providers.Registry
    memory      *memory.ConversationMemory
}

// NewWorkflowTool creates a new workflow tool
func NewWorkflowTool(
    name, description string,
    cfg *config.Config,
    registry *providers.Registry,
    mem *memory.ConversationMemory,
) *WorkflowTool {
    wt := &WorkflowTool{
        name:        name,
        description: description,
        schema:      tools.NewSchemaBuilder(),
        cfg:         cfg,
        registry:    registry,
        memory:      mem,
    }

    // Add common workflow schema fields
    wt.addCommonSchema()

    return wt
}

func (t *WorkflowTool) addCommonSchema() {
    t.schema.
        AddString("step", "Current work step content and findings", true).
        AddInteger("step_number", "Current step number (starts at 1)", true, intPtr(1), nil).
        AddInteger("total_steps", "Estimated total steps needed", true, intPtr(1), nil).
        AddBoolean("next_step_required", "Whether another step is needed", true).
        AddString("findings", "Important discoveries and evidence", true).
        AddString("model", "Model to use", false).
        AddString("hypothesis", "Current theory based on evidence", false).
        AddStringEnum("confidence", "Confidence level", []string{
            "exploring", "low", "medium", "high", "very_high", "almost_certain", "certain",
        }, false).
        AddStringArray("relevant_files", "Files relevant to the investigation", false).
        AddStringArray("files_checked", "All files examined", false).
        AddString("continuation_id", "Thread ID for multi-turn conversations", false).
        AddBoolean("use_assistant_model", "Use expert model for analysis", false).
        AddStringEnum("thinking_mode", "Reasoning depth", []string{
            "minimal", "low", "medium", "high", "max",
        }, false).
        AddNumber("temperature", "0 = deterministic, 1 = creative", false, floatPtr(0.0), floatPtr(1.0))
}

func (t *WorkflowTool) Name() string        { return t.name }
func (t *WorkflowTool) Description() string { return t.description }
func (t *WorkflowTool) Schema() map[string]any { return t.schema.Build() }

// WorkflowState holds the current state of a workflow
type WorkflowState struct {
    Step             string
    StepNumber       int
    TotalSteps       int
    NextStepRequired bool
    Findings         string
    Hypothesis       string
    Confidence       types.ConfidenceLevel
    RelevantFiles    []string
    FilesChecked     []string
    ContinuationID   string
    UseAssistant     bool
    ThinkingMode     types.ThinkingMode
    Temperature      float64
    Model            string
}

// ParseWorkflowState extracts workflow state from arguments
func (t *WorkflowTool) ParseWorkflowState(args map[string]any) (*WorkflowState, error) {
    parser := tools.NewArgumentParser(args)

    step, err := parser.GetStringRequired("step")
    if err != nil {
        return nil, err
    }

    findings, err := parser.GetStringRequired("findings")
    if err != nil {
        return nil, err
    }

    return &WorkflowState{
        Step:             step,
        StepNumber:       parser.GetInt("step_number", 1),
        TotalSteps:       parser.GetInt("total_steps", 1),
        NextStepRequired: parser.GetBool("next_step_required", false),
        Findings:         findings,
        Hypothesis:       parser.GetString("hypothesis"),
        Confidence:       types.ConfidenceLevel(parser.GetString("confidence")),
        RelevantFiles:    parser.GetStringArray("relevant_files"),
        FilesChecked:     parser.GetStringArray("files_checked"),
        ContinuationID:   parser.GetString("continuation_id"),
        UseAssistant:     parser.GetBool("use_assistant_model", true),
        ThinkingMode:     types.ThinkingMode(parser.GetString("thinking_mode")),
        Temperature:      parser.GetFloat("temperature", 0.3),
        Model:            parser.GetString("model"),
    }, nil
}

// GetOrCreateThread manages conversation threading
func (t *WorkflowTool) GetOrCreateThread(continuationID string) (*types.ThreadContext, bool) {
    if continuationID == "" {
        return t.memory.CreateThread(t.name), false
    }

    thread := t.memory.GetThread(continuationID)
    if thread == nil {
        return t.memory.CreateThread(t.name), false
    }

    return thread, true
}

// ConsolidateFindings merges findings from multiple steps
func (t *WorkflowTool) ConsolidateFindings(thread *types.ThreadContext, currentFindings string) string {
    var allFindings []string

    for _, turn := range thread.Turns {
        if turn.Role == "user" && strings.Contains(turn.Content, "findings:") {
            // Extract findings from previous turns
            allFindings = append(allFindings, turn.Content)
        }
    }

    if currentFindings != "" {
        allFindings = append(allFindings, currentFindings)
    }

    return strings.Join(allFindings, "\n\n---\n\n")
}

// CallExpertModel calls a high-intelligence model for final analysis
func (t *WorkflowTool) CallExpertModel(
    ctx context.Context,
    prompt string,
    systemPrompt string,
) (*types.ModelResponse, error) {
    // Select best available model
    caps, provider, err := t.registry.SelectBestModel(providers.ModelRequirements{
        MinIntelligence:     80,
        NeedsThinking:       true,
        NeedsCodeGeneration: true,
    })
    if err != nil {
        return nil, fmt.Errorf("selecting expert model: %w", err)
    }

    slog.Info("calling expert model", "model", caps.ModelName, "provider", caps.Provider)

    return provider.GenerateContent(ctx, &providers.GenerateRequest{
        Prompt:       prompt,
        SystemPrompt: systemPrompt,
        Model:        caps.ModelName,
        Temperature:  0.3,
        ThinkingMode: types.ThinkingHigh,
    })
}

// BuildGuidanceResponse creates the response for intermediate steps
func (t *WorkflowTool) BuildGuidanceResponse(state *WorkflowState, guidance string) string {
    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("## Step %d of %d\n\n", state.StepNumber, state.TotalSteps))
    sb.WriteString(fmt.Sprintf("**Confidence:** %s\n", state.Confidence))

    if state.Hypothesis != "" {
        sb.WriteString(fmt.Sprintf("**Current Hypothesis:** %s\n", state.Hypothesis))
    }

    sb.WriteString("\n### Guidance\n\n")
    sb.WriteString(guidance)

    sb.WriteString("\n\n---\n")
    sb.WriteString(fmt.Sprintf("continuation_id: %s\n", state.ContinuationID))

    return sb.String()
}

func intPtr(v int) *int       { return &v }
func floatPtr(v float64) *float64 { return &v }
```

## Debug Tool (internal/tools/workflow/debug.go)

```go
package workflow

import (
    "context"
    "fmt"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/memory"
    "github.com/yourorg/pal-mcp/internal/providers"
    "github.com/yourorg/pal-mcp/internal/tools"
    "github.com/yourorg/pal-mcp/internal/types"
)

// DebugTool performs systematic debugging and root cause analysis
type DebugTool struct {
    *WorkflowTool
}

// NewDebugTool creates a new debug tool
func NewDebugTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *DebugTool {
    return &DebugTool{
        WorkflowTool: NewWorkflowTool(
            "debug",
            "Performs systematic debugging and root cause analysis for any type of issue.",
            cfg, registry, mem,
        ),
    }
}

func (t *DebugTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
    state, err := t.ParseWorkflowState(args)
    if err != nil {
        return nil, err
    }

    // Get or create thread
    thread, isExisting := t.GetOrCreateThread(state.ContinuationID)
    state.ContinuationID = thread.ThreadID

    // Save this step to memory
    t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
        Role:     "user",
        Content:  fmt.Sprintf("Step %d: %s\n\nFindings: %s\n\nHypothesis: %s", state.StepNumber, state.Step, state.Findings, state.Hypothesis),
        ToolName: t.name,
    })

    // If more steps needed, return guidance
    if state.NextStepRequired {
        guidance := t.getNextStepGuidance(state)
        return tools.NewToolResult(t.BuildGuidanceResponse(state, guidance)), nil
    }

    // Final step - call expert model if enabled
    if state.UseAssistant {
        consolidated := t.ConsolidateFindings(thread, state.Findings)

        expertPrompt := fmt.Sprintf(`Analyze this debugging investigation and provide your expert assessment.

## Investigation Summary
%s

## Final Hypothesis
%s

## Files Examined
%v

Please provide:
1. Root cause analysis
2. Recommended fix
3. Prevention strategies
4. Any additional considerations`, consolidated, state.Hypothesis, state.FilesChecked)

        resp, err := t.CallExpertModel(ctx, expertPrompt, t.getSystemPrompt())
        if err != nil {
            return nil, fmt.Errorf("expert analysis: %w", err)
        }

        t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
            Role:     "assistant",
            Content:  resp.Content,
            ToolName: t.name,
        })

        result := fmt.Sprintf("## Debug Analysis Complete\n\n%s\n\n---\ncontinuation_id: %s",
            resp.Content, thread.ThreadID)
        return tools.NewToolResult(result), nil
    }

    // No expert model - return consolidated findings
    result := fmt.Sprintf("## Debug Investigation Complete\n\n**Hypothesis:** %s\n\n**Findings:**\n%s\n\n---\ncontinuation_id: %s",
        state.Hypothesis, state.Findings, thread.ThreadID)
    return tools.NewToolResult(result), nil
}

func (t *DebugTool) getNextStepGuidance(state *WorkflowState) string {
    switch state.Confidence {
    case types.ConfidenceExploring:
        return `Continue exploring the issue:
- Identify symptoms and error messages
- Trace the code path involved
- Look for recent changes that might be related`

    case types.ConfidenceLow:
        return `Gather more evidence:
- Examine related files and dependencies
- Check logs and error outputs
- Test your initial hypothesis with specific scenarios`

    case types.ConfidenceMedium:
        return `Validate your hypothesis:
- Create a minimal reproduction case
- Check edge cases and boundary conditions
- Look for similar patterns elsewhere in the codebase`

    case types.ConfidenceHigh, types.ConfidenceVeryHigh:
        return `Finalize your analysis:
- Document the root cause clearly
- Identify the specific fix needed
- Consider any side effects of the fix`

    default:
        return `Continue investigating and update your findings.`
    }
}

func (t *DebugTool) getSystemPrompt() string {
    return `You are an expert software debugger. Your role is to:
1. Analyze debugging investigations systematically
2. Identify root causes with precision
3. Recommend specific, actionable fixes
4. Suggest prevention strategies

Be thorough but concise. Reference specific code locations when possible.
If the evidence is inconclusive, say so clearly.`
}
```

## File Utilities (internal/utils/files.go)

```go
package utils

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// FileContent holds file path and content
type FileContent struct {
    Path    string
    Content string
}

// ReadFiles reads multiple files
func ReadFiles(paths []string, workDir string) ([]FileContent, error) {
    var results []FileContent

    for _, p := range paths {
        // Validate path
        if !filepath.IsAbs(p) {
            p = filepath.Join(workDir, p)
        }

        if err := validatePath(p, workDir); err != nil {
            continue // Skip invalid paths
        }

        content, err := os.ReadFile(p)
        if err != nil {
            continue // Skip unreadable files
        }

        results = append(results, FileContent{
            Path:    p,
            Content: string(content),
        })
    }

    return results, nil
}

// validatePath ensures the path is safe to read
func validatePath(path, workDir string) error {
    // Clean and resolve the path
    cleanPath := filepath.Clean(path)

    // Ensure it's absolute
    if !filepath.IsAbs(cleanPath) {
        return fmt.Errorf("path must be absolute")
    }

    // Check for path traversal
    if strings.Contains(cleanPath, "..") {
        return fmt.Errorf("path traversal not allowed")
    }

    // Check if path exists
    info, err := os.Stat(cleanPath)
    if err != nil {
        return err
    }

    // Don't allow directories
    if info.IsDir() {
        return fmt.Errorf("path is a directory")
    }

    // Check file size (limit to 1MB)
    if info.Size() > 1024*1024 {
        return fmt.Errorf("file too large")
    }

    return nil
}

// IsBinaryFile checks if a file appears to be binary
func IsBinaryFile(path string) bool {
    ext := strings.ToLower(filepath.Ext(path))
    binaryExts := map[string]bool{
        ".exe": true, ".dll": true, ".so": true, ".dylib": true,
        ".zip": true, ".tar": true, ".gz": true, ".7z": true,
        ".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
        ".pdf": true, ".doc": true, ".docx": true,
        ".bin": true, ".dat": true,
    }
    return binaryExts[ext]
}

// IsCodeFile checks if a file is source code
func IsCodeFile(path string) bool {
    ext := strings.ToLower(filepath.Ext(path))
    codeExts := map[string]bool{
        ".go": true, ".py": true, ".js": true, ".ts": true,
        ".jsx": true, ".tsx": true, ".java": true, ".c": true,
        ".cpp": true, ".h": true, ".hpp": true, ".rs": true,
        ".rb": true, ".php": true, ".swift": true, ".kt": true,
        ".cs": true, ".fs": true, ".scala": true, ".clj": true,
        ".ex": true, ".exs": true, ".erl": true, ".hs": true,
        ".ml": true, ".vue": true, ".svelte": true,
    }
    return codeExts[ext]
}
```

## Next Steps

Continue to [05-CONVERSATION-MEMORY.md](./05-CONVERSATION-MEMORY.md) for the conversation memory system.
