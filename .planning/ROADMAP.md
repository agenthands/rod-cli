# Roadmap: rod-cli

## Milestones

- ✅ **v1.0 Core CLI Foundation** — shipped
- ✅ **v1.1 Stealth & Humanization** — shipped ([archive](milestones/v1.1-ROADMAP.md))
- ✅ **v1.2 First-Class Agent Skills & Documentation** — shipped
- ✅ **v1.3 Godoll Migration** — shipped ([archive](milestones/v1.3-ROADMAP.md))
- ✅ **v1.4 Plugin Architecture** — shipped ([archive](milestones/v1.4-ROADMAP.md))
- ✅ **v1.5 Plugin Ecosystem Documentation** — Phases 21–23 (shipped 2026-06-23) ([archive](milestones/v1.5-ROADMAP.md))
- ✅ **v1.6 Proven & Configurable Stealth** — Phases 24–29 (shipped 2026-06-25) ([archive](milestones/v1.6-ROADMAP.md))
- ✅ **v1.7 Complete Evasion Stack** — Phases 30, 32, 33 (shipped 2026-06-26; Phase 31/TLS cancelled — real-Chrome-only) ([archive](milestones/v1.7-ROADMAP.md))
- ✅ **v1.8 Debt Cleanup & Coding-Assistant Onboarding** — Phases 34–37 (shipped 2026-06-26)
- 🔨 **v1.9 godoll Hygiene & CDP-DEEP-01 Research** — Phases 38–39 (in progress)

Full per-phase detail for each shipped milestone lives under `.planning/milestones/`.

## Phases

**Phase Numbering:**

- Integer phases (24, 25, …): Planned milestone work
- Decimal phases (24.1, 24.2): Urgent insertions (marked with INSERTED)

<details>
<summary>✅ v1.5 Plugin Ecosystem Documentation (Phases 21–23) — SHIPPED 2026-06-23</summary>

- [x] Phase 21: Reference Documentation (4/4 plans) — completed 2026-06-22
- [x] Phase 22: Example Plugins (5/5 plans) — completed 2026-06-23
- [x] Phase 23: Authoring Guide & Docs Index (2/2 plans) — completed 2026-06-23

Delivered a complete `docs/plugins/` tree — lifecycle-hooks, state-api, and cli-reference pages; a flagship XSS-scanner worked example, per-hook recipes, and a copyable starter; an authoring tutorial and a README-linked docs index — backed by runnable example plugins exercising every hook and the state API. Surfaced and fixed three small engine gaps along the way (`api.GetLocalStorage()`, functional `plugin run`, and CDP DOM-domain enable for `onDOMNodeInserted`) and removed the always-on startup banner for token-efficiency. Full detail: [milestones/v1.5-ROADMAP.md](milestones/v1.5-ROADMAP.md).

</details>

Earlier milestones (v1.0–v1.4) are archived under `.planning/milestones/`.

### ✅ v1.6 Proven & Configurable Stealth (Shipped 2026-06-25)

**Milestone Goal:** Turn rod-cli's already-wired godoll stealth from "compiles and runs" into "provably evades detection and is configurable per session" — validated against a deterministic offline detection harness (reading from the live page, never from source), exposed through an agent-friendly session-persistent config surface, and extended with the evasion wiring that matters (WebRTC leak prevention, stable canvas noise, fingerprint consistency). Brownfield: this milestone **proves, configures, and wires** existing godoll capability — it does not rebuild it.

- [x] **Phase 24: Detection Harness & CI Backbone** — Offline, deterministic detection test server + first test CI job, baselined against the current binary so existing leaks surface (completed 2026-06-24)
- [x] **Phase 25: Stealth Config Surface & Per-Session Proxy** — Session-persistent stealth config (flags + named profile file, precedence-resolved at daemon spawn) plus per-session HTTP/SOCKS5 proxy with CDP auth (completed 2026-06-24)
- [x] **Phase 26: Configurable Fingerprint & Consistency Validator** — Pinnable coherent fingerprint from a single source of truth, consistency invariant gate, the CH-121 fix, and a user-facing per-signal stealth-check verdict (completed 2026-06-24)
- [x] **Phase 27: Canvas/WebGL/WebRTC Hardening** — Wire the two genuine godoll gaps: WebRTC IP-leak prevention and stable-per-session canvas/WebGL/audio noise, both harness-asserted (completed 2026-06-24)
- [x] **Phase 28: Human-Behavior Tuning** — Thread godoll humanize options (typing speed, typo rate, delay jitter, mouse path, scroll) through actions as flags/profile fields (completed 2026-06-25)
- [x] **Phase 29: Best-Effort Live Validation** — Opt-in, non-blocking, build-tagged live smoke check (Cloudflare/DataDome/CreepJS) kept out of the CI gate (completed 2026-06-25)

