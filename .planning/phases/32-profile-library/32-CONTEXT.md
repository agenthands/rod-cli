---
author: architect-lead
---

# Phase 32: Profile Library - Context

**Gathered:** 2026-06-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Ship a built-in library of vetted, coherent **Chrome-only** desktop device
profiles embedded in the binary, selectable by name (`--profile=<name>`) with a
`--profile=list` discovery command — each validated against the detection harness
and the v1.6 consistency validator before shipping. Custom user profiles and the
v1.6 CLI>profile>default precedence continue to work unchanged.

**In scope (PROF-01..04):** the curated profile set, embedding + name resolution,
`--profile=list`, the per-profile harness/consistency gate, custom-profile
compatibility.

**Out of scope:** remote profile fetch (v2 PROF-ECO-01), mobile profiles
(v2 MOB-01), profile versioning (v2). **No spoofed TLS in any profile** — all
profiles drive real Chrome whose TLS/JA3 is authentic ([[no-tls-spoofing-real-chrome]]).

</domain>

<decisions>
## Implementation Decisions

### The curated set (operator-selected 2026-06-26)
- **D-01:** Ship a Chrome-only DESKTOP set covering **Windows 11, Windows 10,
  macOS (Apple Silicon + Intel), with resolution/hardware variants** — **no Linux**
  (operator excluded it). Target ~6 profiles (within the 5-10 PROF-01 range). Concrete
  starting set (engineer may refine fields for coherence, but keep the families +
  count):
  1. `windows-11-chrome` — Win32, 1920×1080 @1.0, en-US / America/New_York, HW 8, mem 8.
  2. `windows-11-desktop-1440p` — Win32, 2560×1440 @1.0, HW 16, mem 16 (high-end desktop variant).
  3. `windows-10-chrome` — Win32, 1920×1080 @1.0, HW 8, mem 8.
  4. `windows-10-laptop` — Win32, 1366×768 @1.0, HW 4, mem 8 (laptop variant).
  5. `macos-applesilicon-chrome` — MacIntel, 2560×1600 @2.0 (Retina), HW 8, mem 16, America/Los_Angeles.
  6. `macos-intel-chrome` — MacIntel, 1920×1080 @1.0, HW 8, mem 16.
