---
author: engineer
phase: 54
status: complete
---

# Phase 54: Documentation & Discoverability — SUMMARY

## What was done

All 6 doc files edited per PLAN.md §§1–6:

| File | Change |
|---|---|
| `extensions/pi/README.md` | Replaced 2-way table with D-01 three-way OMP/Pi/Skill table; labelled npm install "(available once published)" with local path shown first |
| `extensions/omp/README.md` | Stub → full README: prerequisites, install (local + npm labelled), 13-tool catalog, OMP specifics (pi.zod, folded guidance, ungated shutdown), three-path table, verify step, release-doc ref |
| `extensions/core/README.md` | Stub → library doc: shared library (not installed directly), public surface (findRodCli, ExecFn/ExecResult, execRodCli, 13 argv-builders, waitForSelector/sleep), consumed by pi/omp, release-doc ref |
| `README.md` | Added OMP Extension to Documentation list; added Integration Paths section with canonical D-01 three-way table + links to both READMEs + SKILL.md |
| `docs/onboarding/pi.md` | Labelled npm install "(available once published)" with local path; added OMP cross-link; kept Skill fallback |
| `skills/rod-cli/SKILL.md` | Replaced Pi-only section with Extensions section covering both OMP + Pi + Skill three-path table; labelled npm-install claims |

## Verification gates

### Gate 1 — No unpublished-install claim presented as verified
Every `install npm:` / `plugin install npm:` occurrence has an adjacent "(once published)" label + working local alternative:
- `extensions/pi/README.md:28-29` — `# Once published to npm (available once published):` + `pi install ./extensions/pi` above
- `extensions/omp/README.md:28-29` — `# Once published to npm (available once published):` + `omp plugin link ./extensions/omp` above
- `docs/onboarding/pi.md:61-62` — same pattern
- `README.md:37` — table cell: `(once published) / ... (today)`
- `extensions/pi/README.md:69` — table cell: same
- `extensions/omp/README.md:69` — table cell: same
- `skills/rod-cli/SKILL.md:141` — table cell: same

### Gate 2 — No stale flags
`grep -rn '\-\-no-banner'` across all 6 files → **empty** (exit code 1). Banner flag is `--banner` (opt-in, off by default) as documented in SKILL.md line 33.

### Gate 3 — Accurate names + catalog
OMP tool names from source (`extensions/omp/src/tools/*.ts` `name:` fields) vs OMP README table:
```
browse_click     browse_click
browse_cookies   browse_cookies
browse_eval      browse_eval
browse_fill_form browse_fill_form
browse_goto      browse_goto
browse_navigate  browse_navigate
browse_screenshot browse_screenshot
browse_scroll    browse_scroll
browse_snapshot browse_snapshot
browse_storage   browse_storage
browse_tabs      browse_tabs
browse_type      browse_type
browse_wait      browse_wait
```
**13/13 match.** Package names verified against `extensions/*/package.json`: `@agenthands/rod-cli-{core,pi,omp}`.

### Gate 4 — Three-path present
- `README.md` — "## Integration Paths" section with full D-01 table (lines 30–42)
- `docs/onboarding/pi.md` — mentions Pi extension, Skill fallback, and OMP cross-link (lines 51–76)
- `skills/rod-cli/SKILL.md` — "## Extensions" section with full three-path table (lines 134–150)

### Gate 5 — Links resolve
All relative links in all 6 files resolve to existing files/directories. Verified via script.

### Gate 6 — No accidental code touch
`git status` shows exactly the 6 doc files modified (no code). All three test suites pass:
- core: **139 passed** (vitest)
- pi: **38 passed** (vitest)
- omp: **30 passed** (bun test)

## Files changed (uncommitted)
- `README.md`
- `docs/onboarding/pi.md`
- `extensions/core/README.md`
- `extensions/omp/README.md`
- `extensions/pi/README.md`
- `skills/rod-cli/SKILL.md`
