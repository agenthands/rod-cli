# Lifecycle Hooks

`rod-cli` plugins react to browser activity by defining lifecycle hook functions in their JavaScript source. When a plugin is loaded, the engine attaches CDP (Chrome DevTools Protocol) event listeners to the active page and forwards each event to the matching JS handler. You write a plain top-level function for each event you care about — the engine calls it with the raw CDP event payload.

## Hook Mapping

The engine wires exactly four hooks. Each row below maps the JavaScript handler name you define (lowercase, what plugin authors write) to the CDP event it fires on (the go-rod `proto` type) and the payload fields you reach most often. The internal Go `LifecycleEmitter` interface uses the uppercase forms (`OnRequest`, `OnResponse`, `OnLoad`, `OnDOMNodeInserted`); those are implementation detail — your plugin defines the lowercase JS names.

| JS handler | CDP event (proto type) | Key payload fields |
| --- | --- | --- |
| `onRequest` | `proto.NetworkRequestWillBeSent` | `event.Request.URL`, `event.Request.Method`, `event.Request.Headers` |
| `onResponse` | `proto.NetworkResponseReceived` | `event.Response.URL`, `event.Response.Status`, `event.Response.Headers` |
| `onLoad` | `proto.PageLoadEventFired` | load `event.Timestamp` |
| `onDOMNodeInserted` | `proto.DOMChildNodeInserted` | `event.ParentNodeID`, `event.Node` (the inserted node) |

The table lists only the commonly-used fields. For the complete field set of each payload, consult the go-rod proto type by its Go name (`proto.NetworkRequestWillBeSent`, `proto.NetworkResponseReceived`, `proto.PageLoadEventFired`, `proto.DOMChildNodeInserted`) in the [go-rod CDP proto reference](https://pkg.go.dev/github.com/go-rod/rod/lib/proto). The event object passed into your handler is the JSON projection of that Go struct, so its field names match the Go field names (e.g. `event.Request.URL`).

## Handlers Are Optional

Every hook is optional. The engine's `invokeJSFunc` looks up the handler by name on the JS runtime and silently no-ops if the function is not defined. Define only the hooks your plugin needs — there is no base class to extend and no registration call to make. A plugin that defines only `onResponse` works fine; the other three events are simply ignored.

## The `api` Global

The `api` global (page state/context accessors such as `api.GetSnapshot()`) is bound to the JS runtime when the engine runs `BindLifecycle`, which happens on `plugin load`. It is not available at script parse time — only inside your hook handlers, which run after load. Guard against an undefined `api` (`typeof api !== "undefined"`) if your code must tolerate being evaluated outside a loaded session. See [state-api.md](./state-api.md) for the full state API.

## Worked Snippets

### onRequest

Fires before each network request leaves the browser. Read the outgoing URL and method:

```javascript
function onRequest(event) {
  if (event && event.Request && event.Request.URL) {
    requestLog.push({
      url: event.Request.URL,
      method: event.Request.Method || "GET",
    });
  }
}
```

### onResponse

Fires when a response arrives. Inspect the response URL and status code:

```javascript
function onResponse(event) {
  if (event && event.Response) {
    var url = event.Response.URL || "";
    if (event.Response.Status >= 400) {
      // No console in the goja sandbox — record it instead, e.g. push to a
      // module-level array and read it back with `plugin run <getter>`.
    }
  }
}
```

### onLoad

Fires when the page finishes loading. This is the natural point to read page state through the `api` global — for example, taking an HTML snapshot:

```javascript
function onLoad(event) {
  if (typeof api !== "undefined") {
    try {
      var snapshot = api.GetSnapshot();
      // inspect the rendered DOM after load
    } catch (e) {
      // api may not be available in all contexts
    }
  }
}
```

### onDOMNodeInserted

Fires each time a child node is inserted into the DOM, carrying the parent node id and the inserted node. The shipped example plugin does not use this hook, so here is a minimal stub that reads the inserted node:

```javascript
function onDOMNodeInserted(event) {
  if (event && event.Node) {
    // event.ParentNodeID is the parent; event.Node is the newly inserted node
    insertedNodes.push(event.Node.NodeName);
  }
}
```

## See Also

- [state-api.md](./state-api.md) — the `api` global's page state accessors. Note: network request/response context is reached through these hook payloads (`event.Request`, `event.Response`), **not** through an `api` method.
- [cli-reference.md](./cli-reference.md) — how to load a plugin so its hooks begin firing (`plugin load <path>`).

## Source

The hook → event mapping documented here is wired in [`internal/plugin/lifecycle.go`](../../internal/plugin/lifecycle.go) (`BindLifecycle` attaches the CDP listeners and sets the `api` global; `invokeJSFunc` performs the optional-handler no-op). Worked snippets are adapted from `plugins/examples/xss_scanner.js`.
