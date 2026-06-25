# Structure

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)

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
  - `plugin.go`: plugin command surface.
- `/daemon/`
  - `daemon.go`: per-session daemon protocol — `EnsureDaemon`, `ClientExecute`,
    `StartServer`.
- `/types/`
  - `config.go`: `Config`, `StealthConfig`, `StealthFlags`, `ResolveStealth`, the
    fingerprint + humanize validators, `DefaultConfig`, `LoadConfig`.
  - `context.go`: browser/page lifecycle, `profileFromStealth`, `createPage`,
    `parseProxyConfig`, the network interceptor, console/request logging.
  - `snapshot.go`, `logger.go`; `/js/`: injected client-side JS (`*_raw.js` +
    minified `*.js`).
- `/internal/detect/`
  - Offline detection fixture: `detect.html`, `detect.js`, `probe.js`,
    `server.go`, `embed.go`.
- `/internal/plugin/`
  - JS plugin engine: `engine.go`, `lifecycle.go`, `api.go`, `scanner/`.
- `/utils/`, `/banner/`: helpers and the CLI startup banner.
- `/tests/`: end-to-end + integration tests (drive the built binary), incl.
  `detection_test.go` (offline Tier 1) and `detection_live_test.go` (`//go:build
  detection_live`, Tier 2).
- `/.github/workflows/`: `test.yml` (the blocking gate), `release*.yml`.

## Key Locations

- **Stealth resolution (single funnel):** `types/config.go` `ResolveStealth`.
- **Active identity build:** `types/context.go` `profileFromStealth` → `createPage`.
- **Daemon spawn / protocol:** `daemon/daemon.go`.
- **Detection harness + shared probe:** `internal/detect/` (`probe.js` =
  `detect.Probe`, shared with `actions/stealth_check.go`).
- **Snapshot logic:** client-side in `types/js/snapshotter_raw.js`, server-side
  in `types/snapshot.go` / `actions`.
