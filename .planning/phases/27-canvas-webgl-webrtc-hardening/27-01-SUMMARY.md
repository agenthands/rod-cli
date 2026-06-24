---
author: engineer
responsible: engineer
phase: 27-canvas-webgl-webrtc-hardening
plan: 01
subsystem: stealth-config
status: complete
tags: [webrtc, canvas, config, cli, stealth, hardening]
dependency_graph:
  requires:
    - "Phase 25/26 StealthConfig substrate (types/config.go ResolveStealth three-tier resolver, StealthFlags, daemon-spawn flag forwarding)"
  provides:
    - "StealthConfig.WebRTCLeakProtection + StealthConfig.CanvasNoise bool fields (default-true semantics)"
    - "StealthFlags.WebRTCLeakProtection + StealthFlags.CanvasNoise *bool tri-state raw flags"
    - "--webrtc-protection / --canvas-noise BoolFlags (Value:true) registered, conditionally forwarded, captured daemon-side"
    - "ResolveStealth default-true baseline + explicit-flag override for both toggles"
  affects:
    - "Plan 03 reads cfg.Stealth.WebRTCLeakProtection / cfg.Stealth.CanvasNoise to gate the actual evasion calls"
tech_stack:
  added: []
  patterns:
    - "Tri-state *bool flag capture (nil=unset=keep-default-true) gated on c.IsSet, so default-on is distinguishable from explicit --flag=false"
    - "Conditional verbatim forwarding into the daemon spawn argv only when c.IsSet (keeps the daemon *bool nil otherwise)"
key_files:
  created: []
  modified:
    - "types/config.go"
    - "cmd.go"
    - "main.go"
decisions:
  - "Flags use urfave BoolFlag Value:true (default-on) and *bool capture so the resolver can tell 'unset' from 'explicit false' — a plain bool cannot, because the default is true."
  - "ResolveStealth establishes a hardened-by-default baseline (both toggles true) then applies the explicit-flag override. The godoll stealth.Profile carries no WebRTCLeakProtection/CanvasNoise field, so there is no Tier-2 profile leg for these toggles — the precedence is explicit --flag=false > built-in default(true)."
  - "Both toggles ORed into the stealthRequested already-running warning predicate so an explicit toggle on a live daemon warns it won't retroactively apply (consistent with the Phase-25/26 stealth-flag UX)."
metrics:
  completed: "2026-06-24"
  tasks: 3
  files_modified: 3
  commit: "7f2e162"
---

# Phase 27 Plan 01: Hardening Config Toggles Summary

Added the two Phase-27 hardening toggles to the stealth config spine: `WebRTCLeakProtection` and `CanvasNoise` bool fields on `StealthConfig` (both default-true), the matching `*bool` raw fields on `StealthFlags`, the `--webrtc-protection` / `--canvas-noise` BoolFlags (default-on), conditional argv forwarding, daemon-side capture, and the `ResolveStealth` default-true + explicit-override precedence. Pure rod-cli config layer — no godoll, no browser.

## What Was Built

### Task 1 — fields
- `types/config.go`: `StealthConfig` gains `WebRTCLeakProtection bool` and `CanvasNoise bool` (yaml/json lowerCamel tags). The reserved Phase-27 doc comment block was replaced with an "implemented" note (Phase-28 humanize knobs stay reserved). `StealthFlags` gains `WebRTCLeakProtection *bool` and `CanvasNoise *bool` (tri-state: nil=unset).

### Task 2 — resolution
- `DefaultConfig` now ships `Stealth: StealthConfig{WebRTCLeakProtection: true, CanvasNoise: true}` (hardened-by-default).
- `ResolveStealth` establishes the default-true baseline for both toggles, then applies the explicit-flag override (`*flags.X` when `flags.X != nil`). Additive — does not disturb the existing string-field resolution.

### Task 3 — flags
- `cmd.go`: registered both BoolFlags (Value:true) in the global flags; forwarded each into the daemon spawn argv only when `c.IsSet(...)` (verbatim `--flag=%t`); ORed both `c.IsSet` checks into the `stealthRequested` predicate.
- `main.go`: captured both into `StealthFlags` as `*bool`, only when `c.IsSet(...)`.

## Verification

- `go build .` / `go build ./types/` clean; `go test ./types/` PASS (10.6s).
- `./rod-cli --help` lists both `--webrtc-protection` and `--canvas-noise` with `(default: true)`.

## Deviations from Plan

- The plan's Task 2 mentioned an optional Tier-2 "overlay the profile value when a profile was loaded" leg. The godoll `stealth.Profile` struct carries no `WebRTCLeakProtection`/`CanvasNoise` field (these are rod-cli-only config concepts), so there is no profile value to overlay — implemented as default-true baseline + explicit-flag override only. The net precedence (explicit --flag=false > default true) matches the plan's stated `must_haves`.

## Known Limitations

- A `rod-cli.yaml` config file that explicitly set `webRTCLeakProtection: false` / `canvasNoise: false` would be clobbered back to true by the ResolveStealth baseline (a bool zero-value is indistinguishable from a deliberate false at the config-struct level — which is exactly why the *flags* are `*bool`). The plan lists no yaml tier; the override path is the CLI flag. Noted for the lead/qa in case yaml-driven disable is later desired.

## Self-Check: PASSED
- types/config.go — `WebRTCLeakProtection bool` + `CanvasNoise bool` (StealthConfig), `WebRTCLeakProtection *bool` + `CanvasNoise *bool` (StealthFlags), default-true baseline + `flags.WebRTCLeakProtection != nil` / `flags.CanvasNoise != nil` overrides.
- cmd.go — both BoolFlags registered, `IsSet("webrtc-protection")` forwarding.
- main.go — `IsSet("webrtc-protection")` capture into `stealthFlags.WebRTCLeakProtection`.
- Commit 7f2e162 — FOUND.
