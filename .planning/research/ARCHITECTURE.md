# Architecture Research

**Domain:** Brownfield integration — configurable stealth + a detection test harness into an existing Go CLI (rod-cli, persistent-daemon model)
**Researched:** 2026-06-24
**Confidence:** HIGH (grounded in the actual source: `types/context.go`, `actions/actions.go`, `types/config.go`, `cmd.go`, `main.go`, `daemon/daemon.go`, `internal/plugin/scanner/testserver/`, and the vendored `../godoll` packages)

> This is an integration study, not a greenfield design. The verb is **integrate**, not redesign. Every recommendation names the exact file and the existing seam it plugs into. Where godoll already exposes a capability, the work is *wiring*, not building.

---

## 1. The Existing System (ground truth)

### Process model

```
┌──────────────────────────────────────────────────────────────────────┐
│ CLIENT INVOCATION  (rod-cli open <url>)                                │
│   main.go → getApp() (cmd.go) → command Action → runClientCommand()    │
│     • packs global flags into []string                                 │
│     • daemon.EnsureDaemon(session, exe, flags)  ── spawns if absent    │
│     • daemon.ClientExecute(session, Request{Command,Args})  ── HTTP    │
└───────────────────────────────┬──────────────────────────────────────┘
                                 │  re-exec:  rod-cli --session S <flags> daemon
                                 ▼
┌──────────────────────────────────────────────────────────────────────┐
│ DAEMON PROCESS  (long-lived, per named session)                        │
│   main.go runDaemonServer():                                           │
│     cfg = LoadConfig(--config)  →  override with flags  →  *Config     │
│     rodCtx = types.NewContext(ctx, *cfg)   ◄── config frozen here      │
│   daemon.StartServer(session, ppid, rodCtx):                           │
│     HTTP /execute → executeAction(rodCtx, req) → actions.*             │
│     idle 15min timeout · PPID poll · port file in os.TempDir()         │
└───────────────────────────────┬──────────────────────────────────────┘
                                 ▼
┌──────────────────────────────────────────────────────────────────────┐
│ types.Context  (the session state struct — the integration center)     │
│   config Config · browser *rod.Browser · page *rod.Page                │
│   interceptor *network.Interceptor · fingerprint *Fingerprint · routes │
│   launchBrowser(cfg)  ─ proxy/headless/launcher (lines 32-86)          │
│   createPage()        ─ EvasionManager.Apply() + interceptor (281-327) │
│   updateInterceptorRules() ─ header spoofing from Profile (342-401)    │
└──────────────────────────────────────────────────────────────────────┘
```

### The two evasion seams that already exist

| Seam | Location | What it owns today | Stealth feature it gates |
|------|----------|--------------------|--------------------------|
| `launchBrowser(ctx, cfg)` | `types/context.go:32-86` | launcher flags, `Headless`, `NoSandbox`, **`cfg.Proxy` (already wired, line 71-73)**, `StealthPreset()` browser prefs | proxy, headless, browser-level WebRTC pref |
| `createPage(urls...)` | `types/context.go:281-327` | builds `fingerprint`, `EvasionManager`, calls `em.Apply()`, injects snapshot JS, wires `network.NewInterceptor` | fingerprint, canvas/WebGL/audio, navigator spoof |
| `updateInterceptorRules()` | `types/context.go:342-401` | derives `stealth.Profile` from fingerprint, sets UA / Accept-Language / Client-Hints headers | header consistency |

**Config persistence mechanism (critical):** `Config` (`types/config.go`) is loaded once per daemon, then `NewContext(ctx, *cfg)` freezes it for the daemon's whole life. Every subsequent client command hits the same in-memory `rodCtx` — so **any stealth setting placed in `Config` and passed at daemon-spawn time is automatically session-persistent across commands.** This is the existing daemon-persistence backbone; v1.6 must ride it, not invent a new one.

---

## 2. godoll surface already available (wiring, not building)

Confirmed by reading `../godoll`:

