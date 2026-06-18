# Phase 4: Advanced Web Interactions

**Status:** Planned
**Goal:** Support complex inputs, storage, and networking.
**Requirements:** ADV-01, ADV-02, ADV-03, ADV-04

<domain>
## Context & Scope
Now that `rod-cli` has a persistent background daemon capable of holding state, it is uniquely positioned to handle advanced, stateful web interactions. This phase will expose native keyboard and mouse inputs, allowing agents to interact with non-standard DOM elements (like canvases or games). It will also expose DevTools networking controls (for intercepting requests or reading console logs) and storage controls (for manipulating cookies and local storage).
</domain>

<design>
## Technical Design
1. **Raw Input Simulations (ADV-01)**:
   - Expose `rodCtx.Page.Keyboard` and `rodCtx.Page.Mouse` methods directly.
   - Map CLI commands to `actions` package: `press`, `mousemove`, `mousedown`, etc.
2. **Dialog Handlers (ADV-02)**:
   - Provide a mechanism to asynchronously handle or immediately dismiss alert/confirm dialogs using `rodCtx.Page.HandleDialog()`.
3. **DevTools & Networking (ADV-03)**:
   - Expose basic route interceptions via `Page.HijackRequests`.
   - Allow dumping of the console log history.
4. **Storage Controls (ADV-04)**:
   - Implement commands to get and set cookies (`cookie-get`, `cookie-set`, `cookie-clear`).
   - Implement JS-based evaluation to read/write `localStorage` and `sessionStorage`.
</design>

<tasks>
## Task Breakdown

### 1. Implement Input Simulators (ADV-01)
- Add functions to `actions` package for `Press`, `MouseMove`, `MouseDown`, `MouseUp`.
- Expose these as new CLI commands (`press`, `mousemove`, `mousedown`, `mouseup`) in `cmd.go`.
- Ensure the daemon maps these correctly in `executeAction`.

### 2. Implement Dialog Handlers (ADV-02)
- Add `dialog-accept` and `dialog-dismiss` commands.
- Implement logic in `actions` to trigger `Page.HandleDialog()`.

### 3. Implement Network & Console Commands (ADV-03)
- Add `console` command to dump recent console logs.
- *(Note: complex async request routing via CLI strings is notoriously difficult. We will start by implementing a basic `requests` command to dump recent network requests, and defer complex active `route` interception to a potential future JSON-DSL configuration).*

### 4. Implement Storage Controls (ADV-04)
- Add `cookie-get`, `cookie-set`, and `cookie-clear` commands using `rodCtx.Browser.GetCookies()`, etc.
- Add `localstorage-get`, `localstorage-set`, `localstorage-clear` via `Page.Eval`.
- Add `sessionstorage-get`, `sessionstorage-set`, `sessionstorage-clear` via `Page.Eval`.
</tasks>

<verification>
## Acceptance Criteria
- [ ] Keyboard input command (`rod-cli press Enter`) executes without error.
- [ ] Mouse input command (`rod-cli mousemove 100 200`) executes without error.
- [ ] Storage commands can read, write, and clear cookies and localStorage.
- [ ] All new commands route successfully through the daemon IPC layer.
</verification>

---
*Plan created: 2026-06-18*
