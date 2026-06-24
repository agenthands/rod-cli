# Phase 25: Stealth Config Surface & Per-Session Proxy - Context

**Gathered:** 2026-06-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Build the **session-persistent stealth configuration substrate** every later v1.6 phase rides — CLI flags plus a named JSON profile file, resolved with deterministic precedence once at daemon-spawn time and inherited by every later command on the same session. Then ship the first feature riding it: a **per-session HTTP/SOCKS5 proxy with CDP-based auth**, which validates the whole flag → config → godoll path end-to-end.

Covers: PROFILE-01, PROFILE-02, PROXY-01, PROXY-02. Does NOT add fingerprint pinning, hardening, or humanize tuning (Phases 26–28) — but the config struct must be shaped to accommodate them.

**Resolved architectural finding (closes the flagged Phase 25/26 blocker):** rod-cli runs **one daemon process per named session** (`daemon.EnsureDaemon(session, ...)` → `rod-cli-<session>.port` → its own `StartServer` + `*types.Context` + browser + `Config`). Config is loaded once and frozen at `NewContext(cfg)`. Therefore stealth config forwarded at daemon-spawn time is **automatically both session-persistent and session-isolated** — there is NO shared `*rod.Browser` across sessions, so the research Pitfall-8 "per-BrowserContext isolation" work is NOT needed. Per-session isolation is process isolation.
</domain>

<decisions>
## Implementation Decisions

### Config Surface & Precedence
- **Config shape:** add a cohesive `Stealth` sub-struct to `types/config.go`'s `Config` (fields: `Proxy`, `ProxyAuth`, `Profile`, and room for the fingerprint pins Phases 26–28 will add). The bare top-level `Proxy string` migrates into it (or is bridged) so all stealth config lives in one place.
- **Profile file:** a godoll `stealth.Profile` JSON file (its native `Save`/`LoadProfile` format), referenced by `--profile <name-or-path>`. No custom YAML format.
- **Precedence:** `CLI flag > profile file > built-in default`, resolved **once at daemon-spawn time** (in the config load path before `NewContext`). No per-command stealth state.
- **Flag passed to an already-running session:** **warn to stderr** (e.g. "session 'X' already running; stealth flags apply at session spawn — run `close` first to re-apply") and proceed with the existing daemon's config. Do NOT silently ignore (confusing) and do NOT auto-restart the daemon (would surprise-kill in-flight session state). This is safe and predictable precisely because sessions are process-isolated. Same semantics as today's `--headless`.

### Per-Session Proxy
- **Proxy flag + API:** `--proxy <url>` with the scheme in the URL (`http://…`, `socks5://…`). Route through **godoll's proxy API** (`browser.ProxyConfig.ApplyToLauncher` / `SetupBrowserAuth`), replacing the current bare `launcher.Proxy(cfg.Proxy)` call at `types/context.go:71-72`. Proxy is bound per named session (automatic via the per-session daemon).
- **Proxy auth:** `--proxy-auth user:pass` handled via **CDP `Fetch.continueWithAuth`** (godoll `SetupBrowserAuth`). Never URL-embedded credentials (Chrome removed them).
- **SOCKS5-with-auth:** Chrome's `--proxy-server` cannot do SOCKS5 auth; use **godoll's local authenticated relay (`StartProxyRelay`)** for that case. Document the mechanism.
- **Credential safety:** redact proxy credentials everywhere — never emit them to stdout/stderr logs, and never persist them to the session `.port`/state files.

### Claude's Discretion
- Exact flag names beyond the agreed `--proxy` / `--proxy-auth` / `--profile` (e.g. whether profile save is a subcommand `stealth profile save` vs a flag).
- Internal struct field names and whether to keep a compatibility shim for the old top-level `Config.Proxy`.
- Exact precedence-resolution implementation site and the warning's precise wording.
- Whether `--profile <name>` resolves names against a default profiles dir and what that path is.
</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `types/config.go:14` — `Config` struct (has `Headless`, bare `Proxy string`). This is where the `Stealth` sub-struct lands.
- `types/context.go:44-72` — `launchBrowser`: builds the launcher, currently `if cfg.Proxy != "" { browserLauncher.Proxy(cfg.Proxy) }` (bare, no SOCKS5/auth). This is the proxy-rewire site.
- `cmd.go:16` `runClientCommand` + `daemon.EnsureDaemon(session, os.Args[0], flags)` — the daemon-spawn boundary where stealth flags must be packed/forwarded (`flags []string`). Global flags defined ~`cmd.go:55-62` (`config`, `cdp-endpoint`, `session`).
- `daemon/daemon.go:92` `EnsureDaemon` (pings, else spawns `--session <s>` + flags); `daemon.go:120` `StartServer(session, ppid, rodCtx)`; `portFilePath(session)` = `rod-cli-<session>.port`. Confirms one-daemon-per-session.
- godoll: `browser.ProxyConfig` (`ApplyToLauncher`, `SetupBrowserAuth`, `StartProxyRelay`), `stealth.Profile` (`Save`/`LoadProfile`) — verified in STACK.md as already-built.

### Established Patterns
- Config flows: CLI flags → packed into daemon spawn args (`EnsureDaemon`) → `runDaemonServer` loads `Config` → `NewContext(cfg)` freezes it → every later command hits the same in-memory `rodCtx`.
- The detection harness (Phase 24, `tests/detection_test.go` + `internal/detect`) is the regression net — proxy/config behavior should be asserted via the live binary where feasible.

### Integration Points
- New stealth flags registered in `cmd.go` global flags; forwarded through `EnsureDaemon`'s `flags`.
- `types/config.go` Stealth struct; `types/context.go` launchBrowser proxy rewire.
- Per-session isolation test: two concurrent `-s` sessions with different `--proxy` must report different egress identity (extends the Phase 24 harness style).
</code_context>

<specifics>
## Specific Ideas

- Keep CLI output token-efficient (the retrospective lesson): the already-running-session warning goes to **stderr**, never stdout, so `--raw`/piped callers are unaffected.
- A session-isolation test is explicitly wanted: prove that session A's `--proxy` does not bleed into session B (different daemons, different egress).
- Credentials must never appear in `rod-cli-<session>.port` or any state file.
</specifics>

<deferred>
## Deferred Ideas

- Fingerprint pinning (UA/locale/timezone/screen) and the consistency validator → Phase 26 (the Stealth sub-struct should leave room for these fields, but they are not wired here).
- Canvas/WebGL/WebRTC hardening → Phase 27.
- Humanize tuning knobs → Phase 28.
- Proxy IP-geo ↔ timezone consistency check → belongs with the Phase 26 consistency validator (proxy here just routes; coherence with spoofed timezone is a Ph26 concern).
</deferred>
