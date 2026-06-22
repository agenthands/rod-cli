# Phase 21: Reference Documentation - Pattern Map

**Mapped:** 2026-06-22
**Files analyzed:** 5 (3 new docs, 1 modify source, 1 new/modify test)
**Analogs found:** 5 / 5

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `docs/plugins/lifecycle-hooks.md` (NEW) | documentation | reference | `skills/rod-cli/references/storage-state.md` | role-match |
| `docs/plugins/state-api.md` (NEW) | documentation | reference | `skills/rod-cli/references/storage-state.md` | role-match |
| `docs/plugins/cli-reference.md` (NEW) | documentation | reference | `skills/rod-cli/references/storage-state.md` + `USAGE.md` | role-match |
| `internal/plugin/api.go` (MODIFY) | service/accessor | transform (page read) | same-file `GetCookies()` / `GetSnapshot()` | exact |
| `internal/plugin/integ_test.go` (MODIFY, or new `api_test.go`) | test | request-response | same-file `TestPluginAPI_GetCookies_RealPage` | exact |

Recommendation: add the new test directly into the existing `internal/plugin/integ_test.go` rather than a new `api_test.go`. The shared harness (`TestMain`, `integBrowser`, `integServerURL`, `openIntegPage`) lives there and is package-private; a separate file in the same package works too but adds no value and risks duplicate `TestMain`. Use the existing file.

## Pattern Assignments

### `internal/plugin/api.go` — `GetLocalStorage()` (service/accessor, transform)

**Analog:** same file, `GetCookies()` (lines 17-27) and `GetSnapshot()` (lines 29-35).

**Existing imports** (lines 1-5) — only `rod` is imported; no new import needed for `page.Eval`:
```go
package plugin

import (
	"github.com/go-rod/rod"
)
```

**Accessor shape to mirror** (`GetCookies`, lines 17-27) — nil-page guard returning the zero value, then `(value, error)`:
```go
func (a *PluginAPI) GetCookies() (interface{}, error) {
	if a.page == nil {
		return nil, nil
	}
	cookies, err := a.page.Browser().GetCookies()
	if err != nil {
		return nil, err
	}
	return cookies, nil
}
```

**page.Eval pattern to copy** (from `internal/plugin/scanner/scanner.go` `CheckDOMForPayload`, lines 180-205) — `page.Eval` takes a JS arrow function string, returns `result, err`, and `result.Value` is a goja value queried with `.Get(...)`:
```go
result, err := page.Eval(`(payload) => { ... return { found: true, evidence: ... }; }`, payload)
if err != nil { ... }
obj := result.Value
found := obj.Get("found").Bool()
```

