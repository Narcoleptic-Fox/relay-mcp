# 02 - MCP Protocol & Server

## Overview

The MCP (Model Context Protocol) server communicates over stdio using JSON-RPC 2.0. It exposes tools that AI assistants can discover and invoke.

## MCP Protocol Basics

### Communication Flow

```
┌─────────────┐                      ┌─────────────┐
│   Claude    │  ─── stdin ───────>  │  MCP Server │
│   Code      │  <── stdout ───────  │  (pal-mcp)  │
└─────────────┘                      └─────────────┘
```

### Key Methods

| Method | Description |
|--------|-------------|
| `tools/list` | Return available tools and their schemas |
| `tools/call` | Execute a tool with arguments |
| `initialize` | Client capability negotiation |
| `ping` | Health check |

## Using mcp-go SDK

The [mcp-go](https://github.com/mark3labs/mcp-go) SDK handles the protocol layer.

```go
import "github.com/mark3labs/mcp-go/server"
```

## Server Implementation (internal/server/server.go)

```go
package server

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/memory"
    "github.com/yourorg/pal-mcp/internal/providers"
    "github.com/yourorg/pal-mcp/internal/tools"
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
        "pal-mcp",
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
    s.registerTool(tools.NewVersionTool(s.cfg))
    s.registerTool(tools.NewListModelsTool(s.cfg, s.registry))
    s.registerTool(tools.NewChatTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewAPILookupTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewChallengeTool(s.cfg, s.registry, s.memory))

    // CLI linking
    s.registerTool(tools.NewClinkTool(s.cfg, s.memory))

    // Workflow tools
    s.registerTool(tools.NewThinkDeepTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewDebugTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewCodeReviewTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewPrecommitTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewPlannerTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewConsensusTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewAnalyzeTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewRefactorTool(s.cfg, s.registry, s.memory))
    s.registerTool(tools.NewTestGenTool(s.cfg, s.registry, s.memory))
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
        args, ok := request.Params.Arguments.(map[string]any)
        if !ok {
            return nil, fmt.Errorf("invalid arguments type")
        }

        // Execute tool
        result, err := t.Execute(ctx, args)
        if err != nil {
            slog.Error("tool execution failed", "name", t.Name(), "error", err)
            return mcp.NewToolResultError(err.Error()), nil
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
```

## Tool Schema Generation

MCP tools need JSON Schema for their parameters. Here's a schema builder:

```go
// internal/tools/schema.go
package tools

import (
    "encoding/json"
    "reflect"
)

// SchemaBuilder generates JSON Schema for tool parameters
type SchemaBuilder struct {
    schema map[string]any
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder() *SchemaBuilder {
    return &SchemaBuilder{
        schema: map[string]any{
            "type":       "object",
            "properties": map[string]any{},
            "required":   []string{},
        },
    }
}

// AddString adds a string property
func (b *SchemaBuilder) AddString(name, description string, required bool) *SchemaBuilder {
    props := b.schema["properties"].(map[string]any)
    props[name] = map[string]any{
        "type":        "string",
        "description": description,
    }
    if required {
        b.schema["required"] = append(b.schema["required"].([]string), name)
    }
    return b
}

// AddStringEnum adds a string property with enum values
func (b *SchemaBuilder) AddStringEnum(name, description string, values []string, required bool) *SchemaBuilder {
    props := b.schema["properties"].(map[string]any)
    props[name] = map[string]any{
        "type":        "string",
        "description": description,
        "enum":        values,
    }
    if required {
        b.schema["required"] = append(b.schema["required"].([]string), name)
    }
    return b
}

// AddInteger adds an integer property
func (b *SchemaBuilder) AddInteger(name, description string, required bool, min, max *int) *SchemaBuilder {
    props := b.schema["properties"].(map[string]any)
    prop := map[string]any{
        "type":        "integer",
        "description": description,
    }
    if min != nil {
        prop["minimum"] = *min
    }
    if max != nil {
        prop["maximum"] = *max
    }
    props[name] = prop
    if required {
        b.schema["required"] = append(b.schema["required"].([]string), name)
    }
    return b
}

// AddNumber adds a number property
func (b *SchemaBuilder) AddNumber(name, description string, required bool, min, max *float64) *SchemaBuilder {
    props := b.schema["properties"].(map[string]any)
    prop := map[string]any{
        "type":        "number",
        "description": description,
    }
    if min != nil {
        prop["minimum"] = *min
    }
    if max != nil {
        prop["maximum"] = *max
    }
    props[name] = prop
    if required {
        b.schema["required"] = append(b.schema["required"].([]string), name)
    }
    return b
}

// AddBoolean adds a boolean property
func (b *SchemaBuilder) AddBoolean(name, description string, required bool) *SchemaBuilder {
    props := b.schema["properties"].(map[string]any)
    props[name] = map[string]any{
        "type":        "boolean",
        "description": description,
    }
    if required {
        b.schema["required"] = append(b.schema["required"].([]string), name)
    }
    return b
}

// AddStringArray adds a string array property
func (b *SchemaBuilder) AddStringArray(name, description string, required bool) *SchemaBuilder {
    props := b.schema["properties"].(map[string]any)
    props[name] = map[string]any{
        "type":        "array",
        "description": description,
        "items": map[string]any{
            "type": "string",
        },
    }
    if required {
        b.schema["required"] = append(b.schema["required"].([]string), name)
    }
    return b
}

// AddObject adds an object property
func (b *SchemaBuilder) AddObject(name, description string, required bool, properties map[string]any) *SchemaBuilder {
    props := b.schema["properties"].(map[string]any)
    props[name] = map[string]any{
        "type":        "object",
        "description": description,
        "properties":  properties,
    }
    if required {
        b.schema["required"] = append(b.schema["required"].([]string), name)
    }
    return b
}

// AddObjectArray adds an array of objects
func (b *SchemaBuilder) AddObjectArray(name, description string, required bool, itemProperties map[string]any) *SchemaBuilder {
    props := b.schema["properties"].(map[string]any)
    props[name] = map[string]any{
        "type":        "array",
        "description": description,
        "items": map[string]any{
            "type":       "object",
            "properties": itemProperties,
        },
    }
    if required {
        b.schema["required"] = append(b.schema["required"].([]string), name)
    }
    return b
}

// Build returns the completed schema
func (b *SchemaBuilder) Build() map[string]any {
    return b.schema
}

// BuildJSON returns the schema as JSON
func (b *SchemaBuilder) BuildJSON() ([]byte, error) {
    return json.Marshal(b.schema)
}
```

## Alternative: Direct MCP Implementation

If you prefer not to use the mcp-go SDK, here's a raw implementation:

```go
// internal/server/raw_mcp.go
package server

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "sync"
)

// JSONRPCRequest is a JSON-RPC 2.0 request
type JSONRPCRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      any             `json:"id,omitempty"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response
type JSONRPCResponse struct {
    JSONRPC string       `json:"jsonrpc"`
    ID      any          `json:"id,omitempty"`
    Result  any          `json:"result,omitempty"`
    Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError is a JSON-RPC 2.0 error
type JSONRPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}

// ToolInfo describes a tool
type ToolInfo struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    InputSchema map[string]any `json:"inputSchema"`
}

