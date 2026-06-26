---
author: engineer
responsible: engineer
phase: 33-advanced-evasion
plan: 01
wave: 1
status: complete
requirements: [EVAD-01, EVAD-02, EVAD-03]
commits:
  - repo: rod-cli
    hash: cdeffe0
    files: [types/context.go, types/config.go, cmd.go, main.go]
  - repo: godoll
    hash: f19d29b
    files: [fingerprint/generator.go, stealth/evasion.go, "+accumulated prior-phase work"]
---

# Phase 33 wave 1 — SUMMARY (Advanced Evasion: dimension activation)

## What landed

Activated godoll's dormant fingerprint-dimension injectors (fonts / media devices
/ battery / codecs / plugins) in rod-cli, coherently and under per-vector toggles.
The single missing call was `em.SetFingerprint(fp)` — without it `em.fingerprint`
was nil and `applyFingerprintDimensions()` was skipped on every page.

### rod-cli (commit cdeffe0)
- **`types/context.go` createPage** — now:
  1. Builds the config-pinned `prof` first (identity source of truth).
  2. Generates the fingerprint **constrained to the profile OS + locale**:
     `FPWithBrowserNames("chrome")` + `FPWithOS(osForPlatform(prof.Platform))` +
     `FPWithLocales(prof.Locale)` (when set), via the new
     `NewFingerprintGeneratorSeeded(int64(ctx.noiseSeed), …)` so dimensions are
     stable across page recreation within a session.
  3. Calls `em.SetFingerprint(fp)` **before** `em.SetProfile(prof)` — the load-
     bearing ordering: `SetFingerprint` derives `em.profile` via `FromFingerprint`,
     so `SetProfile` runs after to restore the config identity, while
     `em.fingerprint` stays set to activate the dimension path.
  4. Calls `em.SetDimensionOptions(...)` gated by the 4 resolved toggles.
- **`osForPlatform` helper** — maps `navigator.platform` → godoll OS key
  (Win32→windows, MacIntel→macos, Linux*→linux), mobile tokens tested first;
  unknown→windows. Coherent-not-random: dimensions match the profile OS.
- **`types/config.go`** — 4 new `*bool` `StealthConfig` fields (FontSpoof,
  MediaDevicesSpoof, BatterySpoof, CodecSpoof) + matching `StealthFlags` fields +
  `ResolveStealth` precedence blocks (CLI>profile>default, default ON) +
  `DefaultConfig` defaults.
- **`cmd.go`** — `--font-spoof`/`--media-devices-spoof`/`--battery-spoof`/
  `--codec-spoof` BoolFlags (default true), argv forwarding (only when explicitly
  set), and `stealthRequested` inclusion.
- **`main.go`** — daemon-side capture of the 4 flags into `StealthFlags`.

### godoll (commit f19d29b)
- **`fingerprint/generator.go`** — exported `NewFingerprintGeneratorSeeded(seed,
  opts...)` (delegates to the existing unexported seeded ctor).
- **`stealth/evasion.go`** — `DimensionOptions{Codecs,Plugins,Battery,
  MediaDevices,Fonts}` + `DefaultDimensionOptions()` (all-on) + `dimOpts` field +
  `SetDimensionOptions`; each injector in `applyFingerprintDimensions` gated on
  its option. **nil dimOpts = historical all-on**, so existing godoll callers are
  unaffected (backward compatible). A toggle OFF skips `InjectScript` entirely.

## must_haves.truths — status
- SetFingerprint(OS-constrained fp) before Apply, activates dimensions — **DONE**.
- OS-coherent (not unconstrained random) — **DONE** (FPWithOS + osForPlatform).
- 4 `*bool` toggles, CanvasNoise pattern, default ON, CLI>profile>default — **DONE**.
- Toggles independent + effective (OFF skips injection) — **DONE** (godoll gating;
  proven in wave 2).
- CLI flags forwarded into daemon argv, precedence preserved — **DONE**.
- Evasion failures log-and-continue — **DONE** (fp-gen failure warns; SetProfile
  still runs so identity survives a nil fp).
- build + vet pass (both repos); v1.6 tests pass — **DONE** (`go build ./...`,
  `go vet ./...` clean in rod-cli; `./types/` Resolve/Stealth/Profile green;
  godoll build + scoped vet + stealth/fingerprint tests green).

## Independent review
Spawned `anvil-code-reviewer` on the diff → **SHIP — minor notes**. All five
load-bearing invariants traced clean. Fixed both minors: (1) `osForPlatform`
insertion had split `profileFromStealth`'s doc comment — relocated; (2) mobile
branches re-ordered before linux/mac and the inherent platform-sniffing limit
documented honestly. Nit (Plugins hardcoded-on per D-01) accepted as intentional.

## Deviations / findings (routed to architect)
- **godoll font spoof is an observable no-op.** `scriptMockFonts`
  (fingerprint_bridge.go:245-279) returns the original `measureText` width on
  EVERY branch (available, base, and fallthrough all `return result`). So the
  FontSpoof toggle is wired and gates injection, but the underlying spoof changes
  no observable. Wave 2 therefore proves coherence/stability/toggle-off on
  **mediaDevices + battery + codecs** (≥2 independent surfaces, satisfies
  EVAD-02); fonts ships the toggle but is not separately harness-asserted.
- **godoll vendored tree was already dirty.** Substantial uncommitted v1.6/v1.7
  evasion work (canvas/audio/webgl seeding, client-hints derive, raw-CDP
  injection) had accumulated across earlier phases and is interdependent with my
  evasion.go hunks (a Phase-33-only godoll commit would not compile). Committed
  the full source tree as f19d29b with an honest message. A stray 1.1MB `ar`
  archive (`godoll/rod-cli`) was left untracked (not committed). Pre-existing vet
  warning in `internal/testing/testing_test.go:83` is unrelated and untouched.
