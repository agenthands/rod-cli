---
author: engineer
responsible: engineer
phase: 33-advanced-evasion
plan: 02
wave: 2
status: complete
requirements: [EVAD-02, EVAD-03]
depends_on: ["33-01"]
---

# Phase 33 wave 2 — SUMMARY (harness assertions + probes + docs)

## What landed

Proved the wave-1 hardening from the LIVE page (EVAD-02 "asserted by harness") and
documented the toggles (EVAD-03).

### `internal/detect/probe.js` + `internal/detect/detect.js`
The two parallel probe scripts (probe.js = injected by the `stealth-check`
command; detect.js = the harness fixture page that populates `window.__detect`)
both gained, kept in sync:
- **sync probes:** `fonts` (measureText width signature) and `codecs` (canPlayType
  signature for representative MIME types).
- **async probes:** `mediaDevices` (`enumerateDevices()` → `"<count>:<sorted kind
  set>"`) and `battery` (`getBattery()` → present + level + charging), each wired
  into the readiness gate (`window.__detect.ready`) defensively (settle on every
  resolve/reject/absent/throw path; 3s timeout fallback).
- detect.js gate count `pending 2 → 4`; probe.js uses a reference-counted gate
  with a **registration guard** so it cannot flip `ready` early if a probe settles
  synchronously (review fix).

### `tests/detection_test.go`
- **`TestAdvancedEvasionDimensions`** (live, offline, loopback fixture, zero
  egress) — 5 subtests:
  - `media_devices_coherent_on` — enumerateDevices count ≥1 with only known kinds.
  - `battery_coherent_on` — level ∈ [0,1] + boolean charging (skips if getBattery
    genuinely absent; the injector only overrides an existing API).
  - `codecs_coherent_on` — at least one representative codec reports support.
  - `media_devices_consistent_on_reread` — a (deliberately weak) live guard that
    the injected vector isn't re-randomized/blanked on re-read.
  - `media_devices_toggle_off_reverts` — captures ON signature, respawns with
    `--media-devices-spoof=false`, asserts OFF ≠ ON (toggle provably effective).
- **`TestSeededFingerprintDimensions`** (browser-free, instant) — the LOAD-BEARING
  proof of two guarantees the live harness can't establish:
  - **success criterion 4 (stability):** same seed ⇒ byte-identical dimensions
    (fonts/devices/codecs/battery), so a page recreated in one session reproduces
    them. (The live re-read is weak because `ctx.page` is created once per session
    and `close` regenerates the seed — see review note below.)
  - **coherent-not-random:** OS-constrained generation — a Windows draw contains
    NO mac/linux-only fonts (and symmetrically). This is the coherence proof that
    the no-op live font injector blocks.

### `docs/stealth-config.md`
Added §2 "Advanced fingerprint-dimension toggles (Phase 33, default ON)" with the
4-flag table, the coherent-per-profile-OS explanation, on/off guidance, the
performance note, real-Chrome/no-TLS cross-ref, and the honest font caveat;
extended §5 (`*bool` pattern) to include the 4 new toggles + the createPage wiring.

## must_haves.truths — status
- probe.js gains live probes for fonts/mediaDevices/battery/codecs — **DONE**
  (mirrored in detect.js, the fixture-page script the harness actually reads).
- Harness asserts each vector coherent/non-headless-default when ON, from the live
  page — **DONE** (mediaDevices/battery/codecs; ≥2 EVAD-02 surfaces).
- STABILITY proof for criterion 4 — **DONE** via `TestSeededFingerprintDimensions`
  (seeded determinism) + the live consistency guard.
- TOGGLE-OFF proof for ≥1 vector — **DONE** (`media_devices_toggle_off_reverts`).
- docs document the 4 toggles (names, default ON, precedence, on/off, perf,
  coherent-per-OS, real-Chrome/no-TLS) — **DONE**.
- offline/deterministic, default `go test`; build clean — **DONE**.

## Live evidence captured
- Windows default profile, toggles ON: `enumerateDevices` = 5 devices
  [audioinput, audiooutput, videoinput]; `getBattery` = level 0.38, charging
  false; mp4 codec = "probably" — all coherent / non-headless-default.
- `--media-devices-spoof=false`: enumerateDevices = 3 devices (headless default) →
  differs from ON's 5 (toggle effective; note count, not kinds, is the
  discriminator — the test signature is count+kinds).
- `--battery-spoof=false`: getBattery = level 1.0, charging true (the textbook
  headless tell) vs ON's 0.38/false.

## Independent review (wave 2)
Spawned `anvil-code-reviewer` → FIX FIRST (0 blockers, 1 major, several minors).
All accepted + fixed:
- **[MAJOR] tautological stability subtest** — the live re-read proves nothing
  about seeded stability (memoized value, single page per session). FIXED:
  reframed the live subtest honestly as a weak consistency guard and added
  `TestSeededFingerprintDimensions` as the real criterion-4 proof (also covers the
  OS-coherence I couldn't prove live). Note: the reviewer's suggested
  close+re-goto fix would NOT work here — `close` regenerates the seed, so a new
  session legitimately yields different dimensions.
- **[MINOR] probe.js early-flip** — added a registration guard so `ready` can't
  flip before all probes register.
- **[MINOR] doc battery overclaim** — scoped the "harness proves this" clause to
  mediaDevices; battery toggle-off shown as illustrative values.
- **[MINOR] toggle-off discriminator fragility** — acceptable as-is (count+kinds
  signature; evidence above confirms ON=5 vs OFF=3).
- **[NIT] probe.js/detect.js divergence** — acknowledged; the two were already
  hand-synced duplicates. Kept the gate semantics equivalent; full unification
  deferred.

## Regression
- `TestDetectionHarness` (the v1.6 harness): 17/17 subtests PASS after the
  detect.js gate refactor (no regression to canvas/audio/webgl/webrtc/consistency).
- `stealth-check` smoke: PASS with the updated probe.js readiness guard.
- `go build ./...`, `go vet ./tests/ ./internal/detect/` clean.

## Coverage / honesty note (routed to architect)
- **Fonts is NOT live-asserted.** godoll's `scriptMockFonts` returns the original
  `measureText` width on every branch (observable no-op), so the FontSpoof toggle
  gates injection but produces no live-readable change. OS-coherence of the font
  *set* is instead proven at the generation layer (`TestSeededFingerprintDimensions`
  os_coherent_fonts). The 2+ harness-proven EVAD-02 surfaces are mediaDevices +
  battery (+ codecs). If a future godoll makes the font injector observable, add a
  live fonts coherence assertion. This is a candidate follow-up godoll fix.
