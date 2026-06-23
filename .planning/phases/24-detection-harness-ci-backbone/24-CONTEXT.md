# Phase 24: Detection Harness & CI Backbone - Context

**Gathered:** 2026-06-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver a deterministic, **offline** browser-detection test harness that drives the real `rod-cli` binary, asserts each table-stakes stealth signal by reading it back from the **live page** (never from a Go config field), and runs in CI as the repo's first test job — baselined against the *current* binary so existing leaks (unwired WebRTC, hardcoded Client-Hints) surface as documented known-red rather than silent passes. Also lands VALIDATE-03: evasion/fingerprint errors that are currently swallowed (`_ = em.Apply()`) become observable.

Covers: HARNESS-01, HARNESS-02, HARNESS-03, VALIDATE-03. This phase is the regression backbone every later v1.6 phase asserts against — it does NOT add new evasion features (those are Phases 25–28).

</domain>

<decisions>
## Implementation Decisions

### Detection Harness Composition
- **Detection page is self-authored**, an offline sannysoft-style page covering the table-stakes signals (`navigator.webdriver`, plugins/mimeTypes, UA-without-`HeadlessChrome`, WebGL vendor/renderer, `navigator.permissions`, `navigator.languages`, screen dims, `window.chrome`/`chrome.runtime`, timezone). Do NOT vendor bot.sannysoft (no clean license). A curated CreepJS-style subset is acceptable but CreepJS-live stays a Phase 29 best-effort target, not a CI dependency.
- **Test matrix:** headless is the **blocking CI gate**; headful is a **local/opt-in matrix row** (headful in CI needs xvfb and is slow/flaky). The harness code supports both; CI runs headless.
- **Location:** new files in the existing `tests/` package, mirroring `tests/cli_test.go` (drives `../rod-cli` via `exec.Command`) and `tests/server.go`'s `SetupTestServer()` pattern; bundle the detection page via `go:embed`.
- **Baseline of known leaks:** assert the **current actual truth** of each signal, marking signals known to be red with `// KNOWN-RED (Phase 26/27: <REQ>)` comments. CI stays green against the baseline; each marked assertion flips to required-green when its fix lands in the later phase. Do NOT `t.Skip` known-red signals (they must stay visible) and do NOT fail CI on the documented baseline.

### CDP Spike, Loud-Failure Scope & CI Trigger
- **CDP-footprint spike:** include **informational, non-blocking** CDP-tell probes in the harness plus a short findings note documenting the YES/NO ceiling (can the `Runtime.enable`/CDP footprint be meaningfully hidden given rod-cli relies on Runtime/Network events?). Do NOT attempt to *fix* CDP detectability this phase.
- **VALIDATE-03 ("fail loudly"):** surface evasion-manager `Apply()` and fingerprint-generation errors as a **stderr log/warning** and make them observable by the harness. Do **not** hard-fail the daemon on evasion error (that would break existing flows) — the requirement is "no silent no-op," not "abort."
- **CI trigger:** run the test workflow on **push + PR to `main`**.
- **Stray artifacts:** gitignore and remove leftover test/build artifacts as part of standing up clean CI — `tests/rod`, `tests/coverage.out`, `tests/log`, `tests/*.orig`, and root-level `rod-cli`, `test_rod`, `state.json`, `init_output.json`, `fix_test.patch`.

### Claude's Discretion
- Exact file split within `tests/` (e.g. `detection_harness.go` + `detection_test.go`), embed layout, and helper naming.
- Precise wording of the CDP findings note and where it lives (e.g. a `docs/` note or a comment block + the harness output).
- Exact `.gitignore` entries and whether to consolidate with any existing ignore rules.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `tests/cli_test.go` — `runCli(args...)` helper runs the compiled `../rod-cli` binary with `--no-banner`, captures stdout/stderr. The harness e2e test should reuse this exact pattern to prove the *shipped binary*.
- `tests/server.go` — `SetupTestServer()` returns a local `httptest`-style server; mirror it (or the `internal/plugin/scanner/testserver` `127.0.0.1:0` goroutine pattern) for the detection fixture server.
- `tests/stealth_test.go`, `tests/network_evasion_test.go` — existing stealth assertions to extend/converge with, not duplicate.
- `types/js/` already uses `//go:embed` — same mechanism for the detection page.

### Established Patterns
- E2E tests drive the real daemon via `exec.Command("../rod-cli")` and assert on stdout (`eval`/`snapshot` commands), proving live behavior — the v1.5/retrospective "validate the live binary, not source" lesson.
- Evasion is set up in `types/context.go` `createPage()` (≈ lines 281–327): `EvasionManager.Apply()`, fingerprint generation — currently error-swallowed (`_ = em.Apply()`, `if err == nil`). VALIDATE-03 touches this.

### Integration Points
- New `.github/workflows/test.yml` (none exists today — only `release.yml`, `release-binary.yml`).
- Detection assertions read back via the `eval` command / `page.Eval` against the live page.
- VALIDATE-03 edit localized to `types/context.go` evasion path (log instead of discard errors).

</code_context>

<specifics>
## Specific Ideas

- `--raw`/`--no-banner` output discipline must hold — harness output and any new logging stay token-efficient; evasion warnings go to stderr, not stdout, so they don't pollute piped results.
- The baseline's first CI run is expected to show WebRTC + Client-Hints `121` as KNOWN-RED; that is the point — it proves the harness sees real leaks before Phases 26/27 fix them.

</specifics>

<deferred>
## Deferred Ideas

- Fixing WebRTC leak / canvas-noise stub → Phase 27 (HARDEN-01/02).
- Client-Hints `121` derivation fix → Phase 26 (FINGERPRINT-03).
- Reducing the CDP footprint itself (vs merely probing it) → v2 (CDP-01); this phase only documents the ceiling.
- Live Cloudflare/DataDome/CreepJS checks → Phase 29 (LIVEWAF-01).

</deferred>