| Need | godoll API (exists) | Currently wired in rod-cli? |
|------|---------------------|------------------------------|
| Proxy | `launcher.Proxy()` + `cfg.Proxy` | **Yes in launch, NO flag** — `--proxy` missing from `cmd.go` |
| Pinned fingerprint | `fingerprint.FPWithBrowserNames/FPWithOS/FPWithLocales(...)` Options; `stealth.Profile{UserAgent,Platform,Timezone,Locale,Screen,...}`; `stealth.LoadProfile(path)` / `Profile.Save(path)` | Partial — `createPage` always *generates random* chrome FP; no pinning |
| Canvas/WebGL | `WebGLManager.MockVendorRenderer`, `ApplyCanvasNoise()` — **called inside `em.Apply()`** | Yes (on by default via Apply) |
| WebRTC leak | `EvasionManager.EvadeWebRTC()` **and** `BrowserOptions.WithWebRTCLeakProtection(bool)` | **NO** — `EvadeWebRTC` is *not* called by `Apply()` (confirmed: commented-out region in `evasion.go`); a real gap |
| Audio noise | `Profile.SpoofAudioContext` gates `scriptAudioContextNoise` in `Apply()` | Off by default (profile field false) |
| Humanize tuning | `WithTypingSpeed(min,max)`, `WithTypoRate(r)`, `WithMouseSpeed/Steps/Tremor/Deviation`, `ScrollBy(...WithPhysics/WithDuration)` | **NO** — `actions.go` calls `typeWithHumanize`/`clickWithMouse`/`humanizeScrollBy` with **no options** |
| Cloudflare best-effort | `stealth.NewCloudflareBypass(page).Bypass(timeout)` | NO (out of blocking CI per PROJECT.md) |

**Implication:** Most v1.6 "features" are option-passing, not new algorithms. The bulk of the work is a *config plumbing layer* + a *test harness* to prove it.

---

## 3. Recommended Integration Design

### 3.1 New component: `StealthConfig` (the config surface)

Add a struct that aggregates all stealth knobs, embed it in `Config`, and thread it through the two existing seams.

```go
// types/config.go  (MODIFY — add type + field)
type StealthConfig struct {
    Profile     string `yaml:"profile" json:"profile"`         // named profile to load (file layer)
    UserAgent   string `yaml:"userAgent" json:"userAgent"`
    Locale      string `yaml:"locale" json:"locale"`
    Timezone    string `yaml:"timezone" json:"timezone"`
    Platform    string `yaml:"platform" json:"platform"`
    Screen      string `yaml:"screen" json:"screen"`            // "1920x1080"
    WebRTC      bool   `yaml:"webrtcLeakProtection" json:"webrtcLeakProtection"`
    CanvasNoise bool   `yaml:"canvasNoise" json:"canvasNoise"`
    AudioNoise  bool   `yaml:"audioNoise" json:"audioNoise"`
    // humanize tuning
    TypingMinMs int     `yaml:"typingMinMs" json:"typingMinMs"`
    TypingMaxMs int     `yaml:"typingMaxMs" json:"typingMaxMs"`
    TypoRate    float32 `yaml:"typoRate" json:"typoRate"`
    MouseSpeedMin int   `yaml:"mouseSpeedMin" json:"mouseSpeedMin"`
    MouseSpeedMax int   `yaml:"mouseSpeedMax" json:"mouseSpeedMax"`
}

type Config struct {
    // ...existing fields...
    Proxy   string        `yaml:"proxy" json:"proxy"`   // already present
    Stealth StealthConfig `yaml:"stealth" json:"stealth"` // ADD
}
```

`Proxy` already lives on `Config` and is already consumed by `launchBrowser` — only the **flag** is missing.

### 3.2 New component: profile loader / precedence resolver

```
flag  >  named profile file  >  Config(rod-cli.yaml)  >  godoll default
```

A small resolver (`types/stealth_profile.go`, NEW) materialises the effective `stealth.Profile` + humanize options *before* `NewContext`:

