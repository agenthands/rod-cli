# Conventions

**Mapped:** 2026-06-18

## Code Style
- **Idiomatic Go**: Uses standard Go formatting (`gofmt`).
- **Error Handling**: Uses `github.com/pkg/errors` for error wrapping with stack traces (`errors.Wrapf`). 
- **Logging**: Uses `github.com/charmbracelet/log` for leveled, colorful, structured logging.

## Architectural Patterns
- **Dependency Injection**: The `types.Context` struct is passed down to all tool handlers, ensuring tools share the same browser state and configuration without relying on globals.
- **Mode-based Routing**: The `server.go` file checks the current mode (`types.Text`, `types.Vision`) to decide which tool sets (`tools.TextTools` vs others) to register to the MCP server.

## Scripts and Assets
- **Client-Side JS**: Unminified JavaScript logic is kept in `*_raw.js` files and compiled into `*.js` using `terser` (via `npm run dev`). Go code then loads or embeds the minified versions.
