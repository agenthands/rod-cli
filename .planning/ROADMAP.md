# Roadmap: rod-cli

**Created:** 2026-06-18
**Goal:** Transform rod-mcp into a full-featured CLI capable of serving both interactive human use cases and background LLM interactions seamlessly via stdio.

## Phases

### Phase 1: Core CLI Foundation
**Status:** Complete
**Goal:** Establish the root CLI structure and rename the project.
**Requirements:** CLI-01, CLI-02, CLI-03
- Restructure project to replace purely MCP-driven startup with `urfave/cli/v2` subcommands.
- Rename module and build outputs from `rod-mcp` to `rod-cli`.
- Implement `--raw` and `--json` foundational output formatting flags.

### Phase 2: Core Automation Commands
**Status:** Complete
**Goal:** Port basic browser controls into the CLI framework.
**Requirements:** AUTO-01, AUTO-02, AUTO-03, AUTO-04
- Connect rod actions (navigation, clicking, typing, evaluation) to CLI commands (`goto`, `click`, `eval`, etc.).
- Ensure that the commands gracefully wrap around the existing MCP tool handlers or extract their logic for reuse.
- Implement snapshot and screenshot generation commands.

### Phase 3: Background Daemon & Session Management
**Status:** Complete
**Goal:** Transform rod-cli into a persistent background daemon, enabling shared state and robust zombie browser prevention.
**Requirements:** DAEM-01, DAEM-02, DAEM-03, DAEM-04, SESS-01, SESS-02, SESS-03, SESS-04
- Implement the Background Session Daemon architecture so `rod-cli` processes commands against a single shared, running instance.
- Implement Zombie Safeguards: PPID polling, explicit `rod-cli close` teardown hooks, and a 15-minute idle timeout.
- Enable named sessions (`-s`) and user data dir persistence (`--persistent`).
- Support attaching to external browsers using `rod-cli attach --cdp`.

### Phase 4: Advanced Web Interactions
**Status:** Pending
**Goal:** Support complex inputs, storage, and networking.
**Requirements:** ADV-01, ADV-02, ADV-03, ADV-04
- Add DevTools networking interceptions (`route`, `unroute`, `requests`).
- Add storage manipulation commands (Cookies, LocalStorage, SessionStorage).
- Add raw mouse and keyboard simulators.

### Phase 5: Annotation & Debugging
**Status:** Pending
**Goal:** Deliver powerful feedback and diagnostic tools for developers.
**Requirements:** DBG-01, DBG-02, DBG-03
- Add visual highlighting and video recording capabilities.
- Implement the interactive `show --annotate` UI flow for design feedback.

---
*Roadmap generated: 2026-06-18*
