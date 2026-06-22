# Roadmap: v1.5 Plugin Ecosystem Documentation

## Overview

The v1.4 plugin engine, lifecycle hooks, state/context API, and CLI commands already ship and work (phases 17–20). This milestone makes them *usable by others*: a complete `docs/plugins/` documentation tree backed by runnable example plugins that exercise every hook and API. We document the system as built — no new engine code. The journey runs reference-first: lock down authoritative reference pages for the hooks, state API, and CLI; ship polished runnable example plugins (flagship XSS scanner, per-hook recipes, copyable starter); then weave it together with a first-plugin authoring tutorial and a docs index linked from the README.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Continuing from v1.4 (ended at Phase 20). This milestone runs Phases 21–23.

- [ ] **Phase 21: Reference Documentation** - Authoritative reference pages for lifecycle hooks, state/context API, and CLI commands
- [ ] **Phase 22: Example Plugins** - Polished XSS scanner, per-hook recipe plugins, and a copyable starter template
- [ ] **Phase 23: Authoring Guide & Docs Index** - First-plugin tutorial plus a `docs/plugins/` index linked from the README

## Phase Details

### Phase 21: Reference Documentation

**Goal**: A plugin author can look up every lifecycle hook, every state/context API call, and every plugin CLI command in authoritative reference pages, each grounded in the v1.4 implementation.
**Depends on**: Nothing (first phase of this milestone; builds on shipped v1.4 code)
**Requirements**: PDOC-02, PDOC-03, PDOC-04
**Success Criteria** (what must be TRUE):

  1. A reader can open `docs/plugins/lifecycle-hooks.md` and find each of `onRequest`, `onResponse`, `onLoad`, `onDOMNodeInserted` with its JS handler name, the CDP event it maps to, and the payload shape (e.g. `event.Request.URL`).
  2. A reader can open `docs/plugins/state-api.md` and find the `api` global's `GetSnapshot()` and `GetCookies()` calls documented with return shapes and a worked usage snippet, plus how localStorage and network context are reached.
  3. A reader can open `docs/plugins/cli-reference.md` and find `plugin load <path>`, `plugin list`, and `plugin run <name>` documented with their arguments, behavior, output format, and exit/error conditions (e.g. missing path error).
  4. Every documented hook name, API method, and command in these pages matches the actual `internal/plugin/` and `actions/plugin.go` source (no invented surfaces).

**Plans**: 4 plans
**Wave 1**

- [x] 21-01-PLAN.md — Add GetLocalStorage() accessor + integration test (corrective engine fix)
- [x] 21-02-PLAN.md — Write docs/plugins/lifecycle-hooks.md (four lifecycle hooks)
- [x] 21-03-PLAN.md — Write docs/plugins/cli-reference.md (plugin load/list/run)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 21-04-PLAN.md — Write docs/plugins/state-api.md (GetSnapshot/GetCookies/GetLocalStorage)

### Phase 22: Example Plugins

**Goal**: A user can read and run a complete polished worked example, a small recipe for each lifecycle hook, and a copyable starter — all under a documented examples tree.
**Depends on**: Phase 21
**Requirements**: PEX-01, PEX-02, PEX-03
**Success Criteria** (what must be TRUE):

  1. A user can read `docs/plugins/examples/xss-scanner.md` alongside the polished `xss_scanner.js`, follow it, load the plugin, and observe it collecting findings end-to-end.
  2. A user can open a per-hook recipe plugin for each of the four lifecycle hooks (a small script + short note) and run it to see that hook fire.
  3. A plugin author can copy a `starter`/template plugin that defines empty stubs for all four hooks plus a results accessor, load it unchanged, and confirm it runs without errors.
  4. Each example references the Phase 21 reference pages for the hooks and API calls it uses, and every example script actually loads via `plugin load`.

**Plans**: TBD

### Phase 23: Authoring Guide & Docs Index

**Goal**: A first-time plugin author can follow a single tutorial from zero to a running plugin, and any reader can discover all plugin docs from an index linked off the README.
**Depends on**: Phase 22
**Requirements**: PDOC-01, PDOC-05
**Success Criteria** (what must be TRUE):

  1. A reader can open `docs/plugins/authoring.md` and follow it end-to-end — project structure, writing handlers, `plugin load`, and `plugin run`/inspecting results — to get their first plugin running, with steps that work against the shipped binary.
  2. The authoring guide links out to the Phase 21 reference pages and the Phase 22 starter/example plugins instead of duplicating them.
  3. A reader can open `docs/plugins/README.md` (the index) and reach every plugin doc page and example from it.
  4. A reader starting at the top-level `README.md` can find a link into `docs/plugins/` and reach the plugin documentation index in one click.

**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 21 → 22 → 23

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 21. Reference Documentation | 4/4 | Complete   | 2026-06-22 |
| 22. Example Plugins | 0/TBD | Not started | - |
| 23. Authoring Guide & Docs Index | 0/TBD | Not started | - |

---

<details>
<summary>✅ v1.4 Plugin Architecture (Phases 17–20) — SHIPPED 2026-06-22</summary>

### Phase 17: Engine Sandbox

**Goal**: Implement the plugin engine sandbox and script loader.
**Requirements**: PLUG-01, PLUG-02
**Plans**: 1/1 complete

- [x] 17-01-PLAN.md

### Phase 18: Lifecycle Emitters

**Goal**: Expose godoll browser events to the plugin engine.
**Requirements**: PLUG-03, PLUG-04, PLUG-05, PLUG-06
**Plans**: 1/1 complete

- [x] 18-01-PLAN.md

### Phase 19: State Context Sharing

**Goal**: Allow scripts to securely read the DOM snapshot and network state.
**Requirements**: PLUG-10, PLUG-11
**Plans**: 1/1 complete

- [x] 19-01-PLAN.md

### Phase 20: Plugin CLI Interface

**Goal**: Expose commands to load, list, and run plugins manually.
**Requirements**: PLUG-07, PLUG-08, PLUG-09
**Plans**: 1/1 complete

- [x] 20-01-PLAN.md

</details>
