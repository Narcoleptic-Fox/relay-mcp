# 07 - Consensus & Multi-Model Workflows

## Overview

The consensus system enables multi-model orchestration where multiple AI models debate and synthesize recommendations. This is one of the key features for AI-to-AI communication.

## Consensus Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                       Claude Code                                 │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                Step 1: Define proposal + models
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                     Consensus Tool                                │
│  proposal: "Should we use microservices?"                        │
│  models: [{gpt-5, for}, {pro, against}, {o3, neutral}]          │
└─────────────────────────────┬────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
    Step 2             Step 3              Step 4
         ▼                    ▼                    ▼
   ┌──────────┐        ┌──────────┐        ┌──────────┐
   │  GPT-5   │        │  Gemini  │        │   O3     │
   │  (FOR)   │        │(AGAINST) │        │(NEUTRAL) │
   └────┬─────┘        └────┬─────┘        └────┬─────┘
        │                   │                   │
        └───────────────────┼───────────────────┘
                            │
                       Step 5: Synthesize
                            │
                            ▼
             ┌─────────────────────────┐
             │   Unified Recommendation │
             │   with all perspectives  │
             └─────────────────────────┘
```

## Consensus Tool (internal/tools/workflow/consensus.go)

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

// ConsensusTool orchestrates multi-model debate and synthesis
type ConsensusTool struct {
    *WorkflowTool
}

// NewConsensusTool creates a new consensus tool
func NewConsensusTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *ConsensusTool {
    tool := &ConsensusTool{
        WorkflowTool: NewWorkflowTool(
            "consensus",
            "Builds multi-model consensus through systematic analysis and structured debate.",
            cfg, registry, mem,
        ),
    }

    // Add consensus-specific schema fields
    tool.schema.
        AddObjectArray("models", "Models to consult with stance", true, map[string]any{
            "model": map[string]any{
                "type":        "string",
                "description": "Model name",
            },
            "stance": map[string]any{
                "type":        "string",
                "description": "Stance: for, against, or neutral",
                "enum":        []string{"for", "against", "neutral"},
            },
            "stance_prompt": map[string]any{
                "type":        "string",
                "description": "Custom prompt for this stance",
            },
        }).
        AddInteger("current_model_index", "Current model index (0-based)", false, intPtr(0), nil).
        AddObjectArray("model_responses", "Accumulated model responses", false, nil)

    return tool
}

// ConsensusState holds the state for consensus workflow
type ConsensusState struct {
    *WorkflowState
    Proposal          string
    Models            []ConsensusModel
    CurrentModelIndex int
    ModelResponses    []ModelResponseRecord
}

// ConsensusModel defines a model to consult
type ConsensusModel struct {
    Model       string       `json:"model"`
    Stance      types.Stance `json:"stance"`
    StancePrompt string      `json:"stance_prompt,omitempty"`
}

// ModelResponseRecord holds a model's response
type ModelResponseRecord struct {
    Model    string       `json:"model"`
    Stance   types.Stance `json:"stance"`
    Response string       `json:"response"`
}

func (t *ConsensusTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
    state, err := t.parseConsensusState(args)
    if err != nil {
        return nil, err
    }

    // Get or create thread
    thread, _ := t.GetOrCreateThread(state.ContinuationID)
    state.ContinuationID = thread.ThreadID

    // Step 1: CLI provides the proposal
    if state.StepNumber == 1 {
        // Validate models
        if len(state.Models) < 2 {
            return nil, fmt.Errorf("consensus requires at least 2 models")
        }

        // Check for duplicate model+stance combinations
        if err := t.validateModels(state.Models); err != nil {
            return nil, err
        }

        // Store proposal in thread
        t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
            Role:     "user",
            Content:  fmt.Sprintf("Proposal: %s\n\nModels: %v", state.Step, state.Models),
            ToolName: t.name,
        })

        // Return guidance for next step
        guidance := fmt.Sprintf(`Proposal recorded. Ready to consult %d models.

**Models to consult:**
%s

Proceed to step 2 to start consulting models.`, len(state.Models), t.formatModels(state.Models))

        return tools.NewToolResult(t.buildResponse(state, guidance)), nil
    }

    // Steps 2 to N: Consult models one by one
    if state.NextStepRequired && state.CurrentModelIndex < len(state.Models) {
        model := state.Models[state.CurrentModelIndex]

        // Generate response from this model
        response, err := t.consultModel(ctx, model, state)
        if err != nil {
            return nil, fmt.Errorf("consulting model %s: %w", model.Model, err)
        }

        // Record the response
        state.ModelResponses = append(state.ModelResponses, ModelResponseRecord{
            Model:    model.Model,
            Stance:   model.Stance,
            Response: response,
        })

        // Save to thread
        t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
            Role:     "assistant",
            Content:  fmt.Sprintf("[%s - %s stance]\n%s", model.Model, model.Stance, response),
            ToolName: t.name,
        })

        // Check if more models to consult
        if state.CurrentModelIndex+1 < len(state.Models) {
            guidance := fmt.Sprintf(`## Model %d/%d: %s (%s stance)

