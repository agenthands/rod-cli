---
author: engineer
responsible: engineer
phase: 32-profile-library
plan: 02
wave: 2
type: execute-summary
status: complete
requirements: [PROF-02]
commit: 15d4fb7
depends_on: ["32-01"]
---

# Phase 32 Wave 2 — The "Vetted" Gate + Docs (SUMMARY)

## What landed

1. **Offline consistency gate** (`types/profiles_test.go`, `TestBuiltinProfilesAreVetted`):
   iterates `BuiltinProfileNames()` and for EACH built-in:
   - overlays it onto `DefaultConfig` exactly as `ResolveStealth`'s Tier-2 block does
     (`overlayProfile` helper), then runs `deriveAndValidateFingerprint` with
     `userSetFingerprint{Platform:true, Locale:true}`. This is **deliberately stronger**
     than `ResolveStealth`'s empty `userSet`: a built-in PINS platform+locale, and a
     vetting gate must REJECT a pinned value that contradicts the UA (the empty-userSet
     path would silently derive instead). Documented in the test.
   - structural backstops: UA carries a parseable Chrome major; `uaOSToPlatform(UA)`
     equals the pinned platform (catches Win-UA + MacIntel even though both are "known");
     `languages` non-empty (also forces the validator's locale↔languages branch to
     engage); screen/HW/mem all positive; vendor `Google Inc.`; canvas+audio noise on.
   - count guard (5–10, PROF-01). Runs under plain `go test ./...` — no skip, no build
     tag, offline, deterministic (0.006s). An incoherent profile **fails the build**.
   - plus `TestBuiltinProfileResolvesBeforeUserDir` (built-in-first + path-verbatim).

2. **Live harness leg** (`tests/detection_test.go`, `TestDetectionHarness/builtin_profiles`):
   drives a representative subset — `windows-11-chrome`, `windows-10-laptop`,
   `macos-applesilicon-chrome` — through the offline `internal/detect` fixture via
   `--profile=<name>` (the EMBEDDED built-in, no temp file) and asserts LIVE-page reads:
   UA major + `navigator.platform` reach the page, no `HeadlessChrome`,
   `navigator.webdriver` not `true`, WebGL not a software rasterizer, `Sec-Ch-Ua` major
   == UA major, `Sec-Ch-Ua-Platform` OS family == UA OS family (unconditional — fails on
   an absent header). Offline (127.0.0.1 + loopback echo), part of the default suite.

3. **Docs** (`docs/stealth-config.md`): new §8 "Built-in profile library" (the 6-name
   table, real-Chrome/no-spoofed-TLS cross-ref to stealth-validation.md, `--profile=list`
   + by-name select, the vetted-gate description, precedence PROF-04, reserved-name
   semantics + the path escape-hatch for a same-named custom profile). Tier-2 + `--profile`
   catalog row updated; the `deviceMemory` bound corrected to 1–8 (Device Memory API cap).

## Subset rationale (PROF-02 harness)
Each built-in is a full browser spawn (~5s), so driving all 6 live would bloat the suite.
The harness drives 3 (one per OS family + a 4-core/1366x768 HW variant + the Retina
MacIntel nuance); the OFFLINE gate (Task 1) covers ALL 6 deterministically. Documented
in the test per the plan's allowance.

## Test evidence
- `go build ./...` + `go vet ./...` clean.
- `go test ./types/ -run "Builtin|Profile"` → PASS (all 6 vetted; resolution test green).
- `go test ./tests/ -run "TestDetectionHarness/builtin_profiles"` → PASS (22.8s) —
  win11/win10-laptop/macos-applesilicon each drive end-to-end through real Chrome with
  coherent live-page signals.

## Independent review
`anvil-code-reviewer` on the wave-2 diff (cost $2.18, 265s) → verdict FIX-FIRST, 0
blockers, 2 majors + 1 minor — ALL **latent coverage holes** (the gate is REAL, not
vacuous, for the 6 shipped profiles; the reviewer traced + ran it). All three accepted
and fixed BEFORE this commit:
- MAJOR harness `Sec-Ch-Ua-Platform` skipped on empty header → now unconditional (fails on absent).
- MAJOR offline gate had no backstop for empty `languages` (would disable the locale check)
  → added a `languages`-pinned structural assertion.
- MINOR gate accepted zero hardware/screen → added screen/HW/mem positivity assertions.
These strengthen the gate to catch the full coherence space so "vetted" stays honest for
FUTURE profiles, not just today's 6.

## PROF-02 honesty note
The gate certifies INTERNAL coherence (the identity tells one consistent OS+version story)
and table-stakes live signals on the offline fixture — it does NOT claim "undetectable
against any real anti-bot." That distinction is the v1.6 validate-live-not-source stance
(docs/stealth-validation.md). All profiles are real Chrome; there is no TLS-spoofing
surface in the schema.
