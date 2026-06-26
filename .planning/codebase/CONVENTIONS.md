# Conventions

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)
**Refreshed:** 2026-06-26 (milestone v1.7 close)

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
  so `nil` (unset → default) is distinct from an explicit `false`/zero. v1.7
  extended this to the dimension toggles (`FontSpoof`/`MediaDevicesSpoof`/
  `BatterySpoof`/`CodecSpoof`, default ON) and the CDP-capture toggles
  (`ConsoleCapture`/`RequestCapture`, default OFF) — all `*bool`, read via
  `boolVal(p, default)`.
- **Footprint follows the feature (v1.7, CDP-01)**: a CDP domain is enabled only
  when a feature that needs it is actually used — console/request capture are
  opt-in flags; the network interceptor (`Fetch.enable`) is created lazily on the
  first `AddRoute` and torn down on the last `RemoveRoute`. HTTP identity rides on
  the zero-`enable` `Emulation` domain. A plain `goto` therefore enables none of
  Runtime/Network/Fetch — the falsifiable baseline asserted by the harness.
- **Instrument every new CDP enable-point**: footprint-adding enables are recorded
  in the per-session ledger via `recordCDPDomainLocked` (caller holds `stateLock`)
  so `GetEnabledCDPDomains` stays a truthful, testable inventory. NOTE the known
  gap: the plugin lifecycle binder bypasses this ledger (see CONCERNS).
- **Coherent-not-random dimensions (v1.7, EVAD)**: generated fingerprint
  dimensions are constrained to the active profile's OS/locale (`FPWithOS`/
  `FPWithLocales`, OS mapped by `osForPlatform`) and seeded with the per-session
  noise seed — never an unconstrained random draw on a pinned identity, which
  would be a NEW cross-layer tell.

## Scripts and Assets
- **Client-Side JS**: Unminified JavaScript logic is kept in `*_raw.js` files and compiled into `*.js` using `terser` (via `npm run dev`). Go code then loads or embeds the minified versions.
