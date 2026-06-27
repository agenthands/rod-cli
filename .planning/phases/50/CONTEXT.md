---
author: architect
responsible: architect
phase: 50
phase_type: documentation
hard_bar: false
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (DOCS-01..04)
  - .planning/phases/49/SUMMARY.md (all tools built)
---

# Phase 50: Documentation & Discoverability — CONTEXT

## Goal

Ship complete documentation so Pi users discover, install, and use the extension —
with clear guidance on when to use the extension vs the existing Agent Skill.

## {P} What the phase can assume

- All 13 tools built and tested (Phases 47-49)
- Extension lives at `extensions/pi/`
- Existing docs: `docs/onboarding/pi.md`, `skills/rod-cli/SKILL.md`, top-level `README.md`

## {Q} What the phase must establish

1. `extensions/pi/README.md` — install instructions, prerequisites, tool catalog, verify step (DOCS-01)
2. Skill-vs-Extension comparison table in README (DOCS-02)
3. `docs/onboarding/pi.md` updated — references extension as richer path (DOCS-03)
4. `skills/rod-cli/SKILL.md` updated — mentions extension for Pi users (DOCS-03)
5. Top-level `README.md` updated — `extensions/pi/` in docs index (DOCS-04)

## Content requirements

### DOCS-01: extensions/pi/README.md
- Package name: `@agenthands/rod-cli-pi`
- Prerequisites: rod-cli binary (`go install github.com/agenthands/rod-cli@latest`), `rod-cli install` for Chromium, Pi coding agent
- Install: `pi install npm:@agenthands/rod-cli-pi` (once published) or local path
- Tool catalog: table of all 13 tools with descriptions
- Verify step: "Browse to example.com and tell me the page title"

### DOCS-02: Skill-vs-Extension
| Dimension | Skill (existing) | Extension (v2.2) |
|---|---|---|
| Setup | Copy SKILL.md | pi install |
| Tool discovery | LLM reads markdown | Pi "Available tools" catalog |
| Parameter validation | None | TypeBox schema |
| Lifecycle hooks | None | session_start/end |
| Cross-tool | All Agent-Skills assistants | Pi-only |

### DOCS-03: Cross-links
- `docs/onboarding/pi.md`: add "For a richer experience, install the Pi extension: ..."
- `skills/rod-cli/SKILL.md`: add Pi extension reference in the Pi section

### DOCS-04: Top-level README
- Add `extensions/pi/` link under existing docs index

## Files

| File | Status |
|------|--------|
| `extensions/pi/README.md` | NEW |
| `docs/onboarding/pi.md` | MODIFIED |
| `skills/rod-cli/SKILL.md` | MODIFIED |
| `README.md` | MODIFIED |

## Success criteria

1. README install instructions are copy-paste runnable
2. Tool catalog lists all 13 tools with correct descriptions
3. Skill-vs-Extension table accurately describes both paths
4. Cross-links resolve (no broken references)
5. No tool claim in README contradicts the actual code
