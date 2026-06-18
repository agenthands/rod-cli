# Requirements: rod-cli

**Defined:** 2026-06-18
**Core Value:** Native, token-efficient browser automation via standard I/O explicitly designed for LLM integration, avoiding the overhead of heavy Node.js runtimes.

## v1 Requirements

### Core CLI Foundation

- [ ] **CLI-01**: Rename project module and references from `rod-mcp` to `rod-cli`.
- [ ] **CLI-02**: Implement root CLI command structure using `urfave/cli/v2` for all planned actions.
- [ ] **CLI-03**: Implement global `--raw` and `--json` flags for output formatting.

### Core Automation Commands

- [ ] **AUTO-01**: Implement `open`, `goto`, `reload`, `go-back`, `go-forward`.
- [ ] **AUTO-02**: Implement interaction commands: `click`, `dblclick`, `type`, `fill`, `drag`, `drop`, `hover`, `select`.
- [ ] **AUTO-03**: Implement DOM evaluation commands: `eval`, `generate-locator`.
- [ ] **AUTO-04**: Implement save/export commands: `snapshot`, `screenshot`, `pdf`.

### Background Daemon & Session Management

- [ ] **DAEM-01**: Implement Background Session (Default). The first command boots up a browser instance in the background and keeps it running. Subsequent commands communicate with it.
- [ ] **DAEM-02**: Implement Zombie Safeguard: Parent Process ID (PPID) Polling. The daemon captures the caller's PID and periodically polls it. If the parent dies, the daemon gracefully kills the browser and exits.
- [ ] **DAEM-03**: Implement Zombie Safeguard: Explicit Teardown hooks. Create `rod-cli close` and `rod-cli kill-all` commands.
- [ ] **DAEM-04**: Implement Zombie Safeguard: Idle Timeout. Enforce a 15-minute inactivity timer that kills the daemon if no commands are received.
- [ ] **SESS-01**: Support named sessions via global `-s` flag to spawn separate isolated browser sessions.
- [ ] **SESS-02**: Implement `attach` command to connect to an external browser via `--cdp`.
- [ ] **SESS-03**: Support persistent profiles (`--persistent`, `--profile`).
- [ ] **SESS-04**: Implement tab management: `tab-list`, `tab-new`, `tab-close`, `tab-select`.

### Advanced Web Interactions

- [ ] **ADV-01**: Implement input simulations: `press`, `keydown`, `keyup`, `mousemove`, `mousedown`, `mouseup`, `mousewheel`.
- [ ] **ADV-02**: Implement dialog handlers: `dialog-accept`, `dialog-dismiss`.
- [ ] **ADV-03**: Implement DevTools/network commands: `route`, `unroute`, `console`, `requests`, `tracing-start`, `tracing-stop`.
- [ ] **ADV-04**: Implement storage commands: `cookie-*`, `localstorage-*`, `sessionstorage-*`, `state-save`, `state-load`.

### Annotation & Debugging

- [ ] **DBG-01**: Implement `video-*` commands for action annotation and recording.
- [ ] **DBG-02**: Implement `highlight` commands for persistent visual overlays.
- [ ] **DBG-03**: Implement `show --annotate` for interactive user feedback sessions.

## v2 Requirements

### Extended Integration

- **EXT-01**: Create installable wrapper scripts (e.g., `npx rod-cli` equivalent if needed, though Go binary is preferred).
- **EXT-02**: Implement specialized extraction commands tailored for specific complex websites.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Native Node.js bindings | The project explicitly aims to replace Node.js setups with a compiled Go binary. |
| Raw HTML dumping | Goes against the core value of context-window optimization for LLMs. |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CLI-01 | Phase 1 | Pending |
| CLI-02 | Phase 1 | Pending |
| CLI-03 | Phase 1 | Pending |
| AUTO-01 | Phase 2 | Pending |
| AUTO-02 | Phase 2 | Pending |
| AUTO-03 | Phase 2 | Pending |
| AUTO-04 | Phase 2 | Pending |
| DAEM-01 | Phase 3 | Pending |
| DAEM-02 | Phase 3 | Pending |
| DAEM-03 | Phase 3 | Pending |
| DAEM-04 | Phase 3 | Pending |
| SESS-01 | Phase 3 | Pending |
| SESS-02 | Phase 3 | Pending |
| SESS-03 | Phase 3 | Pending |
| SESS-04 | Phase 3 | Pending |
| ADV-01 | Phase 4 | Pending |
| ADV-02 | Phase 4 | Pending |
| ADV-03 | Phase 4 | Pending |
| ADV-04 | Phase 4 | Pending |
| DBG-01 | Phase 5 | Pending |
| DBG-02 | Phase 5 | Pending |
| DBG-03 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-18*
*Last updated: 2026-06-18 after initial definition*
