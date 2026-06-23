# Plugin CLI Reference

`rod-cli` exposes three plugin subcommands under the top-level `plugin` command: `plugin load`, `plugin list`, and `plugin run`. These are thin clients — each one packages your arguments into a `daemon.Request{Command: "plugin-load" | "plugin-list" | "plugin-run"}` envelope, sends it to the running daemon, prints the daemon's response, and exits non-zero if the command fails. This page documents each subcommand's arguments, behavior, output format, and error/exit conditions exactly as they are implemented in `cmd.go` and `actions/plugin.go`.

## `plugin load <path>`

Loads a plugin script into the active daemon session.

```bash
rod-cli plugin load ./plugins/examples/xss_scanner.js
```

**Arguments:** one positional `<path>` — the filesystem path to the plugin's JavaScript source. Required.

**Behavior:** the daemon's `PluginLoad` compiles and runs the script in the goja VM (`engine.LoadScript`), then, if a controlled page is available, calls `engine.BindLifecycle` to attach the CDP event listeners and bind the `api` global. The plugin's lifecycle hooks (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`) begin firing from this point on — loading a plugin is what activates its hooks. The path is also recorded in the session's loaded-plugin list (surfaced by `plugin list`).

**Output (success):** the literal string

```
Plugin loaded successfully from <path>
```

**Error / exit conditions:**

- **Missing path argument** — the CLI guards this client-side in `cmd.go` and returns the error `plugin path is required` without ever contacting the daemon. The process exits non-zero.
- **Missing or unreadable file** — the daemon's `engine.LoadScript` cannot open the file and surfaces `failed to open script file <path>` (wrapping the underlying OS error). The command prints the error and exits non-zero.

Because the hooks begin firing after a successful load, `plugin load` is the primary execution path for a plugin today. See [lifecycle-hooks.md](./lifecycle-hooks.md) for the hooks that run and [state-api.md](./state-api.md) for the page state a loaded plugin can read.

## `plugin list`

Lists the plugins currently loaded in the active daemon session.

```bash
rod-cli plugin list
```

**Arguments:** none.

**Behavior:** the daemon's `PluginList` reads the session's loaded-plugin paths and reports them.

**Output:** two forms —

- When no plugins are loaded, the literal string `No active plugins loaded.`
- Otherwise, a JSON array of the loaded plugin paths, e.g.:

```json
["./plugins/examples/xss_scanner.js"]
```

**Error / exit conditions:** a marshaling failure (rare) surfaces as an error and a non-zero exit; otherwise the command always succeeds.

## `plugin run <name>`

Invokes a named JavaScript function in the loaded plugin and prints its result.

```bash
rod-cli plugin run getFindings
```

**Arguments:** one positional `<name>` — the name of a top-level function defined in the currently loaded plugin's VM.

**Behavior:** `plugin run <name>` invokes the JS function named `<name>` in the loaded plugin's VM and returns its result stringified. The daemon's `PluginRun` delegates to `engine.RunFunc`, which looks the name up in the goja runtime (`vm.Get` + `goja.AssertFunction`), calls it with no arguments, and stringifies the returned value. This is the read path for in-VM plugin state: the canonical use is `rod-cli plugin run getFindings`, which calls the example XSS scanner's `getFindings` accessor and returns its collected findings as JSON. It operates on the single loaded plugin VM — there is no multi-plugin registry, so `<name>` resolves against whatever plugin is currently loaded.

This complements `plugin load`: a plugin's hooks (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`) still fire automatically after load and accumulate state as you browse; `plugin run` is how you invoke a named accessor or function — such as `getFindings` or `getRequestLog` — to read that state back. See [lifecycle-hooks.md](./lifecycle-hooks.md) for the hooks and the [XSS scanner worked example](./examples/xss-scanner.md) for the full load → drive page → `plugin run getFindings` flow.

**Output:** the function's returned value, stringified. The example accessors call `JSON.stringify(...)`, so the CLI prints clean JSON. A function that returns `undefined` or `null` prints an empty string.

**Error / exit conditions:**

- **Missing name argument** — an empty `<name>` is guarded with `plugin function name is required`.
- **No such function** — a name that is not defined in the loaded plugin's VM yields `function "<name>" not found`.
- **Not callable** — a name bound to a non-callable value yields `"<name>" is not a callable function`.
- **Runtime error inside the function** — an exception thrown while the function runs is wrapped as `error calling "<name>"`.
- **Uninitialized engine** — if no plugin VM is available, `engine.RunFunc` returns `plugin engine not initialized`.

Each of these causes a non-zero process exit, as does a daemon transport failure.

## Thin-Client Model

All three subcommands are thin clients over the daemon. `cmd.go` builds a `daemon.Request` with the matching command name (`"plugin-load"`, `"plugin-list"`, `"plugin-run"`) and arguments, the daemon dispatches it to the corresponding `actions.PluginLoad` / `actions.PluginList` / `actions.PluginRun` function, and the CLI prints the response. Any error returned by the action — or by the client-side argument guard — causes a non-zero process exit.

## See Also

- [lifecycle-hooks.md](./lifecycle-hooks.md) — the hooks that fire after `plugin load`; the live execution path for a plugin.
- [state-api.md](./state-api.md) — the `api` global a loaded plugin uses to read page state (HTML snapshot, cookies, localStorage).

## Source

The CLI argument parsing, the `plugin path is required` client guard, and the `daemon.Request` dispatch live in [`cmd.go`](../../cmd.go). The daemon-side behavior and output strings live in [`actions/plugin.go`](../../actions/plugin.go) (`PluginLoad`, `PluginList`, `PluginRun`); `PluginRun` delegates to `engine.RunFunc`. The `failed to open script file` error originates in `internal/plugin/engine.go` (`LoadScript`), the `function "<name>" not found` / `"<name>" is not a callable function` / `error calling "<name>"` errors come from `RunFunc` in the same file, and the command dispatch table is in `daemon/daemon.go`.
