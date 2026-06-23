# Milestones

## v1.5 Plugin Ecosystem Documentation (Shipped: 2026-06-23)

**Phases completed:** 3 phases, 11 plans, 7 tasks

**Key accomplishments:**

- Polished flagship XSS-scanner example plus per-hook recipes and a copyable starter (`plugins/examples/` + `docs/plugins/examples/`), validated live end-to-end against the bundled vulnerable test app; landed three small engine fixes (`api.GetLocalStorage()`, functional `plugin run` via `RunFunc`, CDP DOM-domain enable for `onDOMNodeInserted`) and removed the always-on startup banner for token-efficiency.
- Authoritative `docs/plugins/lifecycle-hooks.md` documenting all four plugin lifecycle hooks (onRequest, onResponse, onLoad, onDOMNodeInserted) with their CDP proto types, key payload fields, and a worked snippet each, grounded in `internal/plugin/lifecycle.go`.
- docs/plugins/authoring.md — a zero-to-running tutorial (copy starter → write onResponse hook → plugin load → goto → plugin run getResults) that reflects the shipped binary and links out to the Phase 21 refs and Phase 22 examples.
- Plugin docs index (docs/plugins/README.md) grouping all seven plugin doc pages into Getting Started / Reference / Examples, with a one-click Plugin Development link added to the top-level README.

---

## v1.4 Plugin Architecture (Shipped: 2026-06-22)

**Phases completed:** 4 phases (17–20), 4 plans

**Key accomplishments:**

- Phase 17 — Engine Sandbox: embedded script engine and plugin loader inside the daemon.
- Phase 18 — Lifecycle Emitters: `OnRequest`, `OnResponse`, `OnLoad`, `OnDOMNodeInserted` hooks.
- Phase 19 — State Context Sharing: plugins read token-optimized snapshots, cookies, and localStorage.
- Phase 20 — Plugin CLI Interface: `rod-cli plugin load` / `list` / `run`.
- Shipped a working XSS scanner plugin demonstrating the engine end-to-end.

---

## v1.3 Godoll Migration (Shipped: 2026-06-21)

**Phases completed:** 1 phases, 1 plans, 0 tasks

**Key accomplishments:**

- (none recorded)

---
