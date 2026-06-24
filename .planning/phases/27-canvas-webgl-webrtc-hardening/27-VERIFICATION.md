---
author: qa
responsible: qa
phase: 27-canvas-webgl-webrtc-hardening
verified: 2026-06-24T00:00:00Z
status: passed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: none
  previous_score: n/a
requirements:
  - id: HARDEN-01
    status: satisfied
    evidence: "Both WebRTC legs fire on the default path and are gated on the single toggle. Browser-pref leg: launchBrowser (types/context.go:176-178) `if boolVal(cfg.Stealth.WebRTCLeakProtection, true) { opts = opts.WithWebRTCLeakProtection(true) }`, which sets webrtc pref ip_handling_policy=disable_non_proxied_udp (godoll browser/options.go:112-117). JS leg: createPage (types/context.go:535-539) calls em.EvadeWebRTC() inside `if boolVal(ctx.config.Stealth.WebRTCLeakProtection, true)`, log-and-continue (VALIDATE-03). EvadeWebRTC injects scriptEvadeWebRTC (godoll js.go:342) clearing iceServers + overriding getConfiguration. Apply() previously omitted both — now wired by rod-cli. Live harness webrtc_ice PASS: required-green no-leak assertion (net.ParseIP rejects any routable token, loud-fail on undefined, default-deny on unrecognized)."
  - id: HARDEN-02
    status: satisfied
    evidence: "ApplyCanvasNoise is no longer a no-op (godoll webgl.go:60-65 injects scriptCanvasNoise(w.seed)). Canvas + audio noise are seeded deterministic pure functions of (seed, byte-index) — mulberry32-style hash of (seed^idx) returning {-1,+1} for canvas (js.go:165-173) and [-1,1) scaled by 1e-7/0.1 for audio (js.go:297-306); no Math.random in any injected script, no global state. toDataURL reads RAW via saved-original getImageData and serializes a throwaway noised canvas (source never mutated → stable re-reads, js.go:194-213); getChannelData copies-then-perturbs (CR-01 fix, js.go:319-324). One toggle (CanvasNoise) gates BOTH via profileFromStealth canvasNoise→SpoofCanvas+SpoofAudioContext (context.go:458-460); seed generated once per Context via crypto/rand (context.go:261-269). Live harness canvas_noise_stable PASS (two toDataURL reads byte-identical AND a flat rgb(128) fill reads back >=1 perturbed byte) + audio_noise_stable PASS (two getChannelData reads sample-identical)."
---

# Phase 27: Canvas/WebGL/WebRTC Hardening — Verification Report

**Phase Goal:** The two genuine godoll-gap fixes land — WebRTC local-IP leak prevention is wired so a real host IP cannot leak past a proxy, and the no-op `ApplyCanvasNoise` stub is replaced with canvas/WebGL/audio noise stable within a session — with the harness asserting both the absence of a leaked host IP and identical noise hashes across re-reads.
**Verified:** 2026-06-24
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (mapped to ROADMAP Success Criteria)

| # | Truth (Success Criterion) | Status | Evidence |
|---|---------------------------|--------|----------|
| 1 | With a proxy active, a WebRTC-leak probe reads back no real local/host IP; `EvadeWebRTC` + `WithWebRTCLeakProtection` are ACTUALLY called by the evasion path (previously omitted by `Apply()`), gated on the toggle and firing on the default path | ✓ VERIFIED | Both legs traced to real call sites: launchBrowser (context.go:176-178) gates `WithWebRTCLeakProtection(true)` on `boolVal(cfg.Stealth.WebRTCLeakProtection, true)`; createPage (context.go:535-539) gates `em.EvadeWebRTC()` on the same toggle, log-and-continue. Default path: nil resolves to true (config.go:240-244), so the no-flag daemon fires both. godoll `Apply()` itself still does NOT call them (confirmed evasion.go) — rod-cli does the wiring, exactly the phase intent. Live `webrtc_ice` subtest PASS (required-green; `net.ParseIP` rejects any routable token; loud-fail retained on `undefined`). |
| 2 | Canvas, WebGL, audio noise exposed as profile toggles; two reads of canvas/audio within a session are identical (stable-per-session), and the noise is genuinely applied (not a no-op that trivially stabilizes) | ✓ VERIFIED | `canvas_noise_stable` PASS: asserts r1===r2 on two `toDataURL` reads (stability) AND a separate flat `rgb(128)` fill reads back >=1 perturbed byte via `getImageData` (applied-ness — closes the "stable no-op" hole). `audio_noise_stable` PASS: two `getChannelData` reads sample-identical (CR-01 compounding-drift regression guard). Both subtests verified to RUN (verbose), not skip. `ApplyCanvasNoise` body now injects `scriptCanvasNoise(w.seed)` (webgl.go:60-65) — the no-op stub is gone. CanvasNoise gates both surfaces (context.go:458-460). |
| 3 | Spoofed noise values fall in plausible ranges (subtle ±1 LSB, no per-pixel-uniform pattern, no impossible GPU/renderer); a deterministic pure function of (seed, index), not random-per-call | ✓ VERIFIED | Canvas `__delta(idx)` (js.go:165-173): pure hash of `(seed^idx)` → `{-1,+1}` only, applied ±1 to the R channel every 4th byte, clamped [0,255] — bounded, subtle, per-index-varying (NOT per-pixel-uniform). Audio `__afrac(idx)` (js.go:297-306): same hash → [-1,1) scaled by 1e-7 (channel) / 0.1 (freq) — bounded. NO `Math.random` in any injected script (the lone hit is a doc comment), NO global PRNG state — same (seed,index) always maps to the same delta, so re-reads are identical, not random-per-call. WebGL `MockVendorRenderer` reports plausible "Intel Inc."/"Intel Iris OpenGL Engine" (webgl.go:30-46, evasion.go:140-141) — no impossible GPU. Enabling hardening adds ±1 LSB, not entropy. |

