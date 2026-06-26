# Phase 30: CDP Footprint Reduction - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md - this log preserves the alternatives considered.

**Date:** 2026-06-26
**Phase:** 30-cdp-footprint-reduction
**Areas discussed:** Stealth-vs-functionality, Harness assertion (CDP-03), Configurability & default, Inventory doc & ceiling, Always-on exception

---

## Stealth-vs-functionality

| Option | Description | Selected |
| ------ | ----------- | -------- |
| Footprint follows features | Drive baseline to minimum; make console-capture, request-logging, interceptor lazy/opt-in; document footprint each feature adds. | ✓ |
| Keep features, document only | Leave always-on wiring; just inventory/document. Little actual reduction. | |
| Strict stealth mode | Keep features on by default + one mode that hard-disables them. More surface area. | |

**User's choice:** Footprint follows features
**Notes:** Grounded in code reality — `types/context.go createPage()` enables Runtime (console), Network, and Fetch (interceptor) unconditionally today; these are the bulk of the footprint.

---

## Harness assertion (CDP-03)

| Option | Description | Selected |
| ------ | ----------- | -------- |
| Instrumented command count | Count CDP commands/enabled domains per session; Tier-1 harness asserts baseline at/below threshold. Deterministic, offline. | ✓ |
| JS leak probe | Add Runtime.enable-leak detector to fixture page; assert clean. Less deterministic offline. | |
| Both | Command-count + JS leak probe. Strongest, most work. | |

**User's choice:** Instrumented command count
**Notes:** The harness already exposes an informational `window.__detect.cdpTell` probe (Phase 24); it stays as the realism signal while command-count is the deterministic gate.

---

## Configurability & default

| Option | Description | Selected |
| ------ | ----------- | -------- |
| Minimal by default, features opt-in | No dedicated --cdp-stealth flag; footprint-adding features become their own opt-in flags (CLI>profile>default). | ✓ |
| Dedicated toggle (precedence chain) | Add --cdp-stealth following v1.6 chain, default on. Overlaps with per-feature control. | |
| You decide | Defer flag shape to planning. | |

**User's choice:** Minimal by default, features opt-in

---

## Inventory doc & ceiling (CDP-02)

| Option | Description | Selected |
| ------ | ----------- | -------- |
| Table + honest ceiling | Per-command/domain table (command→when→status) + explicit ceiling (browser-patching/MITM → v2 CDP-DEEP-01), cross-linked to stealth-validation.md. | ✓ |
| Lean inventory | Just the command/status table, minimal prose. | |
| You decide | Defer depth to documentarian. | |

**User's choice:** Table + honest ceiling
**Notes:** `docs/cdp-footprint.md` already exists as a Phase 24 findings note — this is an expand/finalize, not a create.

---

## Always-on exception (follow-up)

| Option | Description | Selected |
| ------ | ----------- | -------- |
| Nothing always-on | Truly minimal baseline — no Runtime.enable on a plain session; all capture opt-in. | ✓ |
| Console errors always-on | Keep lightweight console-error capture always-on (still triggers Runtime.enable). | |
| You decide | Defer to planning. | |

**User's choice:** Nothing always-on
**Notes:** Cleanest stealth story; debugging is one flag away; also trims unsolicited output (token-efficiency).

---

## Claude's Discretion

- Exact opt-in flag names (console capture, request logging, interceptor) within CLI>profile>default.
- Instrumentation shape for the per-session domain/command ledger and threshold encoding.
- Single "needs-network" predicate vs per-feature gating for the lazy interceptor.
- Exact table columns in `cdp-footprint.md` beyond command→when→status.

## Deferred Ideas

- Full CDP obfuscation (browser-patching/MITM/alternate transport) → v2 CDP-DEEP-01.
- JS leak probe as a blocking gate (kept informational).
- Live-WAF pass/fail guarantee (Out of Scope; CDP-03 live measurement stays informational).
