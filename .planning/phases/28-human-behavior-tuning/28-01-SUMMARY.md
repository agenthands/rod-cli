---
author: engineer
responsible: engineer
phase: 28-human-behavior-tuning
plan: 01
milestone: v1.6
type: execute
status: complete
requirements: [HUMANIZE-01]
commits: [244bf45, 869653d]
files_modified:
  - types/config.go
  - cmd.go
  - main.go
  - types/context.go
  - actions/actions.go
---

# Phase 28 Plan 01 — SUMMARY (implementation)

## Objective achieved

Threaded godoll's existing `humanize.With*` options through the rod-cli config
spine and into the `actions/actions.go` call sites, tunable via CLI flags and the
named JSON profile (HUMANIZE-01). `actions.go` previously invoked
`TypeWithHumanize`/`ClickWithMouse`/`ScrollBy` with defaults only; the resolved
`cfg.Stealth` humanize knobs now reach those call sites, while an unset knob emits
no option (godoll default ⇒ zero regression).

## The threading path (mirrors phases 25–27)

```
--flag (cmd.go def + IsSet-gated forward) → daemon argv
  → main.go capture into StealthFlags (*T, only when IsSet)
    → ResolveStealth precedence (explicit flag > non-nil yaml/profile > nil)
      → Config.Stealth (frozen at NewContext)
        → Context.HumanizeTuning() accessor
          → actions.{typingOpts,mouseOpts,scrollOpts}() builders
            → variadic godoll call site
```

## Must-have truths — status

- **Pointer-typed tuning fields on StealthConfig** — DONE. `TypingSpeedMin/Max`
  (`*int`), `TypoRate` (`*float32`), `MouseTremor` (`*bool`), `MouseSteps`
  (`*int`), `MouseSpeedMin/Max` (`*int`), `MouseDeviation` (`*float64`),
  `ScrollDuration` (`*int`), `ScrollPhysics` (`*bool`). nil = unset = godoll
  default applies.
- **Settable via CLI flag, forwarded, captured, precedence-resolved** — DONE.
  Flags `--typing-speed-min/max`, `--typo-rate`, `--mouse-tremor`,
  `--mouse-steps`, `--mouse-speed-min/max`, `--mouse-deviation`,
  `--scroll-duration`, `--scroll-physics` (cmd.go); IsSet-gated forward into the
  daemon argv + session-affecting set; main.go captures each set flag as a
  non-nil pointer; ResolveStealth applies explicit-flag > non-nil-profile/yaml >
  nil.
- **Round-trips through a saved JSON profile** — DONE. Every field carries
  matching `yaml:` + `json:` tags, so the named-JSON-profile persists/loads the
  humanize knobs through the same `StealthConfig` (de)serialization as the
  Phase-26/27 fields.
- **actions.go threads resolved tuning into the variadic call sites** — DONE.
  `typingOpts` → `TypeWithHumanize` (Fill + Type), `mouseOpts` → `ClickWithMouse`
  (Click), `scrollOpts` → `ScrollBy` (MouseWheel, all four directions). Built
  from `ctx.HumanizeTuning()`.
- **All-nil ⇒ empty slices ⇒ byte-for-byte default (zero regression)** — DONE +
  VERIFIED. `tests/humanize_test.go` passes UNCHANGED (exit 0) against the fresh
  binary.
- **Option emitted only when non-nil; pair only when both ends set** — DONE.
  `typingOpts`/`mouseOpts` guard `WithTypingSpeed`/`WithMouseSpeed` on
  `Min != nil && Max != nil`; each scalar option guarded on its own `!= nil`.

## Reasoning frame (Hoare)

Per builder, the postcondition Q is "the returned slice contains exactly the
godoll options whose backing knobs are explicitly set". P is "the StealthConfig
is the frozen, resolved session config (each field nil-or-set)". The guard
`field != nil` before each `append` establishes Q's "only when set" conjunct;
the pair guard `min != nil && max != nil` establishes the pair conjunct. The
empty-slice case (all nil) is the zero-regression invariant: the variadic call
`f(args)` with no options is identical to the pre-phase call.

## Scope constraints honored

- `humanize.Hover` and `humanize.DragAndDrop` call sites UNCHANGED (fixed godoll
  signatures, no variadic opts — out of scope per CONTEXT).
- `../godoll` NOT modified — only existing options threaded.

## Honesty notes (documented, not invented)

- **"delay jitter" has no godoll option.** It is the variance produced by the
  `WithTypingSpeed(min,max)` spread; documented in the `--typing-speed-*` flag
  Usage strings. No separate jitter flag was invented.
- **`--scroll-physics=false` cannot disable physics.** godoll's `WithPhysics()`
  can only ENABLE physics; physics is godoll's own default and there is no
  disable option. `scrollOpts` emits `WithPhysics()` only for an explicit true; a
  false/nil value emits nothing and godoll's default physics still applies.
  Disabling physics would require a godoll signature change (out of scope for
  v1.6). The flag Usage string and the `ScrollPhysics` doc comment both state
  this; the `*bool` is kept for round-trip symmetry and forward-compat.

## Verification run at execute time

- `go build ./...` clean; `go vet ./types ./actions ./tests` clean.
- `go test ./types/` ok (incl. `TestResolveStealth*`).
- `go test ./actions/` ok.
- `go test ./tests/ -run TestHumanizedInteractions` ok (existing humanize test,
  unchanged — zero-regression evidence).

## Independent review (anvil-code-reviewer gate, commit 869653d)

A spawned, isolated `anvil-code-reviewer` reviewed the diff before handoff
(`/tmp/anvil-spawn-out/28-REVIEW.md`). Verdict: 0 blockers, 2 majors, 2 minors.
The core nil/precedence/zero-regression wiring was confirmed correct; all four
findings were accepted and resolved:

- **[MAJOR robustness] godoll rand panic on unvalidated min/max & negative
  input.** `--typing-speed-min 200 --typing-speed-max 100` (or any negative
  value) reached `humanize.WithTypingSpeed` → `rand.RandomDuration(min,max)`,
  which PANICS on negative or min>max (`../godoll/internal/rand/rand.go:20-28`);
  same for `WithMouseSpeed` → `RandomInt`. The panic fires per-keystroke inside a
  frozen daemon session — an opaque, unrecoverable break of every
  type/fill/mouse action. **Resolved:** `validateHumanizeTuning` runs in
  `ResolveStealth` (the fail-fast spawn seam, mirroring
  `deriveAndValidateFingerprint`): speed pairs require `0 <= min <= max`,
  `typo-rate`/`mouse-deviation` ∈ [0,1], `mouse-steps`/`scroll-duration`
  positive. A bad value now refuses the daemon spawn with a clear error instead
  of panicking. Locked by `TestResolveStealth_HumanizeTuningValidation`.
- **[MINOR usability] incomplete min/max pair silently dropped.** Folded into the
  MAJOR fix: an incomplete pair (only one end set) is now a hard error
  (`incomplete --typing-speed range — set both …`), consistent with the existing
  "incomplete screen geometry" rejection — stronger than the suggested warning.
- **[MINOR doc] `HumanizeTuning()` immutability comment was false** (shallow copy
  aliases the pointer knobs). **Resolved:** comment softened to "value fields
  copied; pointer knobs read-only by convention" (the builders only deref).

Foundation note (Hoare): the panic was a violated precondition of the godoll
option (`P_godoll: 0 <= min <= max`). The fix establishes that precondition at
the config-resolution boundary, so every option the builders emit downstream is
known-valid — `wp(builder, godoll-call-safe)` is discharged once at spawn rather
than re-risked per keystroke.
