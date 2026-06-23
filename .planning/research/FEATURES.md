# Feature Research

**Domain:** Browser-stealth validation + configurable fingerprinting for an LLM-facing Go CLI (rod-cli v1.6)
**Researched:** 2026-06-24
**Confidence:** HIGH (godoll surface read directly from source; detection-suite + WAF behavior cross-checked across multiple 2026 sources)

## Scope Note

This milestone *proves and extends* stealth that godoll already wires. The godoll surface
(`stealth`, `fingerprint`, `humanize`, `network`, `browser` packages) was read directly from
`/home/john/go/src/github.com/agenthands/godoll`. Almost every "feature" below is **expose +
validate an existing godoll capability**, not build-from-scratch. The work is config plumbing,
a deterministic test harness, and honest scoping — not new evasion engines.

**The single most important cross-cutting fact:** modern detection scores **fingerprint
*consistency*, not individual values.** CreepJS computes a trust grade from "lies"
(values that contradict each other or native-API behavior); DataDome requires *all three*
of correct TLS + clean IP + coherent JS fingerprint and "two out of three is not enough."
This reshapes every configurable-fingerprint requirement: **a pinned field that contradicts
another pinned field is worse than not pinning at all.** See the Consistency Invariants section.

---

## Feature Landscape

### Table Stakes (Users Expect These)

Pass these and you clear `bot.sannysoft.com` and the structural half of CreepJS. Missing any
one is an instant, binary tell. godoll's `EvasionManager.Apply()` + `AntiDetection.ApplyAll()`
already emit all of them — v1.6 must **prove** them with a harness, not re-implement them.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| `navigator.webdriver` = false/undefined | The #1 binary headless tell; `true` by default in automation | LOW | godoll `AntiDetection.ApplyAll`. Harness must assert it. |
| Non-empty, realistic `navigator.plugins` / mimeTypes | Headless reports empty array; real Chrome lists PDF Viewer etc. | LOW | godoll `scriptMockPlugins` from fingerprint `PluginsData`. |
| UA without `HeadlessChrome`; UA ↔ Client Hints ↔ platform agree | UA literally containing "HeadlessChrome" is the classic tell; **mismatch is itself a tell** | LOW | godoll spoofs UA + `scriptMockUserAgentData`. Consistency is the real work (see invariants). |
| WebGL vendor/renderer not SwiftShader/llvmpipe | Headless GPU is `SwiftShader`/`Google Inc.`; real = `ANGLE (NVIDIA ...)` | LOW | godoll `WebGLManager.MockVendorRenderer`. Must match claimed OS/GPU family. |
| `navigator.permissions` query behaves natively | Headless `Notification.permission` returns inconsistent values | LOW | godoll `AntiDetection.MockPermissions`. |
| `navigator.languages` populated + matches Accept-Language | Empty/mismatched languages is a tell | LOW | godoll `Profile.Languages` + `AcceptLanguage`. Pin them together. |
| Screen dims not 800×600 / availWidth!=0 | Headless default screen is a giveaway | LOW | godoll `applyFingerprintDimensions` + `Profile.Screen`. |
| `chrome.runtime` / `window.chrome` present | Real Chrome exposes it; bare automation omits it | LOW | godoll anti-detection. Assert in harness. |
| Timezone spoof matches locale intent | `Intl` timezone disagreeing with locale/IP is a tell | LOW | godoll `EvasionManager.SpoofTimezone` / `scriptOverrideTimezone`. |
| **Repeatable bundled detection harness (CI-able)** | The deterministic backbone — without it "stealth" is unverifiable | MEDIUM | **This is the headline deliverable.** See note below. |

**Harness note (the headline table-stakes item):** Ship a self-contained, offline,
deterministic test page (vendored copy of sannysoft-style assertions + a curated subset of
CreepJS-style checks) served from a local fixture, driven by rod-cli against itself, asserting
each signal above. It must run in CI with **zero network egress** so it is deterministic and
green-or-red, not flaky. This is what turns "compiles and runs" into "provably evades." It is
also the regression net for every other feature here.

### Differentiators (Competitive Advantage)

