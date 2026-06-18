# Description:
rod-cli is a lightweight, zero-dependency command-line interface (CLI) that gives AI assistants native web browsing, scraping, and interaction capabilities. Built on top of Go-rod, it replaces bulky, bug-prone Node.js setups with a fast, compiled Go binary.

Inspired directly by the architecture of microsoft/playwright-cli, this tool operates as a "Skill" rather than a traditional stateful MCP server. It is designed explicitly for LLMs (Claude, Gemini, and custom autonomous agents), communicating via standard input/output (stdio). It features aggressive context-window optimization—stripping out DOM noise and converting web pages to LLM-friendly Markdown—so your AI can read the web without burning through token limits or hallucinating on messy HTML.

## Key Benefits:
    - Token-Efficient by Design: Like playwright-cli, it avoids loading massive accessibility trees or verbose JSON tool schemas into the model context. It relies on concise, purpose-built commands.
    - Zero Dependency Hell: It’s a single Go binary. No Node.js, no node_modules, no Python environments. Just download and run.
    - Agent-Ready via Stdio: Works seamlessly as a background process using the standard MCP stdio transport, making it a perfect drop-in skill for coding agents.
    - Rock-Solid Stability: Leverages Go-rod's auto-waiting and crash resilience. No more flaky timeouts or zombie browser processes.   

# SKILL.MD

---
name: rod-cli
description: Automate browser interactions, test web pages
allowed-tools: Bash(rod-cli:*) 
---

# Browser Automation with rod-cli

## Quick start

```bash
# open new browser
rod-cli open
# navigate to a page
rod-cli goto https://playwright.dev
# interact with the page using refs from the snapshot
rod-cli click e15
rod-cli type "page.click"
rod-cli press Enter
# take a screenshot (rarely used, as snapshot is more common)
rod-cli screenshot
# close the browser
rod-cli close
```

## Commands

### Core

```bash
rod-cli open
# open and navigate right away
rod-cli open https://example.com/
rod-cli goto https://google.com
rod-cli type "search query"
rod-cli click e3
rod-cli dblclick e7
# --submit presses Enter after filling the element
rod-cli fill e5 "user@example.com"  --submit
rod-cli drag e2 e8
# drop files or data onto an element (from outside the page)
rod-cli drop e4 --path=./image.png
rod-cli drop e4 --data="text/plain=hello world"
rod-cli hover e4
rod-cli select e9 "option-value"
rod-cli upload ./document.pdf
rod-cli check e12
rod-cli uncheck e12
rod-cli snapshot
rod-cli eval "document.title"
rod-cli eval "el => el.textContent" e5
# get element id, class, or any attribute not visible in the snapshot
rod-cli eval "el => el.id" e5
rod-cli eval "el => el.getAttribute('data-testid')" e5
rod-cli dialog-accept
rod-cli dialog-accept "confirmation text"
rod-cli dialog-dismiss
rod-cli resize 1920 1080
rod-cli close
```

### Navigation

```bash
rod-cli go-back
rod-cli go-forward
rod-cli reload
```

### Keyboard

```bash
rod-cli press Enter
rod-cli press ArrowDown
rod-cli keydown Shift
rod-cli keyup Shift
```

### Mouse

```bash
rod-cli mousemove 150 300
rod-cli mousedown
rod-cli mousedown right
rod-cli mouseup
rod-cli mouseup right
rod-cli mousewheel 0 100
```

### Save as

```bash
rod-cli screenshot
rod-cli screenshot e5
rod-cli screenshot --filename=page.png
rod-cli pdf --filename=page.pdf
```

### Tabs

```bash
rod-cli tab-list
rod-cli tab-new
rod-cli tab-new https://example.com/page
rod-cli tab-close
rod-cli tab-close 2
rod-cli tab-select 0
```

### Storage

```bash
rod-cli state-save
rod-cli state-save auth.json
rod-cli state-load auth.json

# Cookies
rod-cli cookie-list
rod-cli cookie-list --domain=example.com
rod-cli cookie-get session_id
rod-cli cookie-set session_id abc123
rod-cli cookie-set session_id abc123 --domain=example.com --httpOnly --secure
rod-cli cookie-delete session_id
rod-cli cookie-clear

# LocalStorage
rod-cli localstorage-list
rod-cli localstorage-get theme
rod-cli localstorage-set theme dark
rod-cli localstorage-delete theme
rod-cli localstorage-clear

# SessionStorage
rod-cli sessionstorage-list
rod-cli sessionstorage-get step
rod-cli sessionstorage-set step 3
rod-cli sessionstorage-delete step
rod-cli sessionstorage-clear
```

### Network

```bash
rod-cli route "**/*.jpg" --status=404
rod-cli route "https://api.example.com/**" --body='{"mock": true}'
rod-cli route-list
rod-cli unroute "**/*.jpg"
rod-cli unroute
```

### DevTools

```bash
rod-cli console
rod-cli console warning
rod-cli requests
rod-cli request 5
rod-cli run-code "async page => await page.context().grantPermissions(['geolocation'])"
rod-cli run-code --filename=script.js
rod-cli tracing-start
rod-cli tracing-stop
rod-cli video-start video.webm
rod-cli video-chapter "Chapter Title" --description="Details" --duration=2000
rod-cli video-stop

# annotate each subsequent action (click, type, ...) with a callout naming the action and highlighting the target
rod-cli video-show-actions --duration=600 --position=top-right
rod-cli video-hide-actions

# launch the dashboard for UI review / design feedback — user annotates the page, you receive the annotated screenshot, snapshot, and notes
rod-cli show --annotate

# generate a locator for an element from its ref or selector
rod-cli generate-locator e5 --raw

# show a persistent highlight overlay for an element, optionally with a custom style
rod-cli highlight e5
rod-cli highlight e5 --style="outline: 3px dashed red"
# hide a single element highlight, or all page highlights when no target is given
rod-cli highlight e5 --hide
rod-cli highlight --hide
```

