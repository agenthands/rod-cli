# Stack

**Mapped:** 2026-06-18
**Refreshed:** 2026-06-25 (milestone v1.6 close)
**Refreshed:** 2026-06-26 (milestone v1.7 close)

## Core Technologies
- **Language:** Go — `go.mod` declares `go 1.25.1`; CI `test.yml` pins `1.25.x`;
  the release workflows pin `1.23`/`1.23.7`; the dev machine builds on `go1.26.0`.
  This skew is a tracked CONCERN (and a v1.7 security follow-up: bump to the
  CVE-patched `go1.26.1`). No `toolchain` directive is present in `go.mod`.
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
- `godoll/stealth` (EvasionManager, Profile, DefaultProfile/LoadProfile,
  `SetProfile`/`Apply`/`EvadeWebRTC`/`SetNoiseSeed`). v1.7 added
  `EvasionManager.SetFingerprint` + `SetDimensionOptions(stealth.DimensionOptions)`
  (Fonts/MediaDevices/Battery/Codecs/Plugins) — the API that activates godoll's
  previously-dormant `applyFingerprintDimensions` injectors.
- `godoll/browser` (ProxyConfig, launch, `WithWebRTCLeakProtection`),
  `godoll/network` (interceptor), `godoll/humanize` (typing/mouse/scroll options).
- `godoll/fingerprint`: the generator, now driven via
  `NewFingerprintGeneratorSeeded(seed, opts...)` with `FPWithBrowserNames`,
  `FPWithOS`, `FPWithLocales` so dimensions are OS/locale-coherent and
  session-stable.
- **KNOWN godoll limitation (v1.7):** `stealth.scriptMockFonts` is an observable
  no-op — it overrides `measureText` but returns the original width in every
  branch, so `--font-spoof` gates injection but does not change the live font
  readout. Verified in `../godoll/stealth/fingerprint_bridge.go` (see CONCERNS).

## JavaScript Assets
- Injected JS lives under `types/js/` (unminified `*_raw.js` → minified `*.js`)
  and `internal/detect/` (`detect.js`, `probe.js`). `probe.js` is embedded into
  the binary via `go:embed`.

## Environment & Build
- Single cross-platform Go binary (Windows/macOS/Linux); release via GoReleaser
  (`.github/workflows/release*.yml`).