- **D-01a:** Coherence nuance: Chrome reports `navigator.platform="MacIntel"` for
  BOTH Apple Silicon and Intel Macs (Chrome doesn't expose ARM in platform), so the
  two macOS profiles differ in screen/scale + hardwareConcurrency/deviceMemory (and
  WebGL renderer, which godoll's fingerprint layer drives), not in `platform`.

### Storage + selection
- **D-02:** Built-in profiles are **embedded via `//go:embed`** as `stealth.Profile`
  JSON files in a `profiles/` dir shipped in the binary. Name resolution: a bare
  `--profile=<name>` resolves to a **built-in first**, then falls back to the
  existing `~/.rod-cli/profiles/<name>.json` / `./profiles/<name>.json` path
  (`resolveProfilePath`, types/config.go:264). A value that looks like a path
  (separator/.json/exists) is still used verbatim (custom profiles, PROF-04).
- **D-03:** `--profile=list` prints the built-in profile names (one per line; a
  short description column is fine, but keep it token-efficient / `--raw`-friendly,
  consistent with the project's quiet-output ethos [[output-noise-and-autonomy]]).
  `list` is a reserved name — it lists, it does not try to load a profile.

### Quality gate
- **D-04:** **PROF-02 is a hard gate:** a test iterates EVERY built-in profile and
  asserts it (a) passes the v1.6 consistency validator (`deriveAndValidateFingerprint`,
  types/config.go:535) and (b) passes the relevant detection-harness signals. A
  profile that fails a key signal is NOT shipped (the test fails, forcing a fix or
  removal). This is the "vetted" in PROF-01/02.
- **D-05:** **PROF-04 precedence preserved:** CLI flags override profile fields,
  which override defaults (the v1.6 `ResolveStealth` funnel). Built-in profiles
  overlay at the same Tier 2 as custom profile files — adding built-ins must not
  change the precedence semantics.

### Claude's Discretion
- Exact UA strings + Chrome major (use a CURRENT major; keep the binary current per
  the version-skew caveat in docs/stealth-validation.md). Exact timezone/locale per
  profile, exact HW/mem values, and whether to ship 6 or stretch to 7-8 variants.
- `--profile=list` output format (human + `--raw`/`--json`), and where the list
  logic lives (client vs daemon).
- Built-in embed layout (one dir, naming), and whether built-in lookup is a new
  resolver or an extension of `resolveProfilePath`.

</decisions>

<disciplines>
## Lenses surfaced for this phase

> **Lenses, not rules.** Downstream agents engage: adopt, refine, or reason past with cause.

~~~yaml
disciplines:
  - section: "inline"
    name: "ship-only-vetted"
    scope: ["profiles/*.json (built-in set)", "the PROF-02 harness/consistency gate"]
    guidance: "A built-in profile is a CLAIM that this identity blends in. Don't ship
      a profile the gate hasn't validated — if a candidate fails a key harness signal
      or the consistency validator, fix its fields or drop it from the set; never ship
      it red just to hit a profile count."
    rationale: "PROF-01/02 exist because an incoherent 'vetted' profile is worse than
      no profile — it actively fingerprints the user. The count (5-10) is a range, not
      a quota to pad."
    override_signals:
      - "A harness signal is itself flaky/environment-dependent (not a real coherence
        defect) — then document the exemption in the test rather than mutilating the
        profile to satisfy a noisy check."
~~~

</disciplines>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap
- `.planning/REQUIREMENTS.md` — PROF-01 (5-10 vetted profiles shipped), PROF-02 (each harness-tested; failing ones not shipped), PROF-03 (`--profile=list` + select by name), PROF-04 (custom profiles still work; CLI overrides).
- `.planning/ROADMAP.md` §"Phase 32: Profile Library" — goal + 4 success criteria; note Phase 31 cancelled, Phase 32 depends on Phase 30.

### Profile schema + resolution + precedence
- `types/config.go` — `StealthConfig` (the identity pin fields, ~24-130); `resolveProfilePath` (264-286, bare-name → file resolution to extend with built-in lookup); `ResolveStealth` (288+, the CLI>profile>default funnel — Tier 2 is the profile file); `deriveAndValidateFingerprint` (535, the v1.6 consistency validator the PROF-02 gate must call).
- `../godoll/stealth/profile.go` — `stealth.Profile` struct (11-60): the JSON schema each built-in profile file must match (userAgent, platform, acceptLanguage, languages, timezone, locale, screen{width,height,deviceScaleFactor}, hardwareConcurrency, deviceMemory, vendor, spoofClientHints, spoofCanvas, spoofAudioContext).
- `cmd.go` — the `--profile` flag (152) + forwarding (50); where `--profile=list` and built-in selection wire in.

### Harness + constraint
- `internal/detect/` + `tests/detection_test.go` — the offline harness each profile is validated against (PROF-02).
- `docs/stealth-validation.md` — real-Chrome-TLS stance (no spoofed TLS in profiles); the consistency/coherence story.
- `docs/stealth-config.md` — the config surface + precedence the profile flags plug into; document the built-in profiles here.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `resolveProfilePath` (types/config.go:264) — extend for built-in lookup; bare-name
  path already exists.
- `deriveAndValidateFingerprint` (types/config.go:535) — the consistency validator to
  reuse for the PROF-02 gate (don't reinvent coherence checks).
- `stealth.Profile` JSON schema — built-in files are just this struct serialized.
- The offline harness (`internal/detect/`, `tests/detection_test.go`) — drive each
  built-in through it.

### Established Patterns
- v1.6 precedence: `ResolveStealth` is the single funnel; profile files overlay at
  Tier 2. Built-ins must overlay the same way (D-05).
- A malformed/missing `--profile` is a LOUD failure (ResolveStealth aborts) — built-in
  resolution must keep that discipline.

### Integration Points
- `--profile` flag in cmd.go (client) → forwarded into daemon argv → ResolveStealth.
- `//go:embed` for the built-in set (new `profiles/` dir embedded by a Go file).

</code_context>

<specifics>
## Specific Ideas

- The "vetted" claim is the whole point: PROF-02's iterate-all-built-ins gate is the
  load-bearing deliverable — it's what makes the set trustworthy and prevents a
  future edit from silently shipping an incoherent profile.
- All profiles are real Chrome — no TLS spoofing ([[no-tls-spoofing-real-chrome]]).

</specifics>

<deferred>
## Deferred Ideas

- Remote profile repository → v2 PROF-ECO-01.
- Mobile profiles → v2 MOB-01 (desktop-only here; mobile-from-desktop is exactly the
  inconsistency detectors flag).
- Profile versioning/deprecation → v2.
- Linux/Chrome profile — operator excluded from this set (could revisit later).

</deferred>

---

*Phase: 32-profile-library*
*Context gathered: 2026-06-26*
