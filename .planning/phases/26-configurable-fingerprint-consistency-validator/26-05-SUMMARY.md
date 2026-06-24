---
phase: 26
plan: 05
subsystem: detection-harness
status: complete
tags: [fingerprint, stealth, client-hints, harness, validate-live-not-source, FINGERPRINT-01, FINGERPRINT-02, FINGERPRINT-03, VALIDATE-01, VALIDATE-02]
dependency_graph:
  requires:
    - "Plan 02 godoll UA-derived navigator.userAgentData brand version"
    - "Plan 03 createPage applies cfg.Stealth profile; UA-derived Sec-Ch-Ua interceptor (no 121 literal)"
    - "Plan 04 stealth-check command (human/--raw/--json) reading verdicts from the live page"
    - "Phase 24 internal/detect offline fixture + evalDetect/waitForDetectReady harness cadence"
  provides:
    - "client_hints_ua_derived: required-green assertion (was KNOWN-RED) — live Sec-Ch-Ua major == UA Chrome major == navigator.userAgentData version"
    - "consistency_invariant: BLOCKING phase-gate subtest — Sec-CH-UA, userAgentData, UA, navigator.platform + WebGL tell one OS+version story, read live"
    - "pinned_identity_macos: proves a --user-agent pin reaches all live surfaces on a non-default (Mac Chrome/130) identity"
    - "stealth_check: proves per-signal verdicts + single-line --raw (no page dump) + parseable --json from the live page"
  affects:
    - "Phase 26 verifier (this is the phase gate); Phase 27 hardening builds on a coherent baseline"
tech_stack:
  added: []
  patterns:
    - "Activate-the-feature-under-test: Client-Hints spoofing is opt-in, so the coherence subtests enable it via a temp --profile JSON (writeProfile) — assertions stay hard live-page reads, never weakened"
    - "Read JS surfaces on the fixture page FIRST, header surfaces LAST (header-echo navigation drops window.__detect)"
    - "restoreDefaultDaemon defer guard so a pinned/profile session never bleeds into a later subtest"
key_files:
  created:
    - .planning/phases/26-configurable-fingerprint-consistency-validator/deferred-items.md
  modified:
    - tests/detection_test.go
decisions:
  - "Discovered Client-Hints spoofing is OFF by default (DefaultProfile.SpoofClientHints=false; no --spoof-client-hints flag). The coherence subtests therefore ACTIVATE it via a temp --profile JSON so they prove REAL cross-surface coherence rather than passing vacuously on empty CH. Turning a feature on is legitimate test setup; every assertion remains a hard t.Errorf reading from the live page."
  - "pinned_identity_macos pins UA/platform via the --user-agent/--platform GLOBAL flags (the plan's required mechanism) and enables CH via --profile; CLI flags override the profile per ResolveStealth precedence, so the pin's reach is what is proven."
  - "Reported the default-CH-off gap (a Plan-03 regression that also breaks the pre-existing TestNetworkEvasionHeaders) in deferred-items.md + this SUMMARY rather than masking it, per Task 4 'do not paper over a coherence gap'. The fix is production-code (types/context.go/godoll), outside this plan's tests-only scope."
metrics:
  duration: "~16m"
  completed: "2026-06-24"
  tasks: 4
  files_modified: 1
  files_created: 1
---

# Phase 26 Plan 05: Configurable Fingerprint Consistency Validator (the phase gate) Summary

Turned the Phase-26 detection harness into the phase's **blocking gate**: flipped the Client-Hints `121` KNOWN-RED baseline into a required-green assertion, added a BLOCKING `consistency_invariant` subtest proving Sec-CH-UA / `navigator.userAgentData` / UA / `navigator.platform` / WebGL all tell one OS+version story, added a `pinned_identity_macos` subtest proving a `--user-agent` pin reaches every live surface on a non-default (Mac Chrome/130) identity, and added a `stealth_check` subtest proving per-signal verdicts plus a single-line `--raw` form. Every assertion reads back from the LIVE (daemon-reused) browser via `eval` / loopback header-echo — never a Go config field — with zero network egress.