%s

---
Proceed to next step to consult: %s`,
                state.CurrentModelIndex+1,
                len(state.Models),
                model.Model,
                model.Stance,
                response,
                state.Models[state.CurrentModelIndex+1].Model,
            )
            return tools.NewToolResult(t.buildResponse(state, guidance)), nil
        }
    }

    // Final step: Synthesize all responses
    synthesis, err := t.synthesize(ctx, state)
    if err != nil {
        return nil, fmt.Errorf("synthesizing: %w", err)
    }

    // Save synthesis
    t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
        Role:     "assistant",
        Content:  synthesis,
        ToolName: t.name,
    })

    result := fmt.Sprintf(`## Consensus Analysis Complete

%s

---
continuation_id: %s`, synthesis, thread.ThreadID)

    return tools.NewToolResult(result), nil
}

// consultModel calls a specific model with its stance
func (t *ConsensusTool) consultModel(ctx context.Context, model ConsensusModel, state *ConsensusState) (string, error) {
    // Get provider for this model
    provider, err := t.registry.GetProviderForModel(model.Model)
    if err != nil {
        return "", err
    }

    // Build stance-specific prompt
    prompt := t.buildStancePrompt(model, state)
    systemPrompt := t.getStanceSystemPrompt(model.Stance)

    slog.Info("consulting model",
        "model", model.Model,
        "stance", model.Stance,
        "provider", provider.GetProviderType(),
    )

    resp, err := provider.GenerateContent(ctx, &providers.GenerateRequest{
        Prompt:       prompt,
        SystemPrompt: systemPrompt,
        Model:        model.Model,
        Temperature:  0.7,
    })
    if err != nil {
        return "", err
    }

    return resp.Content, nil
}

func (t *ConsensusTool) buildStancePrompt(model ConsensusModel, state *ConsensusState) string {
    var sb strings.Builder

    sb.WriteString("## Proposal\n\n")
    sb.WriteString(state.Proposal)
    sb.WriteString("\n\n")

    if model.StancePrompt != "" {
        sb.WriteString("## Specific Instructions\n\n")
        sb.WriteString(model.StancePrompt)
        sb.WriteString("\n\n")
    }

    // Include previous responses for context
    if len(state.ModelResponses) > 0 {
        sb.WriteString("## Previous Perspectives\n\n")
        for _, resp := range state.ModelResponses {
            sb.WriteString(fmt.Sprintf("### %s (%s)\n%s\n\n", resp.Model, resp.Stance, truncateText(resp.Response, 500)))
        }
    }

    sb.WriteString("## Your Task\n\n")
    sb.WriteString(fmt.Sprintf("Evaluate this proposal from a **%s** perspective. ", model.Stance))

    switch model.Stance {
    case types.StanceFor:
        sb.WriteString("Advocate for why this proposal is a good idea. Highlight benefits and opportunities.")
    case types.StanceAgainst:
        sb.WriteString("Critically analyze this proposal. Identify risks, problems, and potential issues.")
    case types.StanceNeutral:
        sb.WriteString("Provide a balanced evaluation. Weigh both benefits and drawbacks objectively.")
    }

    return sb.String()
}

func (t *ConsensusTool) getStanceSystemPrompt(stance types.Stance) string {
    switch stance {
    case types.StanceFor:
        return `You are an advocate analyzing a proposal.
Your role is to identify and articulate the strengths, benefits, and opportunities.
Be persuasive but honest. Don't invent benefits that don't exist.
Structure your response with clear arguments.`

    case types.StanceAgainst:
        return `You are a critical analyst evaluating a proposal.
Your role is to identify risks, problems, potential failures, and downsides.
Be thorough but fair. Don't manufacture problems that aren't real.
Structure your response with specific concerns and their implications.`

    case types.StanceNeutral:
        return `You are an objective analyst evaluating a proposal.
Your role is to provide a balanced assessment weighing both pros and cons.
Consider multiple perspectives. Be fair to all sides.
Structure your response to cover benefits, risks, and overall recommendation.`

    default:
        return `Analyze the given proposal thoroughly and provide your assessment.`
    }
}

// synthesize combines all model responses into a unified recommendation
func (t *ConsensusTool) synthesize(ctx context.Context, state *ConsensusState) (string, error) {
    // Use the best available model for synthesis
    caps, provider, err := t.registry.SelectBestModel(providers.ModelRequirements{
        MinIntelligence: 80,
    })
    if err != nil {
        return "", err
    }

    // Build synthesis prompt
    var sb strings.Builder
    sb.WriteString("## Proposal\n\n")
    sb.WriteString(state.Proposal)
    sb.WriteString("\n\n## Model Perspectives\n\n")

    for _, resp := range state.ModelResponses {
        sb.WriteString(fmt.Sprintf("### %s (%s stance)\n%s\n\n", resp.Model, resp.Stance, resp.Response))
    }

    sb.WriteString(`## Your Task

Synthesize these perspectives into a unified recommendation:

