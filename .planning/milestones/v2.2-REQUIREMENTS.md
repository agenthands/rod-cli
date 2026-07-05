# Requirements — v2.2 Pi Extension

**Milestone goal:** Build a first-class TypeScript Pi extension that wraps rod-cli's browser automation as native Pi tools — so a Pi coding agent discovers and drives a browser through rod-cli without manual shell commands.

**Grounding:** rod-cli already ships a Pi Agent Skill (`skills/rod-cli/SKILL.md`, `docs/onboarding/pi.md`). This milestone upgrades that to a native TypeScript extension with typed tools, lifecycle hooks, and proper LLM prompt guidance. No Go code changes — the extension is pure TypeScript and communicates with rod-cli via `pi.exec()`.

---

## v2.2 Requirements

### Extension Foundation (FOUND)

- [ ] **FOUND-01:** npm package scaffold — `package.json` with `"pi": { "extensions": ["./src/index.ts"] }`, `tsconfig.json` with `"noEmit": true`, and the three peer dependencies (`@earendil-works/pi-coding-agent`, `@earendil-works/pi-ai`, `typebox`). The extension lives under `extensions/pi/`.
- [ ] **FOUND-02:** ExtensionAPI entry point — `src/index.ts` exports a default function that receives `ExtensionAPI`, resolves the rod-cli binary path, and registers all tools + lifecycle hooks synchronously during load.
- [ ] **FOUND-03:** Binary discovery — `findRodCli()` resolves the rod-cli binary at extension load via `ROD_CLI_PATH` env → `PATH` → `$GOBIN`/`~/go/bin`, with cross-platform support (`.exe` on Windows, `;` PATH separator, `USERPROFILE`). Cache the result; notify user on `session_start` if missing.
- [ ] **FOUND-04:** Shell-out wrapper — `execRodCli(args, opts)` wraps `pi.exec("rod-cli", [...args])` with per-command timeouts (60s goto, 15s snapshot/click/type/eval, 30s screenshot/wait, 5s close/version), error handling (throw on non-zero exit, never return `isError`), AbortSignal propagation, and input validation (URL format check, selector non-empty sanity, expression length cap ≤10KB, reject empty required params).
- [ ] **FOUND-05:** Integration smoke test — a `vitest`-based test in `src/__tests__/smoke.test.ts` verifies the extension loads without error and `findRodCli()` resolves or returns null gracefully.

### Core Browser Tools (TOOLS)

All tools use the flat `browse_` prefix (Pi ecosystem convention), accept an optional `session` param for named sessions, use `StringEnum` from `@earendil-works/pi-ai` for enum parameters (never `Type.Enum` — incompatible with Google API), and include `promptGuidelines` naming the tool explicitly.