## What Was Built

### Task 1 — Flip CH KNOWN-RED + blocking consistency invariant (commit 5cbb419)
- Replaced `client_hints_known_red` (a `t.Logf`-only baseline) with **`client_hints_ua_derived`**: a hard `t.Errorf` that the live `Sec-Ch-Ua` major == UA Chrome major == `navigator.userAgentData` "Google Chrome" version, on the default-OS identity (Chrome/121, Win32).
- Added **`consistency_invariant`** (success criterion 1, FINGERPRINT-01/02): reads Sec-CH-UA (header-echo), `navigator.userAgentData` major, UA + Chrome major, `navigator.platform`, and WebGL vendor/renderer from the LIVE page and asserts ONE OS+version story (version triple-agreement; UA-OS == platform == Sec-Ch-Ua-Platform; WebGL not a software rasterizer).
- Added helpers: `chromeMajor`, `liveEval`, `liveUserAgentDataMajor`, `liveHeader`, `liveSecChUaMajor`, `stripQuotes`, `osFromUA/osFromPlatform/osFromChPlatform`, `writeProfile`, `restoreDefaultDaemon`.
- Removed the Phase-26 KNOWN-RED marker + `t.Skip`-free guarantee preserved; updated the file-header doc to record the flip.

### Task 2 — Pinned-identity subtest (commit bc1df98)
- **`pinned_identity_macos`**: `runCli("close")`, then spawn with `--user-agent <Mac Chrome/130>` `--platform MacIntel` as GLOBAL flags (plus a `--profile` enabling Client-Hints), then read back from the live page: UA contains `Chrome/130` and not `HeadlessChrome`, `navigator.platform == MacIntel`, `userAgentData` major == 130, `Sec-Ch-Ua` major == 130, `Sec-Ch-Ua-Platform == macOS`. Asserts the daemon-spawn consistency gate does NOT reject the coherent pin; `restoreDefaultDaemon` defer prevents leak (FINGERPRINT-01/03).

### Task 3 — stealth-check subtest (commit 946806b)
- **`stealth_check`**: drives `stealth-check` (human), `--raw stealth-check`, `--json stealth-check` against the offline fixture. Human mode names per-signal labels (`webdriver`, `webglVendor`); `--raw` is a single trimmed line starting PASS/FAIL with no full UA/page dump (FAIL form carries only `name=FAIL(reason)` tokens); `--json` parses as a JSON object (VALIDATE-01/02).

### Task 4 — Full harness gate + regression sweep
- `go build -o rod-cli .` + `go test ./tests/ -run TestDetectionHarness -count=1` — **PASS**, all 11 subtests green. Ran the full `go test ./...` sweep and triaged two out-of-scope failures (below).

## Verification Results

`go test ./tests/ -run TestDetectionHarness -count=1` — **PASS (25.5s)**:

```
--- PASS: TestDetectionHarness (25.58s)
    --- PASS: webgl_not_software
    --- PASS: permissions_consistent
    --- PASS: timezone_resolved
    --- PASS: window_chrome_present
    --- PASS: languages_present
    --- PASS: screen_nonzero
    --- PASS: webrtc_ice_known_red
    --- PASS: client_hints_ua_derived   (6.19s)   <- flipped, required-green
    --- PASS: consistency_invariant      (7.20s)   <- BLOCKING gate (criterion 1)
    --- PASS: pinned_identity_macos      (7.22s)   <- --user-agent pin reaches live page
    --- PASS: stealth_check              (1.12s)   <- per-signal + --raw single line
```

- `go build -o rod-cli .` — PASS. `go vet ./tests/` — PASS.
- Gate hygiene: non-comment grep for `KNOWN-RED (Phase 26` == 0; non-comment grep for `t.Skip` == 0; `userAgentData` present.
- Zero network egress: grep for external URLs in `detection_test.go` finds none (only `ds.URL()` offline fixture + loopback `httptest` header-echo).

