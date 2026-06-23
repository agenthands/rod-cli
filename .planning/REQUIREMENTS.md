# Requirements: rod-cli — v1.6 Proven & Configurable Stealth

**Defined:** 2026-06-24
**Core Value:** Native, token-efficient browser automation via standard I/O explicitly designed for LLM integration.

> **Milestone framing:** godoll already implements nearly every stealth capability below — v1.6 is **expose + validate + wire**, not build-from-scratch. The genuine new engineering is the deterministic detection harness, the session-persistent config surface, fingerprint *consistency* (single source of truth), and the two unwired godoll gaps (`EvadeWebRTC`, the `ApplyCanvasNoise` stub). The harness gates everything: each requirement below is "done" only when the harness asserts it against the live binary.

## v1 Requirements

### Detection Harness

- [x] **HARNESS-01**: A self-contained, offline detection test page (sannysoft-style assertions + a curated CreepJS-style subset) is bundled via `go:embed` and served from a local `127.0.0.1:0` fixture server, mirroring the existing `internal/plugin/scanner/testserver` pattern.
- [ ] **HARNESS-02**: An end-to-end test drives the real `rod-cli` binary against the harness, navigates, and asserts each table-stakes signal by reading it back from the live page (not from source) — running with zero network egress so it is deterministic green-or-red.
- [ ] **HARNESS-03**: A test CI job runs the harness on every push (no test CI exists today), baselined against the current binary so existing leaks are surfaced rather than hidden.

### Stealth Validation

- [ ] **VALIDATE-01**: A user can run a stealth-check command against a page and get a per-signal verdict (`navigator.webdriver`, `navigator.plugins`, UA-without-`HeadlessChrome`, WebGL vendor ≠ SwiftShader/llvmpipe, `navigator.permissions`, `navigator.languages`, screen dims, `window.chrome`/`chrome.runtime`, timezone).
- [ ] **VALIDATE-02**: With `--raw`, the stealth-check emits a single-line machine-readable `PASS`/`FAIL` plus only the failing signals (e.g. `webdriver=ok webgl=FAIL(SwiftShader)`), token-efficient for LLM callers — no full-page dump.
- [ ] **VALIDATE-03**: A silent no-op in the evasion path fails loudly — fingerprint generation / `EvasionManager.Apply()` errors surface instead of being swallowed (today `_ = em.Apply()` discards them).

### Configurable Fingerprint

- [ ] **FINGERPRINT-01**: A user can pin a coherent fingerprint per session — browser/OS/locale tuple (UA, platform, locale, timezone, screen, hardwareConcurrency, deviceMemory, vendor) — via CLI flags mapped onto godoll `stealth.Profile`, deriving dependent fields rather than setting raw contradictory values.
- [ ] **FINGERPRINT-02**: A consistency validator rejects or auto-derives incoherent combinations per the invariants (UA ↔ Client-Hints ↔ platform, locale ↔ languages ↔ Accept-Language, timezone ↔ proxy-IP geo, plausible screen geometry / hardware) and fails loudly on contradiction.
- [ ] **FINGERPRINT-03**: Client-Hints are derived from the active UA/OS instead of the current hardcoded version `121`, eliminating the UA↔UA-CH mismatch tell.

### Stealth Profiles & Config Surface

- [ ] **PROFILE-01**: A user can save and load a named stealth profile as a JSON config file (godoll `stealth.Profile` `Save`/`LoadProfile`), set once and inherited by every later command hitting the same daemon session.
- [ ] **PROFILE-02**: Stealth configuration resolves with deterministic precedence — CLI flag > profile file > built-in default — once at daemon-spawn time, with no per-command stealth state bleeding across named sessions.

### Per-Session Proxy

- [ ] **PROXY-01**: A user can route a session through an HTTP **or** SOCKS5 proxy via `--proxy`, bound per named session (not a single global IP), using godoll's proxy API rather than the current bare `launcher.Proxy()`.
- [ ] **PROXY-02**: Proxy authentication is handled via CDP (`Fetch.continueWithAuth`), not URL-embedded credentials (which Chrome removed), with `--proxy-auth` for credentials.

### Anti-Fingerprint Hardening

- [ ] **HARDEN-01**: WebRTC local-IP leak prevention is wired (godoll `EvadeWebRTC` + `WithWebRTCLeakProtection`, which `Apply()` does not currently call) and asserted by the harness so a real local IP cannot leak past a proxy.
- [ ] **HARDEN-02**: Canvas/WebGL/Audio noise is exposed as profile toggles with the noise **stable within a session** (re-reads return an identical hash), filling godoll's no-op `ApplyCanvasNoise` stub; the harness asserts hash stability.

### Human-Behavior Tuning

