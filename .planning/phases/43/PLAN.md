---
author: architect
responsible: architect
phase: 43
status: planned
parent_artifacts:
  - .planning/phases/43/CONTEXT.md
---

# Phase 43: Proxy Integration Test — PLAN

## Plan §1: `TestProxyTraffic` (PROXY-01)

New test in `tests/proxy_integration_test.go`:

1. Start a loopback HTTP fixture server
2. Run `runCli("--cdp-proxy", "goto", fixtureURL)`
3. Run `runCli("cdp-traffic", "--json")`
4. Parse JSON output, assert array is non-empty
5. Assert at least one message has `direction: "recv"` (Chrome responses)
6. Run `runCli("close")`

## Plan §2: `TestProxyCdpTell` (PROXY-02)

New test in `tests/proxy_integration_test.go`:

1. Start the detect fixture server (or loopback HTTP fixture)
2. Run `runCli("--cdp-proxy", "--console-capture", "goto", fixtureURL)`
3. Run `runCli("eval", "String(window.__detect ? window.__detect.cdpTell : 'no-detect')")`
4. If detect fixture not available, use a JS probe that checks `console.debug` behavior
5. Assert output contains `"no-signal"`
6. Run `runCli("close")`

## Plan §3: Build & regression gate

- `go build ./...` passes
- `go test -run TestProxy ./tests/` passes
- Existing tests pass (no regression)
