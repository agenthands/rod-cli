---
phase: 25-stealth-config-surface-per-session-proxy
plan: 03
subsystem: testing
tags: [proxy, stealth, profile, session-isolation, cdp-auth, e2e, fixture, godoll, credential-safety]

# Dependency graph
requires:
  - phase: 25-stealth-config-surface-per-session-proxy
    plan: 01
    provides: StealthConfig (Proxy, ProxyAuth, ProfilePath) + ResolveStealth frozen at daemon spawn; --proxy/--proxy-auth/--profile flags
  - phase: 25-stealth-config-surface-per-session-proxy
    plan: 02
    provides: parseProxyConfig + ApplyToLauncher/SetupBrowserAuth wiring + per-session relay cleanup
  - phase: 24-detection-harness-and-ci-backbone
    provides: live-binary runCli harness + offline-fixture / validate-live-not-source test conventions
provides:
  - tests/proxyfixture — offline loopback HTTP forward proxy (no-auth + 407-enforcing auth variants) with per-fixture egress id and request/407 counters
  - tests/proxy_test.go — four live-binary e2e proofs (session isolation, CDP proxy auth, profile round-trip + inheritance, credential non-leak)
  - executable proof of T-25-09 (cred non-leak), T-25-10 (session isolation), T-25-11 (auth enforced not bypassed)
affects: [26 (fingerprint pins overlay onto live page — profile-identity assertion flips to required-green there), 27 (WebRTC/DNS leak-past-proxy harness builds on this fixture)]

# Tech tracking
tech-stack:
  added: []  # pure Go stdlib fixture; godoll/stealth already in go.mod
  patterns:
    - "Offline forward-proxy fixture mirrors testserver New/Start/Close/URL; serves an id-tagged page on plain-HTTP forward so a test reads egress identity from the live DOM without any real IP"
    - "Auth-path egress proven via fixture accepted-CONNECT vs 407 counters (the godoll relay tunnels CONNECT, so the served-page id is not DOM-observable in that path)"
    - "Counter-window phases are SERIALIZED (close session before measuring the next) so a shared-fixture counter is not contaminated by a still-running daemon's relay retries"

key-files:
  created:
    - tests/proxyfixture/server.go
    - tests/proxy_test.go
  modified: []

key-decisions:
  - "Isolation egress is read from the live page (fixture serves its id at #egress / data-proxy-id) and cross-checked against each fixture's request counter — observable behavior, not Go internals."
  - "Auth success/failure is asserted on the fixture's accepted-CONNECT vs 407 counters, not on the goto outcome (Chrome reports a navigation even to an unreachable upstream target)."
  - "Profile round-trip asserts the Phase-25 observable truths: stealth.Profile JSON Save/LoadProfile round-trips, and a --profile session inherits its spawn-time config on a second command. Profile IDENTITY overlay onto the live page is deliberately Phase 26 (FINGERPRINT) and is NOT asserted here."
  - "No skip directive anywhere; sandbox/Phase-boundary limitations are documented here rather than skipped (verify gate negative-greps test files)."

patterns-established:
  - "Egress identity via a self-serving offline proxy fixture: navigating through fixture X to ANY http target returns X's id page — deterministic, offline, no public IP."
  - "Serialize shared-counter measurement windows across concurrent-daemon phases to avoid relay-retry contamination."

requirements-completed: [PROFILE-01, PROFILE-02, PROXY-01, PROXY-02]

# Metrics
duration: ~25min
completed: 2026-06-24
status: complete
---

# Phase 25 Plan 03: End-to-End Proxy, Profile & Credential-Safety Proof Summary

**An offline loopback HTTP forward-proxy fixture plus four live-binary e2e tests that prove per-session proxy isolation (no bleed), CDP `--proxy-auth` is answered for correct creds and 407-enforced for wrong, a `stealth.Profile` JSON round-trips and is inherited across commands on a session, and `--proxy-auth` secrets never reach stdout/stderr or the `.port` file.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-06-24
- **Tasks:** 2 of 2
- **Files created:** 2

## Accomplishments
- Built `tests/proxyfixture/server.go`: a pure-stdlib OFFLINE HTTP forward proxy on `127.0.0.1:0`, with `New(id)` (no-auth) and `NewAuth(id,user,pass)` (407-enforcing) variants. On a plain-HTTP forward it serves its OWN id-tagged page (egress marker), on `CONNECT` it auth-gates then tunnels; exposes `URL()/Addr()/Start()/Close()/RequestCount()/AuthFailureCount()`.
- `TestProxySessionIsolation` (PROXY-01 / PROFILE-02): two concurrent `-s` sessions through distinct fixtures egress through their OWN proxy — egress id read back from each live page (FIXTURE-A vs FIXTURE-B) and cross-checked against per-fixture counters — proving no cross-session bleed.
- `TestProxyAuthViaCDP` (PROXY-02): correct `--proxy-auth` is answered (accepted CONNECT, zero 407s); wrong creds produce 407s and zero accepted CONNECTs — proving auth is enforced via CDP, not bypassed. Phases serialized so the shared-fixture counters are clean.
- `TestProfileRoundTripInherited` (PROFILE-01): a `stealth.Profile` saved to JSON round-trips through `Save`/`LoadProfile`, and a `--profile` session inherits its spawn-time config on a second command issued WITHOUT `--profile`.
- `TestProxyCredentialsNotLeaked` (T-25-09): no `--proxy-auth` username/password appears in the folded stdout/stderr output nor in `rod-cli-<session>.port` (which carries only the port; asserted credential-free and URL-character-free).

