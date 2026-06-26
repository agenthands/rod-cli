---
author: architect
responsible: architect
phase: 37
status: executed
parent_artifacts:
  - .planning/phases/37/CONTEXT.md
  - .planning/REQUIREMENTS.md (DOC-01..DOC-09)
  - .planning/research/assistant-onboarding-SUMMARY.md
---

# Phase 37: Coding-Assistant Onboarding Docs — SUMMARY

## Execution result: ✅ COMPLETE

7 new files created, 2 existing files updated. All 9 DOC requirements met.

## Files created

| File | Content |
|------|---------|
| `docs/onboarding/README.md` | Shared binary install + per-assistant index (DOC-01) |
| `docs/onboarding/claude-code.md` | Claude Code: `~/.claude/skills/rod-cli/`, restart gotcha, verify (DOC-03) |
| `docs/onboarding/codex-cli.md` | Codex CLI: `.agents/skills/`, path-conflict note, AGENTS.md (DOC-04) |
| `docs/onboarding/gemini-cli.md` | Gemini CLI: `GEMINI.md`, no skills primitive, no MCP (DOC-05) |
| `docs/onboarding/pi.md` | Pi: `~/.pi/agent/skills/`, npm install, trust gotcha, no MCP (DOC-06) |
| `docs/onboarding/opencode.md` | opencode: native + Claude compat, AGENTS.md (DOC-07) |

## Files updated

| File | Change |
|------|--------|
| `skills/rod-cli/SKILL.md` | Enhanced `description` with explicit "when to use" trigger phrases (DOC-02) |
| `README.md` | Added link to `docs/onboarding/` under Documentation (DOC-09) |

## Requirement traceability

| REQ-ID | Status | Evidence |
|--------|--------|----------|
| DOC-01 | ✅ | `docs/onboarding/README.md` §Prerequisites: `go install`, prebuilt, `rod-cli install`, verify |
| DOC-02 | ✅ | `SKILL.md` frontmatter now has explicit trigger phrases |
| DOC-03 | ✅ | `claude-code.md`: path, restart gotcha, `/rod-cli` verify |
| DOC-04 | ✅ | `codex-cli.md`: `.agents/skills/`, path-conflict note, AGENTS.md |
| DOC-05 | ✅ | `gemini-cli.md`: `GEMINI.md` context, no MCP, no skills primitive |
| DOC-06 | ✅ | `pi.md`: `~/.pi/agent/skills/`, npm install, trust gotcha, no MCP |
| DOC-07 | ✅ | `opencode.md`: native + Claude compat paths |
| DOC-08 | ✅ | Every guide has copy-paste sequence + concrete verify step |
| DOC-09 | ✅ | No MCP install path documented; README links to onboarding; claims verified against real command surface |