These align with rod-cli's Core Value: *deterministic, token-efficient, agent-driven* stealth.
No competing CLI exposes stealth as agent-pinnable flags + a named-profile file + a one-line
machine-readable validation verdict.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Configurable/pinnable fingerprint** (UA, locale, timezone, screen, platform, hardwareConcurrency, deviceMemory, vendor) | Agents need *deterministic, reproducible* sessions, not random-only | MEDIUM | godoll `stealth.Profile` already has **every one of these fields** + JSON `Save`/`LoadProfile`. Work = CLI flags → Profile + consistency validation. |
| **Named stealth-profile config file**, flag-overridable, session-persistent | Daemon model: set profile once, every command inherits it; agents edit a file | MEDIUM | `stealth.Profile` JSON is literally the file format. Wire load + per-flag override + persist on the session. |
| **Machine-readable validation verdict** (`--raw` PASS/FAIL + failing-signal list) | Token-efficient: an LLM gets one line + only what failed, not a screenshot of a fingerprint page | MEDIUM | Output discipline is rod-cli's identity. Don't dump full reports; emit `webdriver=ok plugins=ok webgl=FAIL(SwiftShader)`. |
| **Per-session proxy** (`--proxy`, HTTP + SOCKS5), proxy-per-named-session | Stealth is moot if every session shares one IP; agents rotate egress per task | MEDIUM | Launcher already calls `.Proxy(cfg.Proxy)`. Extend to SOCKS5 + per-session + auth (see anti-feature on auth). |
| Canvas/WebGL noise + AudioContext noise toggles | Advanced suites hash canvas/audio; *stable* noise raises entropy without screaming "spoofed" | MEDIUM | godoll `ApplyCanvasNoise`, `SpoofAudioContext`, `scriptAudioContextNoise`. Expose as profile bools. |
| **WebRTC local-IP leak prevention** | A real local IP leaking past a proxy de-anonymizes instantly; classic mismatch | MEDIUM | godoll `EvasionManager.EvadeWebRTC` + fingerprint `WithMockWebRTC`. High value, low marginal cost. |
| Human-behavior tuning (typing speed, mouse-path realism, delays, scroll) | Behavioral models (DataDome Bezier/Fitts) catch teleporting cursors | MEDIUM | godoll `humanize` already powers scroll/type (`actions.go`). Expose knobs; defaults already humanized. |
| Font-enumeration consistency | Font list contradicting OS/UA is a CreepJS lie | LOW–MEDIUM | godoll `scriptMockFonts` + per-OS `generateFonts`. Mostly already correct; validate. |

### Anti-Features (Commonly Requested, Often Problematic)

