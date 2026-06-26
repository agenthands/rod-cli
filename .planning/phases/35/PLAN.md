---
author: architect
responsible: architect
phase: 35
status: ready_for_execution
parent_artifacts:
  - .planning/phases/35/CONTEXT.md
  - .planning/REQUIREMENTS.md (LEDGER-01, LEDGER-02)
---

# Phase 35: Plugin-Path CDP-Ledger Closure — PLAN

## §1. types/context.go — Export recorder + add DOM constant

**1.1** Add `CDPDomainDOM` constant:
```go
const (
    CDPDomainRuntime = "Runtime"
    CDPDomainNetwork = "Network"
    CDPDomainFetch   = "Fetch"
    CDPDomainDOM     = "DOM"     // <-- ADD
)
```
Location: after line 267 (after `CDPDomainFetch`).

**1.2** Rename `recordCDPDomainLocked` → `RecordCDPDomain` (export):
```go
// RecordCDPDomain marks a footprint-adding CDP domain as enabled for this
// session in the per-session domain-enable ledger (Phase 30 D-04 instrumentation).
// ...
func (ctx *Context) RecordCDPDomain(domain string) {
```
Location: line 774. Update the 3 internal call sites (lines 724, 742, 865) to use the new name.

## §2. internal/plugin/lifecycle.go — Record DOM after enable

**2.1** Add import:
```go
import (
    "context"
    "github.com/agenthands/rod-cli/types"
    "github.com/dop251/goja"
    "github.com/go-rod/rod"
    "github.com/go-rod/rod/lib/proto"
)
```

**2.2** Change `BindLifecycle` signature to accept a recorder callback:
```go
func (e *PluginEngine) BindLifecycle(ctx context.Context, page *rod.Page, recordCdpDomain func(string)) {
```

**2.3** After `proto.DOMEnable{}.Call(page)`, add the ledger record:
```go
_ = proto.DOMEnable{}.Call(page)
if recordCdpDomain != nil {
    recordCdpDomain(types.CDPDomainDOM)
}
```

(Note: use `context.Context` for the Go context — the `*types.Context` import is only needed for the `CDPDomainDOM` constant.)

## §3. actions/plugin.go — Pass recorder to BindLifecycle

**3.1** Update the `BindLifecycle` call site (line 27):
```go
engine.BindLifecycle(context.Background(), page, ctx.RecordCDPDomain)
```

## §4. types/cdp_footprint_test.go — Add DOM positive-control test

**4.1** Add a new subtest to `TestCDPFootprintPositiveControls`:
```go
t.Run("plugin lifecycle enables DOM", func(t *testing.T) {
    ctx := newCDPContext(t, Config{})
    if _, err := ctx.EnsurePage(); err != nil {
        t.Fatalf("EnsurePage: %v", err)
    }
    // Simulate the plugin lifecycle path: BindLifecycle enables the DOM domain
    // so DOMChildNodeInserted events fire. The ledger must record it.
    engine := ctx.GetPluginEngine()
    engine.Init()
    engine.BindLifecycle(context.Background(), ctx.page, ctx.RecordCDPDomain)
    enabled := ctx.GetEnabledCDPDomains()
    if !enabled[CDPDomainDOM] {
        t.Errorf("BindLifecycle should record DOM, ledger=%v", enabled)
    }
    // DOM is a footprint-adding domain — it must NOT enable Runtime/Network/Fetch.
    if enabled[CDPDomainRuntime] || enabled[CDPDomainNetwork] || enabled[CDPDomainFetch] {
        t.Errorf("BindLifecycle must not enable Runtime/Network/Fetch, ledger=%v", enabled)
    }
})
```

## §5. Verification gates

| Gate | Command | Expected |
|------|---------|----------|
| Build | `go build ./...` | exit 0 |
| CDP footprint tests | `go test ./types/ -run TestCDPFootprint -count=1 -v` | all pass (including new DOM subtest) |
| Full test suite | `go test ./... -count=1` | pass (module + types) |
| Compile (no import cycle) | `go vet ./...` | exit 0 |

## §6. Edge cases / risks

- **Import cycle**: `internal/plugin` imports `types`, but `types` does NOT import `internal/plugin`. Verify with `go vet ./...`.
- **PluginEngine not initialized**: The test calls `engine.Init()` before `BindLifecycle`. In production, `LoadScript` calls `Init` first. The test must match.
- **`recordCdpDomain` is nil**: BindLifecycle checks for nil before calling — safe when called from code that doesn't pass a recorder (backward compat).
- **Test requires browser**: The CDP footprint tests use `newCDPContext` which launches a headless browser. They skip if no browser is available (unless `REQUIRE_BROWSER=1`). CI must have Chromium installed.

## §7. Acceptance criteria (engineer handshake)

The engineer accepts this plan if:
1. The `CDPDomainDOM` constant is added without changing existing constants.
2. `RecordCDPDomain` is the only exported API change — no other public surface changes.
3. The recorder callback pattern avoids a tight coupling between `plugin` and `types`.
4. The harness test follows the existing positive-control pattern.

The engineer should refuse if:
- The plan creates an import cycle.
- The required test can't exercise the plugin path in the existing harness setup.
- A simpler approach exists that avoids the callback indirection.
