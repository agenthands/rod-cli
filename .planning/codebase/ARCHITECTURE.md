# Architecture

**Mapped:** 2026-06-18

## System Design

The `rod-cli` application is a Go-based Model Context Protocol (MCP) server that provides browser automation tools for LLMs. 

### Core Components

1. **CLI Layer (`cmd.go`, `main.go`)**: 
   - Handles startup configuration, logging initialization, and command-line flags.
   - Parses flags (like `--headless`, `--config`, `--cdp-endpoint`) and injects them into the global configuration.

2. **Runner & Server (`runner.go`, `server.go`)**:
   - `Runner`: Orchestrates the lifecycle of the application, managing signals and graceful shutdown.
   - `Server`: Wraps the `mcp-go` library to instantiate an MCP server using Stdio transport.
   - Tool Registration: Based on the operational mode (e.g., `Text` or `Vision`), `server.go` registers specific tool subsets to the MCP server.

3. **Tool Handlers (`tools/` package)**:
   - Contains the logic for the actual browser commands (e.g., open, click, type, snapshot).
   - Tools interact with the browser via the `go-rod/rod` library.
   - Focuses on generating context-optimized outputs (e.g., converting heavy DOM to lightweight markdown/JSON).

4. **Context & Types (`types/` package)**:
   - Holds shared domain models, configuration structures, and the global execution context (`context.go`).
   - Maintains the active `rod.Browser` and `rod.Page` instances.
   - Includes JavaScript snippets (e.g., `snapshotter.js`) that are injected into the page to extract accessibility trees or clean DOM representations.

## Data Flow

1. LLM sends a tool execution request via stdin (MCP protocol).
2. `server.MCPServer` routes the request to the matching `ToolHandler` in the `tools` package.
3. The `ToolHandler` retrieves the active browser context from `types.Context`.
4. It executes the rod action (e.g., clicking an element based on a snapshot ref).
5. The result (or a new page snapshot) is serialized and returned via stdout.
