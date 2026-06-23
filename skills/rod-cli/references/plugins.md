# Plugins

`rod-cli` runs user-written JavaScript **plugins** that hook into the browser lifecycle. A plugin is a plain `.js` file executed in an embedded goja sandbox — there is **no** `console`, `fs`, `require`, or `window`; you get only the lifecycle hooks and a small `api` global.

## Lifecycle hooks

Define any of these top-level functions in your plugin; the engine no-ops the ones you omit:

| Hook | Fires on | Key payload |
|------|----------|-------------|
| `onRequest(event)` | each outgoing request | `event.Request.URL`, `event.Request.Method` |
| `onResponse(event)` | each response | `event.Response.URL`, `event.Response.Status` |
| `onLoad(event)` | page load | (use `api` to read the loaded page) |
| `onDOMNodeInserted(event)` | a node inserted into the DOM | `event.Node.NodeName` |

## The `api` global

Available after `plugin load` once a page is open:

- `api.GetSnapshot()` → full-page HTML string
- `api.GetCookies()` → cookie array
- `api.GetLocalStorage()` → `localStorage` as a key/value object

Network context comes from the `onRequest`/`onResponse` payloads, not from `api`.

## Commands

```bash
rod-cli goto https://example.com                 # open a page first so hooks/api bind
rod-cli plugin load ./plugins/examples/starter.js  # load the plugin
rod-cli plugin list                              # list loaded plugins
rod-cli plugin run getResults                    # invoke a named function, print its JSON result
```

Pattern: a plugin accumulates state in module-level arrays inside its hooks and exposes a getter (e.g. `getResults`, `getFindings`) that returns `JSON.stringify(...)`. `plugin run <getter>` is how you read that state back from the CLI.

## Full documentation

The complete plugin docs live under [`docs/plugins/`](../../../docs/plugins/README.md):

- [Authoring guide](../../../docs/plugins/authoring.md) — write, load, and run your first plugin end-to-end.
- [Lifecycle hooks reference](../../../docs/plugins/lifecycle-hooks.md)
- [State / context API reference](../../../docs/plugins/state-api.md)
- [CLI reference](../../../docs/plugins/cli-reference.md)
- Examples: [XSS scanner](../../../docs/plugins/examples/xss-scanner.md), [per-hook recipes](../../../docs/plugins/examples/recipes.md), [copyable starter](../../../docs/plugins/examples/starter.md).
