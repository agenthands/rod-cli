---
phase: 26-configurable-fingerprint-consistency-validator
reviewed: 2026-06-24T09:04:27Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - types/config.go
  - types/context.go
  - cmd.go
  - main.go
  - actions/stealth_check.go
  - daemon/daemon.go
  - internal/detect/embed.go
  - internal/detect/probe.js
  - ../godoll/stealth/evasion.go
  - ../godoll/stealth/fingerprint_bridge.go
  - ../godoll/stealth/js.go
  - tests/detection_test.go
findings:
  critical: 2
  warning: 5
  info: 4
  total: 11
status: issues_found
---

# Phase 26: Code Review Report

**Reviewed:** 2026-06-24T09:04:27Z
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

Phase 26 wires a configurable, UA-anchored fingerprint identity through a single
precedence resolver (`ResolveStealth` → `deriveAndValidateFingerprint` →
`profileFromStealth`), kills the hardcoded `121` Client-Hints literal by deriving
the Chrome major from the active UA, and adds the `stealth-check` command. The
precedence/derivation plumbing and the credential-safety posture from Phase 25
are intact (proxy-auth stays out of argv and out of any serialized struct). The
FINGERPRINT-02 triple-agreement (header == JS == UA major) is correctly threaded
through one derivation path.

Two correctness problems undercut the headline deliverables, however:

1. **`stealth-check` mis-verdicts its own default stealth output.** The probe
   reads `navigator.webdriver`, which the shipped stealth makes `undefined`, but
   `computeVerdict` only PASSes the literal `"false"`. The command therefore
   reports `webdriver=FAIL` against a correctly-stealthed page. This path is not
   asserted by any test.
2. **User-controlled fingerprint inputs (UA / platform / vendor / timezone) are
   interpolated raw into injected JS string literals** with no escaping. The
   digits-only-major mitigation only protects the *version* field; every other
   string field can break out of its `"%s"` literal — both a JS-injection vector
   (attacker-supplied `--profile` JSON) and a silent stealth-injection breakage
   (a value containing `"` corrupts the override script and re-exposes the tell).

The remaining findings are robustness and documentation defects.

## Critical Issues

### CR-01: `stealth-check` reports FAIL for `webdriver` on a correctly-stealthed page

**File:** `actions/stealth_check.go:182-186` (with `internal/detect/probe.js:40-42` and godoll `stealth/js.go:10-36`)
**Issue:**
The probe records `navigator.webdriver` verbatim:

```js
probe("webdriver", function () { return navigator.webdriver; });
```

godoll's `scriptHideAutomation` (js.go:13-18) defines the getter as
`get: () => undefined` and then `delete navigator.webdriver`, so on a stealthed
page `navigator.webdriver` evaluates to `undefined`. `readSignal` String-coerces
this to the string `"undefined"`. But `computeVerdict` only passes the exact
literal `"false"`:

```go
case "webdriver":
    if val == "false" {
        return pass()
    }
    return fail(val)   // val == "undefined" on the shipped default → FAIL
```

Result: running `stealth-check` against the tool's own default-stealthed page
yields `webdriver=FAIL(undefined)`, i.e. the headline command of the phase
contradicts the actual stealth the binary ships. The `stealth_check` subtest in
`tests/detection_test.go:560-618` only asserts the output *shape* (PASS/FAIL
prefix, label presence), never that `webdriver` PASSes, so the defect is
untested.

**Fix:** Accept the stealthed sentinel. `undefined` (property masked/deleted) and
`false` (genuine human value) are both non-tells; only `true` is the automation
tell:

```go
case "webdriver":
    if val == "false" || val == "undefined" {
        return pass()
    }
    return fail(val) // only "true" (or a read-error) fails
```

Add a test assertion that `webdriver` PASSes on the default fixture so this
cannot regress.

### CR-02: User-controlled fingerprint strings are injected into JS unescaped

**File:** godoll `stealth/js.go:78-113` (`scriptOverrideAttributes`), `stealth/js.go:144-158` (`scriptOverrideTimezone`), `stealth/js.go:163-183` (`scriptMockUserAgentData`); reached from `stealth/evasion.go:82-99,113`
**Issue:**
`UserAgent`, `Platform`, `Vendor`, and `Timezone` flow from `--user-agent` /
`--platform` / `--timezone` flags and from `--profile` JSON straight into JS
string literals via raw `fmt.Sprintf("%s", ...)`:

