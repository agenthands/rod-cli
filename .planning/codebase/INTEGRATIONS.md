# Integrations

**Mapped:** 2026-06-18

## External Integrations

- **Model Context Protocol (MCP)**: Implements the MCP server protocol using `mark3labs/mcp-go` to expose browser control tools to LLMs.
  - Transport: Stdio (standard input/output), running as a background process.
  - LLM Clients: Claude, Gemini, and other agents that consume MCP servers.
- **Chrome DevTools Protocol (CDP)**: Inherited via `go-rod/rod`, allowing low-level browser interaction. Can attach to existing browser sessions (`--cdp` flags).

## APIs & Data Flow

- **Command Line Input**: Accepts CLI arguments using `urfave/cli/v2`.
- **Stdio RPC**: Main interaction loop for the MCP server. Receives tool calls and outputs tool results over stdin/stdout.
