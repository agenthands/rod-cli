---
author: architect
responsible: architect
phase: 41
phase_type: implementation
hard_bar: false
security_relevant: true
design_fork: false
status: locked
parent_artifacts:
  - .planning/phases/39/CONTEXT.md (CDP-DEEP-01 research)
  - .planning/phases/39/CDP-DEEP-01-PLAN.md (MITM approach chosen)
  - .planning/phases/40/SUMMARY.md (proxy core built)
---

# Phase 41: Runtime Domain Normalization — CONTEXT

## Goal

When Runtime is enabled (opt-in via `--console-capture` or plugins), normalize `Runtime.getProperties` CDP responses to suppress property-getter triggering. The proxy intercepts CDP responses from Chrome, detects `Runtime.getProperties` results, and strips `value` fields from accessor properties (where `get` is present) — preventing the `console.debug` stack-getter tell even when Runtime IS active.

## Background: the `cdpTell` probe

When Chrome's `Runtime` domain is enabled, CDP `Runtime.getProperties` serializes property descriptors including accessor (getter) properties. When Chrome sends `value` alongside `get`, that `value` was obtained by **calling the getter** — triggering any side effects. 

The classic tell: `console.debug(e)` creates an Error object with a `stack` getter. When Runtime is enabled, the CDP protocol calls `Error.stack`'s getter to serialize the `value`, and a detector can observe this call:

```
Runtime.getProperties response:
  { name: "stack",
    value: { type: "string", value: "Error\n    at foo (..." },  ← getter was called!
    get: { type: "function", ... }  ← this IS an accessor
  }
```

**Current v1.7/v1.8 state:** Runtime.enable is NOT sent on the baseline path (a plain `goto`). The `cdpTell` probe returns `"no-signal"`. However, features like `--console-capture` and plugins that enable Runtime RE-OPEN this signal.

**Phase 40 state:** The CDP proxy is built and wired via `--cdp-proxy` (default OFF). All messages currently pass through unchanged.

## What Phase 41 builds

A **filter layer** in the CDP proxy that normalizes `Runtime.getProperties` responses:

1. In `Read()`, after reading bytes from Chrome, detect JSON-RPC responses to `Runtime.getProperties`
2. For each property descriptor with `get` set (accessor), strip the `value` field
3. Re-serialize and return the normalized response

### Normalization logic

```go
// For each property in result.result[]:
if property["get"] exists:
    delete property["value"]  // strip the side-effect-triggered value
```

This is **defense-in-depth**: when Runtime IS enabled (opt-in), the proxy sanitizes the responses so getter-triggered values don't leak back through CDP.

### CDP message format

JSON-RPC 2.0 over WebSocket:
- Request: `{"id": N, "method": "Domain.method", "params": {...}}`
- Response: `{"id": N, "result": {...}}` or `{"id": N, "error": {...}}`

`Runtime.getProperties` response shape:
```json
{"id": 5, "result": {"result": [
  {"name": "prop", "value": {...}, "get": {...}, ...},
  ...
]}}
```

## Requirements

| REQ | Description |
|-----|-------------|
| CDP-NORM-01 | Intercept `Runtime.getProperties` responses in the proxy's `Read()` path |
| CDP-NORM-02 | Strip `value` from accessor properties (where `get` is present) |
| CDP-NORM-03 | Non-`Runtime.getProperties` messages pass through unchanged |
| CDP-NORM-04 | Normalization does not break console-capture (messages still captured) |
| CDP-NORM-05 | With `--cdp-proxy` but WITHOUT Runtime enabled, no normalization needed (but harmless if applied) |

## Success criteria

1. `go build ./...` succeeds.
2. With `--cdp-proxy`, `Runtime.getProperties` responses have `value` stripped from accessor properties.
3. Non-`Runtime.getProperties` messages pass through unchanged.
4. Existing tests pass (no regression).

## Risk notes

- JSON parse/re-serialize per message adds overhead. Acceptable because gated behind `--cdp-proxy` (default OFF).
- The normalization is a heuristic — Chrome's CDP protocol may add fields or change structure. The filter is best-effort; errors in parsing fall through to pass-through (fail-safe).
- Cannot fully prevent getter-triggering at the CDP protocol level (Chrome calls the getter during serialization). What we CAN prevent is the **result** of that getter call from being observable by go-rod/rod-cli.