### Full-suite sweep (`go test ./...`)
Most packages PASS (`rod-cli`, `actions`, `banner`, `internal/detect`, `internal/plugin*`, `types`, `utils`). Two FAILURES, both **outside this plan's scope** and pre-existing/regression — see "Deferred Issues".

## Deviations from Plan

### Auto-fixed Issues
None requiring code change in scope. One plan-anticipated adaptation:

**[Plan-anticipated setup] Activate Client-Hints in the coherence subtests.**
- **Found during:** Task 1 — the default identity emits NO `Sec-Ch-Ua` and an empty `navigator.userAgentData.brands` because `SpoofClientHints` defaults to `false`.
- **Resolution:** The `consistency_invariant` / `client_hints_ua_derived` / `pinned_identity_macos` subtests enable CH via a temp `--profile` JSON (`writeProfile`, `spoofClientHints:true`). This activates the feature-under-test so the live-page assertions prove REAL coherence; no assertion was weakened. Verified live: with CH on, UA/platform/userAgentData/Sec-Ch-Ua/Sec-Ch-Ua-Platform all agree (130/MacIntel/macOS and 121/Win32/Windows).

## Deferred Issues (out of scope — reported per Task 4, not masked)

See `deferred-items.md` for full detail.

1. **BLOCKING coherence gap (Plan-03 regression): Client-Hints OFF by default.**
   `stealth.DefaultProfile().SpoofClientHints == false` and Plan 03 gates the
   interceptor's `Sec-Ch-Ua` injection behind that flag, so the **default
   identity emits no Client-Hints** and `navigator.userAgentData.brands` is `[]`.
   There is also no `--spoof-client-hints` CLI flag (only a profile/config field),
   so `--user-agent` alone does NOT auto-derive Client-Hints. This breaks the
   pre-existing `tests/network_evasion_test.go::TestNetworkEvasionHeaders` (expects
   `Sec-Ch-Ua` on the default identity). The FINGERPRINT-02/03 wiring WORKS when
   activated (proven by this plan's subtests), but coherence is **opt-in, not the
   shipped default**. Fix is production-code (`types/context.go` / godoll
   `DefaultProfile`, plus a flag), outside this plan's `tests/detection_test.go`-only
   scope — flagged for the Phase-26 verifier / a follow-up plan.

2. **Pre-existing / unrelated:** `daemon/daemon_more_test.go::TestStartServerWithPpid`
   timed out ("server with ppid did not become ready") under the loaded full sweep;
   unrelated to stealth/fingerprint, in a file with an uncommitted pre-session
   signature edit. Left untouched.

## Authentication Gates
None.

## Known Stubs
None.

## Threat Flags
None. All new assertions read from the live page (T-26-13 upheld); only `ds.URL()`
offline fixture + loopback `httptest` servers (T-26-14 zero egress); the flipped
hard `t.Errorf` + blocking `consistency_invariant` make a future UA/CH/platform
mismatch fail CI (T-26-15); no new test deps — only stdlib `net/http/httptest`,
`encoding/json`, `regexp`, `os` plus the existing `internal/detect` (T-26-SC).

## Self-Check: PASSED

- tests/detection_test.go — FOUND (modified; contains `consistency_invariant`, `client_hints_ua_derived`, `pinned_identity_macos`, `stealth_check`, `userAgentData`; no `KNOWN-RED (Phase 26` and no `t.Skip` on non-comment lines)
- .planning/phases/26-configurable-fingerprint-consistency-validator/deferred-items.md — FOUND (created)
- Commit 5cbb419 — FOUND
- Commit bc1df98 — FOUND
- Commit 946806b — FOUND

---
*Phase: 26-configurable-fingerprint-consistency-validator*
*Completed: 2026-06-24*
