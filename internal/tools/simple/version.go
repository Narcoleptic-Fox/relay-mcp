package simple

import (
	    "context"
	    "fmt"
	
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools"
	)
	
	// VersionTool returns server version information
	type VersionTool struct {
	    cfg *config.Config
	}
	
	// NewVersionTool creates a new version tool
	func NewVersionTool(cfg *config.Config) *VersionTool {
	    return &VersionTool{cfg: cfg}
	}
	
	func (t *VersionTool) Name() string {
	    return "version"
	}
	
	func (t *VersionTool) Description() string {
	    return "Get server version, configuration details, and list of available tools."
	}
	
	func (t *VersionTool) Schema() map[string]any {
	    return tools.NewSchemaBuilder().Build()
	}
	
	func (t *VersionTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	    content := fmt.Sprintf(`RELAY MCP Server
	Version: %s
	Commit: %s
	Build Time: %s
	
	This server provides AI-powered development tools including:
	
- chat: Multi-turn conversations with AI models
- thinkdeep: Extended reasoning and analysis
- debug: Root cause analysis for bugs
- codereview: Systematic code review
- consensus: Multi-model debate and synthesis
- clink: Bridge to external AI CLIs
- And more...

Use 'listmodels' to see available AI models.`,
		t.cfg.Version,
		t.cfg.Commit,
		t.cfg.BuildTime,
	)

	return tools.NewToolResult(content), nil
}
