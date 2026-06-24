---
phase: 26-configurable-fingerprint-consistency-validator
plan: 01
subsystem: stealth-config
status: complete
tags: [fingerprint, config, validation, cli, stealth]
dependency_graph:
  requires:
    - "Phase 25 StealthConfig substrate (types/config.go ResolveStealth three-tier resolver, StealthFlags, resolveProfilePath)"
    - "godoll stealth.Profile / DefaultProfile / LoadProfile"
  provides:
    - "StealthConfig fingerprint identity fields (UserAgent/Locale/Timezone/Platform/Screen/AcceptLanguage/Languages/HardwareConcurrency/DeviceMemory/Vendor/SpoofClientHints)"
    - "4 curated override flags --user-agent/--locale/--timezone/--platform (registered, forwarded, captured daemon-side)"
    - "deriveAndValidateFingerprint consistency validator + parseChromeMajor / uaOSToPlatform UA-anchor helpers in ResolveStealth"
    - "stealth-check [url] command surface (registered; behavior is Plan 04)"
  affects:
    - "Plan 02 (godoll CH derivation) and Plan 03 (runtime injector) consume the validated, coherent cfg.Stealth"
    - "Plan 04 implements the stealth-check daemon dispatch + StealthCheck action behind the registered command"
tech_stack:
  added: []
  patterns:
    - "Loud-failure config resolution (errors.Errorf naming conflicting fields) -> main.go aborts before NewContext/browser launch"
    - "Derive-when-unset, reject-when-user-conflicts policy gated on a userSetFingerprint flag-origin struct"
    - "Non-secret flags forwarded verbatim into the daemon spawn argv (persistence linchpin); warn-only stderr for timezone<->proxy"
key_files:
  created: []
  modified:
    - "types/config.go"
    - "cmd.go"
    - "main.go"
decisions:
  - "The UA string is the single derivation anchor: Chrome major via parseChromeMajor, OS token via uaOSToPlatform. The brand-string formatting is deliberately NOT done here (it belongs to the Plan 02/03 runtime injector); this plan only validates+derives the coherent identity in cfg.Stealth."
  - "Reject only genuine user-set contradictions. A userSetFingerprint struct records whether Platform/Locale came from a CLI flag, so a profile-supplied value never triggers a hard reject — only a flag-set value that contradicts the UA does. Unset dependents are silently auto-derived."
  - "timezone<->proxy-geo is warn-only on stderr (no geo-IP lookup), preserving offline-determinism per CONTEXT decision."
  - "Locale<->languages coherence compares primary subtags case-insensitively (en-US matches en-GB at the language level; fr-FR does not match en-US), so reasonable regional variation is not falsely rejected."
metrics:
  duration: "3m47s"
  completed: "2026-06-24"
  tasks: 3
  files_modified: 3
---

# Phase 26 Plan 01: Configurable Fingerprint Config + Consistency Validator Summary

Built the configuration + validation spine for the pinnable, coherent fingerprint: extended `StealthConfig`/`StealthFlags` with the 11 reserved identity fields, registered+forwarded+captured the 4 curated override flags (`--user-agent`/`--locale`/`--timezone`/`--platform`), implemented `deriveAndValidateFingerprint` inside `ResolveStealth` (UA-anchored derive-when-unset, loud reject-when-user-conflicts, warn-only timezone↔proxy), and registered the `stealth-check [url]` command surface that Plan 04 will give behavior.

## What Was Built

### Task 1 — Identity fields + 4 override flags (commit f899f1d)
- `types/config.go`: `StealthConfig` gains `UserAgent, Locale, Timezone, Platform, AcceptLanguage, Languages, HardwareConcurrency, DeviceMemory, Vendor, SpoofClientHints` and a `Screen{Width,Height,DeviceScaleFactor}` sub-struct, all with the existing yaml/json lowerCamel tag convention. The Phase-26 reserved comment block was replaced with an "implemented" note. `StealthFlags` gains `UserAgent/Locale/Timezone/Platform` raw values.
- `cmd.go`: registered `--user-agent/--locale/--timezone/--platform` global StringFlags; appended each non-empty one verbatim into the daemon spawn argv (right after `--proxy`/`--profile`); ORed all four into the `stealthRequested` already-running warning predicate.
- `main.go`: captured all four into `StealthFlags` in `runDaemonServer`.

### Task 2 — Consistency validator + UA-anchor derivation (commit 68b7fec)
- `parseChromeMajor(ua)` — extracts the integer after `Chrome/` via a package-level compiled `chromeMajorRe`.
- `uaOSToPlatform(ua)` — maps `Windows NT`→(`Win32`,`Windows`), `Macintosh`/`Mac OS X`→(`MacIntel`,`macOS`), `X11`/`Linux`→(`Linux`,`Linux`).
- `deriveAndValidateFingerprint(cfg, userSet)` — runs on the overlaid `cfg.Stealth`: derives Platform/Locale from the UA/Languages when unset; rejects a user-set Platform/Locale that contradicts the anchor (message names both fields + the remedy); range-checks Screen geometry and HardwareConcurrency(1–256)/DeviceMemory(1–64); emits a warn-only stderr line when both Timezone and Proxy are set. Wired into `ResolveStealth` after the Tier-2 overlay + Tier-1 flag application, before the `cfg.Proxy` bridge, returning its error so it aborts the daemon before `NewContext`.
- Tier-2 overlay now copies the loaded profile's identity fields onto `cfg.Stealth` (was a reserved no-op); Tier-1 applies the 4 flags with the existing `if flags.X != "" {}` shape.

### Task 3 — stealth-check command surface (commit 262d8a0)
- `cmd.go`: registered `stealth-check [url]` mirroring the `open` optional-positional shape; routes through `runClientCommand` forwarding `url` (optional) + `raw` + `json` in `Args`. No daemon dispatch case or `StealthCheck` action added (those are Plan 04) — confirmed `daemon/daemon.go` has no `stealth-check` case yet.

## Verification

- `go build ./...` and `go vet ./types/ ./...` clean.
- `go test ./types/` passes (10.7s).
- `--help` lists `stealth-check`.
- Validator behavior confirmed via a throwaway table test (added, run, removed — not committed):
  - UA(`Windows NT 10.0` … `Chrome/121`) + `--platform=MacIntel` → non-nil error containing both `MacIntel` and `Windows`.
  - Same UA, platform unset → `nil` error and `cfg.Stealth.Platform == "Win32"`.
  - `parseChromeMajor(UA) == 121`.
  - Timezone + Proxy both set → `nil` error + a `warning:` stderr line (no hard fail).

## Deviations from Plan

None — plan executed exactly as written.

The plan suggested expressing the behavior assertions "as a unit-style check the executor adds OR a manual reasoning check"; I used a throwaway in-package table test to verify empirically, then removed it (the committed test suite for these behaviors is Plan 05's `consistency_invariant` in `tests/detection_test.go`). No production test files were added or left behind.

## Authentication Gates

None.

## Known Stubs

None. The `stealth-check` command intentionally has no behavior yet — that is scoped to Plan 04 (per the plan and the phase artifact index), not a stub gap in this plan.

## Self-Check: PASSED

- types/config.go — FOUND (modified, contains `deriveAndValidateFingerprint`, `parseChromeMajor`, `uaOSToPlatform`)
- cmd.go — FOUND (modified, `--user-agent` flag + `append(flags, "--user-agent"` + `stealth-check` command)
- main.go — FOUND (modified, `UserAgent: c.String("user-agent")`)
- Commit f899f1d — FOUND
- Commit 68b7fec — FOUND
- Commit 262d8a0 — FOUND
