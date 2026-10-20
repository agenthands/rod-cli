---
name: rod-cli
description: "Browse the web, scrape a page, fill forms, click elements, take screenshots, run JavaScript on a page, or automate any browser task with rod-cli, the token-efficient Go Rod browser CLI."
version: v1.7
---

# Go Rod Skill (`rod-cli`)

`rod-cli` is a native Go Rod web automation CLI for agents. It returns token-efficient, accessibility-tree-based DOM snapshots instead of raw HTML. Prefer it over curl/wget for JavaScript-rendered pages, and over heavy browser automation frameworks.

## Core Concepts

1. **Daemon Architecture**: `rod-cli` runs as a persistent background daemon that holds state.
   - The first command boots a persistent browser in the background.
   - **Idle timeout**: the daemon outlives the command that started it and exits after 15 minutes without a command (`sessions` only pings it, which does not count).
   - **Always** run `rod-cli close` when finished to clean up resources.
   - **Profiles**: each session's Chrome profile lives in `~/.cache/rod-cli/profiles/<id>` on Linux (the user cache dir: on disk, never `/tmp` or the working directory; `browserTempDir` in `./rod-cli.yaml` overrides the location). Chrome's lock dir and temp files go to a per-session `chrome-*` dir in the system temp dir. `close`, the idle timeout and SIGINT/SIGTERM/SIGHUP remove both. A daemon killed with `kill -9` or crashed leaves them; the next session that launches Chrome removes them (default location only). Every session file and its cleanup: [references/session-management.md](references/session-management.md#session-files).

2. **Named Sessions**:
   - For multi-target workflows (e.g., logging in as two distinct users simultaneously), use the global `--session` or `-s` flag to multiplex daemons.
   - `rod-cli -s=admin goto http://...`
   - `rod-cli -s=guest goto http://...`

3. **Element Targeting**: Every command below that takes `[selector]` (`click`, `dblclick`, `type`, `fill`, `select`, `check`, `uncheck`, `hover`, `upload`, `drag`, `drop`, `highlight`) accepts **either**:
   - a plain **CSS selector** (e.g. `#submit-button`, `.nav-link`) — resolves immediately, even before any `snapshot` call, or
   - an opaque **snapshot ref** — the literal token from a `[ref=sNeM]` annotation printed by `rod-cli snapshot` (e.g. `s1e5`), copied verbatim. Use a ref when no stable CSS selector exists, or to disambiguate elements. Refs are tied to the snapshot that produced them: they stop resolving after the next `snapshot`, a navigation or a tab switch — re-run `snapshot` for a fresh ref.

4. **Prefer Native Actions over `eval`**: Use the native commands to interact with the page — not `eval`.
   - `click`/`dblclick`/`fill`/`type`/`hover`/`drag`/`mousewheel` dispatch real, trusted, humanized input: genuine CDP mouse movement and wheel events (tunable tremor/speed/deviation) and per-keystroke typing cadence, which this tool's stealth design depends on.
   - `select`/`check`/`uncheck` set state through JS and fire synthetic events (`isTrusted=false`), but they are still preferred over hand-written `eval`.

   Simulating interaction via `eval` (`el.click()`, setting `.value` directly, dispatching synthetic events) produces untrusted (`event.isTrusted === false`) events with none of that humanization; checking `isTrusted` on interaction events is a standard headless-detection technique, making eval-driven interaction a strong, trivially-checked bot signal. Reserve `eval` for reading/extracting page state — attributes, computed values, JSON data — never for acting on the page.

## Commands

**Flag placement**: Command-specific flags must come **before** the command's positional arguments (e.g. `rod-cli fill --submit "#q" "text"`). This applies to every command-local flag in this skill: `fill --submit`, `drop --path`, `route --body`, `screenshot --name`/`--selector`/`--out`/`--full-page`, `pdf --name`/`--out`, `console --clear`, `requests --clear`, `mousewheel --end`/`--max`, `snapshot -i`, `dialog-accept --prompt`, `show --annotate`. A flag placed after a positional is treated as a positional itself and silently ignored. Global flags (`--session`, `--json`, `--raw`) may go anywhere.

### Global Options
- `--max-output N`: Cap any command's output at N bytes (0 = unlimited, the default). Use it to bound token cost on big pages, e.g. `rod-cli --max-output 4000 snapshot`.
- `--json`: Wraps any command's result in structured JSON (`{"result": "…"}`, or passed through verbatim when the daemon already returns structured JSON, e.g. `stealth-check`). Useful when a caller needs to parse output programmatically.
- `--raw`: Only changes output for `install` (suppresses the "Downloading…" progress line), `stealth-check` (single-line PASS/FAIL instead of a table) and `--profile=list` (names only). For every other command (`goto`, `snapshot`, `eval`, `close`, `sessions`, …) it is a no-op, byte-identical to omitting the flag — omit it there.
- The startup banner is off by default (bare `rod-cli` only); pass `--banner`/`-b` to show it.
- `-s, --session`: Specify a named session string. Session names use letters, digits, `.`, `_` or `-` (1 to 64 characters, not `.` or `..`); any other name fails with `invalid session name`.

Session-spawn flags take effect only when a session's daemon spawns (its first command). To change them, run `rod-cli close` first, then pass them on the next command:
- `--headless`: enable headless.
- `--proxy`: route this session through an HTTP or SOCKS5 proxy (scheme in URL, e.g. http://host:port or socks5://host:port).
- `--proxy-auth`: proxy credentials as user:pass (handled via CDP, never URL-embedded).
- `--profile`: stealth profile: a built-in name (see --profile=list), a custom name resolved under ~/.rod-cli/profiles/, or a path to a JSON profile file.
- `--user-agent`: pin navigator.userAgent / HTTP UA for this session (the fingerprint derivation anchor).
- `--locale`: pin the BCP-47 locale (e.g. en-US); derived from languages when unset.
- `--timezone`: pin the IANA timezone (e.g. America/New_York).
- `--platform`: pin navigator.platform (e.g. Win32, MacIntel, Linux); derived from the UA OS token when unset, so `--user-agent` also replaces a `--profile`'s platform.
- `--cdp-endpoint`: control browser by cdp.
- `--console-capture`: Capture browser console messages for the console command (default off; enables Runtime CDP domain).
- `--request-capture`: Capture network requests for the requests/`request` commands (default off; enables Network CDP domain).

### Setup
- `rod-cli install`: Install the Chromium browser required by rod-cli.

### Browser Lifecycle & Tabs
- `rod-cli sessions`: List all currently active background daemon sessions.
- `rod-cli close`: Terminate the background daemon and clean up the browser.
- `rod-cli open [url]` (or `goto`): Navigate to a URL.
- `rod-cli reload`, `go-back`, `go-forward`: Standard navigation.
- `rod-cli resize [width] [height]`: Resize the browser window.
- `rod-cli tab-list`: List all open tabs as `[index] <8-char ID> <title> (<url>)`; the current tab is marked `*`.
- `rod-cli tab-new [url]`: Create a new tab (prints its ID) and switch to it.
- `rod-cli tab-select [index|id]`: Select a browser tab by index or 8-char ID (prefer the ID; indexes shift).
- `rod-cli tab-close [index|id]`: Close a browser tab by index or 8-char ID.
- Rule: Drive only tabs created with tab-new (it prints their IDs); under --cdp-endpoint never navigate the user's existing tabs.

### Interaction
`[selector]` below accepts a CSS selector or a snapshot ref — see **Element Targeting** in Core Concepts above.

Use a selector or ref first; use pixel input (mousemove/mousedown/mouseup) only when no ref or selector works (canvas, sliders without ARIA, hover-only targets). Coordinates are viewport CSS px.

- `rod-cli click [selector]`: Click an element.
- `rod-cli dblclick [selector]`: Double-click an element.
- `rod-cli type [selector] [text]`: Type sequentially into an element.
- `rod-cli fill [selector] [text]`: Fill an entire input at once. Use `--submit` to press enter afterwards.
- `rod-cli select [selector] [values...]`: Select dropdown options. Each value matches an option by exact text, then by `value`, then by case-insensitive text substring; every value must match or the command fails. Prints the selected option labels.
- `rod-cli check [selector]`: Check a checkbox or radio button.
- `rod-cli uncheck [selector]`: Uncheck a checkbox or radio button.
- `rod-cli hover [selector]`: Hover over an element.
- `rod-cli upload [selector] [file]`: Upload a file to an element.
- `rod-cli drag [start-selector] [end-selector]`: Drag one element to another.
- `rod-cli drop --path [file-path] [selector]`: Drop a file onto an element.

### Advanced Input & Dialogs
- `rod-cli press [key]`: Simulate a raw keyboard key press. Key names: `Enter`, `Tab`, `Backspace`, `Escape`, `Space`, `Delete`, `Insert`, `Home`, `End`, `PageUp`, `PageDown`, `ArrowUp`, `ArrowDown`, `ArrowLeft`, `ArrowRight`, `Shift`, `Control`, `Alt`, `Meta`, `CapsLock`, `ContextMenu`, `F1`–`F12`, or one US-keyboard character (`a`, `A`, `1`, `/`). Any other key (such as `é`) fails with `unknown key`.
- `rod-cli mousemove [x] [y]`: Move the mouse to absolute coordinates.
- `rod-cli mousedown [left|right|middle]`: Trigger mouse down.
- `rod-cli mouseup [left|right|middle]`: Trigger mouse up.
- `rod-cli mousewheel [--end [--max n]] [dx dy] [ref]`: Scroll with real wheel input. No `dx dy` scrolls one viewport down; a `ref` scrolls that element's nearest scrollable pane instead of the page. `--end` keeps scrolling (absorbing lazy loads) until the end, at most `--max` scrolls (default 10). Negative values need `--`: `rod-cli mousewheel --end -- 0 -300`. Output: `scrolled dy=<moved> y=<pos>/<max>` (plus `dx=… x=…/…` for horizontal moves); a trailing ` end` means the pane hit its limit, ` cap` means `--end` stopped at the `--max`/time budget before reaching it.
- `rod-cli dialog-accept [--prompt <text>]`: Automatically accept the next javascript alert/confirm/prompt; `--prompt` is the text entered into a `window.prompt()`.
- `rod-cli dialog-dismiss`: Automatically dismiss the next javascript alert/confirm.

### Storage & State Controls
- `rod-cli state-save [path]`: Save the browser's cookies to a JSON file (path required). The file is written owner-only (0600) because it holds session cookies.
- `rod-cli state-load [path]`: Load cookies saved by `state-save` into the session (path required).
- `rod-cli cookie-get [name]`: Read all browser cookies, or only the cookie with that exact name (errors if none matches).
- `rod-cli cookie-set [name] [value]`: Set a cookie.
- `rod-cli cookie-delete [name]`: Delete a specific cookie.
- `rod-cli cookie-clear`: Clear all browser cookies.
- `rod-cli localstorage-get [key]`: Retrieve a localStorage item (or omit key for all).
- `rod-cli localstorage-set [key] [value]`: Set a localStorage item.
- `rod-cli localstorage-delete [key]`: Delete a localStorage entry.
- `rod-cli localstorage-clear`: Clear localStorage.
- `rod-cli sessionstorage-get [key]`: Retrieve a sessionStorage item (or omit key for all).
- `rod-cli sessionstorage-set [key] [value]`: Set a sessionStorage item.
- `rod-cli sessionstorage-delete [key]`: Delete a sessionStorage entry.
- `rod-cli sessionstorage-clear`: Clear sessionStorage.

### Observability & Debugging
- `rod-cli highlight [selector]`: Highlight an element with a persistent red border (useful for visual reasoning validation before clicking). Accepts the same CSS-selector-or-snapshot-ref `[selector]` as the Interaction commands above.
- `rod-cli highlight-clear`: Remove all injected highlights from the DOM.
- `rod-cli show`: Without flags, prints how to make the browser visible (it cannot un-hide a headless browser: `close`, then restart without `--headless`).
- `rod-cli show --annotate`: Provide human-in-the-loop interactive bounding box feedback.
- `rod-cli console [--clear]`: List browser console messages; uncaught page errors appear as `[exception] …` entries. `--clear` prints the entries, then empties the buffer. Filter with the shell (`rod-cli console | grep exception`, `| tail -20`). Only works when the session was started with `--console-capture` (off by default) — pass it on the session's first command, e.g. `rod-cli --console-capture goto https://example.com`.

### Network Interception
- `rod-cli route --body [body] [pattern]`: Mock network requests matching a URL pattern via the browser's own network interception.
- `rod-cli route-list`: List all active network interception routes.
- `rod-cli unroute [pattern]`: Remove a mock network route.
- `rod-cli requests`: List all captured network requests. Only works when the session was started with `--request-capture` (off by default) — pass it on the session's first command.
- `rod-cli request [index]`: Show details for a specific request (also requires the session started with `--request-capture`).

### Evaluation & Export
- `rod-cli eval [script] [ref]`: Evaluate raw JavaScript in the browser context and print the bare value (no wrapper; strings unquoted). With a ref, `this` is bound to that element. Use it for reading/extracting page state (attributes, computed values, JSON data) — prefer the Interaction commands above (`click`, `fill`, `type`, etc.) for anything that acts on the page; see **Prefer Native Actions over `eval`** in Core Concepts.
- `rod-cli snapshot [-i] [ref|css]`: Return the token-efficient Markdown representation of the DOM. `-i`/`--interactive` keeps only interactive elements and their ancestors; a ref or CSS selector limits it to that subtree. Both cut tokens on large pages.
- `rod-cli screenshot [--name] [--selector] [--out] [--full-page]`: Capture a PNG of the viewport (`--selector` = one element by ref or CSS, `--full-page` = whole scrollable page, `--out` = write to this path instead of `tmp/screenshots/`). Prints `saved <path> WxH`.
- `rod-cli pdf [--name] [--out]`: Export the page as a PDF (`--out` = write to this path instead of `tmp/pdfs/`).

### Diagnostics
- `rod-cli stealth-check [url]`: Run per-signal stealth verdicts against the current page (or navigate to URL first). `--raw` prints a single PASS/FAIL line; `--json` prints the structured verdicts.
- `rod-cli cdp-traffic`: Print logged CDP protocol traffic from the proxy (requires --cdp-proxy). WARNING: output may contain sensitive CDP payload data (URLs, cookies, page content). `--cdp-proxy` is a global session-spawn flag, like those in Global Options.

### Plugins
- `rod-cli plugin load [path]`: Load a JavaScript plugin into the session. The plugin's lifecycle hooks (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`) start firing automatically — open a page first (e.g. `goto`) so the hooks and the `api` global bind to it.
- `rod-cli plugin list`: List the plugins loaded in the session.
- `rod-cli plugin run [name]`: Invoke a named function defined in the loaded plugin (e.g. an accessor like `getFindings`) and print its returned value. Use it to read back the state a plugin accumulated during browsing.

## Usage Example

```bash
# Navigate to the site
rod-cli goto https://example.com

# Verify running sessions
rod-cli sessions

# Interact
rod-cli fill --submit "#search-input" "go-rod automation"

# Extract state
rod-cli snapshot

# Teardown
rod-cli close
```

## Installation

Install this skill folder whole (`SKILL.md` together with `references/`) so the links under "Specific tasks" resolve.
Pi and OMP users can use the typed extensions instead: [Pi](https://github.com/agenthands/rod-cli/blob/main/extensions/pi/README.md), [OMP](https://github.com/agenthands/rod-cli/blob/main/extensions/omp/README.md).

## Specific tasks

Detailed guides on advanced features:

* **Inspecting element attributes**: [references/element-attributes.md](references/element-attributes.md)
* **Request mocking**: [references/request-mocking.md](references/request-mocking.md)
* **Running code**: [references/running-code.md](references/running-code.md)
* **Browser session management**: [references/session-management.md](references/session-management.md)
* **Storage state**: [references/storage-state.md](references/storage-state.md)
* **File uploads and drag-and-drop**: [references/file-uploads.md](references/file-uploads.md)
* **Tabs and windows management**: [references/tabs-and-windows.md](references/tabs-and-windows.md)
* **Writing plugins (lifecycle hooks, state API, CLI)**: [references/plugins.md](references/plugins.md)
