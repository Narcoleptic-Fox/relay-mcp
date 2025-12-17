package workflow

import (
	    "context"
	    "fmt"
	    "strings"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/memory"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/providers"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
	)
	// PlannerTool builds plans through interactive, sequential planning
type PlannerTool struct {
	*WorkflowTool
}

// NewPlannerTool creates a new planner tool
func NewPlannerTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *PlannerTool {
	tool := &PlannerTool{
		WorkflowTool: NewWorkflowTool(
			"planner",
			"Breaks down complex tasks through interactive, sequential planning with revision and branching.",
			cfg, registry, mem,
		),
	}

	// Add planner-specific schema fields
	tool.schema.
		AddBoolean("is_step_revision", "True when replacing a previous step", false).
		AddInteger("revises_step_number", "Step number being replaced", false, intPtr(1), nil).
		AddBoolean("is_branch_point", "True when creating a new branch", false).
		AddString("branch_id", "Name for this branch (e.g., 'approach-A')", false).
		AddInteger("branch_from_step", "Step number this branch starts from", false, intPtr(1), nil).
		AddBoolean("more_steps_needed", "True when more steps expected", false)

	return tool
}

// PlannerState holds the state for planning workflow
type PlannerState struct {
	*WorkflowState
	IsRevision      bool
	RevisesStep     int
	IsBranchPoint   bool
	BranchID        string
	BranchFromStep  int
	MoreStepsNeeded bool
	Steps           []PlanStep
}

// PlanStep represents a step in the plan
type PlanStep struct {
	Number   int
	Content  string
	BranchID string
	Revised  bool
}

func (t *PlannerTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	state, err := t.parsePlannerState(args)
	if err != nil {
		return nil, err
	}

	// Get or create thread
	thread, _ := t.GetOrCreateThread(state.ContinuationID)
	state.ContinuationID = thread.ThreadID

	// Record this step
	// stepRecord := PlanStep{
	//     Number:   state.StepNumber,
	//     Content:  state.Step,
	//     BranchID: state.BranchID,
	//     Revised:  state.IsRevision,
	// }

	    // Save to memory
	    _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
	        Role:     "user",
	        Content:  fmt.Sprintf("Step %d: %s", state.StepNumber, state.Step),
	        ToolName: t.name,
	    })
		// If more steps needed, return guidance
	if state.NextStepRequired {
		guidance := t.getStepGuidance(state)
		return tools.NewToolResult(t.buildPlannerResponse(state, guidance)), nil
	}

	// Final step - get expert analysis if enabled
	if state.UseAssistant {
		// Gather all steps from conversation
		allSteps := t.gatherSteps(thread)

		expertPrompt := fmt.Sprintf(`Review this implementation plan:

## Steps
%s

## Current Findings
%s

Please provide:
1. Plan completeness assessment
2. Potential gaps or missing steps
3. Risk areas to watch
4. Suggested order of execution
5. Dependencies between steps`, allSteps, state.Findings)

		resp, err := t.CallExpertModel(ctx, expertPrompt, t.getPlannerSystemPrompt())
		if err != nil {
			return nil, fmt.Errorf("expert analysis: %w", err)
		}

		        _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
		            Role:     "assistant",
		            Content:  resp.Content,
		            ToolName: t.name,
		        })
				result := fmt.Sprintf("## Plan Analysis Complete\n\n%s\n\n---\ncontinuation_id: %s",
			resp.Content, thread.ThreadID)
		return tools.NewToolResult(result), nil
	}

	// Return final plan summary
	result := fmt.Sprintf("## Plan Complete\n\n%s\n\n---\ncontinuation_id: %s",
		state.Findings, thread.ThreadID)
	return tools.NewToolResult(result), nil
}

func (t *PlannerTool) getStepGuidance(state *PlannerState) string {
	if state.IsBranchPoint {
		return fmt.Sprintf(`Branch point created: **%s**

Continue exploring this alternative approach.
Document how it differs from the main branch.
You can switch back to the main branch or continue here.`, state.BranchID)
	}

	if state.IsRevision {
		return fmt.Sprintf(`Step %d revised.

Continue refining the plan or proceed to the next step.
Consider how this revision affects subsequent steps.`, state.RevisesStep)
	}

	return `Continue developing the plan:
- Add implementation details
- Identify dependencies
- Note any open questions
- Consider alternative approaches if needed`
}

func (t *PlannerTool) gatherSteps(thread *types.ThreadContext) string {
	var steps []string
	for _, turn := range thread.Turns {
		if turn.Role == "user" && strings.Contains(turn.Content, "Step") {
			steps = append(steps, turn.Content)
		}
	}
	return strings.Join(steps, "\n\n")
}

func (t *PlannerTool) getPlannerSystemPrompt() string {
	return `You are a senior software architect reviewing implementation plans.
Your role is to:
1. Ensure plans are complete and actionable
2. Identify gaps, risks, and dependencies
3. Suggest improvements to the plan structure
4. Validate feasibility

Be thorough but constructive. Focus on practical concerns.
Do not add time estimates - focus on what needs to be done.`
}

func (t *PlannerTool) buildPlannerResponse(state *PlannerState, guidance string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Planning Step %d/%d", state.StepNumber, state.TotalSteps))

	if state.BranchID != "" {
		sb.WriteString(fmt.Sprintf(" [Branch: %s]", state.BranchID))
	}
	sb.WriteString("\n\n")

	sb.WriteString(guidance)
	sb.WriteString(fmt.Sprintf("\n\n---\ncontinuation_id: %s", state.ContinuationID))

	return sb.String()
}

func (t *PlannerTool) parsePlannerState(args map[string]any) (*PlannerState, error) {
	base, err := t.ParseWorkflowState(args)
	if err != nil {
		return nil, err
	}

	parser := tools.NewArgumentParser(args)

	return &PlannerState{
		WorkflowState:   base,
		IsRevision:      parser.GetBool("is_step_revision", false),
		RevisesStep:     parser.GetInt("revises_step_number", 0),
		IsBranchPoint:   parser.GetBool("is_branch_point", false),
		BranchID:        parser.GetString("branch_id"),
		BranchFromStep:  parser.GetInt("branch_from_step", 0),
		MoreStepsNeeded: parser.GetBool("more_steps_needed", false),
	}, nil
}