// RawMCPServer is a raw MCP implementation
type RawMCPServer struct {
    name    string
    version string
    tools   map[string]ToolInfo
    handlers map[string]func(ctx context.Context, params json.RawMessage) (any, error)
    mu      sync.RWMutex
}

// NewRawMCPServer creates a new raw MCP server
func NewRawMCPServer(name, version string) *RawMCPServer {
    return &RawMCPServer{
        name:     name,
        version:  version,
        tools:    make(map[string]ToolInfo),
        handlers: make(map[string]func(ctx context.Context, params json.RawMessage) (any, error)),
    }
}

// RegisterTool adds a tool
func (s *RawMCPServer) RegisterTool(info ToolInfo, handler func(ctx context.Context, params json.RawMessage) (any, error)) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.tools[info.Name] = info
    s.handlers[info.Name] = handler
}

// ServeStdio runs the server on stdio
func (s *RawMCPServer) ServeStdio(ctx context.Context) error {
    reader := bufio.NewReader(os.Stdin)
    writer := os.Stdout

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        line, err := reader.ReadBytes('\n')
        if err != nil {
            if err == io.EOF {
                return nil
            }
            return fmt.Errorf("reading stdin: %w", err)
        }

        response := s.handleRequest(ctx, line)
        if response != nil {
            responseBytes, err := json.Marshal(response)
            if err != nil {
                continue
            }
            writer.Write(responseBytes)
            writer.Write([]byte("\n"))
        }
    }
}

