---
author: project-researcher
responsible: architect
emitted_at: 2026-06-26T00:00:00Z
phase: null
artifact_kind: research_note
parent_artifacts:
  - .planning/milestones/v1.6-MILESTONE-AUDIT.md
status: draft
---

# Research — OpenAI Codex CLI: how to register rod-cli so the `codex` agent can invoke it

**Answer (1–3 sentences):** Codex CLI (the `codex` terminal agent) offers two first-class
mechanisms for rod-cli, both verified against current (2026) official OpenAI docs: (a) register
rod-cli as a **stdio MCP server** in `~/.codex/config.toml` under `[mcp_servers.<name>]` (or via
`codex mcp add`), so the agent calls rod-cli's tools over MCP; and (b) ship a **SKILL.md agent
skill** (the same open SKILL.md standard) and/or document rod-cli usage in **AGENTS.md**, where
the skill/prompt instructs the agent to shell out to the `rod` binary. Confidence: HIGH for MCP
and AGENTS.md (official docs, exact syntax confirmed); HIGH for skills-exist, MEDIUM on the exact
skills storage path (official docs and third-party guides disagree — see Unknowns).
**Recommendation:** Document BOTH paths for Codex users. Primary = the SKILL.md agent skill (rod-cli
already ships as a skill; Codex natively supports SKILL.md) instructing the agent to run the `rod`
CLI; secondary = AGENTS.md custom instructions for projects that prefer that. Offer the MCP-server
path only if rod-cli exposes an MCP stdio server (it does stdio JSON — confirm it speaks MCP before
documenting `[mcp_servers]`; otherwise the skill/AGENTS.md "shell out to `rod`" path is the fit).

## Install mechanism

### Installing Codex CLI
- **finding:** Three official install paths. Standalone installer:
  `curl -fsSL https://chatgpt.com/codex/install.sh | sh` (macOS/Linux);
  npm: `npm install -g @openai/codex`; Homebrew: `brew install --cask codex`.
  Windows: PowerShell `irm https://chatgpt.com/codex/install.ps1 | iex` (or WSL2).
- **source:** github.com/openai/codex README + developers.openai.com/codex/cli · **quality:** HIGH official · **as of:** 2026-06
- **bears on:** the "prerequisites" section of the INSTALL doc for Codex users.

### Authentication
- **finding:** Two auth pathways: (1) **ChatGPT account sign-in** (recommended) — first run of
  `codex` prompts to sign in; works with ChatGPT Plus/Pro/Business/Edu/Enterprise. (2) **API key** via
  `OPENAI_API_KEY` env var. Auth is required before the agent runs.
- **source:** developers.openai.com/codex/cli; github.com/openai/codex README · **quality:** HIGH official · **as of:** 2026-06
- **bears on:** Gotchas — user must be signed in / have a key before rod-cli is reachable.

## Tool/skill registration (exact path + format)

Codex CLI gives rod-cli three viable registration surfaces. In order of fit:

### A) Agent Skill (SKILL.md) — best fit, rod-cli already ships as a skill
- **finding:** A skill is a directory with a required `SKILL.md` (plus optional `scripts/`,
  `references/`, `assets/`, `agents/openai.yaml`). Required frontmatter is exactly:
  ```yaml
  ---
  name: skill-name
  description: Explain exactly when this skill should and should not trigger.
  ---

  Skill instructions for Codex to follow.
  ```
  The `description` drives implicit invocation. Skills use progressive disclosure (Codex sees
  name+description+path first, loads full SKILL.md only when chosen). Invocation: `/skills` or `$`
  mention in CLI/IDE (explicit), name-in-prompt (explicit), or task-matches-description (implicit).
  Skills may include scripts and "external tooling" — i.e. the SKILL.md body instructs the agent to
  run the `rod` binary directly. The SKILL.md format is an open cross-tool standard (Codex, Claude
  Code, Gemini CLI, Cursor, etc.), so rod-cli's existing skill is reusable.
- **finding (storage paths, per official docs):** Codex scans, in priority order:
  `$CWD/.agents/skills` → `$REPO_ROOT/.agents/skills` (REPO) → `$HOME/.agents/skills` (USER) →
  `/etc/codex/skills` (ADMIN) → bundled (SYSTEM).
- **source:** developers.openai.com/codex/skills · **quality:** HIGH official · **as of:** 2026-06
- **CONFLICT / version-sensitivity:** Multiple third-party 2026 guides (agensi.io, codegateway,
  danielvaughan KB) instead state skills live at `~/.codex/skills/` (personal) and `.codex/skills/`
  (project). The official docs page says `.agents/skills` / `$HOME/.agents/skills`. These disagree.
  Trust the official `.agents/skills` paths, but **verify on the target Codex version** before
  writing the path into INSTALL — the skills feature launched Dec 2025 and the path may have moved
  from `~/.codex/skills` to the open-standard `.agents/skills` layout. See Unknowns.
- **bears on:** the primary "register rod-cli skill with Codex" instruction.

