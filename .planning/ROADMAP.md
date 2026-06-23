# Roadmap: rod-cli

## Milestones

- ✅ **v1.0 Core CLI Foundation** — shipped
- ✅ **v1.1 Stealth & Humanization** — shipped ([archive](milestones/v1.1-ROADMAP.md))
- ✅ **v1.2 First-Class Agent Skills & Documentation** — shipped
- ✅ **v1.3 Godoll Migration** — shipped ([archive](milestones/v1.3-ROADMAP.md))
- ✅ **v1.4 Plugin Architecture** — shipped ([archive](milestones/v1.4-ROADMAP.md))
- ✅ **v1.5 Plugin Ecosystem Documentation** — Phases 21–23 (shipped 2026-06-23) ([archive](milestones/v1.5-ROADMAP.md))

Full per-phase detail for each shipped milestone lives under `.planning/milestones/`.
Start the next milestone with `/gsd-new-milestone`.

## Phases

<details>
<summary>✅ v1.5 Plugin Ecosystem Documentation (Phases 21–23) — SHIPPED 2026-06-23</summary>

- [x] Phase 21: Reference Documentation (4/4 plans) — completed 2026-06-22
- [x] Phase 22: Example Plugins (5/5 plans) — completed 2026-06-23
- [x] Phase 23: Authoring Guide & Docs Index (2/2 plans) — completed 2026-06-23

Delivered a complete `docs/plugins/` tree — lifecycle-hooks, state-api, and cli-reference pages; a flagship XSS-scanner worked example, per-hook recipes, and a copyable starter; an authoring tutorial and a README-linked docs index — backed by runnable example plugins exercising every hook and the state API. Surfaced and fixed three small engine gaps along the way (`api.GetLocalStorage()`, functional `plugin run`, and CDP DOM-domain enable for `onDOMNodeInserted`) and removed the always-on startup banner for token-efficiency. Full detail: [milestones/v1.5-ROADMAP.md](milestones/v1.5-ROADMAP.md).

</details>

Earlier milestones (v1.0–v1.4) are archived under `.planning/milestones/`.

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 21. Reference Documentation | v1.5 | 4/4 | Complete | 2026-06-22 |
| 22. Example Plugins | v1.5 | 5/5 | Complete | 2026-06-23 |
| 23. Authoring Guide & Docs Index | v1.5 | 2/2 | Complete | 2026-06-23 |
