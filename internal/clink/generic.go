package clink

import (
	"context"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/config"
)

// GenericAgent is a generic CLI agent for custom CLIs
type GenericAgent struct {
	*BaseAgent
}

// NewGenericAgent creates a new generic CLI agent
func NewGenericAgent(clientCfg config.CLIClientConfig, cfg *config.Config) *GenericAgent {
	return &GenericAgent{
		BaseAgent: NewBaseAgent(clientCfg, cfg),
	}
}

// Run executes the CLI (uses base implementation)
func (a *GenericAgent) Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error) {
	return a.BaseAgent.Run(ctx, req)
}
