package simple

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
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/utils"
	)
	// ChatTool handles multi-turn conversations
type ChatTool struct {
	*BaseTool
}

// NewChatTool creates a new chat tool
func NewChatTool(cfg *config.Config, registry *providers.Registry, mem *memory.ConversationMemory) *ChatTool {
	tool := &ChatTool{
		BaseTool: NewBaseTool("chat", "General chat and collaborative thinking partner", cfg, registry, mem),
	}

	// Define schema
	tool.schema.
		AddString("prompt", "Your question or idea for collaborative thinking", true).
		AddString("working_directory_absolute_path", "Absolute path to working directory", true).
		AddString("model", "Model to use (or 'auto' for automatic selection)", false).
		AddStringArray("absolute_file_paths", "Full paths to relevant code files", false).
		AddStringArray("images", "Image paths or base64 strings", false).
		AddString("continuation_id", "Thread ID for multi-turn conversations", false).
		AddNumber("temperature", "0 = deterministic, 1 = creative", false, ptr(0.0), ptr(1.0)).
		AddStringEnum("thinking_mode", "Reasoning depth", []string{"minimal", "low", "medium", "high", "max"}, false)

	return tool
}

func (t *ChatTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	parser := tools.NewArgumentParser(args)

	prompt, err := parser.GetStringRequired("prompt")
	if err != nil {
		return nil, err
	}

	workDir, err := parser.GetStringRequired("working_directory_absolute_path")
	if err != nil {
		return nil, err
	}

	modelName := parser.GetString("model")
	filePaths := parser.GetStringArray("absolute_file_paths")
	images := parser.GetStringArray("images")
	continuationID := parser.GetString("continuation_id")
	temperature := parser.GetFloat("temperature", 0.7)
	thinkingMode := types.ThinkingMode(parser.GetString("thinking_mode"))

	// Get or create conversation thread
	thread, isExisting := t.GetOrCreateThread(continuationID)
	slog.Debug("chat thread", "id", thread.ThreadID, "existing", isExisting)

	// Resolve model
	resolvedModel, provider, err := t.ResolveModel(modelName)
	if err != nil {
		return nil, fmt.Errorf("resolving model: %w", err)
	}

	// Read files
	fileContents, err := utils.ReadFiles(filePaths, workDir)
	if err != nil {
		slog.Warn("error reading files", "error", err)
	}

	// Build prompt with file contents
	fullPrompt := t.buildPrompt(prompt, fileContents)

	// Get conversation history
	history := t.memory.GetHistory(thread.ThreadID)

	// Generate response
	resp, err := t.GenerateContent(ctx, provider, &providers.GenerateRequest{
		Prompt:              fullPrompt,
		SystemPrompt:        t.getSystemPrompt(),
		Model:               resolvedModel,
		Temperature:         temperature,
		ThinkingMode:        thinkingMode,
		ConversationHistory: history,
		Images:              images,
	})
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	// Save turns
	t.AddTurn(thread.ThreadID, "user", prompt, filePaths, images)
	t.AddTurn(thread.ThreadID, "assistant", resp.Content, nil, nil)

	// Build response with continuation ID
	result := fmt.Sprintf("%s\n\n---\ncontinuation_id: %s", resp.Content, thread.ThreadID)

	return tools.NewToolResult(result), nil
}

func (t *ChatTool) buildPrompt(prompt string, files []utils.FileContent) string {
	if len(files) == 0 {
		return prompt
	}

	var sb strings.Builder
	sb.WriteString(prompt)
	sb.WriteString("\n\n## Referenced Files\n\n")

	for _, f := range files {
		sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", f.Path, f.Content))
	}

	return sb.String()
}

func (t *ChatTool) getSystemPrompt() string {
	// Load from prompts/chat.txt or use default
	return `You are a helpful AI assistant for software development.
You help with coding questions, debugging, architecture decisions, and general development tasks.
Be concise but thorough. When discussing code, reference specific files and line numbers.
If you're unsure about something, say so rather than making assumptions.`
}

func ptr[T any](v T) *T {
	return &v
}
