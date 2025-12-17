package server

import (
	"context"
	"log/slog"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/config"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/memory"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/providers"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools/simple"
    "github.com/Narcoleptic-Fox/relay-mcp/internal/tools/workflow"
)

// Server is the MCP server
type Server struct {
    cfg      *config.Config
    registry *providers.Registry
    memory   *memory.ConversationMemory
    tools    map[string]tools.Tool
    mcp      *server.MCPServer
}

// New creates a new MCP server
func New(cfg *config.Config, registry *providers.Registry) *Server {
    s := &Server{
        cfg:      cfg,
        registry: registry,
        memory:   memory.New(cfg.MaxConversationTurns, cfg.ConversationTimeoutHours),
        tools:    make(map[string]tools.Tool),
    }

    // Create MCP server
    s.mcp = server.NewMCPServer(
        "relay-mcp",
        cfg.Version,
        server.WithToolCapabilities(true),
    )

    // Register tools
    s.registerTools()

    return s
}

// registerTools registers all available tools
func (s *Server) registerTools() {
	// Simple tools
	s.registerTool(simple.NewVersionTool(s.cfg))
	s.registerTool(simple.NewListModelsTool(s.cfg, s.registry))
	s.registerTool(simple.NewChatTool(s.cfg, s.registry, s.memory))
	s.registerTool(simple.NewAPILookupTool(s.cfg, s.registry, s.memory))
	s.registerTool(simple.NewChallengeTool(s.cfg, s.registry, s.memory))

	// CLI linking
	s.registerTool(simple.NewClinkTool(s.cfg, s.memory))

	// Workflow tools
	s.registerTool(workflow.NewThinkDeepTool(s.cfg, s.registry, s.memory))
	s.registerTool(workflow.NewDebugTool(s.cfg, s.registry, s.memory))
	s.registerTool(workflow.NewCodeReviewTool(s.cfg, s.registry, s.memory))
	s.registerTool(workflow.NewPrecommitTool(s.cfg, s.registry, s.memory))
	s.registerTool(workflow.NewPlannerTool(s.cfg, s.registry, s.memory))
	s.registerTool(workflow.NewConsensusTool(s.cfg, s.registry, s.memory))
	s.registerTool(workflow.NewAnalyzeTool(s.cfg, s.registry, s.memory))
	s.registerTool(workflow.NewRefactorTool(s.cfg, s.registry, s.memory))
	s.registerTool(workflow.NewTestGenTool(s.cfg, s.registry, s.memory))
}

// registerTool adds a tool to the server
func (s *Server) registerTool(t tools.Tool) {
	name := t.Name()

	// Check if disabled
	if s.cfg.IsToolDisabled(name) {
		slog.Info("tool disabled", "name", name)
		return
	}

	s.tools[name] = t

	// Register with MCP server
	s.mcp.AddTool(
		mcp.NewTool(name,
			mcp.WithDescription(t.Description()),
			mcp.WithString("arguments",
				mcp.Description("Tool arguments as JSON"),
			),
		),
		s.handleToolCall(t),
	)

	slog.Debug("registered tool", "name", name)
}

// handleToolCall creates a handler for a specific tool
func (s *Server) handleToolCall(t tools.Tool) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Info("tool call", "name", t.Name(), "arguments", request.Params.Arguments)

		// Parse arguments
		// request.Params.Arguments is likely map[string]interface{}
		args := request.Params.Arguments

		// Execute tool
		result, err := t.Execute(ctx, args)
		if err != nil {
			slog.Error("tool execution failed", "name", t.Name(), "error", err)
			res := mcp.NewToolResultText(err.Error())
			res.IsError = true
			return res, nil
		}

		// Return result
		return mcp.NewToolResultText(result.Content), nil
	}
}

// Run starts the MCP server on stdio
func (s *Server) Run(ctx context.Context) error {
	// Start conversation memory cleanup goroutine
	go s.memory.StartCleanup(ctx)

	// Run MCP server on stdio
	return server.ServeStdio(s.mcp)
}