- [ ] **HUMANIZE-01**: A user can tune human-like interaction (typing speed, typo rate, delay jitter, mouse-path realism, scroll behavior) via flags/profile, threading godoll `humanize` options that `actions.go` currently calls with defaults only.

### Best-Effort Live Validation

- [ ] **LIVEWAF-01**: An opt-in, non-blocking live smoke check (Cloudflare/DataDome/CreepJS) behind a `//go:build detection_live` tag, documented as best-effort and manual — explicitly kept out of the blocking CI gate.

## v2 Requirements

### CDP Footprint

- **CDP-01**: Reduce or document the detectable CDP/`Runtime.enable` footprint (spike with a documented ceiling; rod-cli relies on Runtime/Network events for console/request logging).

### Network-Layer Identity

- **TLS-01**: TLS/JA3-JA4 fingerprint alignment (uTLS-style) — structurally outside a JS-injection CLI; would need a network-layer rewrite.

### Profile Library

- **PROFILE-LIB-01**: A library of many vetted, coherent device profiles beyond a curated desktop set.

## Out of Scope

| Feature | Reason |
|---------|--------|
| "Bypasses Cloudflare/DataDome" guarantee or blocking CI gate against live WAFs | They score TLS (JA3/JA4) + IP reputation + behavior — layers a JS-injecting CLI cannot control. Non-deterministic → flaky CI. rod-cli hardens the browser/JS layer only; TLS+IP are the operator's job. |
| URL-embedded proxy credentials (`http://user:pass@host`) | Chrome removed them; SOCKS5 auth unsupported by `--proxy-server`. Use CDP `Fetch.continueWithAuth` instead. |
| Random-everything-per-request fingerprints | Breaking consistency *lowers* detector trust (mismatched lies) and breaks session state. Pin a coherent profile per session instead. |
| Over-noised / per-call-varying canvas/audio | Unstable hash across reads is itself a tell. Use subtle, stable-per-session noise. |
| Mobile fingerprints from desktop headless | Desktop reporting mobile signals is exactly the inconsistency detectors flag. Ship vetted desktop profiles only. |
| Bundled CAPTCHA solving | Scope creep, ToS/legal exposure, external paid deps; breaks the zero-dependency single-binary constraint. |
| Live CreepJS/sannysoft as a blocking CI dependency | Remote, evolving, network-dependent → non-deterministic. Vendor a curated offline subset for CI; keep live as best-effort. |
| TLS/JA3-JA4 alignment in v1.6 | Out of the JS-injection layer; deferred to v2 (TLS-01). |

## Traceability

Phase mapping assigned by the roadmapper (v1.6 = Phases 24–29).

| Requirement | Phase | Status |
|-------------|-------|--------|
| HARNESS-01 | Phase 24 | Complete |
| HARNESS-02 | Phase 24 | Pending |
| HARNESS-03 | Phase 24 | Pending |
| VALIDATE-03 | Phase 24 | Pending |
| PROFILE-01 | Phase 25 | Pending |
| PROFILE-02 | Phase 25 | Pending |
| PROXY-01 | Phase 25 | Pending |
| PROXY-02 | Phase 25 | Pending |
| FINGERPRINT-01 | Phase 26 | Pending |
| FINGERPRINT-02 | Phase 26 | Pending |
| FINGERPRINT-03 | Phase 26 | Pending |
| VALIDATE-01 | Phase 26 | Pending |
| VALIDATE-02 | Phase 26 | Pending |
| HARDEN-01 | Phase 27 | Pending |
| HARDEN-02 | Phase 27 | Pending |
| HUMANIZE-01 | Phase 28 | Pending |
| LIVEWAF-01 | Phase 29 | Pending |

**Coverage:**

- v1 requirements: 17 total
- Mapped to phases: 17 ✓ (100% — every requirement maps to exactly one phase)
- Unmapped: 0

**Phase → Requirement summary:**

- Phase 24 (Detection Harness & CI Backbone): HARNESS-01, HARNESS-02, HARNESS-03, VALIDATE-03
- Phase 25 (Stealth Config Surface & Per-Session Proxy): PROFILE-01, PROFILE-02, PROXY-01, PROXY-02
- Phase 26 (Configurable Fingerprint & Consistency Validator): FINGERPRINT-01, FINGERPRINT-02, FINGERPRINT-03, VALIDATE-01, VALIDATE-02
- Phase 27 (Canvas/WebGL/WebRTC Hardening): HARDEN-01, HARDEN-02
- Phase 28 (Human-Behavior Tuning): HUMANIZE-01
- Phase 29 (Best-Effort Live Validation): LIVEWAF-01

---
*Requirements defined: 2026-06-24*
*Last updated: 2026-06-24 after roadmap creation (Phases 24–29 mapped)*
</content>
