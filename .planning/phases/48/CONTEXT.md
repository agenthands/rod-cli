---
author: architect
responsible: architect
phase: 48
phase_type: implementation
hard_bar: true
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (TOOLS-01..07, INTEG-01)
  - .planning/ROADMAP.md (Phase 48 section)
  - .planning/phases/47/SUMMARY.md (foundation built)
  - .planning/research/FEATURES.md (tool catalog + naming)
---

# Phase 48: Core Browser Tools + Integration Test — CONTEXT

## Goal

Ship the 7 table-stakes browser tools so a Pi agent can navigate, read, click, type,
evaluate JS, screenshot, and wait — plus an automated integration test that exercises
the full workflow against a fixture page. This is the MVP vertical slice: a Pi user can
browse the web through rod-cli.

## {P} What the phase can assume (precondition)

- Phase 47 foundation is built: `execRodCli()` wrapper exists with input validation,
  timeout handling, and throw-on-error discipline
- `findRodCli()` resolves the binary; lifecycle hooks are registered
- `src/types.ts` exports `SessionParam` (TypeBox schema)
- `src/index.ts` has `setPi(pi)` called before any tool executes
- rod-cli commands exist: `goto`, `snapshot`, `click`, `type`, `eval`, `screenshot`,
  `wait`, `close` — all confirmed via SMTC against `cmd.go`
- `--raw` flag is prepended by `execRodCli` for machine-readable output

## {Q} What the phase must establish (postcondition)

1. 7 tool files under `extensions/pi/src/tools/` — each exports a `register*` function
2. `browse_goto` navigates to a URL; first call implicitly starts the daemon via rod-cli's
   `EnsureDaemon`. Optional `session` param for named sessions. (TOOLS-01)
3. `browse_snapshot` returns accessibility-tree markdown, truncated to 50KB/2000 lines.
   Optional `selector` to scope. (TOOLS-02)
4. `browse_click` clicks an element by CSS selector, with optional `doubleClick`. (TOOLS-03)
5. `browse_type` types text with humanized keystrokes (maps 1:1 to `rod-cli type`).
   Does NOT submit — `browse_fill_form` (Phase 49) handles instant fill+submit. (TOOLS-04)
6. `browse_eval` evaluates JavaScript (max 10KB expression), returns result truncated to
   50KB. (TOOLS-05)
7. `browse_screenshot` captures a screenshot — optional `selector`, `fullPage`, and
   `format: "png"|"jpeg"`. (TOOLS-06)
8. `browse_wait` waits for a selector to appear or a fixed `timeout`. (TOOLS-07)
9. `src/index.ts` registers all 7 tools during the default export.
10. Integration test starts a loopback fixture page, runs the full workflow
    (goto → snapshot → click → type → eval → screenshot → wait → close), asserts
    correct output, verifies daemon cleanup. (INTEG-01)

## Invariants (cross-cutting, must preserve across phase)

- **I1 (error discipline):** All errors in `execute()` are THROWN, never returned as `isError`.
- **I2 (enum discipline):** All enum params use `StringEnum` from `@earendil-works/pi-ai`
  (never `Type.Enum`).
- **I3 (promptGuidelines):** Each tool's `promptGuidelines` NAMES the tool explicitly
  ("Use browse_snapshot..."), never "Use this tool..."
- **I4 (session param):** Every tool accepts optional `session` param from `SessionParam`.
- **I5 (1:1 CLI mapping):** Each tool maps to exactly one rod-cli command (no multi-call
  tools in this phase — `browse_type` uses `type`, not `fill`).

## Tool-to-CLI mapping

| Tool | rod-cli command | Key params |
|------|----------------|------------|
| `browse_goto` | `goto <url>` | url (required), session |
| `browse_snapshot` | `snapshot [--selector <sel>]` | selector (optional), session |
| `browse_click` | `click <sel> [--double]` | selector (required), doubleClick (optional), session |
| `browse_type` | `type <sel> <text>` | selector, text (required), session |
| `browse_eval` | `eval <expression>` | expression (required, ≤10KB), session |
| `browse_screenshot` | `screenshot [--selector <sel>] [--format png\|jpeg] [--full-page]` | selector (optional), fullPage (optional), format (optional), session |
| `browse_wait` | `wait <sel>` or `wait --timeout <ms>` | selector (optional), timeout (optional), session |

## Integration test design (INTEG-01)

A vitest test in `src/__tests__/integration.test.ts`:
1. Starts a Node HTTP server on a random port serving a fixture HTML page
   (form with input + button, heading, paragraph)
2. Calls `execRodCli(["goto", "http://127.0.0.1:<port>"])` — asserts success
3. Calls `execRodCli(["snapshot"])` — asserts output contains fixture text
4. Calls `execRodCli(["click", "#btn"])` — asserts success
5. Calls `execRodCli(["type", "#input", "hello"])` — asserts success
6. Calls `execRodCli(["eval", "document.title"])` — asserts correct title
7. Calls `execRodCli(["screenshot", "--format", "png"])` — not empty
8. Calls `execRodCli(["wait", "#input"])` — asserts success
9. Calls `execRodCli(["close"])` — asserts daemon cleanup
10. Asserts `rod-cli sessions` returns empty or no daemon running

Test is skipped if `findRodCli()` returns null (CI without rod-cli installed).
Uses `describe.skipIf(!rodCliPath)`.

## File layout

| File | Purpose |
|------|---------|
| `extensions/pi/src/tools/goto.ts` | `browse_goto` |
| `extensions/pi/src/tools/snapshot.ts` | `browse_snapshot` |
| `extensions/pi/src/tools/click.ts` | `browse_click` |
| `extensions/pi/src/tools/type.ts` | `browse_type` |
| `extensions/pi/src/tools/eval.ts` | `browse_eval` |
| `extensions/pi/src/tools/screenshot.ts` | `browse_screenshot` |
| `extensions/pi/src/tools/wait.ts` | `browse_wait` |
| `extensions/pi/src/tools/index.ts` | Barrel export: `registerAllCoreTools(pi)` |
| `extensions/pi/src/index.ts` | Updated: calls `registerAllCoreTools` |
| `extensions/pi/src/__tests__/integration.test.ts` | INTEG-01 workflow test |

## Success criteria (falsifiable)

1. Each tool file exports a `register*` function that calls `pi.registerTool({...})`. (TOOLS-01..07)
2. `pi.registerTool` is called with `name`, `label`, `description`, `promptSnippet`,
   `promptGuidelines` (naming the tool explicitly), `parameters` (TypeBox), `execute`.
3. Every `execute()` builds args from params, calls `execRodCli(args, { signal })`,
   and returns `{ content: [{ type: "text", text: result.stdout }] }`.
4. `StringEnum` used for `format` and any enum param. `Type.Optional` for optional params.
5. Integration test passes: full workflow succeeds against loopback fixture.
6. Existing vitest suite (107 tests from Phase 47) still passes — no regression.
