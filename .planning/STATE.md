---
gsd_state_version: 1.0
milestone: v1.9
milestone_name: godoll Hygiene & CDP-DEEP-01 Research
status: defining_requirements
stopped_at: v1.9 started — defining requirements
last_updated: "2026-06-26T22:00:00.000Z"
last_activity: 2026-06-26
progress:
  total_phases: 2
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# STATE.md — rod-cli

## Project Reference

See: .planning/PROJECT.md

**Core value:** Native, token-efficient browser automation via standard I/O explicitly designed for LLM integration.
**Current focus:** v1.9 — close v1.7 F2/F4 godoll hygiene and produce a grounded CDP-DEEP-01 design.

## Current Position

Phase: Not started (defining requirements)
Plan: —
Status: Defining requirements
Last activity: 2026-06-26 — Milestone v1.9 started

## Roadmap Summary

v1.9 covers: (A) F2/F4 godoll hygiene from the v1.7 security review — backslash reject in rod-cli, json.Marshal in godoll's EnableRequestInterception; and (B) CDP-DEEP-01 research — evaluate browser-patching / MITM / patched-endpoint approaches against Chrome's detection surface, produce a grounded, executable PLAN for the chosen approach.

## Accumulated Context

### Decisions (carried forward)
- rod-cli is a pure CLI/daemon (no MCP). All onboarding docs teach shell-out only.
- TLS spoofing stays OUT (real Chrome only — lives in "munch").
- Font spoofing is now real (godoll `1d90494`, v1.8 Phase 36).
- v1.7 F1 (toolchain) closed in v1.8. F2/F4 are the last unaddressed v1.7 security findings.
- CDP-DEEP-01 is research/design only — not a build phase. The build is gated on this milestone's output.

### Pending Todos
None.

### Blockers/Concerns
- F2/F4 are trivial (one-line changes each). The real substance is CDP-DEEP-01 research.
- CDP-DEEP-01 requires understanding go-rod's CDP transport, Chrome's debugger-observable surface, and evaluating three architecturally different approaches — this is a research-heavy phase.

## Session Continuity

Last session: 2026-06-26
Stopped at: v1.9 kickoff
Resume options: requirements confirmed, proceed to phase loop.
