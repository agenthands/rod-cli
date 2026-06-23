# Example: Starter Template

[`plugins/examples/starter.js`](../../plugins/examples/starter.js) is a copyable scaffold for writing your own `rod-cli` plugin. It defines empty stubs for all four lifecycle hooks (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`) plus a `getResults` accessor that returns a `results` array as JSON. Every hook is optional — the engine silently no-ops any hook a plugin does not define — so the starter loads and runs unchanged, doing nothing until you fill in the hooks you care about and push into `results`.

The workflow is copy-and-fill: copy the starter to your own file, fill in the hook bodies you need (and delete the ones you don't), then load and run it. Because the stubs are inert, the unmodified starter is also a sanity check that your plugin pipeline works end-to-end before you write any logic.

## Copy, load, and run it unchanged

```bash
# 1. Copy the starter to your own plugin file.
cp ./plugins/examples/starter.js ./my_plugin.js

# 2. Load it. With no hook bodies filled in, it loads cleanly and the
#    four (empty) hooks no-op as you browse.
rod-cli plugin load ./my_plugin.js

# 3. Confirm it runs. getResults returns the empty results array as JSON
#    until you populate `results` from inside a hook.
rod-cli plugin run getResults
```

```json
[]
```

A clean load with no errors and an empty-array response from `getResults` confirms the plugin compiled in the goja VM. The `api` global and the CDP lifecycle listeners bind when a browser page is open in the session — so open a page (e.g. `rod-cli goto <url>`) to see the hooks fire. See [../cli-reference.md](../cli-reference.md) for the full `plugin load` and `plugin run` behavior and error conditions.

## How the template is structured

The starter runs in the goja sandbox: there is no `console`, `fs`, `require`, or `import`. You work only with the hook payloads and the `api` global.

- **`results`** — a module-level array that lives for the session. Fill it from your hooks.
- **`onRequest(event)` / `onResponse(event)` / `onLoad(event)` / `onDOMNodeInserted(event)`** — empty stubs, one per lifecycle hook. Fill in the bodies you need and remove the rest; an undefined hook changes nothing.
- **`getResults()`** — returns `JSON.stringify(results)` so `plugin run getResults` prints clean JSON. Rename or add accessors as your plugin grows.

## Next steps

To turn the scaffold into a real plugin:

- See [../lifecycle-hooks.md](../lifecycle-hooks.md) for which hooks exist and the exact payload shapes (`event.Request`, `event.Response`, `event.Node`).
- See [../state-api.md](../state-api.md) for the `api` global available inside hooks (e.g. `api.GetSnapshot()` to read the rendered DOM).
- See [./recipes.md](./recipes.md) for per-hook recipes you can crib from — each shows one hook filled in and read back via `plugin run`.

## See Also

- [../lifecycle-hooks.md](../lifecycle-hooks.md) — the four lifecycle hooks and their event payload shapes.
- [../state-api.md](../state-api.md) — the `api` global for reading page state from inside hooks.
- [../cli-reference.md](../cli-reference.md) — `plugin load`, `plugin list`, and `plugin run <name>`.
- [./recipes.md](./recipes.md) — single-hook recipes to crib hook bodies from.
- [./xss-scanner.md](./xss-scanner.md) — a fully worked example built from these same hooks.

## Source

The starter template is [`../../plugins/examples/starter.js`](../../plugins/examples/starter.js). The engine that loads it and invokes `getResults` lives in [`../../internal/plugin/engine.go`](../../internal/plugin/engine.go) (`LoadScript`, `RunFunc`); the hook → CDP event wiring and the `api` global are in [`../../internal/plugin/lifecycle.go`](../../internal/plugin/lifecycle.go) and [`../../internal/plugin/api.go`](../../internal/plugin/api.go).
