---
author: architect
responsible: architect
phase: 45-46
phase_type: implementation
hard_bar: true
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (FONT-04..FONT-07)
---

# Phase 45-46: Real Font Spoofing + Harness Gate — CONTEXT/PLAN

## Discovery: godoll font injector is ALREADY real

godoll commit `1d90494` ("scriptMockFonts: replace no-op with real font-spoof injector") added actual canvas `measureText` offset injection. The injector path in rod-cli is already wired: `SetDimensionOptions({Fonts: true})` → `applyFingerprintDimensions` → `scriptMockFonts(fp.Fonts, seed)`.

This means the "font-spoof no-op" is ALREADY FIXED. Phase 45 shifts to: **prove it with a harness test.**

## Phase 45: Harness test (FONT-04..07)

New test in `tests/` or `types/`:

1. **FONT-04 (differs when on):** Navigate to detect fixture with `--font-spoof=false`, read `window.__detect.fonts`. Navigate with `--font-spoof` (default ON), read `fonts`. Assert values differ.
2. **FONT-05 (stable):** In a `--font-spoof` session, read `fonts` twice. Assert identical.
3. **FONT-06 (restores when off):** `--font-spoof=false` reads match the host OS baseline.
4. **FONT-07 (harness):** The test itself IS the harness gate.

## Plan

1. Write `TestFontSpoof` in `tests/proxy_integration_test.go` (reuse existing file)
2. Uses `runCli` + detect fixture page via eval
3. Assert on/off difference + stability
4. Update `internal/detect/detect.js` comment: remove "observable no-op" language

## Success criteria

1. `go test -run TestFontSpoof ./tests/` passes
2. `go build ./...` passes
3. detect.js comment updated
