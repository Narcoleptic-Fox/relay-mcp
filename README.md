# Relay MCP Server

A high-performance Model Context Protocol (MCP) server written in Go. Relay MCP provides a suite of powerful AI tools for software development, including multi-model consensus, extended reasoning, and CLI integration.

## Features

*   **Single Binary**: Deploy as a single executable with no dependencies.
*   **Multi-Provider Support**:
    *   Google Gemini
    *   OpenAI
    *   Azure OpenAI
    *   X.AI (Grok)
    *   DIAL
    *   OpenRouter
    *   Custom/Local (Ollama, vLLM)
*   **Advanced Workflows**:
    *   `thinkdeep`: Multi-stage problem analysis.
    *   `consensus`: Orchestrate debates between multiple AI models.
    *   `debug`: Systematic root cause analysis.
    *   `planner`: Interactive step-by-step planning.
*   **CLI Linking (Clink)**: Bridge external AI CLIs (like Gemini CLI) into your workflow.
*   **Conversation Memory**: Thread-based context management across tool calls.

## Quick Start

### Prerequisites

*   Go 1.22+ (to build from source)
*   API keys for your preferred providers

### Configuration

Copy `.env.example` to `.env` and configure your keys:

```bash
cp .env.example .env
```

Edit `.env`:

```bash
GEMINI_API_KEY=your-key-here
# OPENAI_API_KEY=your-key-here
# ...
```

### Build & Run

```bash
# Build
go build -o relay-mcp.exe ./cmd/relay-mcp

# Run
./relay-mcp.exe
```

### Integration with Claude Code

Configure Claude Code to use Relay MCP:

```json
{
  "mcpServers": {
    "relay": {
      "command": "/absolute/path/to/relay-mcp.exe",
      "args": [],
      "env": {
        "GEMINI_API_KEY": "your-key"
      }
    }
  }
}
```

## Tools

### Simple Tools
*   `chat`: General purpose chat with file context.
*   `apilookup`: Find documentation for libraries/APIs.
*   `challenge`: Critically analyze ideas or code.
*   `listmodels`: View available models.
*   `version`: Server version info.
*   `clink`: Execute external CLI agents.

### Workflow Tools
*   `thinkdeep`: Extended reasoning for complex problems.
*   `debug`: Debugging assistant with hypothesis testing.
*   `codereview`: Systematic code review.
*   `consensus`: Multi-model synthesis and debate.
*   `planner`: Implementation planning.
*   `analyze`: Codebase analysis.
*   `refactor`: Refactoring strategies.
*   `testgen`: Test suite generation.
*   `precommit`: Pre-commit validation.

## License

MIT
