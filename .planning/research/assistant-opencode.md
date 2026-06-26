---
author: project-researcher
responsible: architect
emitted_at: 2026-06-26T00:00:00Z
phase: null
artifact_kind: research_note
parent_artifacts: []
status: draft
dimension: opencode
verify_method: WebFetch/WebSearch against opencode.ai/docs (2026)
---

# Research — opencode onboarding for rod-cli (v1.8 INSTALL + skill docs)

**Answer (1-3 sentences):** opencode has FOUR independent ways to give an agent access
to an external CLI like rod-cli: (a) a **native agent-skills mechanism** (`SKILL.md`,
discovered from `.opencode/skills/`, `~/.config/opencode/skills/`, and Claude-compatible
`.claude/skills/` / `~/.claude/skills/`) — this is the direct fit since rod-cli ships AS
an agent skill; (b) an **MCP server** block in `opencode.json` (`type: "local"` for
stdio); (c) a **custom tool** (`.ts`/`.js` in `.opencode/tools/`) that wraps the CLI; and
(d) **AGENTS.md** instructions. All confirmed against current (2026) official docs.
**Confidence: HIGH** — every claim below is from opencode.ai/docs fetched 2026-06-26.

**Recommendation:** Document rod-cli for opencode primarily via the **native skills
mechanism** — because opencode natively reads `.claude/skills/<name>/SKILL.md` and
`~/.claude/skills/<name>/SKILL.md`, an existing rod-cli Claude-Code skill works in
opencode with **zero extra config**. Provide AGENTS.md as the lightweight always-on
alternative, and an MCP/custom-tool block only if rod-cli should be exposed as a callable
tool rather than skill instructions.

## Install mechanism
- **Install methods (HIGH, opencode.ai/docs):**
  - curl: `curl -fsSL https://opencode.ai/install | bash`
  - npm: `npm install -g opencode-ai` (also Bun/pnpm/Yarn: `bun install -g opencode-ai`)
  - Homebrew: `brew install anomalyco/tap/opencode`
  - Arch: `sudo pacman -S opencode`; Windows: `choco install opencode` / `scoop install opencode`
- **Auth:** run `opencode`, then `/connect` in the TUI → choose provider, visit auth page,
  paste API key. (opencode is a model-agnostic terminal agent; you bring your own model
  provider/key — Anthropic, OpenAI, etc.)
- **Project init:** `opencode` then `/init` analyzes the repo and writes an `AGENTS.md`.

## Tool/skill registration (exact path + format)
opencode has a **native agent-skills feature** (this is the canonical fit for rod-cli).
Skills are discovered (HIGH, opencode.ai/docs/skills) from, walking up from cwd to the git
worktree root:
- Project: `.opencode/skills/<name>/SKILL.md`
- Global: `~/.config/opencode/skills/<name>/SKILL.md`
- **Claude-compatible project: `.claude/skills/<name>/SKILL.md`**
- **Claude-compatible global: `~/.claude/skills/<name>/SKILL.md`**
- Agent-compatible: `.agents/skills/<name>/SKILL.md` and `~/.agents/skills/<name>/SKILL.md`

`SKILL.md` YAML frontmatter — **required:** `name` (1-64 chars, lowercase alphanumeric +
hyphens), `description` (1-1024 chars). **Optional:** `license`, `compatibility`,
`metadata` (string→string map).

Invocation: a native `skill` tool lists available skills by description; the agent loads
full content on demand, e.g. `skill({ name: "rod-cli" })`. Access is governed by
permission rules (`allow` / `deny` / `ask`, wildcard patterns). Claude-Code skill support
can be disabled via env var `OPENCODE_DISABLE_CLAUDE_CODE_SKILLS`.

**Implication for rod-cli:** an existing Claude-Code-format skill at
`.claude/skills/rod-cli/SKILL.md` (or `~/.claude/skills/rod-cli/SKILL.md`) is read by
opencode with NO opencode-specific config. The opencode-native path is identical content
under `.opencode/skills/rod-cli/SKILL.md`.

## MCP path
Config file: **`opencode.json`** (or `opencode.jsonc`). Scopes (HIGH, opencode.ai/docs/config):
- Global: `~/.config/opencode/opencode.json`
- Project: `opencode.json` in project root (configs are **merged**, project overrides global)

