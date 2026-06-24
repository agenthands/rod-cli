# Phase 26 — Deferred / Out-of-Scope Items

## RESOLVED in-phase: Client-Hints-off-by-default coherence gap (Plan 03 regression)

**Discovered:** Plan 26-05, Task 4 (full-suite regression run).
**Resolved:** post-Plan-05 orchestrator regression fix, before phase verification.

**Was:** `stealth.DefaultProfile()` ships `SpoofClientHints: false`, and Plan 03's
`updateInterceptorRules` gates the entire `Sec-Ch-Ua` / `navigator.userAgentData`
injection behind `if prof.SpoofClientHints`. Because Plan 03 switched `createPage`'s
active-profile source from the old `stealth.FromFingerprint(fp)` path (which set
`SpoofClientHints = UserAgentData != nil`, i.e. on) to `profileFromStealth` based on
`DefaultProfile()` (off), the **default** identity stopped emitting Client-Hints —
empty `Sec-Ch-Ua` + empty `userAgentData.brands` (itself a detection tell), and a
regression of the pre-v1.6 behavior. This broke the pre-existing v1.3
`tests/network_evasion_test.go::TestNetworkEvasionHeaders`.

**Fix:** `profileFromStealth` (`types/context.go`) now sets `p.SpoofClientHints = true`
on the resolved active profile, so rod-cli emits coherent, UA-derived Client-Hints by
default. `Sec-Ch-Ua` major == UA Chrome major == `navigator.userAgentData` brand version
== platform OS story — the FINGERPRINT-02 triple-agreement now holds on the DEFAULT
identity, not only when a profile opts in. Verified live (fresh `go build -o rod-cli .`):
- `tests/network_evasion_test.go::TestNetworkEvasionHeaders` → PASS (Sec-Ch-Ua present).
- `tests/detection_test.go::TestDetectionHarness` → PASS (all 11 subtests, incl.
  `client_hints_ua_derived`, `consistency_invariant`, `pinned_identity_macos`,
  `stealth_check`), zero network egress.

**Genuine follow-up (NOT a phase-26 gap):** there is no `--no-client-hints` flag to
*disable* CH. Intentionally omitted — shipping an incoherent CH-off identity contradicts
the phase's coherence goal, and no requirement asks for it. A disable knob (via profile
JSON `spoofClientHints:false` honored as an explicit override) is a possible future
enhancement, not outstanding work for this phase.

## Pre-existing / unrelated (not introduced by Phase 26)

- `daemon/daemon_more_test.go::TestStartServerWithPpid` — intermittently FAILS
  ("server with ppid did not become ready") under a heavily-loaded full
  `go test ./...` run. Daemon-readiness/ppid test, no relation to stealth/fingerprint.
  The file also carries an UNCOMMITTED pre-session signature edit (a 4th `nil` arg to
  `EnsureDaemon`) present at session start, unrelated to Phase 26. Left untouched.
