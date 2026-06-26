---
author: architect
responsible: architect
phase: 36
status: executed
parent_artifacts:
  - .planning/phases/36/CONTEXT.md
  - .planning/phases/36/PLAN.md
---

# Phase 36: Real Font Spoofing — SUMMARY

## Execution result: ✅ COMPLETE

The godoll `scriptMockFonts` no-op is replaced with a real injector. The rod-cli harness asserts the fix is observable, stable, and restoreable.

## Changes

### godoll (separate repo: `../godoll`)

| File | Change |
|------|--------|
| `stealth/fingerprint_bridge.go` | `scriptMockFonts` replaced: adds per-session deterministic 1-7px width offset for spoofed fonts; caches results; non-spoofed fonts return original measurement |
| `stealth/evasion.go` | Passes `em.noiseSeed` to `scriptMockFonts` for session-stable offsets |
| `stealth/export_test.go` | Updated `ExportScriptMockFonts` to pass seed=0 (test compat) |

### rod-cli

| File | Change |
|------|--------|
| `tests/detection_test.go` | `TestFontSpoof` harness: baseline/differs-when-on/stability/restore; `evalFontHash` helper; `fontProbeJS` constant |

## Verification gates

| Gate | Result |
|------|--------|
| `go build ./...` (both repos) | ✅ PASS |
| FONT-01: differs-when-on | ✅ baseline=399611779 ≠ spoofed=-11579784 |
| FONT-02: stable-across-re-reads | ✅ re-read identical on same session |
| FONT-02: restored-when-off | ✅ off-again=399611779 matches baseline |
| FONT-03: harness-asserted | ✅ all 3 legs pass in `TestFontSpoof` |

## Requirement traceability

| REQ-ID | Status | Evidence |
|--------|--------|----------|
| FONT-01 | ✅ | Font hash changes when font-spoof is ON (was no-op — now produces different hash) |
| FONT-02 | ✅ | Same-session re-reads identical; OFF restores baseline; coherent with profile's OS (already enforced by godoll's `FPWithOS`) |
| FONT-03 | ✅ | `TestFontSpoof` in the offline detection harness asserts all 3 legs |
