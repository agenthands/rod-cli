# Structure

**Mapped:** 2026-06-18

## Directory Layout

- `/` (Root)
  - `main.go`, `cmd.go`: Application entry point and CLI flag parsing.
  - `server.go`, `runner.go`: MCP server initialization, lifecycle management, and tool registration.
  - `go.mod`, `go.sum`: Go dependency definitions.
  - `package.json`: Contains Node.js scripts for JS minification (`terser`).
- `/tools/`
  - Implementation of individual MCP tools.
  - `common.go`: Shared utilities for tool execution.
  - `snapshot.go`: Logic for capturing page state and formatting it for LLMs.
  - `tools.go`: Defines the registry of available tools.
- `/types/`
  - `context.go`: Wrapper around `go-rod`'s browser context.
  - `config.go`: Configuration structures.
  - `logger.go`: Logging initialization.
  - `snapshot.go`: Data models for page snapshots.
  - `/js/`: Contains client-side JavaScript (`snapshotter_raw.js` and minified `snapshotter.js`) injected into the browser.
- `/utils/`
  - Generic helper functions (e.g., file system ops, formatting).
- `/banner/`
  - CLI startup banner logic.
- `/assets/`, `/resources/`
  - Static files and templates if any.

## Key Locations

- **Tool Registration**: `server.go` (`registerTools` method) and `tools/tools.go`.
- **Browser Context**: `types/context.go` controls the `rod.Browser` lifecycle.
- **Snapshot Logic**: Client-side extraction is in `types/js/snapshotter_raw.js`, while server-side handling is in `tools/snapshot.go`.
