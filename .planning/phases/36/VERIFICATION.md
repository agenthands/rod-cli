---
author: architect
responsible: architect
phase: 36
status: verified
verdict: passed
parent_artifacts:
  - .planning/phases/36/PLAN.md
  - .planning/phases/36/SUMMARY.md
---

# Phase 36: Real Font Spoofing — VERIFICATION

## Verdict: ✅ PASSED

FONT-01, FONT-02, and FONT-03 are all met. The godoll font-injector no-op is replaced with a real injector.

## Evidence

| Claim | Result |
|-------|--------|
| FONT-01: font-spoof ON changes font hash | baseline=399611779 ≠ spoofed=-11579784 ✅ |
| FONT-02: stable within session | re-read identical on same session ✅ |
| FONT-02: OFF restores baseline | restored=399611779 matches baseline ✅ |
| FONT-03: harness asserts all legs | `TestFontSpoof` PASS ✅ |

## Notes

- Godoll changes committed separately in `../godoll` repo (commit `1d90494`).
- The injector adds a 1-7px deterministic width offset for spoofed fonts derived from the per-session noise seed.
- The `replace` directive in go.mod picks up local godoll changes immediately.