func (s *RawMCPServer) handleRequest(ctx context.Context, data []byte) *JSONRPCResponse {
    var req JSONRPCRequest
    if err := json.Unmarshal(data, &req); err != nil {
        return &JSONRPCResponse{
            JSONRPC: "2.0",
            Error: &JSONRPCError{
                Code:    -32700,
                Message: "Parse error",
            },
        }
    }

    switch req.Method {
    case "initialize":
        return s.handleInitialize(req)
    case "tools/list":
        return s.handleListTools(req)
    case "tools/call":
        return s.handleCallTool(ctx, req)
    case "ping":
        return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
    default:
        return &JSONRPCResponse{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error: &JSONRPCError{
                Code:    -32601,
                Message: "Method not found",
            },
        }
    }
}

func (s *RawMCPServer) handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
    return &JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result: map[string]any{
            "protocolVersion": "2024-11-05",
            "serverInfo": map[string]any{
                "name":    s.name,
                "version": s.version,
            },
            "capabilities": map[string]any{
                "tools": map[string]any{},
            },
        },
    }
}

func (s *RawMCPServer) handleListTools(req JSONRPCRequest) *JSONRPCResponse {
    s.mu.RLock()
    defer s.mu.RUnlock()

    toolList := make([]ToolInfo, 0, len(s.tools))
    for _, tool := range s.tools {
        toolList = append(toolList, tool)
    }

    return &JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result: map[string]any{
            "tools": toolList,
        },
    }
}

func (s *RawMCPServer) handleCallTool(ctx context.Context, req JSONRPCRequest) *JSONRPCResponse {
    var params struct {
        Name      string          `json:"name"`
        Arguments json.RawMessage `json:"arguments"`
    }

    if err := json.Unmarshal(req.Params, &params); err != nil {
        return &JSONRPCResponse{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error: &JSONRPCError{
                Code:    -32602,
                Message: "Invalid params",
            },
        }
    }

    s.mu.RLock()
    handler, ok := s.handlers[params.Name]
    s.mu.RUnlock()

    if !ok {
        return &JSONRPCResponse{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error: &JSONRPCError{
                Code:    -32602,
                Message: fmt.Sprintf("Unknown tool: %s", params.Name),
            },
        }
    }

    result, err := handler(ctx, params.Arguments)
    if err != nil {
        return &JSONRPCResponse{
            JSONRPC: "2.0",
            ID:      req.ID,
            Result: map[string]any{
                "content": []map[string]any{
                    {
                        "type": "text",
                        "text": fmt.Sprintf("Error: %s", err.Error()),
                    },
                },
                "isError": true,
            },
        }
    }

    return &JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result: map[string]any{
            "content": []map[string]any{
                {
                    "type": "text",
                    "text": result,
                },
            },
        },
    }
}
```

## Error Handling (internal/server/errors.go)

```go
package server

import "fmt"

// Error codes
const (
    ErrCodeParse          = -32700
    ErrCodeInvalidRequest = -32600
    ErrCodeMethodNotFound = -32601
    ErrCodeInvalidParams  = -32602
    ErrCodeInternal       = -32603

    // Custom error codes
    ErrCodeToolNotFound    = -32001
    ErrCodeToolDisabled    = -32002
    ErrCodeProviderError   = -32003
    ErrCodeInvalidArgument = -32004
)

// MCPError is a custom error type
type MCPError struct {
    Code    int
    Message string
    Data    any
}

