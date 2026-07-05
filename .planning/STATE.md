---
gsd_state_version: 1.0
milestone: v2.3
milestone_name: Distribution & Publishing
status: roadmap_complete
stopped_at: awaiting_phase_51
last_updated: "2026-07-05T00:00:00.000Z"
last_activity: 2026-07-05
progress:
  total_phases: 4
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# STATE.md — rod-cli

## Current Position

Milestone: v2.3 — Distribution & Publishing
Status: Roadmap approved (4 phases, 51–54; red-team folded). Ready to plan Phase 51.
Next: `/anvil-discuss-phase 51` (or `/anvil-plan-phase 51`).
Last activity: 2026-07-05 — v2.3 roadmap approved. v2.2 (Pi Extension) closed & archived.

## Accumulated Context

- **v2.2 Pi Extension shipped** (close commit `e90b5d4`, audit + security PASSED): `extensions/pi/` (`@agenthands/rod-cli-pi`), 13 `browse_*` tools, 116 tests. Not yet on npm.
- **OMP loads the Pi extension only via a local symlink** — `~/.omp/plugins/node_modules/@agenthands/rod-cli-pi → extensions/pi` (`omp plugin link`). A native OMP-scoped package does not yet exist.
- **Portability landmine**: 5 enum tools runtime-import `StringEnum` from `@earendil-works/pi-ai` + `Type` from `typebox`; OMP wants injected `pi.zod`/`pi.typebox` (OMP's `pi-ai` dropped `StringEnum`).
- **CI is Go-only** — `test.yml`, `release.yml` (goreleaser on `v*`). No TS typecheck/test, no npm-publish job.

## v2.3 Decisions (locked 2026-07-05)

- Version: **v2.3** (additive minor).
- Architecture: **shared core + two thin adapters**, realized as **three published raw-TS packages** — `@agenthands/rod-cli-core` (real dependency of both adapters) + `@agenthands/rod-cli-pi` + `@agenthands/rod-cli-omp`. (Red-team B1: a relative-sibling core import is excluded from `npm pack`; no copy/bundle step — preserves raw-TS/no-build.)
- Publishing: **auto-publish on an `ext-v*` git tag via CI**, gated on typecheck + tests, `--access public`, `NPM_TOKEN`. Extension semver independent of the `v*` Go binary tags; core + adapters versioned together per release.

## v2.3 Roadmap (approved)

| Phase | Name | Requirements |
|-------|------|--------------|
| 51 | Shared Core Extraction + Pi Refactor + CI Gate | CORE-01, CORE-02, PI-01, CI-01 |
| 52 | Native OMP Extension | OMP-01..04 |
| 53 | Publish Readiness + Auto-Publish Pipeline | CORE-03, PI-02, PI-03, CI-02, CI-03 |
| 54 | Documentation & Discoverability | DOCS-01, DOCS-02 |

Independent red-team verdict: NO-GO → resolved (B1, B2, M1–M6 folded). Typed `gate_result` recorded in `ROADMAP.md` `## Records`.
