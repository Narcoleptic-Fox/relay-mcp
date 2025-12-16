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

// ThinkDeepTool performs multi-stage investigation and reasoning
type ThinkDeepTool struct {
	*WorkflowTool
}

// NewThinkDeepTool creates a new thinkdeep tool
func NewThinkDeepTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *ThinkDeepTool {
	tool := &ThinkDeepTool{
		WorkflowTool: NewWorkflowTool(
			"thinkdeep",
			"Performs multi-stage investigation and reasoning for complex problem analysis.",
			cfg, registry, mem,
		),
	}

	// Add thinkdeep-specific schema
	tool.schema.
		AddStringArray("focus_areas", "Areas to focus on (architecture, performance, security)", false).
		AddString("problem_context", "Additional context about the problem", false).
		AddStringArray("relevant_context", "Methods/functions involved in the issue", false).
		AddObjectArray("issues_found", "Issues with severity levels", false, nil)

	return tool
}

func (t *ThinkDeepTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	state, err := t.ParseWorkflowState(args)
	if err != nil {
		return nil, err
	}

	parser := tools.NewArgumentParser(args)
	focusAreas := parser.GetStringArray("focus_areas")
	problemContext := parser.GetString("problem_context")

	// Get or create thread
	thread, _ := t.GetOrCreateThread(state.ContinuationID)
	state.ContinuationID = thread.ThreadID

	    // Save this investigation step
	    _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
	        Role: "user",
	        Content: fmt.Sprintf("Step %d: %s\n\nFindings: %s\n\nHypothesis: %s\n\nFocus: %v",
	            state.StepNumber, state.Step, state.Findings, state.Hypothesis, focusAreas),
	        ToolName: t.name,
	    })
		// If more steps needed, provide guidance
	if state.NextStepRequired {
		guidance := t.getInvestigationGuidance(state, focusAreas)
		return tools.NewToolResult(t.BuildGuidanceResponse(state, guidance)), nil
	}

	// Final analysis
	if state.UseAssistant {
		consolidated := t.ConsolidateFindings(thread, state.Findings)

		expertPrompt := fmt.Sprintf(`Analyze this investigation and provide expert insights.

## Problem Context
%s

## Investigation Summary
%s

## Current Hypothesis
%s

## Focus Areas
%v

Provide:
1. Assessment of the investigation
2. Validation or refinement of the hypothesis
3. Key insights and recommendations
4. Areas that may need further investigation`, problemContext, consolidated, state.Hypothesis, focusAreas)

		resp, err := t.CallExpertModel(ctx, expertPrompt, t.getThinkDeepSystemPrompt())
		if err != nil {
			return nil, fmt.Errorf("expert analysis: %w", err)
		}

		        _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
		            Role:     "assistant",
		            Content:  resp.Content,
		            ToolName: t.name,
		        })
				result := fmt.Sprintf("## Deep Analysis Complete\n\n%s\n\n---\ncontinuation_id: %s",
			resp.Content, thread.ThreadID)
		return tools.NewToolResult(result), nil
	}

	result := fmt.Sprintf("## Analysis Complete\n\n**Hypothesis:** %s\n\n**Findings:**\n%s\n\n---\ncontinuation_id: %s",
		state.Hypothesis, state.Findings, thread.ThreadID)
	return tools.NewToolResult(result), nil
}

func (t *ThinkDeepTool) getInvestigationGuidance(state *WorkflowState, focusAreas []string) string {
	guidance := `Continue your investigation:

`
	if len(focusAreas) > 0 {
		guidance += fmt.Sprintf("**Focus Areas:** %v\n\n", focusAreas)
	}

	switch state.Confidence {
	case types.ConfidenceExploring:
		guidance += `- Gather initial information
- Identify key components involved
- Form initial hypotheses`
	case types.ConfidenceLow, types.ConfidenceMedium:
		guidance += `- Deepen your analysis
- Test your hypothesis against evidence
- Look for patterns and connections`
	case types.ConfidenceHigh, types.ConfidenceVeryHigh:
		guidance += `- Validate your conclusions
- Document key findings
- Prepare final recommendations`
	}

	return guidance
}

func (t *ThinkDeepTool) getThinkDeepSystemPrompt() string {
	return `You are an expert analyst providing deep insights on complex problems.
Your role is to:
1. Analyze investigations systematically
2. Validate or refine hypotheses
3. Identify patterns and root causes
4. Provide actionable recommendations

Be thorough and precise. Reference specific evidence.
Distinguish between facts, inferences, and speculation.`
}
