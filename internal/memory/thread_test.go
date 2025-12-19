package memory

import (
	"strings"
	"testing"
	"time"

	"github.com/Narcoleptic-Fox/relay-mcp/internal/types"
)

func TestThreadBuilder_TurnCountFormatting(t *testing.T) {
	// Create a thread with 15 turns
	thread := &types.ThreadContext{
		ThreadID:  "test-123",
		CreatedAt: time.Now(),
		ToolName:  "chat",
		Turns:     make([]types.ConversationTurn, 15),
	}

	builder := NewThreadBuilder(thread, 10000)
	summary := builder.BuildContextSummary()

	// Verify the output contains the correct number as a string
	if !strings.Contains(summary, "**Turns:** 15") {
		t.Errorf("expected '**Turns:** 15' in summary, got: %s", summary)
	}

	// Verify it doesn't contain unexpected control characters
	for _, r := range summary {
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			t.Errorf("unexpected control character in summary: %d", r)
		}
	}
}

func TestThreadBuilder_TurnCountZero(t *testing.T) {
	thread := &types.ThreadContext{
		ThreadID:  "test-456",
		CreatedAt: time.Now(),
		ToolName:  "analyze",
		Turns:     []types.ConversationTurn{},
	}

	builder := NewThreadBuilder(thread, 10000)
	summary := builder.BuildContextSummary()

	if !strings.Contains(summary, "**Turns:** 0") {
		t.Errorf("expected '**Turns:** 0' in summary, got: %s", summary)
	}
}

func TestThreadBuilder_TurnCountLarge(t *testing.T) {
	// Test with a large number to ensure no overflow/weird behavior
	turns := make([]types.ConversationTurn, 1000)
	thread := &types.ThreadContext{
		ThreadID:  "test-789",
		CreatedAt: time.Now(),
		ToolName:  "consensus",
		Turns:     turns,
	}

	builder := NewThreadBuilder(thread, 10000)
	summary := builder.BuildContextSummary()

	if !strings.Contains(summary, "**Turns:** 1000") {
		t.Errorf("expected '**Turns:** 1000' in summary, got: %s", summary)
	}
}

func TestThreadBuilder_BuildConversationHistory(t *testing.T) {
	thread := &types.ThreadContext{
		ThreadID:  "test-history",
		CreatedAt: time.Now(),
		ToolName:  "chat",
		Turns: []types.ConversationTurn{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "How are you?"},
		},
	}

	builder := NewThreadBuilder(thread, 10000)
	history := builder.BuildConversationHistory()

	if len(history) != 3 {
		t.Errorf("expected 3 turns, got %d", len(history))
	}

	if history[0].Content != "Hello" {
		t.Errorf("expected first turn to be 'Hello', got '%s'", history[0].Content)
	}
}
