# Usage Guide

`rod-cli` is a powerful tool for automating the browser via CLI, designed explicitly for LLM agent integration.

## Quick Start

```bash
# install Chromium (only required once)
rod-cli install

# navigate to a page (automatically starts daemon)
rod-cli goto https://example.com

# interact with the page using refs from the snapshot
rod-cli click e15
rod-cli type e20 "search query"
rod-cli press Enter

# take a screenshot
rod-cli screenshot --name=demo.png

# close the browser
rod-cli close
```

## Raw Output

The global `--raw` option strips banners and structured JSON, returning only the raw result string. This is ideal for pipelining.

```bash
rod-cli --raw eval "document.title" > title.txt
TOKEN=$(rod-cli --raw cookie-get session_id)
```

For JSON output, pass `--json`:
```bash
rod-cli --json sessions
```

## Snapshots

After many commands, you may want to extract the token-efficient representation of the DOM.
```bash
rod-cli snapshot
```

## Browser Sessions

`rod-cli` runs as a persistent background daemon. To handle multiple isolated browsers (e.g., simulating two different users chatting), use named sessions via `-s`:

```bash
rod-cli -s=admin goto https://example.com/admin
rod-cli -s=user goto https://example.com/user

rod-cli -s=admin click e5
rod-cli -s=user type e2 "Hello!"

# View all running sessions
rod-cli sessions

# Close sessions to free up memory
rod-cli -s=admin close
rod-cli -s=user close
```

## Targeting Elements

Always use refs from the snapshot to interact with page elements.

```bash
# get snapshot with refs
rod-cli snapshot

# interact using a ref
rod-cli click e15
```

## Network Interception

`rod-cli` leverages `godoll/network` to mock requests dynamically.

```bash
# Block all analytics scripts
rod-cli route "*analytics.js*" --body="{}"

# Mock an API response
rod-cli route "*/api/user*" --body='{"id": 1, "name": "Admin"}'

# View active intercepts
rod-cli route-list

# Remove a specific intercept
rod-cli unroute "*analytics.js*"
```

For more advanced use cases like file uploads, drag-and-drop, and multiple tabs, please refer to the documents in `skills/rod-cli/references/`.
