---
author: architect
responsible: architect
phase: 29-best-effort-live-validation
milestone: v1.6
status: ready_for_planning
parent_artifacts:
  - .planning/ROADMAP.md
  - .planning/REQUIREMENTS.md
---

# Phase 29: Best-Effort Live Validation - Context

**Gathered:** 2026-06-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Add an **opt-in, non-blocking** live smoke check against real anti-bot challenges
(Cloudflare / DataDome / CreepJS) behind a `//go:build detection_live` tag, and
document the **honest ceiling** of what rod-cli's stealth can and cannot prove
(LIVEWAF-01). This is the Tier-2 best-effort counterpart to the Tier-1 offline
detection harness from Phase 24 (`internal/detect/` + `tests/detection_test.go`).

The three success criteria, and the **verifiable bar for each** (important — the
bar is EXCLUSION + HONESTY, never a green live result, which is flaky by nature):

1. **Excluded by construction.** A live smoke check exists behind
   `//go:build detection_live` and is excluded from the default test run AND the
   push CI job. Verifiable: the file compiles ONLY with `-tags detection_live`;
   a plain `go test ./...` does not compile/run it; `.github/workflows/test.yml`
   runs plain `go test ./... -count=1` and never passes `-tags detection_live`.
2. **Manually runnable, informational.** A developer can run the suite with the
   build tag to get an informational pass/fail against the three targets, with
   **failures reported as non-blocking**. Verifiable: the suite compiles under
   the tag and is structured to report per-target verdicts informationally
   (log/skip, never a hard gate). Actually hitting live Cloudflare is best-effort
   and may have no egress in the verify env — a live green is NOT the bar.
3. **Honest ceiling documented.** Docs state what Tier-1 proves (JS-layer
   fingerprint signals, read back from the live page, deterministic offline) vs
   what the live suite is best-effort about (the **TLS/IP/CDP layers a
   JS-injecting CLI cannot control**), with **no "undetectable" guarantee**.

Out of scope: making the live suite pass/block CI (the whole point is it does
NOT gate); any new evasion capability (this phase validates + documents, it does
not harden — hardening was Phases 26/27); changing the Tier-1 harness.

</domain>

<decisions>
## Implementation Decisions

### The live smoke test (tests/)
- New file `tests/detection_live_test.go` with `//go:build detection_live` as the
  FIRST line (Go 1.25 syntax; this is the repo's first build-tagged test — model
  the build-tag form, not the legacy `// +build`).
- Drive the REAL binary via the existing `runCli` helper (the same pattern
  `tests/detection_test.go` uses): `runCli("goto", <target-url>)` then read a
  verdict via `eval` / page state. One subtest per target (Cloudflare, DataDome,
  CreepJS), each independently skippable.
- **Non-blocking discipline (the honesty invariant):** a *detected* or
  *unreachable* outcome is **informational** — report it with `t.Logf` (and
  `t.Skip`/`t.Skipf` when the target is unreachable / no network egress), NEVER
  `t.Fatal`/`t.Errorf` that would turn a flaky third-party challenge into a red
  build. The developer runs this for information, not a pass/fail gate. Engineer's
  discretion on the exact log/skip shape, but it MUST NOT hard-fail on a live
  detection or a network error.
- Pin the target URLs as named constants with a comment that they are
  third-party and may change/disappear (best-effort by nature).

### CI exclusion (verify, don't change the gate)
- `.github/workflows/test.yml` already runs `go test ./... -count=1` with no
  `-tags`, so the tagged file is excluded automatically — **do not add the tag to
  CI**. Add a short comment in `test.yml` documenting the deliberate exclusion
  (so a future maintainer does not "helpfully" add `-tags detection_live`).