## Phase Details

### Phase 24: Detection Harness & CI Backbone

**Goal**: A deterministic, offline detection harness drives the real `rod-cli` binary and asserts each table-stakes stealth signal by reading it back from the live page, running on every push as the repo's first test CI job — baselined against the current binary so existing leaks (e.g. unwired WebRTC) are exposed up front rather than hidden.
**Depends on**: Phase 23 (prior milestone)
**Requirements**: HARNESS-01, HARNESS-02, HARNESS-03, VALIDATE-03
**Success Criteria** (what must be TRUE):

  1. Running the harness test suite boots a local `127.0.0.1` fixture server (no network egress) and drives the actual `rod-cli` binary against a `go:embed`-bundled sannysoft-style detection page, producing a deterministic green-or-red result.
  2. Each table-stakes assertion (`navigator.webdriver`, plugins, UA-without-`HeadlessChrome`, WebGL vendor, permissions, languages, screen, `window.chrome`, timezone) is verified by reading the value back via `page.Eval` on the live page — never from a Go config field — and runs in both headless and headful matrix rows.
  3. A `.github/workflows/test.yml` job runs the harness on every push and its first run records a baseline where pre-existing leaks (e.g. WebRTC, any hardcoded-CH mismatch) show as known red signals rather than silent passes.
  4. The evasion path fails loudly: `EvasionManager.Apply()` and fingerprint-generation errors are surfaced/logged instead of being discarded (`_ = em.Apply()`), so a silent no-op is observable in the harness output.

**Plans**: 4/4 plans complete

- [x] 24-01-PLAN.md — Offline 127.0.0.1:0 detection fixture server + self-authored go:embed page (HARNESS-01)
- [x] 24-02-PLAN.md — VALIDATE-03: surface swallowed fingerprint/Apply() errors to stderr (no hard-fail)
- [x] 24-03-PLAN.md — E2e harness driving the live binary, reading each signal from the live page via eval, KNOWN-RED baseline markers (HARNESS-02)
- [x] 24-04-PLAN.md — First test CI workflow (go 1.25.x, push + PR to main) + stray-artifact cleanup + CDP-footprint findings note (HARNESS-03)

### Phase 25: Stealth Config Surface & Per-Session Proxy

**Goal**: A session-persistent stealth configuration surface (CLI flags plus a named JSON profile file) resolves with deterministic precedence once at daemon-spawn time and is inherited by every later command on the same session — and the first feature riding it, a per-session HTTP/SOCKS5 proxy with CDP-based auth, validates the whole flag → config → godoll path end-to-end without bleeding across named sessions.
**Depends on**: Phase 24
**Requirements**: PROFILE-01, PROFILE-02, PROXY-01, PROXY-02
**Success Criteria** (what must be TRUE):

  1. A user can save a named stealth profile to a JSON file and load it; settings applied once are inherited by every subsequent command hitting the same daemon session (verified by reading the applied value back from the live browser on a second command).
  2. Stealth configuration resolves with the precedence CLI flag > profile file > built-in default, fixed at daemon spawn, with no per-command stealth state bleeding between two concurrently-running named (`-s`) sessions (asserted by a session-isolation test).
  3. A user can route a named session through an HTTP **or** SOCKS5 proxy via `--proxy`, bound to that session, and a second session with a different `--proxy` reports its own egress identity (no shared-daemon bleed).
  4. Proxy authentication succeeds against an authenticated proxy via CDP (`Fetch.continueWithAuth`) using `--proxy-auth`, with no URL-embedded credentials and no 407/dialog hang; credentials never appear in default output.

**Plans**: 3/3 plans complete

- [x] 25-01-PLAN.md — StealthConfig substrate: sub-struct, --proxy/--proxy-auth/--profile flags, daemon-boundary forwarding, precedence resolver, already-running warning (PROFILE-01/02)
- [x] 25-02-PLAN.md — Per-session proxy wiring: parse into godoll ProxyConfig, ApplyToLauncher + CDP SetupBrowserAuth replacing bare launcher.Proxy, relay cleanup on Close (PROXY-01/02)
- [x] 25-03-PLAN.md — Session-isolation/auth/profile-roundtrip/credential-leak e2e tests + offline proxy fixture (PROFILE-01/02, PROXY-01/02)

### Phase 26: Configurable Fingerprint & Consistency Validator

