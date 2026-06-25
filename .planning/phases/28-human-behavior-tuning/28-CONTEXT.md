---
author: architect
responsible: architect
phase: 28-human-behavior-tuning
milestone: v1.6
status: ready_for_planning
parent_artifacts:
  - .planning/ROADMAP.md
  - .planning/REQUIREMENTS.md
---

# Phase 28: Human-Behavior Tuning - Context

**Gathered:** 2026-06-25
**Status:** Ready for planning

<domain>
## Phase Boundary

Thread godoll's existing `humanize.With*` options through `actions/actions.go`
(which currently invokes the humanize action wrappers with **defaults only**) so
a user can tune human-like interaction via CLI flags or the stealth profile
(HUMANIZE-01). Five tuning dimensions, mapped to the real godoll option surface:

| Dimension (requirement) | godoll option(s) | action site |
|---|---|---|
| typing speed | `humanize.WithTypingSpeed(min, max int)` | `TypeWithHumanize` |
| typo rate | `humanize.WithTypoRate(rate float32)` | `TypeWithHumanize` |
| delay jitter | the `WithTypingSpeed(min,max)` **spread** (no separate jitter option exists in godoll) | `TypeWithHumanize` |
| mouse-path realism | `WithMouseTremor(bool)`, `WithMouseSteps(int)`, `WithMouseSpeed(min,max)`, `WithMouseDeviation(float64)` | `ClickWithMouse` |
| scroll behavior | `WithPhysics()`, `WithDuration(int)` | `ScrollBy` |

