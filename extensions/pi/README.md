# @agenthands/rod-cli-pi

Pi extension for [rod-cli](https://github.com/agenthands/rod-cli) -- native browser automation tools for the Pi coding agent.

## Prerequisites

1. **rod-cli binary** (Go 1.23+):
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
pi install npm:@agenthands/rod-cli-pi
```

Or from a local checkout during development:

```bash
pi install ./extensions/pi
```

## Tool Catalog

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `browse_goto` | Navigate to a URL | url (required), session |
| `browse_snapshot` | Accessibility-tree markdown snapshot | selector (optional), session |
| `browse_click` | Click element by CSS selector | selector (required), doubleClick (optional), session |
| `browse_type` | Type text with humanized keystroke timing | selector (required), text (required), session |
| `browse_eval` | Evaluate JavaScript on the page (max 10KB) | expression (required), session |
| `browse_screenshot` | Capture a screenshot of the page or element | selector (optional), session |
| `browse_wait` | Wait for a selector to appear or a fixed duration | selector (optional), timeout (optional), session |
| `browse_tabs` | Manage browser tabs | action (list/new/close/select), url, index, session |
| `browse_navigate` | Page history navigation | action (reload/back/forward), session |
| `browse_scroll` | Scroll the page viewport | direction (down/up), distance, selector, session |
| `browse_cookies` | Manage browser cookies | action (get/set/delete/clear), name, value, session |
| `browse_storage` | Manage localStorage and sessionStorage | action (get/set/delete/clear), storageType (local/session), key, value, session |
| `browse_fill_form` | Fill form fields instantly (with optional submit) | selector (required), text (required), submit (optional), session |

All tools accept an optional `session` parameter for multi-browser/multi-identity workflows.

## Skill vs Extension

rod-cli offers two integration paths for Pi. Both use the same `rod-cli` binary underneath.

| | Agent Skill | Pi Extension (this package) |
|---|---|---|
| **Setup** | Copy `skills/rod-cli/SKILL.md` to `~/.pi/agent/skills/` | `pi install npm:@agenthands/rod-cli-pi` |
| **Tool discovery** | LLM reads markdown, invokes shell commands | Tools appear in Pi's "Available tools" catalog |
| **Parameter validation** | None -- LLM constructs shell commands | TypeBox schema validation before execution |
| **Prompt guidance** | Embedded in SKILL.md prose | `promptGuidelines` per tool, injected into system prompt |
| **Error handling** | LLM parses stderr | Structured errors thrown to the LLM |
| **Lifecycle hooks** | None | Auto-verifies binary on session start, cleans up daemon on quit |
| **Compatibility** | Works with any Agent-Skills assistant (Claude Code, Codex, Gemini, etc.) | Pi-only |

**Recommendation:** Use the Extension if you primarily use Pi. Use the Skill if you switch between assistants or prefer zero npm dependencies.

## Verify

Start a Pi session and ask:

> Browse to https://example.com and tell me the page title.

Pi should invoke `browse_goto` and `browse_snapshot` to complete the task.
