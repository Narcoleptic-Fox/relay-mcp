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

// RefactorTool plans and validates code refactoring
type RefactorTool struct {
	*WorkflowTool
}

// NewRefactorTool creates a new refactor tool
func NewRefactorTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *RefactorTool {
	tool := &RefactorTool{
		WorkflowTool: NewWorkflowTool(
			"refactor",
			"Plans and validates code refactoring strategies.",
			cfg, registry, mem,
		),
	}

	tool.schema.
		AddString("goal", "Refactoring goal", true).
		AddStringArray("files", "Files to refactor", true)

	return tool
}

func (t *RefactorTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	state, err := t.ParseWorkflowState(args)
	if err != nil {
		return nil, err
	}

	// Get thread
	thread, _ := t.GetOrCreateThread(state.ContinuationID)
	state.ContinuationID = thread.ThreadID

	    _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
	        Role:     "user",
	        Content:  fmt.Sprintf("Refactor Step %d: %s", state.StepNumber, state.Step),
	        ToolName: t.name,
	    })
		if state.UseAssistant {
		resp, err := t.CallExpertModel(ctx, fmt.Sprintf("Analyze refactoring: %s", state.Findings), "You are a refactoring expert.")
		if err != nil {
			return nil, err
		}
		return tools.NewToolResult(resp.Content), nil
	}

	return tools.NewToolResult("Refactoring step recorded."), nil
}