**Goal**: A user can pin a coherent fingerprint per session from a single source of truth that feeds JS properties, HTTP headers, and Client-Hints alike (killing the hardcoded CH `121`), with a consistency validator that rejects or auto-derives incoherent combinations and fails loudly on contradiction — and can read a per-signal stealth-check verdict, including a token-efficient `--raw` machine-readable form, against any page.
**Depends on**: Phase 25
**Requirements**: FINGERPRINT-01, FINGERPRINT-02, FINGERPRINT-03, VALIDATE-01, VALIDATE-02
**Success Criteria** (what must be TRUE):

  1. A user can pin a browser/OS/locale tuple (UA, platform, locale, timezone, screen, hardwareConcurrency, deviceMemory, vendor) via flags/profile mapped onto godoll `stealth.Profile`, and the harness confirms `Sec-CH-UA`, `navigator.userAgentData`, UA, `navigator.platform`, and WebGL all tell one consistent OS+version story (the consistency invariant is a blocking test).
  2. Client-Hints are derived from the active UA/OS — the `Sec-CH-UA` version equals the UA's Chrome major version — with no hardcoded `121` literal remaining anywhere in the header path.
  3. The consistency validator rejects or auto-derives incoherent combinations (UA↔Client-Hints↔platform, locale↔languages↔Accept-Language, timezone↔proxy-geo, plausible screen/hardware) and fails loudly with a clear message on contradiction rather than silently shipping a mismatched lie.
  4. A user can run a stealth-check command against a page and get a per-signal verdict (`navigator.webdriver`, plugins, UA, WebGL vendor, permissions, languages, screen, `window.chrome`, timezone), read from the live page.
  5. With `--raw`, the stealth-check emits a single-line `PASS`/`FAIL` plus only the failing signals (e.g. `webdriver=ok webgl=FAIL(SwiftShader)`) — no full-page dump.

**Plans**: 5/5 plans complete
**Wave 1**

- [x] 26-01-PLAN.md — Config + validation layer: StealthConfig identity fields, 4 override flags, stealth-check registration, consistency validator + UA-anchor derivation in ResolveStealth (FINGERPRINT-01/02, VALIDATE-01/02)
- [x] 26-02-PLAN.md — godoll: kill the hardcoded CH `121` in both runtime injectors (UA-derived Sec-Ch-Ua + userAgentData) and derive FromFingerprint timezone (FINGERPRINT-03)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 26-03-PLAN.md — rod-cli runtime wiring: pin the resolved profile as the active stealth.Profile in createPage + UA-derived interceptor Sec-Ch-Ua (FINGERPRINT-01/03)
- [x] 26-04-PLAN.md — stealth-check behavior: shared extracted probe, StealthCheck action (live-page reads, raw/json/human), daemon dispatch (VALIDATE-01/02)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 26-05-PLAN.md — Harness gate: flip the CH KNOWN-RED to required-green, add blocking consistency-invariant + pinned-identity + stealth-check subtests (FINGERPRINT-01/02/03, VALIDATE-01/02)

### Phase 27: Canvas/WebGL/WebRTC Hardening

**Goal**: The two genuine godoll-gap fixes land — WebRTC local-IP leak prevention is wired so a real host IP cannot leak past a proxy, and the no-op `ApplyCanvasNoise` stub is replaced with canvas/WebGL/audio noise that is stable within a session — with the harness asserting both the absence of a leaked host IP and identical noise hashes across re-reads.
**Depends on**: Phase 26
**Requirements**: HARDEN-01, HARDEN-02
**Success Criteria** (what must be TRUE):

  1. With a proxy active, a WebRTC-leak probe in the harness reads back no real local/host IP from the live page — `EvadeWebRTC` + `WithWebRTCLeakProtection` are actually called by the evasion path (previously omitted by `Apply()`).
  2. Canvas, WebGL, and audio noise are exposed as profile toggles, and the harness confirms that two reads of the canvas/WebGL hash within the same session return an identical value (stable-per-session, not per-read-varying).
  3. Spoofed noise values fall in plausible ranges (no impossible GPU/renderer or per-pixel-uniform pattern), so enabling hardening does not raise entropy/uniqueness versus the unhardened baseline.

**Plans**: TBD

### Phase 28: Human-Behavior Tuning

**Goal**: A user can tune human-like interaction — typing speed, typo rate, delay jitter, mouse-path realism, scroll behavior — via flags or the stealth profile, threading godoll `humanize` options that `actions.go` currently invokes with defaults only, without slowing the CI harness.
**Depends on**: Phase 27
**Requirements**: HUMANIZE-01
**Success Criteria** (what must be TRUE):

  1. A user can set typing speed, typo rate, delay jitter, mouse-path, and scroll tuning via flags/profile, and the values are threaded into the corresponding godoll `humanize.With*` options in `actions.go` (not silently ignored).
  2. A configured tuning value observably changes interaction behavior (e.g. a slower typing speed produces measurably longer per-keystroke timing) when driven through the real binary.
  3. CI-pinned humanize speeds keep the detection harness fast — the tuning surface does not regress test wall-clock time.

**Plans**: TBD

### Phase 29: Best-Effort Live Validation

