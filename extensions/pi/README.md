# @agenthands/rod-cli-pi

Pi extension for [rod-cli](https://github.com/agenthands/rod-cli) -- native browser automation tools for the Pi coding agent.

## Prerequisites

1. **rod-cli binary** (Go 1.26.1+):
   ```bash
   go install github.com/agenthands/rod-cli@latest
   ```

2. **Chromium browser:**
   ```bash
   rod-cli install
   ```

3. **Pi coding agent:**
   ```bash
   npm install -g --ignore-scripts @earendil-works/pi-coding-agent
   ```

## Install

```bash
# Working path today (local checkout):
pi install ./extensions/pi

# Once published to npm (available once published):
pi install npm:@agenthands/rod-cli-pi
```

## Tool Catalog

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `browse_goto` | Navigate to a URL | url (required), session |
| `browse_snapshot` | Accessibility-tree markdown snapshot of the page or one subtree | selector (optional ref or CSS: subtree only), interactive (optional: controls only), session |
| `browse_click` | Click element by CSS selector or snapshot ref | selector (required: CSS selector or `[ref=...]` from browse_snapshot), doubleClick (optional), session |
| `browse_type` | Type text with humanized keystroke timing | selector (required: CSS selector or `[ref=...]` from browse_snapshot), text (required), session |
| `browse_eval` | Evaluate JavaScript on the page (max 10,000 characters) | expression (required), session |
| `browse_screenshot` | Capture a PNG and return the image itself | selector (optional ref or CSS), fullPage (optional), session |
| `browse_wait` | Wait for a selector to appear or a fixed duration | selector (optional), timeout (optional ms, default 30000; without selector, waits this long), session |
| `browse_tabs` | Manage browser tabs | action (list/new/close/select), url, index, session |
| `browse_navigate` | Page history navigation | action (reload/back/forward), session |
| `browse_scroll` | Scroll with real wheel input; reports what moved (`scrolled dy=800 y=1600/4600`, ` end` when nothing more) | direction (optional: up/down/left/right, default down; up/left/right need a distance), distance (optional: default one viewport down), selector (optional ref or CSS of an inner pane), session |
| `browse_cookies` | Manage browser cookies | action (get/set/delete/clear), name, value, session |
| `browse_storage` | Manage localStorage and sessionStorage | action (get/set/delete/clear), storageType (local/session), key, value, session |
| `browse_fill_form` | Fill form fields instantly (with optional submit) | selector (required: CSS selector or `[ref=...]` from browse_snapshot), text (required), submit (optional), session |
| `browse_press` | Press one key (Enter, Tab, Escape, arrows, a character) | key (required), session |
| `browse_select` | Choose options in a `<select>` by text, then value, then substring | selector (required: CSS selector or `[ref=...]`), values (required, at least one), session |
| `browse_hover` | Hover an element with a humanized pointer path | selector (required: CSS selector or `[ref=...]`), session |

All tools accept an optional `session` parameter for multi-browser/multi-identity workflows. Schemas are strict: an unknown or misspelled parameter is rejected, not ignored.

`browse_click`, `browse_type`, and `browse_fill_form` accept either a plain CSS selector (e.g. `#submit-btn`, works immediately, no snapshot needed) or a ref token copied verbatim from the most recent `browse_snapshot`'s `[ref=sNeM]` annotation (e.g. `s1e5`).

**Prefer native actions over `browse_eval` for interaction.** `browse_click`/`browse_type`/`browse_fill_form` dispatch real, trusted, humanized input (mouse tremor/speed, typing cadence); `browse_eval`-driven interaction (`el.click()`, setting `.value`) produces untrusted synthetic events with no humanization, defeating rod-cli's stealth design. Reserve `browse_eval` for reading page state/data.

**Daemon cleanup:** when Pi quits (not on reload, fork, or resume), the extension runs `rod-cli close`, ignoring errors. This closes the default session's browser and removes its profile and Chrome temp dir. A named session's daemon stays until `rod-cli -s <name> close` or 15 minutes idle.

## Integration Paths

rod-cli offers three integration paths. All use the same `rod-cli` binary underneath.

| | OMP extension (`@agenthands/rod-cli-omp`) | Pi extension (`@agenthands/rod-cli-pi`) | Agent Skill (`skills/rod-cli/SKILL.md`) |
|---|---|---|---|
| **Host** | OMP (oh-my-pi) | Pi (earendil) | any Agent-Skills assistant (Claude Code, Codex, opencode, ...); Gemini CLI reads it from `GEMINI.md` |
| **Setup** | `omp plugin install npm:@agenthands/rod-cli-omp` (once published) / `omp plugin link ./extensions/omp` (today) | `pi install npm:@agenthands/rod-cli-pi` (once published) / `pi install ./extensions/pi` (today) | copy the `skills/rod-cli/` directory to the assistant's skills dir |
| **Tools** | 16 typed `browse_*` (injected `pi.zod`) | 16 typed `browse_*` (TypeBox) | LLM shells out to `rod-cli` subcommands |
| **Validation / lifecycle** | schema + session_start/shutdown hooks | schema + session_start/shutdown hooks | none |
| **Best for** | OMP users (native path) | Pi users | multi-assistant / zero-npm |

**Recommendation:** Use the Pi Extension if you primarily use Pi. Use the Skill if you switch between assistants or prefer zero npm dependencies. OMP users should use the [OMP extension](../omp/README.md).

## Verify

Start a Pi session and ask:

> Browse to https://example.com and tell me the page title.

Pi should invoke `browse_goto` and `browse_snapshot` to complete the task.

## Release Process

See [`docs/extension-release.md`](../../docs/extension-release.md) for the release process, versioning convention, and how to cut a release.
