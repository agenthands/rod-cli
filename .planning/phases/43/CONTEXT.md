---
author: architect
responsible: architect
phase: 43
phase_type: implementation
hard_bar: true
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (PROXY-01, PROXY-02)
  - .planning/phases/42/SUMMARY.md (proxy built)
---

# Phase 43: Proxy Integration Test — CONTEXT

## Goal

Add a live-browser integration test that starts a session with `--cdp-proxy`, navigates a page, and asserts:
1. The proxy's `Traffic()` ring buffer contains expected CDP messages (PROXY-01)
2. With `--console-capture`, the `cdpTell` probe returns `"no-signal"` (PROXY-02 — normalization working)

## Test design

Two test functions mirroring the existing `cdp_footprint_test.go` pattern:

### Test 1: `TestProxyTraffic` (PROXY-01)
- Launches a headless browser with `--cdp-proxy` via the binary (`runCli`)
- Navigates to a loopback fixture page
- Asserts `cdp-traffic --json` returns a non-empty JSON array containing expected CDP message structures

### Test 2: `TestProxyCdpTell` (PROXY-02)
- Launches a headless browser with `--cdp-proxy --console-capture`
- Navigates to the detect fixture page
- Reads `window.__detect.cdpTell` via `eval`
- Asserts `"no-signal"`

## Files

- `tests/proxy_integration_test.go` — new test file
- Uses existing `runCli` helper and loopback fixture pattern

## Success criteria

1. `go test -run TestProxyTraffic ./tests/` passes
2. `go test -run TestProxyCdpTell ./tests/` passes
3. Tests work with headless Chrome (CI-compatible)
