# Integrations

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)

> The original map described an MCP server (`mark3labs/mcp-go`, Stdio RPC). That
> is **stale** — rod-cli is a CLI, not an MCP server, and does not depend on
> `mcp-go`.

## External Integrations

- **godoll (`github.com/agenthands/godoll`)**: the stealth + automation engine,
  vendored locally (`replace => ../godoll`). rod-cli consumes `stealth`,
  `browser`, `network`, `fingerprint`, and `humanize`.
- **Chrome DevTools Protocol (CDP)**: inherited via `go-rod`. Used for page
  control, proxy auth (`Fetch.continueWithAuth`), and console/request logging
  (`Runtime.enable` / `Network.enable` — see docs/cdp-footprint.md). Can also
  attach to an existing browser via `--cdp-endpoint`.
- **LLM agents**: rod-cli is consumed as an agent "Skill" — each command is a
  one-shot shell invocation; agents call the binary directly (see
  `skills/rod-cli/SKILL.md`).

## Interfaces & Data Flow

- **CLI input**: `urfave/cli/v2` parses args/flags (`cmd.go`).
- **Client ⇄ daemon**: a small JSON request/response protocol over a per-session
  loopback socket (`daemon/daemon.go`). The client forwards stealth config at
  daemon spawn (argv for non-secrets, env for `--proxy-auth`).
- **Egress proxy**: optional per-session HTTP/SOCKS proxy via `--proxy`
  (`parseProxyConfig` → `godoll/browser.ProxyConfig`).
- **Detection fixture**: a loopback-only `127.0.0.1:0` HTTP server serving
  embedded assets for the offline harness (`internal/detect/server.go`).
