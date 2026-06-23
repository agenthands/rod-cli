# Phase 21: Reference Documentation - Context

**Gathered:** 2026-06-22
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous)

<domain>
## Phase Boundary

Deliver three authoritative reference pages under `docs/plugins/`, each grounded in the shipped v1.4 implementation (`internal/plugin/`, `actions/plugin.go`, `cmd.go`):

1. `docs/plugins/lifecycle-hooks.md` — the four lifecycle hooks.
2. `docs/plugins/state-api.md` — the `api` global's state/context accessors.
3. `docs/plugins/cli-reference.md` — the `plugin load/list/run` CLI commands.

Plus one sanctioned small corrective engine fix: add `GetLocalStorage()` to `internal/plugin/api.go` so the state API page documents a real method (user decision during discuss).

Reference pages only — no tutorial (Phase 23) and no example plugins (Phase 22).
</domain>

<decisions>
## Implementation Decisions

### State / Context API coverage
- The `api` global currently exposes exactly two methods in `internal/plugin/api.go`: `GetSnapshot()` (returns full-page HTML via `page.HTML()`) and `GetCookies()` (returns the CDP cookie array via `page.Browser().GetCookies()`). Document both with their return shapes and a worked usage snippet (see `onLoad` in `xss_scanner.js`).
- **Small corrective fix (user-approved):** add `GetLocalStorage()` to `PluginAPI` so localStorage is a first-class, documentable accessor. Implement it the same way as the existing accessors — bind to the page, read `window.localStorage` via `page.Eval` (iterate keys into a JSON object/map), return `(interface{}/string, error)`, and no-op (nil) when `a.page == nil`, mirroring `GetSnapshot`/`GetCookies`. Add a unit/integration test alongside the existing `TestPluginAPI_*` tests. This is a small fix to surface existing browser state, not a new engine capability.
- **Network context** is reached through the lifecycle hook event payloads (`onRequest` → `event.Request.URL`/`.Method`; `onResponse` → `event.Response.URL`/`.Status`), NOT through an `api` method. The state-api page documents this explicitly and cross-links to `lifecycle-hooks.md`.
- No invented surfaces: every documented method must exist in source after the `GetLocalStorage()` fix lands.

### Lifecycle hooks reference
- Document all four hooks with a mapping table: **JS handler name → CDP event (proto type) → key payload fields**.
  - `onRequest` → `proto.NetworkRequestWillBeSent` → `event.Request.URL`, `event.Request.Method`, headers.
  - `onResponse` → `proto.NetworkResponseReceived` → `event.Response.URL`, `event.Response.Status`, headers.
  - `onLoad` → `proto.PageLoadEventFired` → timestamp.
  - `onDOMNodeInserted` → `proto.DOMChildNodeInserted` → parent node id / inserted node.
- Payload depth: document the commonly-used fields and link to the CDP proto type for the full shape (keeps pages token-efficient).
- Include one concise worked snippet per hook (reuse patterns from `xss_scanner.js`).
- Note that handlers are optional: the engine (`invokeJSFunc`) silently no-ops if the JS function is not defined, and the `api` global is only bound after `BindLifecycle` runs (on `plugin load`).

### CLI reference & page structure
- Document `plugin load <path>`, `plugin list`, `plugin run <name>` from `cmd.go` with arguments, behavior, output format, and error/exit conditions:
  - `plugin load` requires a path → client-side error "plugin path is required"; missing file surfaces the daemon error "failed to open script file …"; non-zero exit on failure.
  - `plugin list` → "No active plugins loaded." or a JSON array of loaded plugin paths.
  - `plugin run <name>` is currently a **stub** (`actions.PluginRun` returns "Triggered plugin <name>" without re-executing a registry-resolved plugin). Document this honestly, noting real per-name execution needs a registry (per the source comment). Hooks fire continuously after `plugin load`, which is the primary execution path today.
- All pages live under `docs/plugins/`, use a consistent template (overview → reference table → return/payload shapes → worked snippet → cross-links → source references), and are Markdown.
- Source grounding: every documented hook name, API method, and command cites the backing source file.

### Claude's Discretion
- Exact Markdown layout/headings within the agreed template, prose wording, and which CDP fields count as "commonly used."
- Exact `GetLocalStorage()` return representation (JSON string vs. map) — match whatever is most consistent with `GetCookies()`/`GetSnapshot()` and easiest to consume from goja JS.
</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/plugin/api.go` — `PluginAPI` with `GetSnapshot()` and `GetCookies()`; `NewPluginAPI(page)`. Target for the `GetLocalStorage()` fix.
- `internal/plugin/lifecycle.go` — `BindLifecycle` wires CDP events to JS handlers `onRequest`/`onResponse`/`onLoad`/`onDOMNodeInserted` and sets the `api` global; `invokeJSFunc` no-ops on missing handlers.
- `internal/plugin/engine.go` — `PluginEngine` (goja runtime), `Init()`, `LoadScript(path)` with its exact error strings.
- `actions/plugin.go` — `PluginLoad`/`PluginList`/`PluginRun` with output strings and the `plugin run` stub.
- `cmd.go` (~line 399) — `plugin` command + `load`/`list`/`run` subcommands, arg parsing, "plugin path is required" guard.
- `daemon/daemon.go` (~line 201) — dispatch of `plugin-load`/`plugin-list`/`plugin-run`.
- `plugins/examples/xss_scanner.js` — real hook usage and `api.GetSnapshot()` call for worked snippets.
- `internal/plugin/integ_test.go` — proves `api` is set on the VM after `BindLifecycle`, and `GetCookies`/`GetSnapshot` behavior on real pages (model for the new `GetLocalStorage` test).

### Established Patterns
- JS handler names are lowercase (`onRequest`); the Go `LifecycleEmitter` interface uses uppercase (`OnRequest`). Document the JS names since that is what plugin authors write.
- Accessors return `(value, error)` and no-op to nil/empty when `page == nil`.
- CLI commands are thin clients that send `daemon.Request{Command, Args}` to the daemon and print the response; errors yield a non-zero exit.

### Integration Points
- New docs live in a new `docs/` tree (none exists yet). README link to the index is Phase 23, not here.
- The `GetLocalStorage()` fix integrates with the existing `api` binding in `BindLifecycle` automatically (no new wiring).
</code_context>

<specifics>
## Specific Ideas

- Page filenames are fixed by the success criteria: `docs/plugins/lifecycle-hooks.md`, `docs/plugins/state-api.md`, `docs/plugins/cli-reference.md`.
- Keep payloads documented at "commonly-used fields + link to CDP proto type" depth per the user's accept-all on depth.
- The `GetLocalStorage()` addition must ship with a test and must not change the existing `api` method signatures.
</specifics>

<deferred>
## Deferred Ideas

- Docs index (`docs/plugins/README.md`) and README link-in → Phase 23.
- Example/recipe/starter plugins → Phase 22.
- A plugin registry to make `plugin run <name>` fully functional → out of scope for v1.5 (documented as a known limitation only).
</deferred>
