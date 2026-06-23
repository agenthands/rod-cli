---
phase: 22-example-plugins
reviewed: 2026-06-23T00:00:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - internal/plugin/engine.go
  - actions/plugin.go
  - internal/plugin/engine_test.go
  - plugins/examples/xss_scanner.js
  - plugins/examples/recipes/on_request.js
  - plugins/examples/recipes/on_response.js
  - plugins/examples/recipes/on_load.js
  - plugins/examples/recipes/on_dom_node_inserted.js
  - plugins/examples/starter.js
  - docs/plugins/examples/xss-scanner.md
  - docs/plugins/examples/recipes.md
  - docs/plugins/examples/starter.md
  - docs/plugins/cli-reference.md
findings:
  critical: 0
  warning: 0
  info: 3
  total: 3
status: clean
---

# Phase 22: Code Review Report

**Reviewed:** 2026-06-23
**Depth:** standard
**Files Reviewed:** 13
**Status:** clean

## Summary

Phase 22 adds one real engine change (`PluginEngine.RunFunc` + its `PluginRun` action wiring) plus example plugin scripts and documentation. I reviewed the engine change for panic safety and error wrapping, the action for delegation/guards, the tests for behavioral coverage, the example `.js` files for sandbox-illegal globals and payload defensiveness, and the docs for factual accuracy against source.

The engine code is correct and cannot panic on a missing or non-callable name. `RunFunc` guards nil VM, checks `vm.Get` for nil, uses `goja.AssertFunction` for callability, wraps call errors, and treats `nil`/undefined/null results as an empty string. `PluginRun` guards empty name and delegates cleanly — no stub remains, and the daemon dispatch (`daemon.go:205` → `actions.PluginRun`) is wired. All four `RunFunc` tests pass and assert the right behaviors (returns value, missing→error, nil VM→error, not-callable→error). `go build ./...` and `go vet` on the changed packages are clean.

The example scripts use only goja-available globals (`JSON`, `encodeURIComponent`, `String`/`Array` methods, `typeof`) — no `console`/`fs`/`require`/`window`/`document` — and every hook defends against missing/partial payloads before reading fields. I verified against the rod CDP proto structs that the field names the scripts read (`event.Request.URL`/`.Method`, `event.Response.URL`/`.Status`, `event.Node.NodeName`, `event.ParentNodeID`) match the exact Go field names goja exposes (no `FieldNameMapper` is registered, so capitalized Go names are correct). I also confirmed goja strips the trailing `error` from `api.GetSnapshot() (string, error)`, so `api.GetSnapshot()` correctly yields the string on success and throws into the `try/catch` on error — the scripts handle this correctly. All doc cross-links and source links resolve. The `/* TODO */` stubs in `starter.js` are intended template content, not a defect.

No blocking or warning-level defects were found. Three minor doc/robustness notes are recorded below.

## Info

### IN-01: Two example docs describe `api`/listener binding as unconditional, but `BindLifecycle` only runs when a controlled page exists

**File:** `docs/plugins/examples/xss-scanner.md:13`, `docs/plugins/examples/starter.md:26`
**Issue:** Both pages state that a successful `plugin load` "binds the `api` global, and attaches the CDP lifecycle listeners" (xss-scanner.md) / "bound the `api` global, and attached its lifecycle listeners" (starter.md) as if it always happens. In `actions/plugin.go:21-24`, `engine.BindLifecycle` is only invoked when `ctx.ControlledPage()` returns a non-nil page with no error. With no controlled page, the script loads but neither `api` nor the listeners are bound. `docs/plugins/cli-reference.md:15` correctly hedges this ("if a controlled page is available, calls `engine.BindLifecycle`"); the two example pages do not.
**Fix:** Mirror the cli-reference hedge, e.g. "...and, if a controlled page is available, binds the `api` global and attaches the CDP lifecycle listeners." This also better aligns with the scripts' own `typeof api !== "undefined"` guards, which exist precisely because `api` may be absent.

### IN-02: cli-reference lists "plugin engine not initialized" as a `plugin run` error path that is effectively unreachable via the daemon

**File:** `docs/plugins/cli-reference.md:75`
**Issue:** The doc lists "Uninitialized engine — if no plugin VM is available, `engine.RunFunc` returns `plugin engine not initialized`." This nil-VM guard in `engine.go:66-68` is sound and worth keeping, but `actions/plugin.go:51` reaches `RunFunc` via `ctx.GetPluginEngine()`, which lazily constructs and `Init()`s the engine (`types/context.go:196-199`). So through the real `plugin run` CLI path the VM is always non-nil and this error cannot fire. The doc presents it as a normal runtime error condition alongside genuinely reachable ones.
**Fix:** Either drop this bullet from the `plugin run` error list or annotate it as a defensive guard that is not reachable through the daemon dispatch (the engine is always initialized by `GetPluginEngine`). The `RunFunc` doc comment in `engine.go:57` could likewise note this is a defensive path.

### IN-03: `RunFunc` stringifies non-string return values via `res.String()`, which yields `[object Object]` for raw objects

**File:** `internal/plugin/engine.go:89`
**Issue:** `return res.String(), nil` produces clean output only when the JS function returns a string (the example accessors all `JSON.stringify(...)`). A function that returns a raw object/array without stringifying will print `[object Object]` rather than JSON. This is correct given the documented contract (accessors JSON.stringify their output, per `engine.go:62-64`) and is not a defect, but it is an undocumented sharp edge for plugin authors who write a getter that returns an object directly.
**Fix:** Optional. Either keep as-is (contract is documented) or note in `docs/plugins/cli-reference.md` under `plugin run` output that accessors should return a string (typically via `JSON.stringify`), since the engine does not JSON-encode Go-side.

---

_Reviewed: 2026-06-23_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
