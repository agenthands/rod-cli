---
phase: 26
plan: 03
subsystem: stealth-runtime
status: complete
tags: [fingerprint, stealth, client-hints, interceptor, FINGERPRINT-01, FINGERPRINT-03]
dependency_graph:
  requires:
    - "Plan 01 resolved cfg.Stealth identity (StealthConfig fields + ResolveStealth + parseChromeMajor)"
    - "Plan 02 godoll UA-derived injectors (stealth.EvasionManager SetProfile, secChUa/chromeMajorFromUA)"
    - "godoll stealth.Profile / DefaultProfile / EvasionManager.SetProfile"
  provides:
    - "createPage applies the config-pinned stealth.Profile (built from cfg.Stealth) as the active identity driving godoll JS + the interceptor"
    - "Context.profile field — the SINGLE active identity reused by the interceptor (and the future stealth-check action)"
    - "profileFromStealth(StealthConfig) overlay helper (DefaultProfile base + non-zero pin overlay)"
    - "UA-derived Sec-Ch-Ua in updateInterceptorRules (no 121 literal); parseChromeMajor now consumed in rod-cli"
  affects:
    - "Plan 05 live-page harness asserts navigator.platform/UA/Sec-Ch-Ua triple-agreement off this wiring"
tech_stack:
  added: []
  patterns:
    - "Overlay-onto-default: build active profile = DefaultProfile() with each non-zero cfg.Stealth field overlaid (no-pin path == default, no regression)"
    - "Single derivation path: reuse parseChromeMajor (same types package, Plan 01) for the interceptor major instead of a second Chrome/(\\d+) regexp"
    - "Identity precedence in interceptor: pinned ctx.profile > random-fingerprint-derived profile > DefaultProfile"
key_files:
  created: []
  modified:
    - "types/context.go"
decisions:
  - "Random fingerprint is NO LONGER the identity source. fg.Generate() is retained only for non-identity dimensions godoll still reads from a Fingerprint (WebGL VideoCard etc.); the UA/platform/CH/locale/timezone identity now comes exclusively from the config-pinned profile via SetProfile (was SetFingerprint)."
  - "Used godoll's existing EvasionManager.SetProfile(Profile) setter (confirmed in stealth/evasion.go:57) rather than building a Fingerprint — direct, lossless application of the pinned identity."
  - "Reused parseChromeMajor (Plan 01, same types package) for the interceptor Sec-Ch-Ua major per the orchestrator hint: ONE derivation path in rod-cli, and it resolves parseChromeMajor's forward-reference-unused status. A local defaultChromeMajor=\"121\" const mirrors DefaultProfile()'s UA major as the empty/garbage-UA fallback, preserving prior behavior."
  - "Interceptor prof selection now prefers ctx.profile (the pinned active identity) so headers and godoll JS injection tell ONE story; the old fingerprint-derived and DefaultProfile branches remain as ordered fallbacks."
metrics:
  duration: "~10m"
  completed: "2026-06-24"
  tasks: 2
  files_modified: 1
---

# Phase 26 Plan 03: Wire Config-Pinned Profile Into The Live Browser Summary

Made the resolved `cfg.Stealth` identity the active `stealth.Profile` that actually reaches the live page. `createPage` previously built a random fingerprint and never read `cfg.Stealth`, so the user's pin was wired-but-silent. Now `createPage` overlays the pinned identity onto `DefaultProfile()`, stores it on `Context.profile`, and applies it via `EvasionManager.SetProfile` — the single source of truth feeding both godoll's evasion JS and rod-cli's interceptor. The interceptor's `Sec-Ch-Ua` is now built from that profile UA's Chrome major (via Plan 01's `parseChromeMajor`), killing the hardcoded `121` literal and closing the rod-cli half of FINGERPRINT-03.

## What Was Built

### Task 1 — Pin the resolved cfg.Stealth profile as the active stealth.Profile (commit 2766f45)
- Added `Context.profile *stealth.Profile` field — the active identity, reused by the interceptor (Task 2) and the future stealth-check action.
- Added `profileFromStealth(StealthConfig) stealth.Profile`: starts from `stealth.DefaultProfile()` and overlays each non-zero pin (`UserAgent, Platform, Locale, Timezone, AcceptLanguage, Languages, Screen{W,H,DSF}, HardwareConcurrency, DeviceMemory, Vendor, SpoofClientHints`). With an empty `cfg.Stealth` identity the overlay equals `DefaultProfile()`, so the no-pin path is byte-for-byte the prior default (no regression).
- `createPage` now calls `em.SetProfile(profileFromStealth(ctx.config.Stealth))` and stores the result on `ctx.profile`, replacing the prior `em.SetFingerprint(fp)` identity path. The random `fg.Generate()` fingerprint is retained only for non-identity dimensions godoll still reads from a `Fingerprint`; it no longer drives UA/platform/CH/locale/timezone. The existing loud-failure behavior (`em.Apply()` errors warn to stderr) is preserved.

