# Architecture

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close — author: codebase-archaeologist)
**Refreshed:** 2026-06-26 (milestone v1.7 close — author: codebase-archaeologist)
**Refreshed:** 2026-06-26 (milestone v2.0 close — author: architect)
**Refreshed:** 2026-06-27 (milestone v2.1 close — author: architect)
**Refreshed:** 2026-06-27 (milestone v2.2 close — author: codebase-archaeologist)

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
> **v2.0 (CDP-DEEP-01 Build, 2026-06-26):** shipped an in-process MITM WebSocket
> CDP proxy (`internal/cdpproxy/`, 402 lines) that sits between go-rod and Chrome's
> debugging WebSocket, implementing `cdp.WebSocketable`. The proxy provides:
> pass-through traffic logging (ring buffer), Runtime.getProperties normalization
> (strips accessor `value` fields), configurable timing jitter (`--cdp-jitter-ms`),
> a diagnostic `cdp-traffic` command (`actions/cdp_traffic.go`), and a
> `--no-cdp-proxy` bypass escape hatch. Default-OFF behind `--cdp-proxy`.

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
   - `cdp_traffic.go`: the `cdp-traffic` command (v2.0) — reads the CDP proxy's
     ring buffer, formats output human or `--json`.
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

7. **CDP proxy (`internal/cdpproxy/`)** — v2.0
   - `proxy.go`: `Proxy` struct implementing `cdp.WebSocketable` — pass-through
     with ring-buffer logging (cap 1024), configurable timing jitter
     (`jitterMaxMs`), and `Traffic()` accessor for diagnostics.
   - `filters.go`: `normalizeCDPResponse()` — JSON-level Runtime.getProperties
     response normalization (strips `value` from accessor properties with `get`
     field), fail-safe pass-through on unparseable input.
   - Wired into `launchBrowser()` via godoll's `WithCDPWrapper`, gated behind
     `--cdp-proxy` (default OFF).
   - **v2.1:** added stderr soft-warning when `--cdp-jitter-ms > 1000`
     (`types/context.go:185`); `cdp-traffic` Usage now includes sensitivity
     caveat about CDP payload data (`cmd.go:516`). Proxy integration tests
     in `tests/proxy_integration_test.go` assert traffic log contents and
     Runtime normalization via a self-contained cdpTell probe.

8. **Pi Extension (`extensions/pi/`)** — v2.2
   - TypeScript Pi extension (not Go). Ships as npm package `@agenthands/rod-cli-pi`.
     Entry point: `extensions/pi/src/index.ts` — `export default function(pi: ExtensionAPI)`
     sets internal pi reference, finds rod-cli binary, registers lifecycle hooks + 13 tools.
   - **Binary resolution** (`cli.ts`, `findRodCli`): three-tier search at extension load
     time — `ROD_CLI_PATH` > `PATH` > `$GOBIN` / `$HOME/go/bin`. Returns path or `null`
     (caller handles not-found via user notification).
   - **Shell-out wrapper** (`cli.ts`, `execRodCli`): wraps `pi.exec("rod-cli", args)` —
     prepends `--raw`, validates input, applies per-command timeouts, throws on non-zero
     exit. The single point of interaction with rod-cli.
   - **Input validation** (`cli.ts`, `validateInput`): client-side enforcement before
     shell-out — URL scheme (`http://`/`https://` only), CSS selector presence, eval
     expression size cap (10KB), empty-text rejection for fill.
   - **Per-command timeouts** (`cli.ts`, `TIMEOUTS`): mapped per subcommand — goto 60s,
     screenshot/wait 30s, snapshot/click/fill/type/eval 15s, close/--version 5s.
   - **13 browse_* tools** (`src/tools/*.ts`): registered via `pi.registerTool()` with
     TypeBox-typed parameters, prompt snippets, and guidelines. Core (Phase 47-48):
     `browse_goto`, `browse_snapshot`, `browse_click`, `browse_type`, `browse_eval`,
     `browse_screenshot`, `browse_wait`. Extended (Phase 49): `browse_tabs`,
     `browse_navigate`, `browse_scroll`, `browse_cookies`, `browse_storage`,
     `browse_fill_form`.
   - **Lifecycle hooks** (`lifecycle.ts`): `session_start` verifies binary via
     `rod-cli --version`; `session_shutdown` runs `rod-cli close` only on
     `reason: "quit"` (not reload/fork/resume).
   - **Trust boundary**: extension runs in Pi agent process. Shell-out to rod-cli via
     `pi.exec()` — the rod-cli binary is a separate OS process. Tool parameters validated
     client-side before shell-out; rod-cli enforces server-side validation
     (defense-in-depth). Thread via `--` separator between flags and user-provided values
     (I6 injection protection for fill/type/eval args).
   - **Testing**: 3 test files — `smoke.test.ts` (parse/export), `adversarial.test.ts`
     (mock-based, 90+ tests), `integration.test.ts` (real binary + HTTP fixture;
     runs only when rod-cli is in PATH).

## Data Flow

1. The client process pings the session daemon; if absent, spawns it with the
   stealth flags baked into argv (secrets via env).
2. The daemon resolves stealth once, launches Chromium via godoll. If
   `--cdp-proxy` is set (v2.0), godoll wraps the CDP WebSocket in a
   `cdpproxy.Proxy` before go-rod connects — all CDP traffic then flows
   through the proxy (normalization + jitter + logging).
3. The client sends a JSON request; the daemon runs the matching `actions`
   function against the live page.
4. Results (snapshot/markdown/verdict) return over the socket; the client prints
   them plain, `--raw`, or `--json`. If the proxy is active, `rod-cli cdp-traffic`
   reads the proxy's ring buffer.
