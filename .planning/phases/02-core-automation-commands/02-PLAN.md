# Phase 2: Core Automation Commands

**Status:** Planned
**Goal:** Port basic browser controls into the CLI framework.
**Requirements:** AUTO-01, AUTO-02, AUTO-03, AUTO-04

<domain>
## Context & Scope
This phase transforms `rod-cli` from an MCP server into a fully functional standalone CLI for browser automation. The goal is to bind the existing `go-rod/rod` interaction logic—currently buried inside MCP tool handlers—into the `urfave/cli/v2` routing layer. This involves commands for navigation (`goto`, `open`), interaction (`click`, `type`, `fill`), DOM evaluation (`eval`), and exporting state (`snapshot`, `screenshot`).
</domain>

<design>
## Technical Design
The current system parses MCP JSON arguments and passes them to tool functions in `tools/tools.go`. To support native CLI usage without JSON overhead:
1. **Action Extraction**: Extract the core `go-rod` execution logic out of the `mcp-go` specific handlers into reusable, strongly-typed Go functions within the `tools` package (or a new `actions` package).
2. **CLI Bindings**: Create `cli.Command` definitions in `cmd.go` (or a dedicated `cli_commands.go` file) that parse command-line arguments and invoke these reusable functions.
3. **Output Formatting**: The CLI commands must respect the `--raw` and `--json` global flags introduced in Phase 1, formatting the output accordingly before writing to standard out.
</design>

<tasks>
## Task Breakdown

### 1. Refactor Tool Logic for Reuse
- Modify `tools/tools.go` to separate the MCP JSON-parsing layer from the underlying execution logic.
- Ensure the underlying functions accept standard Go types (strings, ints) and return typed results or errors, rather than MCP-specific string blobs.

### 2. Implement Navigation Commands (AUTO-01)
- Add CLI commands: `open`, `goto`, `reload`, `go-back`, `go-forward`.
- `goto` and `open` will accept a single URL argument.

### 3. Implement Interaction Commands (AUTO-02)
- Add CLI commands: `click`, `dblclick`, `type`, `fill`, `drag`, `drop`, `hover`, `select`.
- Argument parsing must support snapshot refs (e.g., `e15`) or CSS selectors/locators as fallback.

### 4. Implement DOM Evaluation (AUTO-03)
- Add CLI commands: `eval`, `generate-locator`.
- `eval` will accept a JS snippet string and an optional target element ref.

### 5. Implement Export Commands (AUTO-04)
- Add CLI commands: `snapshot`, `screenshot`, `pdf`.
- Wire up the existing snapshotting logic to output directly to the console or file.
</tasks>

<verification>
## Acceptance Criteria
- [ ] Running `rod-cli goto https://example.com` successfully navigates the browser.
- [ ] Running `rod-cli click e1` successfully clicks an element via its snapshot ref.
- [ ] Running `rod-cli snapshot --raw` outputs the raw YAML snapshot to stdout without extra formatting.
- [ ] The MCP server functionality remains intact (the shared logic is dual-purposed).
</verification>

---
*Plan created: 2026-06-18*