1. **Summary of Key Points**: Main arguments from each perspective
2. **Areas of Agreement**: Where models converge
3. **Areas of Disagreement**: Where models diverge and why
4. **Unified Recommendation**: Your synthesized recommendation
5. **Confidence Level**: How confident you are in this recommendation
6. **Next Steps**: Actionable items if the recommendation is adopted`)

    systemPrompt := `You are a senior decision-maker synthesizing multiple expert perspectives.
Your role is to:
- Understand each perspective fairly
- Identify patterns and key insights
- Synthesize a balanced, actionable recommendation
- Be clear about trade-offs and confidence levels

Structure your response clearly with the sections requested.`

    resp, err := provider.GenerateContent(ctx, &providers.GenerateRequest{
        Prompt:       sb.String(),
        SystemPrompt: systemPrompt,
        Model:        caps.ModelName,
        Temperature:  0.5,
    })
    if err != nil {
        return "", err
    }

    return resp.Content, nil
}

func (t *ConsensusTool) parseConsensusState(args map[string]any) (*ConsensusState, error) {
    base, err := t.ParseWorkflowState(args)
    if err != nil {
        return nil, err
    }

    parser := tools.NewArgumentParser(args)

    state := &ConsensusState{
        WorkflowState:     base,
        Proposal:          base.Step, // Step 1 contains the proposal
        CurrentModelIndex: parser.GetInt("current_model_index", 0),
    }

    // Parse models
    modelsRaw := parser.GetObjectArray("models")
    for _, m := range modelsRaw {
        model := ConsensusModel{
            Model:        getString(m, "model"),
            Stance:       types.Stance(getString(m, "stance")),
            StancePrompt: getString(m, "stance_prompt"),
        }
        if model.Model != "" {
            state.Models = append(state.Models, model)
        }
    }

    // Parse existing responses
    responsesRaw := parser.GetObjectArray("model_responses")
    for _, r := range responsesRaw {
        record := ModelResponseRecord{
            Model:    getString(r, "model"),
            Stance:   types.Stance(getString(r, "stance")),
            Response: getString(r, "response"),
        }
        state.ModelResponses = append(state.ModelResponses, record)
    }

    return state, nil
}

func (t *ConsensusTool) validateModels(models []ConsensusModel) error {
    seen := make(map[string]bool)
    for _, m := range models {
        key := fmt.Sprintf("%s:%s", m.Model, m.Stance)
        if seen[key] {
            return fmt.Errorf("duplicate model+stance combination: %s with %s stance", m.Model, m.Stance)
        }
        seen[key] = true
    }
    return nil
}

func (t *ConsensusTool) formatModels(models []ConsensusModel) string {
    var lines []string
    for i, m := range models {
        lines = append(lines, fmt.Sprintf("%d. %s (%s)", i+1, m.Model, m.Stance))
    }
    return strings.Join(lines, "\n")
}

func (t *ConsensusTool) buildResponse(state *ConsensusState, guidance string) string {
    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("## Consensus Step %d/%d\n\n", state.StepNumber, state.TotalSteps))
    sb.WriteString(guidance)
    sb.WriteString(fmt.Sprintf("\n\n---\ncontinuation_id: %s", state.ContinuationID))

    return sb.String()
}

func getString(m map[string]any, key string) string {
    if v, ok := m[key].(string); ok {
        return v
    }
    return ""
}

func truncateText(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}
```

## Planner Tool (internal/tools/workflow/planner.go)

```go
package workflow

import (
    "context"
    "fmt"
    "strings"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/memory"
    "github.com/yourorg/pal-mcp/internal/providers"
    "github.com/yourorg/pal-mcp/internal/tools"
    "github.com/yourorg/pal-mcp/internal/types"
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
    stepRecord := PlanStep{
        Number:   state.StepNumber,
        Content:  state.Step,
        BranchID: state.BranchID,
        Revised:  state.IsRevision,
    }

    // Save to memory
    t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
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

        t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
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
```

## ThinkDeep Tool (internal/tools/workflow/thinkdeep.go)

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
    t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
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

        t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
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
```

## Usage Examples

### Consensus Example

```
User: Use consensus to decide if we should migrate to microservices

Claude: [Calls consensus tool with:]
- models: [{gpt-5, for}, {pro, against}, {o3, neutral}]
- step: "Should we migrate our monolithic app to microservices?"

PAL Server:
  Step 1: Records proposal
  Step 2: Calls GPT-5 with "for" stance → advocates for microservices
  Step 3: Calls Gemini Pro with "against" stance → highlights risks
  Step 4: Calls O3 with "neutral" stance → balanced view
  Step 5: Synthesizes into unified recommendation

Result: Comprehensive recommendation with all perspectives
```

### ThinkDeep Example

```
User: Analyze why our API response times are increasing

Claude: [Calls thinkdeep with:]
- step: "Initial investigation of API latency"
- findings: "Average response time increased 3x over 2 weeks"
- focus_areas: ["performance", "database"]

[Multiple steps of investigation...]

PAL Server: Returns expert analysis with root cause and recommendations
```

## Next Steps

Continue to [08-CONFIG-DEPLOY.md](./08-CONFIG-DEPLOY.md) for configuration and deployment.
