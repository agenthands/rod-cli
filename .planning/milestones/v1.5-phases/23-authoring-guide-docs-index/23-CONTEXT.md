# Phase 23: Authoring Guide & Docs Index - Context

**Gathered:** 2026-06-23
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous; defaults locked — phase fully determined by shipped Phases 21–22)

<domain>
## Phase Boundary

The capstone documentation phase. Deliver:
1. `docs/plugins/authoring.md` — a single first-plugin tutorial from zero to a running plugin (PDOC-01).
2. `docs/plugins/README.md` — the docs index that reaches every plugin doc page and example (PDOC-05).
3. A one-click link from the top-level `README.md` into `docs/plugins/` (PDOC-05).

No new engine code. Builds entirely on Phases 21 (reference pages) and 22 (example plugins + the now-functional `plugin run`).
</domain>

<decisions>
## Implementation Decisions

### Authoring tutorial (authoring.md)
- Walk zero → running plugin against the CURRENT shipped binary, in this order: (1) what a plugin is (a JS file with optional lifecycle hooks run in the goja sandbox — no console/fs/require), (2) project structure / where to put the script, (3) writing handlers (use one or two hooks, e.g. onRequest + a getter), (4) `rod-cli plugin load <path>`, (5) open/drive a page, (6) `rod-cli plugin run <getter>` to inspect results, (7) next steps.
- Must reflect the current behavior accurately:
  - `plugin run <name>` is functional (invokes the named JS function, returns its result) — show `plugin run getResults`/`getFindings`.
  - The `api` global + hooks bind when a browser page is open in the session — tell the reader to open a page (e.g. `rod-cli goto <url>`) so hooks attach.
  - Output is clean (the startup banner was removed); no need to mention `--no-banner` except as an aside if at all.
- Recommend starting from the Phase 22 starter (`plugins/examples/starter.js`) — copy it, fill in a hook, load it.
- LINK OUT, do not duplicate: reference `lifecycle-hooks.md`, `state-api.md`, `cli-reference.md` (Phase 21) and the starter/examples (Phase 22) for details. The tutorial is the narrative glue, not a re-statement of the reference pages.

### Docs index (docs/plugins/README.md)
- A short intro + a linked list reaching every plugin doc: authoring.md, lifecycle-hooks.md, state-api.md, cli-reference.md, and the examples (examples/xss-scanner.md, examples/recipes.md, examples/starter.md). Group sensibly (Getting Started → Reference → Examples).
- All links relative and verified to resolve.

### README link
- Add a "Plugin Development" (or similarly named) bullet to the top-level `README.md` `## Documentation` section linking to `docs/plugins/README.md` (or `docs/plugins/`), reachable in one click.

### Claude's Discretion
- Exact tutorial prose, the specific example hook used in the walkthrough, headings, and index grouping/wording. All low-stakes and dictated by the existing Phase 21–22 artifacts.
</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets / inputs to link
- `docs/plugins/lifecycle-hooks.md`, `docs/plugins/state-api.md`, `docs/plugins/cli-reference.md` (Phase 21 refs).
- `docs/plugins/examples/xss-scanner.md`, `docs/plugins/examples/recipes.md`, `docs/plugins/examples/starter.md` (Phase 22 example docs).
- `plugins/examples/starter.js` (the copyable starting point the tutorial uses).
- Top-level `README.md` `## Documentation` section (bullet list) — add the plugin-docs link here.

### Established Patterns
- Doc style/structure mirrors the existing `docs/plugins/*.md` pages (intro → sections → code fences → See Also → cross-links).
- `plugin run <getter>` is the canonical "inspect results" UX; open a page before loading so hooks bind.

### Integration Points
- `docs/plugins/README.md` is the hub; the top-level README links to it; authoring.md links to refs + examples.
</code_context>

<specifics>
## Specific Ideas

- Fixed filenames per success criteria: `docs/plugins/authoring.md`, `docs/plugins/README.md`.
- Tutorial steps must work verbatim against the shipped binary (the Phase 22 live validation confirmed the load → drive → `plugin run getX` flow works).
</specifics>

<deferred>
## Deferred Ideas

- A hosted/searchable docs site and a plugin registry/marketplace — explicitly future (REQUIREMENTS Future Requirements), out of scope.
</deferred>
