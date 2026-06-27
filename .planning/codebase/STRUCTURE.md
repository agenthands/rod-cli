# Structure

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)
**Refreshed:** 2026-06-26 (milestone v1.7 close)
**Refreshed:** 2026-06-26 (milestone v2.0 close — author: architect)
**Refreshed:** 2026-06-27 (milestone v2.2 close — author: codebase-archaeologist)

> The original layout (`tools/`, `server.go`, `runner.go`) was stale; HEAD uses
> `actions/`, `daemon/`, and `internal/`.

## Directory Layout

- `/` (root)
  - `main.go`: `runDaemonServer` (daemon entry — loads config, resolves stealth,
    serves) + `main()`.
  - `cmd.go`: the `urfave/cli/v2` app — global flags (incl. the v1.6 stealth
    surface) and all subcommands; `runClientCommand` (client → daemon spawn +
    request forwarding).
  - `go.mod`/`go.sum`: deps; note `replace github.com/agenthands/godoll => ../godoll`.
- `/actions/`
  - `actions.go`: browser command implementations + the humanize option builders
    (`typingOpts`/`mouseOpts`/`scrollOpts`).
  - `stealth_check.go`: the `stealth-check` command (per-signal verdicts).
  - `cdp_traffic.go`: the `cdp-traffic` command (v2.0) — reads the CDP proxy's
    ring buffer; formats human or `--json`.
  - `plugin.go`: plugin command surface.
- `/daemon/`
  - `daemon.go`: per-session daemon protocol — `EnsureDaemon`, `ClientExecute`,
    `StartServer`.
- `/types/`
  - `config.go`: `Config`, `StealthConfig`, `StealthFlags`, `ResolveStealth`, the
    fingerprint + humanize validators, `DefaultConfig`, `LoadConfig`,
    `loadSelectedProfile`/`profileLooksLikePath` (v1.7 built-in resolution),
    `boolPtr`/`boolVal`. v1.7 fields: `ConsoleCapture`/`RequestCapture` (default
    OFF), `FontSpoof`/`MediaDevicesSpoof`/`BatterySpoof`/`CodecSpoof` (default ON).
  - `context.go`: browser/page lifecycle, `profileFromStealth`, `createPage`,
    `parseProxyConfig`. v1.7: `applyEmulationIdentity` (zero-enable Emulation
    identity), `osForPlatform`/`chPlatformFor`/`brandsForUA` (OS-coherent dimension
    + CH derivation), the LAZY interceptor (`ensureInterceptorEnabled`,
    `AddRoute`/`RemoveRoute`), the CDP-domain ledger (`recordCDPDomainLocked`,
    `GetEnabledCDPDomains`, `CDPDomainRuntime`/`Network`/`Fetch` consts), opt-in
    console/request capture. v2.0: `cdpProxy` field + `GetCDPProxy()`, proxy wired
    into `launchBrowser()` via `WithCDPWrapper`, jitter + bypass gating.
  - `profiles_embed.go` + `profiles/*.json` (v1.7, Phase 32): embedded Chrome-only
    profile library (`//go:embed`); `BuiltinProfileNames`, `LoadBuiltinProfile`.
    6 profiles: `windows-11-chrome`, `windows-11-desktop-1440p`,
    `windows-10-chrome`, `windows-10-laptop`, `macos-applesilicon-chrome`,
    `macos-intel-chrome`.
  - `snapshot.go`, `logger.go`; `/js/`: injected client-side JS (`*_raw.js` +
    minified `*.js`).
- `/internal/detect/`
  - Offline detection fixture: `detect.html`, `detect.js`, `probe.js`,
    `server.go`, `embed.go`.
- `/internal/plugin/`
  - JS plugin engine: `engine.go`, `lifecycle.go`, `api.go`, `scanner/`.
- `/internal/cdpproxy/` — v2.0
  - `proxy.go`: `Proxy` struct (`cdp.WebSocketable`), ring-buffer logging,
    timing jitter, `Traffic()` accessor.
  - `filters.go`: `normalizeCDPResponse()` — Runtime.getProperties value
    stripping (fail-safe, pass-through on unparseable).
  - `filters_test.go`: 7 unit tests for normalization logic.
