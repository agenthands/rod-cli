---
phase: 26-configurable-fingerprint-consistency-validator
verified: 2026-06-24T00:00:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: none
  previous_score: n/a
requirements:
  - id: FINGERPRINT-01
    status: satisfied
    evidence: "StealthConfig identity fields + 4 flags + profile overlay (types/config.go); profileFromStealth pins resolved cfg.Stealth as active stealth.Profile (types/context.go:415,486-488); harness pinned_identity_macos PASS (live UA major 130 != default 121, platform macOS reached the page)."
  - id: FINGERPRINT-02
    status: satisfied
    evidence: "deriveAndValidateFingerprint (types/config.go:294) derive-when-unset/reject-when-user-conflicts; blocking harness consistency_invariant PASS (Sec-Ch-Ua == UA == userAgentData version AND UA OS == platform == Sec-Ch-Ua-Platform, read live)."
  - id: FINGERPRINT-03
    status: satisfied
    evidence: "No 'Google Chrome\";v=\"121' brand literal in any rod-cli/godoll runtime path (grep clean). rod-cli interceptor derives via parseChromeMajor (context.go:602-605); godoll secChUa(chromeMajorFromUA(profile.UserAgent)) for header (evasion.go:227) and scriptMockUserAgentData(version) for JS (js.go:163,182). Harness client_hints_ua_derived flipped KNOWN-RED -> required-GREEN, PASS."
  - id: VALIDATE-01
    status: satisfied
    evidence: "actions.StealthCheck reads window.__detect signals live and codifies all VALIDATE-01 thresholds (stealth_check.go:182-247); daemon dispatch wired (daemon.go:242). Harness stealth_check PASS."
  - id: VALIDATE-02
    status: satisfied
    evidence: "formatRaw emits single line 'PASS' or 'FAIL name=FAIL(reason)' with only failing signals, no full-page dump (stealth_check.go:266-282); --raw forwarded via cmd.go:359; harness stealth_check asserts single-line raw form, PASS."
---

# Phase 26: Configurable Fingerprint & Consistency Validator — Verification Report

**Phase Goal:** A user can pin a coherent fingerprint per session from a single source of truth that feeds JS properties, HTTP headers, and Client-Hints alike (killing the hardcoded CH `121`), with a consistency validator that rejects or auto-derives incoherent combinations and fails loudly on contradiction — and can read a per-signal stealth-check verdict, including a token-efficient `--raw` machine-readable form, against any page.
**Verified:** 2026-06-24
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (mapped to ROADMAP Success Criteria)

| # | Truth (Success Criterion) | Status | Evidence |
|---|---------------------------|--------|----------|
| 1 | A user can pin a browser/OS/locale tuple via flags/profile onto godoll `stealth.Profile`, and the harness confirms Sec-CH-UA, navigator.userAgentData, UA, navigator.platform, WebGL all tell one consistent OS+version story (blocking test) | ✓ VERIFIED | `consistency_invariant` subtest PASS (blocking); reads all four surfaces + WebGL live and asserts both the version story and the OS story agree. `pinned_identity_macos` PASS confirms a `--user-agent` pin (Chrome 130, macOS) actually reaches the live page. StealthConfig identity fields + profile overlay + 4 flags wired (config.go:194-227, cmd.go:54-57, main.go:45-48). |
| 2 | Client-Hints derived from active UA/OS — Sec-CH-UA version == UA Chrome major — no hardcoded `121` literal anywhere in the header path | ✓ VERIFIED | grep: no `"Google Chrome";v="121"` brand literal in any rod-cli or godoll non-test runtime file. rod-cli interceptor derives via `parseChromeMajor(prof.UserAgent)` (context.go:602-605). godoll header via `secChUa(chromeMajorFromUA(em.profile.UserAgent))` (evasion.go:227), JS via `scriptMockUserAgentData(platform, version)` (js.go:163). `client_hints_ua_derived` harness subtest (flipped from KNOWN-RED) PASS. The sole remaining `121` is a documented fallback const mirroring DefaultProfile()'s `Chrome/121` UA — only used for empty/garbage UAs, keeping the default identity coherent. |
| 3 | Consistency validator rejects or auto-derives incoherent combinations and fails loudly on contradiction rather than shipping a mismatched lie | ✓ VERIFIED | `deriveAndValidateFingerprint` (config.go:294-345): platform↔UA-OS reject-when-user-conflicts/derive-when-unset, locale↔languages reject/derive, hardware/screen range validation, timezone↔proxy warn-only stderr. Runs in ResolveStealth before NewContext/browser launch (config.go:239, main.go:50). Error propagates non-zero + stderr field-naming message. types/ unit tests PASS. |
| 4 | A user can run stealth-check and get a per-signal verdict read from the live page | ✓ VERIFIED | `actions.StealthCheck` injects shared `detect.Probe`, reads `window.__detect.<signal>` live via RuntimeEvaluate, applies all VALIDATE-01 thresholds (stealth_check.go:55-247). Daemon dispatch wired (daemon.go:242). `stealth_check` harness subtest PASS against the live fixture. |
| 5 | With `--raw`, stealth-check emits a single-line PASS/FAIL plus only failing signals — no full-page dump | ✓ VERIFIED | `formatRaw` returns `"PASS"` or `"FAIL"` + only `name=FAIL(reason)` tokens (stealth_check.go:266-282). `--raw` forwarded from cmd.go:359 → daemon → action. Harness `stealth_check` subtest asserts the single-line raw form. |

