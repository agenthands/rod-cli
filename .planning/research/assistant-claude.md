---
author: project-researcher
responsible: architect
emitted_at: 2026-06-26T00:00:00Z
phase: null
artifact_kind: research-note
parent_artifacts: []
status: draft
---

# Research — Claude Code onboarding for rod-cli (v1.8 INSTALL + skill-registration docs)

**Answer (1–3 sentences):** Claude Code's canonical mechanism for a tool like rod-cli is an **Agent Skill** — a directory containing a `SKILL.md` file dropped into `~/.claude/skills/<name>/` (personal) or `.claude/skills/<name>/` (project). rod-cli's existing `skills/rod-cli/SKILL.md` is already in the correct format; the user just copies/symlinks that directory into a skills location. There is ALSO a valid MCP path (`claude mcp add ... -- <binary>`), but that is a separate, optional integration, not how skills register. Confidence: **HIGH** — verified against current official docs at `code.claude.com` (June 2026).

**Recommendation:** Document the **Skill install** as the primary path: copy/symlink `skills/rod-cli/` → `~/.claude/skills/rod-cli/`. The rod-cli binary must also be installed separately on `PATH` (the skill is just instructions; it does not bundle the binary). Document the MCP path as a secondary/optional section for users who want rod-cli exposed as MCP tools.

## Install mechanism

Claude Code skills are **directory-based, file-installed** — there is no `claude skill install` command for a bare skill directory. The user installs by placing the skill directory at one of the recognized locations:

| Location | Path | Applies to |
| :--- | :--- | :--- |
| Personal | `~/.claude/skills/<skill-name>/SKILL.md` | All the user's projects |
| Project | `.claude/skills/<skill-name>/SKILL.md` | This project only (commit to VCS) |
| Plugin | `<plugin>/skills/<skill-name>/SKILL.md` | Where the plugin is enabled |
| Enterprise | via managed settings | All users in an org |

- **finding:** For rod-cli, the user installs the skill by copying or symlinking the shipped `skills/rod-cli/` directory to `~/.claude/skills/rod-cli/` (personal, recommended for a CLI tool used everywhere) or to a repo's `.claude/skills/rod-cli/` (project-scoped). The directory name (`rod-cli`) becomes the invocation command `/rod-cli`.
- **source:** https://code.claude.com/docs/en/skills ("Where skills live" table) · **quality:** HIGH (official) · **as of:** June 2026 docs
- **bears on:** The exact `cp`/`ln -s` command the INSTALL doc gives users.

- **finding:** The binary is separate. A skill is markdown instructions only; it does NOT install or bundle the `rod-cli` executable. The INSTALL doc must cover installing the `rod-cli` binary onto `PATH` (go install / release download) AS WELL AS placing the skill directory. The SKILL.md body references `rod-cli install` (Chromium) — that is a runtime step the agent runs, distinct from skill registration.
- **source:** skills/rod-cli/SKILL.md (existing) + skills doc (skill = SKILL.md + supporting files, no binary delivery) · **quality:** HIGH · **as of:** current repo HEAD
- **bears on:** INSTALL doc must have two install steps: (1) binary on PATH, (2) skill directory in place.

- **finding:** Plugin/marketplace is an OPTIONAL upgrade, not required. Dropping a `.claude-plugin/plugin.json` into the skill folder makes it load as a plugin named `<name>@skills-dir` that can also bundle agents/hooks/MCP servers. Not needed for rod-cli's simple skill; only relevant if rod-cli later wants to bundle its MCP server config alongside the skill.
- **source:** https://code.claude.com/docs/en/skills (Note on `.claude-plugin/plugin.json`) · **quality:** HIGH · **as of:** June 2026

## Skill registration (exact path + format)

**Exact path:** `~/.claude/skills/rod-cli/SKILL.md` (personal) or `<repo>/.claude/skills/rod-cli/SKILL.md` (project). Entry point file MUST be named `SKILL.md`. Supporting files (rod-cli already ships `references/*.md`) live in the same directory and are fine.

**Frontmatter schema (current).** YAML between `---` markers. **All fields are optional; only `description` is recommended.** Verified field table:

