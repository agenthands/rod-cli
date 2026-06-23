# State / Context API

A loaded `rod-cli` plugin reads the live page's state through the `api` global. The engine binds `api` to the JavaScript runtime when it runs `BindLifecycle` (on `plugin load`), exposing a small set of accessors that wrap the underlying go-rod page. Each accessor mirrors the same shape: it reads from the bound page and returns its value, no-opping to a nil/empty result when no page is attached.

This page documents the three accessors that exist in `internal/plugin/api.go` — `GetSnapshot()`, `GetCookies()`, and `GetLocalStorage()` — with their return shapes and a worked snippet. Network request/response context is **not** an `api` method; it arrives through the lifecycle hook payloads (see [Network context](#network-context) below).

## When `api` Is Available

The `api` global is bound only **after** `BindLifecycle` runs, which happens on `plugin load`. It is not available at script parse time — only inside your hook handlers, which run after load. If your code might be evaluated outside a loaded session, guard against an undefined `api`:

```javascript
if (typeof api !== "undefined") {
  // safe to call api.GetSnapshot(), api.GetCookies(), api.GetLocalStorage()
}
```

See [cli-reference.md](./cli-reference.md) for how to load a plugin (`plugin load <path>`), which is what binds `api` and starts the hooks firing.

## `api.GetSnapshot()`

Returns the **full-page HTML** of the current page as a string. It is backed by go-rod's `page.HTML()`, so you get the rendered DOM at the moment of the call — the natural point to call it is inside `onLoad`, after the page has finished loading.

**Return shape:** a single HTML `string`.

> **Token-cost caveat:** a snapshot is the entire serialized DOM and can be very large. If you only need a fragment, search or substring the returned string rather than logging or storing the whole snapshot.

Worked snippet, adapted from the `onLoad` handler in `plugins/examples/xss_scanner.js`:

```javascript
function onLoad(event) {
  if (typeof api !== "undefined") {
    try {
      var snapshot = api.GetSnapshot();
      // scan the rendered HTML for a marker
      if (snapshot.indexOf("<script>alert('XSS')</script>") > -1) {
        findings.push({ type: "reflected_in_dom" });
      }
    } catch (e) {
      // api may not be available in all contexts
    }
  }
}
```

## `api.GetCookies()`

Returns the page's **cookie array** — the CDP cookie objects collected via `page.Browser().GetCookies()`, handed back as a structured value.

**Return shape:** an array of cookie objects. The fields a plugin reads most often are:

| Field | Meaning |
| --- | --- |
| `Name` | the cookie name |
| `Value` | the cookie value |
| `Domain` | the domain the cookie is scoped to |
| `Path` | the path the cookie is scoped to |

These are only the commonly-used fields. For the complete cookie shape (expiry, `HTTPOnly`, `Secure`, `SameSite`, etc.) consult the go-rod `proto.NetworkCookie` type in the [go-rod CDP proto reference](https://pkg.go.dev/github.com/go-rod/rod/lib/proto#NetworkCookie). The objects passed to JS are the JSON projection of that Go struct, so field names match.

```javascript
function onLoad(event) {
  if (typeof api !== "undefined") {
    var cookies = api.GetCookies();
    for (var i = 0; i < cookies.length; i++) {
      if (cookies[i].Name === "session") {
        // Found the session cookie — record it (no console in the sandbox;
        // accumulate into a module-level array and read it back via `plugin run`).
      }
    }
  }
}
```

## `api.GetLocalStorage()`

Returns the current page's `window.localStorage` as a **key/value object**. The accessor evaluates a small JS function in the page that iterates every `localStorage` key into a plain object; go-rod/gson hands that object back through `Val()`, so on the Go side it is a `map[string]interface{}` and inside your plugin JS it is a plain object keyed by storage name.

**Return shape:** a key/value object — keys are the `localStorage` keys, values are their string contents.

Like the other accessors, it no-ops to a nil result when no page is bound.

```javascript
function onLoad(event) {
  if (typeof api !== "undefined") {
    var store = api.GetLocalStorage();
    if (store && store["authToken"]) {
      // authToken present — record it (sandbox has no console; accumulate and
      // read back via `plugin run`).
    }
  }
}
```

## Network context

There is **no `api` method for network requests or responses.** Network context is reached through the lifecycle hook event payloads, not through the `api` global:

- `onRequest(event)` → outgoing request: `event.Request.URL`, `event.Request.Method` (and headers).
- `onResponse(event)` → incoming response: `event.Response.URL`, `event.Response.Status` (and headers).

Define those handlers to observe traffic; the engine delivers the raw CDP payload to each. The full payload shapes and per-hook worked snippets are documented in [lifecycle-hooks.md](./lifecycle-hooks.md).

```javascript
function onResponse(event) {
  if (event && event.Response && event.Response.Status >= 400) {
    // Error response — record it (no console; accumulate to an array exposed
    // via a getter, then `plugin run <getter>`).
  }
}
```

## See Also

- [lifecycle-hooks.md](./lifecycle-hooks.md) — the four lifecycle hooks; the source of network request/response context and the place where `api` becomes available.
- [cli-reference.md](./cli-reference.md) — `plugin load <path>`, which binds the `api` global and starts hooks firing.

## Source

The accessors documented here are defined in [`internal/plugin/api.go`](../../internal/plugin/api.go): `GetSnapshot()` (returns `page.HTML()`), `GetCookies()` (returns `page.Browser().GetCookies()`), and `GetLocalStorage()` (evaluates a `window.localStorage` iterator via `page.Eval`). The `api` global is bound to the JS runtime by `BindLifecycle` in `internal/plugin/lifecycle.go`. Worked snippets are adapted from `plugins/examples/xss_scanner.js`.
