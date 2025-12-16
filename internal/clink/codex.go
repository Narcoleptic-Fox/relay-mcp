package clink

import (
	"context"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/config"
)

// CodexAgent is a CLI agent for Codex CLI (or generic OpenAI CLI)
// This is a placeholder/generic implementation similar to GeminiAgent but for Codex
type CodexAgent struct {
	*BaseAgent
}

// NewCodexAgent creates a new Codex CLI agent
func NewCodexAgent(clientCfg config.CLIClientConfig, cfg *config.Config) *CodexAgent {
	return &CodexAgent{
		BaseAgent: NewBaseAgent(clientCfg, cfg),
	}
}

// Run executes Codex CLI and returns output
func (a *CodexAgent) Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error) {
	// For now, Codex output parsing is basic (raw output)
	return a.BaseAgent.Run(ctx, req)
}
