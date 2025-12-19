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
	// WorkflowTool provides common functionality for multi-step workflow tools
type WorkflowTool struct {
	name        string
	description string
	schema      *tools.SchemaBuilder
	cfg         *config.Config
	registry    *providers.Registry
	memory      *memory.ConversationMemory
}

// NewWorkflowTool creates a new workflow tool
func NewWorkflowTool(
	name, description string,
	cfg *config.Config,
	registry *providers.Registry,
	mem *memory.ConversationMemory,
) *WorkflowTool {
	wt := &WorkflowTool{
		name:        name,
		description: description,
		schema:      tools.NewSchemaBuilder(),
		cfg:         cfg,
		registry:    registry,
		memory:      mem,
	}

	// Add common workflow schema fields
	wt.addCommonSchema()

	return wt
}

func (t *WorkflowTool) addCommonSchema() {
	t.schema.
		AddString("step", "Current work step content and findings", true).
		AddInteger("step_number", "Current step number (starts at 1)", true, intPtr(1), nil).
		AddInteger("total_steps", "Estimated total steps needed", true, intPtr(1), nil).
		AddBoolean("next_step_required", "Whether another step is needed", true).
		AddString("findings", "Important discoveries and evidence", true).
		AddString("model", "Model to use", false).
		AddString("hypothesis", "Current theory based on evidence", false).
		AddStringEnum("confidence", "Confidence level", []string{
			"exploring", "low", "medium", "high", "very_high", "almost_certain", "certain",
		}, false).
		AddStringArray("relevant_files", "Files relevant to the investigation", false).
		AddStringArray("files_checked", "All files examined", false).
		AddString("continuation_id", "Thread ID for multi-turn conversations", false).
		AddBoolean("use_assistant_model", "Use expert model for analysis", false).
		AddStringEnum("thinking_mode", "Reasoning depth", []string{
			"minimal", "low", "medium", "high", "max",
		}, false).
		AddNumber("temperature", "0 = deterministic, 1 = creative", false, floatPtr(0.0), floatPtr(1.0))
}

func (t *WorkflowTool) Name() string           { return t.name }
func (t *WorkflowTool) Description() string    { return t.description }
func (t *WorkflowTool) Schema() map[string]any { return t.schema.Build() }

// WorkflowState holds the current state of a workflow
type WorkflowState struct {
	Step             string
	StepNumber       int
	TotalSteps       int
	NextStepRequired bool
	Findings         string
	Hypothesis       string
	Confidence       types.ConfidenceLevel
	RelevantFiles    []string
	FilesChecked     []string
	ContinuationID   string
	UseAssistant     bool
	ThinkingMode     types.ThinkingMode
	Temperature      float64
	Model            string
}

// ParseWorkflowState extracts workflow state from arguments
func (t *WorkflowTool) ParseWorkflowState(args map[string]any) (*WorkflowState, error) {
	parser := tools.NewArgumentParser(args)

	step, err := parser.GetStringRequired("step")
	if err != nil {
		return nil, err
	}

	findings, err := parser.GetStringRequired("findings")
	if err != nil {
		return nil, err
	}

	return &WorkflowState{
		Step:             step,
		StepNumber:       parser.GetInt("step_number", 1),
		TotalSteps:       parser.GetInt("total_steps", 1),
		NextStepRequired: parser.GetBool("next_step_required", false),
		Findings:         findings,
		Hypothesis:       parser.GetString("hypothesis"),
		Confidence:       types.ConfidenceLevel(parser.GetString("confidence")),
		RelevantFiles:    parser.GetStringArray("relevant_files"),
		FilesChecked:     parser.GetStringArray("files_checked"),
		ContinuationID:   parser.GetString("continuation_id"),
		UseAssistant:     parser.GetBool("use_assistant_model", true),
		ThinkingMode:     types.ThinkingMode(parser.GetString("thinking_mode")),
		Temperature:      parser.GetFloat("temperature", 0.3),
		Model:            parser.GetString("model"),
	}, nil
}

// GetOrCreateThread manages conversation threading
func (t *WorkflowTool) GetOrCreateThread(continuationID string) (*types.ThreadContext, bool) {
	if continuationID == "" {
		return t.memory.CreateThread(t.name), false
	}

	thread := t.memory.GetThread(continuationID)
	if thread == nil {
		return t.memory.CreateThread(t.name), false
	}

	return thread, true
}

// AddTurn adds a conversation turn with error logging
func (t *WorkflowTool) AddTurn(threadID string, turn types.ConversationTurn) {
	turn.ToolName = t.name
	if err := t.memory.AddTurn(threadID, turn); err != nil {
		slog.Warn("failed to add conversation turn",
			"threadID", threadID,
			"tool", t.name,
			"role", turn.Role,
			"error", err)
	}
}

// ConsolidateFindings merges findings from multiple steps
func (t *WorkflowTool) ConsolidateFindings(thread *types.ThreadContext, currentFindings string) string {
	var allFindings []string

	for _, turn := range thread.Turns {
		if turn.Role == "user" && strings.Contains(turn.Content, "findings:") {
			// Extract findings from previous turns
			allFindings = append(allFindings, turn.Content)
		}
	}

	if currentFindings != "" {
		allFindings = append(allFindings, currentFindings)
	}

	return strings.Join(allFindings, "\n\n---\n\n")
}

// CallExpertModel calls a high-intelligence model for final analysis
func (t *WorkflowTool) CallExpertModel(
	ctx context.Context,
	prompt string,
	systemPrompt string,
) (*types.ModelResponse, error) {
	// Select best available model
	caps, provider, err := t.registry.SelectBestModel(providers.ModelRequirements{
		MinIntelligence:     80,
		NeedsThinking:       true,
		NeedsCodeGeneration: true,
	})
	if err != nil {
		return nil, fmt.Errorf("selecting expert model: %w", err)
	}

	slog.Info("calling expert model", "model", caps.ModelName, "provider", caps.Provider)

	return provider.GenerateContent(ctx, &providers.GenerateRequest{
		Prompt:       prompt,
		SystemPrompt: systemPrompt,
		Model:        caps.ModelName,
		Temperature:  0.3,
		ThinkingMode: types.ThinkingHigh,
	})
}

// BuildGuidanceResponse creates the response for intermediate steps
func (t *WorkflowTool) BuildGuidanceResponse(state *WorkflowState, guidance string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Step %d of %d\n\n", state.StepNumber, state.TotalSteps))
	sb.WriteString(fmt.Sprintf("**Confidence:** %s\n", state.Confidence))

	if state.Hypothesis != "" {
		sb.WriteString(fmt.Sprintf("**Current Hypothesis:** %s\n", state.Hypothesis))
	}

	sb.WriteString("\n### Guidance\n\n")
	sb.WriteString(guidance)

	sb.WriteString("\n\n---\n")
	sb.WriteString(fmt.Sprintf("continuation_id: %s\n", state.ContinuationID))

	return sb.String()
}

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }
