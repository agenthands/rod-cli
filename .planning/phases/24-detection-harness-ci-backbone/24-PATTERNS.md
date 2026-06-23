# Phase 24: Detection Harness & CI Backbone - Pattern Map

**Mapped:** 2026-06-24
**Files analyzed:** 6 (4 new, 2 modified) + 1 CI workflow + .gitignore
**Analogs found:** 6 / 6 (all in-repo)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/detect/server.go` (NEW) | testserver | request-response (HTTP fixture) | `internal/plugin/scanner/testserver/server.go` | exact (same `127.0.0.1:0` goroutine pattern) |
| `internal/detect/detect.html` + `detect.js` (NEW) | fixture asset | file-I/O (`go:embed`) | `types/js/js.go` + `types/js/snapshotter.js` | exact (embed mechanism) |
| `internal/detect/embed.go` (NEW) | config/embed glue | file-I/O (`go:embed`) | `types/js/js.go` | exact |
| `tests/detection_test.go` (NEW) | test (e2e) | request-response (drives live binary) | `tests/stealth_test.go` + `tests/cli_test.go` | exact (same `runCli`/`SetupTestServer` shape) |
| `types/context.go` `createPage()` (MODIFY) | service (page setup) | event-driven (evasion apply) | self (VALIDATE-03 edit site, lines 281-327) | in-place edit |
| `.github/workflows/test.yml` (NEW) | config (CI) | batch (CI job) | `.github/workflows/release.yml` | role-match (workflow shape) |
| `.gitignore` (MODIFY) | config | — | existing `.gitignore` | in-place edit |

## Pattern Assignments

### `internal/detect/server.go` (testserver, request-response)

**Analog:** `internal/plugin/scanner/testserver/server.go` (preferred over `tests/server.go` because it binds an explicit `127.0.0.1:0` listener with `Start()`/`Close()`/`URL()` — the ephemeral-port goroutine pattern the harness wants; `tests/server.go` uses `httptest.NewServer` which is also valid but less explicit about lifecycle).

**Struct + lifecycle pattern** (`testserver/server.go:23-60`):
```go
type VulnServer struct {
	listener net.Listener
	mux      *http.ServeMux
	mu       sync.Mutex
	stored   []StoredEntry
}

func New() (*VulnServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &VulnServer{listener: listener, mux: http.NewServeMux(), stored: []StoredEntry{}}
	s.registerRoutes()
	return s, nil
}

