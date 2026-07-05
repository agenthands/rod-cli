---
gsd_state_version: 1.0
milestone: v2.3
milestone_name: Distribution & Publishing
status: milestone_complete
stopped_at: v2.3_shipped
last_updated: "2026-07-05T00:00:00.000Z"
last_activity: 2026-07-05
progress:
  total_phases: 4
  completed_phases: 4
  total_plans: 4
  completed_plans: 4
  percent: 100
---

# STATE.md — rod-cli

## Current Position

Milestone: v2.3 — Distribution & Publishing — **COMPLETE** (shipped 2026-07-05)
Status: All 4 phases (51–54) built + independently verified; MILESTONE-AUDIT PASSED. Security review skipped by co-driver decision.
Next: cut the live release once a maintainer `@agenthands` npm org + `NPM_TOKEN` exist — `git tag ext-v0.1.0 && git push --tags` triggers `publish-extensions.yml`. Otherwise, start the next milestone.
Last activity: 2026-07-05 — v2.3 shipped: shared core + Pi refactor + native OMP extension + publish pipeline + docs; B51-1 and rod-cli `--version` fixes folded in.

## Accumulated Context

- **v2.3 shipped**: three raw-TS packages in an `extensions/*` npm workspace — `@agenthands/rod-cli-core` (host-agnostic shared core: normalized binary-bound `ExecFn`/`ExecResult`, 13 pure argv-builders), `@agenthands/rod-cli-pi` (refactored onto the core, `cli.ts` deleted), `@agenthands/rod-cli-omp` (native OMP, injected `pi.zod`, bun:test). Gates: core 139 vitest, pi 39 vitest, omp 30 bun; all `tsc --noEmit` clean.
- **CI**: `.github/workflows/extensions.yml` (blocking core/pi/omp gate) + `publish-extensions.yml` (`ext-v*` gated auto-publish, core→pi→omp, `--access public` via `NPM_TOKEN`). Release convention: `docs/extension-release.md`.
- **Not yet on npm** — live publish blocked on a maintainer `@agenthands` org + `NPM_TOKEN`; verified to `npm pack` + scratch-install host-LOAD + `npm publish --dry-run`.
- **OMP-native path**: OMP now loads `@agenthands/rod-cli-omp` (the pi-symlink was removed from `~/.omp/plugins`).
- **rod-cli binary**: `banner.Version` defaults to `"dev"` so `rod-cli --version` works on any build (goreleaser ldflag still overrides for releases); `~/bin/rod-cli` rebuilt this session.
- **Deferred**: F52-2 — the 13 `pi.registerTool({…} as never)` casts (zod/v4 vs OMP's TypeBox-typed `ToolDefinition`); follow-up = narrow the cast to `parameters` only.

## Key artifacts
- `.planning/v2.3-MILESTONE-AUDIT.md`, `.planning/v2.3-SECURITY.md`, `.planning/research/omp-host-api.md`
- `.planning/phases/51..54/{CONTEXT,PLAN,SUMMARY,VERIFICATION}.md`
- `.planning/codebase/ARCHITECTURE.md` (§8 refreshed for v2.3)
