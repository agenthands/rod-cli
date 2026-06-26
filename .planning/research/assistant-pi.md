---
author: project-researcher
responsible: architect
emitted_at: 2026-06-26T00:00:00Z
phase: null
artifact_kind: research_note
parent_artifacts: []
status: draft
dimension: pi
verify_basis: live pi.dev/docs/latest + earendil-works/pi README (fetched 2026-06-26)
---

# Research — Pi coding agent onboarding for rod-cli (skill registration)

**Answer (HIGH confidence):** Pi is a minimal terminal harness whose only built-in
tools are `read`, `write`, `edit`, `bash`. The idiomatic way to let the agent drive an
external CLI like `rod` is an **Agent Skill** (a `SKILL.md` instructing the agent to
shell out via the bash tool). Pi implements the cross-tool **Agent Skills / `SKILL.md`
standard** (with minor intentional leniency). **Pi has no native MCP support** — by
design ("No MCP"); MCP would require writing a custom TypeScript extension.

**Recommendation for v1.8 INSTALL docs:** document the **Skill** path as primary —
install the `rod` binary, then drop a `SKILL.md` into a discovered skills directory
(`~/.pi/agent/skills/<name>/SKILL.md` global, or `.pi/skills/<name>/SKILL.md` project).
Mention the **Extension** path only as the heavier alternative (a first-class
LLM-callable tool via `pi.registerTool` + `pi.exec`). Do **not** document an MCP path —
state explicitly Pi does not support MCP.

## Install mechanism

- **Best fit for "agent shells out to `rod`": a Skill.** Pi ships four default tools
  (`read`, `write`, `edit`, `bash`); a Skill is "self-contained capability package the
  agent loads on-demand" whose markdown instructs the agent to run shell commands (the
  docs show e.g. `./search.js "query"` executed via bash). No tool registration needed
  — the agent invokes `rod ...` through the existing bash tool. (source: skills doc,
  packages doc · HIGH)
- **Heavier alternative: an Extension** — a TypeScript module that registers a
  first-class LLM-callable tool with `pi.registerTool(...)` and runs the binary via
  `pi.exec("rod", [...], { signal })`. Gives custom rendering/cancellation but requires
  shipping TS and project trust. Overkill for simple CLI invocation. (source:
  extensions doc · HIGH)
- **Bundling/distribution: a Pi Package** can wrap skills/extensions/prompts/themes and
  is installed with `pi install npm:@scope/pkg@ver` / `pi install git:host/repo@ref` /
  `pi install ./path`. Optional if we just ship a `SKILL.md` for users to copy. (source:
  packages doc · HIGH)

## Skill/extension registration (exact path + format)

### Skill (recommended) — `SKILL.md`, Agent Skills standard
- **Filename:** `SKILL.md` (required) inside a skill directory. Optional sibling dirs:
  `scripts/`, `references/`, `assets/`.
- **Discovery locations** (verified from live docs):
  - User/global: `~/.pi/agent/skills/` and `~/.agents/skills/`
  - Project (after the project is *trusted*): `.pi/skills/` and `.agents/skills/` in cwd
    and ancestor dirs
  - Also: package `skills/` dirs or `pi.skills` in `package.json`; a `skills` array in
    settings; CLI `--skill <path>` flags.
- **Frontmatter schema** (YAML):
  ```yaml
  ---
  name: my-skill            # REQUIRED; 1-64 chars, lowercase letters/numbers/hyphens,
                            #   no leading/trailing/consecutive hyphens
  description: What this skill does and when to use it. Be specific.   # REQUIRED
  license: ...              # optional
  compatibility: ...        # optional
  metadata: ...             # optional
  allowed-tools: ...        # optional (experimental)
  disable-model-invocation: ...   # optional
  ---
  ```
