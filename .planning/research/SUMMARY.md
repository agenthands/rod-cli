# Project Research Summary

**Project:** rod-cli — milestone v1.6 "Proven & Configurable Stealth"
**Domain:** Brownfield extension of a Go browser-automation CLI (persistent-daemon model) — proving & exposing godoll's existing stealth surface
**Researched:** 2026-06-24
**Confidence:** HIGH

## Executive Summary

rod-cli v1.6 is a **brownfield "expose + validate, not build" milestone.** All four research tracks converge on one finding: godoll (the vendored driver wrapper at `../godoll`) already implements nearly every target capability — proxy + auth relay, constrained/pinnable fingerprints, canvas/WebGL/audio spoofing, WebRTC evasion, and humanize tuning all exist as exported APIs. The work is not new evasion engines; it is (a) a **deterministic offline detection test harness** that proves what the shipped binary actually evades, (b) a **session-persistent stealth config surface** (CLI flags + a named-profile file) that rides the existing daemon-spawn persistence mechanism, (c) **wiring godoll bits that are exposed but never called** — `EvadeWebRTC()` (which `Apply()` omits), `WithWebRTCLeakProtection`, CDP timezone/locale overrides, and humanize functional options — plus replacing the no-op `ApplyCanvasNoise` stub, and (d) **fingerprint single-source-of-truth / consistency** so spoofed signals never contradict each other. No new third-party Go module is required for the core milestone; the harness is pure stdlib (`net/http` + `embed`).

The recommended approach is **harness-first**: build the offline, deterministic Tier-1 detection server (mirroring the existing `internal/plugin/scanner/testserver` pattern) and a test CI job (the repo has *none* today) before any evasion feature, so every later feature lands behind a real assertion that reads back from the **live page**, never from Go config state. Build order then proceeds config-surface → proxy (smallest, already half-wired) → configurable fingerprint (with the consistency invariant) → canvas/WebGL/WebRTC hardening (the one genuine godoll gap) → humanize tuning → best-effort live validation behind a build tag.

The dominant risk is **over-claiming and false-green validation.** Modern detectors (CreepJS, DataDome, Cloudflare) score fingerprint *consistency* and probe the CDP transport itself — layers a JS-injecting CLI cannot fully control ("two of three is not enough"). The project's single most expensive recurring lesson (the v1.5 `onDOMNodeInserted` wired-but-silent bug) means every stealth assertion must read from `page.Eval`, not from a Config struct; a hardcoded Client-Hints version `"121"` and random-per-page fingerprints are live inconsistency bugs to fix. Mitigation is baked into the build order: an offline harness that matrixes headless/headful, a single fingerprint source of truth feeding all surfaces, honest scoping of WAF/CDP as best-effort/non-blocking, and a quiet-by-default output discipline carried forward from v1.5.

## Key Findings

### Recommended Stack

See [STACK.md](STACK.md). **Zero new third-party modules** for the core milestone — everything maps onto existing godoll + go-rod APIs, and the harness is pure Go stdlib. v1.6 is *proving and exposing* godoll's surface, not replacing it.

**Core technologies:**
- `github.com/agenthands/godoll` (local `replace ../godoll`): definitive driver layer (stealth, fingerprint, proxy, humanize) — the integration surface, not something to rebuild.
- `github.com/go-rod/rod` v0.116.2: CDP transport + `proto.Emulation*` overrides (`SetTimezone/Locale/Geolocation/DeviceMetrics/UserAgentOverride`, all verified present) — engine-level pins ride on these.
- Go 1.25.1 stdlib (`net/http`, `embed`, `encoding/json`, `testing`): the offline detection harness, mirroring the existing `testserver` + `types/js` embed patterns. Keeps the single-static-binary, zero-Node/Python constraint.

### Expected Features

See [FEATURES.md](FEATURES.md). The cross-cutting fact: detectors score **fingerprint consistency, not individual values** — a pinned field that contradicts another pinned field is *worse* than not pinning. Nearly every feature is "expose + validate an existing godoll capability."

**Must have (table stakes / v1.6 core):**
- Offline deterministic detection harness in CI — the headline deliverable; without it nothing is "proven."
- Table-stakes signal validation green (webdriver, plugins, UA-no-HeadlessChrome, WebGL non-SwiftShader, permissions, languages, screen, `chrome.runtime`, timezone) — godoll already emits these; harness must assert them.
- Configurable fingerprint via flags + named profile file (mapped onto `stealth.Profile`) **with a consistency validator** that rejects/auto-derives incoherent combos.
- Per-session proxy (`--proxy`, HTTP + SOCKS5), auth via CDP not URL creds, bound per named session.
- Machine-readable verdict (`--raw` PASS/FAIL + only failing signals) — token-efficient agent output.

