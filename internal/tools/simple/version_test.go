package simple

import (
	"context"
	"testing"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/config"
)

func TestVersionTool_Execute(t *testing.T) {
	cfg := &config.Config{
		Version:   "1.0.0",
		Commit:    "abc1234",
		BuildTime: "2024-01-01",
	}

	tool := NewVersionTool(cfg)

	if tool.Name() != "version" {
		t.Errorf("expected name 'version', got %s", tool.Name())
	}

	result, err := tool.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Error("expected success result")
	}

	expected := "ZEN MCP Server"
	if len(result.Content) < len(expected) || result.Content[:len(expected)] != expected {
		t.Errorf("expected content starting with %q, got %q", expected, result.Content)
	}
}
