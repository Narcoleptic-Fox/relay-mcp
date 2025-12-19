package workflow

import (
	"context"
	"fmt"
	    "log/slog"
	    "strings"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/memory"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/providers"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/types"
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
	Model        string       `json:"model"`
	Stance       types.Stance `json:"stance"`
	StancePrompt string       `json:"stance_prompt,omitempty"`
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
		t.AddTurn(thread.ThreadID, types.ConversationTurn{
			Role:    "user",
			Content: fmt.Sprintf("Proposal: %s\n\nModels: %v", state.Step, state.Models),
		})
				// Return guidance for next step
		guidance := fmt.Sprintf(`Proposal recorded. Ready to consult %d models.

**Models to consult:**
%s

Proceed to step 2 to start consulting models.`, len(state.Models), t.formatModels(state.Models))

		return tools.NewToolResult(t.buildResponse(state, guidance, 0)), nil
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
		t.AddTurn(thread.ThreadID, types.ConversationTurn{
			Role:    "assistant",
			Content: fmt.Sprintf("[%s - %s stance]\n%s", model.Model, model.Stance, response),
		})
				// Check if more models to consult
		nextIndex := state.CurrentModelIndex + 1
		if nextIndex < len(state.Models) {
			nextModel := state.Models[nextIndex]
			guidance := fmt.Sprintf(`## Model %d/%d: %s (%s stance)

%s

---
Proceed to next step to consult: %s`,
				state.CurrentModelIndex+1,
				len(state.Models),
				model.Model,
				model.Stance,
				response,
				nextModel.Model,
			)
			return tools.NewToolResult(t.buildResponse(state, guidance, nextIndex)), nil
		}
	}

	// Final step: Synthesize all responses
	synthesis, err := t.synthesize(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("synthesizing: %w", err)
	}

	// Save synthesis
	t.AddTurn(thread.ThreadID, types.ConversationTurn{
		Role:    "assistant",
		Content: synthesis,
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

func (t *ConsensusTool) buildResponse(state *ConsensusState, guidance string, nextModelIndex int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Consensus Step %d/%d\n\n", state.StepNumber, state.TotalSteps))
	sb.WriteString(guidance)
	sb.WriteString(fmt.Sprintf("\n\n---\ncontinuation_id: %s", state.ContinuationID))
	sb.WriteString(fmt.Sprintf("\ncurrent_model_index: %d", nextModelIndex))
	sb.WriteString(fmt.Sprintf("\nmodels_consulted: %d/%d", len(state.ModelResponses), len(state.Models)))

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