**Should have (competitive / v1.6.x after validation):**
- WebRTC local-IP leak prevention (`EvadeWebRTC` + `WithWebRTCLeakProtection` — both currently unwired).
- Canvas/WebGL/Audio noise toggles as profile bools with **stable-hash** assertions.
- Human-behavior tuning knobs (typing speed, typo rate, mouse path, scroll) — godoll options exist, called with none today.
- Best-effort, non-blocking WAF smoke check (documented manual-only).

**Defer (v2+):**
- TLS/JA3-JA4 alignment (uTLS-style) — structurally outside a JS-injection CLI.
- Live CreepJS trust-grade tracking as an optional online benchmark; profile marketplace.

### Architecture Approach

See [ARCHITECTURE.md](ARCHITECTURE.md). This is an **integration study, not a redesign.** The daemon freezes `Config` once at `NewContext` per session, so any stealth setting placed in `Config` and forwarded at daemon-spawn time is automatically session-persistent — v1.6 must *ride* this backbone, not invent a new one. There are exactly two evasion seams (`launchBrowser`, `createPage`) plus the header path (`updateInterceptorRules`); all wiring lands there.

**Major components:**
1. `StealthConfig` struct embedded in `Config` (NEW) — aggregates all knobs (UA/locale/timezone/platform/screen, WebRTC/canvas/audio bools, humanize tuning); YAML+JSON tagged.
2. Precedence resolver `types/stealth_profile.go` (NEW) — `flag > named profile file > rod-cli.yaml > godoll default`, resolved **once at daemon boot** so every later command inherits one identity.
3. Detection harness `internal/detect/testserver/` + `tests/detection_test.go` (NEW) — offline probe server (mirror `testserver`) driven through the real binary via the existing `eval`/`snapshot` commands; plus the repo's **first** test CI job (`.github/workflows/test.yml`).
4. Two-seam wiring (MODIFY `context.go`/`actions.go`) — `launchBrowser` adds `WithWebRTCLeakProtection`; `createPage` adds config-driven fingerprint + `EvadeWebRTC()`; `actions.go` threads `humanize.With*` options; `cmd.go`/`main.go` add + **forward** stealth flags across the daemon boundary (the persistence linchpin).

### Critical Pitfalls

See [PITFALLS.md](PITFALLS.md). Reality anchor: as of April 2026, most commercial stealth APIs still fail consistency tests — the milestone's value is *proving* and being *honest about the ceiling*, not winning an arms race.

1. **Flaky live third-party sites in blocking CI** — never gate merge on a network call to CreepJS/sannysoft/Cloudflare. Two-tier harness: offline vendored probes block CI; live suites behind `-tags=live`, non-blocking/nightly.
2. **Asserting on Go config instead of live browser behavior** (the `onDOMNodeInserted` lesson) — every stealth assertion must read back via `page.Eval`, in the **daemon-reused** browser context. Stop swallowing `_ = em.Apply()` and the fingerprint-generate error.
3. **Fingerprint inconsistency** — UA/UA-CH/`userAgentData`/platform/WebGL must tell one OS story. Kill the hardcoded Client-Hints `"121"`; build a single source of truth feeding JS *and* headers; make the consistency invariant a blocking test.
4. **Per-session config baked at launch in a SHARED daemon browser** — proxy is launch-time, fingerprint is per-page; a second command's `--proxy` is silently ignored or bleeds. Prefer per-BrowserContext proxy keyed to `-s`; fail loudly if a knob can't change live; add a session-isolation test. (Architectural — flag for discuss-phase.)
5. **CDP itself is detectable** regardless of JS spoofing — add protocol-layer probes to the harness; treat "hide CDP" as a spike with an explicit YES/NO, not a guaranteed deliverable; document the ceiling. Secondary: over-spoofing/entropy spikes (random-per-page) and proxy WebRTC/DNS leaks.

## Implications for Roadmap

Based on research, suggested phase structure (harness-first, dependency-respecting):

