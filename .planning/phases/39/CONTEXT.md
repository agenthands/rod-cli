---
author: architect
responsible: architect
phase: 39
phase_type: research
hard_bar: false
security_relevant: true
design_fork: true
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (CDP-DEEP-01, CDP-DEEP-02, CDP-DEEP-03)
  - .planning/ROADMAP.md (Phase 39)
  - docs/cdp-footprint.md (current honest ceiling)
---

# Phase 39: CDP-DEEP-01 Research & Design — RESEARCH

## Executive summary

**Recommendation: MITM WebSocket proxy (Approach B).** It is the most practical first step — feasible without a Chromium fork, leaves the go-rod transport layer unchanged, and can measurably reduce the observable CDP surface. Browser-patching (Approach A) is more comprehensive but requires maintaining a Chromium fork — only justified if detectors escalate beyond what the proxy can address. Patched endpoint (Approach C) is a variant of B with worse separation of concerns.

**Chosen approach: B (MITM WebSocket proxy).** A local proxy sits between go-rod's `cdp.Client` and Chrome's debugging WebSocket, selectively filtering or normalizing CDP protocol messages that are observable from the page.

## What's detectable from the page

The CDP transport leaves three categories of observable signal:

### 1. Runtime domain behavioral changes (the `cdpTell` probe)

When Chrome's `Runtime` domain is enabled (which go-rod enables by default), CDP remote-object-preview can observe property getters on objects serialized to the protocol. The classic tell: `console.debug(e)` triggers the `stack` getter on an `Error` object, and CDP's Runtime observer fires it.

**Current rod-cli state:** v1.7 Phase 30 eliminated `Runtime.enable` from the baseline — a plain `goto` session never enables Runtime. The `cdpTell` probe returns `"no-signal"` on the baseline. However, any feature that enables Runtime (console capture, certain plugins) re-opens this signal.

**Why this matters:** This is the most commonly cited CDP detection vector. It's already addressed for the baseline path. The MITM proxy would add defense-in-depth — normalizing Runtime messages even when Runtime IS enabled.

### 2. CDP protocol message patterns

Chrome sends `Page.frameNavigated`, `Target.attachedToTarget`, and other events over the WebSocket. The timing and pattern of these events is characteristic of CDP-driven automation (predictable sequence, no human interaction gaps). Detectors can observe these indirectly via page load timing and event loop behavior.

**Current rod-cli state:** These are accepted-visible (irreducible on the CDP transport). The MITM proxy could delay/jitter messages to break predictable patterns.

### 3. WebSocket connection existence

Chrome's debugging interface listens on a local TCP port. A page in a normal browsing context **cannot** enumerate local TCP connections or WebSocket endpoints from JavaScript. This is NOT a practical detection vector for web pages. It IS detectable from the OS level (process inspection, port scanning on localhost).

**Current rod-cli state:** Accepted-visible. Only detectable by OS-level inspection, not by page JS.

## Approach evaluation

### Approach A: Browser-patching

**What it is:** Patch Chromium's source to remove or hide CDP signals from page-observable JS. Could suppress the `console.debug` stack-getter behavior, hide Runtime domain behavioral differences, or replace the debugging WebSocket with an opaque transport.

**Feasibility:** High (technically possible). Chromium's DevTools protocol implementation is in `content/browser/devtools/`. Modifying the `Runtime` domain's object preview behavior or the `console.debug` serialization path is a bounded change.

**Cost:** Very high ongoing. Requires:
- Building and maintaining a Chromium fork
- Re-basing on every Chrome release (every 4 weeks)
- CI/CD for patched builds across platforms (Linux, macOS, Windows)
- Distribution: users must install the patched Chromium instead of stock

**Detection impact:** Could eliminate Runtime domain tells completely. Could hide the debugging WebSocket from localhost enumeration. This is the "comprehensive" option.

**Verdict:** Only justified if MITM proxy proves insufficient against real detectors. The maintenance burden is the dominant factor.

### Approach B: MITM WebSocket proxy (RECOMMENDED)

**What it is:** A local proxy that sits between go-rod's `cdp.Client` and Chrome's debugging WebSocket. Intercepts JSON-RPC messages in both directions, selectively filtering or normalizing protocol traffic.

**How it works:**
1. On browser launch, the proxy starts a local WebSocket server
2. go-rod connects to the proxy instead of Chrome directly
3. The proxy connects to Chrome's debugging WebSocket
4. Messages pass through; the proxy can:
   - **Filter:** Drop or modify specific CDP events/methods that are detectable
   - **Delay/jitter:** Add random delays to break timing patterns
   - **Normalize:** Strip or replace Runtime remote-object descriptors in responses
   - **Log:** Record all CDP traffic for analysis

