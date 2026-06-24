# Phase 26: Configurable Fingerprint & Consistency Validator - Context

**Gathered:** 2026-06-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Make the fingerprint **configurable and coherent** on top of the Phase 25 stealth-config substrate. A user pins a browser/OS/locale tuple per session from a **single in-memory source of truth** (godoll `stealth.Profile`) that feeds JS properties, HTTP headers, and Client-Hints alike. A consistency validator derives dependent fields when unset and rejects loudly on user-set contradictions. The hardcoded Client-Hints `121` literal is eliminated across the runtime header/JS path (rod-cli **and** godoll) so `Sec-CH-UA`, `navigator.userAgentData`, and the UA all tell one OS+version story. Finally, a user-facing `stealth-check` command reads per-signal verdicts back from the live page, with a token-efficient `--raw` form.

Covers: FINGERPRINT-01, FINGERPRINT-02, FINGERPRINT-03, VALIDATE-01, VALIDATE-02.
Does NOT cover: canvas/WebGL/WebRTC hardening (Phase 27), humanize tuning (Phase 28), live WAF validation (Phase 29). The fingerprint must be coherent *before* hardening adds noise on top of it (consistency-before-hardening ordering).

</domain>

<decisions>
## Implementation Decisions

### Fingerprint Configuration Surface (FINGERPRINT-01)
- **Source of truth:** the godoll `stealth.Profile` object is the single in-memory truth that feeds JS injection, HTTP headers, and Client-Hints. Profile JSON file (`--profile`, Phase 25's `Stealth.ProfilePath`) is the primary configuration surface.
- **Override flags:** add a *curated* set of high-value CLI override flags — `--user-agent`, `--locale`, `--timezone`, `--platform` — registered as global flags alongside the Phase 25 stealth flags and forwarded through `EnsureDaemon`. NOT a per-field flag for every Profile field (avoid flag bloat); remaining fields (screen, hardwareConcurrency, deviceMemory, vendor, languages) come from the profile or are derived.
- **Derivation anchor:** the **UA string** is the anchor. Chrome major version → Client-Hints version + `navigator.userAgentData`; OS token in the UA → `navigator.platform` + `Sec-Ch-Ua-Platform`. Pinning UA without platform/CH **auto-derives** the dependent fields rather than requiring an explicit full tuple (FP-01: "deriving dependent fields rather than setting raw contradictory values").
- **Preset profiles:** ship 2–3 vetted desktop presets resolvable by bare name via `--profile <name>` (the Phase 25 `resolveProfilePath` already maps bare names → `~/.rod-cli/profiles/<name>.json`). A large profile library is explicitly v2 (PROFILE-LIB-01). Desktop-only (mobile-from-desktop is out of scope).

### Consistency Validator (FINGERPRINT-02)
- **Policy:** **derive-when-unset, reject-when-user-conflicts.** If a dependent field is unset, silently auto-derive it from the anchor. If the user explicitly set two contradictory fields (e.g. UA reports Windows but `--platform=MacIntel`), **reject loudly** — never silently ship the mismatched lie.
- **Invariants (hard gates):** UA ↔ Client-Hints ↔ platform; locale ↔ languages ↔ Accept-Language; plausible screen geometry / hardware (reject impossible values).
- **Invariant (warn-only):** timezone ↔ proxy-IP geo. Geo-IP resolution needs network access and would break the offline-deterministic harness, so this is a non-blocking warning when both a proxy and timezone are set — NOT a hard gate. (Carried from Phase 25 deferred note.)
- **When:** validation runs at **daemon-spawn config-resolution time, before the browser launches** (extends Phase 25's `ResolveStealth()` path), so a contradiction fails fast before a wasted browser launch.
- **Failure surface:** non-zero exit + a clear **stderr** message naming the conflicting fields and the remedy (e.g. "platform 'MacIntel' contradicts UA OS 'Windows' — remove --platform to auto-derive, or fix the UA"). Token-efficient; never pollutes stdout for `--raw`/piped callers.

### Client-Hints Derivation / Kill `121` (FINGERPRINT-03)
- **Scope:** fix **both rod-cli and godoll.** The `121` literal executes at runtime in multiple places — rod-cli `types/context.go:509` (`Sec-Ch-Ua` interceptor), godoll `stealth/evasion.go:200` (`Sec-Ch-Ua` header), godoll `stealth/js.go:168` (`navigator.userAgentData` brand version), godoll `fingerprint/headers.go:264` (fallback). godoll is a local sibling repo (`../godoll`, same org, editable) and Phase 27 already edits it, so cross-repo fixes are in scope.
- **Major-version source:** parse `Chrome/(\d+)` from the **active resolved UA** (the pinned anchor). godoll already has UA→client-hint version machinery (`fingerprint/headers.go:344` builds `sec-ch-ua` from a version entry) — reuse it rather than re-deriving.
- **Injector topology:** converge on a **single UA-derived Client-Hints injector**; eliminate the duplicate hardcoded path so rod-cli's interceptor and godoll's evasion path don't fight over `Sec-Ch-Ua`. All three surfaces (header CH, JS `userAgentData`, UA string) must agree — that agreement IS the FINGERPRINT-02 invariant.
- **Result:** no hardcoded `121` literal remaining in the runtime header path of either repo; `Sec-CH-UA` major == UA Chrome major == `userAgentData` brand version.

### stealth-check Command (VALIDATE-01, VALIDATE-02)
- **Shape:** a `stealth-check [url]` subcommand — checks the current session's active page, or navigates to `url` first if one is given.
- **Signal source:** **reuse the Phase 24 `detect.js` probe** (`internal/detect/detect.js`, `window.__detect`) injected via `eval` and read back from the live page (the same reads the harness already proved green/red). Extract a shared snippet/subset rather than authoring a second probe, keeping command and harness consistent.
- **Verdict criteria:** codify the VALIDATE-01 table-stakes thresholds — `navigator.webdriver`=false, `navigator.plugins`>0, UA without `HeadlessChrome`, WebGL vendor ≠ SwiftShader/llvmpipe, `navigator.permissions` consistent, `navigator.languages` present, plausible screen dims, `window.chrome`/`chrome.runtime` present, timezone set.
- **Output modes:** default = human-readable per-signal verdict table; `--raw` = single line `PASS`/`FAIL` plus **only the failing signals** with reason (e.g. `FAIL webgl=FAIL(SwiftShader) webdriver=ok`) — no full-page dump; `--json` = structured object. Mirror the existing global `--raw`/`--json` output handling in `runClientCommand` (cmd.go:68–75).

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- **StealthConfig substrate (Phase 25):** `types/config.go:15-52` `StealthConfig` (fields `Proxy`, `ProxyAuth`, `ProfilePath`; reserved-fields comment at `:25-27` names exactly the Phase 26 pins — `UserAgent, Locale, Timezone, Platform, Screen, AcceptLanguage, Languages, HardwareConcurrency, DeviceMemory, Vendor, SpoofClientHints`). Precedence resolver `ResolveStealth()` at `types/config.go:115-150` (CLI > profile > default; loud `stealth.LoadProfile` failure). Profile-path resolver `resolveProfilePath()` at `config.go:89-105` (bare name → `~/.rod-cli/profiles/<name>.json`).
- **Daemon-spawn threading:** flags forwarded in `cmd.go` (`--proxy`/`--profile` as argv `cmd.go:37-38`; `--proxy-auth` out-of-band via `ROD_CLI_PROXY_AUTH` env `cmd.go:44-46`). `ResolveStealth()` called once before `NewContext` freezes config (`main.go:37-47`). One daemon process per named session ⇒ config is automatically session-persistent + session-isolated.
- **godoll `stealth.Profile`:** `../godoll/stealth/profile.go:11-50` — fields UserAgent, Platform, AcceptLanguage, Languages, Timezone, Locale, Screen{W,H,DSF}, HardwareConcurrency, DeviceMemory, Vendor, Geolocation, SpoofClientHints, SpoofAudioContext. Methods `DefaultProfile()` (:67), `LoadProfile(path)` (:85), `Save(path)` (:100). Bridge `stealth.FromFingerprint(fp)` at `../godoll/stealth/fingerprint_bridge.go:12-51` (currently hardcodes Timezone="America/New_York" at :36).
- **Fingerprint generation today:** `types/context.go:395-520` — generates `rodfingerprint.Fingerprint`, `em.SetFingerprint(fp)`, `em.Apply()` (:408). `updateInterceptorRules()` (:460-519) injects headers incl. the hardcoded `Sec-Ch-Ua` at `:509`.
- **godoll CH machinery:** `../godoll/fingerprint/headers.go:344` builds `sec-ch-ua` from a version entry (reusable for UA-derived derivation); `:264` is the `121` fallback. Hardcoded runtime injectors: godoll `stealth/evasion.go:200` (header), `stealth/js.go:168` (`userAgentData` brand "121").
- **Detection harness / probe (Phase 24):** `internal/detect/detect.js` (`window.__detect`, ~12 sync signals + async permissions/webrtc, `ready` flag); `internal/detect/server.go` offline fixture; `tests/detection_test.go:76-249` drives the real binary, `evalDetect()` helper (:46-57) reads signals via `runCli("eval", "String(window.__detect."+signal+")")`, `waitForDetectReady()` (:62-74). detection_test.go:203-230 marks CH-`121` as `KNOWN-RED (Phase 26 FINGERPRINT-03)` — flips to required-green this phase.
- **CLI command pattern:** `cmd.go:80-728` urfave/cli/v2 `&cli.App`; global flags `:86-98` (incl. `--raw`, `--json`, `--session`, `--proxy`, `--proxy-auth`, `--profile`); subcommand = `&cli.Command{Name,Usage,Flags,Action}`; client path `runClientCommand()` (`cmd.go:23-78`) → `EnsureDaemon` → `ClientExecute`; daemon dispatch switch `daemon/daemon.go:199-330`; `Request{Command,Args map[string]string}` / `Response{Result,Error}` (`daemon.go:34-42`). `--raw`/`--json` output at `cmd.go:68-75`.

### Established Patterns
- Config flows: CLI flags → packed into daemon spawn args (`EnsureDaemon`) → daemon loads `Config` → `ResolveStealth()` → `NewContext(cfg)` freezes → every later command reuses the same in-memory `rodCtx`.
- Every stealth assertion reads back via `page.Eval`/`window.__detect` from the **live** browser, never from a Go config field (carried-forward v1.5 "wired-but-silent" lesson).
- Token-efficiency: warnings/errors → stderr only; stdout stays clean for `--raw`/piped/`--json` callers.
- `*_raw.js` unminified → minified `*.js` via terser (`npm run dev`) for client-side JS assets.

### Integration Points
- `types/config.go` — extend `StealthConfig` with the reserved fingerprint fields; extend `ResolveStealth()` with flag precedence + the consistency validator (reject/derive) before `NewContext`.
- `cmd.go` — register `--user-agent`/`--locale`/`--timezone`/`--platform` global flags + forward via `EnsureDaemon`; register the `stealth-check` subcommand; daemon dispatch entry in `daemon.go`.
- `types/context.go:509` + godoll `stealth/evasion.go:200`, `stealth/js.go:168`, `fingerprint/headers.go` — UA-derived Client-Hints, single injector, remove `121`.
- `internal/detect/detect.js` — extract the shared probe snippet for `stealth-check`.
- `tests/detection_test.go` — flip the CH-`121` KNOWN-RED to required-green; add consistency-invariant + stealth-check assertions (harness gates the phase).

</code_context>

<specifics>
## Specific Ideas

- The consistency invariant is a **blocking** harness test (success criterion 1) — the harness must confirm `Sec-CH-UA`, `navigator.userAgentData`, UA, `navigator.platform`, and WebGL all tell one consistent OS+version story.
- `--raw` stealth-check output must be a single line with only failing signals — explicitly NOT a full-page dump (token-efficiency for LLM callers).
- Reuse godoll's existing `headers.go:344` version→`sec-ch-ua` builder rather than writing a parallel derivation in rod-cli.
- Fix `stealth.FromFingerprint`'s hardcoded `Timezone="America/New_York"` (godoll fingerprint_bridge.go:36) as part of making timezone configurable/derived.

</specifics>

<deferred>
## Deferred Ideas

- Canvas/WebGL/Audio noise toggles + WebRTC leak prevention → Phase 27 (HARDEN-01/02). The fingerprint must be coherent first.
- Humanize tuning knobs → Phase 28 (HUMANIZE-01).
- Live WAF/CreepJS smoke check → Phase 29 (LIVEWAF-01).
- Large vetted device-profile library (beyond the 2–3 curated desktop presets) → v2 (PROFILE-LIB-01).
- Hard timezone↔proxy-geo enforcement (geo-IP lookup) → out of scope (network dependency breaks offline determinism); warn-only here.
- Authenticated SOCKS5 relay correctness (WR-02, root cause in godoll) → revisit when SOCKS-auth is actually needed; not a Phase 26 concern.

</deferred>
