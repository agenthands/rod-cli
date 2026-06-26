# Codex CLI — rod-cli Onboarding

Codex CLI onboards rod-cli as an **Agent Skill** — the same `SKILL.md` format used by Claude Code, Pi, and opencode.

## Install

### 1. Place the skill

Copy the shipped skill directory to Codex's skills path:

```bash
# Global (recommended)
mkdir -p ~/.agents/skills/rod-cli
cp skills/rod-cli/SKILL.md ~/.agents/skills/rod-cli/

# Project-scoped
mkdir -p .agents/skills/rod-cli
cp skills/rod-cli/SKILL.md .agents/skills/rod-cli/
```

> **Path note:** The official Codex skills path is `.agents/skills/` (project) or `$HOME/.agents/skills/` (global). The community fork path `~/.codex/skills/` differs — if you're using the community fork, adjust accordingly. Check your Codex version with `codex --version`.

### 2. Optional: AGENTS.md instructions

You can also add rod-cli instructions to a project's `AGENTS.md` file for additional context:

```markdown
# AGENTS.md

## rod-cli

This project uses rod-cli for web automation. The rod-cli binary is installed
globally. Use it for any web browsing, scraping, or form-interaction tasks.

Key commands:
- `rod-cli goto <url>` — navigate
- `rod-cli snapshot` — get token-efficient DOM snapshot
- `rod-cli click <ref>` — click an element
- `rod-cli eval <js>` — run JavaScript
```

### 3. Verify

Start a Codex session and ask it to browse:

```
Browse to https://example.com and tell me the page title.
```

Codex should invoke `rod-cli goto` followed by `rod-cli snapshot` or `rod-cli eval`.

## References

- [Codex CLI Skills documentation](https://github.com/openai/codex)
- `codex --help` for skills-path configuration
