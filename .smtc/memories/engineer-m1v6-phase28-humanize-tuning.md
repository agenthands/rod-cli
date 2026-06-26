# engineer — Phase 28 (v1.6) Human-behavior tuning (HUMANIZE-01)

## What it did
Threaded godoll's existing humanize.With* options through the rod-cli config
spine into actions/actions.go (was defaults-only). Same path as 25-27:
flag(cmd.go) -> daemon argv -> StealthFlags(main.go) -> ResolveStealth precedence
-> Config.Stealth -> Context.HumanizeTuning() accessor -> actions builders ->
variadic godoll call site. Commits 244bf45 (impl) 715b2c4 (test) 869653d (CR fix)
9a23d35 (SUMMARYs).

## The load-bearing pattern: pointer = unset = emit-no-option = zero regression
Every tunable is a pointer (*int/*float32/*bool/*float64). nil means the builder
emits NO godoll option, so godoll's ApplyDefaults applies => byte-for-byte current
behavior. This is THE zero-regression invariant. Builders (typingOpts/mouseOpts/
scrollOpts in actions.go) guard each append on `field != nil`, and a min/max PAIR
option (WithTypingSpeed/WithMouseSpeed) on `min != nil && max != nil`.
ResolveStealth overlays humanize knobs OVERRIDE-ONLY (no default-true baseline,
unlike the Phase-27 toggles) — nil is left nil.

## CR-MAJOR caught by anvil-code-reviewer (the real bug): godoll rand PANICS
godoll ../godoll/internal/rand/rand.go: RandomDuration panics on min<0, max<0, or
min>max; RandomInt panics on min>max. WithTypingSpeed/WithMouseSpeed feed these.
So `--typing-speed-min 200 --typing-speed-max 100` or a negative IntFlag value
panicked PER-KEYSTROKE inside the frozen daemon (net/http per-request recover
keeps daemon alive but client gets opaque EOF; every type/fill broken till
re-spawn). FIX: validateHumanizeTuning() in ResolveStealth (the fail-fast spawn
seam, same as deriveAndValidateFingerprint): speed pairs 0<=min<=max, typo-rate &
mouse-deviation in [0,1], mouse-steps & scroll-duration >=1, incomplete pair is a
hard error. RULE: when threading a user value into a 3rd-party (godoll) option,
check that lib's precondition (it may panic, not error) and validate at the
config boundary, not at the call site.

## godoll humanize option asymmetries (honesty, documented not invented)
- "delay jitter" has NO godoll option — it IS the WithTypingSpeed(min,max) spread.
  Don't invent a jitter flag.
- WithPhysics() can only ENABLE physics (godoll's own default; no disable option).
  So --scroll-physics=false CANNOT turn physics off without a godoll change. Emit
  WithPhysics() only on explicit true; false/nil emits nothing (default still on).
- WithMouseTremor(false) DOES work (sets EnableTremor=false; ApplyMouseDefaults
  doesn't override it) — tremor is genuinely tunable both ways.
- ApplyDefaults runs BEFORE opts in godoll, so an explicit 0/false IS honored
  (only the *zero-value-means-default* godoll fields get defaulted when unset).

## CR-MAJOR test soundness: kill the variance term
The timing test compared two unseeded runs with godoll's default 0.02 typo rate
active — typo-correction sleeps are hardcoded +400ms constants INDEPENDENT of the
speed config, firing on both paths. That could erode the slow-vs-default gap =>
intermittent false-fail. FIX: pin --typo-rate 0 on BOTH compared paths so the
base-delay difference is the only signal; widen slow pin to 300-400ms, margin to
>=1s. Floor for zero-regression (>=300ms) stays deterministic: 11 keys*30ms=330ms
with typos off. RULE: a timing/behavior comparison test must pin/disable every
RNG term that isn't the variable under test.

## anvil tooling note
mcp__anvil__spawn / roster_list NOT present as MCP tools this session; the
`anvil-cc spawn <role> --seat engineer --phase NN --task "..."` Bash sidecar DOES
work — spawns a real `claude -p` reviewer, writes /tmp/anvil-spawn-out/NN-REVIEW.md
with typed handoff/evidence records. Used it for the pre-handoff code-review gate.

## Operational (unchanged from 27, still true)
go build -o rod-cli . before any runCli test; pkill -9 -x rod-cli only; never
pkill -f; never the user's Chromium; never `go test ./...` (browser timeout +
flaky daemon_more_test.go); targeted -run only. See
[[engineer-m1v6-phase27-canvas-webrtc-hardening]].
