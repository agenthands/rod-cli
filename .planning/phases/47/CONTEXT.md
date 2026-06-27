---
author: architect
responsible: architect
phase: 47
phase_type: implementation
hard_bar: true
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (FOUND-01..05, LIFECYCLE-01..03)
  - .planning/ROADMAP.md (v2.2 section)
  - .planning/research/STACK.md
  - .planning/research/FEATURES.md
  - .planning/research/PITFALLS.md
  - .planning/research/ARCHITECTURE.md
---

# Phase 47: Extension Foundation + Lifecycle — CONTEXT

## Goal

Ship a loadable Pi extension skeleton — npm package with correct `pi.extensions` metadata,
TypeScript config (`noEmit: true`), ExtensionAPI entry point that resolves the rod-cli
binary and registers lifecycle hooks. The extension loads without error in Pi;
`session_start` verifies the binary; `session_shutdown` cleans up the daemon on quit.
No tools yet — this is the verified foundation.

## {P} What the phase can assume (precondition)

- rod-cli binary exists and is buildable (`go build ./...` passes)
- Pi extension API: `import type { ExtensionAPI } from "@earendil-works/pi-coding-agent"`
- Pi loads `.ts` files directly; `tsconfig.json` has `"noEmit": true`
- rod-cli's `EnsureDaemon` is already lazy (starts browser on first command)
- Pi canonical teardown event is `session_shutdown` (NOT `session_end`)

## {Q} What the phase must establish (postcondition)

1. `extensions/pi/package.json` declares `"pi": { "extensions": ["./src/index.ts"] }` and
   the three peer dependencies (`@earendil-works/pi-coding-agent`, `@earendil-works/pi-ai`,
   `typebox`).
2. `extensions/pi/tsconfig.json` uses `"noEmit": true`, ES2022 target, bundler module resolution.
3. `extensions/pi/src/index.ts` exports a default function that receives `ExtensionAPI` and
   synchronously registers lifecycle hooks + the `execRodCli` helper.
4. `extensions/pi/src/cli.ts` exports `findRodCli()` (cross-platform binary resolution:
   `ROD_CLI_PATH` → `PATH` → `$GOBIN`/`~/go/bin`; `.exe` + `;` PATH + `USERPROFILE` on Windows)
   and `execRodCli(args, opts)` (timeout per command, input validation, throw-on-error,
   AbortSignal propagation).
5. `extensions/pi/src/lifecycle.ts` registers `session_start` (verify binary, notify user)
   and `session_shutdown` (close daemon gated on `reason === "quit"`, best-effort, never throws).
6. `extensions/pi/src/types.ts` exports shared TypeBox schemas (`SessionParam`).
7. Vitest smoke test (`src/__tests__/smoke.test.ts`) verifies extension loads + binary resolves.
8. Lifecycle integration tests (actual daemon start/stop) are explicitly deferred to Phase 48 —
   Phase 47 verifies hooks are REGISTERED and the binary is resolved.

## Invariants (cross-cutting, must preserve across phase)

- **I1 (error discipline):** All errors in `execute()` are THROWN, never returned as `isError`.
- **I2 (enum discipline):** All enum params use `StringEnum` from `@earendil-works/pi-ai`
  (never `Type.Enum` — incompatible with Google API).
- **I3 (hook names):** The exact Pi event names `session_start` and `session_shutdown` are
  used — `session_end` does not exist in Pi's event catalog.
- **I4 (no browser at load):** The extension factory and `session_start` hook never launch
  a browser. The daemon starts lazily on the first `browse_goto` call (Phase 48).

## Design decisions (none contentious — settled by research)

1. **Package location:** `extensions/pi/` at repo root. Not in `src/` (Go convention) —
   this is a TypeScript artifact distributed via npm, not a Go package.
2. **Package name:** `@agenthands/rod-cli-pi` (scoped under the agenthands org on npm).
3. **Binary resolution order:** `ROD_CLI_PATH` → `PATH` → `$GOBIN`/`~/go/bin`. Resolved
   once at extension load time, cached for the session.
4. **Cross-platform strategy:** `platform() === "win32"` detection for `.exe` extension,
   `;` PATH separator, and `USERPROFILE` (not `HOME`) on Windows. The extension fails
   gracefully (null return, user notification) if binary not found — it does not crash.
5. **Input validation:** Added to `execRodCli` — URL format check, selector non-empty
   sanity, eval expression ≤10KB, reject-empty-required-params. This is ~20 lines in
   `cli.ts` and prevents the most common LLM mistakes from reaching rod-cli.
6. **No Go changes:** Zero modifications to the Go codebase. The extension communicates
   exclusively via `pi.exec("rod-cli", [...args])`.

## Files

| File | Purpose |
|------|---------|
| `extensions/pi/package.json` | npm metadata + `pi.extensions` entry point |
| `extensions/pi/tsconfig.json` | TypeScript config (`noEmit: true`) |
| `extensions/pi/.gitignore` | node_modules, etc. |
| `extensions/pi/src/index.ts` | Default export: findRodCli + registerLifecycle |
| `extensions/pi/src/cli.ts` | `findRodCli()` + `execRodCli()` + `validateInput()` |
| `extensions/pi/src/lifecycle.ts` | `session_start` + `session_shutdown` hooks |
| `extensions/pi/src/types.ts` | Shared TypeBox schemas (`SessionParam`) |
| `extensions/pi/src/__tests__/smoke.test.ts` | Vitest smoke test |
| `extensions/pi/vitest.config.ts` | Vitest configuration |

## Success criteria (falsifiable)

1. `extensions/pi/package.json` has `"pi": { "extensions": ["./src/index.ts"] }` and the
   three peer dependencies declared. (FOUND-01)
2. `extensions/pi/tsconfig.json` has `"noEmit": true`, ES2022, bundler resolution. (FOUND-01)
3. `src/index.ts` default export registers hooks + resolves binary path synchronously. (FOUND-02)
4. `findRodCli()` resolves correctly on Linux/macOS. Windows support coded (tested if Windows
   available, otherwise graceful). (FOUND-03)
5. `execRodCli(args, opts)` throws on non-zero exit, validates inputs, applies per-command
   timeouts. (FOUND-04)
6. `npm test` (vitest) passes — smoke test loads extension, `findRodCli()` returns path or null. (FOUND-05)
7. `session_start` calls `rod-cli --version` and notifies. `session_shutdown` calls
   `rod-cli close` gated on `reason === "quit"`. (LIFECYCLE-01, LIFECYCLE-02)
