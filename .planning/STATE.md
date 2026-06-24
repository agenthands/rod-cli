---
gsd_state_version: 1.0
milestone: v1.6
milestone_name: Proven & Configurable Stealth
current_phase: 26
current_phase_name: Configurable Fingerprint & Consistency Validator
status: executing
stopped_at: Completed Phase 25 (config surface + per-session proxy) — verified 4/4; code review fixed CR-01 proxy-auth argv leak + credential-safety warnings
last_updated: "2026-06-24T08:12:22.024Z"
last_activity: 2026-06-24
last_activity_desc: Phase 26 execution started
progress:
  total_phases: 6
  completed_phases: 2
  total_plans: 12
  completed_plans: 11
  percent: 33
---

# STATE.md — rod-cli

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-24)

**Core value:** Native, token-efficient browser automation via standard I/O explicitly designed for LLM integration.
**Current focus:** Phase 26 — Configurable Fingerprint & Consistency Validator

## Current Position

Phase: 26 (Configurable Fingerprint & Consistency Validator) — EXECUTING
Plan: 5 of 5
Status: Ready to execute
Last activity: 2026-06-24 — Phase 26 execution started

Progress: [██░░░░░░░░] 17% (1 of 6 phases)

## Roadmap Summary

Brownfield "prove + configure + wire" stealth milestone. godoll already implements the capabilities; v1.6 proves them against a deterministic offline harness (read from the live page, never source), exposes a session-persistent config surface, and wires the two genuine godoll gaps. Harness-first ordering — every later phase is "done" only when the harness asserts it.

- Phase 24 — Detection Harness & CI Backbone: offline `go:embed` detection server + first test CI job, baselined against the current binary; loud-failure on `Apply()`/fingerprint errors (HARNESS-01/02/03, VALIDATE-03).
- Phase 25 — Stealth Config Surface & Per-Session Proxy: flags + named JSON profile, precedence resolved at daemon spawn, no cross-session bleed; per-session HTTP/SOCKS5 proxy with CDP auth (PROFILE-01/02, PROXY-01/02).
- Phase 26 — Configurable Fingerprint & Consistency Validator: single source of truth for UA/CH/platform/WebGL, consistency invariant gate, CH-121 fix, user-facing stealth-check verdict + `--raw` (FINGERPRINT-01/02/03, VALIDATE-01/02).
- Phase 27 — Canvas/WebGL/WebRTC Hardening: wire `EvadeWebRTC`/`WithWebRTCLeakProtection`, replace the `ApplyCanvasNoise` no-op stub with stable-per-session noise (HARDEN-01/02).
- Phase 28 — Human-Behavior Tuning: thread godoll `humanize.With*` options (typing/typo/jitter/mouse/scroll) through actions (HUMANIZE-01).
- Phase 29 — Best-Effort Live Validation: opt-in `//go:build detection_live` smoke check, non-blocking, honest ceiling (LIVEWAF-01).

All 17 v1 requirements mapped, 100% coverage.

## Operator Next Steps

- Plan the first phase with `/gsd-plan-phase 24`

## Accumulated Context

### Decisions