| Field | Required | Notes |
| :--- | :--- | :--- |
| `name` | No | Display name in listings. **Defaults to the directory name.** Does NOT set the `/command` name except for a plugin-root SKILL.md. |
| `description` | Recommended | What it does + when to use it. Claude matches on this for auto-invocation. Combined `description`+`when_to_use` is **truncated at 1,536 characters** in the listing. |
| `when_to_use` | No | Extra trigger phrases; appended to description, counts toward the 1,536-char cap. |
| `allowed-tools` | No | Tools pre-approved (no permission prompt) while skill active. Space- or comma-separated string, or YAML list. Does NOT restrict the tool pool. |
| `disallowed-tools` | No | Tools removed from pool while active. |
| `disable-model-invocation` | No | `true` = only the user can invoke via `/name` (Claude won't auto-trigger). Default `false`. |
| `user-invocable` | No | `false` = hide from `/` menu (Claude-only). Default `true`. |
| `argument-hint`, `arguments` | No | Autocomplete hint / named positional args for `$name` substitution. |
| `model`, `effort`, `context`, `agent`, `hooks`, `paths`, `shell` | No | Advanced (model/effort override, `context: fork` subagent execution, path-glob gating, etc.). |

- **finding:** rod-cli's existing `skills/rod-cli/SKILL.md` frontmatter is VALID and current:
  ```yaml
  ---
  name: rod-cli
  description: "Token-efficient, native web automation CLI for LLMs using Go Rod"
  ---
  ```
  This parses and works. **Improvement (not a fix):** the `description` is a capability statement but lacks "when to use" trigger phrases. Claude auto-invocation matches against `description`; adding when-to-use keywords (e.g. "Use when the user needs to automate a browser, scrape a page, fill a form, take a screenshot, or test a web UI") materially improves auto-trigger accuracy. Optional but recommended.
- **source:** skills/rod-cli/SKILL.md (repo) vs https://code.claude.com/docs/en/skills (Frontmatter reference + Troubleshooting "Skill not triggering") · **quality:** HIGH · **as of:** June 2026
- **bears on:** Whether the doc tells users to enrich the description; whether registration "just works" with the shipped file (it does).

- **finding:** `allowed-tools` is the field that pre-approves tool calls. If rod-cli's skill wants the agent to run the binary without a per-call permission prompt, add e.g. `allowed-tools: Bash(rod-cli *)`. For project-scoped skills this only takes effect after the user accepts the workspace trust dialog.
- **source:** https://code.claude.com/docs/en/skills ("Pre-approve tools for a skill") · **quality:** HIGH · **as of:** June 2026
- **bears on:** Optional doc tip to reduce permission prompts when driving rod-cli.

## MCP path (optional/secondary)

rod-cli historically ran as an MCP server; that path is still fully supported and independent of the skill. Current mechanism:

- **finding:** Add a local stdio server with:
  ```
  claude mcp add [options] <name> -- <command> [args...]
  # e.g.
  claude mcp add --scope user rod-cli -- rod-cli <mcp-serve-subcommand>
  ```
  The `--` separates Claude's flags from the server command; everything after `--` is passed to the binary untouched. `--env KEY=value` sets env vars. **Note:** the exact rod-cli subcommand that starts an MCP stdio server is NOT verified here (see Unknowns) — the SKILL.md shown documents a CLI/daemon model, not an MCP serve command.
- **source:** https://code.claude.com/docs/en/mcp (Option 3: Add a local stdio server) · **quality:** HIGH (for the `claude mcp add` syntax) · **as of:** June 2026

- **finding:** Scope flags and config file locations:
  - `--scope local` (default) — only you, current project. Stored in `~/.claude.json` (per-project section).
  - `--scope project` — shared with everyone via a **`.mcp.json`** file at the project root (commit to VCS). Project servers show as `⏸ Pending approval` until the user approves them interactively.
  - `--scope user` — available across all your projects. Stored in `~/.claude.json`.
  - JSON config (for hand-editing or `claude mcp add-json`) lives in `.mcp.json` (project) or `~/.claude.json` (user/local). A stdio entry shape:
    ```json
    { "mcpServers": { "rod-cli": { "command": "rod-cli", "args": ["..."], "env": {} } } }
    ```
- **source:** https://code.claude.com/docs/en/mcp (Tips: `--scope`; "JSON in `.mcp.json`, `~/.claude.json`") · **quality:** HIGH · **as of:** June 2026
- **bears on:** If docs include an MCP section, give both `claude mcp add` and the `.mcp.json` snippet, and state the per-project (`~/.claude.json`) vs shared (`.mcp.json`) distinction.

- **finding:** Management/verification commands: `claude mcp list`, `claude mcp get <name>`, `claude mcp remove <name>`, and `/mcp` inside a session to check status/tool count. Stdio servers are NOT auto-reconnected (local processes).
- **source:** https://code.claude.com/docs/en/mcp (Managing your servers) · **quality:** HIGH · **as of:** June 2026

## Gotchas (Claude Code specific)

- **Scope choice matters.** Personal `~/.claude/skills/` = everywhere (best for a general CLI tool like rod-cli). Project `.claude/skills/` = that repo only, and needs the **workspace trust dialog** accepted before `allowed-tools` permissions apply. Source: skills doc "Pre-approve tools for a skill" — HIGH.
- **Restart semantics (precise).** Editing/adding/removing a `SKILL.md` under an ALREADY-WATCHED skills dir (`~/.claude/skills/`, project `.claude/skills/`) takes effect **within the session, no restart** (live change detection). BUT **creating a top-level skills directory that did not exist when the session started requires restarting Claude Code** so the new dir gets watched. So: first-time install of `~/.claude/skills/` when none existed → restart `claude`. Source: skills doc "Live change detection" — HIGH.
- **Live detection covers SKILL.md text only.** rod-cli ships `references/*.md` supporting files — those are loaded on demand when SKILL.md references them; fine. If the skill folder is also a plugin, changes to `hooks/`/`.mcp.json`/`agents/` need `/reload-plugins`. Source: skills doc — HIGH.
- **Command name = directory name, not `name` frontmatter.** The skill is invoked `/rod-cli` because the directory is `rod-cli`. Renaming the dir renames the command. The `name:` field only sets the display label. Source: skills doc "How a skill gets its command name" — HIGH.
- **Description budget.** With many skills installed, descriptions get shortened to fit a context budget (1% of context window by default); put rod-cli's key use case FIRST in the description. `/doctor` reports shortening. Source: skills doc "Skill descriptions are cut short" — HIGH.
- **Malformed YAML fails soft.** If the frontmatter is malformed, the body still loads with empty metadata, so `/rod-cli` works but auto-invocation breaks (no description to match). Run `claude --debug` to see parse errors. Source: skills doc Troubleshooting — HIGH.
- **Binary not bundled.** Re-stated because it is the most likely user error: installing the skill does NOT install `rod-cli`. The binary must be on `PATH` independently, and `rod-cli install` must be run once to fetch Chromium. Source: repo SKILL.md + skills doc — HIGH.
- **Docs host moved.** Canonical Claude Code docs are now `https://code.claude.com/docs/en/...` (the old `docs.claude.com/en/docs/claude-code/...` 301-redirects there). Use the new host in any doc links. Source: observed 301 during this research — HIGH.

## Unknowns (stated, not papered over)

- **rod-cli's MCP serve subcommand is unverified.** The `claude mcp add ... -- rod-cli <X>` line needs the actual rod-cli subcommand/flags that start a stdio MCP server. The current `skills/rod-cli/SKILL.md` documents a CLI/daemon model (`open`, `goto`, `close`, `--session`), not an MCP serve command. Settle by: `rod-cli --help` / grepping the repo's command tree for an `mcp`/`serve` command, or confirming with the maintainer whether the MCP server still exists in v1.x. If rod-cli no longer ships an MCP server, drop the MCP section entirely and document Skill-only.
- **Whether rod-cli wants project-scope distribution.** Not a doc-blocking unknown — both scopes are documented above; the architect picks which to lead with (recommend personal `~/.claude/skills/` for a general tool).
- **Exact minimum Claude Code version for skills.** Skills/merged-commands are current as of June 2026 docs; a specific "minimum version that supports `.claude/skills/`" was not stated on the page (only the unrelated `/run`,`/verify` bundled-skill floor of v2.1.145 was given). Settle by checking the changelog if a version floor must be cited.

## Sources (URLs)
- https://code.claude.com/docs/en/skills — Agent Skills: install locations, SKILL.md frontmatter schema, live change detection, command naming, troubleshooting (HIGH, official, fetched June 2026)
- https://code.claude.com/docs/en/mcp — MCP: `claude mcp add` stdio syntax, `--scope` flags, `.mcp.json` / `~/.claude.json` config locations, management commands (HIGH, official, fetched June 2026)
- /home/john/go/src/github.com/agenthands/rod-cli/skills/rod-cli/SKILL.md — existing shipped skill (current repo HEAD)
- Old host https://docs.claude.com/en/docs/claude-code/{skills,mcp} 301-redirects to code.claude.com (observed during research)
