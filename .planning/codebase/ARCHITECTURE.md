# Architecture

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close — author: codebase-archaeologist)
**Refreshed:** 2026-06-26 (milestone v1.7 close — author: codebase-archaeologist)

> **v1.7 (Complete Evasion Stack)** changed three things inside the same spine,
> all in `types/context.go createPage`. (1) **CDP footprint reduction (Phase 30):**
> console + request capture are now OPT-IN (`--console-capture`/`--request-capture`,
> default OFF); the network interceptor is LAZY (created on first `AddRoute`); HTTP
> identity coherence moved to the zero-enable `Emulation.setUserAgentOverride`
> (`applyEmulationIdentity`); a per-session CDP-domain ledger
> (`GetEnabledCDPDomains`) records footprint-adding enables, and a plain `goto`
> enables NONE of Runtime/Network/Fetch. (2) **Profile library (Phase 32):** 6
> embedded Chrome-only profiles (`types/profiles/*.json`); bare `--profile=<name>`
> resolves a built-in first; `--profile=list`. (3) **Advanced evasion (Phase 33):**
> activated godoll's dormant fingerprint dimensions via `em.SetFingerprint` +
> `SetDimensionOptions`, gated by 4 new toggles (`--font-spoof`/`--media-devices-spoof`/
> `--battery-spoof`/`--codec-spoof`, default ON). **Phase 31 (TLS spoofing) was
> CANCELLED** — real-Chrome-only, NO TLS/JA3 spoofing (that lives in a separate
> project, "munch"); the authentic-Chrome TLS handshake is treated as a strength.

> NOTE: the original 2026-06-18 map described an MCP-server design (`tools/`,
> `server.go`, `runner.go`, `mark3labs/mcp-go`). That is **stale** — the project
> is a CLI driven by a per-session background **daemon**, with no MCP server and
> no `mcp-go` dependency. This section reflects HEAD.

## System Design

`rod-cli` is a Go CLI that gives LLM agents browser automation. Each command is a
one-shot client invocation backed by a persistent per-session daemon that keeps
the browser alive between calls. It is built on `godoll` (a wrapper over
`go-rod`), vendored locally via a `replace => ../godoll` directive.

### Core Components

1. **CLI client (`cmd.go`, `main.go`)**
   - `cmd.go` defines the `urfave/cli/v2` app: global flags (session, headless,
     vision, raw, json, and the full v1.6 stealth surface) and every subcommand
     (open, click, type, fill, snapshot, eval, screenshot, cookies, storage,
     tabs, plugin, `stealth-check`, …).
   - `runClientCommand` pings the session daemon, forwards non-secret stealth
     flags into the daemon spawn argv (and `--proxy-auth` out-of-band via
     `ROD_CLI_PROXY_AUTH`), ensures the daemon, then sends the request.

2. **Daemon (`daemon/`)**
   - `EnsureDaemon` spawns a detached `rod-cli --session <s> <flags…> daemon`
     process if none answers a ping. `ClientExecute` / `StartServer` implement a
     small JSON request/response protocol over a per-session loopback socket.
   - `runDaemonServer` (`main.go`) loads config, **resolves the stealth surface
     once** (`types.ResolveStealth`), builds the `types.Context`, and serves.

3. **Actions (`actions/`)**
   - `actions.go`: the browser command implementations (navigation, click, type,
     fill, scroll, screenshot, eval, cookies, storage, tabs…), using godoll's
     retry/auto-wait + humanize helpers rather than raw go-rod primitives.
   - `stealth_check.go`: the `stealth-check` command — per-signal fingerprint
     verdicts (human / `--raw` / `--json`) using the shared `detect.Probe`.
   - `plugin.go`: the plugin command surface.

4. **Context & types (`types/`)**
   - `context.go`: owns the `rod.Browser`/`rod.Page` lifecycle; builds the active
     `stealth.Profile`, applies evasion, threads the per-session noise seed. v1.7:
     `createPage` now calls `em.SetFingerprint(OS-constrained, seeded fp)` +
     `em.SetDimensionOptions(...)` to activate godoll's dimension injectors;
     carries HTTP identity via `applyEmulationIdentity` (zero-enable Emulation
     domain) instead of the old always-on Fetch catch-all; opt-in console/request
     capture and a LAZY network interceptor (`ensureInterceptorEnabled`); and a
     per-session CDP-domain ledger (`recordCDPDomainLocked` / `GetEnabledCDPDomains`,
     keyed by `CDPDomainRuntime`/`Network`/`Fetch`).
   - `config.go`: the `Config` / `StealthConfig` / `StealthFlags` structures and
     `ResolveStealth` — the single stealth resolver (see CONVENTIONS). v1.7 added
     the `ConsoleCapture`/`RequestCapture` (default OFF) and `FontSpoof`/
     `MediaDevicesSpoof`/`BatterySpoof`/`CodecSpoof` (default ON) `*bool` knobs.
   - `profiles_embed.go` + `profiles/*.json` (v1.7, Phase 32): the embedded
     Chrome-only profile library — `//go:embed profiles/*.json`,
     `BuiltinProfileNames`, `LoadBuiltinProfile`, `isBuiltinProfile`. Built-in
     resolution is wired into `ResolveStealth` via `loadSelectedProfile`.
   - `snapshot.go`, `logger.go`, `js/`: page snapshots, logging, injected JS.

5. **Detection harness (`internal/detect/`)**
   - An offline, `go:embed`-bundled bot-detection fixture (`detect.html`,
     `detect.js`, `probe.js`) served on `127.0.0.1:0` (`server.go`, `embed.go`).
     `probe.js` is the single source of truth shared by the harness and the
     `stealth-check` command.

6. **Plugin engine (`internal/plugin/`)**
   - JS plugin loading/lifecycle/scanner (`engine.go`, `lifecycle.go`, `api.go`,
     `scanner/`).

## Data Flow

1. The client process pings the session daemon; if absent, spawns it with the
   stealth flags baked into argv (secrets via env).
2. The daemon resolves stealth once, launches Chromium via godoll, and listens.
3. The client sends a JSON request; the daemon runs the matching `actions`
   function against the live page.
4. Results (snapshot/markdown/verdict) return over the socket; the client prints
   them plain, `--raw`, or `--json`.
