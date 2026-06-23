# Phase 22: Example Plugins - Context

**Gathered:** 2026-06-23
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous)

<domain>
## Phase Boundary

Ship runnable example plugins plus their docs, all under documented trees, exercising every lifecycle hook and the state API:

1. Flagship: polished `plugins/examples/xss_scanner.js` + worked doc `docs/plugins/examples/xss-scanner.md` (PEX-01).
2. Per-hook recipe plugins — one small script per lifecycle hook under `plugins/examples/recipes/` + a recipes doc (PEX-02).
3. Copyable starter/template plugin `plugins/examples/starter.js` with empty stubs for all four hooks + a results accessor; loads unchanged and runs without errors (PEX-03).

Plus a sanctioned small corrective engine fix (user decision): make `plugin run <name>` functional so example findings are observable from the CLI — required by the phase's "observe findings end-to-end" / "see a hook fire" success criteria.

Out of scope: the authoring tutorial and the `docs/plugins/` index/README (Phase 23).
</domain>

<decisions>
## Implementation Decisions

### Observability fix (user-approved small corrective fix)
- The shipped engine has no `console` binding and `actions.PluginRun` is a stub, so plugin-VM state (findings arrays) has no CLI read path. This blocks PEX-01 ("observe it collecting findings end-to-end") and PEX-02 ("run it to see that hook fire").
- **Fix:** add `func (e *PluginEngine) RunFunc(name string) (string, error)` to `internal/plugin/engine.go` — look up `name` in the VM via `vm.Get(name)`, assert it's a callable (goja.AssertFunction), call it with no args, and return the result stringified (JSON or `.String()`); return a clear error if the VM is nil or the function is missing/not callable. Then rewrite `actions.PluginRun(ctx, name)` to call `ctx.GetPluginEngine().RunFunc(name)` and return its output (instead of the stub string).
- This is the minimal change that makes ALL examples genuinely demonstrable: `rod-cli plugin run getFindings` returns the collected findings JSON. Same "small corrective fix" class as Phase 21's GetLocalStorage — surfaces existing in-VM state, does not add a new hook or a registry.
- Add/extend tests in `internal/plugin/` (engine_test.go / integ_test.go) covering RunFunc: defined function returns its value; missing function → error; nil VM → error.
- **Cross-phase doc update (REQUIRED):** Phase 21's `docs/plugins/cli-reference.md` documents `plugin run` as a stub. After this fix lands, update that page so `plugin run <name>` is documented accurately as "invokes the JS function named `<name>` in the loaded plugin and returns its result" (with the missing-function error). Keep all other cli-reference content intact. This keeps the reference truthful — not a Phase 21 regression.

### Examples tree layout
- Runnable scripts live under `plugins/examples/` (existing `xss_scanner.js` convention): flagship `xss_scanner.js`, per-hook recipes in `plugins/examples/recipes/` (e.g. `on_request.js`, `on_response.js`, `on_load.js`, `on_dom_node_inserted.js`), and `plugins/examples/starter.js`.
- Docs live under `docs/plugins/examples/`: `xss-scanner.md` (flagship worked example), a recipes doc (one section per hook, or one note per recipe), and a starter doc. Exact doc file split is Claude's discretion as long as every example is documented and discoverable.
- Every example doc cross-links the Phase 21 reference pages (`../lifecycle-hooks.md`, `../state-api.md`, `../cli-reference.md`) for the hooks/API/commands it uses, instead of duplicating them.

### XSS scanner polish
- Polish the EXISTING `plugins/examples/xss_scanner.js` (improve comments/structure, ensure it loads cleanly under `plugin load`, use the `api` global correctly, keep/expose `getFindings`/`getRequestLog` accessors so `plugin run getFindings` works). Do not rewrite from scratch.
- The JS plugin is the flagship plugin example. The Go `internal/plugin/scanner/scanner.go` stays internal — may be mentioned as the built-in scanner but is NOT the PEX-01 example (it is not a plugin).
- XSS payloads / exploit logic stay in the example script, never in the core binary (per REQUIREMENTS Out of Scope).

### Recipe & starter design
- One recipe script per hook; each defines just that hook plus a results array and a getter (e.g. `getRequestLog`) so the user can `plugin load` it, drive the browser, then `plugin run <getter>` to see the hook fired. The recipe note explains what to do and what to expect.
- Starter defines empty stubs for all four hooks (`onRequest`/`onResponse`/`onLoad`/`onDOMNodeInserted`) plus a results accessor; must `plugin load` unchanged with no errors.

### Verification expectation
- Every example script must actually load via `rod-cli plugin load <path>` without error (load-test each), and `go build ./...` + `go test ./internal/plugin/...` must pass after the engine fix.

### Claude's Discretion
- Exact doc file split under `docs/plugins/examples/` (single recipes.md vs per-hook), recipe script contents, and wording.
- RunFunc result stringification details (JSON vs raw string) — pick what reads best from the CLI and is consistent with existing action outputs.
</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `plugins/examples/xss_scanner.js` — existing flagship script (onRequest/onResponse/onLoad + getFindings/getRequestLog). Polish target.
- `internal/plugin/engine.go` — `PluginEngine.vm` (goja runtime), `invokeJSFunc` already does `vm.Get(name)` + `goja.AssertFunction` — the exact pattern `RunFunc` reuses.
- `actions/plugin.go` — `PluginRun(ctx, name)` stub to rewrite; `PluginLoad`/`PluginList` unchanged.
- `types/context.go` — `GetPluginEngine()` returns the live `*PluginEngine` bound to the session.
- `internal/plugin/api.go` — `api` global (GetSnapshot/GetCookies/GetLocalStorage) examples use.
- `internal/plugin/integ_test.go` / `engine_test.go` — test harness for the RunFunc tests.
- `docs/plugins/cli-reference.md`, `docs/plugins/lifecycle-hooks.md`, `docs/plugins/state-api.md` — Phase 21 refs to cross-link; cli-reference.md must be updated for the now-functional `plugin run`.

### Established Patterns
- JS handler names are lowercase; hooks no-op if undefined; `api` bound on `plugin load`.
- Plugins are user-supplied scripts in goja; no fs/network/console in the sandbox.

### Integration Points
- `RunFunc` plugs into the existing `actions.PluginRun` → daemon `plugin-run` dispatch → `cmd.go` `plugin run <name>` (no CLI signature change needed; the command already passes `name`).
</code_context>

<specifics>
## Specific Ideas

- `rod-cli plugin run getFindings` (and `getRequestLog`) is the canonical "see results" UX the docs should show.
- Recipe filenames mirror hook names for discoverability.
- The xss-scanner doc must let a reader load the plugin and observe findings end-to-end (load → drive a vulnerable test page → `plugin run getFindings`).
</specifics>

<deferred>
## Deferred Ideas

- Authoring tutorial (`docs/plugins/authoring.md`) and docs index (`docs/plugins/README.md`) + README link → Phase 23.
- A real named-plugin registry (multiple plugins addressable by name) → out of scope; `RunFunc` operates on the single loaded plugin VM.
</deferred>
