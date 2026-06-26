# Phase 33: Advanced Evasion — Verification

**Verdict:** ✅ PASSED
**Verified:** 2026-06-26 (independent qa, goal-backward against HEAD)
**Requirements:** EVAD-01, EVAD-02, EVAD-03 — all delivered in code.

## Success criteria (ROADMAP Phase 33)
1. **≥2 new vectors beyond v1.6** — ✅ fonts/mediaDevices/battery/codecs activated;
   ≥2 harness-asserted (EVAD-01). Achieved by activating godoll's dormant
   `applyFingerprintDimensions` via `em.SetFingerprint(OS-constrained fp)` in
   createPage (types/context.go:660-690) — previously nil-skipped.
2. **Toggles CLI>profile>default + harness-asserted** — ✅ 4 `*bool` toggles
   (FontSpoof/MediaDevicesSpoof/BatterySpoof/CodecSpoof), precedence resolved
   (config.go:126-140, 428-446), flags forwarded (cmd.go); toggle-off proven live
   (`media_devices_toggle_off_reverts`).
3. **Documented (EVAD-03)** — ✅ docs/stealth-config.md:76-117 (table, default-ON,
   precedence, on/off + perf, coherent-per-OS, real-Chrome, font caveat).
4. **Stable within session** — ✅ per-session `ctx.noiseSeed` → seeded generator;
   `TestSeededFingerprintDimensions/same_seed_same_dimensions` proves byte-identical.

## Coherence (the trap) — held
`osForPlatform` (context.go:495-512) maps platform→OS for `FPWithOS`, so injected
fonts/devices/codecs match the profile OS (proven by `os_coherent_fonts`). Never an
unconstrained random fingerprint on a pinned profile (coherent-not-random lens).

## Disclosed limitation (assessed acceptable)
godoll's `scriptMockFonts` (godoll/stealth/fingerprint_bridge.go:245-279) is an
OBSERVABLE NO-OP — every branch returns the original measureText width. So
`--font-spoof` gates injection but produces no live-readable change. **EVAD-02's
"≥2 surfaces asserted" still holds** via media devices + battery + codecs (live
coherence + toggle-off + stability). The font no-op is honestly disclosed in the
SUMMARYs AND the docs (caveat box) — never claimed as working. Font SET coherence is
proven at the generation layer. Follow-up (non-blocking): make godoll's font injector
observable, then add a live fonts assertion.

## godoll changes (vendored, allowed)
- `fingerprint/generator.go`: exported `NewFingerprintGeneratorSeeded`.
- `stealth/evasion.go`: `DimensionOptions` + `SetDimensionOptions` + per-dimension
  gating; nil dimOpts = all-on (existing callers unaffected — backward compatible).

## Evidence
`go build ./...` + `go vet ./...` clean (rod-cli + godoll). `go test ./tests/ -run
"AdvancedEvasion|Seeded|Dimension"` PASS (7 subtests). `TestDetectionHarness` PASS
(17/17, no regression). Constraint honored: real Chrome, no TLS spoofing.

## Hygiene (handled / routed to milestone close)
- Stray 1.13MB `ar` archive `../godoll/rod-cli` (pre-existing Jun-24 build artifact) —
  REMOVED during verification.
- godoll commit f19d29b bundled accumulated prior-phase v1.6/v1.7 evasion work
  (interdependent; a Phase-33-only commit wouldn't compile) — engineer-disclosed,
  noted for provenance.
