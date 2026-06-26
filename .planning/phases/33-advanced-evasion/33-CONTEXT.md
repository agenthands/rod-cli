---
author: architect-lead
---

# Phase 33: Advanced Evasion - Context

**Gathered:** 2026-06-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Expand fingerprint hardening beyond v1.6 (canvas/WebGL/WebRTC/audio) by wiring
godoll's already-implemented but currently-DORMANT fingerprint dimensions —
**fonts, media devices, battery, media codecs** (plugins comes along) — into
rod-cli as **coherent, per-session-stable, individually-toggleable** hardening
surfaces, each **asserted by the offline harness**.

**In scope (EVAD-01/02/03):** identify + harden ≥2 new vectors (we do 4), per-vector
toggles on the v1.6 precedence chain, harness assertions, stability.

**Out of scope:** TLS (cancelled, real-Chrome-only — [[no-tls-spoofing-real-chrome]]);
new godoll JS evasion *invention* (godoll already has the injectors — this is wiring
+ coherence + assertion, not new spoofing JS).

</domain>

<decisions>
## Implementation Decisions

### The vectors (operator-selected 2026-06-26: ALL FOUR)
- **D-01:** Harden **fonts, media devices, battery, and media codecs** (operator picked
  all four; `plugins` is injected by the same godoll path so it comes along). These are
  the EVAD-01 "vectors not covered by v1.6" — each is a real fingerprinting surface
  currently leaking headless-Chrome defaults in rod-cli.

### The key finding — godoll HAS it, rod-cli doesn't USE it
- **D-02:** godoll's `applyFingerprintDimensions()` (../godoll/stealth/evasion.go:173)
  already injects fonts/mediaDevices/battery/codecs/plugins via
  `scriptMockFonts`/`scriptMockMediaDevices`/`scriptMockBattery`/`scriptMockCodecs`/
  `scriptMockPlugins`. **BUT it only runs `if em.fingerprint != nil`, and rod-cli's
  `createPage` never calls `em.SetFingerprint()`** (only SetProfile + SetNoiseSeed +
  Apply at context.go:629-633) — so `em.fingerprint` is nil and ALL these injectors are
  SKIPPED today. The vectors are currently unhardened in rod-cli. This is the gap.

### Approach — godoll-driven + coherent (the operator's steer: "godoll should handle this")
- **D-03:** Wire it by generating the godoll fingerprint **constrained to the active
  profile's OS + locale** using `fingerprint.FPWithOS(<os from profile.Platform>)` and
  `FPWithLocales(<profile locale>)` (generator options confirmed to exist:
  generator.go:531-538), then calling `em.SetFingerprint(fp)` before `Apply()`. This
  makes godoll inject **OS-coherent** fonts/devices/codecs (Windows fonts on a Windows
  profile, macOS fonts on a Mac profile) instead of the unconstrained random fingerprint.
  Map `profile.Platform` → OS string: Win32→"windows", MacIntel→"macos".
- **D-04 (coherence is mandatory):** NEVER enable godoll's unconstrained random
  fingerprint on a pinned profile — that risks e.g. Linux fonts on a Windows identity.
  Always constrain to the profile's OS/locale (see the `coherent-not-random` lens).

### Toggles (EVAD-02) — per-vector, v1.6 precedence
- **D-05:** Add per-vector `*bool` toggles in `StealthConfig` following the exact
  `CanvasNoise`/`WebRTCLeakProtection` pattern (config.go:106-113) — e.g. `FontSpoof`,
  `MediaDevicesSpoof`, `BatterySpoof`, `CodecSpoof` — each resolved CLI>profile>default
  via the existing funnel, **default ON** (match v1.6 hardening defaults). EVAD-02 needs
  ≥2 independently-toggleable surfaces; we expose 4. To gate per-vector, modify godoll's
  `applyFingerprintDimensions` to accept a per-dimension options/mask (godoll is vendored
  `replace => ../godoll`, so this is allowed) OR have rod-cli zero out the fingerprint
  fields whose toggle is off before SetFingerprint. Engineer's discretion on mechanism;
  the bar is ≥2 genuinely independent toggles that the harness can prove on/off.

### Stability (success criterion 4)
- **D-06:** Within a session the injected values must be stable on re-read (consistent
  hash) — satisfied by injecting once via EvalOnNewDocument with a fixed fingerprint;
  use a seeded generator tied to the session (reuse `ctx.noiseSeed` discipline) so a
  recreated page in the same session reproduces the same dimensions. Assert stability in
  the harness (read a vector twice, compare).

### Harness assertions (EVAD-02 "asserted by harness")
- **D-07:** Add probes to `internal/detect/probe.js` for each vector — font presence/
  measureText signature, `mediaDevices.enumerateDevices()` count/labels,
  `navigator.getBattery()` presence+values, codec `canPlayType` results — read back
  from the LIVE page (validate-live-not-source). Harness asserts (a) the vector is
  coherent/non-headless-default when the toggle is ON, and (b) stable on re-read.

### Claude's Discretion
- Exact toggle names + flag spellings; the per-dimension gating mechanism (godoll
  options vs rod-cli field-zeroing); whether to assert all 4 or a representative subset
  in the slow harness (consistency can cover the rest); exact probe signatures.
