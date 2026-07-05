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

rod-cli has completed its **v2.2 Pi Extension** milestone (shipped 2026-06-27,
close commit `e90b5d4`, audit + security review PASSED). v2.2 delivered a
first-class TypeScript Pi extension (`extensions/pi/`, `@agenthands/rod-cli-pi`)
wrapping rod-cli as 13 native `browse_*` tools with lifecycle hooks, argv-injection
hardening, and 116 tests. The extension is **not yet published to npm** and is
consumed by OMP only through a local `omp plugin link` symlink.

Previous milestones: v2.1 hardened the CDP proxy + diagnostics; v2.0 shipped the
in-process MITM WebSocket CDP proxy (`internal/cdpproxy/`); v1.7–v1.9 built and
proved the stealth/evasion stack and closed hygiene/debt; earlier milestones built
the CLI, godoll migration, plugin architecture, and docs.

**TLS fingerprint spoofing remains permanently out of scope** — rod-cli drives
real Chrome, whose TLS/JA3 is authentic by construction.

## Current Milestone: v2.3 Distribution & Publishing

**Goal:** Make the browser-automation extension actually consumable — publish the
Pi extension to npm, and build/test/publish a native OMP extension (OMP is the
primary coding agent) — on a shared, host-agnostic core with a CI-driven release
pipeline.

**Target features:**
- Shared runtime-agnostic core (binary discovery, `execRodCli`, input validation,
  timeouts, per-tool argv builders) consumed by both host adapters — no logic
  duplication, no drift.
- `@agenthands/rod-cli-pi` made publish-ready and shipped to npm (public scoped
  package, LICENSE, verified tarball, working `pi install npm:…`).
- `@agenthands/rod-cli-omp` — a native OMP extension (`@oh-my-pi/pi-coding-agent`,
  `omp.extensions` manifest, injected `pi.zod`/`pi.typebox` schemas, `bun:test`),
  all 13 `browse_*` tools + lifecycle, installable via `omp plugin install`.
- CI extended to typecheck + test both packages; auto-publish on an `ext-v*` tag
  with `NPM_TOKEN`; extension semver independent of the Go binary `v*` tags.
- Documentation: corrected install paths, Pi-vs-OMP-vs-Skill guidance, discoverable
  from the README.

<details>
<summary>Archived: v2.2 Pi Extension (Shipped 2026-06-27)</summary>

**Goal:** Build a first-class TypeScript Pi extension that wraps rod-cli's browser
automation as native Pi tools — so a Pi coding agent discovers and drives a browser
through rod-cli without manual shell commands.

**Delivered (4 phases, 47–50):** npm scaffold + binary discovery + `pi.exec` wrapper +
lifecycle hooks (Phase 47); 7 core `browse_*` tools + integration test (Phase 48);
6 extended tools + argv-injection `--` backfill (Phase 49); README + skill-vs-extension
docs + cross-links (Phase 50). 13 tools, 116 tests, `tsc --noEmit` strict clean, security
review PASSED. Zero Go changes. Not yet on npm — that is v2.3's job.
</details>

<details>
<summary>Archived: v2.1 CDP Proxy Hardening & Diagnostics (Shipped 2026-06-27)</summary>

**Goal:** Close the v2.0 carry-forward items (live-browser proxy test, jitter validation,
cdp-traffic sensitivity warning) and the long-standing v1.7 font-spoof no-op.

**Delivered (4 phases consolidated to 3):**
- **Phase 43: Proxy Integration Test** — `TestProxyTraffic` (cdp-traffic JSON assertion)
  and `TestProxyCdpTell` (live-page normalization probe)
- **Phase 44: Jitter Validation + Sensitivity Warning** — stderr soft-warning when
  `--cdp-jitter-ms > 1000`, cdp-traffic Usage sensitivity caveat
- **Phase 45-46: Font Spoofing** — confirmed godoll injector already real (godoll
  `1d90494`); existing `TestFontSpoof` covers FONT-04..07; detect.js comment updated
</details>

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
- See **Current Milestone: v2.3** target features. Detailed REQ-IDs live in `.planning/REQUIREMENTS.md`.

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
| Ship extensions as shared-core + 2 thin adapters | One host-agnostic core; per-host adapters differ only in scope/schema/test runner — DRY, no drift, both npm-publishable. | 🔨 v2.3 |

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
*Last updated: 2026-07-05 — v2.3 milestone started (Distribution & Publishing). v2.2 (Pi Extension) complete and archived.*
