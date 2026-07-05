# Milestones

## v2.2 Pi Extension (Shipped: 2026-06-27)

**Phases completed:** 4 phases (47–50), 4 plans

**Key accomplishments:**

- First-class TypeScript Pi extension under `extensions/pi/` (`@agenthands/rod-cli-pi`) wrapping rod-cli as 13 native `browse_*` tools (goto, snapshot, click, type, eval, screenshot, wait, tabs, navigate, scroll, cookies, storage, fill_form) — TypeBox-typed parameters, `promptGuidelines`, shell-out via `pi.exec()`. Zero Go changes.
- Lifecycle hooks: `session_start` verifies the binary (`--version`, read-only), `session_shutdown` closes the daemon best-effort gated on `reason === "quit"`; lazy daemon start on first `browse_goto`.
- Hardened: `--` argv-injection separator on all 13 tools, input validation (URL/selector/expression caps), per-command timeouts, AbortSignal propagation.
- 116 tests (4 smoke + 103 adversarial + 9 integration, vitest); `tsc --noEmit` strict clean. Independent security review PASSED (no gaps, no tech-debt).
- **Not yet published to npm** — that is v2.3's job.

See `.planning/v2.2-MILESTONE-AUDIT.md`, `.planning/v2.2-SECURITY.md`, `.planning/milestones/v2.2-REQUIREMENTS.md`.

---

## v2.1 CDP Proxy Hardening & Diagnostics (Shipped: 2026-06-27)

**Phases completed:** 4 phases (43–46; consolidated to 3)

**Key accomplishments:**

- `TestProxyTraffic` (cdp-traffic JSON assertion) + `TestProxyCdpTell` (live-page Runtime normalization probe) — closed the v2.0 carry-forward live-browser proxy integration tests.
- `--cdp-jitter-ms` sanity validation with a stderr soft-warning above 1000ms; `cdp-traffic` help gained a sensitivity caveat.
- Confirmed the godoll font-spoof injector is already real (godoll `1d90494`), covered by the existing `TestFontSpoof` harness gate.

See `.planning/v2.1-MILESTONE-AUDIT.md`.

---

## v2.0 CDP-DEEP-01 Build — MITM WebSocket Proxy (Shipped: 2026-06-26)

**Phases completed:** 3 phases (40–42)

**Key accomplishments:**

- In-process MITM CDP WebSocket proxy (`internal/cdpproxy/`) implementing `cdp.WebSocketable`: pass-through with an in-memory ring buffer, wired into browser launch behind `--cdp-proxy` (default OFF).
- `Runtime.getProperties` response normalization to suppress getter-triggering accessor descriptors when Runtime is enabled (opt-in via `--console-capture`/plugins).
- Configurable timing jitter on CDP dispatch + a `rod-cli cdp-traffic` diagnostic command reading the ring buffer.

See `.planning/v2.0-MILESTONE-AUDIT.md`, `.planning/v2.0-SECURITY.md`.

---

## v1.9 godoll Hygiene & CDP-DEEP-01 Research (Shipped: 2026-06-26)

**Phases completed:** 2 phases (38–39)

**Key accomplishments:**