**Goal**: An opt-in, non-blocking live smoke check against real anti-bot challenges (Cloudflare / DataDome / CreepJS) exists behind a `//go:build detection_live` tag, documented honestly as best-effort and manual, explicitly kept out of the blocking CI gate so flaky third-party network calls never gate a merge.
**Depends on**: Phase 28
**Requirements**: LIVEWAF-01
**Success Criteria** (what must be TRUE):

  1. A live smoke check exists behind `//go:build detection_live` and is excluded from the default test run and the push CI job (verified: a normal `go test ./...` and the `test.yml` job do not invoke it).
  2. A developer can manually run the live suite with the build tag to get an informational pass/fail against Cloudflare/DataDome/CreepJS, with failures reported as non-blocking.
  3. Documentation states the honest ceiling — what Tier-1 proves vs what the live suite is best-effort about (TLS/IP/CDP layers a JS-injecting CLI cannot control) — with no "undetectable" guarantee.

**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 24 → 25 → 26 → 27 → 28 → 29

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 21. Reference Documentation | v1.5 | 4/4 | Complete | 2026-06-22 |
| 22. Example Plugins | v1.5 | 5/5 | Complete | 2026-06-23 |
| 23. Authoring Guide & Docs Index | v1.5 | 2/2 | Complete | 2026-06-23 |
| 24. Detection Harness & CI Backbone | v1.6 | 4/4 | Complete   | 2026-06-24 |
| 25. Stealth Config Surface & Per-Session Proxy | v1.6 | 3/3 | Complete   | 2026-06-24 |
| 26. Configurable Fingerprint & Consistency Validator | v1.6 | 5/5 | Complete    | 2026-06-24 |
| 27. Canvas/WebGL/WebRTC Hardening | v1.6 | 4/4 | Complete    | 2026-06-24 |
| 28. Human-Behavior Tuning | v1.6 | 0/TBD | Complete    | 2026-06-25 |
| 29. Best-Effort Live Validation | v1.6 | 2/2 | Complete    | 2026-06-25 |

### ✅ v1.7 Complete Evasion Stack (Shipped 2026-06-26)

**Milestone Goal:** Extend rod-cli's stealth from JS-layer fingerprinting toward a fuller evasion solution — reducing CDP signals, providing curated (Chrome-only) device profiles, and expanding fingerprint hardening surfaces. **TLS fingerprint spoofing is explicitly out of scope: rod-cli drives real Chrome, so its TLS/JA3 handshake is already authentic — we never spoof it (operator constraint, 2026-06-26).**

- [x] **Phase 30: CDP Footprint Reduction** — Reduce or obfuscate Runtime.enable and other CDP signals; measure impact against detection targets; document the honest ceiling for what remains detectable. ✅ completed 2026-06-26 (qa PASSED; plain `goto` enables none of Runtime/Network/Fetch).
- [x] ~~**Phase 31: Network-Layer Identity (TLS)**~~ — ❌ **CANCELLED (2026-06-26)** — was uTLS-style TLS spoofing; vetoed by operator constraint "always stick to real Chrome, never spoof TLS." Real-Chrome TLS is authentic by construction. TLS-01..04 moved to Out of Scope.
- [x] **Phase 32: Profile Library** — Ship 5-10 vetted **Chrome-only** device profiles with the binary; test against harness; users list and select by name. (No spoofed TLS in any profile.) ✅ completed 2026-06-26 (qa PASSED; 6 profiles, `--profile=list`, vetting gate).
- [x] **Phase 33: Advanced Evasion** — Expand fingerprint hardening surfaces beyond v1.6; implement 2+ new hardening toggles; assert via harness. ✅ completed 2026-06-26 (qa PASSED; activated godoll fonts/devices/battery/codecs dimensions coherently, 4 toggles, harness-asserted).

### Phase 30: CDP Footprint Reduction

**Goal**: Reduce or obfuscate Runtime.enable and other CDP signals; measure impact against detection targets; document the honest ceiling for what remains detectable.
**Depends on**: Phase 29 (v1.6)
**Requirements**: CDP-01, CDP-02, CDP-03
**Success Criteria** (what must be TRUE):

  1. A detection-harness test asserts reduced CDP visibility (e.g., Runtime.enable signal count or detection score) compared to the v1.6 baseline.
  2. All CDP commands sent by rod-cli are inventoried in `docs/cdp-footprint.md` with mitigation status for each (reduced, obfuscated, accepted-visible).
  3. The honest ceiling (what remains detectable despite reduction) is documented and measured against live detection targets (Cloudflare/DataDome/CreepJS).

**Plans**: TBD

### Phase 31: Network-Layer Identity (TLS) — ❌ CANCELLED (2026-06-26)

