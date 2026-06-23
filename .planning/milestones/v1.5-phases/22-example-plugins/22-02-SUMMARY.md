---
phase: 22-example-plugins
plan: 02
subsystem: plugins
status: complete
tags: [plugin, examples, recipes, javascript]
requires:
  - internal/plugin/lifecycle.go (hook names + payload shapes)
  - internal/plugin/engine.go (LoadScript/RunFunc execution path)
provides:
  - plugins/examples/recipes/on_request.js (onRequest recipe + getRequestLog)
  - plugins/examples/recipes/on_response.js (onResponse recipe + getResponseLog)
  - plugins/examples/recipes/on_load.js (onLoad recipe + getLoadLog)
  - plugins/examples/recipes/on_dom_node_inserted.js (onDOMNodeInserted recipe + getInsertedNodes)
  - plugins/examples/starter.js (copyable four-hook scaffold + getResults)
affects:
  - docs/plugins (recipes referenced by lifecycle-hooks/cli docs in later plans)
tech-stack:
  added: []
  patterns:
    - "Recipe shape: module-level var array + one lowercase hook + JSON.stringify getter"
    - "Sandbox-clean JS: no console/fs/require/import (goja exposes none)"
    - "api global guarded with typeof + try/catch for parse-outside-session safety"
key-files:
  created:
    - plugins/examples/recipes/on_request.js
    - plugins/examples/recipes/on_response.js
    - plugins/examples/recipes/on_load.js
    - plugins/examples/recipes/on_dom_node_inserted.js
    - plugins/examples/starter.js
  modified: []
decisions:
  - "Task 3 (load-test) added no code commit: all five scripts loaded clean on first attempt, so no script fixes were needed; verification proven via the engine's LoadScript+RunFunc path."
metrics:
  duration: ~6m
  completed: 2026-06-23
  tasks: 3
  files: 5
---

# Phase 22 Plan 02: Example Plugin Recipes and Starter Summary

Shipped the four per-hook recipe plugins (PEX-02) and the copyable starter template (PEX-03) under `plugins/examples/`, each mirroring the existing `xss_scanner.js` shape (module-level array + one lowercase lifecycle hook + a `JSON.stringify` getter) and verified to load and execute cleanly through the goja engine.

## What Was Built

- **Four recipes** under `plugins/examples/recipes/`, one lifecycle hook each:
  - `on_request.js` — `onRequest` pushes `{url, method}`; `getRequestLog()` returns JSON.
  - `on_response.js` — `onResponse` pushes `{url, status}`; `getResponseLog()` returns JSON.
  - `on_load.js` — `onLoad` reads `api.GetSnapshot()` under a `typeof api !== "undefined"` + try/catch guard, pushes `{snapshotLength}`; `getLoadLog()` returns JSON.
  - `on_dom_node_inserted.js` — `onDOMNodeInserted` pushes `event.Node.NodeName`; `getInsertedNodes()` returns JSON. Based on the lifecycle-hooks.md snippet (no `xss_scanner` analog exists).
- **Starter** `plugins/examples/starter.js` — usage comment block, `var results = []`, no-op `// TODO` stubs for all four hooks, and `getResults()` accessor. Loads unchanged.

All scripts use lowercase hook names (the engine looks up lowercase) and are sandbox-clean — no `console`, `fs`, `require`, or `import`.

## Verification

- `go build -o /tmp/rod-cli-22 .` exits 0.
- Forbidden-global scan (`grep -lE 'console\.|require\(|^import '`) returns no files across all five scripts.
- Load-test through the real engine: a throwaway in-module program called `PluginEngine.Init()` → `LoadScript(path)` → `RunFunc(getter)` for each of the five scripts. All five loaded with no goja execute error and every getter returned clean JSON (`[]` on an empty session). The throwaway program was removed after the run (no stray files committed).

```
OK on_request.js          -> getRequestLog()   = []
OK on_response.js         -> getResponseLog()  = []
OK on_load.js             -> getLoadLog()       = []
OK on_dom_node_inserted.js-> getInsertedNodes() = []
OK starter.js             -> getResults()       = []
```

## Deviations from Plan

None - plan executed exactly as written. Task 3 is verification-only and required no script fixes (all five scripts loaded clean on the first attempt), so it produced no additional code commit.

## Requirements Satisfied

- **PEX-02**: one runnable recipe per lifecycle hook, each loadable and readable via `plugin run <getter>`.
- **PEX-03**: a copyable starter stubbing all four hooks + a results accessor that loads unchanged.

## Known Stubs

`plugins/examples/starter.js` intentionally ships empty `/* TODO */` hook bodies — this is the scaffold's purpose (PEX-03), not an unfinished stub. It loads and runs cleanly; users copy it and fill in the hooks they need.

## Self-Check: PASSED

- FOUND: plugins/examples/recipes/on_request.js
- FOUND: plugins/examples/recipes/on_response.js
- FOUND: plugins/examples/recipes/on_load.js
- FOUND: plugins/examples/recipes/on_dom_node_inserted.js
- FOUND: plugins/examples/starter.js
- FOUND commit 041f5fd (recipes)
- FOUND commit d796aef (starter)
