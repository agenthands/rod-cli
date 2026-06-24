---
phase: 26
plan: 02
subsystem: stealth-fingerprint
tags: [client-hints, fingerprint, godoll, cross-repo, FINGERPRINT-03]
requires:
  - Phase 25 stealth config substrate (godoll stealth.Profile as single source of truth)
provides:
  - UA-derived Sec-Ch-Ua header in godoll stealth/evasion.go (no 121 literal)
  - UA-derived navigator.userAgentData brand version in godoll stealth/js.go (no 121 literal)
  - Locale-derived Timezone in godoll stealth.FromFingerprint (no hardcoded America/New_York)
  - chromeMajorFromUA + secChUa helpers shared by header and JS injectors (triple-agreement)
affects:
  - godoll stealth runtime injectors (header CH, JS userAgentData)
  - godoll fingerprint_bridge timezone derivation
tech-stack:
  added: []
  patterns:
    - "package-level regexp.MustCompile(`Chrome/(\\d+)`) for digits-only major extraction"
    - "mirror fingerprint/headers.go addClientHints brand-string shape across injectors"
    - "locale->IANA timezone lookup map with fallback (offline-deterministic)"
key-files:
  created:
    - ../godoll/stealth/clienthints_derive_test.go (UNCOMMITTED in godoll working tree)
  modified:
    - ../godoll/stealth/evasion.go (UNCOMMITTED in godoll working tree)
    - ../godoll/stealth/js.go (UNCOMMITTED in godoll working tree)
    - ../godoll/stealth/fingerprint_bridge.go (UNCOMMITTED in godoll working tree)
    - ../godoll/stealth/export_test.go (UNCOMMITTED in godoll working tree)
decisions:
  - "Fingerprint exposes no timezone/geo signal, so FromFingerprint derives the zone from the resolved Locale via a small localeToTimezone map (en-US/en-GB/de-DE/fr-FR/ja-JP) with America/New_York as the unknown-locale fallback."
  - "chromeMajorFromUA extracts digits-only via regexp; defaultChromeMajor='121' preserves prior behavior for empty/garbage UA while the DefaultProfile UA (Chrome/121) keeps every existing assertion green unchanged."
metrics:
  duration: 9min
  completed: 2026-06-24
status: complete
---

# Phase 26 Plan 02: Configurable Fingerprint Consistency Validator (Kill CH-121) Summary

UA-derived Client-Hints across godoll's two runtime injectors plus locale-derived timezone in FromFingerprint: the `Sec-Ch-Ua` header brand version, the `navigator.userAgentData` brand version, and the UA `Chrome/(major)` now all derive from `em.profile.UserAgent` via one shared `chromeMajorFromUA` helper, so they tell one version story (the FINGERPRINT-02 cross-surface invariant). The hardcoded `121` literal is gone from both injector paths; `FromFingerprint` no longer hardcodes `America/New_York`.

## Cross-Repo Note (IMPORTANT)

All code changes in this plan live in the **godoll working tree** (`/home/john/go/src/github.com/agenthands/godoll`), a separate git repo consumed by rod-cli via the `replace => ../godoll` directive. Per the phase's cross-repo protocol:

- **No commits were made in the godoll repo.** The godoll changes are intentionally left uncommitted in its working tree — this matches the Phase 24/25 pattern where godoll edits (`browser/*`, `network/interceptor.go`, the Phase-25 evasion.go interception hunk) are also uncommitted. rod-cli builds against godoll's working tree, so the edits are live without any commit.
- **Nothing was git-added into the rod-cli repo from `../godoll/*`.** rod-cli cannot track sibling-repo files. The only rod-cli commit for this plan is this SUMMARY.md plus STATE.md/ROADMAP.md tracking updates.
- The pre-existing Phase-25 commented-out `// go em.EnableRequestInterception()` hunk in evasion.go was preserved (now at line 151 after the added helper block above it).

## What Was Built

### Task 1 — UA-derived Sec-Ch-Ua header (godoll stealth/evasion.go)
- Added package-level `chromeMajorRe = regexp.MustCompile(\`Chrome/(\d+)\`)`, `defaultChromeMajor = "121"`, and helpers `chromeMajorFromUA(ua string) string` (digits-only extraction, fallback to default) and `secChUa(major string) string` (mirrors the `fingerprint/headers.go` addClientHints shape `"Not A(Brand";v="99", "Google Chrome";v="%s", "Chromium";v="%s"`).
- Replaced the hardcoded `Sec-Ch-Ua` literal in `EnableRequestInterception` with `secChUa(chromeMajorFromUA(em.profile.UserAgent))`. Left Sec-Ch-Ua-Mobile and Sec-Ch-Ua-Platform untouched.

### Task 2 — Parameterize navigator.userAgentData by major (godoll stealth/js.go)
- Changed `scriptMockUserAgentData(platform string)` → `scriptMockUserAgentData(platform, version string)`; the two `version: "121"` brand entries now interpolate the `version` param (the `Not A(Brand` v99 stays constant).
- Updated the sole caller in evasion.go (`Apply`) to pass `chromeMajorFromUA(em.profile.UserAgent)`, achieving header CH == JS userAgentData == UA Chrome major.

### Task 3 — Derived Timezone (godoll stealth/fingerprint_bridge.go)
- Added package-level `localeToTimezone` map and replaced the unconditional `p.Timezone = "America/New_York"` with a locale-driven lookup (fallback `America/New_York` only for unknown locales). The Fingerprint carries no timezone/geo field, so locale is the available signal — keeps the path offline-deterministic (no geo-IP).

### Task 4 — Test suite (godoll)
- No existing test asserted the `121` literal, so no assertions needed weakening/realigning. Added `stealth/clienthints_derive_test.go` (+ three exports in `export_test.go`) covering: `chromeMajorFromUA` (121/130/empty/garbage), header+JS triple-agreement for Chrome/130, and locale-derived `Europe/London` for `en-GB`.

## Verification Results

- `go build ./...` in godoll: PASS
- `go build ./...` in rod-cli (against godoll working tree): PASS
- Negative grep `"Google Chrome";v="121"` in stealth/evasion.go (non-comment): 0
- Negative grep `version: "121"` in stealth/js.go (non-comment): 0
- `go test ./stealth/... ./fingerprint/...` in godoll: PASS (incl. new derivation tests)
- gofmt clean on all modified files

## Deviations from Plan

None — plan executed as written. Tasks 1–3 production changes plus a Task-4 additive test file (the plan permitted "a godoll unit test the executor adds"); no existing test encoded the old `121` literal so none needed realigning.

## Self-Check: PASSED
- ../godoll/stealth/evasion.go: FOUND (UA-derived secChUa wired)
- ../godoll/stealth/js.go: FOUND (version-parameterized)
- ../godoll/stealth/fingerprint_bridge.go: FOUND (localeToTimezone)
- ../godoll/stealth/clienthints_derive_test.go: FOUND
- Negative greps return 0; godoll + rod-cli builds and godoll tests pass.
- No commits made in godoll repo; no godoll files staged into rod-cli (verified via git status).
