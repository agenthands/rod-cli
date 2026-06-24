---
phase: 26-configurable-fingerprint-consistency-validator
plan: 04
subsystem: stealth-validation
tags: [stealth, fingerprint, validation, cli, detect, probe, go-embed]

# Dependency graph
requires:
  - phase: 26-01
    provides: "stealth-check [url] command surface registered in cmd.go (forwards url/raw/json in Args via runClientCommand)"
  - phase: 24
    provides: "internal/detect/detect.js window.__detect probe + signal semantics; evalDetect/waitForDetectReady harness cadence"
provides:
  - "internal/detect/probe.js — shared canonical table-stakes probe (one source for command + harness) embedded as detect.Probe"
  - "actions.StealthCheck — injects the probe, reads 11 signals from the LIVE page, applies VALIDATE-01 thresholds, formats raw/json/human"
  - "daemon stealth-check dispatch case delegating to actions.StealthCheck"
  - "cmd.go --json passthrough (isJSONValue) so structured daemon results are not double-wrapped"
affects: [26-05, phase-27, phase-29]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared //go:embed JS probe as a single source of truth for both the command and the e2e harness (no divergent second probe)"
    - "Action produces the final mode-specific string (raw/json/human); the client prints it as-is, with --json passthrough for already-structured results"
    - "Live-page-only verdict reads via String(window.__detect.<signal>) — never a Go config field"

key-files:
  created:
    - internal/detect/probe.js
    - actions/stealth_check.go
  modified:
    - internal/detect/embed.go
    - daemon/daemon.go
    - cmd.go

key-decisions:
  - "Embedded the hand-written probe.js directly (no probe_raw.js -> terser step). detect.js is itself a hand-written unminified embed with no _raw.js sibling, and the snippet is small/dependency-free; this follows the existing detect.js precedent and avoids any npm install (threat T-26-SC: no new package)."
  - "probe.js is the canonical extract; detect.js stays the harness fixture page (it keeps the extra informational WebRTC/CDP probes it owns). Signal NAMES and semantics match exactly, so command and harness agree."
  - "Resolved the --json double-wrap concern by making runClientCommand pass through already-structured JSON object/array results (isJSONValue) instead of re-wrapping into {\"result\":\"<escaped>\"}. Plain human-message results (every other command) never start with { or [ and still wrap as before — verified against `snapshot --json`."

patterns-established:
  - "Token-efficient --raw: single line, PASS or FAIL + only failing name=FAIL(reason) tokens; no full-signal dump"
  - "Bounded probe-ready poll (~30x300ms) mirroring the harness waitForDetectReady before reading async-settled signals"

requirements-completed: [VALIDATE-01, VALIDATE-02]

# Metrics
duration: 9min
completed: 2026-06-24
status: complete
---

# Phase 26 Plan 04: Configurable Fingerprint Consistency Validator (stealth-check behavior) Summary

**`stealth-check [url]` now reads 11 table-stakes signals back from the LIVE page via a shared embedded probe and emits a human per-signal table, a single-line `--raw` PASS/FAIL-with-only-failing-signals, or a clean structured `--json` object.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-06-24T08:01:33Z
- **Completed:** 2026-06-24T08:10:54Z
- **Tasks:** 3
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments
- Extracted the shared table-stakes probe into `internal/detect/probe.js`, embedded as `detect.Probe` — one source of truth for both the `stealth-check` command and the detect.js harness (no divergent second probe).
- Implemented `actions.StealthCheck`: injects the probe via eval, polls `window.__detect.ready`, reads every signal back from the live page, codifies the 11 VALIDATE-01 thresholds, and formats raw/json/human.
- Wired the `stealth-check` daemon dispatch case (thin delegate, mirroring eval/snapshot).
- Fixed the `--json` double-wrap so the structured per-signal object reaches stdout cleanly (VALIDATE-02), without changing wrapping for any other command.
- Verified end-to-end against the offline detect fixture: human table, `--raw` (`PASS`), and `--json` (clean object) all correct; all 11 signals read PASS against the shipped stealth fingerprint.

## Task Commits

Each task was committed atomically:

1. **Task 1: Extract shared probe into probe.js + embed** - `1bdc106` (feat)
2. **Task 2: Implement StealthCheck action (+ cmd.go --json passthrough)** - `bcf9fef` (feat)
3. **Task 3: Wire stealth-check daemon dispatch case** - `68d3915` (feat)

## Files Created/Modified
- `internal/detect/probe.js` - Shared canonical IIFE computing the table-stakes signals (webdriver, pluginsLength, mimeTypesLength, userAgent, webglVendor, webglRenderer, languages, screen, windowChrome, chromeRuntime, timezone) + async `permissionsConsistent`, writing `window.__detect` with a `ready` flag and a 3s timeout fallback.
- `internal/detect/embed.go` - Added `//go:embed probe.js` exported as `var Probe string`.
- `actions/stealth_check.go` - `StealthCheck(ctx, url, raw, jsonOut)`: navigate-first when url set, inject `detect.Probe`, poll ready (~30x300ms), read each signal from the LIVE page, `computeVerdict` per signal, and `formatRaw`/`formatJSON`/`formatHuman`.
- `daemon/daemon.go` - `case "stealth-check"` delegating to `actions.StealthCheck`.
- `cmd.go` - `isJSONValue` helper + `--json` passthrough for already-structured daemon results.