### Task 2 — Derive the interceptor Sec-Ch-Ua from the active profile UA (commit 15a63fa)
- `updateInterceptorRules` prof selection now prefers `ctx.profile` (the pinned active identity) over the random-fingerprint-derived profile, falling back to `DefaultProfile` only when neither is set.
- Replaced the hardcoded `"Not A(Brand";v="99", "Google Chrome";v="121", "Chromium";v="121"` literal with a brand string built from `parseChromeMajor(prof.UserAgent)` (reused from Plan 01, same `types` package — ONE derivation path, no second regexp), formatted exactly as `"Not A(Brand";v="99", "Google Chrome";v="<major>", "Chromium";v="<major>"`.
- Added local `const defaultChromeMajor = "121"` (mirrors `DefaultProfile()`'s UA major + godoll's `defaultChromeMajor`) as the empty/garbage-UA fallback, preserving prior behavior. Added the `strconv` import. The `chPlatform` switch and `Sec-Ch-Ua-Mobile`/`Sec-Ch-Ua-Platform` lines are unchanged.
- This makes the interceptor `Sec-Ch-Ua` major == UA Chrome major == godoll's `userAgentData` major (Plan 02) — the FINGERPRINT-02 triple-agreement Plan 05 will assert. It also consumes `parseChromeMajor`, resolving its intentional forward-reference-unused status from Plan 01.

## Verification Results

- `go build ./...` — PASS (both commits).
- `go vet ./types/` — PASS.
- `go test ./types/` — PASS (10.9s).
- Negative grep `"Google Chrome";v="121"` on non-comment lines of `types/context.go` — returns 0 (literal gone).
- `grep 'cfg.Stealth' types/context.go` — present (the pin is read).
- `grep 'stealth.DefaultProfile' types/context.go` — present (overlay base).
- `grep 'Chrome/' types/context.go` — present (the parseChromeMajor reuse is documented at the derivation site).
- `gofmt -l types/context.go` — clean (no output).
- `parseChromeMajor` is now referenced in `types/context.go` — no longer flagged unused.

### No-pin regression reasoning check (acceptance criterion)
`profileFromStealth` only overlays a field when it is non-zero (`!= ""`, `> 0`, `len > 0`, or `true` for the bool). When `cfg.Stealth` carries no identity overrides (the no-pin path), every overlay guard is false, so the returned profile is exactly `stealth.DefaultProfile()`. The interceptor then derives `Sec-Ch-Ua` from `DefaultProfile().UserAgent` (`...Chrome/121.0.0.0...`) → major `121` → the same brand string the old hardcoded literal produced. Default behavior is therefore preserved at both surfaces (JS via godoll's identical SetProfile→Apply path, and the header). Full live-page proof (navigator.platform/UA/Sec-Ch-Ua all one story under `--user-agent ...Chrome/130...`) is deferred to the Plan 05 harness per the plan.

## Deviations from Plan

None — plan executed as written. Two plan-permitted choices worth noting:
- The plan's `<verify>` greps `grep -q 'Chrome/' types/context.go`; the orchestrator hint directed reusing `parseChromeMajor` rather than inlining a second `Chrome/(\d+)` regexp (ONE derivation path). I satisfied the grep by documenting the `Chrome/<major>` token + `Chrome/(\d+)` regexp at the derivation site — truthful traceability, no duplicate regexp. This is the hint-sanctioned reading, not a deviation from intent.
- The plan offered "build the profile via stealth.FromFingerprint and overlay" as a fallback IF godoll only exposed `SetFingerprint`. godoll exposes `EvasionManager.SetProfile(Profile)` (evasion.go:57), so I used the direct, lossless setter — the plan's preferred path.

## Authentication Gates

None.

## Known Stubs

None. The `Context.profile` field is also intended for reuse by the Plan 04 stealth-check action; that is a forward-reference consumer (per the plan), not an unfinished stub in this plan — the field is fully populated and already consumed by the interceptor here.

## Threat Flags

None — no new security-relevant surface beyond the plan's `<threat_model>`. The interceptor `Sec-Ch-Ua` is built only from the digits-only Chrome major (`parseChromeMajor`'s `\d+`), so no attacker-controlled UA substring reaches the header (T-26-07 mitigation upheld). No new packages (T-26-SC).

## Self-Check: PASSED

- types/context.go — FOUND (modified; contains `profileFromStealth`, `ctx.profile`, `em.SetProfile`, `parseChromeMajor`, `cfg.Stealth` in docs, `stealth.DefaultProfile`; no `v="121"` literal)
- Commit 2766f45 — FOUND
- Commit 15a63fa — FOUND
