package workflow

import (
	"context"
	"fmt"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/config"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/memory"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/providers"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/tools"
	"github.com/Narcoleptic-Fox/zen-mcp/internal/types"
)

// PrecommitTool performs pre-commit validation
type PrecommitTool struct {
	*WorkflowTool
}

// NewPrecommitTool creates a new precommit tool
func NewPrecommitTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *PrecommitTool {
	tool := &PrecommitTool{
		WorkflowTool: NewWorkflowTool(
			"precommit",
			"Performs validation checks before committing code.",
			cfg, registry, mem,
		),
	}

	tool.schema.
		AddStringArray("files", "Staged files", true)

	return tool
}

func (t *PrecommitTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	state, err := t.ParseWorkflowState(args)
	if err != nil {
		return nil, err
	}

	thread, _ := t.GetOrCreateThread(state.ContinuationID)
	state.ContinuationID = thread.ThreadID

	    _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
	        Role:     "user",
	        Content:  fmt.Sprintf("Precommit Step %d: %s", state.StepNumber, state.Step),
	        ToolName: t.name,
	    })
		if state.UseAssistant {
		resp, err := t.CallExpertModel(ctx, fmt.Sprintf("Validate commit: %s", state.Findings), "You are a code quality gatekeeper.")
		if err != nil {
			return nil, err
		}
		return tools.NewToolResult(resp.Content), nil
	}

	return tools.NewToolResult("Pre-commit check recorded."), nil
}
