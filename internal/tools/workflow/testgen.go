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
	// TestGenTool generates and validates tests
type TestGenTool struct {
	*WorkflowTool
}

// NewTestGenTool creates a new testgen tool
func NewTestGenTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *TestGenTool {
	tool := &TestGenTool{
		WorkflowTool: NewWorkflowTool(
			"testgen",
			"Generates and validates comprehensive test suites.",
			cfg, registry, mem,
		),
	}

	tool.schema.
		AddString("file_to_test", "Path to file needing tests", true).
		AddStringEnum("test_framework", "Testing framework", []string{"go test", "pytest", "jest", "junit"}, false)

	return tool
}

func (t *TestGenTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	state, err := t.ParseWorkflowState(args)
	if err != nil {
		return nil, err
	}

	// Get thread
	thread, _ := t.GetOrCreateThread(state.ContinuationID)
	state.ContinuationID = thread.ThreadID

	    _ = t.memory.AddTurn(thread.ThreadID, types.ConversationTurn{
	        Role:     "user",
	        Content:  fmt.Sprintf("TestGen Step %d: %s", state.StepNumber, state.Step),
	        ToolName: t.name,
	    })
		if state.UseAssistant {
		resp, err := t.CallExpertModel(ctx, fmt.Sprintf("Generate tests: %s", state.Findings), "You are a QA automation expert.")
		if err != nil {
			return nil, err
		}
		return tools.NewToolResult(resp.Content), nil
	}

	return tools.NewToolResult("Test generation step recorded."), nil
}
