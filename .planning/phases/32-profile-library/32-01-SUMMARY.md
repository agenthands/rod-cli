---
author: engineer
responsible: engineer
phase: 32-profile-library
plan: 01
wave: 1
type: execute-summary
status: complete
requirements: [PROF-01, PROF-03, PROF-04]
commit: e6c2e5c
---

# Phase 32 Wave 1 — Built-in Profile Library + Selection (SUMMARY)

## What landed

1. **6 embedded Chrome-only desktop profiles** (`types/profiles/*.json`), godoll
   `stealth.Profile` schema, Chrome major **138** across all six,
   `spoofClientHints/spoofCanvas/spoofAudioContext` all true:
   | name | UA OS token | platform | screen | HW / mem |
   |---|---|---|---|---|
   | windows-11-chrome | Windows NT 10.0 | Win32 | 1920x1080 @1.0 | 8 / 8 |
   | windows-11-desktop-1440p | Windows NT 10.0 | Win32 | 2560x1440 @1.0 | 16 / 8 |
   | windows-10-chrome | Windows NT 10.0 | Win32 | 1920x1080 @1.0 | 8 / 8 |
   | windows-10-laptop | Windows NT 10.0 | Win32 | 1366x768 @1.0 | 4 / 8 |
   | macos-applesilicon-chrome | Intel Mac OS X 10_15_7 | MacIntel | 2560x1600 @2.0 | 8 / 8 |
   | macos-intel-chrome | Intel Mac OS X 10_15_7 | MacIntel | 1920x1080 @1.0 | 8 / 8 |

2. **Embed + lookup API** (`types/profiles_embed.go`): `//go:embed profiles/*.json`,
   `BuiltinProfileNames() []string` (sorted, returns a copy), `LoadBuiltinProfile(name)
   (*stealth.Profile, bool, error)` (ok=false → not a built-in → caller falls back;
   malformed embedded JSON is a hard error). `sync.Once`-cached name set.

3. **Built-in-first resolution** (`types/config.go`): `loadSelectedProfile` +
   `profileLooksLikePath`. A bare name resolves a built-in FIRST, then the existing
   `~/.rod-cli/profiles/<name>.json`; a path-like value (separator or `.json`) loads
   verbatim (PROF-04). `ResolveStealth` Tier-2 overlay and CLI>profile>default
   precedence unchanged; missing/malformed `--profile` stays a LOUD failure.
   `ProfilePath` is the sentinel `builtin:<name>` for built-ins (documented on the field).

4. **`--profile=list`** (`cmd.go`, `maybeHandleProfileList`): intercepted at the top of
   `runClientCommand` AND in the root Action, BEFORE any daemon spawn or profile load —
   `list` is reserved, never loaded. Honors `--raw` (names only), `--json` (structured),
   default human table. `--profile` flag Usage updated.

## Embed-site decision
Profiles live at **`types/profiles/`** with the embed site at `types/profiles_embed.go`
(package `types`), co-located with `resolveProfilePath`/`ResolveStealth`. One embed site;
reachable from the embedding file's package (a repo-root `profiles/` embedded from
package `types` is not reachable, per the watch-for note).

## Deviation from CONTEXT (with cause)
- **deviceMemory 16 → 8** on the three "premium" profiles (macOS x2, win11-1440p).
  CONTEXT D-01 listed "mem 16", but `navigator.deviceMemory` (and `Sec-CH-Device-Memory`)
  is **capped at 8 by the W3C Device Memory API** — real Chrome never reports >8 regardless
  of physical RAM, so 16 is a synthetic fingerprint tell. Per the ship-only-vetted
  discipline (and CONTEXT's "engineer may refine fields for coherence"), corrected to 8
  (the value a 16GB+ machine genuinely reports). Surfaced by the code-reviewer gate.
- **Validator hardened**: `validateHardwareAndScreen` deviceMemory upper bound 64 → 8
  (with the Device Memory API rationale), so a future profile/custom config can't
  reintroduce the tell. No existing test/config relied on >8.

## Test evidence
- `go build ./...` clean; `go vet ./...` clean.
- `go test ./types/ -count=1` → PASS (12.8s) — existing ResolveStealth/profile-tier tests
  green with the new resolution path and tightened bound.
- Throwaway pre-commit gate: all 6 built-ins pass `deriveAndValidateFingerprint` with
  `userSetFingerprint{Platform:true, Locale:true}` (so UA-OS↔platform and locale↔languages
  contradictions WOULD be rejected) and each UA carries a parseable Chrome major. PASS.
- Live binary `--profile=list` (human / `--raw` / `--json`) all print the 6 names and exit
  WITHOUT launching a daemon or loading a profile named "list".

## Independent review
`anvil-code-reviewer` (cost $1.60, 195s) → `/tmp/anvil-spawn-out/32-REVIEW.md`. Verdict
FIX-FIRST: 0 blockers, 1 major, 3 minors. All triaged:
- MAJOR deviceMemory>8 → **fixed** (profiles + validator).
- MINOR --json list drops corrupt built-in → **fixed** (emits name with "unreadable").
- MINOR ProfilePath holds non-path label → **fixed** (field doc note).
- MINOR built-in shadows same-named user-dir profile → by-design (D-02); will be surfaced
  in docs (32-02 Task 3).
Checked-clean by reviewer: concurrency (sync.Once happens-before), loud-failure preserved,
"list" reservation, resolution order, all 6 profiles validate.

## Codebase drift
New structural element: `types/profiles/` directory (new embedded asset dir) — flag to
architect for codebase-archaeologist re-map (non-blocking).
