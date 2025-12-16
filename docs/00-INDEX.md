# PAL MCP Server - Go Rewrite Architecture

## Overview

This document set provides a complete architecture plan for rewriting the PAL MCP Server from Python to Go. The goal is full feature parity with improved performance, single-binary deployment, and better resource efficiency.

## Document Index

| # | Document | Description |
|---|----------|-------------|
| 01 | [Project Structure & Setup](./01-PROJECT-SETUP.md) | Go project layout, dependencies, build system |
| 02 | [MCP Protocol & Server](./02-MCP-SERVER.md) | MCP protocol handling, stdio JSON-RPC |
| 03 | [Provider System](./03-PROVIDERS.md) | AI provider abstraction, registry, implementations |
| 04 | [Tool System](./04-TOOLS.md) | Tool interface, simple tools, workflow tools |
| 05 | [Conversation Memory](./05-CONVERSATION-MEMORY.md) | Thread storage, cross-tool continuation |
| 06 | [CLI Linking (Clink)](./06-CLINK.md) | External CLI spawning, parsers |
| 07 | [Consensus & Workflows](./07-CONSENSUS-WORKFLOWS.md) | Multi-model orchestration |
| 08 | [Configuration & Deployment](./08-CONFIG-DEPLOY.md) | Config files, env vars, building, releasing |

## Why Go?

### Advantages Over Python

| Aspect | Python (Current) | Go (Target) |
|--------|------------------|-------------|
| **Startup time** | ~500ms (interpreter + imports) | ~5ms |
| **Memory usage** | ~100MB baseline | ~10MB baseline |
| **Deployment** | venv + requirements.txt + Python runtime | Single binary |
| **Concurrency** | asyncio (complex) | Goroutines (simple) |
| **Type safety** | Runtime (Pydantic) | Compile-time |
| **Subprocess handling** | Good | Excellent |

### Key Benefits for PAL

1. **Single Binary Distribution** - Users just download and run
2. **Fast Cold Start** - Critical for MCP servers that may restart
3. **Native Concurrency** - Goroutines perfect for managing multiple AI calls
4. **Excellent Subprocess Control** - First-class support for CLI spawning
5. **Cross-Platform** - Easy compilation for Windows, macOS, Linux

## Feature Parity Checklist

### Core Features

- [ ] MCP Protocol (stdio JSON-RPC)
- [ ] Tool registration and execution
- [ ] Conversation memory with threading
- [ ] Cross-tool continuation

### Providers (7 total)

- [ ] Gemini (Google AI)
- [ ] OpenAI
- [ ] Azure OpenAI
- [ ] X.AI (Grok)
- [ ] DIAL
- [ ] OpenRouter
- [ ] Custom/Local (Ollama, vLLM)

### Simple Tools (6 total)

- [ ] `chat` - Multi-turn conversations
- [ ] `apilookup` - Documentation lookup
- [ ] `challenge` - Critical analysis
- [ ] `listmodels` - Model enumeration
- [ ] `version` - Server info
- [ ] `clink` - CLI bridging

### Workflow Tools (9 total)

- [ ] `thinkdeep` - Extended reasoning
- [ ] `codereview` - Code review
- [ ] `precommit` - Pre-commit validation
- [ ] `debug` - Root cause analysis
- [ ] `planner` - Sequential planning
- [ ] `consensus` - Multi-model debate
- [ ] `analyze` - Codebase analysis
- [ ] `refactor` - Refactoring analysis
- [ ] `testgen` - Test generation

### CLI Linking

- [ ] Gemini CLI agent
- [ ] Claude Code agent
- [ ] Codex CLI agent
- [ ] Custom CLI support

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        MCP Client (Claude Code)                  │
└─────────────────────────────────────────────────────────────────┘
                                   │
                                   │ stdio (JSON-RPC)
                                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                         MCP Server                               │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Request Router                            ││
