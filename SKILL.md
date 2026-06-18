---
name: rod-cli
description: "Token-efficient, native web automation CLI for LLMs using Go Rod"
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

### Browser Lifecycle
- `rod-cli sessions`: List all currently active background daemon sessions.
- `rod-cli close`: Terminate the background daemon and clean up the browser.
- `rod-cli open [url]` (or `goto`): Navigate to a URL.
- `rod-cli reload`, `go-back`, `go-forward`: Standard navigation.

### Interaction
- `rod-cli click [selector]`: Click an element.
- `rod-cli dblclick [selector]`: Double-click an element.
- `rod-cli type [selector] [text]`: Type sequentially into an element.
- `rod-cli fill [selector] [text]`: Fill an entire input at once. Use `--submit` to press enter afterwards.
- `rod-cli select [selector] [values...]`: Select dropdown values.
- `rod-cli hover [selector]`: Hover over an element.

### Advanced Input & Dialogs
- `rod-cli press [key]`: Simulate a raw keyboard key press (e.g., `Enter`, `Tab`, `Escape`).
- `rod-cli mousemove [x] [y]`: Move the mouse to absolute coordinates.
- `rod-cli mousedown [left|right|middle]`: Trigger mouse down.
- `rod-cli mouseup [left|right|middle]`: Trigger mouse up.
- `rod-cli dialog-accept`: Automatically accept the next javascript alert/confirm.
- `rod-cli dialog-dismiss`: Automatically dismiss the next javascript alert/confirm.

### Storage Controls
- `rod-cli cookie-get`: Read all browser cookies.
- `rod-cli cookie-clear`: Clear all browser cookies.
- `rod-cli localstorage-get [key]`: Retrieve a localStorage item (or omit key for all).
- `rod-cli localstorage-set [key] [value]`: Set a localStorage item.
- `rod-cli localstorage-clear`: Clear localStorage.
- `rod-cli sessionstorage-get [key]`: Retrieve a sessionStorage item.
- `rod-cli sessionstorage-set [key] [value]`: Set a sessionStorage item.
- `rod-cli sessionstorage-clear`: Clear sessionStorage.

### Observability & Debugging
- `rod-cli highlight [selector]`: Highlight an element with a persistent red border (useful for visual reasoning validation before clicking).
- `rod-cli highlight-clear`: Remove all injected highlights from the DOM.
- `rod-cli video-start --name [name]`: (Stubbed) Begin recording viewport.
- `rod-cli video-stop`: (Stubbed) Stop recording video.
- `rod-cli show --annotate`: Provide human-in-the-loop interactive bounding box feedback.

### Evaluation & Export
- `rod-cli eval [script]`: Evaluate raw JavaScript in the browser context.
- `rod-cli snapshot`: Return the token-efficient Markdown representation of the DOM.
- `rod-cli screenshot --name [name]`: Capture a PNG of the viewport.
- `rod-cli pdf --name [name]`: Export the page as a PDF.

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
