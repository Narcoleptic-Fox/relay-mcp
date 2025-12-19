package simple

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Narcoleptic-Fox/relay-mcp/internal/clink"
	"github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	"github.com/Narcoleptic-Fox/relay-mcp/internal/memory"
	"github.com/Narcoleptic-Fox/relay-mcp/internal/tools"
	"github.com/Narcoleptic-Fox/relay-mcp/internal/types"
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
	registry, err := clink.NewRegistry(cfg)
	if err != nil {
		slog.Error("failed to create clink registry", "error", err)
		registry = clink.NewEmptyRegistry()
	}

	tool := &ClinkTool{
		name: "clink",
		description: "Link to an external AI CLI (Gemini CLI, Claude CLI, Codex CLI) " +
			"to reuse their capabilities inside existing workflows.",
		cfg:      cfg,
		memory:   mem,
		registry: registry,
		schema:   tools.NewSchemaBuilder(),
	}

	// Define schema - use available CLIs or a descriptive message if none
	availableCLIs := registry.List()
	if len(availableCLIs) == 0 {
		// No enum restriction when no CLIs available - Execute will return clear error
		tool.schema.AddString("cli_name",
			"CLI client name (none currently configured - please configure gemini, claude, or codex)", true)
	} else {
		tool.schema.AddStringEnum("cli_name", "CLI client name", availableCLIs, true)
	}

	tool.schema.
		AddString("prompt", "User request to forward to the CLI", true).
		AddStringEnum("role", "Role preset for the CLI", []string{"default", "planner", "codereviewer"}, false).
		AddStringArray("absolute_file_paths", "File paths to share with the CLI", false).
		AddStringArray("images", "Image paths for visual context", false).
		AddString("continuation_id", "Thread ID for conversation continuation", false)

	return tool
}

func (t *ClinkTool) Name() string           { return t.name }
func (t *ClinkTool) Description() string    { return t.description }
func (t *ClinkTool) Schema() map[string]any { return t.schema.Build() }

func (t *ClinkTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	// Check if any CLIs are available
	if len(t.registry.List()) == 0 {
		return tools.NewToolError("No CLI clients are configured. Please configure at least one CLI client (gemini, claude, or codex) in your configuration."), nil
	}

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

func (t *ClinkTool) buildContextualPrompt(thread *types.ThreadContext, currentPrompt string) string {
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
