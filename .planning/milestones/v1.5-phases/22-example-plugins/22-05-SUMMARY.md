---
phase: 22-example-plugins
plan: 05
subsystem: docs
tags: [plugin, docs, examples, recipes, starter]
requires:
  - "Plan 22-01 engine fix (PluginRun -> RunFunc) shipped — plugin run <getter> is functional"
  - "Plan 22-02 per-hook recipe scripts under plugins/examples/recipes/"
  - "Plan 22-02/22-03 starter template plugins/examples/starter.js"
  - "Plan 22-04 docs/plugins/examples/xss-scanner.md (sibling style + cross-link target)"
provides:
  - "docs/plugins/examples/recipes.md — one section per lifecycle hook (load -> drive -> plugin run <getter>)"
  - "docs/plugins/examples/starter.md — copy -> load unchanged -> plugin run getResults workflow"
affects:
  - "docs/plugins/examples/ doc set (now: xss-scanner, recipes, starter)"
tech-stack:
  added: []
  patterns:
    - "Example docs live one level deeper (docs/plugins/examples/) and cross-link refs with ../"
    - "Reference Phase 21 pages for payload shapes / api / commands rather than duplicating them"
key-files:
  created:
    - docs/plugins/examples/recipes.md
    - docs/plugins/examples/starter.md
  modified: []
decisions:
  - "Single recipes.md with one ### section per hook (not per-file docs) — discretion granted by CONTEXT line 49"
  - "Getter/accessor names quoted verbatim from the scripts: getRequestLog, getResponseLog, getLoadLog, getInsertedNodes, getResults"
metrics:
  duration_min: 1
  completed: 2026-06-23
  tasks: 2
  files: 2
status: complete
---

# Phase 22 Plan 05: Example Plugins (Recipes + Starter Docs) Summary

Documented the per-hook recipe plugins (PEX-02) and the copyable starter template (PEX-03) under `docs/plugins/examples/`, matching the style of the sibling `xss-scanner.md` and cross-linking the Phase 21 reference pages instead of duplicating them.

## What Was Built

### Task 1 — `docs/plugins/examples/recipes.md` (commit `af3517b`)

One `###` section per lifecycle hook (`onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted`). Each section names the recipe script, says in one line what it collects, and shows the `plugin load` → drive-the-browser → `plugin run <getter>` flow with the JSON each getter returns. Getters quoted verbatim from the scripts: `getRequestLog`, `getResponseLog`, `getLoadLog`, `getInsertedNodes`. The `onLoad` section links `../state-api.md` for `api.GetSnapshot()`; all sections link `../lifecycle-hooks.md` for payload shapes. Includes `## See Also` (`../lifecycle-hooks.md`, `../state-api.md`, `../cli-reference.md`, `./xss-scanner.md`) and a `## Source` footer pointing at `../../plugins/examples/recipes/`.

### Task 2 — `docs/plugins/examples/starter.md` (commit `ab0a763`)

Documents `plugins/examples/starter.js` as a copy-and-fill scaffold: a `bash` block copying the starter, loading it unchanged with no errors, and `plugin run getResults` returning an empty `[]` until `results` is populated. Explains the four optional hook stubs and the `getResults` accessor (name verbatim from the script), with a "next steps" note linking `../lifecycle-hooks.md`, `../state-api.md`, and `./recipes.md`. Includes `## See Also` (with `../` cross-links plus `./xss-scanner.md`) and a `## Source` footer at `../../plugins/examples/starter.js`.

## Deviations from Plan

None — plan executed exactly as written.

## Verification

- Task 1 automated check: `recipes.md` exists; contains `getRequestLog`, `getResponseLog`, `getLoadLog`, `getInsertedNodes`; cross-links `../lifecycle-hooks.md`, `../state-api.md`, `../cli-reference.md`. PASS.
- Task 2 automated check: `starter.md` exists; contains `starter.js`, `plugin run getResults`; cross-links `../cli-reference.md`, `../lifecycle-hooks.md`, `../state-api.md`. PASS.
- Both docs cross-link the Phase 21 reference pages with the correct `../` depth and do not duplicate their content.

## Requirements Satisfied

- **PEX-02** — each per-hook recipe is documented with run instructions so a reader can see the hook fire.
- **PEX-03** — the starter is documented so a reader can copy it, load it unchanged, and confirm it runs (`plugin run getResults` → `[]`).

## Self-Check: PASSED

- FOUND: docs/plugins/examples/recipes.md
- FOUND: docs/plugins/examples/starter.md
- FOUND commit: af3517b (recipes.md)
- FOUND commit: ab0a763 (starter.md)
