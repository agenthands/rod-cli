---
phase: 26-configurable-fingerprint-consistency-validator
fixed: 2026-06-24
status: verified_green
findings_fixed: [CR-01, CR-02, WR-01, WR-03]
findings_deferred: [WR-02, WR-04, WR-05, INFO-01, INFO-02, INFO-03, INFO-04]
---

# Phase 26: Code Review Fix Report

**Source review:** 26-REVIEW.md (2 Critical, 5 Warning, 4 Info)
**Fixes applied by:** gsd-code-fixer (session 2026-06-24), verified this session

## Verification Results

All targeted test suites passed after fixes:

| Suite | Command | Result |
|-------|---------|--------|
| unit | `go test ./types/ ./actions/` | PASS |
| godoll stealth | `cd ../godoll && go test ./stealth/` | PASS |
| detection harness | `go test ./tests/ -run TestDetectionHarness -count=1` | PASS (11/11) |
| network headers | `go test ./tests/ -run TestNetworkEvasionHeaders -count=1` | PASS |

## Fixes Applied

### CR-01 — stealth-check mis-verdicts webdriver on a correctly-stealthed page

**File:** `actions/stealth_check.go`

godoll's `scriptHideAutomation` sets `navigator.webdriver` to `undefined` (via
`get: () => undefined` then `delete`), so the probe returns the string `"undefined"`.
The original `computeVerdict` only accepted `"false"` as PASS; this made
`stealth-check` report `webdriver=FAIL` against a correctly-stealthed page.

**Fix:** Accept both `"undefined"` and `"false"` as PASS in `computeVerdict`; only
`"true"` is a tell. Strengthened the `stealth_check` subtest in `tests/detection_test.go`
to assert the webdriver signal actually PASSes (not just that output is well-formed).

### CR-02 — fingerprint strings interpolated raw into injected JS literals

**File:** `../godoll/stealth/js.go` (uncommitted by design — separate repo)

UA / platform / vendor / timezone / languages were interpolated with `%s` / `%q`
directly into JS string literals, allowing any quote character to break out of
the literal. `json.Marshal` escaping is now applied at the JS injection boundary,
producing a value safe for embedding in `"..."` without further escaping.

Defense-in-depth added in `types/config.go`: the config validator now rejects
control characters and unescaped quotes in fingerprint string fields (platform,
vendor, timezone, locale) before they reach the injection point.

### WR-01 — stale profileFromStealth comment

**File:** `types/context.go`

Comment incorrectly described the precedence order as `Flag > Profile > Default`
after Plan 03 restructured it to `Profile fields overlay Stealth defaults`.
Corrected to match actual behavior.

### WR-03 — platform validation fail-open too wide

**File:** `types/config.go`

`validatePlatform` previously fell through to allow any non-empty string after
checking the known set, silently accepting invalid values. Tightened to
`fail-open only on genuinely unknown/future platforms`; malformed values (containing
control chars or quotes) are now rejected by the shared string validator.

## Deferred Findings

| ID | Severity | Reason |
|----|----------|--------|
| WR-02 | Warning | SOCKS5 auth relay concern — existing issue, tracked in STATE.md blockers |
| WR-04 | Warning | Harness subtest isolation — pre-existing shared daemon; not introduced by phase 26 |
| WR-05 | Warning | `deriveAndValidateFingerprint` length — acceptable complexity for now |
| INFO-01..04 | Info | Minor code clarity; not worth churn post-verification |

## Committed Files

- `actions/stealth_check.go` — CR-01 verdict fix
- `tests/detection_test.go` — CR-01 strengthened assertion
- `types/config.go` — CR-02 defense-in-depth + WR-03 validator tightening
- `types/context.go` — WR-01 comment correction

godoll (`../godoll/stealth/js.go` + related) remains UNCOMMITTED in its working
tree, consumed via `replace => ../godoll` as established in phases 24/25.
