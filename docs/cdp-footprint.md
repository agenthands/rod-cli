# CDP Footprint — Findings Note (Phase 24)

**Question:** Can `rod-cli` meaningfully hide its Chrome DevTools Protocol (CDP) /
`Runtime.enable` footprint, given the daemon relies on CDP Runtime and Network
events for console and request logging?

**Answer: No — not meaningfully within v1.6's JS-injection layer.** The CDP
transport is observable independently of any JavaScript-property spoofing, so
fingerprint/profile work cannot close this tell. This note documents the ceiling;
it is **informational and non-blocking**. No fix is attempted in Phase 24.

## Why the footprint exists

`types/context.go` `createPage()` subscribes to CDP events to power core
features:

- `Runtime.consoleAPICalled` → the `console` log stream.
- `Network.requestWillBeSent` (and related) → the request/network log.

Subscribing forces the **Runtime** and **Network** CDP domains to be *enabled*.
Domain enablement is exactly the surface that CDP-tell detectors probe (e.g. the
classic `Runtime.enable` / `console.debug` divergence and timing checks). Because
the daemon's observability depends on these domains, spoofing JS properties
(`navigator.webdriver`, plugins, WebGL, etc.) does not remove the underlying CDP
signal — the two live at different layers.

## What this phase does about it

Phase 24 **measures** the exposure rather than assuming spoofing covers it: the
offline detection harness includes informational CDP-tell probes surfaced as
`window.__detect.cdpTell` (alongside `window.__detect.webrtcIce`). These probes
are **non-blocking** — they record current truth so the signal is visible in the
harness baseline, not silently ignored. They do not gate CI and they do not
attempt to suppress the tell.

## The honest ceiling

A JS-injecting CLI cannot fully control the CDP, TLS (JA3/JA4), or IP-reputation
layers. `rod-cli` does **not** claim to be "undetectable." Hardening the
browser/JS layer (the rest of milestone v1.6) is in scope; eliminating the CDP
transport footprint is not.

**Reducing** the CDP footprint (vs. merely probing it) — e.g. opt-in logging so
Runtime/Network domains are only enabled on demand, or an alternate transport —
is deferred to **v2 (requirement CDP-01)**.
