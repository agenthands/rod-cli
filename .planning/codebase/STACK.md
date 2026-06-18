# Stack

**Mapped:** 2026-06-18

## Core Technologies
- **Language:** Go 1.23.7
- **Browser Automation:** `github.com/go-rod/rod` (v0.116.2)
- **MCP Server Protocol:** `github.com/mark3labs/mcp-go` (v0.20.1)
- **CLI Framework:** `github.com/urfave/cli/v2` (v2.27.6)
- **Logging & Terminal UI:** `github.com/charmbracelet/log`, `github.com/charmbracelet/lipgloss`
- **Error Handling:** `github.com/pkg/errors`
- **Configuration Parsing:** `gopkg.in/yaml.v3`

## JavaScript Assets
- Node.js is used as a build-time dependency for minifying injected scripts (`terser` via `package.json` script `dev`).

## Environment & Build
- Cross-platform executable (Windows/macOS/Linux) produced via Go build system.
- Build configuration provided via `.goreleaser.yml`.