- [ ] **TOOLS-01:** `browse_goto` — navigate to a URL. Implicitly starts the browser daemon on first call (rod-cli's existing `EnsureDaemon`). Parameters: `url` (string, required), `session` (string, optional).
- [ ] **TOOLS-02:** `browse_snapshot` — capture a token-efficient accessibility-tree markdown snapshot of the current page. Output truncated to 50KB/2000 lines (Pi's default ceiling). Parameters: `selector` (string, optional — scope to element), `session` (string, optional).
- [ ] **TOOLS-03:** `browse_click` — click an element by CSS selector. Parameters: `selector` (string, required), `doubleClick` (boolean, optional, default false), `session` (string, optional).
- [ ] **TOOLS-04:** `browse_type` — type text with humanized typing (sequential keystrokes, ~50-250ms per character). Maps 1:1 to `rod-cli type <sel> <text>`. Parameters: `selector` (string, required), `text` (string, required), `session` (string, optional). Does NOT submit — use `browse_press_key` or `browse_fill_form` for form submission.
- [ ] **TOOLS-05:** `browse_eval` — evaluate JavaScript on the page. Escape hatch for data extraction. Parameters: `expression` (string, required, max 10KB), `session` (string, optional). Output truncated to 50KB.
- [ ] **TOOLS-06:** `browse_screenshot` — capture a screenshot. Parameters: `selector` (string, optional — scope to element), `fullPage` (boolean, optional, default false), `format` (StringEnum `["png", "jpeg"]`, optional, default png), `session` (string, optional).
- [ ] **TOOLS-07:** `browse_wait` — wait for a selector to appear or a fixed duration. Parameters: `selector` (string, optional), `timeout` (number, optional — max wait ms), `session` (string, optional).
- [ ] **TOOLS-08:** `browse_tabs` — manage browser tabs. Parameters: `action` (StringEnum `["list", "new", "close", "select"]`, required), `url` (string, optional — for new tab), `index` (number, optional — for close/select), `session` (string, optional).
- [ ] **TOOLS-09:** `browse_navigate` — page navigation actions. Parameters: `action` (StringEnum `["reload", "back", "forward"]`, required), `session` (string, optional).
- [ ] **TOOLS-10:** `browse_scroll` — scroll the page or an element. Parameters: `selector` (string, optional — element to scroll), `direction` (StringEnum `["down", "up"]`, optional, default down), `distance` (number, optional — pixels), `session` (string, optional).
- [ ] **TOOLS-11:** `browse_cookies` — manage cookies. Parameters: `action` (StringEnum `["get", "set", "delete", "clear"]`, required), `name` (string, optional), `value` (string, optional), `url` (string, optional — for set), `session` (string, optional).
- [ ] **TOOLS-12:** `browse_storage` — manage localStorage and sessionStorage. Parameters: `action` (StringEnum `["get", "set", "delete", "clear"]`, required), `storageType` (StringEnum `["local", "session"]`, optional, default `"local"`), `key` (string, optional), `value` (string, optional), `session` (string, optional).
- [ ] **TOOLS-13:** `browse_fill_form` — fill form fields instantly (no humanized typing) using `rod-cli fill`. Parameters: `selector` (string, required — input element), `text` (string, required), `submit` (boolean, optional — press Enter after fill), `session` (string, optional). Also used for bulk form filling via repeated calls.

### Integration (INTEG)

- [ ] **INTEG-01:** Automated integration test — a vitest-based test in Phase 48 starts a local fixture page, runs the full 7-tool workflow (goto → snapshot → click → type → eval → screenshot → wait → close), asserts each tool returns correct output, and verifies the daemon is cleaned up. Uses a real rod-cli binary and a loopback HTTP fixture server.

### Daemon Lifecycle (LIFECYCLE)

- [ ] **LIFECYCLE-01:** `session_start` hook — verifies the rod-cli binary is on PATH by running `rod-cli --version`. If present: notify user with version. If missing: `ctx.ui.notify("rod-cli not found. Install: go install github.com/agenthands/rod-cli@latest", "warn")`. Does NOT start the browser daemon.
- [ ] **LIFECYCLE-02:** `session_shutdown` hook — runs `rod-cli close` (best-effort, errors caught, never throws). Gated on `event.reason === "quit"` — does NOT close daemon on session reload/fork/resume. Uses the correct event name `session_shutdown` (NOT `session_end`, which does not exist in Pi's event catalog).
- [ ] **LIFECYCLE-03:** Lazy-start — the browser daemon starts implicitly on the first `browse_goto` call via rod-cli's existing `EnsureDaemon`. No browser is launched for non-browsing Pi sessions. The extension never explicitly calls a "start" command.

### Documentation & Discoverability (DOCS)

- [ ] **DOCS-01:** `extensions/pi/README.md` — install instructions (`go install` + `pi install`), prerequisites (Go 1.23+, rod-cli binary), tool catalog with descriptions, and a verify-on-install step (ask Pi to "browse to example.com and tell me the page title").
- [ ] **DOCS-02:** Skill-vs-Extension comparison — a table in README.md contrasting the existing Agent Skill (`SKILL.md` shell-out) with the new Extension (typed tools, lifecycle hooks, error handling). Both paths are supported; neither is deprecated.
- [ ] **DOCS-03:** Cross-link — update `docs/onboarding/pi.md` to reference the extension as the richer path, keeping the skill instructions as the zero-install fallback. Update `skills/rod-cli/SKILL.md` to mention the extension for Pi users.
- [ ] **DOCS-04:** `extensions/pi/` added to top-level README's docs index so Pi users discover it from the project root.

## Traceability

| REQ-ID | Phase | Status |
|--------|-------|--------|
| FOUND-01 | TBD | pending |
| FOUND-02 | TBD | pending |
| FOUND-03 | TBD | pending |
| FOUND-04 | TBD | pending |
| FOUND-05 | TBD | pending |
| TOOLS-01 | TBD | pending |
| TOOLS-02 | TBD | pending |
| TOOLS-03 | TBD | pending |
| TOOLS-04 | TBD | pending |
| TOOLS-05 | TBD | pending |
| TOOLS-06 | TBD | pending |
| TOOLS-07 | TBD | pending |
| TOOLS-08 | TBD | pending |
| TOOLS-09 | TBD | pending |
| TOOLS-10 | TBD | pending |
| TOOLS-11 | TBD | pending |
| TOOLS-12 | TBD | pending |
| TOOLS-13 | TBD | pending |
| LIFECYCLE-01 | TBD | pending |
| LIFECYCLE-02 | TBD | pending |
| LIFECYCLE-03 | TBD | pending |
| DOCS-01 | TBD | pending |
| DOCS-02 | TBD | pending |
| DOCS-03 | TBD | pending |
| DOCS-04 | TBD | pending |
| INTEG-01 | TBD | pending |
