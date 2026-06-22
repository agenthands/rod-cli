# 17-01: Engine Sandbox Setup

## Description
Implemented the plugin engine sandbox and script loader using the `goja` JavaScript engine. This satisfies PLUG-01 and PLUG-02 requirements by providing a safe, CGO-free JavaScript execution environment embedded in the daemon.

## Files Modified
- `go.mod`, `go.sum`: Added `github.com/dop251/goja` dependency.
- `internal/plugin/engine.go`: Created `PluginEngine` struct, `Init()` method, and `LoadScript(path string)` method.

## Decisions Made
- Used `github.com/dop251/goja` as the primary JavaScript execution engine instead of native Go plugins or CGO-based runtimes like v8go.
- Kept the `PluginEngine` stateful with a shared `goja.Runtime` instance per engine, which matches the ephemeral port/daemon per instance model.
