# Running Code in the Browser

Evaluate custom JavaScript code directly in the browser's context using the `eval` command.

## Basic Evaluation
Without a target, the script is executed in the context of the global `window`.

```bash
rod-cli eval "document.title"
rod-cli eval "performance.timing.navigationStart"
```

## Targeting Specific Elements
To evaluate code against a specific element from the snapshot, provide the element reference as the second argument. This ref is the exact `[ref=…]` token printed by `rod-cli snapshot` next to each element (e.g. `s1e5`) — copy it verbatim. Refs are scoped to the snapshot that produced them: they change on every navigation/reload and whenever the page's DOM changes significantly, so re-run `snapshot` to get a fresh ref if one stops resolving. The script is evaluated with `this` bound to the DOM element.

```bash
# Get text content of an element
rod-cli eval "() => this.textContent" s1e5
```

To mark an element visually, use `rod-cli highlight <ref>` instead of changing its style with `eval`; `eval` is for reading only.

A failing `click <ref-or-selector>` (or `fill`/`type`/etc.) almost always means the ref is stale or the CSS selector no longer matches — re-run `snapshot` for a fresh ref, or re-check the selector. Do **not** work around it with `eval "() => this.click()"`: that dispatches an untrusted, non-humanized synthetic click (`event.isTrusted === false`), defeating this tool's stealth/humanization design. Use the native `click`/`fill`/`type`/etc. commands for interaction; reserve `eval` for reading and extracting page data.

## Extracting Data

Use standard DOM APIs to extract complex data structures. eval prints the bare value. With --json, a JSON object/array value prints as-is; any other value prints as {"result": "<value>"}.

```bash
rod-cli eval "JSON.stringify([...document.querySelectorAll('a')].map(a => a.href))"
```

## Read recipes

```bash
# Main readable text of the page
rod-cli eval "(document.querySelector('main,article,[role=main]')||document.body).innerText"

# HTML of one element
rod-cli eval "() => this.outerHTML" <ref>

# Bring an element into view before a screenshot
rod-cli eval "() => this.scrollIntoView({block:'center'})" <ref>

# Call the site's own API with the page's cookies
rod-cli eval "fetch('/api/items',{credentials:'include'}).then(r=>r.text())"
```

The `fetch` recipe runs on Chrome's own network stack (real TLS, cookies, headers); never fetch from Go.
