---
author: architect
responsible: architect
phase: 35
phase_type: implementation
hard_bar: true
security_relevant: true
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (LEDGER-01, LEDGER-02)
  - .planning/ROADMAP.md (Phase 35)
  - .planning/phases/34/SUMMARY.md (toolchain fixed — builds on go1.26.4)
---

# Phase 35: Plugin-Path CDP-Ledger Closure — CONTEXT

## What this phase delivers

Close the v1.7 CDP-ledger coverage hole: CDP domains enabled via the plugin lifecycle path are recorded in the per-session ledger (identically to the main navigation path), and the offline detection harness asserts the previously-uncovered path so the regression cannot reappear unseen.

## The gap (root cause)

In `internal/plugin/lifecycle.go`, `BindLifecycle` calls:

```go
_ = proto.DOMEnable{}.Call(page)
```

This enables the CDP **DOM** domain so Chrome emits `DOMChildNodeInserted` events to the plugin's `onDOMNodeInserted` hook. However, this enable call is **not recorded** in the per-session CDP domain ledger (`ctx.cdpDomains`). The ledger's sole mutator — `recordCDPDomainLocked()` — is called in 3 places in `types/context.go`:

| Enable-point | Domain | Location |
|---|---|---|
| Console capture | Runtime | context.go:724 |
| Request capture | Network | context.go:742 |
| Route (lazy interceptor) | Fetch | context.go:865 |

None of these call sites are on the plugin lifecycle path. The DOM domain enabled by `BindLifecycle` escapes the ledger entirely. The detection harness (`TestCDPFootprintBaseline`) only tests the main navigation path — it never exercises a plugin, so it cannot catch the gap.

## Success criteria (from REQUIREMENTS)

### LEDGER-01 — Plugin-path domains recorded
- CDP domains enabled via the **plugin lifecycle path** are recorded in the per-session CDP-domain ledger, identically to the main navigation path — no enabled domain escapes the inventory.
- Specifically: `proto.DOMEnable` called in `BindLifecycle` writes `"DOM"` into `ctx.cdpDomains`.

### LEDGER-02 — Harness asserts the plugin path
- The offline detection harness has a test that exercises a plugin enabling the DOM domain and asserts the ledger reflects it (red before the fix, runs green after).
- The test is a positive-control-style subtest in `TestCDPFootprintPositiveControls` (matching the existing pattern for Runtime/Network/Fetch).

## What the work entails (4 files)

1. **`types/context.go`**: Add `CDPDomainDOM = "DOM"` constant. Export `recordCDPDomainLocked` → `RecordCDPDomain` (public method). Update the 3 internal call sites.
2. **`internal/plugin/lifecycle.go`**: Change `BindLifecycle(ctx context.Context, page *rod.Page)` → add a `*types.Context` parameter (or a `func(string)` recorder callback) so DOMEnable is followed by a ledger write. Add `import "github.com/agenthands/rod-cli/types"`.
3. **`actions/plugin.go`**: Update the `BindLifecycle` call site to pass the Context or recorder.
4. **`types/cdp_footprint_test.go`**: Add a positive-control subtest that loads a plugin, calls `BindLifecycle`, and asserts `GetEnabledCDPDomains()["DOM"]` is true.

## Key centerspiece symbols (SMTC-grounding)

| Symbol | File | Role |
|---|---|---|
| `recordCDPDomainLocked` / `RecordCDPDomain` | `types/context.go:774` | Sole ledger mutator — needs exporting |
| `BindLifecycle` | `internal/plugin/lifecycle.go:20` | Plugin path that enables DOM without recording |
| `CDPDomainRuntime/Network/Fetch` constants | `types/context.go:263-267` | Existing domain keys — `CDPDomainDOM` joins them |
| `PluginLoad` | `actions/plugin.go:12` | Caller of BindLifecycle — needs to pass recorder |
| `TestCDPFootprintPositiveControls` | `types/cdp_footprint_test.go:93` | Harness — needs a new DOM subtest |

## Security relevance

This is a ledger-integrity fix — the v1.7 per-session CDP domain inventory is incomplete, so a user running `rod-cli footprint` (or the harness) cannot see that the DOM domain is enabled by plugin-loaded code. Closing the gap makes the footprint inventory complete. `security_relevant: true`.

## Dependencies

- **Depends on**: Phase 34 (go1.26.4 toolchain — builds and tests run on the fixed toolchain).
- **Independent of**: Phase 36 (font spoofing).
- **No upstream/downstream coupling**: the 3 internal `recordCDPDomainLocked` call sites are all in `types/context.go` and don't touch the plugin path.

## Out of scope

- Adding CDP domain recording for any domain other than DOM (rod enables Network/Page/Target automatically; those are structural, not footprint-adding).
- Recording domain enables on other plugin entry points beyond `BindLifecycle`.
- Adding domain-level granularity to the footprint CLI command (`rod-cli footprint` doesn't exist yet — it's the v2/CDP-DEEP candidate).
