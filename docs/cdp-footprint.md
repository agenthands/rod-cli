# CDP Footprint — Inventory & Mitigation Status (Phase 30)

**Question:** Which Chrome DevTools Protocol (CDP) commands / domains does `rod-cli`
cause to be enabled, when, and what is done to reduce or obfuscate each?

**Answer (v1.7):** The always-on footprint is **eliminated on the baseline path.**
A plain session — `goto` + `snapshot`, no capture flags, no mock routes, no plugins
— enables **none** of `Runtime`, `Network`, or `Fetch`. The features that need those
domains now enable them lazily / opt-in ("footprint follows the feature"), and HTTP
identity coherence was moved onto Chrome's zero-footprint `Emulation` domain. This
supersedes the Phase-24 "Answer: No" findings note — the reduction that note
deferred is now real. Deeper obfuscation (browser-patching / MITM / alternate
transport) remains the honest ceiling, deferred to **v2 (CDP-DEEP-01)**.

This is the v1.7 CDP-02 inventory. It is **true against HEAD** and cross-links the
[Stealth Validation](stealth-validation.md) Tier framing.

---

## Per-domain inventory

Status legend: **reduced** = no longer on the baseline (opt-in / lazy / removed) ·
**obfuscated** = emitted but made to look like a normal Chrome · **accepted-visible**
= present and not reducible without leaving the CDP transport (justified per row).

| Domain / Command | When sent / trigger | Status |
|---|---|---|
| **`Runtime.enable`** (via `Runtime.consoleAPICalled` subscription) | ONLY when the session is spawned with `--console-capture` (default OFF). A plain session never subscribes, so it is never enabled. | **reduced** |
| **`Network.enable`** (via `Network.requestWillBeSent` subscription) | (a) when the session is spawned with `--request-capture` (default OFF); OR (b) when **any plugin is loaded** — `BindLifecycle` subscribes to `NetworkRequestWillBeSent` unconditionally on load, whether or not the plugin defines `onRequest`/`onResponse`. Not enabled on the plain baseline. | **reduced** |
| **`Fetch.enable`** (godoll network interceptor) | ONLY when a mock `route` is added (lazily created + enabled on first `AddRoute`, disabled when the last route is removed). A session that never mocks a route never enables Fetch. | **reduced** |
| **`Emulation.setUserAgentOverride`** (identity: UA / Accept-Language / Sec-Ch-Ua* / `navigator.userAgentData`) | Once at page setup, built from the active `stealth.Profile`. The **Emulation domain has no `enable` command**, so it adds **zero** domain-enablement footprint while Chrome natively emits coherent client hints (WIRE-VERIFY confirmed on the outgoing wire with no `Network.enable`). Replaces the old always-on Fetch interceptor identity rule. | **obfuscated** (zero-enable; coherent with the JS identity) |
| **`DOM.enable`** / `DOM.getDocument` | When **any plugin is loaded** — `BindLifecycle` (`internal/plugin/lifecycle.go`) calls `DOMEnable` unconditionally on load, whether or not the plugin defines `onDOMNodeInserted`. Not enabled on the plain baseline. | **reduced** |
| **`Page.enable`** + `Target.*` | Enabled structurally by go-rod for navigation, load events, and tab/page creation — required for `goto` itself. | **accepted-visible** — see below |

### Why `Page`/`Target` are accepted-visible (earn-accepted-visible)
`Page.enable` and the `Target.*` commands are not a rod-cli choice: go-rod enables
them to drive navigation and observe load events — the irreducible cost of driving a
browser over CDP at all. **Any** CDP-controlled Chrome (including a developer with
DevTools open) enables `Page`, so it is not a distinguishing "automation" tell the
way an always-on `Runtime.enable` was. Removing it would mean not using the CDP
transport — i.e. browser-patching or an alternate transport, which is exactly the
deep work deferred to v2 (CDP-DEEP-01). Marking it accepted-visible is the honest
classification, not a convenience dodge.

### Behavior change worth noting
The old always-on Fetch interceptor also force-deleted the `X-Requested-With` header
on every request. That strip is gone with the catch-all rule. It is a non-issue:
Chrome does not emit `X-Requested-With` natively (WIRE-VERIFY observed it absent on
the plain Emulation path), so there is nothing to delete on the baseline, and the
Emulation domain carries no header-deletion knob.

---

## How the reduction is measured (CDP-01 measured, CDP-03 baseline)

- **Deterministic gate (Tier 1, blocking).** The daemon records a per-session
  CDP domain-enable ledger (`GetEnabledCDPDomains`). The offline test
  `TestCDPFootprintBaseline` (`types/cdp_footprint_test.go`) asserts a plain
  session records **zero** of {Runtime, Network, Fetch}; positive controls prove
  the ledger is real (each opt-in feature records exactly its domain). Runs under
  plain `go test ./...`, no network egress.
  - **Scope (honest):** this gate verifies rod-cli's own *instrumented*
    enable-points, not raw CDP wire traffic. It catches a regression that re-routes
    a feature onto the baseline; it cannot by itself catch a domain enabled through a
    path that bypasses the instrumentation. The concrete instance today is the
    **plugin lifecycle binder** (`internal/plugin/lifecycle.go`): loading a plugin
    enables `Network`/`DOM` on the wire but does not call `recordCDPDomainLocked`, so
    `GetEnabledCDPDomains()` does not reflect it. A mechanical completeness guard
    (e.g. an `.smtc/analyzers` dominance spec, routing the plugin binder through the
    ledger, or sniffing real CDP commands) is a candidate strengthening for v2.
- **Wire-level identity confirmation.** `TestNetworkEvasionHeaders` drives a plain
  `goto` and confirms the spoofed `Sec-Ch-Ua` / UA / `Accept-Language` reach the
  outgoing wire under the Emulation override — the WIRE-VERIFY that chose the
  zero-footprint design over a `Network.enable` fallback.
- **Informational realism (Tier 2, non-blocking).** The offline harness exposes
  `window.__detect.cdpTell`; the live suite (`tests/detection_live_test.go`,
  `//go:build detection_live`) logs the enabled-domain ledger / cdpTell verdict per
  live target. Best-effort, **never** gates CI.

See [Stealth Validation](stealth-validation.md) for the Tier-1 (offline, blocking)
vs Tier-2 (live, opt-in, non-blocking) framing.

---

## The honest ceiling

`rod-cli` does **not** claim to be "undetectable." The baseline CDP footprint is now
minimal, but the CDP *transport itself* remains observable to a determined detector:
the `Page`/`Target` domains are enabled (irreducibly, to drive the browser), and any
opted-in feature pays its domain's footprint by design. Fully obfuscating or
eliminating the CDP transport — via **browser-patching, a MITM/alternate transport,
or a patched DevTools endpoint** — is out of scope for v1.7 and deferred to
**v2 (CDP-DEEP-01)**. The TLS/JA3 and IP-reputation layers remain outside the JS/CDP
surface entirely (see the stealth-validation honest ceiling).
