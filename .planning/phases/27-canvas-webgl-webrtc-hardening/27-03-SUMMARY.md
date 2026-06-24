---
author: engineer
responsible: engineer
phase: 27-canvas-webgl-webrtc-hardening
plan: 03
subsystem: runtime-wiring
status: complete
tags: [webrtc, canvas, audio, runtime, seed, stealth, hardening]
dependency_graph:
  requires:
    - "Plan 01 cfg.Stealth.WebRTCLeakProtection / cfg.Stealth.CanvasNoise toggles"
    - "Plan 02 godoll EvasionManager.SetNoiseSeed / EvadeWebRTC / Profile.SpoofCanvas / WithWebRTCLeakProtection"
  provides:
    - "Context.noiseSeed (generated once in NewContext via crypto/rand; one daemon = one seed)"
    - "Gated WithWebRTCLeakProtection in launchBrowser"
    - "Gated EvadeWebRTC + SetNoiseSeed + SpoofCanvas/SpoofAudioContext-from-CanvasNoise in createPage/profileFromStealth"
  affects:
    - "Plan 04 harness asserts the live effects (no leaked host IP, stable canvas hash)"
tech_stack:
  added: []
  patterns:
    - "Per-session seed generated once at Context construction (crypto/rand) so every createPage in the daemon's life shares it (stability), with a non-constant fallback on the near-impossible rand error (no predictable seed 0)"
    - "Log-and-continue (VALIDATE-03) for the new EvadeWebRTC leg — errors to stderr, never abort the daemon"
key_files:
  created: []
  modified:
    - "types/context.go"
decisions:
  - "noiseSeed lives on the Context (not Config) because launchBrowser takes Config BY VALUE — a seed on Config could not propagate back. Generated once in NewContext guarantees one seed for the daemon's life = stable re-reads."
  - "One toggle (CanvasNoise) gates BOTH SpoofCanvas and SpoofAudioContext in profileFromStealth, per CONTEXT. With CanvasNoise defaulting true, the no-pin path noises canvas+audio by default."
  - "Both WebRTC legs (browser pref in launchBrowser + EvadeWebRTC JS in createPage) gate on the single cfg.Stealth.WebRTCLeakProtection toggle so they move together."
metrics:
  completed: "2026-06-24"
  tasks: 3
  files_modified: 1
  commit: "bd08cbc (+ 09fe94a review-fix CR-05)"
---

# Phase 27 Plan 03: Runtime Wiring Summary

Wired both hardening features into the rod-cli runtime: a stable per-session noise seed on the Context, the WebRTC browser-pref + JS legs gated on `cfg.Stealth.WebRTCLeakProtection`, and `SpoofCanvas`/`SpoofAudioContext` driven from `cfg.Stealth.CanvasNoise`. This is the integration seam that makes Plan 01 (config) and Plan 02 (godoll engine) take effect on the live browser.

## What Was Built

### Task 1 — session seed (commit bd08cbc; hardened in 09fe94a)
- `types/context.go`: `Context.noiseSeed uint64`, generated once in `NewContext` via `crypto/rand` (`encoding/binary` for the uint64 decode). CR-05 fix: the `rand.Read` error is now handled — on the near-impossible failure it falls back to `time.Now().UnixNano() ^ os.Getpid()` rather than shipping the predictable seed 0.

### Task 2 — WebRTC browser pref
- `launchBrowser`: `if cfg.Stealth.WebRTCLeakProtection { opts = opts.WithWebRTCLeakProtection(true) }` after the StealthPreset chain, before `NewBrowserE`.

### Task 3 — createPage + profileFromStealth
- `profileFromStealth`: `p.SpoofCanvas = s.CanvasNoise` and `p.SpoofAudioContext = s.CanvasNoise` after `p.SpoofClientHints = true`.
- `createPage`: `em.SetNoiseSeed(ctx.noiseSeed)` right after `em.SetProfile(prof)`, before `em.Apply()`; the gated `em.EvadeWebRTC()` JS leg after Apply() with the same stderr log-and-continue discipline.

## Verification

- `go build -o rod-cli .` clean; `go test ./types/` PASS.
- Live (via `./rod-cli open` + `eval`): `getConfiguration().iceServers` returns `[]` even when set to a STUN server (EvadeWebRTC active); a flat `rgb(128)` canvas reads back as a 127/129 mix (±1 LSB noise active); two `toDataURL` reads identical (stable).
- Full effects asserted by Plan 04's harness (webrtc_ice + canvas_noise_stable).

## Deviations from Plan

None for the wiring. CR-05 (ignored rand error) was an independent-review finding fixed post-implementation (commit 09fe94a).

## Post-handoff update (CR-02, commit 1e5a7f6)

The hardening toggles became `*bool` (lead decision). The two context.go consumers
were updated to deref with a nil→true default: `boolVal(cfg.Stealth.WebRTCLeakProtection, true)`
in launchBrowser, `boolVal(ctx.config.Stealth.WebRTCLeakProtection, true)` in
createPage, and `canvasNoise := boolVal(s.CanvasNoise, true)` driving both
`p.SpoofCanvas` and `p.SpoofAudioContext` in profileFromStealth. Behavior is
unchanged for the default/flag paths; a yaml-persisted false now also flows through.

## Self-Check: PASSED
- types/context.go — `noiseSeed uint64`, `c.noiseSeed = binary.LittleEndian.Uint64`, `WithWebRTCLeakProtection`, `boolVal(cfg.Stealth.WebRTCLeakProtection, true)`, `canvasNoise := boolVal(s.CanvasNoise, true)` → SpoofCanvas/SpoofAudioContext, `em.SetNoiseSeed(ctx.noiseSeed)`, `em.EvadeWebRTC()` inside `if boolVal(ctx.config.Stealth.WebRTCLeakProtection, true)`.
- Commit bd08cbc, 09fe94a, 1e5a7f6 — FOUND.
