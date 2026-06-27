---
author: architect (in-band verification)
phase: 47
verdict: passed
verified_at: 2026-06-27
---

# Phase 47 Verification — Extension Foundation + Lifecycle

## Goal-backward assessment

### FOUND-01: Package scaffold
**Status: ✅ PASSED** — `package.json` has `"pi": { "extensions": ["./src/index.ts"] }`, `"type": "module"`, correct peerDeps (`@earendil-works/pi-coding-agent`, `@earendil-works/pi-ai`, `typebox`). `tsconfig.json` has `noEmit: true`, strict mode with 7 strictness flags.

### FOUND-02: ExtensionAPI entry point
**Status: ✅ PASSED** — `src/index.ts` default export calls `setPi(pi)`, `findRodCli()`, and `registerLifecycle(pi, rodCliPath)` synchronously during load.

### FOUND-03: Binary discovery
**Status: ✅ PASSED** — `findRodCli()` resolves: `ROD_CLI_PATH` → `PATH` (with `;` separator on Windows) → `$GOBIN`/`~/go/bin`. Windows support: `.exe` extension, `USERPROFILE`, `;` PATH. Uses `statSync` with `throwIfNoEntry: false`. Returns `null` (not throw) if binary not found.

### FOUND-04: Shell-out wrapper
**Status: ✅ PASSED** — `execRodCli()` throws on non-zero exit (`if (result.code !== 0) throw new Error(...)`). `validateInput()` checks: goto URL format (http/https), click/fill/type non-empty selector, eval ≤10KB. Per-command timeout table covers goto/snapshot/click/fill/type/eval/screenshot/wait/close/version. AbortSignal propagated. `--raw` flag prepended for clean output.

### FOUND-05: Smoke test
**Status: ✅ PASSED** — vitest config includes `src/__tests__/**/*.test.ts`. Smoke test has 4 tests: findRodCli resolves, SessionParam exported, registerLifecycle is function, default export is function.

### LIFECYCLE-01: session_start
**Status: ✅ PASSED** — Verifies binary via `rod-cli --version`, notifies user with version string. If binary missing: warns with install instructions. Does NOT start browser daemon.

### LIFECYCLE-02: session_shutdown
**Status: ✅ PASSED** — Uses correct event name `session_shutdown` (NOT `session_end`). Gates `rod-cli close` on `event.reason === "quit"` — does NOT close on reload/fork/resume. Best-effort (catch + ignore errors). Daemon's own PPID polling is the safety net.

### LIFECYCLE-03: Lazy-start
**Status: ✅ PASSED** — Extension factory never calls a browser-launch command. `session_start` only runs `--version`. The daemon will start implicitly on first `browse_goto` in Phase 48.

## Invariants

| Invariant | Status |
|-----------|--------|
| I1: errors thrown, never `isError` | ✅ `throw new Error(...)` at line 120 |
| I2: StringEnum for enums | ✅ Deferred to Phase 48 (no tools yet) |
| I3: `session_shutdown` not `session_end` | ✅ Line 25 of lifecycle.ts |
| I4: no browser at load | ✅ Factory + session_start are browser-free |

## Verdict: passed

All 8 postconditions met. All invariants preserved. Ready for Phase 48.
