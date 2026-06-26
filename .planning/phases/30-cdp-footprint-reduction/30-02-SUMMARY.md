---
author: engineer
responsible: engineer
phase: 30-cdp-footprint-reduction
plan: 02
type: execute-summary
status: complete
requirements: [CDP-01, CDP-03]
commit: 3752a95
---

# Phase 30 Wave 2 — SUMMARY (measurement + deterministic baseline gate)

## Outcome
The CDP-footprint reduction from wave 1 is now MEASURED and asserted by a
deterministic, offline, regression-proof gate (CDP-01 "measured", CDP-03
"harness asserts the reduction baseline").

## What landed (per task)
- **Task 1 — instrumentation (D-04).** Added `ctx.cdpDomains map[string]bool`, a
  per-session ledger of footprint-adding CDP domains enabled beyond go-rod's
  structural requirements. Written via `recordCDPDomainLocked` (caller holds
  stateLock) at three set-points:
  - **Runtime** — the console-capture subscription branch in createPage.
  - **Network** — the request-capture subscription branch in createPage.
  - **Fetch** — `ensureInterceptorEnabled`, only after a confirmed `Enable()`.
  Read via `GetEnabledCDPDomains()` (locks, returns a copy). Domain-name string
  constants (`CDPDomainRuntime/Network/Fetch`) prevent set-point/assertion drift.
- **Task 2 — baseline assertion + positive control.** `types/cdp_footprint_test.go`
  (in-process, reads the accessor directly):
  - `TestCDPFootprintBaseline` — a plain session (no capture flags, no routes, no
    plugins) navigates an offline loopback fixture and records ZERO of
    {Runtime, Network, Fetch}. Asserts `len(ledger)==0` so any unexpected key
    fails, not just the three named.
  - `TestCDPFootprintPositiveControls` — proves the ledger is REAL (not vacuously
    empty): `--console-capture`→Runtime, `--request-capture`→Network, mock
    route→Fetch, each recording exactly its domain. The mock-route control also
    asserts Fetch is absent BEFORE the route and present after.
  Offline (httptest loopback, no egress), runs under plain `go test ./...`.

## Deviations / code-review fixes
`anvil-code-reviewer` verdict: **SHIP, no code defects** (lock-safety and
race-freedom traced clean — every `recordCDPDomainLocked` site runs under
stateLock; the async EachEvent goroutines never touch the ledger). One MAJOR was
a claim-scope/honesty fix, applied:
- **Honesty (MAJOR).** Docstrings now scope the gate precisely: it proves
  rod-cli's *instrumented enable-points* record zero — NOT a wire-level proof
  that no domain is enabled via a path bypassing the instrumentation. The
  wire-level confirmation is the separate WIRE-VERIFY (wave 1) + the
  informational live cdpTell probe (wave 3). This is the honest framing of a
  ledger-based gate (the D-04 design).
- **Robustness (MINOR).** `REQUIRE_BROWSER=1` turns the browserless `t.Skip` into
  `t.Fatal` so a misconfigured gating lane fails loud instead of passing green.
- **Precision (MINOR).** Field comment clarifies the semantic asymmetry:
  Runtime/Network recorded at subscription *request* (synchronous, conservative —
  can only over-report), Fetch at *confirmed* Enable; ledger is cumulative.

## Known limitation (documented, by design)
The deterministic gate verifies the ledger, not raw CDP wire traffic. Its strength
depends on the three set-points being the only rod-cli paths that enable those
domains; nothing mechanically enforces that a *future* un-instrumented enable path
gets recorded. The reviewer's stronger options (a `.smtc/analyzers` dominance spec,
or sniffing real CDP commands) are noted for v2; per CONTEXT D-04 the command-count
ledger is the chosen deterministic gate and D-05 keeps `__detect.cdpTell` as the
informational realism signal. This limitation is documented in cdp-footprint.md
(wave 3) under the honest ceiling.

## Tests
- `go build ./...`, `go vet ./...` clean.
- `TestCDPFootprintBaseline` + 3 positive controls PASS; `REQUIRE_BROWSER=1`
  variant PASS (browser present). Full `go test ./types/` PASS.