The end-to-end path mirrors phases 25–27: **flag → `StealthFlags` → `ResolveStealth`
(precedence) → `Config.Stealth` → `NewContext` → action call site**. The struct
already reserves space for these knobs (`types/config.go` comment, "Typing/typo/
jitter/mouse/scroll humanize knobs").

Out of scope for this phase:
- **Hover / DragAndDrop tuning** — `humanize.Hover(el)` and
  `humanize.DragAndDrop(page, src, tgt)` have **fixed signatures (no variadic
  opts)**, so mouse tuning threads ONLY into `ClickWithMouse` for v1.6. Tuning
  hover/drag would require changing godoll signatures — deferred.
- Live WAF validation (Phase 29); any new humanize *algorithms* (we only thread
  existing options, we do not author new humanization behavior in godoll).

</domain>

<decisions>
## Implementation Decisions

### Config surface (types/config.go)
- Add humanize tuning fields to `StealthConfig` in the reserved Phase 28 space.
  Use **pointer types** (`*int`, `*float32`, `*bool`) for every tunable, carrying
  forward the Phase 27 CR-02 `*bool` lesson: a pointer distinguishes "unset"
  (nil → omit the godoll option, godoll's own default applies) from an explicit
  value (including an explicit zero/false). A plain value cannot, and a
  yaml-persisted explicit value would be clobbered by a baseline default.
- Speed ranges that are min/max pairs (`WithTypingSpeed`, `WithMouseSpeed`) are
  exposed as two fields each (e.g. `TypingSpeedMin *int`, `TypingSpeedMax *int`);
  an option is only emitted when **both** ends are set (otherwise nil → godoll
  default). Engineer's discretion on whether a single value sets both ends.
- Mirror the yaml/json tags + the resolve precedence the existing fields use, so
  the named-JSON-profile round-trips the humanize knobs for free.

### ResolveStealth (types/config.go)
- Overlay the new flag values onto `cfg.Stealth` with the SAME precedence the
  existing stealth fields use: **explicit flag > non-nil profile value > unset
  (nil)**. No new resolver shape — extend the existing one.

### Flag registration (cmd.go / main.go)
- Register the new flags where `--webrtc-protection` / `--canvas-noise` are
  declared (`cmd.go` ~135), forwarded (`cmd.go` ~60), and counted as
  session-affecting (`cmd.go` ~78); capture them onto `StealthFlags` in
  `main.go` (~37–56). Session-affecting because they are resolved once at
  daemon-spawn and frozen on the daemon `Config` — the established model.

### Threading into actions (actions/actions.go + types/context.go)
- `Context.config` is unexported and `actions` is a separate package, so add an
  **exported accessor** on `*types.Context` so action handlers can read the
  resolved humanize tuning. Two clean shapes (engineer's discretion):
  (a) `Context` exposes the resolved `StealthConfig` (or typed getters) and
  `actions.go` builds the `[]humanize.TypingOption` / `[]MouseOption` /
  `[]ScrollOption` (keeps the `humanize` import in `actions`, which already has
  it); or (b) `Context` returns the built option slices (moves the `humanize`
  import into `types`). Prefer (a) — narrower dependency surface.
- Change the call sites to pass the derived opts: `typeWithHumanize(el, val,
  opts...)`, `clickWithMouse(page, el, opts...)`, `humanizeScrollBy(page, dir,
  amt, opts...)`. When every knob is nil the opts slice is empty and behavior is
  **byte-for-byte the current default** (zero-regression guarantee).

### CI speed (criterion 3)
- Because unset knobs emit no options, the existing detection harness and
  `tests/humanize_test.go` keep godoll's current default speeds — **no wall-clock
  regression by construction**. Additionally provide the harness a way to pin
  FAST humanize speeds (via flag/profile) so any future slow default cannot leak
  into CI; engineer's discretion on whether the harness sets an explicit fast
  profile or relies on the unchanged defaults.

### Claude's Discretion
- Exact flag names (suggest `--typing-speed-min/max`, `--typo-rate`,
  `--mouse-tremor`, `--mouse-steps`, `--mouse-speed-min/max`, `--mouse-deviation`,
  `--scroll-duration`, `--scroll-physics`), single-value vs min/max ergonomics,
  and whether mouse-path realism is one composite knob or per-option flags.
- Where the option-slice builders live (Context method vs actions helper).
- The CI-fast pin mechanism.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets (godoll — consumed via `replace => ../godoll`, read-only here)
- `../godoll/humanize/typing.go:40 WithTypoRate(float32)`, `:47 WithTypingSpeed(min,max int)`;
  `:60 TypeWithHumanize(el, text, opts ...TypingOption)`.
- `../godoll/humanize/mouse.go:75 WithMouseTremor(bool)`, `:82 WithMouseSteps(int)`,
  `:89 WithMouseSpeed(min,max int)`, `:97 WithMouseDeviation(float64)`;
  `:263 ClickWithMouse(page, el, opts ...MouseOption)`.
- `../godoll/humanize/scroll.go:44 WithPhysics()`, `:51 WithDuration(int)`;
  `:58 ScrollBy(page, dir, amount, opts ...ScrollOption)`.
- `../godoll/humanize/actions.go:14 Hover(el)` and `:29 DragAndDrop(page, src, tgt)`
  — **fixed signatures, no opts** (the out-of-scope constraint above).

### Established Patterns (rod-cli)
- `types/config.go:39 StealthConfig` — precedence-resolved (flag > profile >
  default), frozen at daemon spawn; reserves the Phase 28 humanize knobs.
- `types/config.go ResolveStealth(cfg, flags)` (~197) — the single overlay
  resolver; `StealthFlags` (~141) is the highest-precedence input.
- `cmd.go` — flag defs (~135), `IsSet`-gated forwarding to the daemon argv (~60),
  session-affecting detection (~78). `main.go:37` builds `StealthFlags` from the
  cli.Context.
- `actions/actions.go` — the humanize wrappers are package-level vars
  (`typeWithHumanize`=`humanize.TypeWithHumanize` :53, `clickWithMouse` :52,
  `humanizeScrollBy` :55); call sites at :312/:454 (type), :293 (click),
  :493–508 (scroll). Handlers already take `ctx *types.Context`.
- `types/context.go:217 Context{ config Config; ... }` (line 219) — carries the
  resolved Config; `NewContext` (250) seeds it. The `noiseSeed` field (240) is
  the phase-27 precedent for per-session derived state on Context.

### Integration Points
- Tunability proof (criterion 2): a test driving the **real prebuilt binary**
  (`go build -o rod-cli .` first — stale-binary trap) sets a slow typing speed
  and asserts measurably longer per-keystroke timing vs the default; the existing
  `tests/humanize_test.go:59` already times typing (≥300ms assertion) as a model.
- Zero-regression (criterion 3): assert default (no-flag) timing is unchanged.

</code_context>

<specifics>
## Specific Ideas

- "delay jitter" has **no dedicated godoll option** — it is the variance produced
  by the `WithTypingSpeed(min,max)` spread. Document this mapping honestly in the
  flag usage rather than inventing a knob godoll cannot honor.
- The zero-regression guarantee (empty opts ⇒ identical default behavior) is the
  safety invariant for this phase: existing humanize tests must stay green
  untouched, and the harness wall-clock must not move.

</specifics>

<deferred>
## Deferred Ideas

- Hover / DragAndDrop tuning (needs godoll variadic-opts signature changes).
- Any new humanization algorithm (this phase only threads existing options).

</deferred>
