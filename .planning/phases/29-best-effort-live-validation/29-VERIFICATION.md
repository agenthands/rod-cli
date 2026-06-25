---
author: qa
responsible: qa
phase: 29-best-effort-live-validation
milestone: v1.6
type: verification
status: complete
verdict: passed
requirements: [LIVEWAF-01]
verified_commits: [7526360, 25a1b90, a47e949, 17c600a]
verified_range: 3eafed6..17c600a
---

# Phase 29 (LIVEWAF-01) — VERIFICATION

**Verdict: PASSED.** All three success criteria meet the verifiable bar —
EXCLUSION (by construction) + NON-BLOCKING (within the suite) + HONESTY (the
documented ceiling). Verified entirely offline (no network egress); a live green
was neither required nor attempted, per CONTEXT's explicit framing that a live
green is NOT the bar.

Independent judge: qa. The engineer built it; this is the independent
correctness gate. This is the last build phase before milestone v1.6 close.

## Acceptance handshake (inputs verifiable?)

- Inputs complete: SUMMARY present for both PLANs (01, 02); LIVEWAF-01 consistent
  across CONTEXT/PLAN/SUMMARY.
- PLAN ↔ SUMMARY consistency: the SUMMARY's central claims (build-tag first line;
  no t.Fatal/Errorf; doc names TLS/IP/CDP + disclaims "undetectable") were each
  re-derived against the real file/toolchain below — the SUMMARY does not
  overclaim. No refusal trigger.
- Goals verifiable offline: the bar is provable via `go test -list` (both ways),
  `go build -tags`, grep, and reading the doc — no egress dependency. Confirmed.

## Per-truth verdicts

### Criterion 1 — Excluded by construction — PASSED (toolchain-run)

The build-tag exclusion is mechanically proven both directions:

- `tests/detection_live_test.go` line 1 is exactly `//go:build detection_live`
  (Go 1.25 form; repo's first build-tagged test).
- `go test ./tests/ -list '.*'` (plain, no tag): lists `TestDetectionHarness`,
  **OMITS** `TestLiveDetection`. The live test is invisible without the tag.
- `go test -tags detection_live ./tests/ -list 'TestLive.*'`: **INCLUDES**
  `TestLiveDetection`.
- `go build -tags detection_live ./...`: compiles clean — the file is valid Go,
  not just excluded dead code.
- `go build ./...` (plain): clean (file invisible). `go vet ./tests/` and
  `go vet -tags detection_live ./tests/` BOTH clean — the first build-tagged file
  does not break the toolchain (the CONTEXT caution).
- `.github/workflows/test.yml`: gate command is plain `go test ./... -count=1`
  with **NO `-tags`** (line 42), preceded by a 10-line "DELIBERATE EXCLUSION
  (LIVEWAF-01)" comment (lines 33-41) that marks the exclusion "load-bearing,"
  explains the flakiness rationale, and explicitly says "Do NOT add `-tags
  detection_live` here." Gate command unchanged (comment-only edit).

The CI gate provably never compiles or runs the live suite.

### Criterion 2 — Non-blocking within the suite — PASSED (source-inspection, grounded)

Read the full live test source (149 lines) AND grep-confirmed the exact bar:

- `grep -E "t\.(Fatal|Fatalf|Error|Errorf|FailNow|Fail)\b"` on the file returns
  exactly ONE match — line 19 — which is a **doc-comment** ("NEVER calls
  t.Fatal/t.Errorf …"), exactly as the architect noted. There is **zero**
  t.Fatal/t.Errorf/t.Error/t.FailNow/t.Fail in executable code.
- Every outcome is reported via `t.Logf` (per-target verdicts) or `t.Skipf`
  (unreachable target / no egress / binary not built). `liveEvalBestEffort`
  returns `(val, ok)` and never fails the test; `gotoLiveOrSkip` t.Skipf's on a
  navigation error.
- A live detection (`challenged` true) is `t.Logf`, not a failure. A "no
  challenge observed" branch explicitly logs that it is NOT a pass guarantee
  ("TLS/IP layers are not exercised here").

A flaky third-party challenge or an unreachable target therefore cannot redden
the build. CR fixes (commit a47e949) present and correct in source: the `bodyOK`
guard (lines 96-98) prevents eval-error text feeding the heuristic; the widened
`gotoLiveOrSkip` skip message (lines 72-74) names both causes (unreachable OR
binary-not-built) without asserting a cause not established.

### Criterion 3 — Honest ceiling documented — PASSED (doc-read, JUDGED)

`docs/stealth-validation.md` (108 lines) is a genuinely honest doc, judged
against the crux test ("a doc implying passes-Cloudflare ⇒ undetectable FAILS"):

- **Names all three uncontrollable layers** with a sentence each on *why* a
  JS-injecting CLI cannot close them: **TLS fingerprint (JA3/JA4)** (line 84),
  **IP reputation** (line 88), **CDP transport footprint** (line 91, cross-ref
  to docs/cdp-footprint.md).
- **"undetectable" appears ONLY as a negation** — all three occurrences confirmed
  by grep: TL;DR "does NOT claim to be 'undetectable'" (line 3), "does **not**
  mean 'undetectable'" (line 82), "No 'undetectable' guarantee" (line 97). No
  positive use anywhere.
- **Actively refutes the overclaim** (the inverse of a gap): lines 99-100 state
  "passing Cloudflare on one run, on one IP, at one moment is a best-effort data
  point — not proof you are invisible." The doc does the opposite of implying
  "passes Cloudflare ⇒ undetectable."
- **Tier-1-vs-Tier-2 split** clearly stated: Tier-1 (offline, blocking) enumerates
  the JS-layer signals it deterministically proves (UA/CH/platform/WebGL/canvas/
  audio/WebRTC/webdriver/...) per Phases 24-27; Tier-2 (opt-in, non-blocking)
  framed as best-effort informational.
- **How to run** with the mandatory `go build -o rod-cli .` first (lines 65-68),
  noting outcomes are informational/non-blocking.
- **Discoverable**: linked from README.md (line 19) with an honest one-line
  description that itself names the no-"undetectable" ceiling. Cross-refs
  resolve: docs/cdp-footprint.md exists, ../ARCHITECTURE.md exists.

## Out-of-scope honored

`internal/detect/` Tier-1 harness unchanged; `../godoll` untouched; no new
evasion capability added (validate + document only); CI gate command unchanged.
Confirmed via the diff (only 6 files: live test, test.yml comment, doc, README
link, 2 SUMMARYs).

## Note on no live run

Per CONTEXT, a live green is flaky and explicitly NOT the bar, and the verify env
may lack egress. No live run was attempted as a pass condition — correct. The
suite is structured to SKIP (not fail) under no egress, which IS the verifiable
discipline and was confirmed by source inspection.

## Evidence ledger

| Check | Tier | Result |
|---|---|---|
| `//go:build detection_live` first line | file-read | confirmed |
| plain `-list` omits live, includes harness | toolchain-run | confirmed |
| tagged `-list` includes TestLiveDetection | toolchain-run | confirmed |
| `go build -tags detection_live ./...` compiles | toolchain-run | clean |
| `go build ./...` (plain) | toolchain-run | clean (file invisible) |
| `go vet` plain + tagged | toolchain-run | both clean |
| no t.Fatal/Errorf in executable code | grep + full-read | only line-19 doc-comment |
| informational/skip-only reporting | source-inspection | t.Logf / t.Skipf only |
| test.yml plain gate, no -tags, exclusion comment | file-read | confirmed |
| doc names TLS/JA3/JA4, IP, CDP | grep + read | all present |
| "undetectable" only as negation | grep | 3/3 negations |
| doc refutes passes-Cloudflare⇒undetectable | doc-read (judged) | actively refuted |
| README link + cross-refs resolve | grep + file-exists | confirmed |

```gate_result
id: gate-29-verify
phase: 29-best-effort-live-validation
verdict: pass
requirement: LIVEWAF-01
basis: >
  All 3 criteria meet the EXCLUSION+HONESTY bar (live green not required). C1
  exclusion toolchain-proven both ways (plain -list omits, tagged -list includes,
  tagged build compiles, vet clean, test.yml plain gate + load-bearing exclusion
  comment). C2 non-blocking: zero t.Fatal/Errorf in code (only a line-19
  doc-comment), t.Logf/t.Skipf only. C3 honesty: doc names TLS/JA3/JA4 + IP + CDP,
  "undetectable" only as negation (3/3), actively refutes the passes-Cloudflare
  overclaim, README-linked, cross-refs resolve. CR fixes present and correct.
evidence_tiers: [toolchain_run, source_inspection_grounded, doc_read_judged, grep, git_structural]
accountable_owner: qa
remediation_owner: null
```
