# Retrospective

## Cross-Milestone Trends

| Milestone | Ph | Plans | Time | Efficiency | Major Learnings |
| --------- | -- | ----- | ---- | ---------- | --------------- |
| v1.3      | 5  | 5     | 1d   | High       | Godoll's abstraction drastically simplifies stealth/evasion code compared to manual rod.Page hijacking. |
| v1.5      | 3  | 11    | 1d   | High       | A "docs-only" milestone surfaced 3 real engine gaps; validating examples against the live binary (not just source) is what caught them. |

## Milestone: v1.5 — Plugin Ecosystem Documentation

**Shipped:** 2026-06-23
**Phases:** 3 | **Plans:** 11

### What Was Built
- `docs/plugins/` reference pages: lifecycle-hooks.md, state-api.md, cli-reference.md (grounded verbatim in source).
- Example plugins: polished flagship `xss_scanner.js`, four per-hook recipes, and a copyable `starter.js`, each with a doc under `docs/plugins/examples/`.
- `authoring.md` zero-to-running tutorial + `docs/plugins/README.md` index, linked one-click from the top-level README.
- Three small engine fixes surfaced while documenting: `api.GetLocalStorage()`, a now-functional `plugin run` (`PluginEngine.RunFunc`), and CDP DOM-domain enablement so `onDOMNodeInserted` fires. Removed the always-on startup banner.

### What Worked
- Discuss-phase grey-area review caught the "document-the-gap vs. small-fix" decisions early (localStorage, `plugin run` observability) so the docs describe a system that actually works rather than an aspirational one.
- Live end-to-end validation against the bundled `internal/plugin/scanner/testserver` vulnerable app proved all four hooks fire and `plugin run getX` returns real data — automated/source checks alone would have shipped a silently-broken `onDOMNodeInserted`.

### What Was Inefficient
- The startup banner / output verbosity wasn't caught until manual validation; an LLM-facing CLI should default to quiet output. Several validation round-trips were spent on it.
- ROADMAP milestone-scoping miscounted shipped v1.4 phases (their plan-refs in a `<details>` block) as v1.5 phases, blocking milestone completion until the block was collapsed to prose.

### Patterns Established
- "Docs milestone" gaps are legitimately closed with small corrective engine fixes (user-approved) rather than documenting around them.
- Validate runnable examples by driving the real daemon/browser, not just goja-level load checks.

### Key Lessons
- Documenting against the *live binary* (load the plugin, drive a page, read results) catches wired-but-silent behavior that source inspection misses (the `onDOMNodeInserted` DOM-domain gap).
- Keep CLI output token-efficient by default for LLM/pipe callers; gate decorative output behind a TTY check or remove it.

### Cost Observations
- Heavy use of isolated subagents (planner/executor/verifier/reviewer per phase) kept the orchestrator context lean across a long autonomous run.

## Milestone: v1.3 — Godoll Migration

**Shipped:** 2026-06-21
**Phases:** 5 | **Plans:** 5

### What Was Built
- Godoll Browser Installation Command (`rod-cli install`)
- Stealth and Remote Browser Integration (`godoll.NewBrowser`)
- Network Interception and Evasion (`godoll/network`)
- Robust Retry Mechanism for Actions (`godoll/retry`)
- Comprehensive test coverage for Godoll Migration features

### What Worked
- Replacing brittle manual intercept code with the powerful `godoll/network` interceptor greatly cleaned up `rod-cli`'s context struct.
- Wrapping everything in `retry.Fetch` eliminated dozens of flaky failure modes on page loads.

### What Was Inefficient
- N/A

### Patterns Established
- Use `godoll` as the definitive driver wrapper rather than naked `rod.Page` primitives where possible.

### Key Lessons
- Network hijacking needs careful synchronization, `godoll/network` handles this natively.

### Cost Observations
- N/A
