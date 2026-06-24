# Phase 27: Canvas/WebGL/WebRTC Hardening - Context

**Gathered:** 2026-06-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Land the two genuine godoll-gap fixes (HARDEN-01, HARDEN-02):

1. **WebRTC local-IP leak prevention** — wire godoll's existing-but-unwired
   `EvasionManager.EvadeWebRTC()` (JS RTCPeerConnection wrapper) AND
   `BrowserOptions.WithWebRTCLeakProtection()` (browser preference: disable
   non-proxied UDP) so a real host IP cannot leak past a proxy.
2. **Stable canvas/WebGL/audio noise** — replace the no-op `ApplyCanvasNoise`
   stub (and the equally no-op `scriptCanvasNoise` / `Math.random()`-based
   `scriptAudioContextNoise`) with noise that is **stable within a session**:
   re-reads return an identical hash, driven by a per-session seed.

The harness (`tests/detection_test.go` + `internal/detect/detect.js`) asserts
both: absence of a leaked host IP (`webrtcIce == ""`) and identical noise hashes
across re-reads.

Out of scope for this phase: WebGL `readPixels` noise (deferred to v2), human-
behavior tuning (Phase 28), live WAF validation (Phase 29).

</domain>

<decisions>
## Implementation Decisions

### WebRTC Protection Wiring
- Call `em.EvadeWebRTC()` in `types/context.go createPage()` immediately after
  the existing `em.Apply()` (rod-cli stays in control; godoll changes stay
  minimal and uncommitted-by-design per the phases 24–26 `replace => ../godoll`
  pattern). Surface its error via the same log-and-continue path as `Apply()`.
- Call `.WithWebRTCLeakProtection(true)` on the `browser.BrowserOptions` in
  `launchBrowser()` (`types/context.go`), alongside the existing stealth preset,
  gated on the new config toggle.
- Add `WebRTCLeakProtection bool` to `StealthConfig` (`types/config.go`),
  **default true**, with a `--webrtc-protection` CLI flag (default-on, user can
  pass `--webrtc-protection=false` to disable). Both the JS evasion and the
  browser-preference legs are gated on this single toggle so they move together.
- Harness flip: replace the `webrtc_ice_known_red` baseline-record subtest with
  a required-green assertion `webrtcIce == ""` (detect.js records `""` when no
  ICE candidate is gathered — the EvadeWebRTC-effective clean state). Keep the
  "signal observable, not undefined" guard so a blanked probe still fails loudly.

### Canvas/WebGL/Audio Noise
- **Seed:** generate a random `uint64` session seed at browser-launch time in
  `launchBrowser()`, store it on the daemon `Config`, and inject it as a JS
  constant into the noise scripts so a seeded PRNG produces identical per-pixel /
  per-sample deltas on every read within the session (stable hash). A fresh
  session (new daemon) gets a fresh seed.
- **Canvas scope:** patch `HTMLCanvasElement.prototype.toDataURL` AND
  `CanvasRenderingContext2D.prototype.getImageData` (both primary canvas-
  fingerprint vectors). WebGL `readPixels` is a third surface but deferred to v2.
- **Audio:** include AudioContext noise in this phase — fix the existing
  `scriptAudioContextNoise` to use the session-seeded PRNG instead of
  unstable `Math.random()` (HARDEN-02 covers "Canvas/WebGL/Audio noise").
- **Toggle:** add `CanvasNoise bool` to `StealthConfig` (`types/config.go`),
  **default true**, with a `--canvas-noise` CLI flag — mirrors the
  `WebRTCLeakProtection` toggle pattern. Gates canvas + audio noise injection.

### Claude's Discretion
- Exact PRNG choice for the seeded noise (e.g. mulberry32 / xorshift in JS) and
  the noise magnitude (subtle, ±1 LSB territory — must not be visually obvious
  and must not over-noise into a tell per the "over-noised canvas is itself a
  tell" Out-of-Scope note).
- Exact mechanism for threading the uint64 seed into the injected JS (constant
  string interpolation vs. a small bootstrap), preferring the json.Marshal
  injection-boundary discipline established in Phase 26 for any interpolation.
- Harness hash-stability subtest construction (read canvas twice, assert equal;
  optionally assert two distinct sessions differ — nice-to-have, not required).

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `../godoll/stealth/evasion.go:291 EvadeWebRTC()` — already implemented, injects
  `scriptEvadeWebRTC` (clears iceServers, overrides getConfiguration). Just not
  called by `Apply()`. Wire it from rod-cli's createPage.
- `../godoll/browser/options.go:112 WithWebRTCLeakProtection(enable bool)` —
  already implemented, sets the `webrtc` preference map
  (`ip_handling_policy: disable_non_proxied_udp`). Just not called.
- `../godoll/stealth/webgl.go:47 ApplyCanvasNoise()` — the no-op stub to replace
  (comment literally says "we just return the original result to verify
  injection"). `scriptCanvasNoise` (js.go:144) is the equally-no-op const.
- `../godoll/stealth/js.go scriptAudioContextNoise` — uses `Math.random()`
  (unstable); fix to seeded PRNG.
- Phase-26 injection-boundary discipline: `scriptOverrideTimezone` /
  `scriptMockUserAgentData` use `json.Marshal` to escape interpolated values —
  reuse for any seed/string interpolation.

### Established Patterns
- `types/context.go createPage()` (line 471): builds `EvasionManager`, sets
  profile, calls `em.Apply()`, logs failures to stderr (log-and-continue, never
  abort the daemon — VALIDATE-03 pattern). New `EvadeWebRTC` call follows suit.
- `types/context.go launchBrowser()` (line ~110): builds the `launcher`, applies
  proxy via `ApplyToLauncher`, then `browser.NewBrowserOptions().WithLauncher(..)
  .SetBrowserPreferences(StealthPreset())`. `WithWebRTCLeakProtection` chains here.
- `StealthConfig` (`types/config.go:39`): the single precedence-resolved config
  struct (flag > profile > default), frozen at daemon spawn. New `bool` fields
  overlay onto the same `ResolveStealth` resolver — config.go comment lines 31-32
  already reserve "WebRTC (leak-protection toggle), CanvasNoise" for this phase.
- godoll changes stay UNCOMMITTED in `../godoll` (consumed via `replace`).

### Integration Points
- CLI flag registration: `cmd.go` / `main.go` (where `--proxy`, `--user-agent`
  etc. are declared) — add `--webrtc-protection` and `--canvas-noise`.
- Flag → Config forwarding: the same path Phase 25/26 flags use into
  `StealthConfig` via `ResolveStealth`.
- Harness: `tests/detection_test.go` (flip `webrtc_ice_known_red`, add canvas
  hash-stability subtest); `internal/detect/detect.js` (already probes
  `webrtcIce`; add a canvas-hash probe if not present).

</code_context>

<specifics>
## Specific Ideas

- The `webrtcIce` probe in `detect.js` records the SDP candidate connection-
  address (field 4); `""` means no candidate gathered = the clean post-EvadeWebRTC
  state the harness should now require.
- Noise must be subtle and stable — explicitly NOT per-call-varying (an unstable
  hash across reads is itself a detection tell, per REQUIREMENTS Out of Scope).

</specifics>

<deferred>
## Deferred Ideas

- WebGL `readPixels` noise — third canvas surface, deferred to v2.
- Cross-session noise differentiation as a hard requirement — nice-to-have
  harness assertion only; within-session stability is the v1.6 must-have.

</deferred>
