# Phase 26: Configurable Fingerprint & Consistency Validator - Pattern Map

**Mapped:** 2026-06-24
**Files analyzed:** 8 (5 rod-cli, 3 godoll + 1 new probe asset)
**Analogs found:** 8 / 8 (every artifact has an in-repo analog)

> Two repos in scope. rod-cli root: `/home/john/go/src/github.com/agenthands/rod-cli`.
> godoll (editable sibling via `replace`): `/home/john/go/src/github.com/agenthands/godoll`.

## File Classification

| File (new/modified) | Repo | Role | Data Flow | Closest Analog | Match |
|---------------------|------|------|-----------|----------------|-------|
| `types/config.go` (new fingerprint fields + consistency validator in `ResolveStealth`) | rod-cli | config/model | transform (resolve+validate) | self — `ResolveStealth()` `config.go:115-150`, `StealthFlags` `:74-81` | exact |
| `cmd.go` (4 new global flags + forward) | rod-cli | route/CLI | request-response | `--proxy`/`--profile` flags `cmd.go:95-97` + forward `:37-38` | exact |
| `cmd.go` (`stealth-check` subcommand) | rod-cli | route/CLI | request-response | `open`/`screenshot` cmds `cmd.go:159-168, 316-329` | exact |
| `daemon/daemon.go` (`stealth-check` dispatch case) | rod-cli | controller | request-response | `executeAction` switch `daemon.go:199-333` | exact |
| `types/context.go:509` (UA-derived Sec-Ch-Ua) | rod-cli | service | transform | `updateInterceptorRules()` `:460-519` | exact (in-place) |
| `internal/detect/*` (extract shared probe for stealth-check) | rod-cli | utility/asset | transform | `detect.js` + `embed.go` `//go:embed` | exact |
| godoll `stealth/evasion.go:200`, `stealth/js.go:168` (kill `121`) | godoll | service/JS | transform | `addClientHints()` `fingerprint/headers.go:334-359` | role-match (reuse builder) |
| godoll `stealth/fingerprint_bridge.go:36` (derive Timezone) | godoll | service | transform | self `FromFingerprint()` `:12-51` | exact (in-place) |
| `tests/detection_test.go` (flip CH KNOWN-RED + add invariant/stealth-check tests) | rod-cli | test | request-response | `evalDetect`/`waitForDetectReady` `:46-74`, subtests `:117-230` | exact |

## Pattern Assignments

### `types/config.go` — new fingerprint fields + consistency validator (config, transform)

**Analog:** self. The Phase 25 substrate explicitly reserves these exact fields (`config.go:25-27`).

**Struct-field pattern** (mirror existing `StealthConfig` fields `config.go:41-51` — yaml+json tags, doc comment, `ProxyAuth` shows the `yaml:"-" json:"-"` no-serialize idiom for sensitive/runtime-only fields). Add `UserAgent, Locale, Timezone, Platform, Screen, AcceptLanguage, Languages, HardwareConcurrency, DeviceMemory, Vendor, SpoofClientHints`.

**Flag-capture pattern** — extend `StealthFlags` `config.go:74-81` with `UserAgent, Locale, Timezone, Platform string`.

**Precedence + validator** — extend `ResolveStealth()` `config.go:115-150`. The three-tier skeleton already exists:
```go
// Tier 2: profile file. A bad load is loud — do NOT swallow and fall back.
if flags.Profile != "" {
    path := resolveProfilePath(flags.Profile)
    if _, err := stealth.LoadProfile(path); err != nil {
        return errors.Wrapf(err, "load stealth profile %q", path)
    }
    cfg.Stealth.ProfilePath = path
}
// Tier 1: CLI flags win over the profile and the defaults.
if flags.Proxy != "" { cfg.Stealth.Proxy = flags.Proxy }
```
- Overlay loaded profile identity fields onto `cfg.Stealth` at Tier 2 (currently noted as "reserved" at `:134-136` — implement now).
- Apply the 4 new flags at Tier 1 (same `if flags.X != ""` shape).
- Add the consistency validator AFTER overlay, BEFORE the `cfg.Proxy` bridge return. Policy: **derive-when-unset, reject-when-user-conflicts** — return a wrapped `errors.New`/`errors.Errorf` naming conflicting fields + remedy (loud-failure idiom already established by the `LoadProfile` wrap at `:131`). The error propagates to `main.go:37-47` and aborts before `NewContext` — fail-fast before browser launch.
- `--user-agent` is the derivation anchor: parse `Chrome/(\d+)` to derive CH version + `userAgentData`; parse OS token → `platform` + `Sec-Ch-Ua-Platform` when those are unset.