Exact stdio (local) MCP block (HIGH, opencode.ai/docs/mcp-servers):
```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "rod-cli": {
      "type": "local",
      "command": ["rod-cli", "mcp"],
      "enabled": true,
      "environment": {
        "MY_ENV_VAR": "value"
      }
    }
  }
}
```
Keys: `type` ("local" = stdio, required), `command` (array, required — binary + args),
`environment` (object, optional), `enabled` (bool, optional), `cwd` (optional, relative to
workspace), `timeout` (ms, optional, default 5000). Remote MCP uses `type: "remote"` with a
`url`. **Note:** this only applies if rod-cli exposes an MCP stdio server interface — the
task describes rod-cli as "token-efficient browser automation via stdio," so confirm
whether its stdio interface speaks the MCP protocol or is a bespoke CLI protocol (see
Unknowns).

## AGENTS.md / skills (where rod-cli usage is documented for the agent)
- **AGENTS.md** is opencode's primary instructions file (HIGH, opencode.ai/docs/rules).
  Precedence/lookup:
  1. Local `AGENTS.md` (traversing up from cwd); fallback `CLAUDE.md` if no AGENTS.md
  2. Global `~/.config/opencode/AGENTS.md`; fallback `~/.claude/CLAUDE.md` (unless disabled)
- The `instructions` key in `opencode.json` can pull in extra files/globs/remote URLs:
  `"instructions": ["CONTRIBUTING.md", "docs/guidelines.md", ".cursor/rules/*.md"]`
- AGENTS.md is **always-on context** (loaded every session); a **skill** is **on-demand**
  (loaded only when the agent invokes it). For rod-cli, the skill mechanism is the more
  token-efficient home for usage instructions; a short AGENTS.md pointer can mention rod-cli
  exists.

## Custom tools / plugins
opencode lets you define a **custom tool** in TS/JS (HIGH, opencode.ai/docs/custom-tools):
- Location: `.opencode/tools/` (project) or `~/.config/opencode/tools/` (global). **Filename
  = tool name** (`database.ts` → `database` tool).
- API:
```typescript
import { tool } from "@opencode-ai/plugin"
export default tool({
  description: "Run rod-cli browser automation",
  args: { query: tool.schema.string().describe("...") },
  async execute(args) { /* invoke rod-cli via Bun shell */ },
})
```
- The execute body can shell out to any external binary ("write your tools in any
  language") — a valid way to wrap rod-cli as a first-class callable tool without MCP.
- A separate **plugin** system also exists (`plugin` config key, `@opencode-ai/plugin`).

## Gotchas
- **Skill format is shared with Claude Code** — keep `name` lowercase-alphanumeric-hyphen
  and within length limits or the skill is rejected. `OPENCODE_DISABLE_CLAUDE_CODE_SKILLS`
  turns off the `.claude/` fallbacks.
- **MCP only fits if rod-cli speaks MCP.** `type: "local"` launches the command and talks
  the MCP protocol over stdio; a non-MCP stdio CLI will NOT work as an `mcp` entry — use a
  custom tool or skill instead.
- **Config merge, not replace** — project `opencode.json` merges over global; only
  conflicting keys override.
- **Auth is provider-BYOK**, not an opencode account for model use (the `/connect` flow
  selects your model provider).
- Custom tools/plugins assume a **Bun/Node** runtime is present (the `@opencode-ai/plugin`
  import + Bun shell).

## Unknowns (state them)
- **Does rod-cli's stdio interface implement the MCP protocol?** Not determinable from
  opencode docs — it's a property of rod-cli. If NOT MCP, drop the `mcp` block from the
  opencode docs and lead with skills + a custom-tool wrapper. Settle by checking rod-cli's
  own stdio/server command.
- **Exact `SKILL.md` body conventions opencode honors beyond frontmatter** (e.g. bundled
  scripts/`allowed-tools`): docs specify frontmatter fields but not full Claude-Code skill
  feature parity. Settle with a quick test skill in `.opencode/skills/`.

## Sources (URLs, fetched 2026-06-26)
- Install/auth/init: https://opencode.ai/docs/ — HIGH (official)
- Config files/scopes/precedence: https://opencode.ai/docs/config/ — HIGH (official)
- MCP servers (local stdio config): https://opencode.ai/docs/mcp-servers/ — HIGH (official)
- AGENTS.md / rules / instructions key / Claude fallbacks: https://opencode.ai/docs/rules/ — HIGH (official)
- Agent skills (SKILL.md paths, frontmatter, invocation, permissions): https://opencode.ai/docs/skills/ — HIGH (official)
- Custom tools (.opencode/tools, tool() API): https://opencode.ai/docs/custom-tools/ — HIGH (official)