- v1.6 roadmap: harness-first ordering — the offline deterministic detection harness (Phase 24) gates and regression-nets every later phase; baseline against the *current* binary so existing leaks (e.g. unwired WebRTC) surface up front.
- Config substrate (Phase 25) lands early, paired with proxy (smallest already-half-wired feature) to validate the whole flag→config→godoll daemon-boundary path quickly.
- Consistency before hardening: the single source of truth + consistency invariant (Phase 26) must precede canvas/WebGL/WebRTC noise (Phase 27) so hardening has a coherent stable base rather than *creating* lies.
- Every stealth assertion reads back via `page.Eval` from the live (daemon-reused) browser, never from a Go config field — carried forward from the v1.5 `onDOMNodeInserted` wired-but-silent lesson.
- VALIDATE-01/02 (user-facing stealth-check verdict) placed in Phase 26 alongside the consistency validator since both surface per-signal reads of the pinned identity.
- [Phase ?]: Phase 24-01: detection page self-authored (no bot.sannysoft); offline //go:embed probe page writes per-signal verdicts into window.__detect, ready-gated.
- [Phase ?]: Phase 24-01: webrtcIce + cdpTell are informational/non-blocking KNOWN-RED signals — harness records current truth; HARDEN-01 (Phase 27) fixes WebRTC leak.
- [Phase ?]: VALIDATE-03: swallowed evasion errors now write warning: to stderr (log-and-continue, no daemon abort)
- [Phase ?]: 24-03: e2e detection harness reads window.__detect via eval (validate-live-not-source); WebRTC + Client-Hints kept as KNOWN-RED executing assertions, never skipped
- [Phase ?]: Phase 25 proxy/profile proven e2e via offline self-serving proxy fixture: auth by CONNECT/407 counters, isolation by live-page egress id, profile by JSON round-trip + session inheritance (identity overlay deferred to Phase 26).
- [Phase ?]: 26-02: killed hardcoded CH-121 in godoll injectors (evasion.go Sec-Ch-Ua + js.go userAgentData); both derive the Chrome major from em.profile.UserAgent via chromeMajorFromUA so header CH == JS userAgentData == UA major (FINGERPRINT-03/02). godoll edits uncommitted in its working tree, consumed via replace.
- [Phase ?]: 26-02: FromFingerprint timezone locale-derived (localeToTimezone map, America/New_York fallback) instead of hardcoded; locale is the only timezone signal available, keeps offline-determinism.
- [Phase ?]: Plan 26-03: config-pinned cfg.Stealth is the active stealth.Profile (SetProfile in createPage); interceptor Sec-Ch-Ua UA-derived via parseChromeMajor (no 121 literal)
- [Phase ?]: stealth-check probe is a single shared //go:embed probe.js used by both the command and the detect.js harness
- [Phase ?]: stealth-check --json passes through structured daemon results (isJSONValue) instead of double-wrapping

### Pending Todos

None yet.

### Blockers/Concerns

- ✅ RESOLVED (Phase 25): Daemon-shared per-session config concern does NOT apply — rod-cli is one-daemon-per-named-session (separate `rod-cli-<session>.port` + process + browser + Config). Per-session isolation is process isolation; NO per-BrowserContext work needed. Proven by the concurrent `-s` session-isolation test (`tests/proxy_test.go`).
- [Phase 26+] WR-02 follow-up: authenticated SOCKS5 proxy (`socks5://` + `--proxy-auth`) is currently accepted but godoll's auth relay speaks HTTP CONNECT to the SOCKS upstream — may mishandle it. Root cause is in godoll. Either reject the combo loudly or fix the relay; revisit when SOCKS-auth is actually needed.
- [Phase 24] CDP transport is detectable regardless of JS spoofing. Treat "hide CDP" as a small spike with an explicit YES/NO and a documented ceiling, not a guaranteed deliverable; add CDP-tell probes to the harness. (Research Pitfall 3.)

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-06-24T08:11:59.019Z
Stopped at: Completed 25-01-PLAN.md (stealth config substrate; PROFILE-01/02)
Resume file: None

## Performance Metrics

**Velocity:**

- Total plans completed: 0 (v1.6)
- Average duration: —
- Total execution time: —

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |
</content>
| Phase 24 P01 | 2min | 2 tasks | 5 files |
| Phase 24 P02 | 5m | 1 tasks | 1 files |
| Phase 24 P03 | 6min | 1 tasks | 1 files |
| Phase 25 P01 | 20min | 3 tasks | 6 files |
| Phase 25 P02 | 15min | 2 tasks | 2 files |
| Phase 25 P03 | 25min | 2 tasks | 2 files |
| Phase 26 P01 | 3m47s | 3 tasks | 3 files |
| Phase 26 P02 | 9min | 4 tasks | 5 files |
| Phase 26 P03 | 10m | 2 tasks | 1 files |
| Phase 26 P04 | 9min | 3 tasks | 5 files |