### B) AGENTS.md custom instructions — document rod-cli usage for the agent
- **finding:** AGENTS.md is Codex's project/global instruction file. Discovery + precedence
  (official): (1) Global — in `~/.codex` (or `$CODEX_HOME`), Codex reads `AGENTS.override.md` if
  present, else `AGENTS.md`. (2) Project — starting at the Git root, walks down to CWD; in each dir
  checks `AGENTS.override.md`, then `AGENTS.md`, then configured fallback names; at most one file per
  dir. Files concatenate root→CWD; **deeper (closer to CWD) files weigh more** (appear later in
  prompt). `AGENTS.override.md` *replaces* `AGENTS.md` at the same level (not additive). Combined
  size capped at **32 KiB** (default; configurable via `project_doc_max_bytes`).
- **source:** developers.openai.com/codex/guides/agents-md; developers.openai.com/codex/config-reference · **quality:** HIGH official · **as of:** 2026-06
- **bears on:** alternative/secondary path — tell the agent "use `rod ...` for browser automation"
  in a project AGENTS.md, or globally in `~/.codex/AGENTS.md`.

### C) MCP server — only if rod-cli speaks MCP (see MCP path below)

## MCP path

- **finding:** Codex CLI supports MCP servers (stdio and streamable-HTTP). Config lives in
  `config.toml` — user-level `~/.codex/config.toml`, or project-scoped `.codex/config.toml` (loaded
  only for **trusted** projects). A stdio server is registered as:
  ```toml
  [mcp_servers.rod]
  command = "rod"
  args = ["mcp"]            # whatever subcommand starts rod's MCP stdio server
  env = { KEY = "value" }   # optional inline env
  env_vars = ["LOCAL_TOKEN"] # optional: env vars to pass through
  # optional knobs:
  # cwd = "/path"
  # enabled = true
  # startup_timeout_sec = 10   (default 10s)
  # tool_timeout_sec = 60      (default 60s)
  # required = false
  ```
  `command` is required; `args`, `env`, `env_vars`, `cwd`, `enabled`, `startup_timeout_sec`,
  `tool_timeout_sec`, `required` are optional. CLI equivalent:
  `codex mcp add <name> --env VAR=VALUE -- <command> <args...>`
  (official example: `codex mcp add context7 -- npx -y @upstash/context7-mcp`).
- **source:** developers.openai.com/codex/mcp; developers.openai.com/codex/config-reference · **quality:** HIGH official · **as of:** 2026-06
- **bears on:** the MCP registration path — **gated on rod-cli exposing an MCP-protocol stdio
  server**. rod-cli's memory says it does "token-efficient browser automation via stdio"; confirm
  that stdio is MCP-framed (JSON-RPC/MCP) and not a bespoke protocol before documenting `[mcp_servers]`.
  If rod's stdio is NOT MCP, do NOT document this path — use the Skill/AGENTS.md "run the `rod` binary"
  path instead.

## Gotchas

- **Auth before use:** the agent cannot run until the user has signed in (ChatGPT) or set
  `OPENAI_API_KEY`. State this as a prereq. (HIGH, official.)
- **Project-scoped config requires trust:** `.codex/config.toml` (and project MCP servers) load only
  for *trusted* projects; an untrusted project silently ignores them. (HIGH, official.)
- **MCP only fits if rod speaks MCP:** "stdio" ≠ "MCP". Verify rod's protocol before recommending
  the `[mcp_servers]` block. (Determined by rod's own code — verify in-repo, not external.)
- **AGENTS.md 32 KiB cap + precedence:** large skill docs concatenated across nested AGENTS.md can be
  truncated at 32 KiB; deeper files override shallower. Keep rod instructions concise and place at the
  right level. (HIGH, official.)
- **Skills path uncertainty:** official `.agents/skills` vs community `~/.codex/skills` — pin to the
  installed Codex version before hard-coding the path (see Unknowns). (MEDIUM.)
- **No TLS-spoofing constraint (project-internal):** unrelated to Codex, but per project memory rod-cli
  uses real Chrome TLS — no bearing on Codex registration.

## Unknowns (stated, not papered over)

- **Exact current skills storage path.** Official docs (developers.openai.com/codex/skills) list
  `$CWD/.agents/skills`, `$HOME/.agents/skills`, `/etc/codex/skills`; multiple 2026 third-party guides
  list `~/.codex/skills/` and `.codex/skills/`. Settle by running `codex --version` on the target
  install and checking that version's docs, or by testing where Codex actually discovers a placed
  SKILL.md. Do not hard-code a path in INSTALL until pinned.
- **Whether rod-cli's stdio interface is MCP-framed.** Not an external question — verify against
  rod-cli's own code (SMTC/Read) before documenting the `[mcp_servers]` path.
- **Exact `rod` subcommand to launch an MCP server (if any).** The `args=["mcp"]` above is a
  placeholder; confirm rod's actual MCP entrypoint in-repo.

## Sources (URLs)

- https://developers.openai.com/codex/mcp — Model Context Protocol (config + `codex mcp add`)
- https://developers.openai.com/codex/config-reference — full config.toml reference (`[mcp_servers]`, AGENTS.md settings, timeouts)
- https://developers.openai.com/codex/guides/agents-md — AGENTS.md discovery/precedence/size
- https://developers.openai.com/codex/skills — Agent Skills (SKILL.md format, paths, invocation)
- https://developers.openai.com/codex/cli — Codex CLI install + auth
- https://github.com/openai/codex — README (install commands, npm `@openai/codex`, brew cask)
- https://github.com/openai/codex/blob/main/docs/config.md — docs index (points to developers.openai.com config pages)