## Verdict Thresholds Codified (VALIDATE-01)

All read from the live page (`String(window.__detect.<signal>)`):

| Signal | PASS condition | FAIL reason |
|--------|----------------|-------------|
| webdriver | `== "false"` | the value |
| pluginsLength | integer `> 0` | `0` |
| userAgent | does NOT contain `HeadlessChrome` | `HeadlessChrome` |
| webglVendor | not swiftshader/llvmpipe/software (case-insensitive) and not `no-context`/`no-extension` | the value |
| webglRenderer | same as webglVendor | the value |
| permissionsConsistent | `== "true"` | the value |
| languages | non-empty | `empty` |
| screen | `WxH` with both `> 0` | the value |
| windowChrome | `== "true"` | `absent` |
| chromeRuntime | `== "true"` | `absent` |
| timezone | non-empty and contains `/` (IANA) | the value |

Any read-error / probe-error value also fails its signal. (Mapping verified by a throwaway in-package table test covering each PASS/FAIL case and the raw FAIL-line format — added, run green, then removed; the committed end-to-end assertions are Plan 05's harness.)

## Decisions Made
- **Embed probe.js directly (no terser step).** `detect.js` is already a hand-written unminified embed with no `_raw.js` sibling; the probe is small and dependency-free, so mirroring that precedent keeps one convention and installs no npm package (threat T-26-SC satisfied).
- **probe.js canonical, detect.js stays the fixture.** Signal names/semantics match exactly; detect.js retains the extra informational WebRTC/CDP probes that only the fixture page needs.
- **`--json` passthrough over per-command special-casing.** `runClientCommand` is generic; the targeted `isJSONValue` check passes structured object/array results through verbatim while every plain-string result (which never starts with `{`/`[`) keeps the legacy `{"result":...}` wrapping.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking/Coordination] Fixed `--json` double-wrapping in runClientCommand**
- **Found during:** Task 2 (StealthCheck action), confirmed by live smoke test.
- **Issue:** The action returns a structured JSON object for `--json`, but the global `runClientCommand` json branch re-wrapped it as `{"result":"<escaped json string>"}` — a stringified blob, violating the VALIDATE-02 must-have that `--json` emit "a structured object of per-signal verdicts". The plan (Task 2 note) explicitly flagged this coordination point and asked me to choose and document the path.
- **Fix:** Added `isJSONValue` and a `--json` passthrough in `cmd.go`: when the daemon result is an already-valid JSON object/array, print it verbatim; otherwise wrap as before.
- **Files modified:** cmd.go
- **Verification:** `stealth-check --json` now emits a clean object (validated through `python3 -m json.tool`); `snapshot --json` still emits `{"result":"..."}` — no regression for other commands.
- **Committed in:** bcf9fef (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking/coordination, anticipated by the plan).
**Impact on plan:** The fix was required to meet the VALIDATE-02 must-have and was the documented coordination path the plan asked for. No scope creep — wrapping behavior for all other commands is unchanged.

## Issues Encountered
- The `internal/detect` package cannot be imported from outside the module tree, so the throwaway fixture-server driver used to smoke-test live had to be built inside the module (`internal/detecttmp/`) and then removed. No production code touched; tree is clean.

## Threat Flags

None. No new security surface beyond the plan's `<threat_model>`: the probe is a fixed embedded string with no user-input interpolation (T-26-10), `--raw` emits only PASS/FAIL + failing-signal names with no page dump and errors fold into per-signal values (T-26-11), verdicts are read live not from a Go field (T-26-12), and no new npm package was introduced (T-26-SC).

## Known Stubs
None. All three modes are fully wired and verified end-to-end against the detect fixture.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `stealth-check` is fully functional; Plan 05 can now drive `runCli("stealth-check")` and `runCli("--raw","stealth-check")` against the detect harness and assert the single-line raw form + per-signal verdicts (the end-to-end live-page proof).
- `go build ./...` and `go vet ./...` clean; `go test ./daemon/ ./actions/` green.

## Self-Check: PASSED

- internal/detect/probe.js — FOUND (created)
- internal/detect/embed.go — FOUND (modified, `//go:embed probe.js` -> `var Probe`)
- actions/stealth_check.go — FOUND (created, `func StealthCheck`, injects `detect.Probe`)
- daemon/daemon.go — FOUND (modified, `case "stealth-check"`)
- cmd.go — FOUND (modified, `isJSONValue` + --json passthrough)
- Commit 1bdc106 — FOUND
- Commit bcf9fef — FOUND
- Commit 68d3915 — FOUND

---
*Phase: 26-configurable-fingerprint-consistency-validator*
*Completed: 2026-06-24*
