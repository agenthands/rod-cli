# Research Synthesis — Coding-Assistant Onboarding for rod-cli (v1.8)

Synthesized in-band from `assistant-{claude,codex,gemini,pi,opencode}.md` (all verified against live 2026 docs) + the rod-cli command surface at HEAD.

## The decisive finding: rod-cli is NOT an MCP server

Verified at HEAD (`cmd.go`, `go.mod`): rod-cli is a **pure CLI/daemon** — commands `open`/`goto`/`click`/`type`/`eval`/`snapshot`/`stealth-check`/… over a persistent background session. There is **no `mcp` subcommand, no JSON-RPC, no MCP dependency**. The historical `rod-mcp` MCP heritage is gone.

→ **The MCP install path is not documentable for any of the five assistants.** Every doc onboards rod-cli by teaching the agent to **shell out to the `rod-cli` binary**, via that tool's agent-skill or instructions-file mechanism. This makes the docs consistent and simpler (and is more honest than documenting an MCP server that doesn't exist).

## The unifying mechanism: the cross-tool `SKILL.md` standard

rod-cli already ships `skills/rod-cli/SKILL.md`. Four of five assistants consume the cross-tool Agent Skills standard; only Gemini lacks a skills primitive. A single skill directory under the shared `~/.agents/skills/` path is read by **Codex, Pi, and opencode** at once.

| Assistant | Primary onboarding path | Exact location | MCP? |
|-----------|------------------------|----------------|------|
| **Claude Code** | Agent Skill (`SKILL.md`) | `~/.claude/skills/rod-cli/` (user) or `.claude/skills/rod-cli/` (project) | n/a (not an MCP server) |
| **Codex CLI** | Agent Skill (`SKILL.md`), AGENTS.md secondary | official: `.agents/skills/` · `$HOME/.agents/skills/` · `/etc/codex/skills` (**path conflict** w/ community `~/.codex/skills` — pin per version) | n/a |
| **Gemini CLI** | **GEMINI.md context file** (no skills primitive) | `~/.gemini/GEMINI.md` (global) / `.gemini/GEMINI.md` (project); `context.fileName` can add `AGENTS.md` | n/a |
| **Pi** | Agent Skill (`SKILL.md`) via bash; Extension (TS) heavier alt | `~/.pi/agent/skills/` · `~/.agents/skills/` (global); `.pi/skills/` · `.agents/skills/` (project, after trust) | **No — explicit "No MCP" by design** |
| **opencode** | Native skills; reads `.claude/skills/` too | `.opencode/skills/rod-cli/` or `~/.config/opencode/skills/`; **also natively reads `.claude/skills/` + `~/.claude/skills/` + `.agents/skills/`** | n/a |

## SKILL.md frontmatter (the common schema)
Required: `name` (1–64, lowercase-alnum-hyphen), `description`. Optional across tools: `license`, `compatibility`, `metadata`, `allowed-tools` (experimental on Pi). Claude truncates `description`+when-to-use at 1,536 chars. rod-cli's existing frontmatter is valid; **improvement: add explicit "when to use" trigger phrases** to the description to improve auto-invocation.

## Binary install (shared prerequisite, all five)
A skill is markdown only — it does NOT bundle the binary. Every doc needs two parts: **(1)** get `rod-cli` on PATH (`go install` or a prebuilt binary) + run `rod-cli install` to fetch Chromium; **(2)** place the skill/instructions file. Plus a **verify step** (e.g. `rod-cli stealth-check` or `/rod-cli` invocation).

## Per-assistant gotchas
- **Claude Code:** creating `~/.claude/skills/` when it didn't exist at session start needs a `claude` restart; invocation name = directory name, not frontmatter `name`. Docs host is now `code.claude.com/docs`.
- **Codex:** skills-path is the one real conflict — official `.agents/skills` vs community `~/.codex/skills`; pin against the installed Codex version before publishing. Project `.codex/config.toml` loads only for trusted projects.
- **Gemini:** `gemini mcp add` defaults to project scope (moot here); Node ≥ 20.
- **Pi:** install norm is `npm install -g --ignore-scripts @earendil-works/pi-coding-agent`; project skills load only after the project is trusted (global `~/.pi/agent/skills/` avoids it); extensions run with full system permissions.
- **opencode:** install `curl -fsSL https://opencode.ai/install | bash` or `npm i -g opencode-ai`; `.claude/skills` fallback disable via `OPENCODE_DISABLE_CLAUDE_CODE_SKILLS`.

## Corrections to prior assumptions (verified at HEAD)
- The repo has **no `.agents/GEMINI.md`** (config.json points `claude_md_path` there but the file/dir doesn't exist) and **no root `AGENTS.md`/`CLAUDE.md`/`GEMINI.md`**. The docs phase creates the instruction files it references.
- go.mod = `go 1.25.1`, no `toolchain` directive; dev go is go1.26.0; the F1 fix targets **go1.26.1**.

## Open item routed to planning
- **Codex skills path** must be pinned against the installed Codex version during the docs phase (don't hard-code until verified live).
- Whether to also publish a heavier **Extension** (Pi TS / Gemini `gemini-extension.json`) is optional polish, not required for onboarding.

## Sources
Per-assistant source URLs are in the five `assistant-*.md` files (all official docs, fetched 2026-06-26).
