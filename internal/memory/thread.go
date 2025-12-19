package memory

import (
	"strconv"
	"strings"
	"time"

	"github.com/Narcoleptic-Fox/relay-mcp/internal/types"
)
	// ThreadBuilder helps build thread context for AI prompts
type ThreadBuilder struct {
	thread     *types.ThreadContext
	maxTokens  int
	tokenCount int
}

// NewThreadBuilder creates a new thread builder
func NewThreadBuilder(thread *types.ThreadContext, maxTokens int) *ThreadBuilder {
	return &ThreadBuilder{
		thread:    thread,
		maxTokens: maxTokens,
	}
}

// BuildConversationHistory builds the conversation history for the AI
func (b *ThreadBuilder) BuildConversationHistory() []types.ConversationTurn {
	if len(b.thread.Turns) == 0 {
		return nil
	}

	// Strategy: Include newest turns first, up to token budget
	var selected []types.ConversationTurn

	for i := len(b.thread.Turns) - 1; i >= 0; i-- {
		turn := b.thread.Turns[i]
		turnTokens := b.estimateTokens(turn.Content)

		if b.tokenCount+turnTokens > b.maxTokens {
			break
		}

		b.tokenCount += turnTokens
		// Prepend to maintain chronological order
		selected = append([]types.ConversationTurn{turn}, selected...)
	}

	return selected
}

// BuildContextSummary creates a summary of the conversation
func (b *ThreadBuilder) BuildContextSummary() string {
	var sb strings.Builder

	sb.WriteString("## Conversation Context\n\n")
	sb.WriteString("**Thread ID:** " + b.thread.ThreadID + "\n")
	sb.WriteString("**Started:** " + b.thread.CreatedAt.Format(time.RFC3339) + "\n")
	sb.WriteString("**Tool:** " + b.thread.ToolName + "\n")
	sb.WriteString("**Turns:** " + strconv.Itoa(len(b.thread.Turns)) + "\n\n")

	// Collect unique tools used
	tools := make(map[string]bool)
	for _, turn := range b.thread.Turns {
		if turn.ToolName != "" {
			tools[turn.ToolName] = true
		}
	}

	if len(tools) > 1 {
		sb.WriteString("**Tools Used:** ")
		first := true
		for tool := range tools {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(tool)
			first = false
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetRecentFiles returns files from recent turns
func (b *ThreadBuilder) GetRecentFiles(maxFiles int) []string {
	seen := make(map[string]bool)
	var files []string

	for i := len(b.thread.Turns) - 1; i >= 0 && len(files) < maxFiles; i-- {
		for _, f := range b.thread.Turns[i].Files {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}

	return files
}

func (b *ThreadBuilder) estimateTokens(content string) int {
	// Rough estimate: 4 characters per token
	return len(content) / 4
}
