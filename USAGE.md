# Usage Guide

`rod-cli` automates a browser from the command line, built for LLM agent integration.

## Quick Start

```bash
# install Chromium (only required once)
rod-cli install

# navigate to a page (automatically starts daemon)
rod-cli goto https://example.com

# print the page with a [ref=sNeM] on each element
rod-cli snapshot

# interact with the page using refs from the snapshot
rod-cli click s1e15
rod-cli type s1e20 "search query"
rod-cli press Enter

# take a screenshot (saved as tmp/screenshots/demo.png)
rod-cli screenshot --name=demo

# close the browser
rod-cli close
```

## Raw Output

Commands print plain text by default; the startup banner is off unless you pass `--banner`. The global `--raw` option gives the shortest form where a command has one: `--profile=list` prints names only, `stealth-check` prints one `PASS` or `FAIL` line, and `install` drops its progress line. Other commands print the same text with or without `--raw`, so it is safe to pipe.

```bash
rod-cli --raw eval "document.title" > title.txt
TOKEN=$(rod-cli cookie-get session_id | sed -n 's/^- session_id: \(.*\) (domain: .*)$/\1/p')
```

`cookie-get` prints a `Cookies:` line and then one `- <name>: <value> (domain: <domain>)` line per cookie, so the value is cut out with `sed`.

For JSON output, pass `--json`; plain-text results are wrapped as `{"result": "..."}`, results that are already JSON (such as `stealth-check`) print as they are, and `sessions` prints `{"sessions": [...]}`:
```bash
rod-cli --json sessions
```

Global options such as `--raw`, `--json` and `-s` work before or after the command name. A command's own options, such as `route --body`, go before its arguments.

## Snapshots

`snapshot` prints a token-efficient view of the DOM with a `[ref=sNeM]` on each element. `-i` keeps only interactive elements and their ancestors; a ref or CSS selector argument limits it to that subtree.
```bash
rod-cli snapshot
```

## Browser Sessions

Each session runs its own background daemon that keeps its browser alive between commands. To handle multiple isolated browsers (e.g., simulating two different users chatting), use named sessions via `-s` (default: `default`). Session names use letters, digits, `.`, `_` or `-` (1 to 64 characters, not `.` or `..`); any other name fails with `invalid session name`:

```bash
rod-cli -s=admin goto https://example.com/admin
rod-cli -s=user goto https://example.com/user

# each session numbers its own refs; snapshot before using one
rod-cli -s=admin snapshot
rod-cli -s=user snapshot

rod-cli -s=admin click s1e5
rod-cli -s=user type s1e2 "Hello!"

# View all running sessions
rod-cli sessions

# Close sessions to free up memory
rod-cli -s=admin close
rod-cli -s=user close
```

Unless `--cdp-endpoint` attaches it to an existing browser, each session's Chrome gets a fresh profile under `~/.cache/rod-cli/profiles` (the OS user cache dir; the config's `browserTempDir` overrides it) and its own temp dir, `$TMPDIR/chrome-XXXX` (`/tmp/chrome-XXXX` when `TMPDIR` is longer than 40 characters). Both are removed when the session closes, after 15 minutes idle, or when the daemon gets SIGTERM, SIGINT or SIGHUP. A daemon killed with `kill -9` leaves them behind; the next daemon start removes the port files of dead daemons, and the next session that launches Chrome under the default profile root removes their profiles and temp dirs.

Daemon files live in the OS temp dir: `rod-cli-<session>.port` per session, and the shared log `rod-cli-daemon.log`, which a daemon start truncates once it exceeds 1 MiB.

## Targeting Elements

Target an element by the ref from the latest snapshot (`[ref=s1e15]` → `s1e15`) or by a CSS selector.

```bash
# get snapshot with refs
rod-cli snapshot

# interact using a ref
rod-cli click s1e15
```

## Network Interception

`rod-cli` leverages `godoll/network` to mock requests dynamically.

```bash
# Block all analytics scripts
rod-cli route --body="{}" "*analytics.js*"

# Mock an API response
rod-cli route --body='{"id": 1, "name": "Admin"}' "*/api/user*"

# View active intercepts
rod-cli route-list

# Remove a specific intercept
rod-cli unroute "*analytics.js*"
```

For file uploads, drag-and-drop, multiple tabs and other advanced cases, see the documents in `skills/rod-cli/references/`.
