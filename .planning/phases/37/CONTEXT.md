---
author: architect
responsible: architect
phase: 37
phase_type: documentation
hard_bar: false
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (DOC-01..DOC-09)
  - .planning/ROADMAP.md (Phase 37)
  - .planning/research/assistant-onboarding-SUMMARY.md
  - .planning/phases/36/SUMMARY.md (font-spoof now real — docs reflect real behavior)
---

# Phase 37: Coding-Assistant Onboarding Docs — CONTEXT

## What this phase delivers

Authoritative install + agent-skill documentation so any of the five major coding assistants — Claude Code, Codex CLI, Gemini CLI, Pi (pi.dev), and opencode — can adopt rod-cli via the agent-skill / instructions-file → shell-out path. (No MCP — rod-cli is a pure CLI/daemon, verified at HEAD.)

## The deliverables

A `docs/onboarding/` directory with:

1. **`README.md`** — Binary install (shared prerequisite): `go install` + `rod-cli install` for Chromium, with verify step (DOC-01).
2. **`claude-code.md`** — Claude Code: place SKILL.md at `~/.claude/skills/rod-cli/`, restart gotcha, verify step (DOC-03).
3. **`codex-cli.md`** — Codex CLI: official `.agents/skills/` path, path-conflict note, AGENTS.md secondary (DOC-04).
4. **`gemini-cli.md`** — Gemini CLI: `GEMINI.md` context file, no skills primitive, explicit "no MCP" note (DOC-05).
5. **`pi.md`** — Pi: `npm install -g --ignore-scripts @earendil-works/pi-coding-agent`, `~/.pi/agent/skills/` path, trust gotcha, explicit "no MCP" note (DOC-06).
6. **`opencode.md`** — opencode: native skills path, `.claude/skills/` compat, AGENTS.md secondary (DOC-07).

Plus:
- **`skills/rod-cli/SKILL.md`** updated with explicit "when to use" trigger phrases (DOC-02).
- **`README.md`** (root) updated to link to `docs/onboarding/` (DOC-09).

Each assistant section includes a **copy-paste install sequence** and a **concrete verify step** (DOC-08).

## Research grounding

All per-assistant paths and gotchas are verified against live 2026 docs and synthesized in `.planning/research/assistant-onboarding-SUMMARY.md`. Per-assistant detail files: `assistant-{claude,codex,gemini,pi,opencode}.md`.

## Key decisions

- **No MCP install path documented** — rod-cli is not an MCP server (verified at HEAD: no `mcp` subcommand, no JSON-RPC, no MCP dep).
- **Shared `~/.agents/skills/` substrate** noted where it applies (Codex, Pi, opencode).
- **Gemini uses `GEMINI.md`** (no skills primitive).
- **Pi has explicit "No MCP support" note** plus the project-trust gotcha.
