---
author: engineer
responsible: engineer
phase: 27-canvas-webgl-webrtc-hardening
plan: 04
subsystem: detection-harness
status: complete
tags: [harness, webrtc, canvas, detection, hardening, test]
dependency_graph:
  requires:
    - "Plan 03 runtime wiring (EvadeWebRTC + WithWebRTCLeakProtection + seeded canvas/audio noise live on the page)"
  provides:
    - "internal/detect/detect.js canvasHash probe (deterministic-content traceability)"
    - "tests/detection_test.go webrtc_ice required-green (default-deny no-leak assertion)"
    - "tests/detection_test.go canvas_noise_stable (two reads identical + noise-actually-applied anchor)"
  affects:
    - "Phase 27 done-definition gate — harness proves Plans 02+03 work against a real browser"
tech_stack:
  added: []
  patterns:
    - "Required-green flip of a KNOWN-RED baseline subtest with net.ParseIP real-IP detection + .local mDNS allowance + default-deny on unrecognized tokens"
    - "Stability + applied-ness double assertion (stable across reads AND perturbed off a flat-fill baseline) so a no-op regression cannot stay green"
key_files:
  created: []
  modified:
    - "internal/detect/detect.js"
    - "tests/detection_test.go"
decisions:
  - "webrtc_ice PASS = empty OR no-RTCPeerConnection OR all-tokens-.local; FAIL = undefined (loud-fail guard) OR error: OR any routable IP. CR-04 hardened it to default-deny: a non-empty, non-.local, non-parseable token (incl. IPv6 with a zone, stripped before ParseIP) is also a failure rather than a silent pass."
  - "canvas_noise_stable was strengthened (CR-03): beyond r1===r2 + len>0, it now draws a flat rgb(128) fill and asserts >=1 read-back byte was perturbed off 128, so a regression to a no-op canvas (the pre-Phase-27 stub) fails here too — stability alone could not distinguish 'stable noise' from 'no noise'."
metrics:
  completed: "2026-06-24"
  tasks: 3
  files_modified: 2
  commit: "de3ff3a (+ 09fe94a review-fix CR-03/CR-04)"
---

# Phase 27 Plan 04: Detection Harness Flip Summary

Flipped the harness to assert both hardening features on the live page: `webrtc_ice` is now a required-green no-leak assertion (the `webrtc_ice_known_red` baseline-record wrapper is gone), and a new `canvas_noise_stable` subtest proves seeded canvas noise is stable-per-session. A `canvasHash` probe was added to detect.js for traceability.

## What Was Built

### Task 1 — canvasHash probe
- `internal/detect/detect.js`: a synchronous `probe("canvasHash", ...)` in the table-stakes block draws deterministic fixed content (fillRect + fillText, no Date/Math.random) and returns `toDataURL()` — informational/traceability only.

### Task 2 — webrtc_ice required-green
- `tests/detection_test.go`: renamed `webrtc_ice_known_red` -> `webrtc_ice`, flipped to a no-leak assertion (added `net` import). PASS: empty / `no-RTCPeerConnection` / all-`.local`. FAIL: `undefined` (loud-fail guard retained) / `error:` / any routable IP via `net.ParseIP`. CR-04 hardened: zone-strip before ParseIP, default-deny on unrecognized tokens.

### Task 3 — canvas_noise_stable
- `tests/detection_test.go`: new subtest evals a self-contained IIFE that draws deterministic content and reads `toDataURL` twice, asserting `r1===r2` (`stable:<len>`, len>0) — `UNSTABLE` errors. CR-03 strengthened: a second eval draws a flat `rgb(128)` fill and asserts >=1 perturbed byte, so a no-op regression fails too.

## Verification

- `go vet ./tests/` clean; rebuilt binary before every harness run.
- `pkill -9 -x rod-cli` then `go test ./tests/ -run TestDetectionHarness -count=1` -> PASS; `webrtc_ice` and `canvas_noise_stable` both PASS (verbose-confirmed they RUN, not skipped).
- `go test ./tests/ -run TestNetworkEvasionHeaders -count=1` -> PASS (no header regression).
- Live: EvadeWebRTC clears iceServers; canvas noise stable + applied; audio getChannelData stable across two reads (CR-01 fix).

## Deviations from Plan

None for the harness flip. CR-03/CR-04 were independent-review hardening applied post-implementation (commit 09fe94a).

## Resolved follow-ups (lead-directed, commits 1e5a7f6 / bab65d8)

- **CR-02 [config precedence] — RESOLVED (lead decision):** the two `StealthConfig`
  toggles are now `*bool`; ResolveStealth honors a yaml-persisted false (precedence:
  flag > non-nil-yaml > default-true). Guarded by `TestResolveStealth_HardeningTogglesRoundTrip`
  in types. See 27-01-SUMMARY.
- **CR-03 audio coverage — CLOSED:** added the `audio_noise_stable` harness subtest
  (commit bab65d8) — reads `getChannelData` twice in one session and asserts the two
  reads are sample-identical (the regression guard for the CR-01 compounding-drift
  fix), with a no-AudioContext / blanked-surface loud-fail guard. PASSES.

## Open Items Routed to qa / architect

- **CR-06 [low]**: canvas delta keys on read-relative byte index, so a sub-region getImageData and a full toDataURL assign the same physical pixel different deltas. Full-canvas reads agree (the harness path); cross-API sub-region consistency is a deferred nicety.
- **Reviewer note for qa**: godoll `Apply()` has `go em.EnableRequestInterception()` commented out, so Sec-Ch-Ua header spoofing depends entirely on rod-cli's own router — verify it isn't silently lost (pre-existing, not introduced this phase; TestNetworkEvasionHeaders passes).

## Self-Check: PASSED
- detect.js — `canvasHash`, `toDataURL`, no Date/Math.random in the probe.
- detection_test.go — no `webrtc_ice_known_red`, `t.Run("webrtc_ice"`, `net.ParseIP`, loud-fail on `undefined`, `t.Run("canvas_noise_stable"`, `t.Run("audio_noise_stable"`, `UNSTABLE`.
- Commit de3ff3a, 09fe94a, bab65d8 — FOUND.
