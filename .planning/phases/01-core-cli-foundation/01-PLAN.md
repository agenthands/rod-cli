# Phase 1: Core CLI Foundation

**Status:** Planned
**Goal:** Establish the root CLI structure and rename the project.
**Requirements:** CLI-01, CLI-02, CLI-03

<domain>
## Context & Scope
The existing `rod-mcp` project currently acts solely as an MCP server. This phase introduces a standalone CLI execution model using `urfave/cli/v2`, renames all project references from `rod-mcp` to `rod-cli`, and establishes the foundational `--raw` and `--json` flags to format CLI command output.
</domain>

<design>
## Technical Design
1. **Module Renaming:** Update `go.mod` to `github.com/agenthands/rod-cli` (or similar standard target, keeping it consistent) and run `go mod tidy` to update all internal imports.
2. **CLI Commands:** Add specific CLI commands into `cmd.go` (e.g., `open`, `snapshot`) that will eventually wrap the existing tool logic. For Phase 1, we will register these commands as stubs that parse correctly.
3. **Output Formatting:** Introduce a global formatting flag structure in `SubCfg` (or equivalent `types.Config` extension) to capture `--raw` and `--json` and apply formatting to standard output.
</design>

<tasks>
## Task Breakdown

### 1. Rename Project Module (CLI-01)
- Edit `go.mod` to change `module` to `github.com/agenthands/rod-cli` (or current git path).
- Search and replace all internal imports of `github.com/agenthands/rod-cli` with the new module path.
- Update `package.json` names/descriptions if necessary.

### 2. Implement Root CLI Structure (CLI-02)
- Edit `cmd.go` to add `Commands` slice to `cli.App`.
- Create base command scaffolding for expected v1 commands (e.g., `open`, `goto`, `snapshot`).

### 3. Implement Global Output Flags (CLI-03)
- Add `--raw` and `--json` to `cli.App.Flags` in `cmd.go`.
- Plumb these flags into `types.Config` and create an output formatter utility function that tools can use to return raw or JSON structures.
</tasks>

<verification>
## Acceptance Criteria
- [ ] Running `go build` successfully compiles the `rod-cli` binary.
- [ ] Running `./rod-cli --help` shows the new commands and `--raw`/`--json` flags.
- [ ] The codebase has no residual references to `rod-mcp` in import paths.
</verification>

---
*Plan created: 2026-06-18*