**Timezone↔proxy-geo:** warn-only — write to `os.Stderr` (never return error). Use the existing stderr-warning idiom from `cmd.go:55`.

---

### `cmd.go` — 4 new global flags + forwarding (route, request-response)

**Analog:** the Phase 25 `--proxy`/`--profile` flags.

**Flag registration** (mirror `cmd.go:95-97`):
```go
&cli.StringFlag{Name: "proxy", Usage: "route this session through an HTTP or SOCKS5 proxy ..."},
&cli.StringFlag{Name: "profile", Usage: "named stealth profile (name or path to a JSON profile file)"},
```
Add `--user-agent`, `--locale`, `--timezone`, `--platform` in the same `Flags: []cli.Flag{...}` block `cmd.go:86-98`.

**Forwarding into daemon spawn** (mirror `cmd.go:37-38`, the "persistence linchpin" — a flag only sticks if present at spawn):
```go
if c.String("proxy") != "" { flags = append(flags, "--proxy", c.String("proxy")) }
if c.String("profile") != "" { flags = append(flags, "--profile", c.String("profile")) }
```
Append `--user-agent/--locale/--timezone/--platform` identically. These are NOT secrets (unlike `proxy-auth` which goes out-of-band via env `cmd.go:44-46`), so verbatim argv forwarding is correct.

**"Already-running" warning** — extend the `stealthRequested` predicate `cmd.go:53` to include the 4 new flags so a late flag warns to stderr (`cmd.go:54-56`).

> NOTE: the daemon-side flag→`StealthFlags` capture lives in the hidden `daemon` command path (`runDaemonServer`, `cmd.go:118-126`). Locate where `--proxy`/`--profile` are read off the daemon `cli.Context` into `StealthFlags` and add the 4 new reads there (grep `StealthFlags{` / `flags.Profile`).

---

### `cmd.go` — `stealth-check [url]` subcommand (route, request-response)

**Analog:** `open` (optional positional arg) `cmd.go:159-168` + `screenshot` (subcommand-local flags) `cmd.go:316-329`.

**Pattern** (compose both):
```go
{
    Name:  "open",
    Aliases: []string{"goto"},
    Usage: "Navigate to a URL",
    Action: func(c *cli.Context) error {
        url := c.Args().First()
        if url == "" { return fmt.Errorf("URL is required") }
        return runClientCommand(c, daemon.Request{Command: "open", Args: map[string]string{"url": url}})
    },
},
```
For `stealth-check`: `url := c.Args().First()` is OPTIONAL (navigate-first only if given) — pass it in `Args` (empty allowed). Route through `runClientCommand` so the daemon-spawn/flag-forward path is reused. The global `--raw`/`--json` handling already happens in `runClientCommand` `cmd.go:68-75`, so the **command returns one string**; the daemon-side handler is responsible for producing the `--raw` single-line vs `--json` vs human table form. (To know the requested mode daemon-side, forward `raw`/`json` in `Args`, mirroring how `fill` forwards `submit` as a string `cmd.go:237`.)

---

### `daemon/daemon.go` — `stealth-check` dispatch (controller, request-response)

**Analog:** `executeAction` switch `daemon.go:199-333`; `Request{Command, Args map[string]string}` / `Response{Result, Error}` `daemon.go:34-42`.

**Pattern** (add a case mirroring `eval` `:238-239` / `snapshot` `:240-241`):
```go
case "eval":
    return actions.Evaluate(ctx, req.Args["script"], req.Args["ref"])
case "snapshot":
    return actions.Snapshot(ctx)
```
Add `case "stealth-check": return actions.StealthCheck(ctx, req.Args["url"], req.Args["raw"]=="true", req.Args["json"]=="true")` and implement `StealthCheck` in the `actions` package. Inside it: inject the extracted probe via `page.Eval`/`EvalOnNewDocument`, read `window.__detect` back from the LIVE page (carried-forward "never read from a Go config field" rule, CONTEXT `code_context`), apply the VALIDATE-01 thresholds, and format per requested mode. Returning `(string, error)` matches every other action; the `--raw`/`--json` string is the `Result`.

---