## Task Commits

Each task was committed atomically:

1. **Task 1: Offline HTTP forward proxy fixture (no-auth + auth)** — `0a04652` (feat)
2. **Task 2: Session-isolation + auth + profile round-trip + credential-leak tests** — `7738077` (test)

## Files Created/Modified
- `tests/proxyfixture/server.go` (created) — Loopback forward proxy: `New`/`NewAuth`, `handleForward` (serves id-tagged egress page + counts), `handleConnect` (auth-gated tunnel), `checkAuth` (Basic), 407 path, request/407 counters. Pure stdlib (`net`, `net/http`, `encoding/base64`), `WARNING: test-only`.
- `tests/proxy_test.go` (created) — Four subtests driving the live `../rod-cli` via `runCli`, each with the deterministic `close` idiom; helpers `runCliSession`, `evalText`, `pollEgressID`. No skip directives.

## Decisions Made
- Egress identity is read from the LIVE page (fixture `#egress data-proxy-id`) and cross-checked with the fixture request counter — not from any Go config field (validate-live-not-source).
- Auth proof uses the fixture's accepted-CONNECT vs 407 counters because the `--proxy-auth` path routes through godoll's local CONNECT relay (HTTPS tunnel), so the served-page id is not DOM-observable in that path.
- Profile test asserts only the Phase-25 observable truths (JSON round-trip + session inheritance); identity-field overlay onto the live page is Phase 26's job (per Plan 01's recorded boundary) and is intentionally NOT asserted here.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Serialize the auth test's correct/wrong-creds counter windows**
- **Found during:** Task 2 (TestProxyAuthViaCDP)
- **Issue:** The first draft measured the wrong-creds phase while the correct-creds session (`auth_ok`) was still running. Its daemon's relay kept issuing CONNECTs into the SHARED fixture counter, so the wrong-creds window saw 3 "accepted" CONNECTs and the `servedBad != 0` assertion failed (false bypass signal).
- **Fix:** Close the `auth_ok` session BEFORE opening the wrong-creds measurement window and add a settle delay, so each phase's delta reflects only its own session. Verified empirically (wrong: accepted=0, 407s>0; correct: accepted>0, 407s=0).
- **Files modified:** tests/proxy_test.go
- **Verification:** `go test ./tests/ -run TestProxyAuthViaCDP` PASS; full four-test suite PASS together.
- **Committed in:** `7738077` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 test-correctness bug)
**Impact on plan:** The fix was necessary for a correct, non-flaky auth assertion against a shared fixture. No scope creep.

## Issues Encountered
- The `--proxy-auth` path routes through godoll's local CONNECT relay, so for an HTTPS target the egress page id is not observable in the DOM (the relay speaks raw CONNECT, the upstream target `*.test` does not resolve). Resolved by asserting auth via the fixture's accepted-CONNECT vs 407 counters — the authoritative, observable signal — rather than a DOM id. The no-auth isolation path, by contrast, uses plain-HTTP forward where the served id IS DOM-observable.

## Environment / Boundary Limitations (documented, not skipped)
- **Profile identity not yet on the live page:** Phase 25 records only that a profile was selected (`cfg.Stealth.ProfilePath`); the live page still uses a freshly generated fingerprint (`types/context.go:392-399`). So `TestProfileRoundTripInherited` proves JSON round-trip + spawn-time inheritance, NOT that a profile-derived identity value appears on the page. That assertion belongs to Phase 26 (FINGERPRINT) and should flip to required-green there.
- **Auth egress page:** see Issues Encountered — auth-path egress is proven by counters, not DOM id, by design of the godoll relay.

## Verification
- `go build ./...` and `go vet ./...` clean.
- `go test ./tests/ -run 'TestProxySessionIsolation|TestProxyAuthViaCDP|TestProfileRoundTripInherited|TestProxyCredentialsNotLeaked' -count=1 -timeout 300s` — all PASS (≈27s).
- `go test ./tests/ -run 'Proxy|Profile' -timeout 300s` — PASS.
- `gofmt -l` clean on both new files; no `t.Skip`/`Skipf`/`SkipNow` directives in `tests/proxy_test.go`.
- Manual smoke (live binary + fixture): no-auth proxy returns the fixture id on the live page; correct creds tunnel (0x 407), wrong creds 407; `.port` file holds only the port.

## Known Stubs
None. The profile identity-overlay gap is a Phase-26 boundary (documented above), not a stub blocking this plan's goal — the four required proofs (isolation, auth, round-trip+inheritance, credential safety) all pass against the live binary.

## Threat Flags
None — no new security surface beyond the proxy/credential paths already modeled in the plan's `<threat_model>`. This plan adds executable proof for T-25-09/T-25-10/T-25-11.

## Next Phase Readiness
- The offline proxyfixture is reusable for Phase 27's WebRTC/DNS leak-past-proxy harness.
- Phase 26 should extend `TestProfileRoundTripInherited` (or add a sibling) to assert a profile-derived identity value read back from the live page once fingerprint overlay lands.

## Self-Check: PASSED

---
*Phase: 25-stealth-config-surface-per-session-proxy*
*Completed: 2026-06-24*
