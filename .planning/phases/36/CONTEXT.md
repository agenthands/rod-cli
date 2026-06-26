---
author: architect
responsible: architect
phase: 36
phase_type: implementation
hard_bar: true
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (FONT-01, FONT-02, FONT-03)
  - .planning/ROADMAP.md (Phase 36)
  - .planning/phases/34/SUMMARY.md (toolchain: go1.26.4)
---

# Phase 36: Real Font Spoofing — CONTEXT

## What this phase delivers

Replace the godoll `scriptMockFonts` no-op with a **real** font-injector so `--font-spoof` actually changes the page's detectable font availability. The fix lives in godoll (`../godoll/stealth/fingerprint_bridge.go`), and the offline detection harness asserts the change is observable, stable within a session, and restoreable when off.

## The gap (root cause)

In `godoll/stealth/fingerprint_bridge.go`, `scriptMockFonts` constructs a JS script that overrides `CanvasRenderingContext2D.prototype.measureText` — but the override is a **complete no-op**:

```javascript
CanvasRenderingContext2D.prototype.measureText = function(text) {
    const result = origMeasureText.call(this, text);
    for (const af of availableFonts) {
        if (font.includes(af)) { return result; }  // <-- same result!
    }
    for (const bf of baseFonts) {
        if (font.includes(bf)) { return result; }  // <-- same result!
    }
    return result;  // <-- same result!
};
```

Every code path returns the **identical original measurement**. Font detection probes (which measure text width to determine if a font is installed) see the host's real fonts regardless of the spoof toggle. This was documented as a v1.7 follow-up.

## What a real injector does

A real font injector overrides `measureText` so that:
- **Spoofed fonts** (those in the profile's font list) return a width **different from the fallback** — making probes think the font IS installed.
- **Non-spoofed fonts** return a width that matches the fallback — making probes think the font is NOT installed.
- Results are **deterministic and stable within a session** (same font+text → same width every time).
- Results are **coherent with the profile's OS** (the godoll fingerprint generator already constrains fonts to the profile's OS via `FPWithOS`).

## Success criteria (from REQUIREMENTS)

### FONT-01 — Observable change
- With `--font-spoof` enabled, a font-probe on the live page reads a spoofed font set that **differs from the host baseline**.
- The harness has a before-and-after test: enable spoofing → read fonts → compare to no-spoof baseline.

### FONT-02 — Coherent & stable
- The spoofed font set is **coherent with the active profile's OS/locale**.
- **Identical across re-reads** on the same session.
- `--font-spoof=false` restores genuine host font behavior.

### FONT-03 — Harness-asserted
- The detection harness asserts FONT-01 and FONT-02.
- Differs-when-on, stable-across-re-reads, restored-when-off.

## What the work entails

### Godoll change (1 file)
**`../godoll/stealth/fingerprint_bridge.go`**: Replace `scriptMockFonts` with a real injector that adds a deterministic per-session offset to spoofed fonts' `measureText` results.

### rod-cli changes (1-2 files)
**`types/context.go`**: Pass the per-session noise seed to the font script (already available via `ctx.noiseSeed`)

**`tests/detection_test.go`**: Add font-spoof harness subtests:
- Baseline (font-spoof off): read font hashes from the live page
- Spoofed (font-spoof on): read font hashes, assert they differ from baseline
- Stability: two reads within same session, assert identical
- Restore: turn off, assert hashes match baseline

## Key centerspiece symbols

| Symbol | File | Role |
|---|---|---|
| `scriptMockFonts` | `../godoll/stealth/fingerprint_bridge.go:245` | The no-op to replace |
| `applyFingerprintDimensions` | `../godoll/stealth/evasion.go:208` | Caller of scriptMockFonts |
| `createPage` | `types/context.go:641` | Where stealth is applied for rod-cli |
| `fp.Fonts` | godoll fingerprint | The font list injected per-profile |

## Dependencies

- **Depends on**: Phase 34 (toolchain).
- **Independent of**: Phase 35 (CDP ledger) — can run in parallel.
- **Godoll repo**: `../godoll` (separate git repo, linked via `replace` directive).

## Out of scope

- Font spoofing at the OS level (we spoof JS-visible font enumeration only).
- `document.fonts` API override (complex; `measureText` override is the high-leverage fix).
- Non-Chrome browser support (rod-cli = Chrome-only).
