---
author: qa
responsible: qa
phase: 28-human-behavior-tuning
milestone: v1.6
type: verification
status: complete
verdict: passed
requirements: [HUMANIZE-01]
verified_commits: [244bf45, 715b2c4, 869653d, 9a23d35]
verified_range: c833302..9a23d35
---

# Phase 28 (HUMANIZE-01) — VERIFICATION

**Verdict: PASSED.** All three success criteria are satisfied, both engineer
honesty notes are truthful and acceptable, and the load-bearing zero-regression
invariant holds. Verified goal-backward against real code (SMTC) and the real
prebuilt binary (test-run), not against SUMMARY claims.

Independent judge: qa. The engineer built it; this is the independent
correctness gate.

## Acceptance handshake (inputs verifiable?)

- Inputs complete: SUMMARY present for both PLANs (01, 02); requirement
  HUMANIZE-01 is real and consistent across CONTEXT/PLAN/SUMMARY.
- PLAN ↔ SUMMARY consistency: SMTC pre-check on the SUMMARY's claimed symbols
  (`HumanizeTuning`, `typingOpts/mouseOpts/scrollOpts`, `validateHumanizeTuning`)
  confirmed each exists AND is wired into the claimed call sites — the SUMMARY
  does not overclaim. No refusal trigger.
- Goals verifiable: all three criteria are falsifiable (SMTC structural checks +
  a real-binary timing assertion).

## Per-truth verdicts

### Criterion 1 — full threading path, values NOT silently ignored — PASSED (SMTC-grounded)

The flag→action path was walked end-to-end against real code, not greps:

- **flag → forward → capture**: `cmd.go` registers all 10 flags (verified via
  `./rod-cli --help`), forwards each IsSet-gated into the daemon argv, and adds
  each to the session-affecting set; `main.go` captures each set flag into
  `StealthFlags` as a non-nil pointer (unset → nil). (git diff, confirmed.)
- **precedence**: `ResolveStealth` (types/config.go) overlays each flag onto
  `cfg.Stealth` only when the flag pointer is non-nil — explicit flag > non-nil
  profile/yaml > unset(nil). No new resolver; existing overlay shape extended.
- **accessor**: `(*Context).HumanizeTuning()` (types/context.go:323) returns the
  resolved `StealthConfig`; godoll import kept out of `types` (narrow surface, as
  CONTEXT preferred shape (a)).
- **call sites pass the built slices** — SMTC `get_callers`, all `confidence:
  exact`:
  - `actions.Fill` → `typingOpts(ctx.HumanizeTuning())` at actions.go:370
  - `actions.Type` → `typingOpts(ctx.HumanizeTuning())` at actions.go:512
  - `actions.Click` → `mouseOpts(ctx.HumanizeTuning())` at actions.go:351
  - `actions.MouseWheel` → `scrollOpts(ctx.HumanizeTuning())` at actions.go:549
    (one slice reused across all four scroll directions)
  The variadic `opts...` spread reaches `typeWithHumanize`/`clickWithMouse`/
  `humanizeScrollBy` (the godoll wrapper vars). The values are threaded, not
  silently dropped.

All five tuning dimensions from the CONTEXT mapping are honored: typing speed +
typo rate (TypeWithHumanize), delay jitter (= the typing-speed spread, see note
a), mouse-path (tremor/steps/speed/deviation → ClickWithMouse), scroll
(duration + physics-enable → ScrollBy). Hover/DragAndDrop correctly left
untouched (fixed godoll signatures, out of scope per CONTEXT — confirmed against
godoll/humanize/actions.go).

### Criterion 2 — a configured value OBSERVABLY changes behavior on the real binary — PASSED (test-run)

`go build -o rod-cli .` then `go test ./tests/ -run TestHumanizeTuning -count=1`:
**PASS (13.88s)**. The test spawns two fresh daemon sessions over the real
binary — one default, one with `--typing-speed-min 300 --typing-speed-max 400`
(both `--typo-rate 0` to remove the high-variance typo-correction term) — and
asserts the slow path is longer by a `>= 1s` margin. A silently-ignored knob
would clock identically and FAIL. The knob measurably moves real behavior.

### Criterion 3 — CI-pinned speeds stay fast; no default-path regression — PASSED (test-run + structural)

- Existing `tests/humanize_test.go` is **byte-for-byte UNCHANGED** (not in the
  phase diff — structural proof).
- `TestHumanizedInteractions` re-run against the fresh binary: **PASS (10.56s)**,
  consistent with its pre-phase wall-clock — no regression.
- `TestHumanizeTuning` also asserts the default (no-flag) path stays `>= 300ms`
  (the existing humanize bound) — the tuning surface adds no default-path drift.