**Feasibility:** High. go-rod's `cdp.Client` accepts any `WebSocketable` implementation. The proxy implements the same interface and delegates to Chrome. The proxy runs in-process (no separate binary needed).

**Cost:** Low. A new Go package in the rod-cli repo, ~500 lines:
- `internal/cdpproxy/proxy.go` — WebSocket proxy core
- `internal/cdpproxy/filters.go` — message filtering rules
- `types/context.go` — wire the proxy into the browser launch path
- No Chromium fork, no external dependencies beyond Go's stdlib WebSocket

**Detection impact:**
- Can normalize Runtime domain object previews (remove property getter triggering)
- Can jitter event timing to break characteristic patterns
- Cannot hide the WebSocket port from localhost (but this isn't practical page-JS-observable anyway)
- Can log and measure: the proxy becomes the CDP traffic observability layer rod-cli currently lacks

**Limitations:**
- Cannot prevent Chrome from opening the debugging port on localhost
- Runtime normalization adds latency (each message passes through an extra hop)
- Some patterns (Page.enable sequence) are fundamentally required for navigation — they can be delayed but not eliminated

### Approach C: Patched DevTools endpoint

**What it is:** Replace or modify Chrome's DevTools endpoint at the Chrome level — either by launching Chrome with a custom `--devtools-flags` equivalent, or by providing a replacement debugging frontend that filters signals.

**Feasibility:** Low. Chrome's DevTools endpoint is not designed for replacement. The closest equivalent is `--remote-debugging-pipe`, which replaces the WebSocket with a pipe — but this doesn't filter messages, it just changes the transport. A truly patched endpoint would require modifying Chrome's devtools frontend or implementing a custom debugging protocol handler.

**Cost:** High (similar to browser-patching in some variants) or impossible (no stable API for replacement).

**Verdict:** Not a distinct approach from A or B. The "pipe" variant is a transport change, not a filtering mechanism. The "replacement endpoint" variant requires Chrome source changes (back to Approach A).

## Recommendation: Approach B — MITM WebSocket proxy

**Why:**
1. **Feasible now** — no Chromium fork, no external deps, fits in rod-cli's Go codebase
2. **Measurable** — the proxy logs all CDP traffic, giving rod-cli its first real CDP-traffic observability (currently we have only the domain-enable ledger)
3. **Incremental** — start with passive logging, then add active filtering as detectors are identified
4. **Go-native** — go-rod's `WebSocketable` interface is designed for this
5. **Low risk** — the proxy can be bypassed with a flag (`--no-cdp-proxy`) if it causes issues

**Implementation plan:** See [CDP-DEEP-01-PLAN.md](CDP-DEEP-01-PLAN.md) for the concrete, phased execution plan.

## What the proxy can and cannot hide (updated honest ceiling)

| Signal | Current (v1.8) | With proxy | Still visible? |
|--------|---------------|------------|----------------|
| Runtime.enable (baseline) | Not enabled | Not enabled | No |
| Runtime.enable (console capture) | Enabled, logged | Enabled, normalized | Partial — getter triggers suppressed |
| cdpTell stack-getter | No-signal on baseline | No-signal on baseline + normalized when Runtime on | Only when Runtime is enabled (opt-in) |
| Page/Target events | Accepted-visible | Jittered | Yes — structurally required |
| WebSocket port on localhost | Accepted-visible | Accepted-visible | Yes — OS-level only, not page-JS |
| CDP command timing patterns | Accepted-visible | Jittered | Reduced |
| JA3/TLS fingerprint | Out of scope (real Chrome) | Out of scope | N/A |

**Honest ceiling:** The proxy cannot eliminate the CDP transport. A detector with OS-level access can always observe the WebSocket connection. A detector analyzing CDP command timing at high resolution can identify patterns even with jitter. The proxy **measurably reduces** the surface but does not claim to eliminate it. As with all rod-cli stealth: no "undetectable" guarantee.

## Sources

- go-rod `cdp.Client` and `WebSocketable` interface: `lib/cdp/client.go` (rod v0.116.2)
- Chrome DevTools Protocol: `content/browser/devtools/` in Chromium source
- rod-cli `cdpTell` probe: `internal/detect/detect.js:155-175`
- rod-cli CDP footprint docs: `docs/cdp-footprint.md`
