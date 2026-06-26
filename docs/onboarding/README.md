# Installing rod-cli for Coding Assistants

rod-cli is a native, token-efficient web automation CLI designed for LLM integration. It communicates via standard I/O — no MCP server, no JSON-RPC overhead. This guide shows how to install rod-cli and onboard it into your coding assistant.

## Prerequisites: Install rod-cli

Every assistant needs `rod-cli` installed and on `PATH`. Choose one:

### Via Go (recommended)

```bash
go install github.com/agenthands/rod-cli@latest
```

### Via prebuilt binary

Download the latest binary from [GitHub Releases](https://github.com/agenthands/rod-cli/releases) and place it on your `PATH`.

### Install Chromium

rod-cli requires Chromium. After installing the binary, run:

```bash
rod-cli install
```

This downloads Chromium to `~/.cache/rod/browser/`.

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
| **Pi (pi.dev)** | Agent Skill via bash | [pi.md](pi.md) |
| **opencode** | Native skills + Claude compat | [opencode.md](opencode.md) |

**Important:** rod-cli is NOT an MCP server. Every assistant onboards it by shelling out to the `rod-cli` binary — not via MCP tool registration. Do NOT attempt to register rod-cli as an MCP server.

## One Skill, Four Assistants

rod-cli ships a single `skills/rod-cli/SKILL.md` that works across Claude Code, Codex CLI, Pi, and opencode (all four consume the cross-tool Agent Skills standard). Gemini CLI uses a `GEMINI.md` context file instead — see the Gemini guide for details.

The shared `~/.agents/skills/rod-cli/` path is read by Codex, Pi, and opencode at once. Claude Code reads from `~/.claude/skills/rod-cli/` (which opencode also reads if the Claude compat layer is enabled).
