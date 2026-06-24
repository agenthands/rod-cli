---
author: engineer
responsible: engineer
phase: 27-canvas-webgl-webrtc-hardening
plan: 02
subsystem: godoll-stealth
status: complete
tags: [canvas, audio, noise, seeded-prng, godoll, hardening]
dependency_graph:
  requires:
    - "godoll stealth.Profile / EvasionManager / WebGLManager / InjectScript substrate"
    - "Phase-26 json.Marshal injection-boundary discipline"
  provides:
    - "Profile.SpoofCanvas gate (DefaultProfile false; FromFingerprint true)"
    - "Seeded scriptCanvasNoise(seed) — toDataURL + getImageData stable-per-session noise"
    - "Seeded scriptAudioContextNoise(seed) — replaces Math.random() with a per-index PRNG"
    - "WebGLManager.SetSeed + EvasionManager.noiseSeed/SetNoiseSeed + Apply() seed threading & SpoofCanvas gate"
  affects:
    - "Plan 03 calls em.SetNoiseSeed(session seed) and sets profile.SpoofCanvas from cfg.Stealth.CanvasNoise"
    - "Plan 04 harness asserts two toDataURL reads are byte-identical (stability invariant)"
tech_stack:
  added: []
  patterns:
    - "Deterministic per-index delta = pure hash of (seed ^ index) — no global PRNG state, so re-reads of the same (seed,index) are identical (stability invariant)"
    - "toDataURL noise via a throwaway canvas (read raw via saved-original getImageData, render noised copy, serialize that) so the source canvas is NEVER mutated — no compounding drift across reads"
key_files:
  created: []
  modified:
    - "../godoll/stealth/profile.go"
    - "../godoll/stealth/fingerprint_bridge.go"
    - "../godoll/stealth/js.go"
    - "../godoll/stealth/webgl.go"
    - "../godoll/stealth/evasion.go"
decisions:
  - "The canvas delta is a PURE function of (seed, byte-index) computed by a mulberry32-style hash of (seed ^ index) that advances NO global state. This is the load-bearing decision for the HARDEN-02 stability invariant: perturbing the same (seed,index) twice yields the same delta, so re-reads are byte-identical."
  - "toDataURL does NOT write back to the source canvas. The naive 'getImageData -> perturb -> putImageData -> toDataURL' approach compounds: the second read perturbs the already-perturbed pixels and drifts. Instead toDataURL reads RAW pixels via the saved-original getImageData, renders the noised copy onto a throwaway canvas, and serializes that — the source canvas is left pristine, so every read perturbs the ORIGINAL value and is identical."
  - "The patched getImageData uses the saved-original getImageData internally (and toDataURL also uses the saved-original) to avoid double-perturbation through the patched method."
  - "Canvas noise is now gated on Profile.SpoofCanvas in Apply() (was unconditional); MockVendorRenderer stays unconditional. Audio noise stays gated on SpoofAudioContext."
  - "Only the low 32 bits of the seed feed the PRNG lane — sufficient for ±1 LSB noise; seed=0 is valid (scripts inject without panicking)."
metrics:
  completed: "2026-06-24"
  tasks: 3
  files_modified: 5
  commit: "uncommitted-by-design (../godoll consumed via replace => ../godoll)"
---

# Phase 27 Plan 02: Seeded Stable-Per-Session Canvas/Audio Noise (godoll) Summary

Replaced godoll's no-op canvas noise stub and the `Math.random()`-based audio noise with a **seeded, stable-per-session** implementation, and added the `SpoofCanvas` gate. Same seed -> byte-identical canvas/audio readback across re-reads (the HARDEN-02 stability invariant); a different session seed -> a different but equally-stable fingerprint.

## What Was Built