### Phase 1: Detection Harness (offline, deterministic, two-tier) + first test CI
**Rationale:** Every later phase needs something to assert against, and the repo has *no* test CI today. Building the harness against the *current* binary establishes a baseline that reveals which tells already pass (via `StealthPreset`/`Apply`) and which leak (e.g. WebRTC).
**Delivers:** `internal/detect/testserver/` + fixtures, `tests/detection_test.go` reading via `eval`/`snapshot`, headless+headful CI matrix, `.github/workflows/test.yml`. Live suites stubbed behind `//go:build detection_live`.
**Addresses:** the headline table-stakes harness + machine-readable verdict groundwork.
**Avoids:** P1 (flaky live CI), P2 (source-not-live asserts — sets the read-from-page convention), P4 (headful/headless gap), P10 (suite drift), P3 (adds CDP-tell probes).

### Phase 2: Config surface + Proxy
**Rationale:** The config surface is the substrate every feature flag rides; the daemon-boundary flag forwarding is the riskiest plumbing, so build it early with the harness watching. Proxy is the smallest first feature (`cfg.Proxy` is already consumed — only the flag + auth/SOCKS5 are missing) and validates the whole flag→daemon→godoll path end-to-end.
**Delivers:** `StealthConfig` + precedence resolver + profile file load/save + flag forwarding; `--proxy` (HTTP+SOCKS5) via godoll `ProxyConfig`/relay with CDP auth.
**Uses:** godoll `ProxyConfig.ApplyToLauncher`/`SetupBrowserAuth`/`StartProxyRelay`; `stealth.LoadProfile`/`Save`.
**Implements:** components 1, 2, and the proxy half of seam-wiring.
**Avoids:** P7 (proxy auth/DNS leak), P8 (daemon-shared config bleed — resolve per-BrowserContext), P11 (output verbosity — redact creds, quiet default).

### Phase 3: Configurable Fingerprint + Consistency Invariant
**Rationale:** Pinning is trivial; the value is *rejecting incoherent combinations*. Must establish the single source of truth before hardening so noise has a stable base.
**Delivers:** flag/profile-driven `stealth.Profile` (UA/locale/timezone/platform/screen), engine-level CDP overrides, the consistency validator, single-source-of-truth feeding JS + headers (killing hardcoded `"121"`).
**Addresses:** configurable/pinnable fingerprint, named profile file.
**Avoids:** P5 (inconsistency — blocking invariant test), P6 (over-spoof — per-session pinning), P9 (timezone/geo vs proxy), P12 (launcher-flag tells, `no-gpu`<->WebGL).

### Phase 4: Canvas/WebGL/WebRTC Hardening
**Rationale:** The only phase needing a genuine godoll-gap fix (`ApplyCanvasNoise` is a no-op stub; `EvadeWebRTC()` is never called by `Apply()`).
**Delivers:** real canvas noise (prefer contributing to godoll), `EvadeWebRTC()` + `WithWebRTCLeakProtection` wired, audio-noise toggle, stable-hash assertions.
**Avoids:** P6 (stable-per-session, plausible-range noise), P7 (WebRTC leak probe).

### Phase 5: Humanize Tuning
**Rationale:** Lowest detection-risk, mostly UX; thread existing `humanize.With*` options through `actions.go`. Verify pinned CI speeds keep the suite fast.
**Delivers:** typing/typo/mouse/scroll tuning fields + accessors threaded into `actions.go`.

### Phase 6: Best-effort Live Validation (non-blocking)
**Rationale:** Depends on all prior features; explicitly out of blocking CI.
**Delivers:** CreepJS/Cloudflare/DataDome smoke behind `//go:build detection_live`, documented best-effort, honest ceiling.

### Phase Ordering Rationale
- **Harness gates everything:** dependency analysis shows every other feature is "done" only when the harness asserts it from the live page — so it must come first and against the current binary.
- **Config substrate before features:** each feature should land behind a real, testable, session-persistent flag; the daemon-boundary forwarding is the persistence linchpin and the riskiest plumbing.
- **Consistency before hardening:** a single source of truth (Phase 3) must precede canvas/WebGL noise (Phase 4) so spoofing has a coherent, stable base — otherwise hardening *creates* lies.
- **Honest ceiling last:** CDP/WAF best-effort is non-blocking and inherently non-deterministic, so it stays at the edge.

### Research Flags

