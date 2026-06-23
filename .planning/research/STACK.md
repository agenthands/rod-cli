# Stack Research

**Domain:** Go browser-automation CLI — proving & extending godoll stealth (v1.6)
**Researched:** 2026-06-24
**Confidence:** HIGH (godoll APIs verified by reading `../godoll` source directly; go-rod CDP APIs verified in module cache; external detection pages verified via web)

## TL;DR for the roadmapper

**Almost everything v1.6 needs already exists in godoll and go-rod — the work is wiring, configuration, and validation, not new dependencies.** The recommended *new third-party additions are essentially zero*: no new Go module is strictly required. The two genuinely-new code areas are (a) a bundled detection test harness (built with Go stdlib `net/http` + `embed`, mirroring the existing `internal/plugin/scanner/testserver`), and (b) filling one stub (`stealth.ApplyCanvasNoise` is a no-op) plus wiring methods that godoll exposes but `createPage()` never calls (`EvadeWebRTC`, `SpoofTimezone`, proxy-auth relay, humanize tuning options).

A per-feature "build vs. just-wire" map is in the table at the end.

---

## Recommended Stack

### Core Technologies (already in `go.mod` — no version change)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `github.com/agenthands/godoll` | `v0.0.0-20260314...` (local `replace ../godoll`) | Definitive driver wrapper: stealth, fingerprint, proxy, humanize, network | Already the project's driver layer. v1.6 is *proving and exposing* its existing APIs, not replacing it. |
| `github.com/go-rod/rod` | `v0.116.2` | CDP transport + `proto.Emulation*` overrides | Provides `EmulationSetTimezoneOverride`, `…GeolocationOverride`, `…LocaleOverride`, `…DeviceMetricsOverride`, `…UserAgentOverride` — all present and verified in v0.116.2. Engine-level pinning rides on these. |
| Go toolchain | `1.25.1` declared (built on 1.26) | Single static binary, `embed`, `net/http` | Constraint requires zero Node/Python runtime. The detection harness must be pure-Go stdlib. (Note: PROJECT.md says "Go 1.23+" but `go.mod` already declares `go 1.25.1` — harmless, but flag for doc consistency.) |

### godoll Sub-Packages & APIs the v1.6 features map onto

These are **existing godoll exports** — the roadmap should treat them as the integration surface, not as things to build.

