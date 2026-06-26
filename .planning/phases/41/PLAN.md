---
author: architect
responsible: architect
phase: 41
status: planned
parent_artifacts:
  - .planning/phases/41/CONTEXT.md
  - .planning/phases/40/SUMMARY.md
---

# Phase 41: Runtime Domain Normalization — PLAN

## Centerpiece symbol

`internal/cdpproxy.Proxy.Read()` — extended to normalize `Runtime.getProperties` responses.

## Plan §1: JSON-RPC message detection (`internal/cdpproxy/filters.go`)

**Deliverable:** A new file `internal/cdpproxy/filters.go` with:
- `normalizeCDPResponse(data []byte) []byte` — main entry point
- Detects JSON-RPC responses (has `"result"` key, not `"method"`)
- Detects `Runtime.getProperties` responses by checking `result.result` array
- Passes through non-matching messages unchanged

**Implementation:**
```go
func normalizeCDPResponse(data []byte) []byte {
    var msg map[string]json.RawMessage
    if err := json.Unmarshal(data, &msg); err != nil {
        return data // can't parse — pass through (fail-safe)
    }
    // Only responses have "result" and no "method"
    if _, isResult := msg["result"]; !isResult {
        return data
    }
    if _, isEvent := msg["method"]; isEvent {
        return data // events have "method"
    }
    // Check for Runtime.getProperties response shape
    return normalizeGetProperties(data, msg)
}
```

## Plan §2: Accessor property value stripping (`internal/cdpproxy/filters.go`)

**Deliverable:** `normalizeGetProperties()` function:
1. Parse `msg["result"]` as the response result
2. Check for `.result.result` array (the property descriptors)
3. For each descriptor with a `"get"` key, delete `"value"`
4. Re-serialize

```go
func normalizeGetProperties(raw []byte, msg map[string]json.RawMessage) []byte {
    var result struct {
        Result []map[string]json.RawMessage `json:"result"`
    }
    if err := json.Unmarshal(msg["result"], &result); err != nil {
        return raw
    }
    modified := false
    for _, prop := range result.Result {
        if _, hasGet := prop["get"]; hasGet {
            delete(prop, "value")
            modified = true
        }
    }
    if !modified {
        return raw
    }
    // Re-serialize
    newResult, _ := json.Marshal(result)
    msg["result"] = newResult
    out, _ := json.Marshal(msg)
    return out
}
```

## Plan §3: Wire into Proxy.Read() (`internal/cdpproxy/proxy.go`)

**Deliverable:** Modify `Proxy.Read()` to call `normalizeCDPResponse()` before logging.
- Apply normalization to the raw bytes after reading from Chrome
- Log the NORMALIZED message (not the raw)
- Return the normalized bytes

```go
func (p *Proxy) Read() ([]byte, error) {
    data, err := p.inner.Read()
    if err != nil {
        return data, err
    }
    data = normalizeCDPResponse(data)  // NEW
    p.logMessage("recv", data)
    return data, nil
}
```

## Plan §4: Tests (`internal/cdpproxy/filters_test.go`)

**Deliverable:** Unit tests for `normalizeCDPResponse`:
1. `TestNormalizeGetProperties_StripsAccessorValue` — property with `get` has `value` stripped
2. `TestNormalizeGetProperties_PreservesDataProperty` — property without `get` keeps `value`
3. `TestNormalizeGetProperties_PassThroughNonGetProperties` — non-Runtime messages unchanged
4. `TestNormalizeGetProperties_InvalidJSON` — garbage input returns unchanged
5. `TestNormalizeGetProperties_EventPassthrough` — events pass through unchanged

## Plan §5: Build & regression gate

**Deliverable:** `go build ./...` + `go test ./...` pass.

## Verification criteria

1. Unit tests demonstrate normalization works
2. No breaking changes to proxy behavior
3. Pass-through for non-Runtime messages
4. Fail-safe for unparseable messages