- `/utils/`, `/banner/`: helpers and the CLI startup banner.
- `/tests/`: end-to-end + integration tests (drive the built binary), incl.
  `detection_test.go` (offline Tier 1), `detection_live_test.go` (`//go:build
  detection_live`, Tier 2), and `network_evasion_test.go`
  (`TestNetworkEvasionHeaders` — the WIRE-VERIFY of the Emulation identity path).
- v1.7 unit tests: `types/cdp_footprint_test.go` (`TestCDPFootprintBaseline` —
  the falsifiable CDP-01 ledger gate) and `types/profiles_test.go`
  (`TestBuiltinProfilesAreVetted` — the PROF-02 built-in vetting gate).
- `/extensions/pi/` — v2.2 Pi TypeScript extension (separate stack: TypeScript, not Go)
  - `src/index.ts`: extension factory — `export default function(pi)` sets pi reference,
    finds rod-cli binary, registers lifecycle hooks and all 13 tools.
  - `src/cli.ts`: binary resolution (`findRodCli`, three-tier), shell-out wrapper
    (`execRodCli`, wraps `pi.exec("rod-cli", args)`), input validation (`validateInput`),
    per-command timeout table (`TIMEOUTS`).
  - `src/types.ts`: TypeBox schema (`SessionParam` — optional string, shared on every tool).
  - `src/lifecycle.ts`: `session_start` / `session_shutdown` hooks — binary verification
    and daemon cleanup.
  - `src/tools/index.ts`: `registerAllCoreTools` / `registerAllExtendedTools` — splits
    registration into two groups for traceability.
  - `src/tools/*.ts`: 13 `browse_*` tools, each a `pi.registerTool({...})` call
    mapping typed params to rod-cli args: `goto.ts`, `snapshot.ts`, `click.ts`, `type.ts`,
    `eval.ts`, `screenshot.ts`, `wait.ts`, `tabs.ts`, `navigate.ts`, `scroll.ts`,
    `cookies.ts`, `storage.ts`, `fill_form.ts`.
  - `src/__tests__/`: `smoke.test.ts` (parse/export), `adversarial.test.ts`
    (mock-based, 90+ tests), `integration.test.ts` (real binary + HTTP fixture).
  - `package.json`: `@agenthands/rod-cli-pi` v0.1.0, peer-dep on
    `@earendil-works/pi-coding-agent`, `typebox`, `@earendil-works/pi-ai`.
  - `rod-mcp.yaml`: Pi MCP configuration (mode: text, headless: false).
  - `README.md`: extension-level usage docs.
- `/.github/workflows/`: `test.yml` (the blocking gate), `release*.yml`.

## Key Locations

- **Stealth resolution (single funnel):** `types/config.go` `ResolveStealth`.
- **Active identity build:** `types/context.go` `profileFromStealth` → `createPage`.
- **HTTP identity (zero-CDP-enable):** `types/context.go` `applyEmulationIdentity`.
- **CDP footprint ledger:** `types/context.go` `GetEnabledCDPDomains` /
  `recordCDPDomainLocked`; baseline asserted by `types/cdp_footprint_test.go`.
- **Built-in profiles:** `types/profiles_embed.go` (`LoadBuiltinProfile`,
  `BuiltinProfileNames`) + `types/profiles/*.json`.
- **Daemon spawn / protocol:** `daemon/daemon.go`.
- **Detection harness + shared probe:** `internal/detect/` (`probe.js` =
  `detect.Probe`, shared with `actions/stealth_check.go`).
- **Snapshot logic:** client-side in `types/js/snapshotter_raw.js`, server-side
  in `types/snapshot.go` / `actions`.
- **CDP proxy (v2.0):** `internal/cdpproxy/proxy.go` (`Proxy`, `Traffic()`),
  `internal/cdpproxy/filters.go` (`normalizeCDPResponse`), wired in
  `types/context.go launchBrowser()` via godoll's `WithCDPWrapper`.