```go
// Called in main.go runDaemonServer, after LoadConfig, before NewContext.
func ResolveStealth(cfg *Config, flags StealthFlags) {
    // 1. start from godoll default
    p := stealth.DefaultProfile()
    // 2. overlay named profile file if cfg.Stealth.Profile != ""
    if cfg.Stealth.Profile != "" {
        if fp, err := stealth.LoadProfile(profilePath(cfg.Stealth.Profile)); err == nil { p = *fp }
    }
    // 3. overlay cfg.Stealth (rod-cli.yaml)   4. overlay non-zero flags
    // write the merged result back into cfg.Stealth so Context owns one source of truth
}
```

**Why resolve at daemon boot, not per-command:** the daemon freezes `Config` at `NewContext`. Resolving precedence once at spawn time means every later command on that session inherits the same stealth identity for free — matching the "session-persistent CLI flags" requirement with zero per-request state.

### 3.3 Threading into the two seams (MODIFY context.go)

**`launchBrowser` (lines 32-86):** already applies `cfg.Proxy`. Add one line for browser-level WebRTC:
```go
opts := browser.NewBrowserOptions().
    WithLauncher(browserLauncher).
    WithWebRTCLeakProtection(cfg.Stealth.WebRTC).   // ADD
    SetBrowserPreferences(...StealthPreset())
```

**`createPage` (lines 281-327):** replace the unconditional random-chrome generator with a config-driven one, and call the missing WebRTC JS evasion:
```go
// build generator from pinned config instead of always-random
fgOpts := []fingerprint.Option{fingerprint.FPWithBrowserNames("chrome")}
if cfg.Stealth.Locale != "" { fgOpts = append(fgOpts, fingerprint.FPWithLocales(cfg.Stealth.Locale)) }
if cfg.Stealth.Platform != "" { fgOpts = append(fgOpts, fingerprint.FPWithOS(osFromPlatform(cfg.Stealth.Platform))) }
fp, _ := fingerprint.NewFingerprintGenerator(fgOpts...).Generate()

em := stealth.NewEvasionManager(page)
em.SetFingerprint(fp)
// overlay explicit pins (UA/TZ/screen) onto the derived profile via SetProfile
em.Apply()
if cfg.Stealth.WebRTC {
    em.EvadeWebRTC()   // ADD — Apply() does NOT call this; it is the documented gap
}
```
The existing `updateInterceptorRules()` already reads `ctx.fingerprint` to set UA/Accept-Language/Client-Hints headers, so pinned identity propagates to headers automatically — no change needed there beyond ensuring the pinned profile reaches `ctx.fingerprint`.

### 3.4 Threading into actions (MODIFY actions.go)

`actions.go` currently calls humanize with no options. The cleanest, lowest-churn path: have the action functions read tuning from the `*types.Context` they already receive, and pass options through. Add accessors on `Context` (`HumanizeTypingOpts()`, `HumanizeMouseOpts()`, `HumanizeScrollOpts()`) that translate `cfg.Stealth` into `[]humanize.TypingOption` etc.

```go
// actions.go — Fill/Type
typeWithHumanize(element, value, ctx.HumanizeTypingOpts()...)
// Click/DblClick
clickWithMouse(page, element, ctx.HumanizeMouseOpts()...)
// MouseWheel
humanizeScrollBy(page, dir, n, ctx.HumanizeScrollOpts()...)
```
The package-level seam vars (`typeWithHumanize`, `clickWithMouse`, `humanizeScrollBy`) are already variadic-compatible — they wrap the godoll funcs which take `...Option`. This preserves the existing test-seam pattern.

### 3.5 CLI flag surface (MODIFY cmd.go + main.go + daemon.EnsureDaemon)

Two integration tasks:

1. **Add flags** to `getApp()` global `Flags` (and/or to `open`/`goto`): `--proxy`, `--profile`, `--user-agent`, `--locale`, `--timezone`, `--platform`, `--screen`, `--webrtc`, `--canvas-noise`, `--typing-speed`, `--typo-rate`, `--mouse-speed`.
2. **Propagate them across the daemon boundary.** `runClientCommand` (cmd.go:16-46) currently forwards only `--config/--cdp-endpoint/--headless/--vision`. Extend that flag-packing block to forward every stealth flag, and have `runDaemonServer` (main.go) apply them onto `cfg.Stealth` before `NewContext`. **This is the persistence linchpin** — a flag only "sticks" for the session if it is forwarded at spawn time.

