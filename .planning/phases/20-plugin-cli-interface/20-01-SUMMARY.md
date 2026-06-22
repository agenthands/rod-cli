# 20-01: Plugin CLI Interface

## Description
Exposed a comprehensive CLI interface and backend daemon routes for managing the plugin lifecycle within `rod-cli`. This satisfies PLUG-07, PLUG-08, and PLUG-09.

## Files Modified
- `types/context.go`: Updated the `Context` struct to retain a global `PluginEngine` and a list of `loadedPlugins`, ensuring script state persists across standard daemon requests.
- `actions/plugin.go`: Implemented `PluginLoad`, `PluginList`, and `PluginRun` functions to directly interact with the plugin engine and daemon context.
- `daemon/daemon.go`: Extended the `executeAction` switch statement to accept `plugin-load`, `plugin-list`, and `plugin-run` commands.
- `cmd.go`: Added the `plugin` subcommand tree (`load`, `list`, `run`) using `urfave/cli/v2`, which correctly routes the user's terminal commands to the running daemon.

## Decisions Made
- Stored the plugin engine directly in `types.Context` to map engine state naturally to the lifecycle of the active daemon/browser session.
- Treated `PluginRun` primarily as a lifecycle stub for future targeted script executions, since plugins execute intrinsically on load due to `goja`'s parsing behavior.
