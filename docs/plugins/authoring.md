# Writing Your First Plugin

This is a zero-to-running-plugin tutorial: copy a starter, fill in one hook, load it, drive a page, and read the result back. It is the narrative glue for the plugin docs — it links out to the reference pages and worked examples rather than restating them, so reach for those when you want the full details.

## 1. What a plugin is

A `rod-cli` plugin is a plain JavaScript file that defines one or more optional lifecycle hooks. The file runs in the [goja](https://github.com/dop251/goja) sandbox, which is deliberately minimal: there is **no `console`, no `fs`, no `require`, and no `import`**. The only things your script can touch are:

- the **hook payloads** the engine passes in — `event.Request`, `event.Response`, and `event.Node`, and
- the **`api` global** — a small set of page-state accessors that bind once a page is open.

There are four hooks (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`); every one is optional, and an undefined hook is silently no-op'd. See [./lifecycle-hooks.md](./lifecycle-hooks.md) for the hook list and payload shapes, and [./state-api.md](./state-api.md) for the `api` accessors.

## 2. Project structure

A plugin is a single `.js` file and can live at any filesystem path — you pass that path to `plugin load`. The fastest way to start is to copy the Phase 22 starter scaffold, which already wires up the four hook stubs and a `getResults` accessor:

```bash
cp ./plugins/examples/starter.js ./my_plugin.js
```

The starter loads and runs unchanged (doing nothing) so you can confirm your pipeline works before writing any logic. See [./examples/starter.md](./examples/starter.md) for the full starter walkthrough.

## 3. Writing handlers

Open `my_plugin.js` and fill in the hook you need. The common pattern is a module-level array, one hook that pushes into it, and a getter that returns it as JSON. Here we record every response status and URL:

```javascript
var results = [];

// onResponse fires when a response arrives.
// Payload: event.Response.URL, event.Response.Status, event.Response.Headers
function onResponse(event) {
  results.push({ url: event.Response.URL, status: event.Response.Status });
}

// getResults returns the collected state as JSON for `plugin run getResults`.
function getResults() {
  return JSON.stringify(results);
}
```

Delete the hooks you don't use — an undefined hook changes nothing. See [./lifecycle-hooks.md](./lifecycle-hooks.md) for the exact payload shapes and [./examples/recipes.md](./examples/recipes.md) for a per-hook recipe you can crib from.

## 4. Load it

Load the script into the active daemon session:

```bash
rod-cli plugin load ./my_plugin.js
```

On success the command prints the literal line:

```
Plugin loaded successfully from ./my_plugin.js
```

Loading is what activates the hooks — they begin firing from this point on. See [./cli-reference.md](./cli-reference.md) for the full `plugin load` behavior and error conditions.

## 5. Open a page so hooks attach

The `api` global and the CDP lifecycle listeners bind only when a browser page is open in the session. Until then, hooks have nothing to fire against. Open and drive a page so the listeners attach and your hook starts accumulating state:

```bash
rod-cli goto https://example.test
```

As the page loads and makes requests, your `onResponse` hook runs and fills the `results` array.

## 6. Inspect results with `plugin run`

Read the collected state back by invoking your getter by name. `plugin run <name>` calls the named top-level function in the loaded plugin and prints its stringified result — and because the getter calls `JSON.stringify(...)`, the output is clean JSON:

```bash
rod-cli plugin run getResults
```

```json
[{"url":"https://example.test/","status":200}]
```

Any accessor name works — the example plugins expose `getFindings` and `getRequestLog` the same way. See [./cli-reference.md](./cli-reference.md) for `plugin run` behavior and errors, and [./examples/xss-scanner.md](./examples/xss-scanner.md) for a fully worked load → drive → `plugin run` flow.

## 7. Next steps

You now have a running plugin. To go deeper:

- **[./lifecycle-hooks.md](./lifecycle-hooks.md)** — every hook and its event payload shape.
- **[./state-api.md](./state-api.md)** — the `api` global (`GetSnapshot`, `GetCookies`, `GetLocalStorage`) for reading page state inside hooks.
- **[./cli-reference.md](./cli-reference.md)** — full `plugin load`, `plugin list`, and `plugin run` reference.
- **[./examples/starter.md](./examples/starter.md)** — the copyable scaffold this tutorial started from.
- **[./examples/recipes.md](./examples/recipes.md)** — single-hook recipes, one per lifecycle hook.
- **[./examples/xss-scanner.md](./examples/xss-scanner.md)** — a polished, fully worked plugin built from these same pieces.

## See Also

- [./lifecycle-hooks.md](./lifecycle-hooks.md) — the four lifecycle hooks and their payloads.
- [./state-api.md](./state-api.md) — the `api` global for reading page state.
- [./cli-reference.md](./cli-reference.md) — `plugin load`, `plugin list`, and `plugin run <name>`.
- [./examples/starter.md](./examples/starter.md) — the copyable starter template.
- [./examples/recipes.md](./examples/recipes.md) — per-hook recipes.
- [./examples/xss-scanner.md](./examples/xss-scanner.md) — a fully worked example.

## Source

The engine that loads plugins and invokes named functions lives in [`../../internal/plugin/engine.go`](../../internal/plugin/engine.go) (`LoadScript`, `RunFunc`). The hook → CDP event wiring and the `api` global are in [`../../internal/plugin/lifecycle.go`](../../internal/plugin/lifecycle.go) and [`../../internal/plugin/api.go`](../../internal/plugin/api.go).
