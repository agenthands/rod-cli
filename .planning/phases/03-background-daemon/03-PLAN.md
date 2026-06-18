# Phase 3: Background Daemon & Session Management

**Status:** Planned
**Goal:** Transform rod-cli into a persistent background daemon, enabling shared state and robust zombie browser prevention.
**Requirements:** DAEM-01, DAEM-02, DAEM-03, DAEM-04, SESS-01, SESS-02, SESS-03, SESS-04

<domain>
## Context & Scope
In Phase 2, `rod-cli` functioned by launching a browser, running a single command, and tearing it down. This is prohibitively slow for LLM agent workflows that require iterative steps (e.g., open -> read -> click -> type). This phase restructures the CLI to implicitly spawn and communicate with a background daemon. The daemon holds the persistent `rodCtx` (browser instance) and handles incoming CLI requests via local IPC/HTTP. It must include rigorous "Zombie Safeguards" (PPID polling, idle timeouts, explicit teardown) to ensure orphaned Chromium instances never leak.
</domain>

<design>
## Technical Design
1. **Client/Daemon Architecture**:
   - When a CLI command (e.g., `rod-cli goto`) runs, it checks for a local daemon socket or HTTP port.
   - If missing, the CLI spawns a detached background `rod-cli daemon` process.
   - The CLI forwards the command request to the daemon.
2. **Zombie Safeguards**:
   - **PPID Polling**: When spawning the daemon, the CLI passes its parent process ID (the Agent framework's PID). The daemon polls this PID and exits if it dies.
   - **Idle Timeout**: The daemon resets a 15-minute timer on every request. If it expires, the daemon exits.
   - **Teardown Command**: A new `rod-cli close` (or `kill-all`) command forces the daemon to exit immediately.
3. **Session Management**:
   - Named sessions (`-s`) will translate to unique socket files (e.g., `/tmp/rod-cli-SESSION.sock`).
   - The daemon handles attachment (`--cdp`) internally during `rodCtx` initialization.
</design>

<tasks>
## Task Breakdown

### 1. IPC / Daemon Foundation
- Implement an HTTP or Unix Socket server inside `cmd.go` or a new `daemon` package.
- Modify `runWithContext` in `main.go` to act as an HTTP client if not running in daemon mode.
- Create a `daemon` hidden subcommand that actually boots the server.

### 2. Auto-spawning the Daemon (DAEM-01)
- If the client fails to connect to the socket, use `os/exec` to start `rod-cli daemon` in the background.
- Pass the parent's PPID (`os.Getppid()`) to the daemon via a flag or environment variable.

### 3. Zombie Safeguards (DAEM-02, DAEM-03, DAEM-04)
- **DAEM-02**: In the daemon, start a goroutine that checks if the given PPID is alive every X seconds. If dead, call `rodCtx.Close()` and `os.Exit(0)`.
- **DAEM-03**: Add a `close` CLI command that sends a shutdown request to the daemon.
- **DAEM-04**: Wrap the daemon's request handler with logic that resets a 15-minute `time.Timer`. If the timer fires, shut down gracefully.

### 4. Named Sessions & Attachment (SESS-01, SESS-02, SESS-03)
- Use the session name (`-s`) to determine the socket file/port.
- Pass `--cdp` when spawning the daemon to attach to an external browser.
- Support `--persistent` (user data dir) when initializing the browser.
</tasks>

<verification>
## Acceptance Criteria
- [ ] Running multiple CLI commands in sequence uses the same browser window.
- [ ] The background daemon automatically closes if the parent process (the agent) dies.
- [ ] The background daemon automatically closes after 15 minutes of inactivity.
- [ ] Running `rod-cli close` explicitly kills the daemon and browser.
- [ ] Using `-s` allows running two isolated browsers simultaneously.
</verification>

---
*Plan created: 2026-06-18*