- Closed the two remaining v1.7 security-review hygiene items — F2 (backslash added to rod-cli's reject list, defense-in-depth) and F4 (godoll `EnableRequestInterception` now `json.Marshal`s platform).
- CDP-DEEP-01 research & design: evaluated browser-patching vs MITM/alternate-transport vs patched-DevTools-endpoint against Chrome's detection surface; produced the grounded recommendation + executable PLAN that seeded v2.0.

---

## v1.8 Debt Cleanup & Coding-Assistant Onboarding (Shipped: 2026-06-26)

**Phases completed:** 4 phases (34–37)

**Key accomplishments:**

- Toolchain bump + vuln gate (Phase 34): pinned the Go toolchain, aligned CI/release, cleared the 13 stdlib vulns fixed in go1.26.1, gated against regression (govulncheck in CI).
- Plugin-path CDP-ledger closure (Phase 35): CDP domains enabled lazily via the plugin lifecycle path are now observed by the per-session ledger the v1.7 footprint guarantees depend on.
- Font-spoofing work (Phase 36) and coding-assistant onboarding docs (Phase 37): authoritative install + agent-skill docs for the five major coding assistants via the agent-skill → shell-out path (no MCP), README-discoverable.

---
## v1.7 Complete Evasion Stack (Shipped: 2026-06-26)

**Phases completed:** 3 phases, 7 plans (Phase 31 / TLS cancelled — real-Chrome-only)

**Key accomplishments:**

- **CDP footprint reduction (Phase 30):** a plain `goto` session now enables **none** of the `Runtime`/`Network`/`Fetch` CDP domains — console + request capture are opt-in flags (`--console-capture`/`--request-capture`, default OFF), the network interceptor is lazy (created only on a mock route), and HTTP↔JS identity coherence moved to the **zero-enable `Emulation.setUserAgentOverride`** path (design fork resolved C+D, operator-chosen). A per-session CDP-domain ledger + a deterministic offline harness gate (`TestCDPFootprintBaseline`) assert the reduced baseline; `docs/cdp-footprint.md` rewritten into a per-command inventory + honest ceiling (deeper obfuscation → v2 CDP-DEEP-01).
- **TLS spoofing deliberately OUT of scope:** rod-cli drives real Chrome, so its TLS/JA3-JA4 is authentic by construction — Phase 31 + TLS-01..04 cancelled (TLS spoofing lives in the separate "munch" project). The deliberate stance is documented in `docs/stealth-validation.md`.
- **Chrome-only profile library (Phase 32):** 6 vetted, coherent desktop profiles embedded via `//go:embed` (Windows 11/10 + macOS Apple-Silicon/Intel, resolution/hardware variants), `--profile=list`, built-in-first name resolution (custom profiles + CLI precedence preserved), and a real (non-vacuous) PROF-02 vetting gate iterating every built-in through the v1.6 consistency validator + the harness.
- **Advanced evasion (Phase 33):** activated godoll's dormant fingerprint dimensions — fonts, media devices, battery, codecs — **coherently** (generator constrained to the profile's OS/locale via `FPWithOS`/`FPWithLocales` + `em.SetFingerprint`), behind 4 new CLI>profile>default toggles (`--font-spoof`/`--media-devices-spoof`/`--battery-spoof`/`--codec-spoof`, default ON), harness-asserted for coherence + stability + toggle-off (media-devices/battery/codecs live-proven; the godoll font injector is a documented no-op).
- **Milestone gates:** every shipped phase independently qa-verified; an independent security review cleared the milestone with **no blocker** (profile→JS/CDP injection defended by two layers; proxy-auth invariant intact); docs/ + codebase map refreshed and doc-verified (0 drift).

**Known follow-ups:** godoll font-spoof no-op (make observable); plugin-path CDP-ledger coverage hole (SMTC dominance-spec candidate); toolchain bump go1.26.0 → go1.26.1 (security F1, mitigated by localhost-only daemon).

See `.planning/milestones/v1.7-MILESTONE-AUDIT.md`, `-ROADMAP.md`, `-REQUIREMENTS.md`, `-SECURITY.md`.

---

## v1.6 v1.6 (Shipped: 2026-06-25)

**Phases completed:** 6 phases, 20 plans, 38 tasks

**Key accomplishments:**

- Offline `127.0.0.1:0` detection fixture server (`internal/detect`) serving a self-authored, `//go:embed`-bundled sannysoft-style probe page that writes table-stakes + informational WebRTC/CDP verdicts into `window.__detect`.
- A deterministic, offline e2e test (`tests/detection_test.go`) drives the live `../rod-cli` binary against the `internal/detect` fixture and asserts every extended table-stakes stealth signal by reading `window.__detect.<signal>` back from the live page — green against the documented baseline with WebRTC + Client-Hints KNOWN-RED markers kept as executing assertions.
- A cohesive `StealthConfig` substrate with a CLI > profile > default precedence resolver, three forwarded global flags, and a loud-failure profile load — established as the single place every later v1.6 stealth feature plugs into.
- The bare `launcher.Proxy(cfg.Proxy)` is replaced with godoll's `ProxyConfig.ApplyToLauncher` path — `cfg.Stealth.Proxy`/`ProxyAuth` parse into a credential-safe `browser.ProxyConfig`, authenticated proxies get a CDP `SetupBrowserAuth` handler before first navigation and a local relay whose cleanup is stored on `Context` and stopped on session close.
- An offline loopback HTTP forward-proxy fixture plus four live-binary e2e tests that prove per-session proxy isolation (no bleed), CDP `--proxy-auth` is answered for correct creds and 407-enforced for wrong, a `stealth.Profile` JSON round-trips and is inherited across commands on a session, and `--proxy-auth` secrets never reach stdout/stderr or the `.port` file.
- `stealth-check [url]` now reads 11 table-stakes signals back from the LIVE page via a shared embedded probe and emits a human per-signal table, a single-line `--raw` PASS/FAIL-with-only-failing-signals, or a clean structured `--json` object.