**Score:** 5/5 truths verified (0 present-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `types/config.go` | StealthConfig identity fields, StealthFlags additions, deriveAndValidateFingerprint, parseChromeMajor, uaOSToPlatform | ✓ VERIFIED | All present + substantive (lines 65-93, 130-137, 255-380); wired into ResolveStealth. |
| `cmd.go` | 4 override flags + forwarding + stealth-check registration | ✓ VERIFIED | Flags 54-57/126, forwarding 54-57, stealth-check command 348-363. |
| `main.go` | daemon-side flag capture into StealthFlags | ✓ VERIFIED | UserAgent/Locale/Timezone/Platform captured 45-48; ResolveStealth called 50. |
| `types/context.go` | profileFromStealth pin + UA-derived interceptor Sec-Ch-Ua | ✓ VERIFIED | profileFromStealth 415-465; pinned as active profile 486-488; interceptor derives via parseChromeMajor 602-605. |
| `../godoll/stealth/evasion.go` | UA-derived Sec-Ch-Ua header, no 121 brand literal | ✓ VERIFIED | secChUa(chromeMajorFromUA(profile.UserAgent)) at 227; helpers 24-39. |
| `../godoll/stealth/js.go` | scriptMockUserAgentData parameterized by major | ✓ VERIFIED | Signature 163 `(platform, version string)`, version used 182. |
| `../godoll/stealth/fingerprint_bridge.go` | derived (not hardcoded) Timezone | ✓ VERIFIED | localeToTimezone map 13; derived 49-52 (America/New_York only as unknown-locale fallback). |
| `internal/detect/probe.js` + `embed.go` | shared probe writing window.__detect | ✓ VERIFIED | probe.js exists (writes window.__detect); embedded as `detect.Probe` (embed.go:16-17). |
| `actions/stealth_check.go` | StealthCheck action, live reads, raw/json/human | ✓ VERIFIED | StealthCheck exported 55; live reads 134-150; verdicts 182-247; raw/json/human 96-104. |
| `daemon/daemon.go` | stealth-check dispatch | ✓ VERIFIED | case at 242 → actions.StealthCheck. |
| `tests/detection_test.go` | flipped CH + consistency + pinned + stealth-check subtests | ✓ VERIFIED | All four subtests present + PASS live; KNOWN-RED 121 marker removed. |

### Key Link Verification

| From | To | Via | Status |
|------|----|-----|--------|
| cmd.go | main.go | 4 flags forwarded verbatim → StealthFlags capture | ✓ WIRED |
| main.go | types/config.go | ResolveStealth(cfg, &stealthFlags) validates+derives before NewContext | ✓ WIRED |
| context.go createPage | godoll EvasionManager | em.SetProfile(profileFromStealth(cfg.Stealth)) | ✓ WIRED |
| context.go updateInterceptorRules | active profile UA | Sec-Ch-Ua from parseChromeMajor(prof.UserAgent) | ✓ WIRED |
| godoll evasion.go | godoll js.go | both derive version from em.profile.UserAgent | ✓ WIRED |
| daemon.go | actions/stealth_check.go | dispatch case → actions.StealthCheck | ✓ WIRED |
| actions/stealth_check.go | internal/detect/probe.js | injects detect.Probe, reads window.__detect live | ✓ WIRED |

### Behavioral Spot-Checks (live harness — the phase gate)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full detection harness (12 subtests) | `go test ./tests/ -run TestDetectionHarness -count=1` | PASS (25.78s), all subtests green | ✓ PASS |
| client_hints_ua_derived (FINGERPRINT-03) | (subtest) | PASS — Sec-Ch-Ua == UA == userAgentData major, live | ✓ PASS |
| consistency_invariant (SC1, blocking) | (subtest) | PASS — version + OS story agree across all surfaces | ✓ PASS |
| pinned_identity_macos (FINGERPRINT-01) | (subtest) | PASS — pinned Chrome 130 / macOS reached the live page | ✓ PASS |
| stealth_check (VALIDATE-01/02) | (subtest) | PASS — per-signal + single-line raw form | ✓ PASS |
| CH-default regression guard | `go test ./tests/ -run TestNetworkEvasionHeaders -count=1` | PASS (3.83s) — default identity emits coherent Sec-Ch-Ua | ✓ PASS |
| Validator + action unit tests | `go test ./types/ ./actions/ ./daemon/` | PASS | ✓ PASS |
| godoll cross-repo no regression | `(cd ../godoll && go test ./stealth/ ./fingerprint/)` | PASS | ✓ PASS |
| Binary builds (incl. godoll replace) | `go build -o rod-cli .` | OK | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Status | Evidence |
|-------------|----------------|--------|----------|
| FINGERPRINT-01 | 26-01, 26-03, 26-05 | ✓ SATISFIED | pin → profileFromStealth → live page (pinned_identity_macos PASS) |
| FINGERPRINT-02 | 26-01, 26-05 | ✓ SATISFIED | deriveAndValidateFingerprint + blocking consistency_invariant PASS |
| FINGERPRINT-03 | 26-02, 26-03, 26-05 | ✓ SATISFIED | no 121 brand literal; UA-derived CH on all surfaces; client_hints_ua_derived PASS |
| VALIDATE-01 | 26-01, 26-04, 26-05 | ✓ SATISFIED | StealthCheck per-signal live verdict; stealth_check PASS |
| VALIDATE-02 | 26-01, 26-04, 26-05 | ✓ SATISFIED | formatRaw single-line failing-only; harness asserted |

All 5 declared requirement IDs are claimed by plans, mapped to Phase 26 in REQUIREMENTS.md, and verified live. No orphaned requirements.

### Anti-Patterns Found

None. No TBD/FIXME/XXX/HACK/PLACEHOLDER/TODO markers in any phase-modified file. The single `121` occurrence (context.go:406) is a documented fallback constant that mirrors DefaultProfile()'s Chrome major — used only for empty/garbage UAs, not the eliminated hardcoded brand tell.

### Deferred / In-Phase-Resolved Items

- **CH-off-by-default coherence gap (Plan 03 regression):** RESOLVED in-phase (deferred-items.md). `profileFromStealth` sets `SpoofClientHints=true` (context.go:424) so the DEFAULT identity emits coherent UA-derived Client-Hints. Confirmed: TestNetworkEvasionHeaders PASS + default fallback major `121` matches DefaultProfile() UA `Chrome/121`. Not a gap.
- **`--no-client-hints` disable flag:** intentionally omitted (an incoherent CH-off identity contradicts the phase goal; no requirement asks for it). Documented future enhancement, not phase-26 work.

### Human Verification Required

None. Every assertion is exercised by the live harness reading back from the actual browser page (validate-live-not-source); zero network egress. No visual/UX/external-service items.

### Gaps Summary

No gaps. All 5 success criteria and all 5 requirement IDs are verified against the live codebase and a passing live harness. The hardcoded CH `121` brand tell is eliminated across both repos, the consistency validator derives/rejects coherently before browser launch, the single source of truth (profileFromStealth) drives JS + headers + Client-Hints, and stealth-check returns a per-signal verdict with a token-efficient single-line `--raw` form. Phase goal achieved.

---

_Verified: 2026-06-24_
_Verifier: Claude (gsd-verifier)_
