---
phase: 21-reference-documentation
plan: 04
subsystem: docs
tags: [documentation, plugins, state-api, reference]
requires:
  - internal/plugin/api.go (GetSnapshot/GetCookies/GetLocalStorage)
  - docs/plugins/lifecycle-hooks.md (cross-link target)
provides:
  - docs/plugins/state-api.md (authoritative state/context API reference)
affects:
  - docs/plugins/
tech-stack:
  added: []
  patterns:
    - "Reference page template: overview -> per-accessor H2 -> return shape -> worked snippet -> cross-links -> Source footer"
key-files:
  created:
    - docs/plugins/state-api.md
  modified: []
decisions:
  - "Documented exactly the three accessors present in internal/plugin/api.go (GetSnapshot, GetCookies, GetLocalStorage) — no invented surfaces"
  - "GetLocalStorage return shape documented as a key/value object (map[string]interface{} via gson Val()), matching what Plan 01 shipped"
  - "Cookie fields kept to commonly-used (Name/Value/Domain/Path) with a link to proto.NetworkCookie rather than dumping the full struct (token-efficient depth)"
  - "Network context section defers to onRequest/onResponse hook payloads with a cross-link to lifecycle-hooks.md — network is not an api method"
metrics:
  duration: ~3m
  completed: 2026-06-22
requirements: [PDOC-03]
status: complete
---

# Phase 21 Plan 04: State / Context API Reference Summary

Created `docs/plugins/state-api.md`, the authoritative reference for the plugin `api` global's state/context accessors, grounded in `internal/plugin/api.go`. It documents `GetSnapshot()` (full-page HTML string), `GetCookies()` (CDP cookie array), and the Plan-01-added `GetLocalStorage()` (localStorage key/value object) with return shapes and worked snippets, notes that `api` is bound only after `plugin load` (BindLifecycle), and explains network context is reached via lifecycle hook payloads (cross-linking lifecycle-hooks.md).

## Tasks Completed

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Write docs/plugins/state-api.md | 54ff661 | docs/plugins/state-api.md |

## What Was Built

A single Markdown reference page matching the style of the sibling pages (`lifecycle-hooks.md`, `cli-reference.md`):

- **`api.GetSnapshot()`** — documented as a full-page HTML string backed by `page.HTML()`, with a token-cost caveat and a worked `onLoad` snippet adapted from `xss_scanner.js`.
- **`api.GetCookies()`** — documented as a CDP cookie array; commonly-used fields (Name/Value/Domain/Path) in a table with a link to `proto.NetworkCookie` for the full shape.
- **`api.GetLocalStorage()`** — documented as a key/value object (the shape Plan 01 shipped: a JS object iterated from `window.localStorage`, returned via gson `Val()` as `map[string]interface{}`), with a worked snippet reading a key.
- **When `api` is available** — explicit note that `api` is bound only after `BindLifecycle`/`plugin load`, mirroring the `typeof api !== "undefined"` guard from `xss_scanner.js`.
- **Network context** — explicit section stating network data is not an `api` method but arrives via `onRequest`/`onResponse` payloads, cross-linking `lifecycle-hooks.md`.
- **See Also** cross-links to `lifecycle-hooks.md` and `cli-reference.md`; **Source** footer citing `internal/plugin/api.go` and `internal/plugin/lifecycle.go`.

## Verification

Automated check passed:
- `docs/plugins/state-api.md` exists.
- Contains `GetSnapshot`, `GetCookies`, `GetLocalStorage`, and `lifecycle-hooks.md`.
- `internal/plugin/api.go` contains `func (a *PluginAPI) GetLocalStorage` (Plan 01 dependency satisfied — GUARD passed).

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED

- FOUND: docs/plugins/state-api.md
- FOUND: commit 54ff661