- Which OS font/device/codec sets godoll produces — accept godoll's OS-constrained
  output rather than hand-authoring lists, unless a set is incoherent.

</decisions>

<disciplines>
## Lenses surfaced for this phase

> **Lenses, not rules.** Downstream agents engage: adopt, refine, or reason past with cause.

~~~yaml
disciplines:
  - section: "inline"
    name: "coherent-not-random"
    scope: ["types/context.go createPage (SetFingerprint wiring)", "the 4 new hardening vectors"]
    guidance: "When enabling godoll's fingerprint dimensions, constrain the generator to
      the active profile's OS + locale (FPWithOS/FPWithLocales). The injected fonts/
      devices/codecs must tell the SAME OS story as the profile's UA/platform — never
      enable an unconstrained random fingerprint on a pinned identity."
    rationale: "godoll's dimensions are random by default; an unconstrained fingerprint
      could put Linux fonts or a 32-core mobile codec set on a Windows-desktop profile —
      exactly the cross-layer incoherence detectors look for. Hardening a vector
      incoherently is worse than leaving it (it adds a NEW tell)."
    override_signals:
      - "The no-profile default path (no OS pinned): a self-consistent random fingerprint
        from a single FPWithOS-less generate is acceptable, as long as ALL its dimensions
        come from that one coherent draw (not mixed sources)."
~~~

</disciplines>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap
- `.planning/REQUIREMENTS.md` — EVAD-01 (identify+prioritize ≥2 new vectors), EVAD-02 (≥2 new hardening surfaces implemented + harness-asserted), EVAD-03 (toggles follow CLI>profile>default + documented in stealth-config).
- `.planning/ROADMAP.md` §"Phase 33: Advanced Evasion" — goal + 4 success criteria (note criterion 4: stable within session).

### godoll (the engine that already has the injectors)
- `../godoll/stealth/evasion.go` — `Apply()` (81; calls `applyFingerprintDimensions` only when `em.fingerprint != nil`), `applyFingerprintDimensions()` (173; fonts/codecs/battery/mediaDevices/plugins), `SetFingerprint()` (74).
- `../godoll/fingerprint/generator.go` — `NewFingerprintGenerator` (91), `FPWithOS` (531), `FPWithLocales` (538), `FPWithDevices` (535); `../godoll/fingerprint/types.go` — Fingerprint fields (Fonts, Battery, MediaDevices, VideoCodecs/AudioCodecs, PluginsData).
- `../godoll/stealth/profile.go` — Profile.SpoofCanvas/SpoofAudioContext pattern (the per-dimension gating precedent).

### rod-cli wiring points
- `types/context.go` — createPage (~617-655): where `fp` is generated (currently set on ctx.fingerprint but NOT on em); add `em.SetFingerprint(fp)` with OS/locale-constrained generation + per-toggle gating.
- `types/config.go` — `StealthConfig` toggles (CanvasNoise/WebRTCLeakProtection pattern, 106-113); `ResolveStealth` funnel; `StealthFlags` (235+) for CLI flags.
- `cmd.go` — global flag block + daemon-argv forwarding (the canvas/webrtc flags are the template).
- `internal/detect/probe.js` + `tests/detection_test.go` — the offline harness to extend with per-vector probes + stability/coherence assertions.
- `docs/stealth-config.md` — document the new toggles (EVAD-03).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- godoll `applyFingerprintDimensions` + `scriptMock*` — the injectors already exist;
  do NOT rewrite them.
- godoll generator `FPWithOS`/`FPWithLocales` — coherent OS-constrained generation.
- `StealthConfig` `*bool` + `boolVal(x, default)` toggle pattern + the ResolveStealth
  precedence funnel (reuse verbatim for the 4 new toggles).
- `ctx.noiseSeed` — the per-session seed discipline for stability.
- `internal/detect/probe.js` — the single source of truth shared by harness +
  stealth-check; add the new probes here so both see them.

### Established Patterns
- Hardening toggles are `*bool`, default-on via boolVal, resolved once at daemon spawn.
- Evasion failures log-and-continue (never abort the daemon) — match for the new wiring.
- validate-live-not-source: assert by reading the vector back from the live page.

### Integration Points
- `em.SetFingerprint(fp)` in createPage is the one missing call that activates the
  whole godoll dimension path.
- The 6 built-in profiles (Phase 32) should each yield coherent dimensions via FPWithOS
  keyed on their platform.

</code_context>

<specifics>
## Specific Ideas

- The headline: 4 fingerprint vectors that leak headless defaults today become
  coherent, OS-matched, stable, toggleable, and harness-asserted — by activating
  godoll code rod-cli already ships but never calls (the nil-fingerprint gap).
- All real Chrome; no TLS ([[no-tls-spoofing-real-chrome]]).

</specifics>

<deferred>
## Deferred Ideas

- Inventing new JS evasion beyond godoll's injectors — not needed; godoll covers the
  chosen vectors.
- Additional vectors beyond the 4 (e.g. ClientRects, speech voices) — future if the
  harness surfaces them.

</deferred>

---

*Phase: 33-advanced-evasion*
*Context gathered: 2026-06-26*