### Task 1 — SpoofCanvas gate
- `profile.go`: `Profile.SpoofCanvas bool` (json `spoofCanvas`), after `SpoofAudioContext`. `DefaultProfile()` sets `SpoofCanvas: false`.
- `fingerprint_bridge.go`: `FromFingerprint` sets `SpoofCanvas: true` next to `SpoofAudioContext: true`.

### Task 2 — seeded scripts
- `js.go`: `scriptCanvasNoise` and `scriptAudioContextNoise` converted from consts to `func(seed uint64) string` generators. Seed escaped via `json.Marshal` (low 32 bits). Both define a `__delta`/`__afrac` deterministic per-index hash of `(seed ^ index)` — no `Math.random()`, no global state.
  - Canvas: patches `toDataURL` (raw-read -> noised copy on a throwaway canvas -> serialize, source untouched) AND `getImageData` (noised copy of the returned region). ±1 on the R channel of each pixel, clamped [0,255].
  - Audio: patches `AudioBuffer.getChannelData` and `AnalyserNode.getFloatFrequencyData`, same node coverage / stride (every 100th sample) as before, with the seeded `__afrac` replacing every `Math.random()`.
- `encoding/json` was already imported.

### Task 3 — seed threading + gate
- `webgl.go`: `WebGLManager.seed uint64` + `SetSeed`; `ApplyCanvasNoise()` now injects `scriptCanvasNoise(w.seed)` (stub gone).
- `evasion.go`: `EvasionManager.noiseSeed uint64` + `SetNoiseSeed`; `Apply()` threads the seed into the audio script (`scriptAudioContextNoise(em.noiseSeed)`) and into the WebGLManager (`webgl.SetSeed(em.noiseSeed)`), and GATES `webgl.ApplyCanvasNoise()` on `em.profile.SpoofCanvas` (was unconditional). `NewWebGLManager` signature unchanged (used `SetSeed` to avoid touching the 2 existing test callers).

## Verification

- `(cd ../godoll && go build ./...)` clean; `(cd ../godoll && go test ./stealth/)` PASS (15.7s).
- `go build -o rod-cli .` (rod-cli against ../godoll via replace) clean.
- `grep -c 'Math.random()' ../godoll/stealth/js.go` == 1, and that single hit is in a doc COMMENT ("The previous implementation used Math.random()"), not in any injected script — the acceptance criterion (0 inside the audio/canvas scripts) holds.
- godoll changes are UNCOMMITTED by design (`git -C ../godoll status` shows stealth/*.go modified, unstaged).
- Stability reasoning: the delta is a pure function of (seed, byte-index) and toDataURL never mutates the source canvas, so two reads of the same canvas in one session produce identical bytes. Plan 04 asserts this on the live page.

## Deviations from Plan

- Chose `WebGLManager.SetSeed(seed)` over changing `NewWebGLManager`'s signature, to avoid touching the 2 existing godoll test callers (`evasion_extra_test.go`, `stealth_coverage_test.go`) — the plan explicitly allowed either.

## Known Limitations

- `AudioBuffer.getChannelData` mutates the returned channel buffer in place (pre-existing structure, preserved per plan). Re-reading the SAME AudioBuffer would compound the (now deterministic) delta. This is not harness-asserted (Plan 04 asserts canvas stability only) and matches the original implementation's structure; the seeded determinism satisfies HARDEN-02's "not per-call-random" requirement. Noted for the lead/qa.

## Self-Check: PASSED
- profile.go — `SpoofCanvas bool`, `SpoofCanvas: false`.
- fingerprint_bridge.go — `SpoofCanvas: true`.
- js.go — `func scriptCanvasNoise(seed uint64) string`, `func scriptAudioContextNoise(seed uint64) string`, `getImageData` + `putImageData` present, 0 Math.random in scripts.
- webgl.go — `scriptCanvasNoise(` injected; stub removed.
- evasion.go — `func (em *EvasionManager) SetNoiseSeed`, `noiseSeed`, `em.profile.SpoofCanvas` gate.
- godoll uncommitted-by-design — CONFIRMED.
