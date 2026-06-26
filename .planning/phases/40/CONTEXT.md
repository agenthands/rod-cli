---
author: architect
responsible: architect
phase: 40
phase_type: implementation
hard_bar: true
security_relevant: true
design_fork: false
status: locked
parent_artifacts:
  - .planning/phases/39/CDP-DEEP-01-PLAN.md (chosen approach: MITM WebSocket proxy)
  - .planning/phases/39/CONTEXT.md (research grounding)
  - .planning/REQUIREMENTS.md (CDP-DEEP-01, CDP-DEEP-02)
---

# Phase 40: Core CDP WebSocket Proxy — CONTEXT

## Goal

Implement the MITM WebSocket CDP proxy core: a pass-through proxy that sits between go-rod's `cdp.Client` and Chrome's debugging WebSocket, implementing `cdp.WebSocketable`. In v1 (this phase), the proxy is **transparent pass-through** — all messages are forwarded unchanged — but every message is logged to an in-memory ring buffer for diagnostics. This becomes the foundation for Phases 41 (Runtime normalization) and 42 (timing jitter + `cdp-traffic` command).

## What we're building

```text
go-rod cdp.Client  ──►  cdpproxy.Proxy  ──►  Chrome DevTools WebSocket
                         │
                         └── ring buffer (Traffic() for diagnostics)
```

### Core type: `Proxy`

```go
type Proxy struct {
    inner cdp.WebSocketable  // the real Chrome WebSocket
    log   []CDPMessage       // ring buffer
    cap   int                // max log entries
}
```

- Implements `cdp.WebSocketable` (Send, Read)
- Created via `New(inner, cap)` — wraps an existing WebSocketable
- `Send(data)` → logs direction="send", forwards to `inner.Send(data)`
- `Read()` → reads from `inner.Read()`, logs direction="recv", returns data unchanged
- `Traffic()` → returns a copy of the logged messages

### Wiring into rod-cli

- `types/config.go`: `StealthConfig.CDPProxy *bool` field (YAML: `cdpProxy`)
- `cmd.go`: `--cdp-proxy` bool flag (default OFF)
- `types/context.go`: In `launchBrowser()`, when CDPProxy is true, wrap via `browserOptions.WithCDPWrapper()` which godoll threads into the CDP client transport
- The `Proxy` instance is stored on `Context.cdpProxy` for future diagnostics commands (Phase 42)

### Flag gating

- `--cdp-proxy` (default OFF) — enables the proxy
- Future: `--no-cdp-proxy` bypass flag for zero-risk deployment (Phase 42)

## Requirements traceability

| REQ | Description | This phase |
|-----|-------------|------------|
| CDP-DEEP-01 | MITM proxy approach | ✅ Built (Phase 39 chose MITM; this builds it) |
| CDP-DEEP-02 | Executable PLAN | ✅ PLAN.md follows |
| CDP-PROXY-01 | Pass-through proxy implementing WebSocketable | ✅ `internal/cdpproxy/proxy.go` |
| CDP-PROXY-02 | Traffic logging to ring buffer | ✅ `Proxy.Traffic()` |
| CDP-PROXY-03 | Flag-gated via `--cdp-proxy` (default OFF) | ✅ wired in cmd.go + config.go |
| CDP-PROXY-04 | Wired into browser launch path | ✅ `launchBrowser()` via `WithCDPWrapper` |
| CDP-PROXY-05 | No page-observable behavior change (pass-through) | Asserted by verification |

## Success criteria (what must be TRUE)

1. `go build ./...` succeeds with the proxy wired but default-off.
2. A browser session with `--cdp-proxy` navigates a page successfully (no breakage).
3. With `--cdp-proxy`, the proxy's `Traffic()` contains expected CDP messages (Page.frameNavigated, etc.) after navigation.
4. Without `--cdp-proxy`, behavior is identical to pre-phase-40 (no regression).
5. The ring buffer enforces its capacity (doesn't leak memory).

## Risk notes

- The `WithCDPWrapper` API in godoll must exist and accept a `func(cdp.WebSocketable) cdp.WebSocketable` — confirmed present in godoll `browser.BrowserOptions`.
- The proxy adds ~0.1ms per-message latency (one extra localhost hop + logging).
- No filtering in v1 — all messages pass through unchanged. Filtering lands in Phases 41-42.