func (e *MCPError) Error() string {
    return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Error constructors
func ErrToolNotFound(name string) *MCPError {
    return &MCPError{
        Code:    ErrCodeToolNotFound,
        Message: fmt.Sprintf("tool not found: %s", name),
    }
}

func ErrToolDisabled(name string) *MCPError {
    return &MCPError{
        Code:    ErrCodeToolDisabled,
        Message: fmt.Sprintf("tool is disabled: %s", name),
    }
}

func ErrProviderError(provider, message string) *MCPError {
    return &MCPError{
        Code:    ErrCodeProviderError,
        Message: fmt.Sprintf("provider %s error: %s", provider, message),
    }
}

func ErrInvalidArgument(name, reason string) *MCPError {
    return &MCPError{
        Code:    ErrCodeInvalidArgument,
        Message: fmt.Sprintf("invalid argument %s: %s", name, reason),
    }
}
```

## Request/Response Logging

```go
// internal/server/logging.go
package server

import (
    "context"
    "log/slog"
    "time"

    "github.com/google/uuid"
)

// RequestLogger wraps tool execution with logging
type RequestLogger struct {
    logger *slog.Logger
}

// NewRequestLogger creates a request logger
func NewRequestLogger() *RequestLogger {
    return &RequestLogger{
        logger: slog.Default(),
    }
}

// LogToolCall logs a tool execution
func (l *RequestLogger) LogToolCall(
    ctx context.Context,
    toolName string,
    args map[string]any,
    fn func() (string, error),
) (string, error) {
    requestID := uuid.New().String()[:8]
    start := time.Now()

    l.logger.Info("TOOL_CALL",
        "request_id", requestID,
        "tool", toolName,
        "args", args,
    )

    result, err := fn()
    duration := time.Since(start)

    if err != nil {
        l.logger.Error("TOOL_ERROR",
            "request_id", requestID,
            "tool", toolName,
            "duration_ms", duration.Milliseconds(),
            "error", err,
        )
    } else {
        l.logger.Info("TOOL_COMPLETED",
            "request_id", requestID,
            "tool", toolName,
            "duration_ms", duration.Milliseconds(),
            "result_length", len(result),
        )
    }

    return result, err
}
```

## Integration with Claude Code

### MCP Configuration

Claude Code uses a JSON configuration file to discover MCP servers:

```json
// ~/.config/claude/mcp_servers.json (Linux/macOS)
// %APPDATA%\Claude\mcp_servers.json (Windows)
{
    "pal": {
        "command": "/path/to/pal-mcp",
        "args": [],
        "env": {
            "GEMINI_API_KEY": "your-key",
            "OPENAI_API_KEY": "your-key"
        }
    }
}
```

Or use command-line:

```bash
claude --mcp-server /path/to/pal-mcp
```

## Testing the Server

### Manual Testing

```bash
# Build
go build -o pal-mcp ./cmd/pal-mcp

# Test with echo
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./pal-mcp
```

### Unit Testing

```go
// internal/server/server_test.go
package server

import (
    "context"
    "encoding/json"
    "testing"

    "github.com/yourorg/pal-mcp/internal/config"
    "github.com/yourorg/pal-mcp/internal/providers"
)

func TestServerListTools(t *testing.T) {
    cfg := &config.Config{Version: "test"}
    registry := providers.NewRegistry(cfg)
    srv := New(cfg, registry)

    // Test that tools are registered
    if len(srv.tools) == 0 {
        t.Error("no tools registered")
    }

    // Check for required tools
    requiredTools := []string{"version", "listmodels", "chat"}
    for _, name := range requiredTools {
        if _, ok := srv.tools[name]; !ok {
            t.Errorf("required tool %s not registered", name)
        }
    }
}

func TestServerToolExecution(t *testing.T) {
    cfg := &config.Config{Version: "1.0.0"}
    registry := providers.NewRegistry(cfg)
    srv := New(cfg, registry)

    // Execute version tool
    tool := srv.tools["version"]
    result, err := tool.Execute(context.Background(), map[string]any{})
    if err != nil {
        t.Fatalf("version tool failed: %v", err)
    }

    if result.Content == "" {
        t.Error("version tool returned empty content")
    }
}
```

## Next Steps

Continue to [03-PROVIDERS.md](./03-PROVIDERS.md) for the provider system implementation.