The quality gate explicitly demands these be called out. **Do not claim to defeat what a
JS-injection CLI structurally cannot touch.**

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| "Bypasses Cloudflare / DataDome / PerimeterX" claim or **blocking CI gate** against them | Users want a turnkey WAF-beater | These score **TLS (JA3/JA4) + IP reputation + behavior** — layers a JS-injecting CLI **cannot control**. "Two of three is not enough." Live challenges are non-deterministic, IP/Cloudflare-version dependent → flaky CI. | **Best-effort, manual, non-blocking** smoke check documented as such. State plainly: rod-cli hardens the *browser/JS* layer; TLS+IP are the operator's job (proxy/uTLS upstream). |
| URL-embedded proxy credentials (`http://user:pass@host`) | Simplest mental model | Chrome **removed** URL-embedded proxy creds; SOCKS5 auth is unsupported by Chrome's `--proxy-server` | Handle `--proxy-auth` via CDP `Fetch.continueWithAuth` / `Network.setExtraHTTPHeaders` (Proxy-Authorization), or document IP-whitelisting / local forwarder. Don't pretend the flag "just works." |
| Random-everything-every-request fingerprints | "More randomness = harder to track" | Randomization that breaks **consistency** *lowers* CreepJS trust (mismatched lies). Per-request churn also breaks session cookies/state. | **Pin a coherent profile per session.** Randomize the *selection* of a whole consistent profile, never individual fields independently. |
| Over-noised canvas/audio (large/changing noise) | "Defeat the hash" | High or per-call-varying noise is itself detectable (native APIs are deterministic); unstable hash across reads = a lie | Subtle, **stable-per-session** noise (godoll's default). Validate the hash is stable within a session. |
| Spoofing mobile fingerprints from a desktop headless | Wider device coverage | Desktop Chrome reporting mobile-ish signals is exactly the "inconsistency" DataDome flags | Stay within one coherent device class per profile; ship vetted desktop profiles only. |
| Bundling/automating CAPTCHA solving | "Finish the bypass" | Scope creep, ToS/legal exposure, external paid deps, breaks zero-dependency single-binary constraint | Out of scope. Surface the block in output; let the agent decide. |
| Full CreepJS as a CI dependency (live fetch) | "Use the real thing" | Remote, evolving, network-dependent → non-deterministic CI; violates token-efficiency/offline goals | Vendor a curated **subset** of checks offline for CI; keep live CreepJS as a manual/best-effort target. |

---

## Feature Dependencies

```
[Detection test harness (offline, deterministic)]
    └──validates──> [Table-stakes signals]  (webdriver, plugins, UA, WebGL, permissions, languages, screen)
    └──validates──> [Configurable fingerprint]
    └──validates──> [Canvas/WebGL/WebRTC/Audio hardening]

[Configurable fingerprint (CLI flags)]
    └──requires──> [stealth.Profile struct]            (EXISTS in godoll)
    └──requires──> [Consistency validator]             (NEW — the real work)
    └──persisted by──> [Named stealth-profile file]    (stealth.Profile JSON Save/Load EXISTS)

[Named stealth-profile file] ──overridden by──> [Per-flag overrides]
[Per-session proxy] ──independent of──> [JS fingerprint layer]   (different layer; both needed for real evasion)
[Per-session proxy] ──must agree with──> [Timezone/locale/WebRTC]  (proxy IP geo must match spoofed tz — consistency!)

[WebRTC leak prevention] ──enhances──> [Per-session proxy]   (proxy is undone if real local IP leaks)
[Human-behavior tuning] ──orthogonal to──> [fingerprint]     (behavioral vs static signals; both scored independently)

[Best-effort WAF smoke check] ──conflicts──> [Blocking CI]   (non-deterministic; MUST stay out of the gate)
```

### Dependency Notes

- **Harness gates everything:** every other v1.6 feature is only "done" when the harness asserts it. Build/stabilize the harness first.
- **Configurable fingerprint requires a consistency validator, not just flags:** wiring `--ua`/`--platform` to `Profile` is trivial; the value is *rejecting incoherent combinations*. This is the dependency the downstream requirements doc must surface.
- **Profile struct + JSON already exist:** `stealth.Profile` has `UserAgent, Platform, AcceptLanguage, Languages, Timezone, Locale, Screen{W,H,Scale}, HardwareConcurrency, DeviceMemory, Vendor, Geolocation, SpoofClientHints, SpoofAudioContext` with `Save`/`LoadProfile`. The named-profile-file feature is ~90% plumbing.
- **Proxy and fingerprint are *different layers* that must still agree:** proxy IP geolocation should match spoofed timezone/locale, and WebRTC must not leak the real local IP — otherwise the proxy *creates* a consistency lie instead of hiding one.
- **WAF smoke check conflicts with CI gating:** keep it a separate, opt-in, documented-as-best-effort path.

---

## Consistency Invariants (MUST shape the configurable-fingerprint requirements)

These are the lie-detection rules; pinning any field obligates the dependent fields. The
consistency validator should enforce/auto-derive these and **fail loudly** on contradiction.

| If user pins... | These MUST stay coherent | Tell if violated |
|-----------------|--------------------------|------------------|
| `UserAgent` | Client Hints (`Sec-Ch-Ua`, `userAgentData`), `navigator.platform`, UA-CH version all derived from same browser/OS/version | CreepJS "UA vs platform" lie; instant flag |
| `Platform` (`Win32`/`MacIntel`) | WebGL vendor/renderer family, font list, Client Hints platform | "Windows claiming a macOS-only GPU" — the canonical CreepJS lie |
| `Locale` | `navigator.languages`, `Accept-Language` header, `Intl` formatting | Locale ↔ languages mismatch |
| `Timezone` | proxy IP geolocation, `Intl.DateTimeFormat().resolvedOptions().timeZone`, JS `Date` offset | Timezone-vs-IP mismatch (DataDome device check) |
| `Screen` | `availWidth/availHeight ≤ width/height`, `devicePixelRatio`, not 800×600 | Impossible screen geometry |
| `HardwareConcurrency` / `DeviceMemory` | plausible together + with device class (e.g. not 1 core / 64GB) | Implausible hardware combo |
| canvas/audio noise | **stable within a session** (re-reads return identical hash) | Non-deterministic native API = lie |

**Requirements implication:** the configurable-fingerprint feature is "pin a *coherent
profile*," not "pin arbitrary independent fields." Prefer pinning a browser+OS+locale tuple and
**deriving** the rest via godoll's `FingerprintGenerator` (which already produces internally
consistent fingerprints) over letting agents set raw contradictory values.

---

## MVP Definition

### Launch With (v1.6 core)

- [ ] **Offline deterministic detection harness** in CI — asserts every table-stakes signal. Without it nothing else is "proven."
- [ ] **Table-stakes signal validation** green (webdriver, plugins, UA-no-HeadlessChrome, WebGL non-SwiftShader, permissions, languages, screen, chrome.runtime, timezone).
- [ ] **Configurable fingerprint via flags + named profile file** mapped onto `stealth.Profile`, **with a consistency validator** that rejects/auto-derives incoherent combos.
- [ ] **Per-session proxy** `--proxy` for HTTP **and** SOCKS5; auth handled via CDP (not URL creds); proxy bound per named session.
- [ ] **Machine-readable verdict** (`--raw` PASS/FAIL + only failing signals) — token-efficient agent output.

### Add After Validation (v1.6.x)

