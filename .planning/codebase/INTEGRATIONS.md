# Integrations

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)
**Refreshed:** 2026-06-26 (milestone v1.7 close)
**Refreshed:** 2026-06-27 (milestone v2.2 close — author: codebase-archaeologist)

> The original map described an MCP server (`mark3labs/mcp-go`, Stdio RPC). That
> is **stale** — rod-cli is a CLI, not an MCP server, and does not depend on
> `mcp-go`.

## External Integrations

- **godoll (`github.com/agenthands/godoll`)**: the stealth + automation engine,
  vendored locally (`replace => ../godoll`). rod-cli consumes `stealth`,
  `browser`, `network`, `fingerprint`, and `humanize`.
- **Chrome DevTools Protocol (CDP)**: inherited via `go-rod`. Used for page
  control (`Page`/`Target`, irreducible), proxy auth (`Fetch.continueWithAuth`),
  and — v1.7 — HTTP identity coherence via the zero-`enable`
  `Emulation.setUserAgentOverride` (`applyEmulationIdentity`). Console/request
  logging (`Runtime.enable` / `Network.enable`) are now OPT-IN
  (`--console-capture`/`--request-capture`, default OFF) and the mock interceptor
  (`Fetch.enable`) is lazy, so a plain session enables none of Runtime/Network/Fetch.
  The daemon records a per-session domain-enable ledger (`GetEnabledCDPDomains`).
  See docs/cdp-footprint.md. Can also attach to an existing browser via
  `--cdp-endpoint`.
- **LLM agents**: rod-cli is consumed as an agent "Skill" — each command is a
  one-shot shell invocation; agents call the binary directly (see
  `skills/rod-cli/SKILL.md`).
- **Pi (pi.dev)** — two integration paths (v2.2):
  - **Agent Skill** (`skills/rod-cli/SKILL.md`): the shell-out path — Pi reads the
    skill file and invokes rod-cli CLI commands directly. No npm dependency. The
    zero-install fallback for Pi users who switch between assistants.
  - **TypeScript Extension** (`extensions/pi/`): a first-class Pi extension
    (`@agenthands/rod-cli-pi`). Uses `@earendil-works/pi-coding-agent` ExtensionAPI.
    13 typed `browse_*` tools registered via `pi.registerTool()` with TypeBox-typed
    schemas, prompt snippets, and usage guidelines. Shell-out via
    `pi.exec("rod-cli", args)` — the extension is a thin typed wrapper over the CLI,
    not an in-process browser control. Lifecycle hooks (`session_start` binary
    verification, `session_shutdown` daemon cleanup on quit). Input validation
    enforced client-side before shell-out (URL scheme, selector presence, eval size
    limit). See `extensions/pi/README.md` for the full tool catalog.

## Interfaces & Data Flow

- **CLI input**: `urfave/cli/v2` parses args/flags (`cmd.go`).
- **Client ⇄ daemon**: a small JSON request/response protocol over a per-session
  loopback socket (`daemon/daemon.go`). The client forwards stealth config at
  daemon spawn (argv for non-secrets, env for `--proxy-auth`).
- **Egress proxy**: optional per-session HTTP/SOCKS proxy via `--proxy`
  (`parseProxyConfig` → `godoll/browser.ProxyConfig`).
- **Detection fixture**: a loopback-only `127.0.0.1:0` HTTP server serving
  embedded assets for the offline harness (`internal/detect/server.go`).