**Score:** 3/3 success criteria verified (0 present-behavior-unverified)

### Lead-added must-haves

| Must-have | Status | Evidence |
|-----------|--------|----------|
| Both `--webrtc-protection` / `--canvas-noise` flags exist (default-on); `*bool` precedence: explicit flag > yaml-persisted value (incl. explicit false) > default true; round-trip test proves persisted-false survives and omitted resolves true | ✓ VERIFIED | Flags registered Value:true (cmd.go:135-136), conditionally forwarded only on `c.IsSet` (cmd.go:60-61), captured daemon-side as `*bool` only on `c.IsSet` (main.go:52-58) — the tri-state spine. `StealthConfig` fields are `*bool` (config.go:104,111); `ResolveStealth` honors non-nil yaml (incl. false), only nil resolves to true (config.go:240-249). `TestResolveStealth_HardeningTogglesRoundTrip` PASS uncached: `persisted_false_survives` (unmarshals real yaml `false`, asserts non-nil-false precondition, then survives), `omitted_resolves_true`, `flag_overrides_persisted` — all genuine, falsifiable. `--help` lists both with `(default: true)`. |
| CanvasNoise toggle gates BOTH canvas and audio (profileFromStealth sets SpoofCanvas + SpoofAudioContext) | ✓ VERIFIED | profileFromStealth (context.go:458-460): `canvasNoise := boolVal(s.CanvasNoise, true); p.SpoofCanvas = canvasNoise; p.SpoofAudioContext = canvasNoise`. One toggle, both surfaces. Apply() gates audio on SpoofAudioContext (evasion.go:131) and canvas on SpoofCanvas (evasion.go:151). |
| Session noise seed generated ONCE per Context (crypto/rand), stable across createPage calls | ✓ VERIFIED | `Context.noiseSeed uint64` generated once in NewContext via `crypto/rand` (context.go:261-269), with a non-constant fallback (`time.Now().UnixNano()^os.Getpid()`) on the near-impossible rand error (CR-05 fix — no predictable seed 0). One daemon = one seed; threaded via `em.SetNoiseSeed(ctx.noiseSeed)` (context.go:527) before every `em.Apply()`, so all createPage calls share it → stable re-reads. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `types/config.go` | `*bool` toggle fields + StealthFlags + ResolveStealth precedence + boolPtr/boolVal | ✓ VERIFIED | Fields 104/111, flags 159/162, precedence 240-249, helpers 505-512, DefaultConfig boolPtr(true) 528. |
| `cmd.go` | 2 BoolFlags + conditional forwarding | ✓ VERIFIED | Flags 135-136, IsSet forwarding 60-61, stealthRequested OR 78. |
| `main.go` | daemon-side *bool capture | ✓ VERIFIED | IsSet capture into stealthFlags 52-58. |
| `types/context.go` | noiseSeed + WithWebRTCLeakProtection + EvadeWebRTC + SpoofCanvas/Audio wiring | ✓ VERIFIED | Seed 240/261-269, WebRTC pref 176-178, EvadeWebRTC 535-539, SpoofCanvas/Audio 458-460, SetNoiseSeed 527. |
| `../godoll/stealth/js.go` | seeded scriptCanvasNoise/scriptAudioContextNoise (no Math.random in scripts) | ✓ VERIFIED | scriptCanvasNoise 157, scriptAudioContextNoise 289, scriptEvadeWebRTC 342; deterministic (seed,index) hashes; 0 Math.random in injected bodies. |
| `../godoll/stealth/webgl.go` | ApplyCanvasNoise replaces no-op | ✓ VERIFIED | seed field 15, SetSeed 24, ApplyCanvasNoise injects scriptCanvasNoise(w.seed) 60-65. |
| `../godoll/stealth/evasion.go` | noiseSeed + SetNoiseSeed + gated canvas/audio + EvadeWebRTC | ✓ VERIFIED | noiseSeed 49, SetNoiseSeed 69, audio gate 131, seed thread 139, canvas gate 151, EvadeWebRTC 307. |
| `../godoll/stealth/profile.go` + `fingerprint_bridge.go` | SpoofCanvas gate | ✓ VERIFIED | grep confirms SpoofCanvas field + DefaultProfile false + FromFingerprint true (Plan 02). |
| `internal/detect/detect.js` + `tests/detection_test.go` | canvasHash probe + flipped webrtc_ice + canvas/audio stable subtests | ✓ VERIFIED | webrtc_ice required-green (no `webrtc_ice_known_red`), canvas_noise_stable + audio_noise_stable with dual stability+applied/loud-fail assertions. |

