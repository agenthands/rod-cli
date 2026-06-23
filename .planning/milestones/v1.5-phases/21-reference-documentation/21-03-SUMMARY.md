---
phase: 21-reference-documentation
plan: 03
subsystem: plugin-docs
tags: [documentation, plugin, cli, reference]
status: complete
requires:
  - cmd.go (plugin subcommands + client guard)
  - actions/plugin.go (PluginLoad/PluginList/PluginRun behavior)
provides:
  - docs/plugins/cli-reference.md (authoritative plugin CLI reference)
affects:
  - docs/plugins/ (reference tree)
tech-stack:
  added: []
  patterns:
    - Markdown reference template (H1 → intro → H2 per command → bash blocks → cross-links → Source footer)
key-files:
  created:
    - docs/plugins/cli-reference.md
  modified: []
decisions:
  - "Documented plugin run honestly as a current stub with a known registry limitation, per CONTEXT decision and source comment."
  - "Used relative cross-links (./lifecycle-hooks.md, ./state-api.md) matching the sibling lifecycle-hooks.md style; state-api.md is produced by a sibling plan in the same wave."
metrics:
  duration: ~5m
  completed: 2026-06-22
  tasks: 1
  files: 1
---

# Phase 21 Plan 03: Plugin CLI Reference Summary

Authored `docs/plugins/cli-reference.md` documenting the three plugin CLI commands (`plugin load <path>`, `plugin list`, `plugin run <name>`) with arguments, behavior, output format, and error/exit conditions, grounded verbatim in `cmd.go` and `actions/plugin.go`.

## What Was Built

- **`docs/plugins/cli-reference.md`** — H1 title, thin-client intro, one H2 per command, `bash`/`json` fenced blocks, a Thin-Client Model section, See Also cross-links, and a Source footer.
  - `plugin load <path>`: required positional path; success string `Plugin loaded successfully from <path>`; client-side error `plugin path is required` (no path) and daemon error `failed to open script file <path>` (missing/unreadable file), both non-zero exit. Noted that loading is what activates lifecycle hooks.
  - `plugin list`: no args; emits `No active plugins loaded.` or a JSON array of loaded plugin paths (both forms documented).
  - `plugin run <name>`: documented honestly as a **current stub** — `PluginRun` returns `Triggered plugin <name>` without executing a registry-resolved plugin; registry is a known limitation per the source comment; hooks firing after `plugin load` is the real execution path.
  - Thin-client note: each command sends a `daemon.Request{Command: "plugin-load"|"plugin-list"|"plugin-run"}` and prints the response; failures exit non-zero.
  - Cross-links to `lifecycle-hooks.md` and `state-api.md`; Source footer cites `cmd.go`, `actions/plugin.go`, `internal/plugin/engine.go`, and `daemon/daemon.go`.

## Source Grounding

- `actions/plugin.go` lines 11-49: confirmed exact strings `plugin path is required`, `Plugin loaded successfully from %s`, `No active plugins loaded.`, JSON marshal of plugin list, `Triggered plugin %s`, and the registry comment.
- `cmd.go` lines 398-429: confirmed the `plugin path is required` client guard and the `plugin-load`/`plugin-list`/`plugin-run` dispatch.
- `internal/plugin/engine.go` line 34: confirmed `failed to open script file %s`.
- `daemon/daemon.go` lines 201-206: confirmed dispatch to the actions functions.

## Deviations from Plan

None - plan executed exactly as written.

## Verification

Automated grep checks passed: file exists; contains `plugin load`, `plugin list`, `plugin run`, `plugin path is required`, `failed to open script file`, `No active plugins loaded.`, `actions/plugin.go`, `lifecycle-hooks.md`, `state-api.md`.

## Known Stubs

The documented `plugin run` command is itself a runtime stub in the codebase (`actions.PluginRun`), but this is an existing source-level limitation, not a doc stub — it is documented honestly as a known limitation (registry needed). No documentation placeholders were introduced.

## Self-Check: PASSED

- FOUND: docs/plugins/cli-reference.md
- FOUND: commit a603cd3
