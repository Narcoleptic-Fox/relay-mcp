package simple

import (
	"context"
	"fmt"
	"strings"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/config"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/memory"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/providers"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/tools"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/utils"
)

// ChallengeTool provides critical analysis of ideas or code
type ChallengeTool struct {
	*BaseTool
}

// NewChallengeTool creates a new challenge tool
func NewChallengeTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *ChallengeTool {
	tool := &ChallengeTool{
		BaseTool: NewBaseTool("challenge", "Critically analyze ideas, code, or architecture decisions to find flaws and improvements.", cfg, registry, mem),
	}

	tool.schema.
		AddString("topic", "The idea, code, or decision to analyze", true).
		AddString("working_directory_absolute_path", "Absolute path to working directory", false).
		AddStringArray("absolute_file_paths", "Related file paths", false).
		AddString("model", "Model to use", false).
		AddString("continuation_id", "Thread ID", false)

	return tool
}

func (t *ChallengeTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	parser := tools.NewArgumentParser(args)

	topic, err := parser.GetStringRequired("topic")
	if err != nil {
		return nil, err
	}
	workDir := parser.GetString("working_directory_absolute_path")
	filePaths := parser.GetStringArray("absolute_file_paths")
	modelName := parser.GetString("model")
	continuationID := parser.GetString("continuation_id")

	// Get thread
	thread, _ := t.GetOrCreateThread(continuationID)

	// Resolve model
	resolvedModel, provider, err := t.ResolveModel(modelName)
	if err != nil {
		return nil, fmt.Errorf("resolving model: %w", err)
	}

	// Read files
	var fileContents []utils.FileContent
	if len(filePaths) > 0 && workDir != "" {
		fileContents, _ = utils.ReadFiles(filePaths, workDir)
	}

	// Build prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Please critically analyze this topic: %s\n\n", topic))

	if len(fileContents) > 0 {
		sb.WriteString("## Context Files\n\n")
		for _, f := range fileContents {
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", f.Path, f.Content))
		}
	}

	sb.WriteString(`

Please play Devil's Advocate and provide a critical analysis:
1. Potential flaws or edge cases
2. Security implications
3. Performance bottlenecks
4. Maintenance or scalability concerns
5. Alternative approaches that might be better

Be constructive but rigorous.`)

	// Generate response
	resp, err := t.GenerateContent(ctx, provider, &providers.GenerateRequest{
		Prompt:              sb.String(),
		SystemPrompt:        "You are a senior principal engineer performing a critical design review. Your goal is to find flaws before they become problems.",
		Model:               resolvedModel,
		ConversationHistory: t.memory.GetHistory(thread.ThreadID),
	})
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	// Save to memory
	t.AddTurn(thread.ThreadID, "user", fmt.Sprintf("Challenge: %s", topic), filePaths, nil)
	t.AddTurn(thread.ThreadID, "assistant", resp.Content, nil, nil)

	result := fmt.Sprintf("%s\n\n---\ncontinuation_id: %s", resp.Content, thread.ThreadID)
	return tools.NewToolResult(result), nil
}