> Gotcha: flags only take effect when the daemon is *first* spawned for a session. If a daemon is already alive, new stealth flags are ignored (same as today's `--headless`). Document this and/or detect a config-mismatch ping. Roadmap should flag this as a UX decision, not silently change identity mid-session.

---

## 4. The Detection Test Harness (mirror the testserver pattern)

### 4.1 Two real patterns to mirror

- **In-process server:** `internal/plugin/scanner/testserver/server.go` — `New()` binds `127.0.0.1:0`, `Start()` serves in a goroutine, `URL()` returns the base, `Close()` tears down. Deterministic, no external network.
- **Live-binary driver:** `tests/cli_test.go` — `runCli(args...)` does `exec.Command("../rod-cli", ...)`, drives the real daemon over the real CLI, asserts on stdout. `SetupTestServer()` provides a local target.

The detection harness is the *intersection* of these two: a local detection server **plus** a test that drives the real binary against it.

### 4.2 Placement

```
internal/detect/testserver/        # NEW — mirrors plugin/scanner/testserver
    server.go                       # serves bundled detection pages
    fixtures/                       # NEW — vendored CreepJS/sannysoft-style probe HTML+JS
        webdriver.html              # asserts navigator.webdriver === undefined
        plugins.html                # navigator.plugins length / mimeTypes
        useragent.html              # no "HeadlessChrome" token
        webgl.html                  # UNMASKED_VENDOR_WEBGL value
        permissions.html            # Notification.permission vs query() consistency
        webrtc.html                 # local-IP leak probe
    server_test.go                  # unit tests for the server itself (mirror existing)
tests/detection_test.go            # NEW — e2e: build binary, run real daemon, assert
```

**Why bundle local probe pages instead of hitting bot.sannysoft.com:** determinism + offline CI. The local fixtures are the *blocking* CI gate. Real third-party suites (CreepJS, fingerprintjs) and WAF challenges (Cloudflare/DataDome) are **non-blocking, opt-in** (build tag `//go:build detection_live` or an env gate), per PROJECT.md.

### 4.3 How the e2e test drives the live binary deterministically

```go
// tests/detection_test.go (sketch)
func TestStealth_NoWebdriverTell(t *testing.T) {
    ds := detecttestserver.New(); ds.Start(); defer ds.Close()
    runCli("close")                                  // clean daemon state
    runCli("goto", ds.URL()+"/webdriver.html")       // real daemon boots stealth browser
    out, _ := runCli("eval", "navigator.webdriver")  // reuse existing eval action
    if strings.Contains(out, "true") { t.Fatal("webdriver tell leaked") }
}
```

Each probe page writes its verdict into the DOM / a known global; the test reads it with the **already-existing `eval` and `snapshot` commands** — no new browser plumbing. The harness asserts purely through the public CLI surface, which is exactly what proves the *shipped binary* is stealthy (not just a unit of godoll).

**CI integration:** add a `test` job to `.github/workflows/` (today only `release.yml` exists — there is **no test CI job**, a gap to close). The job must `rod-cli install` (or use the GOCOVERDIR-coverage build the daemon already supports, see `daemon.exitDaemon`) then `go test ./...`. The blocking suite uses local fixtures only; `-tags detection_live` runs the best-effort external suite on a manual/nightly trigger.

### 4.4 Determinism levers

| Source of flake | Mitigation |
|-----------------|------------|
| Random fingerprint per page | Pin a profile in the test (`--profile ci`) so assertions are stable |
| Daemon reuse across tests | `runCli("close")` before each test (existing convention) |
| Humanize delays slowing CI | Pin `--typing-speed 1,2` / disable physics in the harness profile |
| Browser download in CI | Cache Chromium; `rod-cli install` step |

---

## 5. New vs Modified — explicit inventory

### New files/components

| Path | Purpose |
|------|---------|
| `internal/detect/testserver/server.go` (+`server_test.go`) | bundled detection probe server (mirror plugin/scanner/testserver) |
| `internal/detect/testserver/fixtures/*.html` | deterministic detection probe pages |
| `tests/detection_test.go` | e2e: real binary vs local probes, blocking CI |
| `tests/detection_live_test.go` (`//go:build detection_live`) | best-effort CreepJS/WAF, non-blocking |
| `types/stealth_profile.go` | `StealthConfig` resolver + precedence + profile file load/save |
| `.github/workflows/test.yml` | **first test CI job** for the repo |

### Modified files

| File | Change |
|------|--------|
| `types/config.go` | add `StealthConfig` type + `Config.Stealth`; default values |
| `types/context.go` | `launchBrowser`: `WithWebRTCLeakProtection`; `createPage`: config-driven fingerprint + `EvadeWebRTC()`; add `HumanizeTypingOpts/MouseOpts/ScrollOpts` accessors |
| `actions/actions.go` | pass `ctx.Humanize*Opts()...` into `typeWithHumanize`/`clickWithMouse`/`humanizeScrollBy` |
| `cmd.go` | add stealth flags; **forward them in `runClientCommand`'s flag-packing block** |
| `main.go` | `runDaemonServer`: apply stealth flags onto `cfg.Stealth`, call `ResolveStealth` before `NewContext` |
| `daemon/daemon.go` | (only if `EnsureDaemon` needs new flags in its spawn arg list — it forwards `flags` verbatim, so likely no change) |

---

## 6. Data flow (CLI flag → session config → godoll option)

```
rod-cli --profile us --proxy socks5://… --webrtc open <url>
        │
   cmd.go Action → runClientCommand
        │  pack ALL stealth flags into []string   ◄── NEW forwarding
        ▼
   daemon.EnsureDaemon(session, exe, flags)
        │  exec: rod-cli --session S --profile us --proxy … --webrtc daemon
        ▼
   main.go runDaemonServer
        cfg = LoadConfig()                         (rod-cli.yaml layer)
        ResolveStealth(cfg, flags):                ◄── precedence merge
            default → profile-file(us) → yaml → flags
        cfg.Stealth / cfg.Proxy now authoritative
        rodCtx = NewContext(ctx, *cfg)             ◄── frozen for session
        ▼
   types.Context (per-session, persists across every later command)
        launchBrowser(cfg):  Proxy(cfg.Proxy) · WithWebRTCLeakProtection(cfg.Stealth.WebRTC)
        createPage():        FPWith*(cfg.Stealth.*) → em.SetFingerprint → em.Apply()
                             em.EvadeWebRTC() if cfg.Stealth.WebRTC
        actions.*:           humanize.WithTypingSpeed/WithMouseSpeed from cfg.Stealth
```

The key insight: **there is no per-command stealth state.** Everything resolves once at daemon boot and is frozen on `Context`. That is both the persistence guarantee and the constraint (mid-session changes require daemon restart).

---

## 7. Suggested build order (dependency-respecting, harness-first)

1. **Detection harness — server + fixtures + e2e skeleton + test CI job.**
   Rationale: every later phase needs something to assert against; today the repo has *no* test CI at all. Build the harness against the *current* binary first to establish a baseline (it will reveal which tells already pass via `StealthPreset`/`Apply` and which leak — e.g. WebRTC, which `Apply()` does not handle).

2. **Config surface — `StealthConfig`, precedence resolver, profile file load/save, flag forwarding across the daemon boundary.**
   Rationale: it is the substrate every feature flag rides; build before the features so each feature lands behind a real, testable flag. The daemon-boundary forwarding is the riskiest plumbing — do it early with the harness watching.

3. **Proxy flag** (smallest: `cfg.Proxy` is already consumed; just expose `--proxy` + forward it). Quick win that validates the whole flag→daemon→godoll path end-to-end.

4. **Configurable fingerprint** (UA/locale/timezone/screen/platform pinning in `createPage`). Validate against harness webdriver/useragent/webgl probes.

5. **Canvas/WebGL/WebRTC hardening** — the only one needing a genuine godoll-gap fix (`EvadeWebRTC()` call + `WithWebRTCLeakProtection`). Validate against the webrtc/webgl probes.

6. **Humanize tuning** — thread options through `actions.go`. Lowest detection-risk, mostly UX; validate that pinned CI speeds keep the suite fast.

7. **Best-effort live validation** (CreepJS / Cloudflare / DataDome) behind `//go:build detection_live`, non-blocking. Last, because it depends on all prior features being in place and is explicitly out of blocking CI.

---

## 8. Anti-patterns to avoid in this integration

**Per-command stealth state.** Don't add stealth args to every `Request`/`executeAction`. The daemon freezes `Config`; resolve once at boot. Adding per-request identity would let an agent accidentally change fingerprint mid-session and *create* a detectable inconsistency.

**A second config-load path.** Don't bypass `LoadConfig`/`Config`. Everything must funnel through `Config.Stealth` so the existing daemon-spawn forwarding is the single persistence mechanism.

**Hitting live bot.sannysoft.com in blocking CI.** Non-deterministic, network-dependent, rate-limited. Bundle local probe pages for the gate; keep external suites opt-in.

**Re-implementing evasions.** godoll already ships canvas/WebGL/audio/WebRTC/header logic. The gap is *calling* `EvadeWebRTC()` (Apply omits it) and *passing* humanize options — wire, don't rewrite.

**Bypassing the public CLI in the e2e harness.** Assert through `eval`/`snapshot` against the real daemon (like `tests/cli_test.go`), not against godoll directly — that is what proves the shipped binary, which is the milestone's whole point ("provably evades").

---

## 9. Integration points summary

### Internal boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| cmd.go ↔ daemon | flag-packing in `runClientCommand` → re-exec args | **must extend** to carry stealth flags; sole persistence path |
| main.go ↔ Context | `Config` frozen at `NewContext` | resolve precedence *before* this call |
| Context ↔ godoll | `launchBrowser` + `createPage` seams | only two places evasion is configured |
| actions ↔ godoll | `typeWithHumanize`/`clickWithMouse`/`humanizeScrollBy` seam vars | already variadic-ready for `...Option` |
| harness ↔ binary | `exec.Command("../rod-cli")` + local httptest server | mirror `tests/cli_test.go` + `testserver` |

### External / capability points

| Capability | godoll API | Status |
|------------|-----------|--------|
| Proxy | `launcher.Proxy` | wired; flag missing |
| Fingerprint pin | `FPWith*` + `Profile` + `LoadProfile/Save` | needs config-driven generator |
| WebRTC | `EvadeWebRTC()` / `WithWebRTCLeakProtection` | **gap: Apply() omits EvadeWebRTC** |
| Canvas/WebGL/Audio | `WebGLManager` / `SpoofAudioContext` | in Apply(); audio off by default |
| Humanize | `WithTypingSpeed/TypoRate/MouseSpeed/...` | needs option threading |

## Sources

- rod-cli source (read directly): `types/context.go`, `types/config.go`, `actions/actions.go`, `cmd.go`, `main.go`, `daemon/daemon.go`, `internal/plugin/scanner/testserver/{server,server_test}.go`, `tests/cli_test.go`, `.github/workflows/release.yml` — HIGH confidence (primary source).
- godoll vendored at `../godoll` (read directly): `stealth/{evasion,profile,webgl}.go`, `fingerprint/*.go`, `humanize/*.go`, `browser/*.go` — HIGH confidence (primary source). Notable: `EvasionManager.Apply()` does **not** call `EvadeWebRTC()` (confirmed via commented-out region in `stealth/evasion.go`).
- `.planning/PROJECT.md` — milestone intent (blocking vs non-blocking CI scope, target features).

---
*Architecture research for: brownfield stealth-config + detection-harness integration into rod-cli*
*Researched: 2026-06-24*
