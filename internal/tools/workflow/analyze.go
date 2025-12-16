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

// AnalyzeTool performs deep codebase analysis
type AnalyzeTool struct {
	*WorkflowTool
}

// NewAnalyzeTool creates a new analyze tool
func NewAnalyzeTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *AnalyzeTool {
	tool := &AnalyzeTool{
		WorkflowTool: NewWorkflowTool(
			"analyze",
			"Performs deep analysis of codebase structure, dependencies, and patterns.",
			cfg, registry, mem,
		),
	}

	// Add analyze-specific schema
	tool.schema.
		AddStringArray("files", "Specific files to analyze", false).
		AddString("query", "Specific analysis question", true)

	return tool
}

func (t *AnalyzeTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	state, err := t.ParseWorkflowState(args)
	if err != nil {
		return nil, err
	}

	// Get or create thread
	thread, _ := t.GetOrCreateThread(state.ContinuationID)
	state.ContinuationID = thread.ThreadID

	    // Save step
	    _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
	        Role:     "user",
	        Content:  fmt.Sprintf("Analysis Step %d: %s\n\nFindings: %s", state.StepNumber, state.Step, state.Findings),
	        ToolName: t.name,
	    })
	
	    if state.UseAssistant {
	        resp, err := t.CallExpertModel(ctx, fmt.Sprintf("Analyze findings: %s", state.Findings), "You are a software architect.")
	        if err != nil {
	            return nil, err
	        }
	        
	        _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
	            Role:     "assistant",
	            Content:  resp.Content,
	            ToolName: t.name,
	        })
	        
	        return tools.NewToolResult(resp.Content), nil
	    }
		return tools.NewToolResult("Analysis step recorded."), nil
}