- [ ] **WebRTC local-IP leak prevention** wired + asserted (godoll `EvadeWebRTC` exists; just expose+test).
- [ ] **Canvas/WebGL/Audio noise toggles** as profile bools with stable-hash assertions.
- [ ] **Human-behavior tuning knobs** (typing speed, delay jitter, mouse-path realism, scroll) — godoll humanize already powers defaults; expose config.
- [ ] **Best-effort, non-blocking WAF smoke check** (Cloudflare/DataDome) documented as manual-only.

### Future Consideration (v2+)

- [ ] TLS/JA3-JA4 alignment (uTLS-style) — **structurally outside a JS-injection CLI**; would need a network-layer rewrite. Defer/keep out of scope.
- [ ] Live CreepJS trust-grade tracking as an optional online benchmark.
- [ ] Profile marketplace / many vetted device profiles beyond a curated desktop set.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Offline deterministic harness | HIGH | MEDIUM | P1 |
| Table-stakes signal validation | HIGH | LOW (godoll exists) | P1 |
| Configurable fingerprint + consistency validator | HIGH | MEDIUM | P1 |
| Named stealth-profile file (flag-overridable) | HIGH | LOW–MEDIUM | P1 |
| Per-session proxy (HTTP+SOCKS5, CDP auth) | HIGH | MEDIUM | P1 |
| Machine-readable PASS/FAIL verdict | HIGH | LOW | P1 |
| WebRTC leak prevention | MEDIUM–HIGH | LOW | P2 |
| Canvas/WebGL/Audio noise toggles | MEDIUM | LOW | P2 |
| Human-behavior tuning knobs | MEDIUM | LOW–MEDIUM | P2 |
| Best-effort WAF smoke (non-blocking) | MEDIUM | MEDIUM | P3 |
| TLS/JA3-JA4 alignment | HIGH (but out of layer) | HIGH | P3 / out of scope |

## Competitor Feature Analysis

| Feature | puppeteer-extra-stealth / playwright-stealth | undetected-chromedriver | Our Approach (rod-cli) |
|---------|----------------------------------------------|--------------------------|------------------------|
| Headless tell evasion | Many evasion modules (webdriver, plugins, WebGL, chrome.runtime) | Patches webdriver/CDP tells in Chromedriver | godoll `EvasionManager`/`AntiDetection` — parity, single Go binary, no Node/Python |
| Fingerprint *consistency* | Per-evasion, no global coherence guarantee | Limited | **Explicit consistency validator + coherent-profile pinning** (differentiator) |
| Configurable pinned profiles | Manual per-property | Minimal | `stealth.Profile` JSON file + flags + per-session override |
| Validation harness shipped | No (users point at sannysoft manually) | No | **Bundled offline deterministic CI harness** (differentiator) |
| Agent/token-efficient output | N/A (libraries) | N/A | `--raw` PASS/FAIL one-liner |
| Proxy per session | Via browser context | Via launcher args | `--proxy` HTTP+SOCKS5, CDP auth, per named session |
| WAF bypass claims | Often over-claimed | Over-claimed | **Honestly scoped: JS layer only, WAF best-effort/manual** |

## Sources

- CreepJS — what it measures & trust/lie scoring: [undetectable.io](https://undetectable.io/blog/creepjs-browser-fingerprint-test/), [ZenRows](https://www.zenrows.com/blog/creepjs), [Scrapfly](https://scrapfly.io/blog/posts/browser-fingerprinting-with-creepjs)
- bot.sannysoft.com checks & headless tells: [bot.sannysoft.com](https://bot.sannysoft.com/), [Databay 28-signal matrix](https://databay.com/blog/how-sites-detect-headless-browsers), [Intoli making headless undetectable](https://intoli.com/blog/making-chrome-headless-undetectable/)
- Cloudflare/DataDome detection layers (TLS JA3/JA4 + IP + behavior): [TorchProxies comparison](https://torchproxies.com/cloudflare-vs-datadome-vs-human-security-what-each-bot-system-actually-checks-2026/), [Cloudflare bot-detection docs](https://developers.cloudflare.com/bots/concepts/bot-detection-engines/), [DataDome deep-dive](https://dev.to/bshahin/what-datadome-actually-checks-and-why-your-cloudflare-playbook-doesnt-transfer-24b5)
- Chrome proxy + SOCKS5/auth constraints: [Chromium proxy.md](https://chromium.googlesource.com/chromium/src/+/689912289c/net/docs/proxy.md), [Chromium SOCKS proxy design](https://www.chromium.org/developers/design-documents/network-stack/socks-proxy/)
- godoll source (read directly): `stealth/{profile,evasion,webgl,anti_detection,fingerprint_bridge}.go`, `fingerprint/*.go`, `humanize/`, `network/` under `/home/john/go/src/github.com/agenthands/godoll`

---
*Feature research for: browser-stealth validation + configurable fingerprinting (rod-cli v1.6)*
*Researched: 2026-06-24*
