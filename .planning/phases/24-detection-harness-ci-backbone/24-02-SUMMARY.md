---
phase: 24-detection-harness-ci-backbone
plan: 02
subsystem: stealth
tags: [validate-03, loud-failure, stealth, evasion, stderr]

# Dependency graph
requires: []
provides:
  - "types/context.go createPage() surfaces fingerprint-generation and EvasionManager.Apply() errors as warning: lines on stderr (no silent no-op)"
affects: [24-03, 24-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "fmt.Fprintf(os.Stderr, \"warning: ...\") as the repo's loud-failure convention (no log.* precedent; lightest convention-consistent choice)"
    - "Warnings to stderr only — stdout stays clean for --raw/piped output"

key-files:
  created: []
  modified:
    - types/context.go

key-decisions:
  - "Renamed inner fingerprint error to fpErr so the outer page-creation err returned at end-of-function is never clobbered (avoids the shadowing trap the planner flagged)"
  - "Log-and-continue, not hard-fail — daemon keeps running on evasion error per VALIDATE-03 / 24-CONTEXT (requirement is 'no silent no-op', not 'abort')"
  - "fmt.Fprintf(os.Stderr) chosen over a structured logger — no log.* precedent in the file; fmt + os already imported, so no new dependency"

patterns-established:
  - "Loud-failure for swallowed errors: replace `_ = x()` / `if err == nil` no-ops with `if err := x(); err != nil { fmt.Fprintf(os.Stderr, \"warning: ...\\n\", err) }`"

requirements-completed: [VALIDATE-03]

metrics:
  duration: ~5m
  completed: 2026-06-23
  tasks: 1
  files-changed: 1

status: complete
---

# Phase 24 Plan 02: VALIDATE-03 Loud-Failure for Swallowed Evasion Errors Summary

Made the two silently-swallowed evasion errors in `types/context.go` `createPage()` observable: fingerprint-generation failures and `EvasionManager.Apply()` failures now each write a short `warning:` line to stderr instead of taking a silent no-op path, without hard-failing the daemon.

## What Was Built

In `createPage()` (~lines 286–296), two error-swallowing paths were replaced:

1. **Fingerprint generation** — the old `fp, err := fg.Generate(); if err == nil && fp != nil { ... }` silent no-op became `fp, fpErr := fg.Generate()` with an explicit `else if fp != nil` success branch and a `warning: fingerprint generation failed: %v` stderr line on the error branch.
2. **Evasion Apply** — `_ = em.Apply()` became `if err := em.Apply(); err != nil { fmt.Fprintf(os.Stderr, "warning: evasion Apply failed: %v\n", err) }`.

The success path is unchanged (`ctx.fingerprint = fp`, `em.SetFingerprint(fp)`), and the daemon does not abort on evasion error — it logs and continues. Because the e2e `runCli` helper folds stderr into its captured buffer, plan 24-03's harness can assert the warning is observable when evasion fails.

## Shadowing Fix

The planner flagged a shadowing trap: the inner fingerprint error originally reused the name `err`, which is also the outer page-creation error returned at end-of-function via `errors.Wrap(err, "create page failed")`. The inner fingerprint error was renamed to `fpErr` so the outer `err` is never touched inside the block; the `Apply()` error is scoped to its own `if` statement. The end-of-function `errors.Wrap(err, ...)` still refers to the page-creation error.

## Deviations from Plan

None — plan executed exactly as written. No new imports needed (`fmt` and `os` were already imported). No structured logger added (no `log.*` precedent in the file).

## Verification

- `go build ./...` — passes.
- `grep -c 'Fprintf(os.Stderr, "warning:' types/context.go` == 2 (both error paths surfaced).
- `grep '_ = em.Apply()' types/context.go` — returns nothing (swallow removed).
- `go test ./types/...` — `ok` (no behavior change on the success path).
- Warnings go to stderr only; stdout unchanged (no `--raw`/pipe pollution).

## Commits

- `7153658` — feat(24-02): surface fingerprint + Apply evasion errors to stderr

## Self-Check: PASSED

- FOUND: types/context.go
- FOUND commit: 7153658
