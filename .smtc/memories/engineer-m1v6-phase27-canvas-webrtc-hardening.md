# engineer — Phase 27 (v1.6) Canvas/WebGL/WebRTC hardening

## Load-bearing pattern: stable-per-session readback noise
The HARDEN-02 invariant is "two reads of the same surface in one session are
byte-identical". The delta MUST be a pure function of (seed, index) — a hash of
(seed ^ index), advancing NO global PRNG state. mulberry32-style hash works.

## The compounding-drift trap (cost a code-review blocker)
A naive `getImageData -> perturb in place -> putImageData -> toDataURL` COMPOUNDS:
read N returns original + N*delta. Two reads differ = a detection tell.
- Canvas toDataURL fix: read RAW pixels via the SAVED-ORIGINAL getImageData,
  render the noised COPY onto a throwaway canvas, serialize THAT — source untouched.
- `getImageData` is safe to perturb-in-place ONLY because it allocates a fresh
  ImageData each call.
- `AudioBuffer.getChannelData` returns a REFERENCE to the live internal
  Float32Array — perturbing in place compounds. Fix: `new Float32Array(src)`, write
  the copy, return the copy. `AnalyserNode.getFloatFrequencyData` is safe (the
  original overwrites arguments[0] wholesale each call).
RULE: any readback API that returns a live/persistent buffer must be copied before
perturbing; one that re-allocates per call can be mutated in place.

## Seed wiring (rod-cli)
- noiseSeed lives on the Context (NOT Config — launchBrowser takes Config by value).
- Generated ONCE in NewContext via crypto/rand; handle the rand.Read error (fall
  back to time^pid) so you never ship the predictable seed 0.
- Threaded via em.SetNoiseSeed before em.Apply().

## Toggle precedence (CR-02 — RESOLVED via *bool, the canonical pattern)
A default-TRUE config toggle that must ALSO round-trip a yaml-persisted `false`
CANNOT be a plain `bool`: zero-value false is indistinguishable from a deliberate
file-false, so an unconditional default-true baseline in the resolver CLOBBERS it.
FIX (commit 1e5a7f6): make BOTH the StealthConfig field AND the StealthFlags field
`*bool`. Helpers `boolPtr(b) *bool` and `boolVal(p, def) bool`. Resolved precedence:
explicit flag (non-nil) > yaml-loaded cfg value (non-nil, incl. false) > default
true (set ONLY when nil). Do NOT unconditionally re-baseline. Consumers deref via
boolVal(p, true). Round-trip test asserts persisted-false-survives /
omitted-resolves-true / flag-overrides-persisted. This is the pattern for any future
default-on-but-yaml-disablable toggle. (Audio CR-01 regression guard:
audio_noise_stable subtest, two getChannelData reads sample-identical.)

## Operational (each cost a Phase-26 failure)
- ALWAYS `go build -o rod-cli .` before any runCli-based test (tests exec the
  prebuilt ../rod-cli binary).
- Kill ONLY with `pkill -9 -x rod-cli` (exact comm). NEVER `pkill -f` (self-matches).
- godoll changes stay UNCOMMITTED (replace => ../godoll). Build with
  `(cd ../godoll && go build ./... && go test ./stealth/)`.
- Never `go test ./...` (browser timeout + flaky daemon_more_test.go). Targeted only.
- Nav command is `open`/`goto`, not `navigate`.

## Verifying noise is REAL (not vacuous)
A flat rgb(128) fill read back via getImageData shows a 127/129 mix (±1 LSB). The
canvas_noise_stable test was strengthened to assert this (stability alone can't tell
"stable noise" from "no noise" — the pre-27 no-op stub was also perfectly stable).
