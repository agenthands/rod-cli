# Example: XSS Scanner

`plugins/examples/xss_scanner.js` is the flagship example plugin for `rod-cli`. It watches a browsing session for signs of reflected XSS: it records every outgoing request, then checks each response URL and the post-load DOM for echoes of a small set of known XSS payloads. Anything that looks reflected is recorded as a finding.

This page walks the scanner end-to-end — loading it, driving a vulnerable page so its hooks collect data, and then reading back what it found with `plugin run getFindings`. The payloads and matching logic live entirely in this user-space script; nothing exploit-specific is baked into the `rod-cli` binary.

## 1. Load the plugin

```bash
rod-cli plugin load ./plugins/examples/xss_scanner.js
```

Loading is what activates a plugin: `plugin load` compiles the script in the goja VM and — when a browser page is already open in the session — binds the `api` global and attaches the CDP lifecycle listeners to it. Open a page first (e.g. `rod-cli goto <url>`) so the listeners attach. From that point the scanner's hooks fire automatically as you browse — there is no separate "start" step. See [../cli-reference.md](../cli-reference.md) for the full `plugin load` behavior.

## 2. How the scanner watches the session

The scanner defines three lifecycle hooks. Each is optional — the engine silently no-ops any hook a plugin does not define — and each defends against a missing or partial event payload before reading fields. The exact payload shapes for these events are documented in [../lifecycle-hooks.md](../lifecycle-hooks.md); this page only describes how the scanner uses them.

- **`onRequest(event)`** — fires on every outgoing request. The scanner reads `event.Request.URL` and `event.Request.Method` and pushes `{ url, method }` onto its request log, so `getRequestLog()` can replay what the session touched.
- **`onResponse(event)`** — fires on each response. The scanner reads `event.Response.URL` and flags a `reflected_candidate` finding whenever the URL contains a URL-encoded echo of one of its payloads.
- **`onLoad(event)`** — fires after the page settles. The scanner reads the rendered DOM via `api.GetSnapshot()` and flags a `reflected_in_dom` finding for any payload that appears verbatim in the HTML, capturing a short surrounding evidence window. The `api` global is only present inside a loaded session, so this call is guarded by a `typeof api` check and a `try/catch`. The `api.GetSnapshot()` accessor and the rest of the state API are documented in [../state-api.md](../state-api.md).

Findings and the request log accumulate in module-level arrays for the life of the session; the accessors at the bottom of the script (`getFindings`, `getRequestLog`) read them back.

## 3. Drive a vulnerable page

With the scanner loaded, browse to a page that reflects input back into its response or DOM, and submit or follow one of the scanner's payloads. For example, navigate to a search or echo endpoint and supply a payload as a query parameter:

```bash
rod-cli goto "https://example.test/search?q=%3Cscript%3Ealert('XSS')%3C/script%3E"
```

As the request goes out and the response/DOM come back, `onRequest`, `onResponse`, and `onLoad` fire in turn and the scanner records what it sees. Use any page you control and are authorized to test — this page intentionally references only a generic example URL and embeds no exploit infrastructure or live targets.

## 4. Observe the findings

Read back what the scanner collected. `plugin run <name>` invokes the JS function named `<name>` in the loaded plugin's VM and prints its returned value; the scanner's accessors return `JSON.stringify(...)`, so the CLI prints clean JSON. See [../cli-reference.md](../cli-reference.md) for the full `plugin run` behavior and error conditions.

```bash
rod-cli plugin run getFindings
```

```json
[
  {
    "type": "reflected_in_dom",
    "payload": "<script>alert('XSS')</script>",
    "evidence": "...value=<script>alert('XSS')</script> name=q..."
  }
]
```

You can also dump every request the session touched:

```bash
rod-cli plugin run getRequestLog
```

```json
[{ "url": "https://example.test/search?q=...", "method": "GET" }]
```

If no payload was reflected, `getFindings` returns an empty array (`[]`) — the scanner saw the traffic but found nothing reflected.

## See Also

- [../lifecycle-hooks.md](../lifecycle-hooks.md) — the four lifecycle hooks and their event payload shapes (`event.Request`, `event.Response`).
- [../state-api.md](../state-api.md) — the `api` global the scanner uses to read the DOM (`api.GetSnapshot()`).
- [../cli-reference.md](../cli-reference.md) — `plugin load`, `plugin list`, and `plugin run <name>`.

## Source

The flagship script is [`../../plugins/examples/xss_scanner.js`](../../plugins/examples/xss_scanner.js). The engine that loads it and invokes its accessors lives in [`../../internal/plugin/engine.go`](../../internal/plugin/engine.go) (`LoadScript`, `RunFunc`); the hook → CDP event wiring and the `api` global are in [`../../internal/plugin/lifecycle.go`](../../internal/plugin/lifecycle.go) and [`../../internal/plugin/api.go`](../../internal/plugin/api.go).