## Requirement-ID Accounting

| ID | Status | Verified Against |
|----|--------|------------------|
| HARDEN-01 | ✓ SATISFIED | Both WebRTC legs wired + gated + default-on + live `webrtc_ice` no-leak PASS. |
| HARDEN-02 | ✓ SATISFIED | Seeded stable-per-session canvas+audio noise (no-op stub replaced) + live `canvas_noise_stable` + `audio_noise_stable` PASS. |

Both Phase-27 requirement IDs accounted for and satisfied. No orphan requirements.

## Evidence (commands run, READ-ONLY verification)

```
$ go build -o rod-cli .                              # clean
$ (cd ../godoll && go build ./...)                   # clean
$ go test ./types/ -run TestResolveStealth_HardeningTogglesRoundTrip -count=1 -v
  --- PASS: .../persisted_false_survives
  --- PASS: .../omitted_resolves_true
  --- PASS: .../flag_overrides_persisted
  ok  types  0.005s
$ go test ./types/                                   # ok
$ (cd ../godoll && go test ./stealth/)               # ok
$ pkill -9 -x rod-cli; go build -o rod-cli .
$ go test ./tests/ -run TestDetectionHarness -count=1 -v   # ALL 13 subtests RUN + PASS:
  --- PASS: TestDetectionHarness/webrtc_ice
  --- PASS: TestDetectionHarness/canvas_noise_stable
  --- PASS: TestDetectionHarness/audio_noise_stable
  ok  tests  26.151s
$ go test ./tests/ -run TestNetworkEvasionHeaders -count=1     # ok (no header regression)
$ ./rod-cli --help | grep -E 'webrtc-protection|canvas-noise'  # both listed (default: true)
$ pkill -9 -x rod-cli
```

Per operational constraints: did not run full `go test ./...`; killed daemons only with `pkill -9 -x rod-cli`; rebuilt binary immediately before every runCli-based harness run.

## Gaps

None blocking. Two known deferrals, neither material to this phase's goal:

- **CR-06 (deferred nicety):** the canvas delta keys on read-relative byte index, so a sub-region `getImageData` and a full `toDataURL` assign the same physical pixel a different delta. Full-canvas reads agree (the harness path and the dominant fingerprinting vector). Cross-API sub-region consistency does not affect within-session stability of either surface independently and is not a success criterion. **Does NOT block the phase goal.**
- **godoll `Apply()` `go em.EnableRequestInterception()` commented out (evasion.go:167):** PRE-EXISTING, not introduced this phase. Sec-Ch-Ua header spoofing depends entirely on rod-cli's own HijackRequests router. Verified NOT silently lost: `TestNetworkEvasionHeaders` PASS (Sec-Ch-Ua present and UA-derived) and harness `client_hints_ua_derived` PASS. The comment is intentional (upstream manages its own router). **Does NOT block the phase goal.**

## Verdict

**PASSED.** Both success-criteria-bearing requirements (HARDEN-01, HARDEN-02) and all three lead-added must-haves are verified true against the live codebase — not against SUMMARY claims. The two godoll gaps are genuinely closed: WebRTC leak protection is wired on both legs and asserted no-leak live; the `ApplyCanvasNoise` no-op is replaced with seeded, bounded (±1 LSB), deterministic-per-(seed,index) canvas+audio noise that is stable within a session and genuinely applied, asserted by dual stability+applied-ness harness subtests. The CR-01/CR-02/CR-03/CR-04/CR-05 review fixes are all present and effective in the code.
