---
author: architect
responsible: architect
phase: 42
status: planned
parent_artifacts:
  - .planning/phases/42/CONTEXT.md
  - .planning/phases/40/SUMMARY.md
---

# Phase 42: Timing Jitter + `cdp-traffic` Command — PLAN

## Plan §1: Proxy jitter (`internal/cdpproxy/proxy.go`)

Add `jitterMaxMs int` field to `Proxy`. In `Send()`, if `jitterMaxMs > 0`, sleep `rand.Intn(jitterMaxMs)` ms before forwarding.

New constructor parameter: `New(inner, cap, jitterMaxMs)`

## Plan §2: Config surface (`types/config.go`, `cmd.go`)

- `StealthConfig.CDPJitterMs *int` (YAML: `cdpJitterMs`, default nil→0)
- `--cdp-jitter-ms` int flag (default 0)
- `--no-cdp-proxy` bool flag (default false)
- Wire `jitterMaxMs` into `cdpproxy.New()` call in `launchBrowser()`

## Plan §3: Bypass flag (`types/context.go`)

Add `StealthConfig.NoCDPProxy *bool`. In `launchBrowser()`, skip proxy creation if true.

## Plan §4: `cdp-traffic` action (`actions/cdp_traffic.go`)

```go
func CDPTraffic(ctx *types.Context, jsonOut bool) (string, error) {
    if ctx.GetCDPProxy() == nil {
        return "", errors.New("CDP proxy not enabled (use --cdp-proxy)")
    }
    msgs := ctx.GetCDPProxy().Traffic()
    // format output
}
```

Need to add `GetCDPProxy()` accessor to `Context`.

## Plan §5: CLI registration (`cmd.go`, `daemon/daemon.go`)

- `cmd.go`: Add `cdp-traffic` subcommand with `--json` flag
- `daemon/daemon.go`: Add `case "cdp-traffic"` dispatching to `actions.CDPTraffic`

## Plan §6: Tests

- `internal/cdpproxy/proxy_test.go`: Test jitter delay range, test `Traffic()` ring buffer behavior, test bypass
- Build + regression gate