- Optional belt-and-suspenders: a tiny always-compiled meta-check asserting the
  live file carries the build tag (so the exclusion can't silently rot). Engineer's
  discretion — nice-to-have, not required.

### Honest-ceiling documentation (the heart of this phase)
- Author `docs/stealth-validation.md` (new) — the Tier-1-vs-Tier-2 honesty doc:
  - **Tier-1 (offline harness, blocking CI):** what it deterministically proves —
    the JS-layer fingerprint signals read back from the live page (UA/CH/platform/
    WebGL/canvas/WebRTC/webdriver/plugins), per Phases 24–27.
  - **Tier-2 (this live suite, opt-in, non-blocking):** best-effort informational
    signal against real WAFs; explicitly subject to TLS-fingerprint (JA3/JA4),
    IP-reputation, and CDP-footprint layers a **JS-injecting CLI cannot control**
    (cross-reference `docs/cdp-footprint.md`).
  - **No "undetectable" guarantee** — state the ceiling plainly; this is the
    project's validate-live-not-source / no-overclaim ethos.
  - **How to run:** `go test -tags detection_live ./tests/ -run TestLive... -v`.
- Link the new doc from `README.md` (the Documentation list) and/or
  `ARCHITECTURE.md` stealth section. The codebase-archaeologist canonicalizes
  `docs/` at milestone close; this phase authors the honest-ceiling content as the
  LIVEWAF-01 deliverable (criterion 3 requires it to exist now).

### Claude's Discretion
- Exact target URLs + subtest names; the verdict-read mechanism per target
  (Cloudflare/DataDome present a challenge page; CreepJS exposes a score in DOM).
- Whether to add the meta-check; doc file split vs. a single doc.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `tests/detection_test.go` — the Tier-1 model: `runCli("goto", url)` + `eval`
  reads, ready-gating (`window.__detect.ready`), per-signal subtests. Mirror its
  binary-driving shape (NOT its offline httptest server — live uses real URLs).
- `tests/server.go` / the `runCli` helper — drives the prebuilt `../rod-cli`.
- `internal/detect/` (detect.html/js, server.go, embed.go) — Tier-1 offline
  fixture; the live suite is its Tier-2 sibling, NOT a modification of it.
- `docs/cdp-footprint.md` — existing honesty doc about the CDP layer; cross-ref it.
- `docs/ARCHITECTURE.md`, `README.md` (Documentation list ~:18) — link targets.

### Established Patterns / Traps
- `.github/workflows/test.yml`: `go test ./... -count=1` is the blocking gate;
  it builds the binary first then installs Chromium. Plain `go test ./...` skips
  build-tagged files automatically — that IS the exclusion mechanism.
- Operational traps (carry forward): `go build -o rod-cli .` before any runCli
  test; `pkill -9 -x rod-cli` (never `-f`); never kill the user's Chromium; never
  run full `go test ./...` locally (timeout + flaky daemon_more_test.go) — run
  targeted, and for THIS suite, `go test -tags detection_live ./tests/ -run ...`.
- This is the repo's FIRST build-tagged test — verify `go vet ./...` and a plain
  `go test ./tests/ -run TestDetectionHarness` still behave (tag absent ⇒ live
  file invisible).

### Integration Points
- Verify exclusion: `go build ./...` (no tag) must NOT compile the live file;
  `go build -tags detection_live ./...` must compile it. `test.yml` unchanged
  except an explanatory comment.

</code_context>

<specifics>
## Specific Ideas

- The honesty ceiling is the deliverable's point, not garnish: a doc that implies
  "passes Cloudflare ⇒ undetectable" would FAIL criterion 3. Name the
  TLS/IP/CDP layers rod-cli cannot touch explicitly.
- Non-blocking means non-blocking in BOTH senses: excluded from CI (build tag)
  AND informational within the suite (log/skip, never t.Fatal on a live result).

</specifics>

<deferred>
## Deferred Ideas

- Automating the live suite on a schedule (cron CI with continue-on-error) —
  possible future, explicitly NOT in v1.6 (would reintroduce flaky third-party
  network into the project's signal).
- TLS-fingerprint / IP-reputation mitigation — outside a JS-injecting CLI's
  control; documented as the ceiling, not attempted.

</deferred>
