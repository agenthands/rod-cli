# Example: Per-Hook Recipes

The recipes under [`../../plugins/examples/recipes/`](../../plugins/examples/recipes/) are minimal, single-hook plugins — one script per lifecycle hook. Each defines exactly one hook, accumulates what that hook sees into a module-level array, and exposes a getter so you can read the collected state back with `plugin run <getter>`. They are the smallest possible demonstration of "load a plugin, drive the browser, watch a hook fire."

Each recipe is loadable on its own. The flow is always the same: `plugin load` the recipe to bind its single hook, drive the browser so that hook fires, then `plugin run <getter>` to print what it collected as JSON. The event payload shapes are documented once in [../lifecycle-hooks.md](../lifecycle-hooks.md); these recipes only show how each hook is wired and read back.

### onRequest — log every outgoing request

[`plugins/examples/recipes/on_request.js`](../../plugins/examples/recipes/on_request.js) records the URL and method of every outgoing network request.

```bash
# 1. Load the recipe (binds onRequest).
rod-cli plugin load ./plugins/examples/recipes/on_request.js

# 2. Drive the browser so requests go out.
rod-cli goto "https://example.test/"

# 3. Read back every request the session touched.
rod-cli plugin run getRequestLog
```

```json
[{ "url": "https://example.test/", "method": "GET" }]
```

`onRequest` fires before each request leaves the browser; the recipe reads `event.Request.URL` and `event.Request.Method`. See [../lifecycle-hooks.md](../lifecycle-hooks.md) for the full `event.Request` payload.

### onResponse — log every response

[`plugins/examples/recipes/on_response.js`](../../plugins/examples/recipes/on_response.js) records the URL and HTTP status of every response received.

```bash
# 1. Load the recipe (binds onResponse).
rod-cli plugin load ./plugins/examples/recipes/on_response.js

# 2. Drive the browser so responses arrive.
rod-cli goto "https://example.test/"

# 3. Read back the responses the session saw.
rod-cli plugin run getResponseLog
```

```json
[{ "url": "https://example.test/", "status": 200 }]
```

`onResponse` fires when a response arrives; the recipe reads `event.Response.URL` and `event.Response.Status`. See [../lifecycle-hooks.md](../lifecycle-hooks.md) for the full `event.Response` payload.

### onLoad — read page state after load

[`plugins/examples/recipes/on_load.js`](../../plugins/examples/recipes/on_load.js) reads the rendered DOM through the `api` global after each page load and records the length of the HTML snapshot.

```bash
# 1. Load the recipe (binds onLoad).
rod-cli plugin load ./plugins/examples/recipes/on_load.js

# 2. Drive the browser so a page settles and onLoad fires.
rod-cli goto "https://example.test/"

# 3. Read back the recorded load events.
rod-cli plugin run getLoadLog
```

```json
[{ "snapshotLength": 1256 }]
```

`onLoad` fires after the page settles. The recipe calls `api.GetSnapshot()` — the `api` global is only present inside a loaded session, so the call is guarded with a `typeof api` check and a `try/catch`. The `api.GetSnapshot()` accessor and the rest of the state API are documented in [../state-api.md](../state-api.md).

### onDOMNodeInserted — log inserted DOM nodes

[`plugins/examples/recipes/on_dom_node_inserted.js`](../../plugins/examples/recipes/on_dom_node_inserted.js) records the node name of every node inserted into the DOM.

```bash
# 1. Load the recipe (binds onDOMNodeInserted).
rod-cli plugin load ./plugins/examples/recipes/on_dom_node_inserted.js

# 2. Drive the browser to a page that inserts DOM nodes (e.g. a page that
#    renders content dynamically after load).
rod-cli goto "https://example.test/"

# 3. Read back the names of the inserted nodes.
rod-cli plugin run getInsertedNodes
```

```json
["DIV", "SPAN", "SCRIPT"]
```

`onDOMNodeInserted` fires each time a child node is inserted; the recipe reads `event.Node.NodeName` (`event.ParentNodeID` identifies the parent). See [../lifecycle-hooks.md](../lifecycle-hooks.md) for the full payload.

## See Also

- [../lifecycle-hooks.md](../lifecycle-hooks.md) — the four lifecycle hooks and their event payload shapes (`event.Request`, `event.Response`, `event.Node`).
- [../state-api.md](../state-api.md) — the `api` global the `onLoad` recipe uses to read the DOM (`api.GetSnapshot()`).
- [../cli-reference.md](../cli-reference.md) — `plugin load`, `plugin list`, and `plugin run <name>`.
- [./xss-scanner.md](./xss-scanner.md) — the flagship example that combines `onRequest`, `onResponse`, and `onLoad` into a working scanner.

## Source

The recipe scripts live under [`../../plugins/examples/recipes/`](../../plugins/examples/recipes/). The engine that loads them and invokes their getters lives in [`../../internal/plugin/engine.go`](../../internal/plugin/engine.go) (`LoadScript`, `RunFunc`); the hook → CDP event wiring and the `api` global are in [`../../internal/plugin/lifecycle.go`](../../internal/plugin/lifecycle.go) and [`../../internal/plugin/api.go`](../../internal/plugin/api.go).
