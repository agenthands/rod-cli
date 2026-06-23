---
phase: 22-example-plugins
plan: 01
subsystem: plugin-engine
tags: [plugin, goja, engine, cli]
requires:
  - "internal/plugin/engine.go PluginEngine.vm (goja runtime)"
  - "internal/plugin/lifecycle.go invokeJSFunc lookup pattern"
  - "types/context.go GetPluginEngine()"
provides:
  - "PluginEngine.RunFunc(name) (string, error) — CLI read path into the plugin VM"
  - "actions.PluginRun delegating to RunFunc (stub removed)"
affects:
  - "daemon plugin-run dispatch (behavior change: returns JS fn result, not stub string)"
tech-stack:
  added: []
  patterns:
    - "Mirror invokeJSFunc (vm.Get + goja.AssertFunction) but surface errors and stringify result"
key-files:
  created: []
  modified:
    - "internal/plugin/engine.go"
    - "actions/plugin.go"
    - "internal/plugin/engine_test.go"
decisions:
  - "RunFunc stringifies via res.String() (no Go-side json.Marshal); accessors already JSON.stringify"
  - "undefined/null result maps to empty string with nil error"
metrics:
  duration: "5m"
  completed: "2026-06-23"
status: complete
---

# Phase 22 Plan 01: Plugin Run Read-Path Fix Summary

Added `PluginEngine.RunFunc(name)` and rewired `actions.PluginRun` so `plugin run <name>` invokes a named JS function in the loaded plugin VM and returns its result, replacing the stub string and unblocking end-to-end observation of example-plugin findings.

## What Was Built

- **Task 1** — `func (e *PluginEngine) RunFunc(name string) (string, error)` in `internal/plugin/engine.go`. Mirrors `invokeJSFunc` lookup (`vm.Get` + `goja.AssertFunction`) but surfaces errors for nil VM (`plugin engine not initialized`, matching `LoadScript`), missing function, and non-callable values; wraps call errors; returns `""` for undefined/null and `res.String()` otherwise. No `encoding/json` import added. Commit `cbfdadd`.
- **Task 2** — Rewrote `actions.PluginRun` to guard empty names (`plugin function name is required`) and return `ctx.GetPluginEngine().RunFunc(name)` directly. Removed the `Triggered plugin %s` stub. `PluginLoad`/`PluginList` and the function signature unchanged. Commit `ec62f43`.
- **Task 3** — Four white-box tests in `internal/plugin/engine_test.go`: defined-fn returns `[{"a":1}]`/nil err, missing fn errors with empty result, nil VM errors without panic, non-callable value errors. Commit `f5f5bc8`.

## Verification

- `go build ./...` — exits 0.
- `go test ./internal/plugin/...` — all pass (engine + scanner packages), no regressions.
- `grep 'func (e \*PluginEngine) RunFunc' internal/plugin/engine.go` — present.
- `grep 'RunFunc(name)' actions/plugin.go` — present; `Triggered plugin` — absent.

## Deviations from Plan

None - plan executed exactly as written.

## TDD Gate Compliance

The plan structured the engine method (Task 1, `feat`) and its tests (Task 3, `test`) as separate tasks, so the commit order is `feat → feat → test` rather than strict test-before-feat. Both `tdd="true"` tasks' behaviors are fully covered by the Task 3 tests, which were added immediately after and pass against the implementation. Build/grep verification gated Tasks 1-2; the full suite gated Task 3. No untested behavior shipped.

## Cross-Phase Note (deferred to a later 22 plan)

22-CONTEXT.md flags a REQUIRED follow-up: `docs/plugins/cli-reference.md` still documents `plugin run` as a stub and must be updated to describe it as "invokes the JS function named `<name>` and returns its result". That doc edit is out of this plan's `files_modified` scope (engine/action/test only) and belongs to a docs-focused plan in this phase.

## Self-Check: PASSED

- FOUND: internal/plugin/engine.go (RunFunc present)
- FOUND: actions/plugin.go (RunFunc(name) present, stub removed)
- FOUND: internal/plugin/engine_test.go (4 RunFunc tests)
- FOUND commits: cbfdadd, ec62f43, f5f5bc8
