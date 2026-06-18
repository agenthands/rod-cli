# rod-cli

## What This Is

rod-cli is a lightweight, zero-dependency command-line interface (CLI) that provides AI assistants with native web browsing, scraping, and interaction capabilities. Built in Go, it replaces bulky Node.js setups and acts as a token-efficient "Skill" for LLMs, communicating via standard input/output (stdio). 

## Core Value

Native, token-efficient browser automation via standard I/O explicitly designed for LLM integration, avoiding the overhead of heavy Node.js runtimes and massive DOM accessibility trees.

## Requirements

### Validated

- ✓ Basic architecture as a standalone Go binary using `go-rod`.
- ✓ Communication via stdio for LLMs (currently via MCP).
- ✓ Initial snapshotting and JavaScript injection logic.

### Active

- [ ] Rename module, imports, and executable references from `rod-mcp` to `rod-cli`.
- [ ] Implement standalone CLI command parsing matching the README specs (e.g., `open`, `goto`, `click`, `type`, `snapshot`).
- [ ] Implement `--raw` flag to strip verbose output and yield direct results for piping.
- [ ] Support multi-session management (`-s=mysession`) and remote browser attachment (`--cdp`, `--extension`).
- [ ] Expand tool coverage to support the full categorized suite: Core, Navigation, Keyboard, Mouse, Storage, Network, and DevTools.
- [ ] Enable `rod-cli show --annotate` for interactive design feedback flows.

### Out of Scope

- Full Node.js runtime compatibility — we are actively moving away from Node.js dependencies for execution.
- Heavy DOM scraping for non-LLM use cases — outputs are explicitly optimized for token-efficiency rather than traditional web scraping verbosity.

## Context

The project currently exists as an MCP server (`rod-mcp`). The goal is to transform it into a direct CLI tool (`rod-cli`) that retains the capability to operate seamlessly with LLMs via stdio but offers a vast array of direct command-line operations (similar to `playwright-cli`). This requires refactoring the tool handler layer to be invokable via `urfave/cli/v2` directly rather than exclusively through the MCP JSON-RPC protocol, though the stdio interaction pattern remains central.

## Constraints

- **Language**: Go 1.23+
- **Dependency Hell**: Must remain a single compiled Go binary. Zero Node.js or Python runtime requirements for the end user.
- **Output Size**: Must aggressively optimize the context window size by converting pages to LLM-friendly Markdown and stripping DOM noise.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Transform from MCP Server to CLI | Allows usage both as an interactive CLI and an automated LLM skill, providing a more versatile developer experience. | — Pending |

---
*Last updated: 2026-06-18 after initialization*