### `types/context.go:509` — UA-derived Client-Hints (service, transform)

**Analog:** in-place `updateInterceptorRules()` `:460-519`. The hardcoded literal:
```go
headers["Sec-Ch-Ua"] = "\"Not A(Brand\";v=\"99\", \"Google Chrome\";v=\"121\", \"Chromium\";v=\"121\""
headers["Sec-Ch-Ua-Mobile"] = "?0"
headers["Sec-Ch-Ua-Platform"] = chPlatform   // already derived from prof.Platform via the switch at :501-508
```
The platform mapping (Win32→"Windows", MacIntel→"macOS") at `:501-508` is the keep-and-reuse derivation shape; only the version string is hardcoded. Replace the `Sec-Ch-Ua` line with a value built from `Chrome/(\d+)` parsed off `prof.UserAgent`. **Prefer calling godoll's builder** (see below) rather than re-implementing the brand string here — converge on the single UA-derived injector (CONTEXT FINGERPRINT-03: "eliminate the duplicate hardcoded path").

---

### godoll — kill `121`, reuse the version→sec-ch-ua builder (service/JS, transform)

**Reuse target (the builder to converge on):** `addClientHints()` `fingerprint/headers.go:334-359`:
```go
func (hg *HeaderGenerator) addClientHints(headers map[string]string, browser string, version versionEntry, os string) {
    majorStr := strconv.Itoa(version.Major)
    brandHint := Sample(hg.rng, notABrandHints)
    browserBrand := "Google Chrome"
    ...
    headers["sec-ch-ua"] = fmt.Sprintf(`"%s";v="%s", "%s";v="%s", "Chromium";v="%s"`,
        brandHint.NotABrand, brandHint.NotABrandVer, browserBrand, majorStr, majorStr)
    headers["sec-ch-ua-mobile"] = "?0"
    ...
    headers["sec-ch-ua-platform"] = `"` + chPlatform + `"`
}
```
This already takes a `versionEntry.Major` and `os` token — the exact UA-derived inputs FINGERPRINT-03 wants. The `121` fallback default is at `headers.go:264` (`versionEntry{Major: 121, ...}`) — that is a *no-version-map* fallback, acceptable to leave or to make UA-derived; the runtime injectors below are the hard `121` tells to remove.

**Hardcoded literal #1** — `stealth/evasion.go:200` (HTTP header injector), same shape as the rod-cli interceptor:
```go
ctx.Request.Req().Header.Set("Sec-Ch-Ua", "\"Not A(Brand\";v=\"99\", \"Google Chrome\";v=\"121\", \"Chromium\";v=\"121\"")
```
Replace with a value derived from `em.profile.UserAgent` (parse `Chrome/(\d+)`); ideally route through the same brand-string formatter used by `addClientHints` so all surfaces share one source.

**Hardcoded literal #2** — `stealth/js.go:168` (`navigator.userAgentData` brand version) in `scriptMockUserAgentData(platform)`:
```go
brands: [
    { brand: "Not A(Brand", version: "99" },
    { brand: "Google Chrome", version: "121" },
    { brand: "Chromium", version: "121" }
],
```
Add a major-version parameter (derive from the profile UA) so the JS brand version == the header CH major == the UA Chrome major (that triple-agreement IS the FINGERPRINT-02 invariant). Note `platform` is already parameterized here — mirror that for `version`.

**Timezone derivation** — `stealth/fingerprint_bridge.go:36` `FromFingerprint()`:
```go
// Timezone default
p.Timezone = "America/New_York"
```
Replace the hardcoded default: derive from the fingerprint's locale/headers when available (the function already derives `Locale` from `Languages[0]` at `:31-33` and `AcceptLanguage` from headers at `:24-28` — same defensive `if ok {} else { default }` idiom applies to timezone).

---

### `internal/detect/*` — extract shared probe for stealth-check (utility/asset, transform)

**Analog:** `detect.js` (the `window.__detect` probe, `probe(name, fn)` swallow-into-value idiom `detect.js:22-28`, table-stakes signals `:30-93`, async `ready` settle `:132-241`) + `embed.go` `//go:embed` + `types/js/js.go` embed-as-`var` pattern:
```go
//go:embed snapshotter.js
var InjectedSnapShot string
```
**Approach:** extract the table-stakes synchronous subset (`webdriver`, `pluginsLength`, `userAgent`, `webglVendor`/`Renderer`, `languages`, `screen`, `windowChrome`/`chromeRuntime`, `timezone` — `detect.js:30-93`) plus the async permissions probe into a reusable snippet, `//go:embed` it as a `string` var (mirror `embed.go` lines 8-9), and inject it via `page.Eval` from the `StealthCheck` action. Do NOT author a second probe — share so the command and the harness agree. Keep the offline `detect.js`/`server.go` fixture intact for the harness.

