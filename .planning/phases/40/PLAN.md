---
author: architect
responsible: architect
phase: 40
status: executed
parent_artifacts:
  - .planning/phases/40/CONTEXT.md
  - .planning/phases/39/CDP-DEEP-01-PLAN.md
---

# Phase 40: Core CDP WebSocket Proxy — PLAN

## Centerpiece symbol

`internal/cdpproxy.Proxy` — implements `cdp.WebSocketable`, wraps the real Chrome WebSocket, logs all traffic.

## Plan §1: Proxy core (`internal/cdpproxy/proxy.go`)

**Deliverable:** A `Proxy` struct that:
- Implements `cdp.WebSocketable` (Send, Read)
- Wraps an inner `cdp.WebSocketable` (the real Chrome connection)
- Logs every message to a ring buffer with direction ("send"/"recv")
- Exposes `Traffic()` for diagnostics

**Implementation details:**
- `New(inner, cap)` constructor, default cap=256
- `Send(data)`: lock, append to ring buffer, forward to inner.Send
- `Read()`: read from inner, lock, append to ring buffer, return data
- Ring buffer: when full (len >= cap), drop oldest entry
- `Traffic()`: return a copy (thread-safe)

**Compile-time assertion:** `var _ cdp.WebSocketable = (*Proxy)(nil)`

## Plan §2: Config surface (`types/config.go`)

**Deliverable:** `StealthConfig.CDPProxy *bool` field
- YAML key: `cdpProxy`
- JSON key: `cdpProxy`
- `*bool` for tri-state (nil = use default)

## Plan §3: CLI flag (`cmd.go`)

**Deliverable:** `--cdp-proxy` bool flag
- Default: false (proxy OFF)
- Usage: "Enable the in-process CDP WebSocket proxy for traffic logging and normalization (default off)"

## Plan §4: Browser launch wiring (`types/context.go`)

**Deliverable:** Proxy wired into `launchBrowser()`:
1. Before browser creation, check `boolVal(cfg.Stealth.CDPProxy, false)`
2. If true, call `opts.WithCDPWrapper(func(inner cdp.WebSocketable) cdp.WebSocketable { ... })`
3. The wrapper creates `cdpproxy.New(inner, 1024)` and stores the instance
4. Return the `*cdpproxy.Proxy` from `launchBrowser()`
5. Store on `Context.cdpProxy` in `initial()`

## Plan §5: Context storage (`types/context.go`)

**Deliverable:** `Context.cdpProxy *cdpproxy.Proxy` field
- Set in `initial()` from `launchBrowser()` return value
- Available for Phase 42's `cdp-traffic` command

## Plan §6: Regression gate

**Deliverable:** `go build ./...` and `go test ./...` pass with no proxy-related regressions.

## Verification plan

1. Build check: `go build ./...` succeeds
2. Test check: existing tests pass (no regression)
3. Functional check: with `--cdp-proxy`, browser navigates and proxy traffic log is non-empty
4. Default-off check: without `--cdp-proxy`, behavior is identical to pre-phase-40
