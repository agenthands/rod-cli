---
author: project-researcher
responsible: architect
emitted_at: 2026-06-26T00:00:00Z
phase: null
artifact_kind: research-note
parent_artifacts: []
status: draft
---

# Research — Gemini CLI: registering rod-cli so the agent can invoke it

**Answer (confidence: HIGH):** Gemini CLI (`@google/gemini-cli`, Google's open-source terminal
agent) invokes external CLIs through the **Model Context Protocol (MCP)**. rod-cli ships a stdio MCP
server (`rod-cli` over stdin/stdout), so registration is an **`mcpServers` block** added either via
the `gemini mcp add` command or by hand-editing `settings.json` (user `~/.gemini/settings.json` or
project `.gemini/settings.json`). Agent-facing usage guidance goes in a **`GEMINI.md`** context file.
There is no skill/plugin concept analogous to Claude Code skills; the packaging primitives are MCP
servers + GEMINI.md, optionally bundled as an **extension** (`gemini-extension.json`).

**Recommendation:** Document two registration paths for v1.8 — (a) the one-line `gemini mcp add`
command, and (b) the equivalent hand-edited `mcpServers` JSON for `~/.gemini/settings.json` (global)
or `.gemini/settings.json` (per-project). Document rod-cli usage instructions in a `GEMINI.md` placed
at project root (or `~/.gemini/GEMINI.md` for global). Note: the repo currently has **no** `.agents/`
or `GEMINI.md` file (the task brief's claim that `.agents/GEMINI.md` exists is incorrect as of HEAD).

## Install mechanism
Gemini CLI is an npm-distributed Node binary. Prerequisite: **Node.js >= 20.0.0** (`engines` field
in package.json, HIGH).

```bash
npx @google/gemini-cli          # run without installing
npm install -g @google/gemini-cli   # global install -> `gemini`
brew install gemini-cli         # Homebrew (macOS/Linux)
sudo port install gemini-cli    # MacPorts
```

Auth (first run): Google account sign-in (browser OAuth, free tier 60 req/min, 1000/day), OR
`export GEMINI_API_KEY="..."` (key from aistudio.google.com/apikey), OR Vertex AI
(`GOOGLE_API_KEY` + `GOOGLE_GENAI_USE_VERTEXAI=true`). Auth is for the *model*, not for rod-cli; it
is a prerequisite to running the agent at all.

## Tool/skill registration (exact path + format)
Canonical mechanism = **MCP server registration**. No Claude-style "skills" exist. Two equivalent ways:

**(a) CLI command** (writes the JSON for you):
```bash
gemini mcp add [options] <name> <command> [args...]
# flags: -s/--scope (user|project, default project), -t/--transport (stdio|sse|http, default stdio),
#        -e/--env KEY=value, --timeout <ms>, --trust
# example from docs:
gemini mcp add -e API_KEY=123 -e DEBUG=true my-stdio-server /path/to/server arg1 arg2 arg3
```
Other subcommands: `gemini mcp list` (names + connection status), `gemini mcp remove <name> [-s scope]`.

**(b) Hand-edit `settings.json`** — add to the `mcpServers` object. For rod-cli (adjust the command
and the rod-cli subcommand/flags that start its stdio MCP server to match its actual invocation):
```json
{
  "mcpServers": {
    "rod-cli": {
      "command": "rod-cli",
      "args": ["mcp"],
      "timeout": 30000,
      "trust": false
    }
  }
}
```

## MCP path
Stdio MCP server config shape (HIGH — official `docs/tools/mcp-server.md`):
```json
{
  "mcpServers": {
    "serverName": {
      "command": "path/to/server",
      "args": ["--arg1", "value1"],
      "env": { "API_KEY": "$MY_API_TOKEN" },
      "cwd": "./server-directory",
      "timeout": 30000,
      "trust": false
    }
  }
}
```
Field semantics: `command` (required, executable for stdio), `args` (optional array), `env` (optional;
values support `$VAR_NAME` expansion), `cwd` (optional working dir), `timeout` (ms, default 600000),
`trust` (bool, default false — `true` bypasses per-tool-call confirmation prompts for that server).

**Scopes / file locations (HIGH):**
- **User / global:** `~/.gemini/settings.json` — applies to all projects.
- **Project:** `.gemini/settings.json` (in the project dir) — applies to that project.
- `gemini mcp add` defaults to **project** scope; pass `-s user` for global.
- (A system-level settings file also exists for org policy; not needed for rod-cli onboarding.)

OAuth (`/mcp auth [serverName]`) applies only to **remote** SSE/HTTP servers — **not relevant** to
rod-cli's stdio transport.

## Context file (GEMINI.md)
Gemini CLI's instruction/context file is **`GEMINI.md`** (HIGH — official `docs/cli/gemini-md.md`).
Contents are concatenated and injected into the model's context every prompt — this is where rod-cli
usage guidance for the agent belongs.

Loading hierarchy / precedence (in order):
1. **Global:** `~/.gemini/GEMINI.md` (default instructions for all projects).
2. **Project + ancestors:** `GEMINI.md` in the working dir and each parent up to the project root
   (the `.git` boundary).
3. **Subdirectory / just-in-time:** `GEMINI.md` files in subdirectories, loaded when tools touch
   those files (component-specific instructions).

- Override the filename via `settings.json` `context.fileName` — accepts a string or a list, e.g.
  `{ "context": { "fileName": ["AGENTS.md", "CONTEXT.md", "GEMINI.md"] } }`. (This is how you'd make
  Gemini also read a shared `AGENTS.md`.)
- Import/modularize with `@file.md` or `@./path/to/file.md` (relative or absolute paths).
- Inspect/refresh with `/memory show` and `/memory reload`.

**For rod-cli:** put usage docs in `GEMINI.md` at the repo root (committed) and/or
`~/.gemini/GEMINI.md` (machine-global). NOTE: the rod-cli repo does **not** currently contain a
`GEMINI.md` or `.agents/` directory (verified at HEAD).

## Custom commands / extensions (optional packaging)
- **Extensions** bundle MCP servers + a context file as one installable unit. Loaded from
  `~/.gemini/extensions/` (and `<workspace>/.gemini/extensions/`); each extension dir has a
  **`gemini-extension.json`** manifest (MED — corroborated by official reference docs + community,
  manifest body not fetched directly from the raw doc). Example:
  ```json
  {
    "name": "my-extension",
    "version": "1.0.0",
    "mcpServers": {
      "my-server": {
        "command": "node",
        "args": ["${extensionPath}/my-server.js"],
        "cwd": "${extensionPath}"
      }
    },
    "contextFileName": "GEMINI.md",
    "excludeTools": ["run_shell_command"]
  }
  ```
  `mcpServers` here is loaded on startup exactly like the `settings.json` block; use `${extensionPath}`
  for portability. `contextFileName` names the bundled context file (defaults to `GEMINI.md` if
  present). An extension would let rod-cli ship as a single `gemini extensions install <repo>` unit —
  heavier than the plain `mcpServers` block, only worth it if distributing a packaged install.
- **Custom commands** (TOML under `~/.gemini/commands/` or `.gemini/commands/`) are reusable prompt
  macros, not a tool-invocation mechanism. Not the right primitive for exposing rod-cli — MCP is.

## Gotchas
- **Node >= 20 required** — older Node fails to install/run.
- **Scope confusion:** `gemini mcp add` defaults to **project** scope (writes `.gemini/settings.json`
  in CWD), NOT global. Use `-s user` for `~/.gemini/settings.json`.
- **rod-cli must be on `PATH`** (or give an absolute `command` path) for the stdio launch to work.
- **`trust: false` (default)** means the user is prompted to confirm each rod-cli tool call; set
  `trust: true` to skip prompts (document the tradeoff, don't default users to it silently).
- **The exact rod-cli subcommand** that starts its MCP stdio server (the `args` value above is a
  placeholder `["mcp"]`) must be filled from rod-cli's real CLI — confirm against rod-cli's own
  command surface before publishing the doc.
- Auth to a Google model is a hard prerequisite before any MCP server is reachable; first-run is an
  interactive browser flow unless `GEMINI_API_KEY` is set.

## Unknowns
- **Node version** is HIGH (package.json engines `>=20.0.0`); the README "system requirements" page
  was not fetched, but engines is authoritative.
- The **extension manifest** (`gemini-extension.json`) example is MED: official reference doc URL
  (`docs/extensions/reference.md`) returned 404 on raw fetch; the manifest shape is corroborated by
  geminicli.com docs + community tutorials but not verified against a fetched official raw file. If
  v1.8 ships an extension path, fetch `https://github.com/google-gemini/gemini-cli/blob/main/docs/extensions/reference.md`
  to confirm current field names.
- rod-cli's exact MCP-start invocation is a **codebase** question (out of this dimension's scope) —
  the architect should confirm it from rod-cli's command surface, not from these docs.

## Sources (URLs)
- MCP server config + `gemini mcp add` (official): https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/mcp-server.md — HIGH, fetched 2026-06-26
- GEMINI.md context file + hierarchy + `context.fileName` (official): https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/gemini-md.md — HIGH, fetched 2026-06-26
- Configuration / settings.json scopes (official): https://github.com/google-gemini/gemini-cli/blob/main/docs/get-started/configuration.md — referenced (raw 404 on some paths); scope facts corroborated by mcp-server.md (HIGH) + geminicli.com docs
- Extension manifest reference (official, not raw-fetched): https://github.com/google-gemini/gemini-cli/blob/main/docs/extensions/reference.md — MED; corroborated: https://geminicli.com/docs/extensions/reference/
- Install + auth + Node engines (official): https://github.com/google-gemini/gemini-cli/blob/main/README.md and https://github.com/google-gemini/gemini-cli/blob/main/package.json — HIGH, fetched 2026-06-26
- GEMINI.md hierarchy corroboration: https://geminicli.com/docs/cli/gemini-md/ — MED
