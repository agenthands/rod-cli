---
phase: 24-detection-harness-ci-backbone
plan: 04
subsystem: ci
tags: [ci, github-actions, cleanup, gitignore, cdp-findings, harness-03]

# Dependency graph
requires:
  - phase: 24-03
    provides: "tests/detection_test.go e2e harness the CI job runs"
provides:
  - ".github/workflows/test.yml: repo's first test CI job — go test ./... on push + PR to main, go 1.25.x, builds binary + installs Chromium, offline"
  - "Clean repo tree: tracked tests/rod/ browser profile (1185 files) + tests/interaction_test.go.orig removed; stray untracked artifacts deleted and gitignored"
  - "docs/cdp-footprint.md: informational CDP-tell ceiling findings note (YES/NO; fix deferred to v2 CDP-01)"
affects: [25-stealth-config, 26-fingerprint-validator, 27-hardening, 28-humanize, 29-livewaf]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "First test CI gate: go test ./... on push + PR to main with go-version pinned to go.mod (1.25.x), builds ../rod-cli + installs Chromium before the offline harness"
    - "Anchored .gitignore rules for root build/test strays (/rod-cli, /state.json, ...) consolidated with existing *.out/*.log globs (no duplication)"

key-files:
  created:
    - .github/workflows/test.yml
    - docs/cdp-footprint.md
  modified:
    - .gitignore
  removed:
    - tests/rod/ (1185-file accidental Chromium user-data-dir)
    - tests/interaction_test.go.orig

key-decisions:
  - "CI go-version pinned to 1.25.x to match go.mod 1.25.1 — explicitly NOT release.yml's stale 1.23, or the harness package won't compile"
  - "Explicit `./rod-cli install` Chromium step so the first detection test isn't slowed/flaky by a mid-test download"
  - "Headless-only blocking gate; headful (ROD_HEADFUL=1) stays local/opt-in, not in CI"
  - "Cleanup landed as a standalone commit (38eda6a) before the session-limit interruption; CI workflow + CDP note completed and committed in 410379b on resume"

patterns-established:
  - "CDP footprint is MEASURED (window.__detect.cdpTell informational probe), not suppressed — honest-ceiling documentation over over-claiming"

requirements-completed: [HARNESS-03]

# Metrics
duration: 12min
completed: 2026-06-24
status: complete
---

# Plan 24-04 Summary — Test CI Backbone, Cleanup & CDP Note

Landed **HARNESS-03** and closed Phase 24.

## What shipped

- **`.github/workflows/test.yml`** — the repo's first test CI job. Runs `go test ./... -count=1` on **push + PR to `main`**, `go-version: 1.25.x` (matches `go.mod` 1.25.1, not release.yml's stale 1.23), builds `rod-cli` so the e2e `runCli` can drive `../rod-cli`, and runs `./rod-cli install` to fetch Chromium up front. The detection harness is fully offline (`internal/detect` fixture on loopback) — zero external-network steps, so the gate is deterministic.
- **Cleanup (commit `38eda6a`)** — `git rm` of the accidentally-committed 1185-file `tests/rod/` Chromium profile and `tests/interaction_test.go.orig`; deleted untracked strays (`rod-cli`, `test_rod`, `state.json`, `init_output.json`, `fix_test.patch`, `tests/coverage.out`, `tests/log/`); extended `.gitignore` with anchored rules consolidated against existing `*.out`/`*.log` globs.
- **`docs/cdp-footprint.md`** — concise findings note: CDP/`Runtime.enable` footprint is **not** meaningfully hideable in v1.6's JS layer (the daemon needs Runtime/Network domains for console/request logging). This phase MEASURES it (`window.__detect.cdpTell`, non-blocking); reducing it is deferred to v2 (CDP-01). No "undetectable" claim.

## Verification

- `go build ./...` passes; all three task acceptance gates green (valid YAML, go 1.25 + push/PR-to-main + `go test`; stray artifacts gone; CDP note documents the ceiling).
- Working tree clean of all listed strays.

## Notes / deviation

- **Session-limit interruption + resume:** Task 1 (cleanup) committed as `38eda6a` before the executor hit a session limit. On resume, the orchestrator completed Tasks 2–3 (`test.yml` + `docs/cdp-footprint.md`, committed `410379b`) directly and wrote this SUMMARY — no re-execution of the already-committed cleanup. All three tasks' acceptance gates verified green.
