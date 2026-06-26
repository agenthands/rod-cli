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

rod-cli has completed its **v1.7 Complete Evasion Stack** milestone (shipped 2026-06-26). Building on v1.6's proven, configurable JS-layer stealth, v1.7 reduced the CDP transport footprint (a plain `goto` now enables none of the Runtime/Network/Fetch CDP domains — capture is opt-in, the interceptor is lazy, and HTTP↔JS identity coherence moved to the zero-enable `Emulation` domain), shipped a curated library of 6 vetted **Chrome-only** device profiles (embedded, `--profile=list`, with a real vetting gate), and activated godoll's dormant fingerprint dimensions (fonts/media-devices/battery/codecs) coherently behind 4 new hardening toggles. **TLS fingerprint spoofing was deliberately ruled out** — rod-cli drives real Chrome, whose TLS/JA3 is authentic by construction (TLS spoofing lives in the separate "munch" project). The milestone passed an independent security review with no blocker.

## Current Milestone: v1.8 Debt Cleanup & Coding-Assistant Onboarding

**Goal:** Retire the three v1.7 follow-ups and ship authoritative install + agent-skill documentation so any of the five major coding assistants can adopt rod-cli.

**Target features:**
- Toolchain bump go1.26.0 → go1.26.1 (security F1); build/CI green.
- Plugin-path CDP-ledger fix — lazy CDP domains enabled on the plugin path are tracked in the per-session ledger and harness-asserted (closes the v1.7 coverage hole).
- Real, observable font-spoof — replace the godoll font-injector no-op with actual font-list spoofing the harness asserts changes the page's detected fonts, stable within a session.
- Coding-assistant install & skill docs — for Claude (Code), Codex CLI, Gemini CLI, Pi (pi.dev), and opencode: binary install AND how each tool discovers/loads rod-cli as an agent-skill (not just `go install`).

**Key context:** TLS spoofing stays out of scope (real Chrome only). Phase numbering continues from 33 (this milestone starts at Phase 34). Detailed REQ-IDs live in `.planning/REQUIREMENTS.md`.

<details>
<summary>Archived: v1.7 Complete Evasion Stack (Shipped 2026-06-26)</summary>

**Goal:** Extend rod-cli's stealth from JS-layer fingerprinting toward a fuller evasion solution — reducing CDP signals, curated Chrome-only device profiles, and expanded fingerprint hardening. (TLS spoofing ruled out: real Chrome only.)

**Delivered (3 phases, 7 plans; Phase 31/TLS cancelled):**
- **CDP Footprint Reduction (Phase 30):** plain `goto` enables none of Runtime/Network/Fetch; opt-in capture flags; lazy interceptor; zero-enable Emulation identity (design fork C+D); per-command inventory + honest ceiling; deterministic harness baseline gate.
- **Profile Library (Phase 32):** 6 embedded Chrome-only profiles, `--profile=list`, built-in-first resolution, real PROF-02 vetting gate.
- **Advanced Evasion (Phase 33):** godoll fonts/media-devices/battery/codecs dimensions activated coherently (OS-constrained), 4 CLI>profile>default toggles, harness-asserted.

**Note:** TLS-01..04 / Phase 31 cancelled by operator constraint (real-Chrome-only; TLS spoofing handled in "munch"). Known follow-ups: godoll font-spoof no-op; plugin-path CDP-ledger gap; go1.26.1 toolchain bump. Full detail: `.planning/milestones/v1.7-*`.
</details>

<details>
<summary>Archived: v1.5 Plugin Ecosystem Documentation</summary>

**Goal:** Ship complete, authoritative documentation for the plugin ecosystem in a `docs/plugins/` tree, backed by runnable example plugins that exercise every hook and API.

**Delivered:** reference pages (lifecycle hooks, state/context API, CLI commands); a flagship XSS-scanner worked example, per-hook recipes, and a copyable starter; an authoring tutorial and a README-linked docs index. Three small engine fixes landed (`api.GetLocalStorage()`, functional `plugin run` via `PluginEngine.RunFunc`, CDP DOM-domain enable for `onDOMNodeInserted`); the startup banner was removed for token-efficiency. Brownfield — documented the shipped v1.4 system; gaps surfaced were treated as small corrective fixes. Full detail: `.planning/milestones/v1.5-ROADMAP.md`.
</details>

<details>
<summary>Archived: v1.4 Plugin Architecture</summary>

**Goal:** Design and implement a generic plugin system for `rod-cli` allowing dynamic execution of external scripts or modules to hook into browser lifecycle events.

**Target features:**
- Engine Selection: Implement embedded script engine (e.g. `goja` or Lua) or Wasm for plugin execution.
- Lifecycle Hooks: Add event emitters to the daemon for `OnRequest`, `OnResponse`, `OnLoad`, and `OnDOMNodeInserted`.
- Plugin CLI Commands: Add `rod-cli plugin load <path>`, `rod-cli plugin list`, and `rod-cli plugin run`.
- State Sharing: Allow plugins to safely read token-optimized snapshots and network context from the `godoll` daemon.
</details>

<details>
<summary>Archived: v1.3 Godoll Migration</summary>

**Goal:** Complete the migration to `godoll` to leverage its full suite of evasion, network interception, and robust interaction features, while achieving parity with Playwright via an automatic browser installer.

**Target features:**
- Implement `rod-cli install` command to auto-download Chromium.
- Replace inline launcher with `godoll.NewBrowser(opts)` and stealth presets.
- Replace raw `HijackRequests` with `godoll/network.NewInterceptor()`.
- Replace `page.Mouse.Scroll` with `humanize.Scroll()`.
- Wrap critical DOM actions with `godoll/retry` exponential backoff.
- Standardize remote connectivity using `godoll/browser.ConnectToRemoteBrowser()`.
</details>

<details>
<summary>Archived: v1.2 First-Class Agent Skills & Documentation</summary>

**Goal:** Elevate the `rod-cli` LLM skill integrations to first-class citizens and overhaul the project documentation to present it as a polished, agent-ready tool.

**Target features:**
- Promote the skill configuration from `example/skills/` to an official `skills/rod-cli/` directory.
- Update the documentation (README, USAGE, INSTALL) to reflect its primary use case: native web automation for agents like Gemini and Codex.
- Strip Node.js/NPM installation references and replace them with `go install`.
</details>

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
- ✓ [STEALTH-03] Implement `humanize` mouse and typing handlers over `rod-cli` actions. - v1.1
- ✓ [STEALTH-01] Import `godoll` network module for advanced request mocking. - delivered v1.3, validated v1.6
- ✓ [STEALTH-02] Replace inline `go-rod` browser initialization with `godoll.NewBrowser()`. - delivered v1.3, validated v1.6

### Active
- See **Current Milestone: v1.6** target features. Detailed REQ-IDs live in `.planning/REQUIREMENTS.md`.

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
| Integrate godoll stealth | Prevent modern bot detection systems from blocking rod-cli sessions. | ✅ Wired v1.3; proven & extended v1.6 |

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
*Last updated: 2026-06-26 — v1.8 milestone started (Debt Cleanup & Coding-Assistant Onboarding). v1.7 complete and archived.*
