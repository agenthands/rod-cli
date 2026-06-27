---
author: architect
responsible: architect
phase: 49
phase_type: implementation
hard_bar: true
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (TOOLS-08..13)
  - .planning/ROADMAP.md (Phase 49 section)
  - .planning/phases/48/SUMMARY.md (core tools built)
---

# Phase 49: Extended Tools — CONTEXT

## Goal

Ship the 6 extended tools completing the browser automation surface — tab management,
page navigation, scrolling, cookie management, localStorage/sessionStorage, and
instant form filling.

## {P} What the phase can assume

- Phase 47 foundation: `execRodCli()`, `findRodCli()`, lifecycle hooks
- Phase 48 core tools: `registerAllCoreTools` pattern established, tool template known
- `SessionParam` exported from `../types`
- All daemon actions confirmed via SMTC against `daemon/daemon.go`

## {Q} What the phase must establish

1. 6 tool files under `extensions/pi/src/tools/` — each exports a `register*` function
2. `browse_tabs` manages tabs via `action: "list"|"new"|"close"|"select"` (TOOLS-08)
3. `browse_navigate` navigates history via `action: "reload"|"back"|"forward"` (TOOLS-09)
4. `browse_scroll` scrolls page via `mousewheel 0 <dy>` (TOOLS-10)
5. `browse_cookies` manages cookies via `action: "get"|"set"|"delete"|"clear"` (TOOLS-11)
6. `browse_storage` manages localStorage + sessionStorage via `storageType` (TOOLS-12)
7. `browse_fill_form` fills fields instantly via `rod-cli fill` with optional `--submit` (TOOLS-13)
8. `src/tools/index.ts` updated: `registerAllExtendedTools(pi)` adds to barrel
9. `src/index.ts` calls `registerAllExtendedTools` after `registerAllCoreTools`

## Tool-to-CLI mapping (SMTC-grounded)

| Tool | rod-cli command(s) | Daemon action(s) |
|------|--------------------|--------------------|
| `browse_tabs` | `tab-list`, `tab-new <url>`, `tab-close <idx>`, `tab-select <idx>` | tab-list, tab-new, tab-close, tab-select |
| `browse_navigate` | `reload`, `go-back`, `go-forward` | reload, go-back, go-forward |
| `browse_scroll` | `mousewheel 0 <dy>` | mousewheel |
| `browse_cookies` | `cookie-get`, `cookie-set <name> <val>`, `cookie-delete <name>`, `cookie-clear` | cookie-get, cookie-set, cookie-delete, cookie-clear |
| `browse_storage` | `localstorage-*` or `sessionstorage-*` | localstorage-get/set/delete/clear, sessionstorage-get/set/delete/clear |
| `browse_fill_form` | `fill <sel> <text> [--submit]` | fill |

## Invariants (cross-cutting, must preserve across phase)

- **I1-I5:** Same as Phase 48 (error discipline, StringEnum, promptGuidelines, session param, 1:1 CLI)
- **I6 (argv injection):** All tools insert `--` before user-controlled positional arguments
  (selectors, text, URLs, expression) to prevent flag smuggling. Example:
  `["click", "--", params.selector]` not `["click", params.selector]`.
  This applies retroactively to Phase 48 tools — fix those too.

## Files

| File | Purpose |
|------|---------|
| `extensions/pi/src/tools/tabs.ts` | browse_tabs |
| `extensions/pi/src/tools/navigate.ts` | browse_navigate |
| `extensions/pi/src/tools/scroll.ts` | browse_scroll |
| `extensions/pi/src/tools/cookies.ts` | browse_cookies |
| `extensions/pi/src/tools/storage.ts` | browse_storage |
| `extensions/pi/src/tools/fill_form.ts` | browse_fill_form |
| `extensions/pi/src/tools/index.ts` | MODIFIED: add `registerAllExtendedTools` |
| `extensions/pi/src/index.ts` | MODIFIED: call `registerAllExtendedTools` |
| `extensions/pi/src/tools/goto.ts` through `wait.ts` | MODIFIED: insert `--` before positional args (I6 fix) |

## Success criteria (falsifiable)

1. Each tool registered with correct `StringEnum` params, promptGuidelines, and execute
2. `browse_tabs` dispatches to correct tab-* command based on action
3. `browse_navigate` dispatches to reload/go-back/go-forward
4. `browse_scroll` computes dy from direction×distance, calls mousewheel
5. `browse_cookies` dispatches to correct cookie-* command
6. `browse_storage` dispatches to correct localstorage-*/sessionstorage-* based on storageType
7. `browse_fill_form` uses `fill` with optional `--submit`
8. `--` inserted before all user-controlled positional args (I6)
9. `tsc --noEmit` + `vitest` pass (all existing + new tests)
