---
author: architect
responsible: architect
phase: 39
status: executed
parent_artifacts:
  - .planning/phases/39/CONTEXT.md
  - .planning/REQUIREMENTS.md (CDP-DEEP-01, CDP-DEEP-02, CDP-DEEP-03)
---

# Phase 39: CDP-DEEP-01 Research & Design — PLAN for Execution

## Chosen approach: MITM WebSocket CDP proxy

A local, in-process WebSocket proxy sits between go-rod's `cdp.Client` and Chrome's debugging WebSocket, implementing go-rod's `WebSocketable` interface. The proxy is transparent by default (pass-through) and becomes the foundation for CDP traffic filtering, normalization, and observability.

## Phased execution (for a follow-on build milestone)

### Phase 1: Core proxy (pass-through, logging)

**Files:** `internal/cdpproxy/proxy.go`

Implement a `Proxy` struct that implements `cdp.WebSocketable`:

```go
type Proxy struct {
    targetURL string       // Chrome's debugging WebSocket URL
    conn      *websocket.Conn
    log       []CDPMessage // ring buffer of recent messages
}

func (p *Proxy) Send(data []byte) error    // forward to Chrome
func (p *Proxy) Read() ([]byte, error)     // read from Chrome
```

- On `NewProxy(targetURL)`, dial Chrome's WebSocket
- `Send` writes directly to Chrome (no filtering in v1)
- `Read` reads from Chrome, logs the message, returns it unchanged
- `GetTraffic()` returns the logged messages for diagnostic commands

**Wire into rod-cli:** Modify `types/context.go` — in `launchBrowser`, after Chrome launches, wrap the CDP WebSocket URL through the proxy before passing to go-rod. Gated behind a flag (`--cdp-proxy`, default OFF in v1).

**Verification:** A new test starts a browser through the proxy, navigates a page, and asserts the proxy's traffic log contains expected CDP messages (Page.frameNavigated, etc.) — proving the proxy can see traffic without breaking navigation. No page-observable behavior changes (the proxy is pass-through).

### Phase 2: Runtime domain normalization

**Files:** `internal/cdpproxy/filters.go`

When Runtime is enabled (opt-in via `--console-capture` or plugins), normalize Runtime domain responses to suppress property-getter triggering:

- Intercept `Runtime.getProperties` responses
- Strip or replace `value` descriptors that are accessor properties (getters)
- This prevents the `console.debug` stack-getter tell even when Runtime is active

**Verification:** The `cdpTell` probe returns `"no-signal"` even when `--console-capture` is enabled.

### Phase 3: Timing jitter + CDP footprint command

Add configurable timing jitter to CDP command dispatch (random 0-N ms delay) to break characteristic automation patterns. Add a `rod-cli cdp-traffic` command that reads the proxy's traffic log for diagnostics.

## Key design decisions

1. **In-process, not a separate binary.** The proxy lives in rod-cli's process. No additional port, no separate lifecycle.
2. **Go's `gorilla/websocket` or stdlib `nhooyr.io/websocket`** for the proxy's WebSocket client. (go-rod itself uses `github.com/gorilla/websocket` internally.)
3. **Flag-gated:** `--cdp-proxy` (default OFF initially, becomes default ON after bake-in).
4. **Bypass flag:** `--no-cdp-proxy` to skip the proxy if it causes issues — ensures zero-risk deployment.
5. **Ring buffer for traffic log** — bounded memory, configurable size.

## Risk/limitation notes

- Adds 1 extra network hop (proxy → Chrome) — ~0.1ms latency per message on localhost.
- Runtime normalization is a heuristic — Chrome's Runtime protocol is complex; some edge cases may escape normalization.
- The proxy cannot hide the WebSocket port from OS-level inspection (accepted-visible ceiling).
- If go-rod changes its WebSocketable interface, the proxy API surface needs updating (low risk — the interface has been stable since rod v0.1).
