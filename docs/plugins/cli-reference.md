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

Triggers a named plugin.

```bash
rod-cli plugin run xss_scanner
```

**Arguments:** one positional `<name>`.

**Behavior — current stub (known limitation):** `plugin run` is **not yet fully implemented**. The daemon's `PluginRun` does not re-execute a registry-resolved plugin; it simply returns the string `Triggered plugin <name>` and does nothing else. The source carries a comment noting that fully running named plugins requires a plugin registry that does not exist yet. Treat the success message as a stub acknowledgement, not as confirmation that any plugin logic ran.

**The real execution path today is `plugin load`.** A plugin's hooks fire continuously after it is loaded — there is no separate "run" step needed to make a loaded plugin do its work. See [lifecycle-hooks.md](./lifecycle-hooks.md).

**Output:** the literal string

```
Triggered plugin <name>
```

**Error / exit conditions:** as with the other subcommands, a daemon transport failure yields a non-zero exit; the stub itself does not validate the name against any registry.

## Thin-Client Model

All three subcommands are thin clients over the daemon. `cmd.go` builds a `daemon.Request` with the matching command name (`"plugin-load"`, `"plugin-list"`, `"plugin-run"`) and arguments, the daemon dispatches it to the corresponding `actions.PluginLoad` / `actions.PluginList` / `actions.PluginRun` function, and the CLI prints the response. Any error returned by the action — or by the client-side argument guard — causes a non-zero process exit.

## See Also

- [lifecycle-hooks.md](./lifecycle-hooks.md) — the hooks that fire after `plugin load`; the live execution path for a plugin.
- [state-api.md](./state-api.md) — the `api` global a loaded plugin uses to read page state (HTML snapshot, cookies, localStorage).

## Source

The CLI argument parsing, the `plugin path is required` client guard, and the `daemon.Request` dispatch live in [`cmd.go`](../../cmd.go). The daemon-side behavior, output strings, and the `plugin run` stub live in [`actions/plugin.go`](../../actions/plugin.go) (`PluginLoad`, `PluginList`, `PluginRun`). The `failed to open script file` error originates in `internal/plugin/engine.go` (`LoadScript`), and the command dispatch table is in `daemon/daemon.go`.
