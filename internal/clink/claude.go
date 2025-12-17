package clink

import (
	"context"
	    "strings"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	)
	// ClaudeAgent is a CLI agent for Claude Code
type ClaudeAgent struct {
	*BaseAgent
	parser *ClaudeParser
}

// NewClaudeAgent creates a new Claude CLI agent
func NewClaudeAgent(clientCfg config.CLIClientConfig, cfg *config.Config) *ClaudeAgent {
	return &ClaudeAgent{
		BaseAgent: NewBaseAgent(clientCfg, cfg),
		parser:    &ClaudeParser{},
	}
}

// Run executes Claude CLI and parses the output
func (a *ClaudeAgent) Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error) {
	output, err := a.BaseAgent.Run(ctx, req)
	if err != nil {
		return output, err
	}

	// Parse Claude-specific output
	parsed := a.parser.Parse(output.Content)
	output.Content = parsed

	return output, nil
}

// ClaudeParser parses Claude CLI output
type ClaudeParser struct{}

// Parse extracts content from Claude CLI output
func (p *ClaudeParser) Parse(raw string) string {
	// Claude Code outputs markdown-formatted responses
	// Strip any ANSI codes and clean up
	raw = stripANSI(raw)
	return strings.TrimSpace(raw)
}

func stripANSI(s string) string {
	// Simple ANSI escape code removal
	result := strings.Builder{}
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}