**Verdict thresholds (VALIDATE-01, codify in the action):** `webdriver==false`, `pluginsLength>0`, UA has no `HeadlessChrome`, WebGL vendor/renderer not SwiftShader/llvmpipe/software (assertion already encoded in `detection_test.go:117-129`), `permissionsConsistent`, `languages` present, plausible screen dims, `windowChrome`/`chromeRuntime` present, `timezone` set.

---

### `tests/detection_test.go` — flip KNOWN-RED + add invariant/stealth-check tests (test, request-response)

**Analog:** self. Helpers `evalDetect` `:46-57` (reads a single signal from the LIVE page), `waitForDetectReady` `:62-74`, subtest shape `t.Run("name", func(t *testing.T){...})` (e.g. `webgl_not_software` `:117-129`).

**Flip the KNOWN-RED** at `:202-230`: replace the "record-don't-fail" `t.Logf` baseline with a hard assertion that the `Sec-Ch-Ua` major == UA `Chrome/(\d+)` major == `userAgentData` brand version. The header-echo httptest pattern `:210-219` (echo `r.Header` as JSON, `goto`, read `document.body.innerText`) is exactly reusable to read the live `Sec-Ch-Ua`.

**New consistency-invariant subtest (BLOCKING, success criterion 1):** read `Sec-CH-UA`, `navigator.userAgentData`, UA, `navigator.platform`, WebGL via `evalDetect`/header-echo and assert one consistent OS+version story.

**New stealth-check subtest:** drive `runCli("stealth-check")` and `runCli("--raw", "stealth-check")`; assert `--raw` is a single PASS/FAIL line with only failing signals (NOT a full-page dump).

## Shared Patterns

### Loud-failure config resolution
**Source:** `types/config.go:128-137` (`ResolveStealth` Tier 2 — `errors.Wrapf` on bad profile, never swallow).
**Apply to:** the consistency validator (reject-on-conflict returns a wrapped error → `main.go:37-47` aborts before `NewContext`/browser launch).

### Token-efficient channels: stderr for warnings, stdout clean
**Source:** `cmd.go:54-56` (`fmt.Fprintf(os.Stderr, "warning: ...")`); CONTEXT "Established Patterns".
**Apply to:** validator warn-only (timezone↔proxy-geo), `stealth-check` `--raw` (failing signals to stdout single-line, never a dump), conflict-reject remedy message to stderr.

### Flag → daemon-spawn argv forwarding (persistence linchpin)
**Source:** `cmd.go:33-38`. Non-secret flags appended to spawn argv; secrets out-of-band via env `cmd.go:44-46`.
**Apply to:** all 4 new flags (non-secret → verbatim argv). UA could be PII-ish but is not a credential — argv is correct.

### Read assertions from the LIVE page, never from a Go field
**Source:** `tests/detection_test.go:42-57` `evalDetect`; CONTEXT carried-forward v1.5 lesson.
**Apply to:** `StealthCheck` action and every new harness assertion.

### `//go:embed` JS-as-string
**Source:** `internal/detect/embed.go:5-9`, `types/js/js.go:3-6`.
**Apply to:** the extracted shared probe snippet for `stealth-check`.

### dispatch case = thin delegate to actions
**Source:** `daemon/daemon.go:199-333` (each case unpacks `req.Args` and calls one `actions.*` returning `(string, error)`).
**Apply to:** `stealth-check` case.

## No Analog Found

None. Every artifact mirrors an existing Phase 24/25 pattern. The only genuinely-new logic is the consistency-validator decision table and the UA→major-version parse; both have a direct host (`ResolveStealth`) and a reuse target (godoll `addClientHints` / `selectVersion`).

## Metadata

**Analog search scope:** rod-cli `types/`, `daemon/`, `cmd.go`, `internal/detect/`, `tests/`; godoll `stealth/`, `fingerprint/`.
**Files scanned:** 10 read in full/targeted ranges across both repos.
**Pattern extraction date:** 2026-06-24
