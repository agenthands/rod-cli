---
author: architect
responsible: architect
phase: 37
status: verified
verdict: passed
parent_artifacts:
  - .planning/phases/37/CONTEXT.md
  - .planning/phases/37/SUMMARY.md
---

# Phase 37: Coding-Assistant Onboarding Docs — VERIFICATION

## Verdict: ✅ PASSED

All 9 DOC requirements are met. The `docs/onboarding/` tree documents rod-cli onboarding for all 5 assistants, accurate to the real (MCP-free) command surface.

## Evidence

| DOC-ID | Verification |
|--------|-------------|
| DOC-01 | Binary install section exists, covers `go install` + `rod-cli install`, has verify step |
| DOC-02 | SKILL.md frontmatter updated with explicit "when to use" trigger phrases |
| DOC-03 | Claude Code guide: user+project paths, restart gotcha, `/rod-cli` verify |
| DOC-04 | Codex guide: `.agents/skills/`, path-conflict note, AGENTS.md instructions |
| DOC-05 | Gemini guide: `GEMINI.md` context file, explicit "no skills primitive", no MCP |
| DOC-06 | Pi guide: `~/.pi/agent/skills/`, npm install, trust gotcha, "No MCP support" note |
| DOC-07 | opencode guide: native path, Claude compat paths, AGENTS.md |
| DOC-08 | Every guide has copy-paste install commands + concrete verify step |
| DOC-09 | No MCP install path in any doc; all MCP mentions are negative/corrective; README links to onboarding |

## Cross-checks

- `grep -rni "mcp" docs/onboarding/` → all hits are "NOT an MCP server" / "no MCP"
- `grep "onboarding" README.md` → link present
- Files: 7 new docs created, 2 updated
