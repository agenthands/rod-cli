# Phase 22: Example Plugins - Pattern Map

**Mapped:** 2026-06-23
**Files analyzed:** 13 (2 Go modify, 1 Go test, 6 JS, 4 docs — some grouped)
**Analogs found:** 13 / 13 (all have strong in-repo analogs)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/plugin/engine.go` (MODIFY: add `RunFunc`) | engine method | request-response | `internal/plugin/lifecycle.go` `invokeJSFunc` (lines 36-49) | exact (same lookup+assert) |
| `actions/plugin.go` (MODIFY: rewrite `PluginRun`) | action | request-response | `actions/plugin.go` `PluginLoad`/`PluginList` (lines 11-42) | exact (same file, same convention) |
| `internal/plugin/engine_test.go` (MODIFY/NEW tests) | test | request-response | `engine_test.go` `TestPluginEngine_InvokeJSFunc*` (lines 103-186) | exact |
| `plugins/examples/xss_scanner.js` (MODIFY) | plugin script | event-driven | itself (polish in place) | self |
| `plugins/examples/recipes/on_request.js` (NEW) | plugin script | event-driven | `xss_scanner.js` `onRequest` + `getRequestLog` (lines 5-24, 72-75) | exact pattern |
| `plugins/examples/recipes/on_response.js` (NEW) | plugin script | event-driven | `xss_scanner.js` `onResponse` (lines 27-41) + lifecycle-hooks.md onResponse | exact pattern |
| `plugins/examples/recipes/on_load.js` (NEW) | plugin script | event-driven | `xss_scanner.js` `onLoad` (lines 44-65) | exact pattern |
| `plugins/examples/recipes/on_dom_node_inserted.js` (NEW) | plugin script | event-driven | lifecycle-hooks.md onDOMNodeInserted snippet (lines 79-86) | doc-snippet match |
| `plugins/examples/starter.js` (NEW) | plugin template | event-driven | `xss_scanner.js` (full structure) | exact pattern |
| `docs/plugins/examples/xss-scanner.md` (NEW) | doc | n/a | `docs/plugins/lifecycle-hooks.md` (structure/tone) | exact tone |
| `docs/plugins/examples/recipes.md` (NEW) | doc | n/a | `docs/plugins/lifecycle-hooks.md` "Worked Snippets" (lines 26-86) | exact tone |
| `docs/plugins/examples/starter.md` (NEW) | doc | n/a | `docs/plugins/cli-reference.md` (structure) | exact tone |
| `docs/plugins/cli-reference.md` (MODIFY: `plugin run` section) | doc | n/a | itself, lines 53-73 (the stub section to replace) | self |

## Pattern Assignments

### `internal/plugin/engine.go` — add `func (e *PluginEngine) RunFunc(name string) (string, error)`

**Analog:** `internal/plugin/lifecycle.go` `invokeJSFunc` (lines 36-49). This is the EXACT pattern. `RunFunc` differs only in: (a) returns errors instead of silent no-op, (b) captures and stringifies the call's return value.

**Lookup + assert pattern to reuse** (`lifecycle.go` lines 36-48):
```go
func (e *PluginEngine) invokeJSFunc(funcName string, arg interface{}) {
	if e.vm == nil {
		return
	}
	fnObj := e.vm.Get(funcName)
	if fnObj == nil {
		return
	}
	if call, ok := goja.AssertFunction(fnObj); ok {
		_, _ = call(goja.Undefined(), e.vm.ToValue(arg))
	}
}
```

**Error-return convention to mirror** (`engine.go` `LoadScript` lines 28-30, 44-46): use `fmt.Errorf` with a clear message; nil-VM guard returns `fmt.Errorf("plugin engine not initialized")` (reuse exact string for test parity).

**RunFunc shape (derived, not copied verbatim):**
```go
// RunFunc looks up the JS function named `name` in the VM, calls it with no
// args, and returns its result stringified. Errors if the VM is nil or the
// function is missing / not callable.
func (e *PluginEngine) RunFunc(name string) (string, error) {
	if e.vm == nil {
		return "", fmt.Errorf("plugin engine not initialized")
	}
	fnObj := e.vm.Get(name)
	if fnObj == nil {
		return "", fmt.Errorf("function %q not found", name)
	}
	call, ok := goja.AssertFunction(fnObj)
	if !ok {
		return "", fmt.Errorf("%q is not a callable function", name)
	}
	res, err := call(goja.Undefined())
	if err != nil {
		return "", fmt.Errorf("error calling %q: %w", name, err)
	}
	if res == nil || goja.IsUndefined(res) || goja.IsNull(res) {
		return "", nil
	}
	return res.String(), nil
}
```
Note (Claude's Discretion, CONTEXT line 50): the example accessors (`getFindings`, `getRequestLog`) already `JSON.stringify(...)` internally and return a JS string, so `res.String()` yields clean JSON at the CLI — no extra Go-side JSON marshaling needed. Keep `res.String()`.

---

### `actions/plugin.go` — rewrite `PluginRun(ctx, name)`

**Analog:** the surrounding functions in the SAME file (`PluginLoad` lines 11-28, `PluginList` lines 31-42). Match their convention exactly: get engine via `ctx.GetPluginEngine()`, validate args, return `(string, error)`, propagate engine errors with `return "", err`.

**Engine-access + return convention** (`PluginLoad` lines 16-19):
```go
engine := ctx.GetPluginEngine()
if err := engine.LoadScript(path); err != nil {
	return "", err
}
```

**Arg-guard convention** (`PluginLoad` lines 12-14):
```go
if path == "" {
	return "", fmt.Errorf("plugin path is required")
}
```

**Stub to replace** (current lines 44-49):
```go
func PluginRun(ctx *types.Context, name string) (string, error) {
	// Note: Fully running named plugins might require a registry.
	return fmt.Sprintf("Triggered plugin %s", name), nil
}
```

**Rewritten shape:**
```go
func PluginRun(ctx *types.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("plugin function name is required")
	}
	return ctx.GetPluginEngine().RunFunc(name)
}
```

---

### `internal/plugin/engine_test.go` — RunFunc tests

**Analog:** `engine_test.go` `TestPluginEngine_InvokeJSFunc*` (lines 103-186). Same package (`package plugin` — white-box, can touch `engine.vm`). Reuse the `NewPluginEngine()` + `Init()` + `LoadScript`/`vm.RunString` harness.

**Harness pattern — defined function via temp file** (lines 103-117):
```go
engine := NewPluginEngine()
engine.Init()
tmpDir := t.TempDir()
jsFile := filepath.Join(tmpDir, "hooks.js")
jsCode := `var lastEvent = null; function onRequest(ev) { lastEvent = ev; }`
os.WriteFile(jsFile, []byte(jsCode), 0644)
engine.LoadScript(jsFile)
```

**Harness pattern — inline via RunString** (`integ_test.go` lines 85-88) — simpler for unit tests with no file:
```go
engine.vm.RunString(`function getFindings() { return JSON.stringify([1,2]); }`)
```

**Nil-VM negative pattern** (lines 171-175):
```go
func TestPluginEngine_InvokeJSFunc_NilVM(t *testing.T) {
	engine := NewPluginEngine()
	engine.invokeJSFunc("anyFunc", "data") // vm nil, no panic
}
```

**Required new tests (per CONTEXT line 28):**
1. `TestPluginEngine_RunFunc` — defined fn returns its value: load `function getFindings(){return JSON.stringify([{a:1}])}`, assert `RunFunc("getFindings")` returns `[{"a":1}]` and nil err.
2. `TestPluginEngine_RunFunc_Missing` — `RunFunc("nope")` returns non-nil err.
3. `TestPluginEngine_RunFunc_NilVM` — fresh engine (no Init), `RunFunc("x")` returns non-nil err, no panic.
4. (recommended) `TestPluginEngine_RunFunc_NotCallable` — `engine.vm.Set("notAFunc","str")` then expect err (mirrors lines 177-186).

---

### `plugins/examples/xss_scanner.js` — polish in place

**Analog:** itself. Keep the existing structure (CONTEXT line 37: "Do not rewrite from scratch"). Preserve all four pieces that make it demonstrable:
- module-level state arrays `var findings = []; var requestLog = [];` (lines 5-6)
- hook functions `onRequest`/`onResponse`/`onLoad` (lines 17-65)
- accessors `getFindings()`/`getRequestLog()` returning `JSON.stringify(...)` (lines 67-75) — these are what `plugin run getFindings` invokes via RunFunc.
- `typeof api !== "undefined"` guard before `api.GetSnapshot()` (line 46) — required per lifecycle-hooks.md line 24.

Polish = improve comments/structure, ensure clean `plugin load`, do not remove accessors. Add `onDOMNodeInserted` only if it strengthens the example (optional).

---

### Recipe scripts (`plugins/examples/recipes/*.js`)

**Analog:** `xss_scanner.js` hook + accessor pattern (the canonical shape every recipe mirrors):
```javascript
var requestLog = [];                      // module-level results array
function onRequest(event) {               // lowercase hook name (CONTEXT line 66)
  if (event && event.Request && event.Request.URL) {
    requestLog.push({ url: event.Request.URL, method: event.Request.Method || "GET" });
  }
}
function getRequestLog() { return JSON.stringify(requestLog); }  // accessor for `plugin run`
```

Per-file payload-field reference (from `lifecycle-hooks.md` table, lines 9-16):
- `on_request.js` → `event.Request.URL`, `event.Request.Method` (xss_scanner lines 17-24).
- `on_response.js` → `event.Response.URL`, `event.Response.Status` (lifecycle-hooks lines 48-56; xss_scanner lines 27-41).
- `on_load.js` → `api.GetSnapshot()` under `typeof api` guard (xss_scanner lines 44-65); accessor returns snapshot length or a flag.
- `on_dom_node_inserted.js` → `event.Node.NodeName`, `event.ParentNodeID` (lifecycle-hooks lines 79-86). **Note:** xss_scanner does not implement this hook, so the doc snippet is the only in-repo analog — copy it.

Each recipe: ONE hook + ONE results array + ONE getter (CONTEXT line 42). Must `plugin load` with no errors (no `console`, no `fs`, no `require` — sandbox forbids them, CONTEXT line 67).

---

### `plugins/examples/starter.js`

**Analog:** `xss_scanner.js` full structure, but with empty hook bodies. Must load unchanged with no errors (CONTEXT line 43). Defines all four hooks (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`) as empty/no-op stubs plus a results array + accessor:
```javascript
var results = [];
function onRequest(event) { /* TODO */ }
function onResponse(event) { /* TODO */ }
function onLoad(event) { /* TODO */ }
function onDOMNodeInserted(event) { /* TODO */ }
function getResults() { return JSON.stringify(results); }
```

---

### Example docs (`docs/plugins/examples/*.md`)

**Analog (tone/structure):** `docs/plugins/lifecycle-hooks.md` and `cli-reference.md`. Mirror these conventions exactly:
- Opening prose paragraph stating what the page documents (cli-reference.md line 3).
- Fenced `bash` command blocks for the CLI flow (cli-reference.md lines 9-11).
- Fenced `javascript` blocks for handler snippets (lifecycle-hooks.md lines 32-41).
- A `## See Also` section cross-linking sibling pages with relative `../` paths (lifecycle-hooks.md lines 88-91; cli-reference.md lines 79-82). **CONTEXT line 34 REQUIRES** every example doc cross-links `../lifecycle-hooks.md`, `../state-api.md`, `../cli-reference.md` for hooks/API/commands it uses — do NOT duplicate their content.
- A `## Source` footer pointing at the real source files (lifecycle-hooks.md lines 93-95).

**xss-scanner.md** must walk: `plugin load ./plugins/examples/xss_scanner.js` → drive a vulnerable page → `rod-cli plugin run getFindings` returns findings JSON (CONTEXT lines 76-78). This end-to-end "observe findings" path is the doc's reason to exist.

Note cross-link paths: example docs live one level deeper (`docs/plugins/examples/`), so links to refs are `../lifecycle-hooks.md` etc. (one `../` more than the existing sibling docs use).

---

### `docs/plugins/cli-reference.md` — update `plugin run` section (REQUIRED, CONTEXT line 29)

**Analog:** the `plugin load` section in the same file (lines 5-28) — the correct "functional command" template (Behavior / Output / Error conditions). Apply that shape to rewrite the `plugin run` stub section (lines 53-73).

Replace the "current stub (known limitation)" Behavior (lines 63-65) and the "Triggered plugin <name>" output (lines 67-71) with accurate docs: "invokes the JS function named `<name>` in the loaded plugin VM and returns its result" (e.g. `rod-cli plugin run getFindings`); document the missing-function error from `RunFunc`. Keep all other cli-reference content intact (no Phase 21 regression). Also update the Thin-Client/Source sections only where they assert `PluginRun` is a stub.

## Shared Patterns

### Go error-return convention
**Source:** `internal/plugin/engine.go` lines 28-30, 44-46; `actions/plugin.go` lines 12-19.
**Apply to:** `RunFunc`, rewritten `PluginRun`.
```go
if e.vm == nil {
	return "", fmt.Errorf("plugin engine not initialized")
}
// ...
return "", err   // propagate engine errors unwrapped from action layer
```

### goja function lookup + assert
**Source:** `internal/plugin/lifecycle.go` lines 41-48.
**Apply to:** `RunFunc`.
```go
fnObj := e.vm.Get(name)
if fnObj == nil { /* missing */ }
call, ok := goja.AssertFunction(fnObj)
if !ok { /* not callable */ }
res, err := call(goja.Undefined())
```

### JS plugin shape (state array + lowercase hook + JSON accessor)
**Source:** `plugins/examples/xss_scanner.js` lines 5-6, 17-24, 67-75.
**Apply to:** all recipe scripts + starter.
- Module-level `var xLog = [];`
- Lowercase hook `function onRequest(event) {...}` (engine looks up lowercase names; hooks are optional/no-op if absent — lifecycle-hooks.md line 20).
- Defensive payload guards (`if (event && event.Request && event.Request.URL)`).
- `typeof api !== "undefined"` guard before any `api.*` call.
- Accessor `function getX() { return JSON.stringify(x); }` so `plugin run getX` returns clean JSON.
- NO `console`/`fs`/`require` — goja sandbox has none (CONTEXT line 67).

### Doc page structure
**Source:** `docs/plugins/lifecycle-hooks.md` (intro → tables/snippets → `## See Also` → `## Source`).
**Apply to:** all `docs/plugins/examples/*.md`.
- Relative `../` cross-links to Phase 21 refs instead of duplication.
- `bash` blocks for CLI, `javascript` blocks for handlers.
- `## Source` footer linking real files with `../../` repo-relative paths.

## No Analog Found

None. Every file has a strong in-repo analog. The only near-gap is `onDOMNodeInserted` — no working implementation exists in `xss_scanner.js`, but `docs/plugins/lifecycle-hooks.md` lines 79-86 provides a tested-shape snippet to copy, so it is covered.

## Metadata

**Analog search scope:** `internal/plugin/`, `actions/`, `plugins/examples/`, `docs/plugins/`
**Files scanned:** engine.go, lifecycle.go, api.go, engine_test.go, integ_test.go, actions/plugin.go, xss_scanner.js, cli-reference.md, lifecycle-hooks.md, 22-CONTEXT.md
**Pattern extraction date:** 2026-06-23
