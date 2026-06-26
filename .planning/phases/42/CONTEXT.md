---
author: architect
responsible: architect
phase: 42
phase_type: implementation
hard_bar: false
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/phases/41/SUMMARY.md (proxy core + normalization built)
  - .planning/phases/40/SUMMARY.md (proxy core built)
---

# Phase 42: Timing Jitter + `cdp-traffic` Command — CONTEXT

## Goal

Add two capstone features to the CDP proxy:
1. **Timing jitter:** Configurable random delay before CDP command dispatch (`Send()` path) to break characteristic automation timing patterns
2. **`cdp-traffic` diagnostic command:** CLI command that reads the proxy's ring buffer and outputs logged CDP messages

Plus a `--no-cdp-proxy` bypass flag for zero-risk deployment.

## Requirements

| REQ | Description |
|-----|-------------|
| CDP-JITTER-01 | Configurable random 0-N ms delay before CDP `Send()` (default 0 = off) |
| CDP-TRAFFIC-01 | `rod-cli cdp-traffic` prints logged CDP messages in human-readable format |
| CDP-TRAFFIC-02 | `rod-cli cdp-traffic --json` emits machine-readable JSON |
| CDP-BYPASS-01 | `--no-cdp-proxy` flag bypasses the proxy even if `--cdp-proxy` is set |

## Success criteria

1. `go build ./...` succeeds.
2. With `--cdp-jitter-ms=10`, CDP `Send()` delays by 0-10ms per message.
3. `rod-cli cdp-traffic` prints traffic log with direction, message preview.
4. `rod-cli cdp-traffic --json` prints valid JSON array.
5. `--no-cdp-proxy` disables the proxy.
6. Existing tests pass (no regression).
