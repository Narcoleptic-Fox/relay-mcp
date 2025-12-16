package clink

import (
	"context"
	"time"
)

// Agent is the interface for CLI agents
type Agent interface {
	// Name returns the agent name
	Name() string

	// Run executes the agent with the given request
	Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error)

	// IsAvailable checks if the CLI is installed and accessible
	IsAvailable() bool
}

// AgentRequest contains the input for an agent
type AgentRequest struct {
	Role         string            // Role preset (default, planner, codereviewer)
	Prompt       string            // The prompt to send
	SystemPrompt string            // System prompt for the role
	Files        []string          // File paths to include
	Images       []string          // Image paths to include
	WorkDir      string            // Working directory
	Timeout      time.Duration     // Execution timeout
	Env          map[string]string // Environment variables
}

// AgentOutput contains the result from an agent
type AgentOutput struct {
	Content      string        // The agent's response
	ExitCode     int           // Process exit code
	Duration     time.Duration // Execution time
	TokensUsed   int           // Estimated tokens (if available)
	ErrorMessage string        // Error message if failed
}