- **Standard:** Pi *implements the Agent Skills standard* (the same `SKILL.md` convention
  as Claude Code et al.), diverging only in leniency — e.g. it permits a skill `name`
  differing from its parent directory (warns, doesn't reject). So a generic
  cross-tool `SKILL.md` works in Pi. (source: skills doc · HIGH)
- **How it invokes the CLI:** narrative markdown instructions telling the agent to run
  the binary via bash (`rod <subcommand> ...`); the `allowed-tools` field exists but is
  experimental — the primary mechanism is instruction + the bash tool. (source: skills
  doc · HIGH)

### Extension (alternative) — TypeScript module
- **Locations:** `~/.pi/agent/extensions/*.ts` or `.../<name>/index.ts` (global);
  `.pi/extensions/*.ts` or `.../<name>/index.ts` (project, after trust). Extra paths via
  `settings.json` `extensions` array.
- **Shape:** default-export factory `export default function (pi: ExtensionAPI) {...}`
  (sync or async); register a tool with `pi.registerTool({ name, label, description,
  parameters, async execute(...) })`; run the binary with
  `await pi.exec("rod", ["..."], { signal })` (returns `stdout/stderr/code/killed`).
  Import type from `@earendil-works/pi-coding-agent`. (source: extensions doc · HIGH)

## MCP path (NOT supported)

**Pi does not support MCP natively — by deliberate design.** README, verbatim:
> "No MCP. Build CLI tools with READMEs (see Skills), or build an extension that adds
> MCP support."

MCP is listed alongside sub-agents, plan mode, permission popups, and built-in to-dos as
features Pi intentionally omits. There is **no stdio-MCP config file or settings key** to
document. The only routes to MCP are (a) writing a custom extension that itself speaks
MCP, or (b) — preferred and what we should document — skip MCP entirely and use a Skill
that drives the `rod` CLI through bash. (source: README + packages doc · HIGH)

## Gotchas

- **`--ignore-scripts` on install is the documented norm:**
  `npm install -g --ignore-scripts @earendil-works/pi-coding-agent` ("Pi does not require
  install scripts for normal npm installs"). curl alternative:
  `curl -fsSL https://pi.dev/install.sh | sh`. (source: quickstart · HIGH)
- **Project trust gate:** project-level skills (`.pi/skills/`) and extensions
  (`.pi/extensions/`) load **only after the project is explicitly trusted**. Global
  skills under `~/.pi/agent/skills/` avoid this. Worth calling out in INSTALL docs.
  (source: skills + extensions docs · HIGH)
- **Auth:** `/login` (subscriptions: Claude Pro/Max, ChatGPT Plus/Pro, GitHub Copilot),
  or env var (`export ANTHROPIC_API_KEY=sk-ant-...`) before launching, or `/login` with
  an API-key provider to store it in `~/.pi/agent/auth.json`. Pi supports 30+ providers.
  (source: quickstart + providers/packages docs · HIGH)
- **Extensions run with full system permissions / arbitrary code** — if we ever ship an
  extension, that's a trust note for users. (source: extensions doc · HIGH)
- **`name` must be a valid slug** (1-64, lowercase, hyphen rules) or Pi warns. (HIGH)

## Unknowns (could not verify from live docs)

- **Node.js version prerequisite:** quickstart does **not** state a minimum Node version.
  Would need the package.json `engines` field on npm to confirm — flag as unverified.
- **`allowed-tools` semantics in a Skill:** docs label it "experimental"; exact behavior
  (does it restrict the agent to listed tools? syntax?) is not specified. Do not rely on
  it in our SKILL.md; depend on plain bash invocation instead.
- **Whether `disable-model-invocation` / `metadata` / `compatibility` have a defined
  schema** beyond being optional keys — not detailed in the live skills page.
- **Exact precedence** if the same skill name exists in multiple discovery locations —
  not documented in pages fetched.

## Sources (URLs)

- https://pi.dev/docs/latest (docs index / navigation) — HIGH official
- https://pi.dev/docs/latest/skills — HIGH official (skill format, discovery, standard)
- https://pi.dev/docs/latest/extensions — HIGH official (TS extension, registerTool, exec)
- https://pi.dev/docs/latest/packages — HIGH official (package.json `pi` key, `pi install`)
- https://pi.dev/docs/latest/quickstart — HIGH official (install, auth, --ignore-scripts)
- https://github.com/earendil-works/pi/blob/main/packages/coding-agent/README.md
  (raw: raw.githubusercontent.com/.../README.md) — HIGH official ("No MCP" verbatim)
- https://github.com/earendil-works/pi/tree/main/packages/coding-agent/docs — HIGH (doc tree)