func (s *VulnServer) Start() { go http.Serve(s.listener, s.mux) }
func (s *VulnServer) Close() { s.listener.Close() }
func (s *VulnServer) URL() string { return fmt.Sprintf("http://%s", s.listener.Addr().String()) }
```
Mirror exactly: `detect.New() (*DetectServer, error)` → `Start()` (goroutine) → `URL()` → `Close()`. Register one route that serves the embedded `detect.html` (and `detect.js`), plus optionally a `POST /result` JSON scorecard sink (per STACK.md lines 53, 107).

**Route registration + HTML serving pattern** (`testserver/server.go:85-104`):
```go
func (s *VulnServer) registerRoutes() {
	s.mux.HandleFunc("/", s.handleIndex)
}
func (s *VulnServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>...`)
}
```
Difference: serve the **embedded** page (`w.Write([]byte(detectHTML))`) instead of an inline string literal, since the page is large and `go:embed`-bundled.

---

### `internal/detect/embed.go` + `detect.html` + `detect.js` (fixture asset, file-I/O)

**Analog:** `types/js/js.go` (the only `go:embed` site in the repo).

**Embed pattern** (`types/js/js.go:1-9`):
```go
package js

import _ "embed"

//go:embed snapshotter.js
var InjectedSnapShot string

//go:embed annotator.js
var AnnotatorUI string
```
Apply directly:
```go
package detect

import _ "embed"

//go:embed detect.html
var detectHTML string

//go:embed detect.js
var detectJS string
```
Note: the blank `import _ "embed"` form is the established convention here (string-typed embeds, not `embed.FS`). Keep `detect.html`/`detect.js` co-located in `internal/detect/` so the relative `//go:embed` path resolves.

**Probe set for `detect.js`** (from STACK.md:107 + CONTEXT.md:19) — assert each table-stakes signal and write verdict into a known global / DOM the e2e test reads via `eval`:
`navigator.webdriver`, `navigator.plugins.length`/`mimeTypes`, UA-without-`HeadlessChrome`, WebGL `UNMASKED_VENDOR_WEBGL`/`UNMASKED_RENDERER_WEBGL` (37445/37446), `Notification.permission` vs `navigator.permissions.query({name:'notifications'})` consistency, `navigator.languages`, screen dims, `window.chrome`/`chrome.runtime`, `Intl.DateTimeFormat().resolvedOptions().timeZone`, plus **informational** WebRTC ICE-leak + CDP-tell probes (non-blocking, CONTEXT.md:25).

---

### `tests/detection_test.go` (test e2e, request-response)

**Analog:** `tests/stealth_test.go` (closest — already asserts webdriver/plugins/UA stealth signals via `runCli` + `eval`). Converge with it, do NOT duplicate: extend coverage to WebGL/permissions/timezone/chrome/languages/screen rather than re-asserting webdriver+plugins+UA.

**Driver helper (reuse, do not redefine)** — `tests/cli_test.go:14-28`:
```go
func runCli(args ...string) (string, error) {
	absPath, _ := filepath.Abs("../rod-cli")
	args = append([]string{"--no-banner"}, args...)
	cmd := exec.Command(absPath, args...)
	cmd.Dir = ".."
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if len(args) > 0 && args[0] == "close" || (len(args) > 1 && args[1] == "close") {
		time.Sleep(1 * time.Second)
	}
	return out.String(), err
}
```
`runCli` lives in the same `tests` package — the new file calls it directly. NOTE: it captures **stderr into the same buffer as stdout** (`cmd.Stderr = &out`), so VALIDATE-03 stderr warnings ARE observable by the harness through `runCli` output. This is the mechanism by which the harness "makes evasion errors observable" (CONTEXT.md:26).

**Test body pattern** (`tests/stealth_test.go:8-56`):
```go
func TestStealthInitialization(t *testing.T) {
	ts := SetupTestServer()
	defer ts.Close()
	runCli("close") // clean daemon state before test

	out, err := runCli("goto", ts.URL+"/dummy")
	if err != nil { t.Fatalf("Failed to navigate: %v (output: %s)", err, out) }

	out, err = runCli("eval", "navigator.webdriver")
	if err != nil { t.Fatalf("Failed to eval navigator.webdriver: %v", err) }
	if !strings.Contains(out, "result: false") && !strings.Contains(out, "result: undefined") && !strings.Contains(out, "false") {
		t.Errorf("Stealth failure: ... got: %s", out)
	}
	runCli("close")
}
```
For the detection server, swap `SetupTestServer()` for `detect.New()` (call `ds.Start()`; `defer ds.Close()`) and `ts.URL` for `ds.URL()`. Drive: `runCli("close")` → `runCli("goto", ds.URL())` → `runCli("eval", "<probe>")` and assert. Per ARCHITECTURE.md:216-227 sketch.

**KNOWN-RED baseline convention** (CONTEXT.md:22, :61) — assert *current actual truth*; for signals known red (WebRTC leak, Client-Hints `121`) keep the assertion present and passing-against-baseline with a marker comment, do NOT `t.Skip`:
```go
// KNOWN-RED (Phase 27 HARDEN-01): WebRTC ICE candidate leaks local IP.
// Asserting current truth so CI stays green on baseline; flips to required-green when fixed.
if !strings.Contains(out, "<current-leaky-value>") { t.Errorf(...) }
```

**JSON scorecard decode (optional path)** — if the page POSTs a scorecard (STACK.md:53), the decode side mirrors `tests/network_evasion_test.go:4-9` which already uses `encoding/json` + `httptest` in this package.

---

### `types/context.go` `createPage()` — VALIDATE-03 edit (service, event-driven)

**Edit site** (`types/context.go:281-327`), specifically lines 286-294 where errors are swallowed today:
```go
em := stealth.NewEvasionManager(page)
fg := rodfingerprint.NewFingerprintGenerator(rodfingerprint.FPWithBrowserNames("chrome"))
fp, err := fg.Generate()
if err == nil && fp != nil {        // <-- silent: err != nil path is a no-op
	ctx.fingerprint = fp
	em.SetFingerprint(fp)
}
_ = em.Apply()                       // <-- swallowed error (the named `_ = em.Apply()`)
```

**VALIDATE-03 change:** make both swallowed errors observable on **stderr** (not stdout — CONTEXT.md:60, must not pollute piped/`--raw` results), without hard-failing the daemon (CONTEXT.md:26):
```go
fp, err := fg.Generate()
if err != nil {
	fmt.Fprintf(os.Stderr, "warning: fingerprint generation failed: %v\n", err)
} else if fp != nil {
	ctx.fingerprint = fp
	em.SetFingerprint(fp)
}
if err := em.Apply(); err != nil {
	fmt.Fprintf(os.Stderr, "warning: evasion Apply failed: %v\n", err)
}
```

**Established conventions to honor:**
- `fmt` and `os` are **already imported** in `types/context.go` (lines 5-6) — no new import needed for the `fmt.Fprintf(os.Stderr, ...)` form.
- The repo has **no existing `log.*` / structured-logger precedent** (grep found zero `log.Printf`/`os.Stderr` writes in non-test source) — `fmt.Fprintf(os.Stderr, "warning: ...")` is the lightest convention-consistent choice and matches the "warnings go to stderr" decision. The package error-wrap convention is `github.com/pkg/errors` (`errors.Wrap`, used at lines 275/320/324) — use it only if propagating, not for these log-and-continue warnings.
- Keep wording short/token-efficient (CONTEXT.md:60 / MEMORY: output noise).

---

### `.github/workflows/test.yml` (config, CI/batch)

**Analog:** `.github/workflows/release.yml` (only existing workflow; mirror its checkout + setup-go shape).

**Shape to mirror** (`release.yml:1-29`):
```yaml
name: Release
on:
  push:
    tags: ['v*']
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
```
**Adapt for the test job** (CONTEXT.md:27, ARCHITECTURE.md:229):
- `name: Test`
- Trigger: `on: { push: { branches: [main] }, pull_request: { branches: [main] } }` (push + PR to main).
- `go-version: '1.23'` to match `release.yml` (NOTE: STACK.md:23 flags `go.mod` declares `go 1.25.1`; planner should reconcile — prefer matching `go.mod`/build toolchain so `go test` compiles, likely bump to the toolchain version rather than `1.23`).
- Steps: checkout → setup-go → build the binary so `../rod-cli` exists for `runCli` (`go build -o rod-cli .`), then a Chromium/`rod-cli install` step (ARCHITECTURE.md:229/238 — cache Chromium), then `go test ./...`.
- Headless is the blocking gate; headful is opt-in/local only (CONTEXT.md:20) — do NOT add an xvfb headful row to the blocking job.

---

### `.gitignore` (config, in-place edit)

**Analog:** existing root `.gitignore` (already ignores `*.log`, `*.out`, `*.png`, etc.).

**Add** (CONTEXT.md:28) the stray artifacts to ignore and remove: `tests/rod`, `tests/coverage.out` (covered by `*.out` already — verify), `tests/log` (covered by `*.log`), `tests/*.orig`, and root `rod-cli`, `test_rod`, `state.json`, `init_output.json`, `fix_test.patch`. Several are already covered by existing globs — consolidate, don't duplicate (CONTEXT.md:33). The untracked artifacts visible in `git status` (`rod-cli`, `test_rod`, `state.json`, `init_output.json`, `fix_test.patch`) should be `git rm`/deleted as part of this.

## Shared Patterns

### Live-binary e2e driving (the v1.5 "validate the shipped binary" lesson)
**Source:** `tests/cli_test.go:14-28` (`runCli`)
**Apply to:** `tests/detection_test.go`
Drive the real `../rod-cli` daemon via `exec.Command` with `--no-banner`; assert through the public `goto`/`eval` CLI surface, never against godoll directly (ARCHITECTURE.md:327). `runCli("close")` before each test for deterministic daemon state (every existing test does this).

### Ephemeral-port offline fixture server
**Source:** `internal/plugin/scanner/testserver/server.go:32-60`
**Apply to:** `internal/detect/server.go`
`net.Listen("tcp","127.0.0.1:0")` → `go http.Serve(...)` → `URL()` from `listener.Addr()`. Deterministic, offline, no external network — the only thing safe to block CI on (STACK.md:108, ARCHITECTURE.md:212).

### `go:embed` string bundling (single-binary constraint)
**Source:** `types/js/js.go:1-9`
**Apply to:** `internal/detect/embed.go`
`import _ "embed"` + `//go:embed <file>` + package-level `string` var. Keeps the zero-Node/Python single-binary constraint (STACK.md:23, :52).

### Stealth-signal assertions (avoid duplication)
**Source:** `tests/stealth_test.go:22-53`, `tests/network_evasion_test.go:34-43`
**Apply to:** `tests/detection_test.go`
Already asserted today: `navigator.webdriver`, `navigator.plugins.length`, UA-no-`HeadlessChrome` (stealth_test); `User-Agent`/`Accept-Language`/`Sec-Ch-Ua` headers (network_evasion_test). New harness should EXTEND with the not-yet-covered signals (WebGL, permissions consistency, timezone, `window.chrome`, languages, screen, WebRTC/CDP info-probes) rather than re-assert these.

### Stderr warnings, not stdout (output discipline)
**Source:** convention from `cmd.go:687-688` (banner gated off stdout) + CONTEXT.md:60
**Apply to:** `types/context.go` VALIDATE-03 edit
Diagnostics/warnings go to `os.Stderr`; stdout stays clean for `--raw`/piped results. `runCli` folds stderr into its captured buffer, so the harness still observes them.

## No Analog Found

None. Every file maps to an in-repo analog. (CDP-footprint findings note — CONTEXT.md:32 — is prose/comment content with no code analog; planner places it per Claude's Discretion, e.g. a `docs/` note or harness comment block.)

## Metadata

**Analog search scope:** `tests/`, `internal/plugin/scanner/testserver/`, `types/js/`, `types/context.go`, `.github/workflows/`, root `.gitignore`
**Files scanned:** 8 source/config files read in full or targeted ranges
**Key findings:**
- `runCli` captures stderr into the same buffer as stdout (`tests/cli_test.go:18-22`) → VALIDATE-03 stderr warnings are automatically harness-observable.
- Repo has zero existing `log.*`/`os.Stderr` write precedent in non-test source → `fmt.Fprintf(os.Stderr, "warning: ...")` is the convention-setting choice for VALIDATE-03; `fmt`+`os` already imported in `context.go`.
- `types/js/js.go` uses the string-form `import _ "embed"` (not `embed.FS`) — match it.
- `.gitignore` already covers `*.out`/`*.log`/`*.png` — several listed stray artifacts are already ignored; consolidate.
**Pattern extraction date:** 2026-06-24
