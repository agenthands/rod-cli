# Plugins

`rod-cli` runs user-written JavaScript **plugins** that hook into the browser lifecycle. A plugin is a plain `.js` file executed in an embedded goja sandbox — there is **no** `console`, `fs`, `require`, or `window`; only the lifecycle hooks and a small `api` global.

## Lifecycle hooks

Define any of these top-level functions in the plugin; the engine no-ops the omitted ones:

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
rod-cli plugin load /path/to/plugin.js           # load the plugin
rod-cli plugin list                              # list loaded plugins
rod-cli plugin run getResults                    # invoke a named function, print its JSON result
```

Pattern: a plugin accumulates state in module-level arrays inside its hooks and exposes a getter (e.g. `getResults`, `getFindings`) that returns `JSON.stringify(...)`. Use `plugin run <getter>` to read that state back from the CLI.

## Full documentation

The complete plugin docs (authoring guide, hooks and state API reference, CLI reference, examples) live in the `docs/plugins/` folder of the rod-cli repo: https://github.com/agenthands/rod-cli/tree/main/docs/plugins
