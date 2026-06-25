---
author: engineer
responsible: engineer
phase: 29-best-effort-live-validation
plan: 01
milestone: v1.6
type: execute
status: complete
requirements: [LIVEWAF-01]
commits: [7526360, a47e949]
files_modified:
  - tests/detection_live_test.go
  - .github/workflows/test.yml
---

# Phase 29 Plan 01 — SUMMARY (live smoke test + CI exclusion)

## Objective achieved

Added the opt-in, non-blocking live smoke check behind `//go:build detection_live`
(LIVEWAF-01 criteria 1 + 2): a build-tagged test driving the real binary against
Cloudflare / DataDome / CreepJS and reporting per-target verdicts
informationally, excluded from the default test run and the push CI job by the
build tag alone.

## Must-have truths — status

- **Live file exists, build-tag first line, compiles ONLY under -tags** — DONE +
  VERIFIED. `tests/detection_live_test.go` line 1 is `//go:build detection_live`
  (Go 1.25 form; the repo's first build-tagged test).
- **Plain build/test does NOT see the file** — DONE + VERIFIED. `go build ./...`
  clean (file invisible); `go test ./tests/ -list` shows `TestDetectionHarness`
  but NOT `TestLiveDetection`.
- **Drives the real binary vs 3 targets as independently-skippable subtests,
  reporting informationally (t.Logf/t.Skip), NEVER t.Fatal/t.Errorf on a live
  detection or network error** — DONE. Subtests `cloudflare`, `datadome`,
  `creepjs`; `gotoLiveOrSkip` t.Skip's an unreachable target; `liveEvalBestEffort`
  returns `(val, ok)` and never fails the test; every verdict is `t.Logf`. There
  is no `t.Fatal`/`t.Errorf` anywhere in the file.
- **test.yml runs plain `go test ./... -count=1` with NO -tags, + an
  exclusion comment** — DONE. Gate command unchanged; a multi-line comment
  documents the deliberate exclusion and warns against adding `-tags
  detection_live`.
- **`go build -tags detection_live ./...` compiles it (valid Go, not dead code)**
  — DONE + VERIFIED. Tagged build clean; `go vet -tags detection_live ./tests/`
  clean; `go test -tags detection_live ./tests/ -list` shows `TestLiveDetection`.

## The exclusion mechanism (proof)

```
# plain — live test INVISIBLE:
$ go test ./tests/ -list '.*' | grep -i live      → (nothing)
$ go test ./tests/ -list TestDetectionHarness     → TestDetectionHarness
# tagged — live test PRESENT:
$ go test -tags detection_live ./tests/ -list 'TestLive.*' → TestLiveDetection
```

This is the load-bearing invariant: the file is invisible without the tag, so the
CI gate (`go test ./... -count=1`, no `-tags`) provably never compiles or runs it.

## Non-blocking discipline (the honesty invariant)

Non-blocking in BOTH senses required by CONTEXT:
1. **Excluded from CI** by the build tag (above).
2. **Informational within the suite:** `gotoLiveOrSkip` → `t.Skipf` on an
   unreachable target / no egress; `liveEvalBestEffort` → `(val, false)` on error,
   never failing; all verdicts → `t.Logf`. A "no challenge observed" log
   explicitly states it is NOT a pass guarantee (the TLS/IP layers are not
   exercised). A flaky third-party challenge therefore cannot redden the build.

Target URLs are pinned as constants flagged third-party / best-effort / may change.

## Why no live run here

A live green is NOT the bar (CONTEXT is explicit) and the verify env may have no
egress. The suite is built to SKIP under no egress, not fail. Exclusion +
non-blocking structure is the verifiable bar, and both are proven above without
needing real network access.

## Scope honored

`internal/detect/` Tier-1 harness unchanged; `../godoll` untouched; CI gate
command unchanged (comment-only edit to test.yml).

## Independent review (anvil-code-reviewer gate, commit a47e949)

A spawned isolated `anvil-code-reviewer` reviewed the diff before handoff
(`/tmp/anvil-spawn-out/29-REVIEW.md`). Verdict: **SHIP** — 0 blockers, 0 majors,
2 minors. It empirically confirmed all four criteria (exclusion via `go test
-list` both ways; zero `t.Fatal/Error` in code; compiles under the tag; doc names
TLS/JA3/JA4/IP/CDP and disclaims "undetectable"). Both minors accepted + fixed:
- **[MINOR] body eval error folded into the challenge heuristic** — `body, _ :=`
  discarded the ok flag, so eval error text could spuriously flip the
  informational verdict (could never redden the build — title was already ok).
  Fixed: guard with `bodyOK` and blank `body` on error.
- **[MINOR] gotoLiveOrSkip mis-attributed a not-built binary to "no egress"** —
  fixed by widening the skip message to name both causes (unreachable OR binary
  not built; the doc's run recipe already lists the build step).
Non-blocking discipline + exclusion re-verified after the fixes.