│  │         (list_tools, call_tool, handle errors)              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                │                                 │
│         ┌──────────────────────┼──────────────────────┐         │
│         ▼                      ▼                      ▼         │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐     │
│  │ Simple Tools│      │Workflow Tools│      │ Clink Tool  │     │
│  │ chat,lookup │      │debug,review │      │ CLI bridge  │     │
│  └─────────────┘      └─────────────┘      └─────────────┘     │
│         │                      │                      │         │
│         └──────────────────────┼──────────────────────┘         │
│                                ▼                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                  Provider Registry                           ││
│  │     Gemini | OpenAI | Azure | XAI | DIAL | OpenRouter       ││
│  └─────────────────────────────────────────────────────────────┘│
│                                │                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                 Conversation Memory                          ││
│  │              Thread Storage + TTL + Cleanup                  ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    ▼              ▼              ▼
              ┌──────────┐  ┌──────────┐  ┌──────────┐
              │Gemini API│  │OpenAI API│  │Local LLM │
              └──────────┘  └──────────┘  └──────────┘
```

## Implementation Order

### Phase 1: Foundation (Week 1-2)
1. Project setup with Go modules
2. MCP protocol handler
3. Basic tool interface
4. Simple `version` and `listmodels` tools

### Phase 2: Providers (Week 2-3)
1. Provider interface
2. Gemini provider (primary)
3. OpenAI provider
4. OpenRouter provider (catch-all)

### Phase 3: Core Tools (Week 3-4)
1. `chat` tool with conversation memory
2. Conversation threading
3. Cross-tool continuation

### Phase 4: CLI Linking (Week 4-5)
1. CLI agent interface
2. Subprocess management
3. Gemini CLI agent
4. Response parsers

### Phase 5: Workflows (Week 5-7)
1. Workflow tool base
2. `thinkdeep`, `debug`, `codereview`
3. `consensus` multi-model orchestration
4. Remaining workflow tools

### Phase 6: Polish (Week 7-8)
1. Remaining providers
2. Configuration system
3. Error handling
4. Testing
5. Documentation

## Quick Start (After Implementation)

```bash
# Build
go build -o pal-mcp ./cmd/pal-mcp

# Configure
export GEMINI_API_KEY="your-key"
export OPENAI_API_KEY="your-key"

# Run
./pal-mcp

# Or with Claude Code
claude --mcp-server ./pal-mcp
```

## Key Dependencies

```go
// go.mod
module github.com/yourorg/pal-mcp

go 1.22

require (
    github.com/mark3labs/mcp-go v0.17.0  // MCP SDK
    github.com/google/uuid v1.6.0        // UUID generation
    github.com/joho/godotenv v1.5.1      // .env loading
)
```

## Repository Structure Preview

```
pal-mcp/
├── cmd/
│   └── pal-mcp/
│       └── main.go              # Entry point
├── internal/
│   ├── server/
│   │   └── server.go            # MCP server
│   ├── providers/
│   │   ├── provider.go          # Interface
│   │   ├── registry.go          # Registry
│   │   ├── gemini.go
│   │   ├── openai.go
│   │   └── ...
│   ├── tools/
│   │   ├── tool.go              # Interface
│   │   ├── simple/
│   │   │   ├── chat.go
│   │   │   └── ...
│   │   └── workflow/
│   │       ├── base.go
│   │       ├── debug.go
│   │       └── ...
│   ├── clink/
│   │   ├── agent.go             # Interface
│   │   ├── gemini.go
│   │   └── parsers/
│   ├── memory/
│   │   └── conversation.go
│   └── config/
│       └── config.go
├── configs/
│   ├── models/
│   │   ├── gemini.json
│   │   └── openai.json
│   └── cli_clients/
│       └── gemini.json
├── prompts/
│   ├── chat.txt
│   ├── debug.txt
│   └── ...
├── go.mod
├── go.sum
└── README.md
```

## Next Steps

1. Read [01-PROJECT-SETUP.md](./01-PROJECT-SETUP.md) for initial setup
2. Implement MCP server following [02-MCP-SERVER.md](./02-MCP-SERVER.md)
3. Continue through documents in order

---

**Total Estimated Effort**: 6-8 weeks for full feature parity
**Recommended Team Size**: 1-2 developers
**Go Version Required**: 1.22+