- Structural guarantee: all-nil knobs ⇒ empty option slices (each builder
  appends only behind a `!= nil` guard) ⇒ the variadic call is identical to the
  pre-phase call. The zero-regression invariant holds by construction.

## Zero-regression invariant (load-bearing) — CONFIRMED

A knob "set but silently ignored" is the failure this phase must not have. It is
excluded two ways: (1) criterion-2 test would fail if the knob were inert; (2)
the builders are pure functions of the nil-ness of each field — an unset knob
appends nothing. An incomplete min/max pair is NOT silently dropped — it is a
hard error at spawn (`validateSpeedPair`), which is stronger than the CONTEXT's
"emit nothing" suggestion and is the right call.

## Engineer honesty notes — JUDGED (not rubber-stamped)

**(a) "delay jitter" = the typing-speed min/max spread.** TRUE. godoll's
typing.go exposes only `WithTypoRate` + `WithTypingSpeed(min,max)` — no separate
jitter option exists. Documented in the `--typing-speed-min/max` flag Usage
("the spread IS the delay jitter"). Acceptable; does not undermine criterion 1 —
the jitter dimension is honored through the real spread mechanism.

**(b) `--scroll-physics=false` cannot disable physics.** TRUE and honestly
surfaced. Confirmed against godoll: `WithPhysics()` (scroll.go:43-48) only sets
`UsePhysics=true`, and `ApplyScrollDefaults` (scroll.go:35-37) forces physics on
by default — there is no disable option in godoll. `scrollOpts` emits
`WithPhysics()` only for an explicit `true`; false/nil emits nothing and
godoll's default physics still applies. Documented in BOTH the flag Usage
("godoll default; cannot be disabled via flag in v1.6") and the `ScrollPhysics`
struct doc.

**Does (b) undermine criterion 1 for the scroll dimension?** No. The scroll
dimension IS tunable: `--scroll-duration` is fully threaded and observably
effective, and physics-enable works. Only the disable *direction* is
unavailable — an upstream godoll constraint, properly surfaced, not a
silent-ignore. `--scroll-physics=false` is handled deterministically and its
no-op is documented with the reason. This is honest documentation of a real
limit, correctly deferred (a godoll signature change is out of v1.6 scope per
CONTEXT). Acceptable.

## CR resolutions — JUDGED sound

The anvil-code-reviewer's [MAJOR] godoll rand-panic finding is real and the fix
is correct: godoll's `rand.RandomDuration`/`RandomInt` PANIC on negative or
min>max input (confirmed at godoll/internal/rand/rand.go:20-28).
`validateHumanizeTuning` (wired into `ResolveStealth` at config.go:367, SMTC
`confidence: exact`) establishes the precondition `0 <= min <= max` and range
bounds at the spawn seam — fail-fast refusal instead of a per-keystroke panic in
a frozen daemon. Locked by `TestResolveStealth_HumanizeTuningValidation` (12 bad
cases + valid + unset-stays-nil). Sound: the godoll option precondition is
discharged once at config resolution, not re-risked per action.

## Evidence ledger

| Check | Tier | Result |
|---|---|---|
| Flag → forward → capture → resolve path | git diff (read) | wired, all 10 flags |
| Call sites pass option slices | SMTC get_callers (exact) | Fill/Type/Click/MouseWheel confirmed |
| validateHumanizeTuning in ResolveStealth | SMTC get_callers (exact) | confirmed at config.go:367 |
| godoll WithPhysics enable-only | godoll source (read) | confirmed scroll.go:35-48 |
| godoll rand panic precondition | godoll source (read) | confirmed rand.go:20-28 |
| `go build ./...` + `go vet` | build-run | clean |
| `go test ./types/` (resolve + roundtrip + validation) | test-run | PASS |
| `go test ./actions/` | test-run | PASS (193s) |
| `--help` shows all 10 flags + honest usage | binary-run | confirmed |
| TestHumanizeTuning (criterion 2) | test-run, real binary | PASS (13.88s) |
| TestHumanizedInteractions (criterion 3) | test-run, real binary | PASS (10.56s) |
| humanize_test.go unchanged | git diff (structural) | not in diff |

```gate_result
id: gate-28-verify
phase: 28-human-behavior-tuning
verdict: pass
requirement: HUMANIZE-01
basis: >
  All 3 criteria pass. Criterion 1 threading SMTC-confirmed end-to-end (all call
  sites confidence:exact). Criterion 2 observable on real binary (TestHumanizeTuning
  PASS, >=1s margin). Criterion 3 no regression (humanize_test.go unchanged,
  TestHumanizedInteractions PASS, all-nil => empty slices by construction). Both
  honesty notes truthful and acceptable; CR rand-panic fix sound.
evidence_tiers: [smtc_exact, test_run_real_binary, godoll_source_read, git_structural]
accountable_owner: qa
remediation_owner: null
```