---

## v1.6 v1.6 (Shipped: 2026-06-25)

**Phases completed:** 6 phases, 20 plans, 38 tasks

**Key accomplishments:**

- Offline `127.0.0.1:0` detection fixture server (`internal/detect`) serving a self-authored, `//go:embed`-bundled sannysoft-style probe page that writes table-stakes + informational WebRTC/CDP verdicts into `window.__detect`.
- A deterministic, offline e2e test (`tests/detection_test.go`) drives the live `../rod-cli` binary against the `internal/detect` fixture and asserts every extended table-stakes stealth signal by reading `window.__detect.<signal>` back from the live page — green against the documented baseline with WebRTC + Client-Hints KNOWN-RED markers kept as executing assertions.
- A cohesive `StealthConfig` substrate with a CLI > profile > default precedence resolver, three forwarded global flags, and a loud-failure profile load — established as the single place every later v1.6 stealth feature plugs into.
- The bare `launcher.Proxy(cfg.Proxy)` is replaced with godoll's `ProxyConfig.ApplyToLauncher` path — `cfg.Stealth.Proxy`/`ProxyAuth` parse into a credential-safe `browser.ProxyConfig`, authenticated proxies get a CDP `SetupBrowserAuth` handler before first navigation and a local relay whose cleanup is stored on `Context` and stopped on session close.
- An offline loopback HTTP forward-proxy fixture plus four live-binary e2e tests that prove per-session proxy isolation (no bleed), CDP `--proxy-auth` is answered for correct creds and 407-enforced for wrong, a `stealth.Profile` JSON round-trips and is inherited across commands on a session, and `--proxy-auth` secrets never reach stdout/stderr or the `.port` file.
- `stealth-check [url]` now reads 11 table-stakes signals back from the LIVE page via a shared embedded probe and emits a human per-signal table, a single-line `--raw` PASS/FAIL-with-only-failing-signals, or a clean structured `--json` object.

---

## v1.5 Plugin Ecosystem Documentation (Shipped: 2026-06-23)

**Phases completed:** 3 phases, 11 plans, 7 tasks

**Key accomplishments:**

- Polished flagship XSS-scanner example plus per-hook recipes and a copyable starter (`plugins/examples/` + `docs/plugins/examples/`), validated live end-to-end against the bundled vulnerable test app; landed three small engine fixes (`api.GetLocalStorage()`, functional `plugin run` via `RunFunc`, CDP DOM-domain enable for `onDOMNodeInserted`) and removed the always-on startup banner for token-efficiency.
- Authoritative `docs/plugins/lifecycle-hooks.md` documenting all four plugin lifecycle hooks (onRequest, onResponse, onLoad, onDOMNodeInserted) with their CDP proto types, key payload fields, and a worked snippet each, grounded in `internal/plugin/lifecycle.go`.
- docs/plugins/authoring.md — a zero-to-running tutorial (copy starter → write onResponse hook → plugin load → goto → plugin run getResults) that reflects the shipped binary and links out to the Phase 21 refs and Phase 22 examples.
- Plugin docs index (docs/plugins/README.md) grouping all seven plugin doc pages into Getting Started / Reference / Examples, with a one-click Plugin Development link added to the top-level README.

---

## v1.4 Plugin Architecture (Shipped: 2026-06-22)

**Phases completed:** 4 phases (17–20), 4 plans

**Key accomplishments:**

- Phase 17 — Engine Sandbox: embedded script engine and plugin loader inside the daemon.
- Phase 18 — Lifecycle Emitters: `OnRequest`, `OnResponse`, `OnLoad`, `OnDOMNodeInserted` hooks.
- Phase 19 — State Context Sharing: plugins read token-optimized snapshots, cookies, and localStorage.
- Phase 20 — Plugin CLI Interface: `rod-cli plugin load` / `list` / `run`.
- Shipped a working XSS scanner plugin demonstrating the engine end-to-end.

---

## v1.3 Godoll Migration (Shipped: 2026-06-21)

**Phases completed:** 1 phases, 1 plans, 0 tasks

**Key accomplishments:**

- (none recorded)

---