**Status**: CANCELLED. Operator constraint: "Don't do TLS fingerprint spoofing — we always stick to real Chrome, with all the profiles." rod-cli drives a real Chrome/Chromium browser, so its TLS/JA3-JA4 handshake is already an authentic Chrome fingerprint; a uTLS-style spoofing layer adds a major network-layer architectural surface for no gain. Requirements TLS-01..TLS-04 are moved to Out of Scope in REQUIREMENTS.md. A short note documenting the deliberate real-Chrome-TLS stance is added to the stealth docs (handled as part of the Phase 31 closeout).

~~**Goal**: TLS/JA3-JA4 fingerprint alignment via uTLS-style spoofing; network-layer rewrite to match JS-layer identity; proxy-aware TLS.~~

### Phase 32: Profile Library

**Goal**: Ship 5-10 vetted **Chrome-only** device profiles with the binary; test against harness; users list and select by name. No profile carries a spoofed TLS fingerprint — the TLS layer is always genuine Chrome.
**Depends on**: Phase 30 (Phase 31 cancelled)
**Requirements**: PROF-01, PROF-02, PROF-03, PROF-04
**Success Criteria** (what must be TRUE):

  1. A `profiles/` directory (or embedded `//go:embed`) contains 5-10 coherent device profiles, each validated against the detection harness with documented fingerprint characteristics.
  2. `--profile=list` prints available built-in profiles; `--profile=<name>` selects by name without requiring a file path.
  3. Custom profiles (user-provided JSON files) continue to work; CLI flags override both built-in and custom profiles (v1.6 precedence chain honored).
  4. Each profile passes the v1.6 consistency validator (UA ↔ Client-Hints ↔ platform; timezone ↔ locale; plausible screen/hardware).

**Plans**: TBD

### Phase 33: Advanced Evasion

**Goal**: Expand fingerprint hardening surfaces beyond v1.6 canvas/WebGL/WebRTC; implement 2+ new hardening toggles; assert via harness.
**Depends on**: Phase 32
**Requirements**: EVAD-01, EVAD-02, EVAD-03
**Success Criteria** (what must be TRUE):

  1. Detection harness identifies at least two fingerprint vectors not covered by v1.6 (e.g., audio context, font enumeration, screen orientation, media codecs).
  2. New hardening toggles (e.g., `--audio-noise`, `--font-spoof`) follow the v1.6 precedence chain (CLI > profile > default) and are asserted by harness tests.
  3. All new toggles are documented in `docs/stealth-config.md` with on/off guidance and performance notes.
  4. New hardening surfaces are stable within a session (consistent hash on re-read, matching v1.6 canvas behavior).

**Plans**: TBD

### 🔨 v2.0 CDP-DEEP-01 Build — MITM WebSocket Proxy (In Progress)

**Milestone Goal:** Build the MITM WebSocket CDP proxy designed in Phase 39 — a local, in-process WebSocket proxy that sits between go-rod's `cdp.Client` and Chrome's debugging WebSocket, implementing `cdp.WebSocketable`. The proxy is transparent by default (pass-through) and becomes the foundation for CDP traffic filtering, Runtime domain normalization, and timing jitter.

- [ ] **Phase 40: Core Proxy (pass-through, logging)** — Fireable CDP WebSocket pass-through proxy with in-memory ring-buffer traffic logging, flag-gated behind `--cdp-proxy` (default OFF), wired into the browser launch path via godoll's `WithCDPWrapper`.
- [ ] **Phase 41: Runtime Domain Normalization** — Intercept and normalize Runtime domain CDP responses to suppress property-getter triggering when Runtime IS enabled (opt-in console/request capture path).
- [ ] **Phase 42: Timing Jitter + `cdp-traffic` Command** — Configurable CDP command dispatch timing jitter + a `rod-cli cdp-traffic` diagnostic command that reads the proxy's ring buffer.

### Phase 40: Core Proxy (pass-through, logging)

