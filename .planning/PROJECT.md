# rod-cli

## What This Is

rod-cli is a lightweight, zero-dependency command-line interface (CLI) that provides AI assistants with native web browsing, scraping, and interaction capabilities. Built in Go, it replaces bulky Node.js setups and acts as a token-efficient "Skill" for LLMs, communicating via standard input/output (stdio). 

## Core Value

Native, token-efficient browser automation via standard I/O explicitly designed for LLM integration, avoiding the overhead of heavy Node.js runtimes and massive DOM accessibility trees.

## Architecture: Persistent Background Daemon

`rod-cli` does not start the browser every time. That would be incredibly slow and resource-heavy. Instead, it relies on persistent background sessions:
1. **The Background Session (Default)**: The first command boots up a browser instance in the background and keeps it running in memory. Subsequent commands communicate with that exact same running instance.
2. **Named Sessions**: Multi-target workflows can spawn separate isolated sessions using the `-s` flag.
3. **The `attach` Command**: Connect the CLI to an external browser already started outside of the CLI using `--cdp`.
4. **Zombie Safeguards**: To prevent leaving zombie browsers, `rod-cli` uses Parent Process ID (PPID) polling, explicit teardown hooks (`kill-all`), and strict idle timeouts (15 minutes).

## Current State

rod-cli has completed its v1.1 Stealth & Humanization milestone. It successfully bypasses bot detection mechanisms natively using Bayesian fingerprint matching and Bezier-curve realistic human interactions.

## Next Milestone Goals

To be defined via `/gsd-new-milestone`.

<details>
<summary>Archived: v1.1 Stealth & Humanization</summary>

**Goal:** Adapt the `godoll` engine into `rod-cli` to provide robust bot detection evasion, realistic browser fingerprinting, and human-like interaction.

**Target features:**
- Replace standard `go-rod` browser initialization with `godoll` stealth launcher (spoofing navigator, WebGL, Canvas, plugins, etc.).
- Implement humanized input commands (realistic mouse movement trajectories, typing delays) replacing raw `go-rod` interactions.
- Introduce dynamic header and fingerprint injection based on `godoll`'s Bayesian network model.
</details>

## Requirements

### Validated

- ✓ Basic architecture as a standalone Go binary using `go-rod`.
- ✓ Communication via stdio for LLMs (currently via MCP).
- ✓ Initial snapshotting and JavaScript injection logic.
- ✓ Rename module, imports, and executable references from `rod-mcp` to `rod-cli`.
- ✓ Implement standalone CLI command parsing matching the README specs (e.g., `open`, `goto`, `click`, `type`, `snapshot`).
- ✓ Implement `--raw` flag to strip verbose output and yield direct results for piping.
- ✓ Support multi-session management (`-s=mysession`) and remote browser attachment (`--cdp`, `--extension`).
- ✓ Expand tool coverage to support the full categorized suite: Core, Navigation, Keyboard, Mouse, Storage, Network, and DevTools.
- ✓ Enable `rod-cli show --annotate` for interactive design feedback flows.

### Active

- [ ] Import `godoll` network, stealth, and browser launcher modules.
- [ ] Replace `go-rod` browser initialization with `godoll.NewBrowser()`.
- [ ] Implement `humanize` mouse and typing handlers over `rod-cli` actions.

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
| Transform from MCP Server to CLI | Allows usage both as an interactive CLI and an automated LLM skill, providing a more versatile developer experience. | ✅ Complete |
| Integrate godoll stealth | Prevent modern bot detection systems from blocking rod-cli sessions. | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-06-21 after initialization*
