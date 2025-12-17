package workflow

import (
	    "context"
	    "fmt"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/memory"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/providers"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
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
	thread, _ := t.GetOrCreateThread(state.ContinuationID)
	state.ContinuationID = thread.ThreadID

	    // Save this step to memory
	    _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
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

		        _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
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
