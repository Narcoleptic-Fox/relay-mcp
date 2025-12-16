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

// CodeReviewTool performs systematic code review
type CodeReviewTool struct {
	*WorkflowTool
}

// NewCodeReviewTool creates a new codereview tool
func NewCodeReviewTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *CodeReviewTool {
	tool := &CodeReviewTool{
		WorkflowTool: NewWorkflowTool(
			"codereview",
			"Performs systematic code review with focus on security, performance, and best practices.",
			cfg, registry, mem,
		),
	}

	// Add codereview-specific schema
	tool.schema.
		AddStringArray("focus_areas", "Areas to focus on (security, performance, style)", false).
		AddString("pr_context", "Pull request or change context", false).
		AddBoolean("generate_fix_suggestions", "Generate code fixes for issues", false)

	return tool
}

func (t *CodeReviewTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	state, err := t.ParseWorkflowState(args)
	if err != nil {
		return nil, err
	}

	parser := tools.NewArgumentParser(args)
	focusAreas := parser.GetStringArray("focus_areas")
	prContext := parser.GetString("pr_context")
	genFixes := parser.GetBool("generate_fix_suggestions", false)

	// Get thread
	thread, _ := t.GetOrCreateThread(state.ContinuationID)
	state.ContinuationID = thread.ThreadID

	    // Save step
	    _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
	        Role: "user",
	        Content: fmt.Sprintf("Review Step %d: %s\n\nFindings: %s",
	            state.StepNumber, state.Step, state.Findings),
	        ToolName: t.name,
	    })
		// Intermediate guidance
	if state.NextStepRequired {
		guidance := `Continue the review:
- Check for security vulnerabilities
- Verify error handling
- Assess performance impact
- Ensure test coverage`
		return tools.NewToolResult(t.BuildGuidanceResponse(state, guidance)), nil
	}

	// Final analysis
	if state.UseAssistant {
		consolidated := t.ConsolidateFindings(thread, state.Findings)

		expertPrompt := fmt.Sprintf(`Perform a final code review assessment.

## Context
%s

## Review Findings
%s

## Focus Areas
%v

Provide:
1. Summary of critical issues
2. Security assessment
3. Performance impact
4. Code quality and maintainability score (1-10)
5. Actionable recommendations`, prContext, consolidated, focusAreas)

		if genFixes {
			expertPrompt += "\n6. Suggested code fixes for major issues"
		}

		resp, err := t.CallExpertModel(ctx, expertPrompt, "You are a senior principal engineer conducting a final code review sign-off.")
		if err != nil {
			return nil, fmt.Errorf("expert analysis: %w", err)
		}

		        _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
		            Role:     "assistant",
		            Content:  resp.Content,
		            ToolName: t.name,
		        })
				result := fmt.Sprintf("## Code Review Complete\n\n%s\n\n---\ncontinuation_id: %s",
			resp.Content, thread.ThreadID)
		return tools.NewToolResult(result), nil
	}

	return tools.NewToolResult(fmt.Sprintf("## Review Complete\n\n%s\n\n---\ncontinuation_id: %s",
		state.Findings, thread.ThreadID)), nil
}
