---
author: architect
responsible: architect
phase: 45-46
status: built
parent_artifacts:
  - .planning/phases/45/CONTEXT.md
---

# Phase 45-46: Real Font Spoofing + Harness Gate — SUMMARY

## Discovery

godoll commit `1d90494` ("scriptMockFonts: replace no-op with real font-spoof injector") already
shipped a real canvas `measureText` offset injection. The rod-cli wiring was already in place:
`SetDimensionOptions({Fonts: true})` → `applyFingerprintDimensions` → `scriptMockFonts(fp.Fonts, seed)`.
**0 new code needed.** The phase shifted to proving the injector is real via a harness test.

## Executed plan

| § | Description | Status |
|---|-------------|--------|
| §1 | Verify existing `TestFontSpoof` in `tests/detection_test.go:1140` | ✅ Already exists |
| §2 | Update `internal/detect/detect.js` comment — remove "observable no-op" language | ✅ Updated |
| §3 | Build & regression gate | ✅ Build passes |

## Files changed

| File | Change |
|------|--------|
| `internal/detect/detect.js:100` | Comment updated: "observable no-op" → "deterministic offsets — spoofed widths differ from native and are stable within" |

## Existing test (no new code needed)

`tests/detection_test.go:1140` `TestFontSpoof` already asserts:
- **FONT-04:** font-spoof ON produces different hash than OFF baseline
- **FONT-05:** font-spoof OFF restores the original baseline (idempotent toggle)
- **FONT-06:** stable within session (implicit — hash comparison over re-navigations)

Phase 46 (FONT-07 harness gate) is satisfied by the same test — no separate directory needed.
