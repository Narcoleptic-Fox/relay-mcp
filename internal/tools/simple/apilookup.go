package simple

import (
	"context"
	"fmt"
	    "strings"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/memory"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/providers"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools"
	)
	// APILookupTool helps users find documentation and API details
type APILookupTool struct {
	*BaseTool
}

// NewAPILookupTool creates a new API lookup tool
func NewAPILookupTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *APILookupTool {
	tool := &APILookupTool{
		BaseTool: NewBaseTool("apilookup", "Find documentation and API details for libraries and frameworks.", cfg, registry, mem),
	}

	// Define schema
	tool.schema.
		AddString("query", "The library, function, or concept to look up (e.g., 'React useEffect', 'Python requests')", true).
		AddString("context", "Additional context about what you're trying to achieve", false).
		AddString("model", "Model to use", false).
		AddString("continuation_id", "Thread ID", false)

	return tool
}

func (t *APILookupTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	parser := tools.NewArgumentParser(args)

	query, err := parser.GetStringRequired("query")
	if err != nil {
		return nil, err
	}
	userContext := parser.GetString("context")
	modelName := parser.GetString("model")
	continuationID := parser.GetString("continuation_id")

	// Get thread
	thread, _ := t.GetOrCreateThread(continuationID)

	// Resolve model
	resolvedModel, provider, err := t.ResolveModel(modelName)
	if err != nil {
		return nil, fmt.Errorf("resolving model: %w", err)
	}

	// Build prompt
	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf("Please provide documentation and usage examples for: %s\n\n", query))
	if userContext != "" {
		promptBuilder.WriteString(fmt.Sprintf("Context: %s\n\n", userContext))
	}
	promptBuilder.WriteString("Include:\n1. Brief explanation\n2. Function signature/syntax\n3. Common usage examples\n4. Best practices/gotchas")

	// Generate response
	resp, err := t.GenerateContent(ctx, provider, &providers.GenerateRequest{
		Prompt:              promptBuilder.String(),
		SystemPrompt:        "You are a technical documentation expert. Provide clear, accurate, and concise API documentation with code examples.",
		Model:               resolvedModel,
		ConversationHistory: t.memory.GetHistory(thread.ThreadID),
	})
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	// Save to memory
	t.AddTurn(thread.ThreadID, "user", fmt.Sprintf("Lookup: %s", query), nil, nil)
	t.AddTurn(thread.ThreadID, "assistant", resp.Content, nil, nil)

	result := fmt.Sprintf("%s\n\n---\ncontinuation_id: %s", resp.Content, thread.ThreadID)
	return tools.NewToolResult(result), nil
}
