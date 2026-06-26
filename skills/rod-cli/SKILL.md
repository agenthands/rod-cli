---
name: rod-cli
description: "Token-efficient, native web automation CLI for LLMs using Go Rod. Use rod-cli whenever you need to browse the web, scrape content, interact with forms, click elements, take screenshots, evaluate JavaScript on a page, or automate any browser-based task. Prefer rod-cli over curl/wget for JavaScript-rendered pages and over heavy browser automation frameworks for token-efficient DOM snapshots."
---

# Go Rod Skill (`rod-cli`)

This skill defines the `rod-cli` binary usage for agentic web workflows. The tool is heavily optimized for token-efficiency, generating structured, accessibility-tree-based DOM snapshots rather than dumping massive raw HTML.

## Core Concepts

1. **Daemon Architecture**: `rod-cli` runs as a background persistent daemon to hold state.
   - The first command you execute implicitly boots a persistent browser in the background.
   - **Zombie Prevention**: The daemon polls your Agent process's PPID. If your agent crashes, the browser dies with it. A 15-minute idle timeout is also enforced.
   - You **MUST** run `rod-cli close` when you are finished to aggressively clean up resources.

2. **Named Sessions**:
   - For multi-target workflows (e.g., logging in as two distinct users simultaneously), use the global `--session` or `-s` flag to multiplex daemons.
   - `rod-cli -s=admin goto http://...`
   - `rod-cli -s=guest goto http://...`

## Commands

### Global Options
- `--json` / `--raw`: Suppress banners and output clean JSON or raw text.
- `-s, --session`: Specify a named session string.

### Setup
- `rod-cli install`: Install the Chromium browser required by rod-cli.

### Browser Lifecycle & Tabs
- `rod-cli sessions`: List all currently active background daemon sessions.
- `rod-cli close`: Terminate the background daemon and clean up the browser.
- `rod-cli open [url]` (or `goto`): Navigate to a URL.
- `rod-cli reload`, `go-back`, `go-forward`: Standard navigation.
- `rod-cli resize [width] [height]`: Resize the browser window.
- `rod-cli tab-list`: List all open tabs.
- `rod-cli tab-new [url]`: Create a new tab.
- `rod-cli tab-select [index]`: Select a browser tab.
- `rod-cli tab-close [index]`: Close a browser tab.

### Interaction
- `rod-cli click [selector]`: Click an element.
- `rod-cli dblclick [selector]`: Double-click an element.
- `rod-cli type [selector] [text]`: Type sequentially into an element.
- `rod-cli fill [selector] [text]`: Fill an entire input at once. Use `--submit` to press enter afterwards.
- `rod-cli select [selector] [values...]`: Select dropdown values.
- `rod-cli check [selector]`: Check a checkbox or radio button.
- `rod-cli uncheck [selector]`: Uncheck a checkbox or radio button.
- `rod-cli hover [selector]`: Hover over an element.
- `rod-cli upload [selector] [file]`: Upload a file to an element.
- `rod-cli drag [start-selector] [end-selector]`: Drag one element to another.
- `rod-cli drop [selector] --path [file-path]`: Drop a file onto an element.

### Advanced Input & Dialogs
- `rod-cli press [key]`: Simulate a raw keyboard key press (e.g., `Enter`, `Tab`, `Escape`).
- `rod-cli mousemove [x] [y]`: Move the mouse to absolute coordinates.
- `rod-cli mousedown [left|right|middle]`: Trigger mouse down.
- `rod-cli mouseup [left|right|middle]`: Trigger mouse up.
- `rod-cli mousewheel [dx] [dy]`: Scroll the mouse wheel.
- `rod-cli dialog-accept`: Automatically accept the next javascript alert/confirm.
- `rod-cli dialog-dismiss`: Automatically dismiss the next javascript alert/confirm.

### Storage & State Controls
- `rod-cli state-save [path]`: Save the browser state to a file.
- `rod-cli state-load [path]`: Load the browser state from a file.
- `rod-cli cookie-get`: Read all browser cookies.
- `rod-cli cookie-set [name] [value]`: Set a cookie.
- `rod-cli cookie-delete [name]`: Delete a specific cookie.
- `rod-cli cookie-clear`: Clear all browser cookies.
- `rod-cli localstorage-get [key]`: Retrieve a localStorage item (or omit key for all).
- `rod-cli localstorage-set [key] [value]`: Set a localStorage item.
- `rod-cli localstorage-delete [key]`: Delete a localStorage entry.
- `rod-cli localstorage-clear`: Clear localStorage.
- `rod-cli sessionstorage-get [key]`: Retrieve a sessionStorage item.
- `rod-cli sessionstorage-set [key] [value]`: Set a sessionStorage item.
- `rod-cli sessionstorage-delete [key]`: Delete a sessionStorage entry.
- `rod-cli sessionstorage-clear`: Clear sessionStorage.

### Observability & Debugging
- `rod-cli highlight [selector]`: Highlight an element with a persistent red border (useful for visual reasoning validation before clicking).
- `rod-cli highlight-clear`: Remove all injected highlights from the DOM.
- `rod-cli show --annotate`: Provide human-in-the-loop interactive bounding box feedback.
- `rod-cli console`: List browser console messages.

### Network Interception
- `rod-cli route [pattern] --body [body]`: Mock network requests via godoll/network interceptor.
- `rod-cli route-list`: List all active network interception routes.
- `rod-cli unroute [pattern]`: Remove a mock network route.
- `rod-cli requests`: List all intercepted network requests.
- `rod-cli request [index]`: Show details for a specific request.

### Evaluation & Export
- `rod-cli eval [script]`: Evaluate raw JavaScript in the browser context.
- `rod-cli snapshot`: Return the token-efficient Markdown representation of the DOM.
- `rod-cli screenshot --name [name]`: Capture a PNG of the viewport.
- `rod-cli pdf --name [name]`: Export the page as a PDF.

### Plugins
- `rod-cli plugin load [path]`: Load a JavaScript plugin into the session. The plugin's lifecycle hooks (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`) start firing automatically — open a page first (e.g. `goto`) so the hooks and the `api` global bind to it.
- `rod-cli plugin list`: List the plugins loaded in the session.
- `rod-cli plugin run [name]`: Invoke a named function defined in the loaded plugin (e.g. an accessor like `getFindings`) and print its returned value. This is how you read back the state a plugin accumulated while you browsed.

## Usage Example

```bash
# Navigate to the site
rod-cli --raw goto https://example.com

# Verify running sessions
rod-cli --raw sessions

# Interact
rod-cli --raw fill "#search-input" "go-rod automation" --submit
rod-cli --raw click "#submit-button"

# Extract state
rod-cli --raw snapshot

# Teardown
rod-cli --raw close
```

## Specific tasks

For more detailed guides on advanced features, refer to the following documents:

* **Inspecting element attributes**: [references/element-attributes.md](references/element-attributes.md)
* **Request mocking**: [references/request-mocking.md](references/request-mocking.md)
* **Running code**: [references/running-code.md](references/running-code.md)
* **Browser session management**: [references/session-management.md](references/session-management.md)
* **Storage state**: [references/storage-state.md](references/storage-state.md)
* **File uploads and drag-and-drop**: [references/file-uploads.md](references/file-uploads.md)
* **Tabs and windows management**: [references/tabs-and-windows.md](references/tabs-and-windows.md)
* **Writing plugins (lifecycle hooks, state API, CLI)**: [references/plugins.md](references/plugins.md)
