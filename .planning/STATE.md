---
gsd_state_version: 1.0
milestone: v1.6
milestone_name: Proven & Configurable Stealth
current_phase: 25
current_phase_name: Stealth Config Surface & Per-Session Proxy
status: executing
stopped_at: Completed 25-01-PLAN.md (stealth config substrate; PROFILE-01/02)
last_updated: "2026-06-24T06:07:30.151Z"
last_activity: 2026-06-24
last_activity_desc: "25-02 complete: per-session HTTP/SOCKS5 proxy wired through godoll ProxyConfig.ApplyToLauncher (replacing bare launcher.Proxy), CDP SetupBrowserAuth for auth, embedded-cred strip, relay cleanup stopped on session close (PROXY-01/02)"
progress:
  total_phases: 6
  completed_phases: 1
  total_plans: 7
  completed_plans: 6
  percent: 17
---

# STATE.md — rod-cli

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-24)

**Core value:** Native, token-efficient browser automation via standard I/O explicitly designed for LLM integration.
**Current focus:** Phase 25 — Stealth Config Surface & Per-Session Proxy

## Current Position

Phase: 25 of 29 (Stealth Config Surface & Per-Session Proxy) — second of 6 v1.6 phases (24–29)
Plan: 2 of 3 complete (25-01 + 25-02 done; 25-03 e2e egress/isolation/auth tests remain)
Status: Plan 25-02 executed — per-session proxy wired through godoll, CDP auth, relay cleanup (PROXY-01/02)
Last activity: 2026-06-24 — 25-02 complete: per-session HTTP/SOCKS5 proxy via godoll ProxyConfig.ApplyToLauncher (replaced bare launcher.Proxy), CDP SetupBrowserAuth, embedded-cred strip, relay stopped on session close

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

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 25/26] Daemon-shared per-session config is architectural: proxy is launch-time, fingerprint is per-page; a shared `*rod.Browser` means per-session config likely needs per-BrowserContext isolation or documented "applies at session spawn" semantics. Flag for discuss-phase; add a concurrent `-s` session-isolation test. (Research Pitfall 8.)
- [Phase 24] CDP transport is detectable regardless of JS spoofing. Treat "hide CDP" as a small spike with an explicit YES/NO and a documented ceiling, not a guaranteed deliverable; add CDP-tell probes to the harness. (Research Pitfall 3.)

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-06-24T06:07:25.199Z
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