```go
// scriptOverrideAttributes
Object.defineProperty(navigator, 'userAgent', { get: () => "%s" });   // userAgent
Object.defineProperty(navigator, 'platform',  { get: () => "%s" });   // platform
Object.defineProperty(navigator, 'vendor',    { get: () => "%s" });   // vendor
// scriptOverrideTimezone
options.timeZone = "%s";                                              // timezone
// scriptMockUserAgentData
platform: "%s"                                                        // chPlatform (default branch = raw Platform)
```

These strings are handed to `InjectScript` → CDP
`Page.addScriptToEvaluateOnNewDocument`, which runs in the page's own context.
The `languages` field is `json.Marshal`ed (safe) and the Chrome *major* is
digits-only via the `Chrome/(\d+)` regexp (safe), but every other string field
has **no escaping and no character whitelist**. The consistency validator
(`deriveAndValidateFingerprint`) only *rejects* a user-set platform when it
contradicts a recognized UA OS token — it never sanitizes characters, and when
the UA carries no recognized OS token the platform passes through entirely
unchecked. UserAgent, Vendor, and Timezone are never validated at all.

A crafted value such as
`--platform 'x"});alert(document.cookie);({a:"'` (or the same inside a
downloaded `--profile` JSON) breaks out of the `"%s"` literal and executes
arbitrary JS on every page the session loads. Even a benign value containing a
double-quote silently corrupts the override script, the `try/catch` swallows the
syntax error, and the navigator override never applies — re-exposing the exact
tell the phase intends to hide.

**Fix:** Escape every string field at the injection boundary with
`encoding/json` (a JSON string literal is a valid JS string literal), e.g. in
`scriptOverrideAttributes`/`scriptOverrideTimezone`/`scriptMockUserAgentData`:

```go
uaJSON, _ := json.Marshal(userAgent)       // yields a quoted, escaped literal
platJSON, _ := json.Marshal(platform)
vendorJSON, _ := json.Marshal(vendor)
// ... then interpolate WITHOUT surrounding quotes:
//   get: () => %s   with uaJSON, platJSON, vendorJSON, tzJSON
```

Additionally, defense-in-depth in `deriveAndValidateFingerprint`: reject UA /
platform / timezone / vendor values containing control characters or double
quotes before launch, since a value that needs escaping in injected JS is almost
certainly malformed anyway.

## Warnings

### WR-01: `profileFromStealth` doc comment contradicts the code (no-pin path is NOT byte-for-byte default)

**File:** `types/context.go:413-424`
**Issue:** The comment states "With an empty cfg.Stealth identity the result
equals DefaultProfile(), so the no-pin path is byte-for-byte the prior default
behavior (no regression)." Immediately below, line 424 forces
`p.SpoofClientHints = true`, while `DefaultProfile()` (godoll profile.go:79)
ships `SpoofClientHints=false`. The result therefore does **not** equal
`DefaultProfile()`, and the no-pin path now emits Sec-Ch-Ua headers +
`navigator.userAgentData` where the prior default emitted neither. The behavior
change is intentional and explained in the second comment block (lines 417-423),
but the first block is stale and directly contradicts it, which will mislead the
next maintainer reasoning about regression risk.
**Fix:** Update the lines 413-414 comment to say the no-pin path equals
`DefaultProfile()` *except* that Client-Hints spoofing is forced on for
coherence, cross-referencing the lines 417-423 rationale.

### WR-02: AcceptLanguage / header values from profile flow into interceptor headers unvalidated

**File:** `types/context.go:578-582`
**Issue:** `prof.AcceptLanguage`, `prof.UserAgent`, and the platform-derived
`Sec-Ch-Ua-Platform` are placed into the interceptor's `ModifiedHeaders` map
directly from profile-controlled values. A profile value containing a newline or
CR would, depending on godoll's CDP header marshaling, risk header smuggling, or
at minimum produce a malformed header that breaks the request silently. The
Chrome major is digits-only (safe), but the free-text fields are not constrained.
**Fix:** Strip/reject control characters (`\r`, `\n`) from header-bound
fingerprint fields in `deriveAndValidateFingerprint` (shares the CR-02
sanitization pass), so the same validated identity feeds both the JS injection
and the header injection.

### WR-03: User-set `--platform` is unchecked when the UA OS token is unrecognized

**File:** `types/config.go:305-315`
**Issue:** The platform contradiction check only runs when `uaOSToPlatform(ua)`
returns `ok == true`. If the user supplies a UA whose OS token is not one of
Windows/Mac/Linux (e.g. a mobile or bespoke UA), `ok` is false and a user-set
`--platform` is accepted with no coherence check at all — so an arbitrary
platform/UA mismatch (a fingerprint tell) ships silently. This is the
reject-when-user-conflicts policy failing open.
**Fix:** When `ok == false` and the user explicitly set `--platform`, either emit
a stderr warning that platform↔UA coherence could not be verified, or validate
the platform against the known `{Win32, MacIntel, Linux}` set, so an
unverifiable pin is at least surfaced rather than silently shipped.