## Raw output

The global `--raw` option strips page status, generated code, and snapshot sections from the output, returning only the result value. Use it to pipe command output into other tools. Commands that don't produce output return nothing.

```bash
rod-cli --raw eval "JSON.stringify(performance.timing)" | jq '.loadEventEnd - .navigationStart'
rod-cli --raw eval "JSON.stringify([...document.querySelectorAll('a')].map(a => a.href))" > links.json
rod-cli --raw snapshot > before.yml
rod-cli click e5
rod-cli --raw snapshot > after.yml
diff before.yml after.yml
TOKEN=$(rod-cli --raw cookie-get session_id)
rod-cli --raw localstorage-get theme
```

For structured output wrapping every reply as JSON, pass --json
```bash
rod-cli list --json
```

## Open parameters
```bash

# Use persistent profile (by default profile is in-memory)
rod-cli open --persistent
# Use persistent profile with custom directory
rod-cli open --profile=/path/to/profile

# Connect to browser via Playwright Extension
rod-cli attach --extension=chrome

# Connect to a running Chrome or Edge by channel name
rod-cli attach --cdp=chrome
rod-cli attach --cdp=msedge

# Connect to a running browser via CDP endpoint
rod-cli attach --cdp=http://localhost:9222

# Start with config file
rod-cli open --config=my-config.json

# Close the browser
rod-cli close
# Detach from an attached browser (leaves the external browser running)
rod-cli -s=msedge detach
# Delete user data for the default session
rod-cli delete-data
```

## URLs with `&` on Windows

On Windows, `cmd.exe` and PowerShell treat `&` as a command separator, so URLs with multiple query parameters get truncated before `rod-cli` runs. Escape `&` with `^&` in `cmd.exe`, or use `--%` in PowerShell:

```batch
rod-cli goto "https://example.com/?a=1^&b=2"
```

```powershell
rod-cli --% goto "https://example.com/?a=1&b=2"
```

## Snapshots

After each command, rod-cli provides a snapshot of the current browser state.

```bash
> rod-cli goto https://example.com
### Page
- Page URL: https://example.com/
- Page Title: Example Domain
### Snapshot
[Snapshot](.rod-cli/page-2026-02-14T19-22-42-679Z.yml)
```

You can also take a snapshot on demand using `rod-cli snapshot` command. All the options below can be combined as needed.

```bash
# default - save to a file with timestamp-based name
rod-cli snapshot

# save to file, use when snapshot is a part of the workflow result
rod-cli snapshot --filename=after-click.yaml

# snapshot an element instead of the whole page
rod-cli snapshot "#main"

# limit snapshot depth for efficiency, take a partial snapshot afterwards
rod-cli snapshot --depth=4
rod-cli snapshot e34

# include each element's bounding box as [box=x,y,width,height]
rod-cli snapshot --boxes
```

## Targeting elements

By default, use refs from the snapshot to interact with page elements.

```bash
# get snapshot with refs
rod-cli snapshot

# interact using a ref
rod-cli click e15
```

You can also use css selectors or Playwright locators.

```bash
# css selector
rod-cli click "#main > button.submit"

# role locator
rod-cli click "getByRole('button', { name: 'Submit' })"

# test id
rod-cli click "getByTestId('submit-button')"
```

## Browser Sessions

```bash
# create new browser session named "mysession" with persistent profile
rod-cli -s=mysession open example.com --persistent
# same with manually specified profile directory (use when requested explicitly)
rod-cli -s=mysession open example.com --profile=/path/to/profile
rod-cli -s=mysession click e6
rod-cli -s=mysession close  # stop a named browser
rod-cli -s=mysession delete-data  # delete user data for persistent session

rod-cli list
# Close all browsers
rod-cli close-all
# Forcefully kill all browser processes
rod-cli kill-all
```

## Installation

If global `rod-cli` command is not available, try a local version via `npx rod-cli`:

```bash
npx --no-install rod-cli --version
```

When local version is available, use `npx rod-cli` in all commands. Otherwise, install `rod-cli` as a global command:

```bash
npm install -g @playwright/cli@latest
```

## Example: Form submission

```bash
rod-cli open https://example.com/form
rod-cli snapshot

rod-cli fill e1 "user@example.com"
rod-cli fill e2 "password123"
rod-cli click e3
rod-cli snapshot
rod-cli close
```

## Example: Multi-tab workflow

```bash
rod-cli open https://example.com
rod-cli tab-new https://example.com/other
rod-cli tab-list
rod-cli tab-select 0
rod-cli snapshot
rod-cli close
```

## Example: Debugging with DevTools

```bash
rod-cli open https://example.com
rod-cli click e4
rod-cli fill e7 "test"
rod-cli console
rod-cli requests
rod-cli close
```

```bash
rod-cli open https://example.com
rod-cli tracing-start
rod-cli click e4
rod-cli fill e7 "test"
rod-cli tracing-stop
rod-cli close
```

## Example: Interactive session

Ask the user for UI review or design feedback. The user draws boxes on the live page and types comments; you receive the annotated screenshot, the snapshot of the marked region, and the user's notes. Use this whenever the user asks for "UI review", "design feedback", or to "ask the user what they think / want / mean":

```bash
rod-cli open https://example.com
rod-cli show --annotate
```

