# Stack

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)

## Core Technologies
- **Language:** Go 1.25.1 (go.mod)
- **Stealth / browser automation:** `github.com/agenthands/godoll`
  (`v0.0.0-20260314193512-…`, vendored locally via `replace => ../godoll`), which
  wraps `github.com/go-rod/rod` (v0.116.2).
- **CLI framework:** `github.com/urfave/cli/v2` (v2.27.6)
- **Logging & terminal UI:** `github.com/charmbracelet/log` (v0.4.1),
  `github.com/charmbracelet/lipgloss` (v1.1.0)
- **Error handling:** `github.com/pkg/errors`
- **Config parsing:** `gopkg.in/yaml.v3`; stealth profiles are JSON
  (`stealth.LoadProfile`).

> NOTE: the original map listed `mark3labs/mcp-go`. rod-cli is **not** an MCP
> server and no longer depends on `mcp-go`; it is a CLI + per-session daemon.

## godoll surface used
- `godoll/stealth` (EvasionManager, Profile, DefaultProfile/LoadProfile),
  `godoll/browser` (ProxyConfig, launch), `godoll/network` (interceptor),
  `godoll/fingerprint` (generator for WebGL dims only), `godoll/humanize`
  (typing/mouse/scroll options).

## JavaScript Assets
- Injected JS lives under `types/js/` (unminified `*_raw.js` → minified `*.js`)
  and `internal/detect/` (`detect.js`, `probe.js`). `probe.js` is embedded into
  the binary via `go:embed`.

## Environment & Build
- Single cross-platform Go binary (Windows/macOS/Linux); release via GoReleaser
  (`.github/workflows/release*.yml`).
