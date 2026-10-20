# Installing rod-cli for Coding Assistants

rod-cli is a native, token-efficient web automation CLI for LLMs. It communicates via standard I/O — no MCP server, no JSON-RPC overhead. This guide shows how to install rod-cli and onboard it into your coding assistant.

## Prerequisites: Install rod-cli

Every assistant needs `rod-cli` installed and on `PATH`. Choose one:

### Via Go (recommended)

```bash
go install github.com/agenthands/rod-cli@latest
```

### Via prebuilt binary

Download the latest binary from [GitHub Releases](https://github.com/agenthands/rod-cli/releases) and place it on your `PATH`.

### Install Chromium

rod-cli launches the system Chrome (or Chromium/Edge) when it finds one. Without one, run:

```bash
rod-cli install
```

This downloads Chromium to `~/.cache/rod/browser/`; sessions use it when no system Chrome is found. `browserBinPath` in `rod-cli.yaml` overrides both.

### Where browser profiles live

Each session's Chrome profile lives in `~/.cache/rod-cli/profiles/<id>` (the OS user cache dir; on disk, never `/tmp` or the working directory). Set `browserTempDir` in a `rod-cli.yaml` in the working directory (or pass `--config <dir>/rod-cli.yaml`; the file must be named `rod-cli.yaml`, and a missing or misnamed file is an error) to use another root; keys the file leaves out keep their defaults, and rod-cli reads that file but never creates one. Chrome's lock dir and temp files go to a per-session `chrome-*` dir in the temp dir (`/tmp` when `TMPDIR` is longer than 40 characters), never loose in `/tmp`. `rod-cli close`, the 15-minute idle timeout and SIGINT/SIGTERM/SIGHUP remove the profile and that dir. A profile in the default root left behind by a killed (`kill -9`) or crashed daemon, or a reboot, is removed, with its `chrome-*` dir, when the next session launches its Chrome.

The daemon also writes `rod-cli-<session>.port` and the shared `rod-cli-daemon.log` (started over once it exceeds 1 MiB) to the temp dir. The next daemon start removes port files of daemons that no longer run.

### Verify

```bash
rod-cli --version
```

## Per-Assistant Guides

| Assistant | Onboarding Path | Guide |
|-----------|----------------|-------|
| **Claude Code** | Agent Skill (`SKILL.md`) | [claude-code.md](claude-code.md) |
| **Codex CLI** | Agent Skill + AGENTS.md | [codex-cli.md](codex-cli.md) |
| **Gemini CLI** | GEMINI.md context file | [gemini-cli.md](gemini-cli.md) |
| **Pi (pi.dev)** | TypeScript extension (recommended) or Agent Skill | [pi.md](pi.md) |
| **opencode** | Native skills + Claude compat | [opencode.md](opencode.md) |

**Important:** rod-cli is NOT an MCP server. Every assistant onboards it by shelling out to the `rod-cli` binary — not via MCP tool registration. Do NOT attempt to register rod-cli as an MCP server.

## One Skill, Four Assistants

rod-cli ships a single `skills/rod-cli/SKILL.md` that works across Claude Code, Codex CLI, Pi, and opencode (all four consume the cross-tool Agent Skills standard). Gemini CLI uses a `GEMINI.md` context file instead — see the Gemini guide for details.

The shared `~/.agents/skills/rod-cli/` path is read by Codex, Pi, and opencode at once. Claude Code reads from `~/.claude/skills/rod-cli/` (which opencode also reads if the Claude compat layer is enabled).