**Goal**: Implement the core CDP WebSocket proxy — a pass-through proxy that implements `cdp.WebSocketable`, logs all messages to an in-memory ring buffer, and is wired into the browser launch path gated behind `--cdp-proxy` (default OFF).
**Depends on**: Phase 39 (CDP-DEEP-01 research & design)
**Requirements**: CDP-PROXY-01, CDP-PROXY-02, CDP-PROXY-03, CDP-PROXY-04, CDP-PROXY-05
**Success Criteria** (what must be TRUE):

  1. `go build ./...` succeeds with the proxy wired but default-off.
  2. A browser session with `--cdp-proxy` navigates a page successfully (no breakage).
  3. With `--cdp-proxy`, the proxy's `Traffic()` contains expected CDP messages (Page.frameNavigated, etc.) after navigation.
  4. Without `--cdp-proxy`, behavior is identical to pre-phase-40 (no regression).
  5. The ring buffer enforces its capacity (doesn't leak memory).

**Plans**: 1/1 plan complete

- [x] 40-PLAN.md — Proxy core + config surface + CLI flag + browser launch wiring + context storage

### Phase 41: Runtime Domain Normalization

**Goal**: When Runtime is enabled (opt-in via `--console-capture` or plugins), normalize Runtime domain CDP responses to suppress property-getter triggering — intercept `Runtime.getProperties` responses and strip or replace accessor property `value` descriptors that trigger getters observable from the page.
**Depends on**: Phase 40
**Requirements**: CDP-NORM-01, CDP-NORM-02
**Success Criteria** (what must be TRUE):

  1. With `--cdp-proxy` and `--console-capture`, the `cdpTell` probe returns `"no-signal"` (no stack-getter triggering).
  2. Normalization does not break `console-capture` functionality (console messages are still captured).
  3. With `--cdp-proxy` but WITHOUT `--console-capture`, behavior is unchanged (Runtime not enabled, no normalization needed).

**Plans**: TBD

### Phase 42: Timing Jitter + `cdp-traffic` Command

**Goal**: Add configurable timing jitter to CDP command dispatch to break characteristic automation timing patterns, and add a `rod-cli cdp-traffic` diagnostic command that reads the proxy's ring buffer for CDP traffic analysis.
**Depends on**: Phase 41
**Requirements**: CDP-JITTER-01, CDP-TRAFFIC-01
**Success Criteria** (what must be TRUE):

  1. CDP command dispatch includes a random 0-N ms delay (`--cdp-jitter-ms`, default 0 = off).
  2. `rod-cli cdp-traffic` prints the proxy's logged CDP messages in a human-readable format.
  3. `rod-cli cdp-traffic --json` emits machine-readable JSON.
  4. A `--no-cdp-proxy` bypass flag exists for zero-risk deployment.

**Plans**: TBD

</invoke>

### ✅ v1.8 Debt Cleanup & Coding-Assistant Onboarding (Shipped 2026-06-26)

**Milestone Goal:** Retire the three v1.7 follow-ups (toolchain bump, plugin-path CDP-ledger hole, font-spoof no-op) and ship authoritative install + agent-skill documentation so any of the five major coding assistants — Claude Code, Codex CLI, Gemini CLI, Pi (pi.dev), and opencode — can adopt rod-cli.

- [x] **Phase 34: Toolchain Bump & Vuln Gate** — Pin go.mod to go1.26.1, align CI/release, clear the 13 F1 stdlib vulns, and wire a govulncheck gate so the supply-chain fix can't silently regress. (BUILD-01, BUILD-02)
- [x] **Phase 35: Plugin-Path CDP-Ledger Closure** — Record CDP domains enabled via the plugin lifecycle path in the per-session ledger and assert the previously-uncovered path in the offline harness. (LEDGER-01, LEDGER-02)
- [x] **Phase 36: Real Font Spoofing** — Replace the godoll font-injector no-op with a real injector so `--font-spoof` actually changes detectable fonts (OS-coherent, stable per session), harness-asserted on/off/stability. (FONT-01, FONT-02, FONT-03)
- [x] **Phase 37: Coding-Assistant Onboarding Docs** — Binary install + the cross-tool SKILL.md update + per-assistant install/skill instructions for Claude Code, Codex CLI, Gemini CLI, Pi, and opencode, each with a copy-paste sequence and verify step, accurate to the real (MCP-free) command surface and linked from the README. (DOC-01..DOC-09)

### Phase 34: Toolchain Bump & Vuln Gate

**Goal**: Resolve v1.7 security follow-up F1 — `go.mod` declares `go 1.25.1` with no `toolchain` directive while dev builds run go1.26.0, and govulncheck flags 13 stdlib vulns fixed in go1.26.1. Pin the toolchain, align CI/release, clear the vulns, and gate against regression.
**Depends on**: Phase 33 (v1.7) — foundation phase: every later v1.8 phase builds/tests on the fixed toolchain.
**Requirements**: BUILD-01, BUILD-02
**Success Criteria** (what must be TRUE):

  1. `go.mod` declares `toolchain go1.26.1` (with the `go` directive aligned); `go build ./...` and `go test ./...` pass on go1.26.1.
  2. `govulncheck ./...` reports no known-vulnerable called paths for the 13 F1 stdlib vulns; any residual finding is documented with justification.
  3. The CI workflow runs on go1.26.1 and includes a `govulncheck` gate so the supply-chain fix cannot silently regress.

**Plans**: TBD

### Phase 35: Plugin-Path CDP-Ledger Closure

**Goal**: Close the v1.7 CDP-ledger coverage hole — CDP domains enabled lazily via the plugin lifecycle path were not observed by the per-session ledger that v1.7's footprint guarantees depend on.
**Depends on**: Phase 34 (clean toolchain) — independent of Phase 36.
**Requirements**: LEDGER-01, LEDGER-02
**Success Criteria** (what must be TRUE):

  1. CDP domains enabled via the plugin lifecycle path are recorded in the per-session CDP-domain ledger, identically to the main navigation path — no enabled domain escapes the inventory.
  2. The offline detection harness has a test that exercises a plugin enabling a lazy CDP domain and asserts the ledger reflects it (red before the fix, green after).
  3. The per-command/per-path CDP inventory in the docs is updated to reflect the now-tracked plugin path.

**Plans**: TBD

### Phase 36: Real Font Spoofing

**Goal**: Replace the documented godoll font-injector no-op (carried as a v1.7 follow-up) with a real injector so the `--font-spoof` toggle has an observable effect on the page's detectable font availability.
**Depends on**: Phase 34 (clean toolchain) — independent of Phase 35.
**Requirements**: FONT-01, FONT-02, FONT-03
**Success Criteria** (what must be TRUE):

  1. With `--font-spoof` enabled, a font-probe on the live page reads a spoofed font set that differs from the host baseline (the no-op is gone).
  2. The spoofed set is coherent with the active profile's OS/locale and identical across re-reads on the same session; `--font-spoof=false` restores genuine host font behavior.
  3. The offline detection harness asserts criteria 1 and 2 (differs-when-on, stable-across-re-reads, restored-when-off).

**Plans**: TBD

### Phase 37: Coding-Assistant Onboarding Docs

**Goal**: Ship authoritative install + agent-skill documentation for the five major coding assistants, onboarding rod-cli via the agent-skill / instructions-file → shell-out path (no MCP, since rod-cli is not an MCP server). Accurate to the real command surface, verified against current tool docs, and discoverable from the README.
**Depends on**: Phase 36 — docs reflect the final command surface and the now-real `--font-spoof` behavior.
**Requirements**: DOC-01, DOC-02, DOC-03, DOC-04, DOC-05, DOC-06, DOC-07, DOC-08, DOC-09
**Success Criteria** (what must be TRUE):

  1. A reader can follow the shared binary-install section (`go install` / prebuilt + `rod-cli install` for Chromium) and verify rod-cli runs (DOC-01).
  2. The shipped `skills/rod-cli/SKILL.md` is updated to the current cross-tool Agent-Skills standard (valid frontmatter + explicit "when to use" trigger phrases) so one skill directory serves Claude Code, Codex, Pi, and opencode (DOC-02).
  3. Each of the five assistants — Claude Code, Codex CLI, Gemini CLI (GEMINI.md context file), Pi (`~/.pi/agent/skills/`, explicit "no MCP"), opencode (native skills + `.claude/skills` compat) — has a section with a copy-paste install sequence and a concrete verify step that confirms the agent can drive rod-cli (DOC-03..DOC-08).
  4. Every documented claim is accurate against the real rod-cli command surface and current tool docs — no MCP install path is documented — and the docs are linked from the top-level README (DOC-09).

**Plans**: TBD

### ✅ v1.9 godoll Hygiene & CDP-DEEP-01 Research (Shipped 2026-06-26)

**Milestone Goal:** Close the two remaining v1.7 security-review hygiene items (F2/F4) in godoll and rod-cli, and produce an evaluated, grounded design for CDP-DEEP-01 (deep CDP signal obfuscation) that a follow-on milestone can execute.

- [x] **Phase 38: Godoll Hygiene** — Close v1.7 F2 (backslash reject in rod-cli) and F4 (json.Marshal in godoll's EnableRequestInterception). (HYGIENE-01, HYGIENE-02)
- [x] **Phase 39: CDP-DEEP-01 Research & Design** — Evaluate the three CDP-DEEP-01 approaches (browser-patching, MITM/alternate transport, patched DevTools endpoint) against Chrome's detection surface. Pick one. Produce a grounded, executable PLAN. Update `docs/cdp-footprint.md`. (CDP-DEEP-01, CDP-DEEP-02, CDP-DEEP-03)

### Phase 38: Godoll Hygiene

**Goal**: Close the two remaining v1.7 security-review hygiene items — F2 (add backslash to rod-cli's reject list as defense-in-depth) and F4 (fix godoll's EnableRequestInterception to json.Marshal platform).
**Depends on**: Phase 37 (v1.8 shipped).
**Requirements**: HYGIENE-01, HYGIENE-02
**Success Criteria** (what must be TRUE):

  1. rod-cli's `rejectUnsafeFingerprintValue` rejects the backslash character in addition to the already-rejected quote and control chars (HYGIENE-01).
  2. godoll's `EnableRequestInterception` uses `json.Marshal` when interpolating `platform` into the JS literal (HYGIENE-02).
  3. Both repos build and pass existing tests (no regression).

**Plans**: TBD

### Phase 39: CDP-DEEP-01 Research & Design

**Goal**: Evaluate the three approaches named in the v1.7 honest ceiling — browser-patching, MITM/alternate transport, and patched DevTools endpoint — against Chrome's current detection surface. Produce a grounded recommendation with a concrete, executable PLAN.
**Depends on**: Phase 38.
**Requirements**: CDP-DEEP-01, CDP-DEEP-02, CDP-DEEP-03
**Success Criteria** (what must be TRUE):

  1. Each approach is evaluated against Chrome's CDP detection surface with real citations (not speculation). (CDP-DEEP-01)
  2. A concrete, executable PLAN exists for the chosen approach, grounded against the real go-rod/rod-cli transport layer. (CDP-DEEP-02)
  3. `docs/cdp-footprint.md` is updated with findings and the updated honest ceiling. (CDP-DEEP-03)

**Plans**: TBD

### ✅ v2.0 CDP-DEEP-01 Build — MITM WebSocket Proxy (Shipped 2026-06-26)

**Milestone Goal:** Build the MITM WebSocket CDP proxy designed in Phase 39 — a local, in-process WebSocket proxy that sits between go-rod's `cdp.Client` and Chrome's debugging WebSocket, implementing `cdp.WebSocketable`. The proxy is transparent by default (pass-through) and becomes the foundation for CDP traffic filtering, Runtime domain normalization, and timing jitter.

- [x] **Phase 40: Core Proxy (pass-through, logging)** — Fireable CDP WebSocket pass-through proxy with in-memory ring-buffer traffic logging, flag-gated behind `--cdp-proxy` (default OFF), wired into the browser launch path via godoll's `WithCDPWrapper`. ✅ Completed
- [x] **Phase 41: Runtime Domain Normalization** — Intercept and normalize Runtime domain CDP responses to suppress property-getter triggering when Runtime IS enabled (opt-in console/request capture path). ✅ Completed
- [x] **Phase 42: Timing Jitter + `cdp-traffic` Command** — Configurable CDP command dispatch timing jitter + a `rod-cli cdp-traffic` diagnostic command that reads the proxy's ring buffer. ✅ Completed

### Phase 40: Core Proxy (pass-through, logging)

**Goal**: Implement the core CDP WebSocket proxy — a pass-through proxy that implements `cdp.WebSocketable`, logs all messages to an in-memory ring buffer, and is wired into the browser launch path gated behind `--cdp-proxy` (default OFF).
**Depends on**: Phase 39 (CDP-DEEP-01 research & design)
**Requirements**: CDP-PROXY-01 through CDP-PROXY-05
**Success Criteria** (what must be TRUE):

  1. `go build ./...` succeeds with the proxy wired but default-off.
  2. A browser session with `--cdp-proxy` navigates a page successfully (no breakage).
  3. With `--cdp-proxy`, the proxy's `Traffic()` contains expected CDP messages (Page.frameNavigated, etc.) after navigation.
  4. Without `--cdp-proxy`, behavior is identical to pre-phase-40 (no regression).
  5. The ring buffer enforces its capacity (doesn't leak memory).

**Plans**: 1/1 plan complete

- [x] 40-PLAN.md — Proxy core + config surface + CLI flag + browser launch wiring + context storage

### Phase 41: Runtime Domain Normalization

**Goal**: When Runtime is enabled (opt-in via `--console-capture` or plugins), normalize Runtime domain CDP responses to suppress property-getter triggering.
**Depends on**: Phase 40
**Requirements**: CDP-NORM-01, CDP-NORM-02
**Success Criteria** (what must be TRUE):

  1. With `--cdp-proxy` and `--console-capture`, the `cdpTell` probe returns `"no-signal"` (no stack-getter triggering).
  2. Normalization does not break `console-capture` functionality.
  3. With `--cdp-proxy` but WITHOUT `--console-capture`, behavior is unchanged.

**Plans**: TBD

### Phase 42: Timing Jitter + `cdp-traffic` Command

**Goal**: Add configurable timing jitter to CDP command dispatch and a diagnostic command that reads the proxy's ring buffer.
**Depends on**: Phase 41
**Requirements**: CDP-JITTER-01, CDP-TRAFFIC-01
**Success Criteria** (what must be TRUE):

  1. CDP command dispatch includes a random 0-N ms delay (`--cdp-jitter-ms`, default 0 = off).
  2. `rod-cli cdp-traffic` prints the proxy's logged CDP messages in human-readable format.
  3. `rod-cli cdp-traffic --json` emits machine-readable JSON.
  4. A `--no-cdp-proxy` bypass flag exists for zero-risk deployment.

**Plans**: TBD
