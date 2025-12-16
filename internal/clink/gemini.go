package clink

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Narcoleptic-Fox/zen-mcp/internal/config"
)

// GeminiAgent is a CLI agent for Gemini CLI
type GeminiAgent struct {
	*BaseAgent
	parser *GeminiParser
}

// NewGeminiAgent creates a new Gemini CLI agent
func NewGeminiAgent(clientCfg config.CLIClientConfig, cfg *config.Config) *GeminiAgent {
	return &GeminiAgent{
		BaseAgent: NewBaseAgent(clientCfg, cfg),
		parser:    &GeminiParser{},
	}
}

// Run executes the Gemini CLI and parses the output
func (a *GeminiAgent) Run(ctx context.Context, req *AgentRequest) (*AgentOutput, error) {
	output, err := a.BaseAgent.Run(ctx, req)
	if err != nil {
		return output, err
	}

	// Parse Gemini-specific output format
	parsed := a.parser.Parse(output.Content)
	output.Content = parsed

	return output, nil
}

// GeminiParser parses Gemini CLI output
type GeminiParser struct{}

// Parse extracts content from Gemini CLI output
func (p *GeminiParser) Parse(raw string) string {
	// Gemini CLI may output JSON or plain text
	// Try to parse as JSON first
	raw = strings.TrimSpace(raw)

	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
		// Try JSON parsing
		var result struct {
			Response string `json:"response"`
			Content  string `json:"content"`
			Text     string `json:"text"`
		}

		if err := json.Unmarshal([]byte(raw), &result); err == nil {
			if result.Response != "" {
				return result.Response
			}
			if result.Content != "" {
				return result.Content
			}
			if result.Text != "" {
				return result.Text
			}
		}

		// Try as array
		var responses []struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(raw), &responses); err == nil && len(responses) > 0 {
			var parts []string
			for _, r := range responses {
				if r.Content != "" {
					parts = append(parts, r.Content)
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n")
			}
		}
	}

	// Return raw output if no special parsing needed
	return raw
}
