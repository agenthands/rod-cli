---
author: architect
responsible: architect
phase: 50
status: ready
parent_artifacts:
  - .planning/phases/50/CONTEXT.md
  - .planning/REQUIREMENTS.md
---

# Phase 50: Documentation & Discoverability — PLAN

## §1 — Extension README (DOCS-01, DOCS-02)

Write `extensions/pi/README.md` with these sections:

### Title + badge
```markdown
# @agenthands/rod-cli-pi

Pi extension for [rod-cli](https://github.com/agenthands/rod-cli) — native browser automation tools for the Pi coding agent.
```

### Prerequisites
```markdown
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
```

### Install
```markdown
## Install

```bash
pi install npm:@agenthands/rod-cli-pi
```

Or from local path during development:
```bash
pi install ./extensions/pi
```
```

### Tool catalog (DOCS-01)
Table of all 13 tools extracted from the actual code:

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `browse_goto` | Navigate to a URL | url (required), session |
| `browse_snapshot` | Accessibility-tree markdown snapshot | selector (optional), session |
| `browse_click` | Click element by CSS selector | selector (required), doubleClick, session |
| `browse_type` | Type text with humanized keystrokes | selector, text (required), session |
| `browse_eval` | Evaluate JavaScript on page | expression (required, ≤10KB), session |
| `browse_screenshot` | Capture screenshot | selector, session |
| `browse_wait` | Wait for selector or timeout | selector, timeout, session |
| `browse_tabs` | Manage browser tabs | action (list/new/close/select), url, index, session |
| `browse_navigate` | Page navigation | action (reload/back/forward), session |
| `browse_scroll` | Scroll page | direction (down/up), distance, session |
| `browse_cookies` | Manage cookies | action (get/set/delete/clear), name, value, session |
| `browse_storage` | localStorage + sessionStorage | action (get/set/delete/clear), storageType (local/session), key, value, session |
| `browse_fill_form` | Fill form fields instantly | selector, text (required), submit, session |

### Skill vs Extension (DOCS-02)
```markdown
## Skill vs Extension

rod-cli offers two integration paths for Pi. Both use the same `rod-cli` binary underneath.

| | [Agent Skill](../skills/rod-cli/SKILL.md) | Pi Extension (this package) |
|---|---|---|
| **Setup** | Copy SKILL.md to `~/.pi/agent/skills/` | `pi install npm:@agenthands/rod-cli-pi` |
| **Tool discovery** | LLM reads markdown, invokes bash | Tools appear in Pi's "Available tools" catalog |
| **Parameter validation** | None — LLM constructs shell commands | TypeBox schema validation before execution |
| **Prompt guidance** | Embedded in SKILL.md prose | `promptGuidelines` per tool, injected into system prompt |
| **Error handling** | LLM parses stderr | Structured errors thrown to LLM |
| **Lifecycle hooks** | None | Auto-verifies binary, cleans up daemon on quit |
| **Cross-tool** | Works with any Agent-Skills assistant | Pi-only |

**Recommendation:** Use the Extension if you use Pi. Use the Skill if you switch between assistants or want zero npm dependencies.
```

### Verify
```markdown
## Verify

Start a Pi session and ask:

> Browse to https://example.com and tell me the page title.

Pi should invoke `browse_goto` and `browse_snapshot` to complete the task.
```

## §2 — Update docs/onboarding/pi.md (DOCS-03)

Add after the "Advanced: TypeScript Extension" section:

```markdown
## First-Class Extension (v2.2+)

rod-cli now ships a first-class Pi TypeScript extension with typed tools, lifecycle hooks,
and prompt guidance. This is the recommended path for Pi users.

### Install

```bash
pi install npm:@agenthands/rod-cli-pi
```

### What you get

- **13 typed browser tools** — `browse_goto`, `browse_snapshot`, `browse_click`, etc.
- **Automatic daemon lifecycle** — browser starts on first use, cleans up on quit
- **Named sessions** — `session` parameter on every tool for multi-browser workflows
- **Token-efficient output** — markdown snapshots, not raw HTML

The [Skill-based path](#install) (SKILL.md) remains available as a zero-install fallback
for users who switch between assistants or prefer no npm dependencies.

See [extensions/pi/README.md](../../extensions/pi/README.md) for the full tool catalog.
```

## §3 — Update skills/rod-cli/SKILL.md (DOCS-03)

Find or add a Pi section after the existing per-assistant sections:

```markdown
### Pi (pi.dev)

**Skill path:** Copy `skills/rod-cli/SKILL.md` to `~/.pi/agent/skills/rod-cli/`.

**Extension path (recommended for Pi users):** Install the first-class Pi extension:
```bash
pi install npm:@agenthands/rod-cli-pi
```
The extension provides 13 typed browser tools with schema validation, lifecycle hooks,
and prompt guidance. See [extensions/pi/README.md](../../extensions/pi/README.md).
```

## §4 — Update top-level README.md (DOCS-04)

Find the docs index section and add:

```markdown
- **[Pi Extension](extensions/pi/README.md)** — First-class Pi TypeScript extension (13 typed browser tools)
```

## §5 — Build verification

1. Read all 4 files: verify no broken markdown links
2. Verify tool catalog in README matches actual tools in `extensions/pi/src/tools/`
3. `git diff --stat` confirms only those 4 files changed

## Execution order

§1 → §2 → §3 → §4 → §5

## Files changed

| File | § | Status |
|------|---|--------|
| `extensions/pi/README.md` | §1 | NEW |
| `docs/onboarding/pi.md` | §2 | MODIFIED |
| `skills/rod-cli/SKILL.md` | §3 | MODIFIED |
| `README.md` | §4 | MODIFIED |
