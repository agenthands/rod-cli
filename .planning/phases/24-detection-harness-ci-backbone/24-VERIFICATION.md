---
phase: 24-detection-harness-ci-backbone
verified: 2026-06-24T00:00:00Z
status: passed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 24: Detection Harness & CI Backbone Verification Report

**Phase Goal:** A deterministic, offline detection harness drives the real `rod-cli` binary and asserts each table-stakes stealth signal by reading it back from the live page, running on every push as the repo's first test CI job — baselined against the current binary so existing leaks (e.g. unwired WebRTC) are exposed up front rather than hidden.
**Verified:** 2026-06-24
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | HARNESS-01: offline `127.0.0.1:0` fixture serves a self-authored `go:embed` detection page | ✓ VERIFIED | `internal/detect/server.go:43` binds `net.Listen("tcp", "127.0.0.1:0")`; `embed.go:5,8` `//go:embed detect.html`/`detect.js`; self-authored page, no vendored sannysoft; `go test ./internal/detect/...` → `ok` |
| 2 | HARNESS-02: e2e test drives real `../rod-cli` and reads each signal back from the LIVE page via eval | ✓ VERIFIED | `tests/detection_test.go` reads `String(window.__detect.<signal>)` via `runCli("eval", ...)` (`evalDetect`, L46); live run PASS (8/8 subtests, 4.87s) against the rebuilt binary |
| 3 | HARNESS-02: KNOWN-RED signals (WebRTC, CH-121) asserted, never `t.Skip`-ped | ✓ VERIFIED | `webrtc_ice_known_red` + `client_hints_known_red` subtests assert observability and log baseline truth; grep for `t.Skip` finds only a comment, no skip directive; live: webrtcIce=`""`, Sec-Ch-Ua observed |
| 4 | HARNESS-03: `.github/workflows/test.yml` runs `go test ./...` on push+PR to main, go 1.25.x, builds binary + installs Chromium, offline | ✓ VERIFIED | Valid YAML; `on.push.branches=[main]`, `on.pull_request.branches=[main]`; `go-version: '1.25.x'` (matches go.mod 1.25.1, not 1.23); steps build `rod-cli`, `./rod-cli install`, `go test ./... -count=1`; no network step |
| 5 | VALIDATE-03: `createPage()` surfaces fingerprint + `Apply()` errors to stderr; no swallow; no daemon hard-fail; stdout clean | ✓ VERIFIED | `types/context.go:289,295` `fmt.Fprintf(os.Stderr, "warning: ...")`; no `_ = em.Apply()` / `if err == nil` swallow; warnings go to stderr (stdout clean); errors logged not returned (no hard-fail); harness asserts success-path emits no `warning:` |
| 6 | Cleanup + CDP note: strays removed from git, `.gitignore` extended, `tests/rod/` untracked; `docs/cdp-footprint.md` documents the YES/NO ceiling | ✓ VERIFIED | `git ls-files tests/rod` → empty; no `.orig`/root-stray tracked; working tree clean; removal commit `38eda6a`; `.gitignore` lists all strays + `tests/rod/`,`tests/log/`,`*.orig`; `docs/cdp-footprint.md` documents probe-don't-fix ceiling, defers to v2 CDP-01 |

**Score:** 6/6 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/detect/server.go` | loopback fixture server | ✓ VERIFIED | binds 127.0.0.1:0, serves embedded assets, exec offline |
| `internal/detect/embed.go` | go:embed bundling | ✓ VERIFIED | embeds detect.html + detect.js |
| `internal/detect/detect.html` | self-authored shell | ✓ VERIFIED | loads /detect.js, no external fetch |
| `internal/detect/detect.js` | table-stakes probes → window.__detect | ✓ VERIFIED | webdriver, plugins, UA, WebGL, permissions, languages, screen, window.chrome, timezone + WebRTC/CDP info probes |
| `internal/detect/server_test.go` | offline unit test | ✓ VERIFIED | asserts loopback URL + embed wiring; passes |
| `tests/detection_test.go` | e2e live harness | ✓ VERIFIED | reads via window.__detect eval; live PASS |
| `.github/workflows/test.yml` | first test CI job | ✓ VERIFIED | push+PR/main, go 1.25.x, offline |
| `types/context.go` (createPage) | VALIDATE-03 stderr surfacing | ✓ VERIFIED | stderr warnings, no swallow, no hard-fail |
| `docs/cdp-footprint.md` | CDP ceiling note | ✓ VERIFIED | informational YES/NO, deferred to v2 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `internal/detect` unit tests pass | `go test ./internal/detect/...` | `ok` | ✓ PASS |
| Live harness drives real binary, reads signals from live page | `go test ./tests/ -run TestDetectionHarness -count=1 -v` | PASS, 8/8 subtests, 4.87s | ✓ PASS |
| Build gate | `go build ./...` | exit 0 | ✓ PASS |
| KNOWN-RED baselines observable | (harness log) | webrtcIce=`""`, Sec-Ch-Ua observed | ✓ PASS |
| No `t.Skip` on KNOWN-RED signals | `grep -n 't.Skip' tests/detection_test.go` | comment-only, no directive | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| HARNESS-01 | 24-01 | offline go:embed fixture on 127.0.0.1:0 | ✓ SATISFIED | server.go + embed.go + detect.{html,js}; tests pass |
| HARNESS-02 | 24-03 | e2e reads each signal back from live page | ✓ SATISFIED | detection_test.go live PASS via window.__detect eval |
| HARNESS-03 | 24-04 | test CI on every push, baselined | ✓ SATISFIED | test.yml push+PR/main, KNOWN-RED markers stay visible |
| VALIDATE-03 | 24-02 | swallowed evasion errors fail loudly | ✓ SATISFIED | context.go stderr warnings, no swallow/hard-fail |

### Anti-Patterns Found

None. No unreferenced `TBD`/`FIXME`/`XXX` debt markers in any phase-modified file. No stub/empty-implementation patterns; no `_ = em.Apply()` swallow remaining.

### Human Verification Required

None. All criteria verified programmatically by reading shipped code and running the offline unit tests, the live e2e harness against the real binary, and the build gate. KNOWN-RED WebRTC/CH-121 signals are intentional baseline markers for Phases 26/27 — their current red truth is correct-by-design for this phase and is asserted (not skipped), so they are NOT gaps.

### Gaps Summary

No gaps. All six success criteria are met in the shipped code and verified by execution:
- The offline fixture binds loopback and serves a self-authored `go:embed` page (no vendored sannysoft).
- The e2e harness drives the real `../rod-cli` binary and reads every table-stakes signal back from the live page via `window.__detect.<signal>` — never from a Go field.
- KNOWN-RED WebRTC and Client-Hints signals are asserted at current truth and logged, never `t.Skip`-ped, so the baseline is visible.
- The CI workflow is valid YAML running `go test ./...` on push+PR to main at go 1.25.x (matching go.mod, not 1.23), building the binary and installing Chromium with no network egress.
- VALIDATE-03 surfaces fingerprint and `Apply()` errors to stderr with no swallow and no daemon hard-fail; stdout stays clean.
- Stray artifacts are removed from git and gitignored; `tests/rod/` is untracked; the CDP-footprint note documents the YES/NO ceiling and defers the fix to v2 (CDP-01).

---

_Verified: 2026-06-24_
_Verifier: Claude (gsd-verifier)_