Phases likely needing deeper research / discuss-phase during planning:
- **Phase 2/3 (daemon-shared per-session config):** architectural — per-BrowserContext proxy vs relaunch vs per-page is a design decision (Pitfall 8). Needs discuss-phase.
- **CDP-footprint feasibility (within Phase 1 or its own small spike):** "can we hide the CDP transport" is the part most likely to be partially-or-not achievable — treat as a spike with an explicit YES/NO and a documented ceiling (Pitfall 3).
- **Proxy auth / SOCKS5-auth path (Phase 2):** verify godoll's CONNECT relay covers SOCKS5 auth; CDP `Fetch.continueWithAuth` mechanics.

Phases with standard patterns (skip research-phase):
- **Phase 1 harness server:** mirrors the existing `testserver` + `tests/cli_test.go` patterns — established in-repo.
- **Phase 5 humanize tuning:** pure functional-option threading, zero new dependency.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | godoll + go-rod v0.116.2 APIs read directly from source/module cache; CDP `Emulation*` overrides verified present. |
| Features | HIGH | godoll surface read directly; detection-suite + WAF behavior cross-checked across multiple 2026 sources. |
| Architecture | HIGH | Grounded in actual rod-cli source (context.go, config.go, cmd.go, main.go, daemon.go, testserver) and vendored godoll. |
| Pitfalls | HIGH | Grounded in current detection research + April 2026 benchmarks + the project's own retrospective lessons. |

**Overall confidence:** HIGH

### Gaps to Address
- **CDP-transport detectability ceiling:** whether rod-cli can meaningfully reduce its `Runtime.enable`/CDP footprint (console/request logging forces domain enablement) is unresolved — resolve via a spike with an explicit outcome; document the ceiling honestly rather than promise it.
- **Daemon per-session isolation mechanism:** whether proxy/fingerprint isolation uses per-incognito-BrowserContext vs relaunch vs fail-loud is a design decision — settle in discuss-phase before Phase 2/3, add a concurrent `-s` isolation test.
- **`ApplyCanvasNoise` ownership:** implement real noise in godoll (preferred, keeps it the source of truth) vs shadow via `stealth.InjectScript` in rod-cli — decide in Phase 4.
- **Profile file format:** godoll JSON `LoadProfile`/`Save` is zero-effort, but `Config` uses YAML (`rod.yaml`); decide JSON-via-godoll vs a thin YAML wrapper in requirements.
- **Doc consistency:** PROJECT.md says "Go 1.23+" while `go.mod` declares `go 1.25.1` — harmless, flag for a doc fix.

## Sources

### Primary (HIGH confidence)
- `../godoll` source read directly — `browser/{proxy,options,browser}.go`, `stealth/{evasion,profile,webgl,anti_detection,fingerprint_bridge,inject,js}.go`, `fingerprint/*.go`, `humanize/*.go` — definitive API names/signatures; confirmed `Apply()` does NOT call `EvadeWebRTC()` and `ApplyCanvasNoise` is a stub.
- `rod-cli` source read directly — `types/context.go`, `types/config.go`, `actions/actions.go`, `cmd.go`, `main.go`, `daemon/daemon.go`, `internal/plugin/scanner/testserver/`, `tests/cli_test.go`, `.github/workflows/release.yml` — current wiring + patterns to mirror; confirmed hardcoded CH `"121"`, no test CI job.
- go-rod module cache `@v0.116.2` `lib/proto/emulation.go` + `definitions.go` — confirmed CDP `Emulation.set{Timezone,Locale,Geolocation,DeviceMetrics,UserAgent}Override`.
- Project retrospective (v1.5 wired-but-silent `onDOMNodeInserted`, validate-live-not-source, quiet-output; v1.3 flaky page-load -> `retry.Fetch`).
- DataDome, Rebrowser, Castle.io, Wilico, ScrapeOps (April 2026 benchmark) — CDP signal, UA-vs-UA-CH inconsistency detection, realistic ceiling.

### Secondary (MEDIUM confidence)
- CreepJS (github.com/abrahamjuliot/creepjs) — MIT, self-hostable, name trademarked; optional nightly target only.
- undetectable.io / ZenRows / Scrapfly — CreepJS trust/lie scoring; svebaa, Sendwin — V8/WebGL fingerprinting.
- Chromium proxy.md / SOCKS design docs — proxy auth + SOCKS5 constraints.

### Tertiary (LOW confidence)
- bot.sannysoft.com + puppeteer-extra #402 — no cleanly-licensed self-host repo; replicate signals rather than vendor.

---
*Research completed: 2026-06-24*
*Ready for roadmap: yes*