### WR-04: `pluginsLength` failure reason is hardcoded to "0" regardless of the real value

**File:** `actions/stealth_check.go:188-192`
**Issue:**
```go
case "pluginsLength":
    if n, err := strconv.Atoi(val); err == nil && n > 0 {
        return pass()
    }
    return fail("0")
```
On any non-passing input the reason is always reported as `"0"`, even when the
live value was `-1` (the probe's no-`navigator.plugins` sentinel,
probe.js:46) or a non-numeric/`undefined`. The `--raw` and human output then
misreport the actual observed value, undermining the command's diagnostic value.
**Fix:** Use the real value in the failure reason: `return fail(val)` (mirroring
the other numeric/string verdicts).

### WR-05: `Vendor` cannot be reconciled by the consistency validator

**File:** `types/config.go:294-345` / `types/context.go:458-460`
**Issue:** `profileFromStealth` overlays `Vendor` and godoll injects it into
`navigator.vendor`, but `deriveAndValidateFingerprint` never checks it for
coherence with the platform/UA (e.g. a Windows/Mac identity should carry
`"Google Inc."`; a non-Chrome vendor on a Chrome UA is a tell). A profile can
therefore pin a contradictory vendor that ships with no warning, partially
defeating the "single coherent identity" goal of the phase. Lower severity than
CR-02 because it is a coherence gap, not an injection or crash.
**Fix:** Either validate vendor against the platform (warn on mismatch) or drop
the field from the user-pinnable surface and always derive it.

## Info

### IN-01: Pinned non-US UA still ships `en-US` languages and `America/New_York` timezone

**File:** `types/config.go:294-345`
**Issue:** There is no `--languages` flag, and a UA string carries no locale or
timezone signal, so `--user-agent` alone (without a `--profile`) leaves
`Languages`/`AcceptLanguage`/`Timezone` at the DefaultProfile US values. A user
pinning, say, a German-region identity via `--user-agent` will still emit
`en-US,en;q=0.9` and a US timezone — an internal-coherence gap the validator
cannot close from the UA. This is a documented limitation of UA-only
derivation, not a regression, but worth surfacing for users.
**Fix:** Document that locale/timezone pinning requires `--locale`/`--timezone`
or a `--profile`; consider a future locale→languages/timezone derivation table
(godoll already ships `localeToTimezone` in fingerprint_bridge.go).

### IN-02: Duplicated `defaultChromeMajor`/`chromeMajorRe`/`Chrome/(\d+)` across three packages

**File:** `types/config.go:249,406`; godoll `stealth/evasion.go:17,22`; `tests/detection_test.go:47`
**Issue:** The `Chrome/(\d+)` regexp, the `"121"` default-major literal, and the
parse helper are independently re-declared in rod-cli `types`, godoll `stealth`,
and the test file. They currently agree, but three copies of the
FINGERPRINT-02 anchor invite future drift (e.g. bumping the default major in one
place only). Acceptable given the module boundary, but flag for consolidation.
**Fix:** Consider exporting one helper from godoll `stealth` and having rod-cli
consume it, leaving the test to assert against the shared value.

### IN-03: Redundant `--raw`/default output branches in `runClientCommand`

**File:** `cmd.go:99-103`
**Issue:** The `else if c.Bool("raw")` and the final `else` branches both execute
exactly `fmt.Println(msg)` — identical bodies. Harmless, but the `raw` branch is
dead distinction.
**Fix:** Collapse to a single `else { fmt.Println(msg) }`.

### IN-04: `EnsureDaemon` ignores the daemon log-file open error

**File:** `daemon/daemon.go:113`
**Issue:** `logFile, _ := os.OpenFile(...)` discards the error; if the open fails
`logFile` is nil and `cmd.Stdout = logFile` silently drops daemon
stdout/stderr (including the credential-safe diagnostics the 0600 mode is meant
to protect). Pre-existing pattern, not introduced this phase, but the Phase 26
credential-safety narrative leans on this log file, so the swallowed error is
worth noting.
**Fix:** Check the error and fall back to discarding explicitly (or log a
one-line warning) rather than silently nil'ing the writer.

---

_Reviewed: 2026-06-24T09:04:27Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
