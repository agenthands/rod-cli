# v1.7 (Complete Evasion Stack) — rod-cli map at milestone close

Builds on the v1.6 stealth spine (see archaeologist-m1v6-stealth-spine). Three
phases landed inside the SAME spine, all centered on `types/context.go createPage`.
Phase 31 (TLS spoofing) was CANCELLED — real-Chrome-only, NO TLS/JA3 spoofing
(lives in a separate project "munch"); authentic Chrome TLS is treated as a
strength, not a gap.

## Phase 30 — CDP footprint reduction (footprint follows the feature)
- Console + request capture are now OPT-IN: `--console-capture`/`--request-capture`,
  default OFF (`ConsoleCapture`/`RequestCapture` *bool in StealthConfig/StealthFlags,
  resolved default false). When off, the Runtime/Network event subscriptions are
  never registered.
- Network interceptor is LAZY: `ensureInterceptorEnabled(page)` creates+enables it
  (godoll Fetch.enable) on first `AddRoute`; `RemoveRoute` of the last route
  disables+discards it.
- HTTP identity moved to zero-enable `Emulation.setUserAgentOverride`
  (`applyEmulationIdentity`) — replaces the old always-on Fetch interceptor
  catch-all identity rule (`updateInterceptorRules` now only carries Mock rules).
  The old X-Requested-With strip is gone (Chrome never emits it natively).
- Per-session CDP-domain ledger: `recordCDPDomainLocked(domain)` (caller holds
  stateLock) + `GetEnabledCDPDomains()`; consts `CDPDomainRuntime`/`Network`/`Fetch`.
  A plain `goto` records NONE — gate: `types/cdp_footprint_test.go`
  `TestCDPFootprintBaseline`. WIRE-VERIFY: `tests/network_evasion_test.go`
  `TestNetworkEvasionHeaders`.
- KNOWN GAP: plugin lifecycle binder (`internal/plugin/lifecycle.go BindLifecycle`)
  calls `proto.DOMEnable` + subscribes NetworkRequestWillBeSent UNCONDITIONALLY on
  plugin load WITHOUT recordCDPDomainLocked → ledger under-reports for plugin
  sessions. SMTC dominance-spec candidate for v2.

## Phase 32 — Built-in profile library
- 6 embedded Chrome-only profiles `types/profiles/*.json` via `//go:embed` in
  `types/profiles_embed.go`: windows-11-chrome, windows-11-desktop-1440p,
  windows-10-chrome (HW=12!), windows-10-laptop, macos-applesilicon-chrome,
  macos-intel-chrome.
- `BuiltinProfileNames()`, `LoadBuiltinProfile(name)`, `isBuiltinProfile`.
- `--profile=list` (cmd.go maybeHandleProfileList, honors --raw/--json).
- Resolution (config.go loadSelectedProfile/profileLooksLikePath): explicit path
  (separator or .json) loads verbatim; else bare name → built-in FIRST → then
  ~/.rod-cli/profiles/<name>.json. Built-in names are RESERVED for bare selection.
  ProfilePath label = "builtin:<name>" sentinel (NOT a real path).
- Vetting gate: `types/profiles_test.go TestBuiltinProfilesAreVetted` (PROF-02).

## Phase 33 — Advanced evasion (activate godoll dormant dimensions)
- createPage: `em.SetFingerprint(fp)` then `em.SetProfile(prof)` — ORDER MATTERS:
  SetFingerprint derives+overwrites em.profile (FromFingerprint), SetProfile must
  run AFTER to restore the config-pinned identity. fp generated via
  `rodfingerprint.NewFingerprintGeneratorSeeded(int64(noiseSeed), FPWithBrowserNames("chrome"),
  FPWithOS(osForPlatform(platform)), FPWithLocales(locale))` — OS/locale-coherent,
  session-stable.
- `em.SetDimensionOptions(stealth.DimensionOptions{Fonts,MediaDevices,Battery,Codecs,Plugins})`
  gated by 4 new *bool toggles `--font-spoof`/`--media-devices-spoof`/`--battery-spoof`/
  `--codec-spoof` (default ON; CLI>profile>default; in DefaultConfig + ResolveStealth).
- KNOWN GAP: godoll `stealth.scriptMockFonts` is an OBSERVABLE NO-OP — overrides
  measureText but returns the original width in every branch
  (../godoll/stealth/fingerprint_bridge.go ~245). So --font-spoof gates injection
  but doesn't change live font readout. Documented honestly in stealth-config.md.
  media-devices is the harness-proven leg.

## Other tracked concern
- Go toolchain skew: go.mod `go 1.25.1`, no toolchain directive; CI test.yml 1.25.x;
  release*.yml 1.23/1.23.7; dev machine go1.26.0. v1.7 security F1 = bump to
  CVE-patched go1.26.1 + align. Build/release hygiene, not a code defect.

## Docs I own — verified current @ HEAD (v1.7 close)
- root ARCHITECTURE.md (fixed v1.7 drift: §2 said console/request enable
  Runtime/Network — now opt-in/off; added Emulation + lazy interceptor + ledger;
  built-in profile precedence; dimension toggles).
- README.md (added missing docs/cdp-footprint.md link to the Documentation index).
- docs/stealth-config.md (fixed: windows-10-chrome HW cores 8->12; title v1.6->v1.7;
  added cdp-footprint cross-link), docs/cdp-footprint.md, docs/stealth-validation.md
  (all accurate at HEAD).
- .planning/codebase/*.md refreshed to v1.7.

## Drift found + fixed this close
1. docs/stealth-config.md §8: windows-10-chrome "8 cores" -> actual JSON is 12.
2. ARCHITECTURE.md §2: "console/request logging ... which is why Runtime/Network
   domains are enabled" -> STALE; now opt-in (default OFF), plain session enables none.
3. README Documentation index omitted docs/cdp-footprint.md -> added.