**Implementation guidance for `GetLocalStorage()`** (match `GetCookies` return style — `(interface{}, error)`, nil-guard, surface eval error):
- Guard: `if a.page == nil { return nil, nil }` (mirror `GetCookies`/`GetSnapshot`).
- Use `page.Eval` with a no-arg arrow fn that iterates `window.localStorage` keys into a plain JS object, e.g. `() => { var o = {}; for (var i=0;i<localStorage.length;i++){var k=localStorage.key(i); o[k]=localStorage.getItem(k);} return o; }`.
- On `err != nil` return `nil, err`.
- Return `result.Value.Export(), nil` so goja hands back a `map[string]interface{}` — most consistent with `GetCookies()` returning a structured `interface{}` and easiest to consume from JS (per Claude's Discretion in CONTEXT, map preferred over JSON string).
- Do NOT change `GetCookies`/`GetSnapshot` signatures (CONTEXT specifics constraint).

### `internal/plugin/integ_test.go` — `TestPluginAPI_GetLocalStorage_RealPage` (test, request-response)

**Analog:** same file, `TestPluginAPI_GetCookies_RealPage` (lines 110-122) and `TestPluginAPI_GetSnapshot_RealPage` (lines 124-139).

**Shared harness already present** — reuse, do not recreate:
- `TestMain` (lines 19-39): starts an in-process HTTP server (`integServerURL`) and a headless browser (`integBrowser`). The server already sets a cookie; for a localStorage test the page must seed localStorage (either extend the served HTML with an inline `<script>localStorage.setItem('k','v')</script>`, or seed via `page.Eval` after navigation in the test).
- `openIntegPage(t, url)` (lines 41-51): opens a page, navigates, waits for load.

**Test pattern to copy** (`TestPluginAPI_GetCookies_RealPage`, lines 110-122):
```go
func TestPluginAPI_GetCookies_RealPage(t *testing.T) {
	page := openIntegPage(t, integServerURL)
	defer page.MustClose()

	api := NewPluginAPI(page)
	cookies, err := api.GetCookies()
	if err != nil {
		t.Fatalf("GetCookies failed: %v", err)
	}
	if cookies == nil {
		t.Fatal("expected non-nil cookies")
	}
}
```

**New test guidance:**
- Open page via `openIntegPage(t, integServerURL)`; `defer page.MustClose()`.
- Seed localStorage (the default served HTML has none): `page.Eval(` + "`() => localStorage.setItem('theme','dark')`" + `)` before reading.
- `api := NewPluginAPI(page)`; `data, err := api.GetLocalStorage()`; `t.Fatalf` on error; assert non-nil and (optionally) that the seeded key is present.
- Also consider mirroring the nil-page no-op: a `GetLocalStorage()` on a `NewPluginAPI(nil)` returns `nil, nil` (cheap unit test, no browser needed) — matches the established "no-op to nil when page == nil" pattern.

### `docs/plugins/*.md` (documentation, reference) — shared doc template

**Analog:** `skills/rod-cli/references/storage-state.md` (whole file) for tone/structure; `USAGE.md` for CLI-command prose.

**Markdown conventions to mirror** (from `storage-state.md`):
- H1 title, short intro paragraph, H2 section per topic.
- Fenced code blocks tagged by language: ```` ```bash ```` for CLI invocations, ```` ```go ```` for Go signatures, ```` ```javascript ```` for plugin handler snippets.
- Concise, imperative prose; one worked snippet per concept.

**Per-page content sources (every documented surface must cite its backing source file — CONTEXT requirement):**

`lifecycle-hooks.md` — from `internal/plugin/lifecycle.go`:
- Mapping table JS handler → CDP proto type → key fields, sourced from `BindLifecycle` (lines 25-33):
  - `onRequest` → `proto.NetworkRequestWillBeSent` → `event.Request.URL`, `.Method`, headers
  - `onResponse` → `proto.NetworkResponseReceived` → `event.Response.URL`, `.Status`, headers
  - `onLoad` → `proto.PageLoadEventFired` → timestamp
  - `onDOMNodeInserted` → `proto.DOMChildNodeInserted` → parent/inserted node id
- Note JS names are lowercase (handler names in `invokeJSFunc` calls) vs. uppercase Go `LifecycleEmitter` interface (lines 12-17). Document the lowercase JS names.
- Note handlers are optional: `invokeJSFunc` (lines 36-49) silently no-ops if `e.vm.Get(funcName) == nil`.
- Worked snippets: reuse `plugins/examples/xss_scanner.js` `onRequest` (lines 16-23), `onResponse` (lines 26-40), `onLoad` (lines 44-48).

`state-api.md` — from `internal/plugin/api.go`:
- Document `api.GetSnapshot()` → `(string HTML, error)` (lines 29-35) and `api.GetCookies()` → CDP cookie array (lines 17-27), plus the new `api.GetLocalStorage()`.
- Worked snippet: `api.GetSnapshot()` call inside `onLoad` in `xss_scanner.js` (line 48).
- Note the `api` global is only bound after `BindLifecycle` runs (`lifecycle.go` line 22), i.e. after `plugin load`.
- Cross-link network context to `lifecycle-hooks.md` (network reached via hook payloads, not an `api` method — CONTEXT decision).

`cli-reference.md` — from `actions/plugin.go` and `cmd.go`:
- `plugin load <path>` — `cmd.go` lines 402-411 ("plugin path is required" client guard) + `actions.PluginLoad` lines 11-28 (success string `"Plugin loaded successfully from %s"`, daemon open-file error). Non-zero exit on failure.
- `plugin list` — `actions.PluginList` lines 31-42 → `"No active plugins loaded."` or JSON array.
- `plugin run <name>` — `actions.PluginRun` lines 45-49 → currently a STUB returning `"Triggered plugin %s"`; document honestly as a known limitation (registry needed, see source comment lines 46-47). Hooks firing after `plugin load` is the real execution path.
- Note thin-client dispatch: `cmd.go` sends `daemon.Request{Command: "plugin-load"|"plugin-list"|"plugin-run"}` (lines 410, 418, 425).

## Shared Patterns

### Accessor / nil-guard (Go)
**Source:** `internal/plugin/api.go` lines 17-35
**Apply to:** `GetLocalStorage()` implementation
```go
if a.page == nil {
	return nil, nil
}
// ... read page state ...
if err != nil { return nil, err }
return value, nil
```

### page.Eval JS-read (Go)
**Source:** `internal/plugin/scanner/scanner.go` lines 180-205
**Apply to:** `GetLocalStorage()` reading `window.localStorage`
```go
result, err := page.Eval(`() => { /* JS */ return obj; }`)
// result.Value is a goja value; .Export() yields map[string]interface{}
```

### Integration-test harness (Go)
**Source:** `internal/plugin/integ_test.go` lines 19-51 (`TestMain`, `openIntegPage`)
**Apply to:** new `TestPluginAPI_GetLocalStorage_RealPage` — reuse, do not duplicate `TestMain`.

### Markdown reference template
**Source:** `skills/rod-cli/references/storage-state.md`
**Apply to:** all three `docs/plugins/*.md`
- H1 → intro → H2 sections → language-tagged fenced blocks → source-citation footer.

## No Analog Found

None. Every file has a concrete in-repo analog.

## Metadata

**Analog search scope:** `internal/plugin/`, `internal/plugin/scanner/`, `actions/`, `cmd.go`, `plugins/examples/`, `skills/rod-cli/references/`, top-level `*.md`
**Files scanned:** ~10
**Pattern extraction date:** 2026-06-22