| godoll API (verified in source) | Feature it serves | Status in `rod-cli` today |
|---|---|---|
| `browser.ProxyConfig{Protocol,Address,Username,Password}` + `.ApplyToLauncher(l)` (returns cleanup) + `.SetupBrowserAuth(b)` | **Proxy (HTTP/SOCKS5 + auth)** | **NOT used.** `launchBrowser` calls raw `launcher.Proxy(cfg.Proxy)` — no SOCKS parsing, no auth. |
| `browser.StartProxyRelay(upstream)` / `ProxyRelay.Stop()` | Authenticated-proxy CONNECT relay (Chrome headless can't auth natively) | **NOT used.** Already fully implemented + tested in godoll `browser/proxy.go`. |
| `fingerprint.NewFingerprintGenerator(opts...)` with `FPWithBrowserNames`, `FPWithOS`, `FPWithLocales`, `FPWithDevices`, `WithScreen` | **Configurable fingerprints** (constrained generation) | Partially used — `createPage` calls it with only `FPWithBrowserNames("chrome")`. Locale/OS/screen constraints unused. |
| `stealth.Profile` (UserAgent, Platform, Timezone, Locale, Languages, Screen, Vendor, HardwareConcurrency, DeviceMemory, Geolocation, SpoofClientHints, SpoofAudioContext) | **Pinned fingerprints** (deterministic, not random) | **NOT used as a pin source.** This struct *is* the per-session pin surface. |
| `stealth.LoadProfile(path)` + `Profile.Save(path)` | **Named-profile stealth config file** (JSON) | **NOT used.** Gives the "named-profile config file" feature almost for free. |
| `stealth.DefaultProfile()` / `stealth.FromFingerprint(fp)` | Profile defaults / derive profile from generated fp | `FromFingerprint` used indirectly via `EvasionManager.SetFingerprint`. |
| `stealth.EvasionManager.SetProfile(p)` / `.SetFingerprint(fp)` / `.Apply()` | Apply all anti-detection + profile + WebGL/canvas + fp dimensions | `SetFingerprint` + `Apply` used. `SetProfile` (pin path) **NOT used.** |
| `stealth.EvasionManager.EvadeWebRTC()` | **WebRTC IP-leak prevention (JS layer)** | **NOT called.** `Apply()` does *not* invoke it; must be called explicitly. |
| `stealth.EvasionManager.SpoofTimezone(tz)` (CDP `EmulationSetTimezoneOverride`) | Engine-level timezone pin | **NOT called** as a standalone; `Apply()` injects a JS timezone override only. |
| `browser.BrowserOptions.WithWebRTCLeakProtection(true)` | **WebRTC leak prevention (Chrome-pref layer: `disable_non_proxied_udp`)** | **NOT used.** Belt-and-suspenders with `EvadeWebRTC()`. |
| `stealth.WebGLManager.MockVendorRenderer(vendor,renderer)` | **WebGL vendor/renderer spoof** | Used via `Apply()`. Works. |
| `stealth.WebGLManager.ApplyCanvasNoise()` | **Canvas noise** | **STUB / NO-OP.** Source literally returns the original `toDataURL` result ("For now we just return the original result to verify injection"). Genuine gap. |
| `humanize.TypeWithHumanize(el,text, WithTypingSpeed(min,max), WithTypoRate(r))` | **Typing-speed / typo tuning** | Called with **no options** in `actions.go`. Knobs exist, unexposed. |
| `humanize.ClickWithMouse / MoveTo(…, WithMouseSpeed, WithMouseSteps, WithMouseDeviation, WithMouseTremor)` | **Mouse-path realism tuning** | Called with no options. Knobs exist, unexposed. |
| `humanize.ScrollBy(…, WithPhysics(), WithDuration(d))` | **Scroll realism tuning** | Called with no options. Knobs exist, unexposed. |

### Supporting Libraries (NEW — for the test harness only)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `net/http` (stdlib) | — | Serve the bundled detection page on `127.0.0.1:0` | Reuse the exact `internal/plugin/scanner/testserver` pattern (`net.Listen("tcp","127.0.0.1:0")` → `http.Serve` in a goroutine → `URL()`). |
| `embed` (stdlib) | — | Bundle the detection HTML/JS into the binary | Already used at `types/js/js.go` (`//go:embed snapshotter.js`). Add `//go:embed detect.html detect.js` in a new `internal/detect/` package. Keeps single-binary constraint. |
| `encoding/json` (stdlib) | — | Detection page posts a JSON scorecard back; harness asserts on it | Page runs probes, `fetch('/result', {body: JSON})`; Go side decodes into a `Scorecard` struct and asserts per-signal. |
| `testing` (stdlib) | — | CI-able assertions on the scorecard | Standard Go test; gated `testing.Short()` for any network/WAF target (matches godoll convention). |

**No new `go get` is required for the core milestone.** The harness is pure stdlib. (Optional non-blocking extras below.)

### Development / Validation Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| Bundled local detection page (`internal/detect/`) | **CI backbone** — deterministic, offline, no flakiness | The only thing that belongs in blocking CI. Embed + serve like testserver. |
| CreepJS (self-hosted, MIT) | Optional deep fingerprint audit | MIT-licensed, self-hostable for CI per the project README; but heavy (TS build, 100s of probes) and the **name is trademarked** (don't rebrand/ship as a product). Use as an *optional, manual / nightly* target, never the blocking gate. |
| bot.sannysoft.com / fingerprintjs-pro demo / Cloudflare/DataDome live pages | Best-effort real-world validation | **Keep OUT of blocking CI** (network-dependent, ToS-sensitive, changes without notice). Run manually or in a non-blocking nightly. sannysoft has **no cleanly-licensed self-host repo** — do not vendor it; replicate the handful of signals you care about in your own page instead. |

## Installation

```bash
# NO new third-party modules required for the core milestone.
# The test harness is pure Go stdlib (net/http, embed, encoding/json, testing).

# New in-repo package (no external dep):
#   internal/detect/            -> detect.html, detect.js (//go:embed), server.go, scorecard.go
#   internal/detect/server_test.go

# OPTIONAL (non-blocking, only if you choose to self-host CreepJS for nightly audits):
#   git submodule add https://github.com/abrahamjuliot/creepjs vendor-ext/creepjs
#   # build per its instructions; serve the built dist from a separate, non-CI target.
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| **Self-authored bundled detection page** (embed + net/http) | Vendor bot.sannysoft.com | Never — sannysoft has no clean license/source for self-hosting. Replicate its signals yourself. |
| **Self-authored page as CI gate** | CreepJS as CI gate | CreepJS is great for a *manual/nightly deep audit*, but its size + trademark + churn make it a poor deterministic CI backbone. |
| **godoll `ProxyConfig` + `ApplyToLauncher`/`SetupBrowserAuth`** | `golang.org/x/net/proxy` (SOCKS5 dialer) | Not needed — godoll already tunnels authed proxies via its own CONNECT relay. Only consider `x/net/proxy` if you needed SOCKS auth that godoll's relay didn't cover (it covers HTTP CONNECT; verify SOCKS5-auth path during impl). |
| **godoll `stealth.Profile` + `LoadProfile`/`Save` (JSON)** | Custom YAML profile loader | `Config` already uses YAML (`gopkg.in/yaml.v3`); you *may* wrap `Profile` in a YAML layer for consistency with `rod.yaml`, but JSON-via-godoll is zero-effort. Decide on file format in requirements. |
| **CDP `EmulationSetTimezoneOverride`/`…LocaleOverride`/`…GeolocationOverride`** (engine-level) | JS-only timezone/locale shims | Engine-level overrides are harder to detect than JS monkeypatches. godoll already has `SpoofTimezone` (CDP) — prefer it over the JS-only path that `Apply()` currently uses; add `EmulationSetLocaleOverride`/`…GeolocationOverride` for full pin parity (godoll does not yet call these). |

## What NOT to Use / NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Adding a new browser-driver or stealth library (playwright-go, chromedp, undetected-* ports) | godoll is the definitive driver; duplicating it fractures the stealth surface and breaks the daemon model | Extend/expose godoll. |
| Vendoring bot.sannysoft.com into the repo | No clean OSS license/source; brittle | Build the bundled page from known public signals. |
| CreepJS in **blocking** CI | Heavy, trademarked name, frequent upstream churn → flaky gate | Bundled page for CI; CreepJS optional/nightly. |
| Node/Python-based fingerprint test runners (fingerprint-suite, puppeteer-extra test pages) | Violates the zero-Node/Python constraint | Pure-Go `net/http` harness. |
| Relying on `EvasionManager.Apply()` alone for WebRTC/canvas/timezone | `Apply()` does NOT call `EvadeWebRTC`, `SpoofTimezone` (CDP), or `WithWebRTCLeakProtection`; and `ApplyCanvasNoise` is a no-op stub | Explicitly call the WebRTC + CDP-timezone methods after `Apply()`, set the WebRTC pref at launch, and **implement** real canvas noise. |
| `page.EvalOnNewDocument` for injection | godoll documents it as buggy; uses raw CDP `Page.addScriptToEvaluateOnNewDocument` via `stealth.InjectScript` | `stealth.InjectScript(page, src)` for any new JS the milestone adds. |
| Adding new modules to `humanize`/typing | All tuning knobs already exist as functional options | Plumb existing `With*` options through `actions.go` + config. |

## Stack Patterns by Variant

**If the feature is "validate stealth" (test harness):**
- New `internal/detect/` package: `embed` the page, serve via `net/http` on an ephemeral port (mirror `testserver`), have the page POST a JSON scorecard, assert in `_test.go`.
- Probe set to bundle (the classic headless tells named in PROJECT.md): `navigator.webdriver`, `navigator.plugins.length`, `navigator.languages`, `window.chrome`, `Notification.permission` vs `navigator.permissions.query({name:'notifications'})` mismatch, `HeadlessChrome` substring in UA, WebGL `UNMASKED_VENDOR/RENDERER` (37445/37446), `Intl.DateTimeFormat().resolvedOptions().timeZone`, canvas `toDataURL` stability, `navigator.userAgentData` consistency, WebRTC `RTCPeerConnection` ICE candidate leak.
- Because: deterministic + offline = the only thing safe to block CI on.

**If the feature is "proxy":**
- Parse `--proxy` into `browser.ProxyConfig{Protocol,Address,Username,Password}` (accept `http://`, `socks5://`, with optional `user:pass@`).
- In `launchBrowser`: replace `browserLauncher.Proxy(cfg.Proxy)` with `proxyCfg.ApplyToLauncher(browserLauncher)` (capture + defer the cleanup for the relay), then after `NewBrowserE` returns the browser, call `proxyCfg.SetupBrowserAuth(browserInstance)` **before** the first `Page()`/`Navigate()`.
- Because: godoll already handles the headless-auth-relay gap; you just route through it. Store the cleanup func on `Context` so `Close()` stops the relay.

**If the feature is "configurable / pinned fingerprint":**
- Random path (today): `fingerprint.NewFingerprintGenerator(FPWithBrowserNames, FPWithOS, FPWithLocales, WithScreen)` → `Generate()` → `em.SetFingerprint(fp)`.
- Pinned path (new): build a `stealth.Profile` from CLI flags / loaded profile file → `em.SetProfile(prof)` instead of `SetFingerprint`. For exact UA/locale/timezone/screen/platform pins, the `Profile` fields map 1:1.
- Engine-level pins after `Apply()`: `em.SpoofTimezone(prof.Timezone)` (CDP), `proto.EmulationSetLocaleOverride{Locale: prof.Locale}.Call(page)`, `proto.EmulationSetDeviceMetricsOverride{…}` for screen, optional `proto.EmulationSetGeolocationOverride{…}` from `prof.Geolocation`.
- Because: `Profile` + `LoadProfile`/`Save` already deliver "named profile config file, flag-overridable" with near-zero new code.

**If the feature is "canvas/WebGL/WebRTC hardening":**
- WebGL vendor spoof: already works via `Apply()` (`MockVendorRenderer`). Validate only.
- Canvas: **implement real noise** — `stealth.ApplyCanvasNoise` is a stub. Either (a) contribute a real implementation to godoll (perturb `getImageData`/`toDataURL`/`getChannelData` pixel data by ±1, seeded per session for stability), or (b) inject your own JS via `stealth.InjectScript(page, …)` from rod-cli. Prefer (a) to keep godoll the source of truth.
- WebRTC: call `em.EvadeWebRTC()` (JS) **and** set `BrowserOptions.WithWebRTCLeakProtection(true)` at launch (Chrome pref `disable_non_proxied_udp`). Neither is currently wired.
- Because: these are the three signals most-checked by CreepJS/fingerprintjs and the ones PROJECT.md calls out.

**If the feature is "human-behavior tuning":**
- Add tuning fields to `Config`/`Profile` (typing min/max ms, typo rate, mouse speed/steps/deviation/tremor, scroll physics/duration).
- Thread them as `humanize.With*` options into the existing `actions.go` calls (`TypeWithHumanize`, `ClickWithMouse`, `ScrollBy`, `MoveTo`).
- Because: zero new dependency; purely plumbing existing functional options + a config surface.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| `godoll @ local replace` | `go-rod v0.116.2` | godoll's own `go.mod` pins `go-rod v0.116.2` — identical to rod-cli. No drift. Keep the `replace ../godoll` until godoll is tagged. |
| `go-rod v0.116.2` | CDP `Emulation.set{Timezone,Locale,Geolocation,DeviceMetrics,UserAgent}Override` | All verified present in the module cache (`lib/proto/emulation.go`, `definitions.go`). `SetLocaleOverride` is marked *experimental* in the protocol but functional. |
| `embed` + `net/http` harness | Go 1.25.1 toolchain | Same stdlib pattern already used by `types/js` and `internal/plugin/scanner/testserver`. |
| `stealth.Profile` JSON | `gopkg.in/yaml.v3` (existing) | If you want YAML profiles, add a thin wrapper; otherwise godoll's JSON `LoadProfile/Save` is sufficient. Decide format in requirements. |

## Integration Points in `rod-cli` (where new wiring lands)

- `types/context.go :: launchBrowser()` — proxy parsing + `ProxyConfig.ApplyToLauncher` + relay cleanup capture; `WithWebRTCLeakProtection(true)`.
- `types/context.go :: createPage()` — after `em.Apply()`: call `em.EvadeWebRTC()`, `em.SpoofTimezone(...)`, locale/geo/screen CDP overrides; switch `SetFingerprint` ↔ `SetProfile` based on pinned-vs-random config. Store proxy cleanup + relay handle on `Context` for `Close()`.
- `types/config.go :: Config` — add `ProxyUser/ProxyPass` (or parse from `Proxy`), `StealthProfilePath`, and humanize tuning fields; YAML-tagged for the daemon-persistent flag model.
- `actions/actions.go` — thread `humanize.With*` options from config into `TypeWithHumanize`/`ClickWithMouse`/`ScrollBy`/`MoveTo`.
- `internal/detect/` (NEW) — embedded detection page + `net/http` server + scorecard struct + CI test.
- `stealth.ApplyCanvasNoise` (in `../godoll`) — replace stub with real pixel perturbation (preferred), or shadow it in rod-cli via `InjectScript`.

## Sources

- `../godoll` source (read directly, HIGH confidence): `browser/proxy.go`, `browser/options.go`, `browser/browser.go`, `stealth/evasion.go`, `stealth/profile.go`, `stealth/fingerprint_bridge.go`, `stealth/anti_detection.go`, `stealth/webgl.go`, `stealth/inject.go`, `stealth/js.go`, `fingerprint/generator.go`, `fingerprint/types.go`, `humanize/{typing,mouse,scroll}.go` — definitive API names/signatures.
- go-rod module cache `@v0.116.2` `lib/proto/emulation.go` + `definitions.go` (HIGH) — confirmed `Emulation.set{Timezone,Locale,Geolocation,DeviceMetrics,UserAgent}Override`.
- `rod-cli` `types/context.go`, `actions/actions.go`, `types/config.go`, `types/js/js.go`, `internal/plugin/scanner/testserver/server.go` (HIGH) — current wiring + the embed/testserver pattern to mirror.
- [CreepJS — github.com/abrahamjuliot/creepjs](https://github.com/abrahamjuliot/creepjs) (MEDIUM) — MIT license, self-hostable, name trademarked. Optional nightly target.
- [creepjs/LICENSE](https://github.com/abrahamjuliot/creepjs/blob/master/LICENSE) (MEDIUM) — MIT confirmation.
- [bot.sannysoft.com](https://bot.sannysoft.com/) and [puppeteer-extra #402 detection-page issue](https://github.com/berstend/puppeteer-extra/issues/402) (MEDIUM/LOW) — sannysoft has no cleanly-licensed self-host repo; replicate signals rather than vendor.

---
*Stack research for: Go browser-automation CLI stealth validation + extension (rod-cli v1.6)*
*Researched: 2026-06-24*
