---
gsd_state_version: 1.0
milestone: v2.3
milestone_name: Distribution & Publishing
status: defining_requirements
stopped_at: roadmap
last_updated: "2026-07-05T00:00:00.000Z"
last_activity: 2026-07-05
progress:
  total_phases: 0
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# STATE.md — rod-cli

## Current Position

Milestone: v2.3 — Distribution & Publishing
Status: Defining requirements → roadmap
Last activity: 2026-07-05 — v2.3 started. v2.2 (Pi Extension) closed & archived.

## Accumulated Context

- **v2.2 Pi Extension shipped** (close commit `e90b5d4`, audit + security PASSED): `extensions/pi/` (`@agenthands/rod-cli-pi`), 13 `browse_*` tools, 116 tests. Followed by 14 post-close hardening commits (daemon/CDP timeout bounds, CSS-selector fallback, `--raw` doc corrections).
- **Not yet on npm** — `npm view @agenthands/rod-cli-pi` → 404. The README's `pi install npm:@agenthands/rod-cli-pi` would fail today.
- **OMP loads the Pi extension only via a local symlink** — `~/.omp/plugins/node_modules/@agenthands/rod-cli-pi → extensions/pi` (`omp plugin link`) plus a full local `node_modules`. A native OMP-scoped package does not yet exist.
- **Portability landmine**: 5 enum tools runtime-import `StringEnum` from `@earendil-works/pi-ai` + `Type` from `typebox`; OMP wants injected `pi.zod`/`pi.typebox` (OMP's `pi-ai` dropped `StringEnum`).
- **CI is Go-only** — `test.yml` (build + `go test` + govulncheck), `release.yml` (goreleaser on `v*`). No TS typecheck/test, no npm-publish job.

## v2.3 Decisions (locked 2026-07-05)

- Version: **v2.3** (additive minor — publishing + sibling extension, no breaking changes).
- Architecture: **shared core + two thin adapters** (`@agenthands/rod-cli-pi` + `@agenthands/rod-cli-omp`), each with its own host scope, schema library, and test runner; shared binary-discovery / `execRodCli` / validation / argv-building core.
- Publishing: **auto-publish on an `ext-v*` git tag via CI**, gated on typecheck + tests, `--access public`, `NPM_TOKEN` secret. Extension semver is independent of the `v*` Go binary tags.
