---
author: architect
responsible: architect
phase: 36
status: ready_for_execution
parent_artifacts:
  - .planning/phases/36/CONTEXT.md
  - .planning/REQUIREMENTS.md (FONT-01, FONT-02, FONT-03)
---

# Phase 36: Real Font Spoofing — PLAN

## §1. godoll/stealth/fingerprint_bridge.go — Real font injector

Replace the `scriptMockFonts` function with a real implementation.

**Current (no-op):** All code paths return the original `measureText` result.

**New (real):** For fonts in the spoofed list, add a deterministic offset to the measured width (making probes think the font is installed and distinguishable from fallback). For non-spoofed fonts, return the original measurement (which already correctly reflects whether the font is installed — rod-cli constrains fonts to the profile's OS, so spoofed fonts are what should appear installed).

**The new function:**

```go
func scriptMockFonts(fonts []string, seed int64) string {
    fontsJSON, _ := json.Marshal(fonts)
    return fmt.Sprintf(`
(function() {
    const availableFonts = new Set(%s);
    const origMeasureText = CanvasRenderingContext2D.prototype.measureText;
    const cache = {};
    const SEED = %d;

    // Deterministic per-session offset (1-7px)
    function fontOffset(fontName) {
        let h = SEED;
        for (let i = 0; i < fontName.length; i++) {
            h = ((h << 5) - h + fontName.charCodeAt(i)) | 0;
        }
        return Math.abs(h %% 7) + 1;
    }

    CanvasRenderingContext2D.prototype.measureText = function(text) {
        const font = (this.font || '').toLowerCase();
        const cacheKey = font + '\x00' + text;
        if (cache[cacheKey]) return cache[cacheKey];

        const result = origMeasureText.call(this, text);

        for (const af of availableFonts) {
            if (font.includes(af.toLowerCase())) {
                const offset = fontOffset(af);
                const fake = {width: result.width + offset};
                cache[cacheKey] = fake;
                return fake;
            }
        }

        cache[cacheKey] = result;
        return result;
    };
})()
`, string(fontsJSON), seed)
}
```

**Note:** This adds a `seed` parameter to `scriptMockFonts`. Update the caller in `evasion.go:applyFingerprintDimensions()` to pass the seed from the EvasionManager's noise seed.

## §2. godoll/stealth/evasion.go — Pass seed to font script

In `applyFingerprintDimensions()`, change the font injection call:
```go
fontsScript := scriptMockFonts(fp.Fonts, em.noiseSeed)
```

(Currently: `fontsScript := scriptMockFonts(fp.Fonts)`)

## §3. types/context.go — Pass seed through dimension options

Verify that `em.SetNoiseSeed(ctx.noiseSeed)` (line 691) is called before `em.Apply()` — it already is. No change needed in rod-cli's context.go for the seed.

## §4. tests/detection_test.go — Font-spoof harness tests

Add to `TestAdvancedEvasionDimensions` or create a new `TestFontSpoof` function:

### 4.1 Baseline (FONT-01, FONT-03 — differs-when-on)
```go
// 1. Spawn session with font-spoof OFF, navigate to fixture, read font hashes
// 2. Spawn session with font-spoof ON, navigate to fixture, read font hashes  
// 3. Assert: the two font sets differ
```

### 4.2 Stability (FONT-02, FONT-03 — stable-across-re-reads)
```go
// 1. With font-spoof ON, read font hashes twice from the same session
// 2. Assert: identical
```

### 4.3 Restore (FONT-02, FONT-03 — restored-when-off)
```go
// 1. With font-spoof OFF, read font hashes
// 2. Assert: matches the baseline from 4.1 step 1
```

The fixture page needs a JS font probe that enumerates fonts via canvas measurement and returns a hash:

```javascript
// Font probe: measure common fonts via canvas, hash results
(function() {
    const testFonts = ['Arial','Times New Roman','Courier New','Georgia','Verdana',
        'Comic Sans MS','Impact','Trebuchet MS','Palatino Linotype','Lucida Console'];
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    const results = {};
    for (const f of testFonts) {
        ctx.font = '72px ' + f + ', sans-serif';
        results[f] = ctx.measureText('abcdefghijklmnopqrstuvwxyz').width;
    }
    return JSON.stringify(results);
})()
```

## §5. Verification gates

| Gate | Command | Expected |
|------|---------|----------|
| Godoll build | `cd ../godoll && go build ./...` | exit 0 |
| rod-cli build | `go build ./...` | exit 0 (picks up godoll changes via replace) |
| Font-spoof harness | `go test ./tests/ -run TestFontSpoof -count=1 -v` | FONT-01, FONT-02 pass |
| Godoll tests | `cd ../godoll && go test ./stealth/ -run Font -count=1 -v` | existing font tests pass |

## §6. Edge cases / risks

- **Seed=0**: The noise seed is generated via `crypto/rand` with a fallback. It won't be 0 in practice, but the hash still works with seed=0 (just produces predictable offsets — same across all daemons with seed=0, which is fine).
- **Empty font list**: godoll generates fonts per-profile. If the list is empty, `scriptMockFonts` returns a valid script that adds no spoofed fonts (no-op by design — no fonts to spoof = no override needed).
- **Godoll is a separate repo**: Changes to `../godoll` must be committed separately. The rod-cli `replace` directive picks up local changes immediately.
- **Existing godoll tests**: The `scriptMockFonts` function signature CHANGE (adding `seed` parameter) will break callers. Update all callers.
