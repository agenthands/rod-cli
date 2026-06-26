---
gsd_state_version: 1.0
milestone: v1.8
milestone_name: Debt Cleanup & Coding-Assistant Onboarding
status: defining_requirements
stopped_at: v1.8 started — defining requirements
last_updated: "2026-06-26T00:00:00.000Z"
last_activity: 2026-06-26
progress:
  total_phases: 0
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# STATE.md — rod-cli

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-26)

**Core value:** Native, token-efficient browser automation via standard I/O explicitly designed for LLM integration.
**Current focus:** v1.8 — retire the three v1.7 follow-ups and ship per-coding-assistant install + agent-skill documentation.

## Current Position

Phase: Not started (defining requirements)
Plan: —
Status: Defining requirements
Last activity: 2026-06-26 — Milestone v1.8 started

## Roadmap Summary

To be created by roadmap step. v1.8 covers: (A) three v1.7 debt follow-ups — go1.26.1 toolchain bump, plugin-path CDP-ledger coverage fix, real/observable font-spoof; and (B) authoritative install + agent-skill docs for Claude Code, Codex CLI, Gemini CLI, Pi (pi.dev), and opencode.

## Operator Next Steps

1. Confirm requirements in REQUIREMENTS.md
2. Review + approve the roadmap

## Accumulated Context

### Decisions

- v1.8 scope = v1.7 debt cleanup + coding-assistant onboarding docs (operator-chosen).
- Font-spoof: make it REAL & observable (replace godoll no-op; harness asserts detected-font change). Largest single item.
- All three v1.7 follow-ups in scope: toolchain bump, plugin-path CDP-ledger hole, font-spoof.
- Docs must cover agent-skill installation per assistant, not just the binary `go install` (standing project preference).
- TLS spoofing stays OUT (real Chrome only — lives in the separate "munch" project).
- Research-first: verify each assistant's current skill-registration mechanism against live docs before scoping docs.
- Phase numbering continues from 33 (v1.8 starts at Phase 34).

### Pending Todos

None yet.

### Blockers/Concerns

- Font-spoof real fix may require touching the vendored/local godoll font injector — verify how rod-cli consumes godoll before committing to the approach.
- Per-assistant skill mechanisms drift fast; docs accuracy depends on current-docs research (Pi = pi.dev / `@earendil-works/pi-coding-agent`).

## Session Continuity

Last session: 2026-06-26
Stopped at: Milestone v1.8 kickoff — PROJECT.md + STATE.md updated, research next.
Resume options: continue `/anvil-new-milestone` (research → requirements → roadmap).

## Performance Metrics

**Velocity:**

- Total plans completed: 0 (v1.8)
- Average duration: —
- Total execution time: —
