# Roadmap: rod-cli

## Milestones

- ✅ **v1.0 Core CLI Foundation** — shipped
- ✅ **v1.1 Stealth & Humanization** — shipped ([archive](milestones/v1.1-ROADMAP.md))
- ✅ **v1.2 First-Class Agent Skills & Documentation** — shipped
- ✅ **v1.3 Godoll Migration** — shipped ([archive](milestones/v1.3-ROADMAP.md))
- ✅ **v1.4 Plugin Architecture** — shipped ([archive](milestones/v1.4-ROADMAP.md))
- ✅ **v1.5 Plugin Ecosystem Documentation** — Phases 21–23 (shipped 2026-06-23) ([archive](milestones/v1.5-ROADMAP.md))
- 🚧 **v1.6 Proven & Configurable Stealth** — Phases 24–29 (in progress)

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

### 🚧 v1.6 Proven & Configurable Stealth (In Progress)

**Milestone Goal:** Turn rod-cli's already-wired godoll stealth from "compiles and runs" into "provably evades detection and is configurable per session" — validated against a deterministic offline detection harness (reading from the live page, never from source), exposed through an agent-friendly session-persistent config surface, and extended with the evasion wiring that matters (WebRTC leak prevention, stable canvas noise, fingerprint consistency). Brownfield: this milestone **proves, configures, and wires** existing godoll capability — it does not rebuild it.

- [x] **Phase 24: Detection Harness & CI Backbone** — Offline, deterministic detection test server + first test CI job, baselined against the current binary so existing leaks surface (completed 2026-06-24)
- [x] **Phase 25: Stealth Config Surface & Per-Session Proxy** — Session-persistent stealth config (flags + named profile file, precedence-resolved at daemon spawn) plus per-session HTTP/SOCKS5 proxy with CDP auth (completed 2026-06-24)
- [ ] **Phase 26: Configurable Fingerprint & Consistency Validator** — Pinnable coherent fingerprint from a single source of truth, consistency invariant gate, the CH-121 fix, and a user-facing per-signal stealth-check verdict
- [ ] **Phase 27: Canvas/WebGL/WebRTC Hardening** — Wire the two genuine godoll gaps: WebRTC IP-leak prevention and stable-per-session canvas/WebGL/audio noise, both harness-asserted
- [ ] **Phase 28: Human-Behavior Tuning** — Thread godoll humanize options (typing speed, typo rate, delay jitter, mouse path, scroll) through actions as flags/profile fields
- [ ] **Phase 29: Best-Effort Live Validation** — Opt-in, non-blocking, build-tagged live smoke check (Cloudflare/DataDome/CreepJS) kept out of the CI gate

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

**Plans**: 3/5 plans executed
**Wave 1**

- [x] 26-01-PLAN.md — Config + validation layer: StealthConfig identity fields, 4 override flags, stealth-check registration, consistency validator + UA-anchor derivation in ResolveStealth (FINGERPRINT-01/02, VALIDATE-01/02)
- [x] 26-02-PLAN.md — godoll: kill the hardcoded CH `121` in both runtime injectors (UA-derived Sec-Ch-Ua + userAgentData) and derive FromFingerprint timezone (FINGERPRINT-03)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 26-03-PLAN.md — rod-cli runtime wiring: pin the resolved profile as the active stealth.Profile in createPage + UA-derived interceptor Sec-Ch-Ua (FINGERPRINT-01/03)
- [ ] 26-04-PLAN.md — stealth-check behavior: shared extracted probe, StealthCheck action (live-page reads, raw/json/human), daemon dispatch (VALIDATE-01/02)

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 26-05-PLAN.md — Harness gate: flip the CH KNOWN-RED to required-green, add blocking consistency-invariant + pinned-identity + stealth-check subtests (FINGERPRINT-01/02/03, VALIDATE-01/02)

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
| 26. Configurable Fingerprint & Consistency Validator | v1.6 | 3/5 | In Progress|  |
| 27. Canvas/WebGL/WebRTC Hardening | v1.6 | 0/TBD | Not started | - |
| 28. Human-Behavior Tuning | v1.6 | 0/TBD | Not started | - |
| 29. Best-Effort Live Validation | v1.6 | 0/TBD | Not started | - |
</content>
</invoke>
