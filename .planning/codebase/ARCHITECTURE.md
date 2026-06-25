# Architecture

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close — author: codebase-archaeologist)

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
     `stealth.Profile`, applies evasion, wires the network interceptor, threads
     the per-session noise seed, and captures console/request logs.
   - `config.go`: the `Config` / `StealthConfig` / `StealthFlags` structures and
     `ResolveStealth` — the single stealth resolver (see CONVENTIONS).
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
