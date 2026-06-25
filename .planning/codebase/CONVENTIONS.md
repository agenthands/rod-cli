# Conventions

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)

## Code Style
- **Idiomatic Go**: Uses standard Go formatting (`gofmt`).
- **Error Handling**: Uses `github.com/pkg/errors` for error wrapping with stack traces (`errors.Wrapf`). 
- **Logging**: Uses `github.com/charmbracelet/log` for leveled, colorful, structured logging.

## Architectural Patterns
- **Dependency Injection**: The `types.Context` struct is passed down to all
  action handlers, sharing one browser state + frozen config without globals.
- **Client/daemon split**: `cmd.go` (client) spawns and talks to a per-session
  daemon (`main.go` `runDaemonServer` + `daemon/`). No MCP server.
- **Resolve-once stealth**: every stealth knob funnels through the single
  `types.ResolveStealth` resolver at daemon spawn (precedence CLI > profile >
  default), validated at that one seam, then frozen into `Config` by
  `NewContext`. New knobs follow the flag → forward → StealthFlags → resolve path
  (see CONCERNS landmine #1).
- **Pointer "unset" semantics**: hardening + humanize config fields are pointers
  so `nil` (unset → default) is distinct from an explicit `false`/zero.

## Scripts and Assets
- **Client-Side JS**: Unminified JavaScript logic is kept in `*_raw.js` files and compiled into `*.js` using `terser` (via `npm run dev`). Go code then loads or embeds the minified versions.
