---
phase: 22-example-plugins
plan: 04
subsystem: docs
tags: [plugin, docs, examples, xss, cli-reference]
requires:
  - "Plan 22-01 engine fix (PluginRun -> RunFunc) shipped"
  - "Plan 22-03 polished plugins/examples/xss_scanner.js with getFindings/getRequestLog accessors"
provides:
  - "docs/plugins/examples/xss-scanner.md — flagship worked example (load -> drive page -> plugin run getFindings)"
  - "Corrected docs/plugins/cli-reference.md plugin run section (no stub wording)"
affects:
  - "docs/plugins/ reference set (cross-linked from the new examples dir)"
tech-stack:
  added: []
  patterns:
    - "Worked-example docs live one level deeper (docs/plugins/examples/) and cross-link refs with ../"
key-files:
  created:
    - docs/plugins/examples/xss-scanner.md
  modified:
    - docs/plugins/cli-reference.md
decisions:
  - "plugin run documented as invoking a named JS function via engine.RunFunc and printing its stringified result; no multi-plugin registry"
  - "Worked example uses a generic example.test URL and embeds no exploit infrastructure (T-22-08 accept)"
metrics:
  duration_min: 1
  completed: 2026-06-23
  tasks: 2
  files: 2
status: complete
---

# Phase 22 Plan 04: Example Plugins (XSS Scanner Doc + CLI Reference Fix) Summary

Documented the flagship XSS scanner as a complete worked example and corrected the Phase 21 CLI reference so `plugin run <name>` is described truthfully now that the Plan 01 engine fix made it functional (PEX-01).

## What Was Built

### Task 1 — `docs/plugins/examples/xss-scanner.md` (commit `4bc266f`)
A worked example walking the reader end-to-end:
- **Load:** `rod-cli plugin load ./plugins/examples/xss_scanner.js`.
- **How it watches:** describes the scanner's `onRequest` / `onResponse` / `onLoad` hooks and its use of `api.GetSnapshot()`, referencing (not duplicating) `../lifecycle-hooks.md` and `../state-api.md`.
- **Drive a vulnerable page:** a generic `example.test` URL with a payload query param — no exploit infrastructure embedded.
- **Observe findings:** `rod-cli plugin run getFindings` (and `getRequestLog`) with sample JSON output, explaining that `plugin run` invokes the named accessor and prints its JSON.
- `## See Also` and `## Source` sections cross-link the Phase 21 refs with the extra `../` (this file is one level deeper) and point at `../../plugins/examples/xss_scanner.js` plus the relevant `internal/plugin/` files.

### Task 2 — `docs/plugins/cli-reference.md` `plugin run` section (commit `ef85dc2`)
Surgically rewrote only the `plugin run <name>` section plus the Source footer clause that called it a stub:
- New behavior: `plugin run <name>` invokes the JS function named `<name>` in the loaded plugin's VM (via `PluginRun` → `engine.RunFunc`), calls it with no args, returns its stringified result. Canonical use `plugin run getFindings` documented; noted single-loaded-plugin (no registry).
- New output description: prints the function's returned value (JSON for the example accessors); undefined/null prints empty.
- New error table verified against `internal/plugin/engine.go` `RunFunc`: `plugin function name is required` (empty name, from `actions/plugin.go`), `function "<name>" not found`, `"<name>" is not a callable function`, `error calling "<name>"`, `plugin engine not initialized`.
- Removed the old `Triggered plugin <name>` output, the "current stub (known limitation)" / "not yet fully implemented" paragraph, and the "real execution path today is `plugin load`" caveat (replaced with a brief note that hooks still fire after load; `plugin run` invokes named accessors).
- `plugin load`, `plugin list`, See Also, and Thin-Client sections left intact.

## Verification

- Task 1 automated: file exists, contains `plugin run getFindings`, cross-links `../lifecycle-hooks.md`, `../state-api.md`, `../cli-reference.md`. PASS.
- Task 2 automated: contains `invokes the JS function` and `plugin run getFindings`; literals `Triggered plugin` and `not yet fully implemented` are GONE; `function "<name>" not found` documented; `plugin load` / `plugin list` headers unchanged. PASS.
- All `plugin run` behavior/error wording verified against `actions/plugin.go` (`PluginRun` guards empty name, delegates to `RunFunc`) and `internal/plugin/engine.go` (`RunFunc` nil-VM / not-found / not-callable / call-error paths). No invented behavior.

## Deviations from Plan

None - plan executed exactly as written.

## Threat Surface

No new security-relevant surface introduced (docs only). T-22-08 honored: the worked example references a generic `example.test` URL and the user-space script only; no exploit infrastructure embedded. T-22-09 mitigated: the cli-reference `plugin run` section now matches the shipped `RunFunc` behavior.

## Self-Check: PASSED

- FOUND: docs/plugins/examples/xss-scanner.md
- FOUND: docs/plugins/cli-reference.md (modified)
- FOUND commit: 4bc266f
- FOUND commit: ef85dc2
