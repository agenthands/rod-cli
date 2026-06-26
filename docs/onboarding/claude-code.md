# Claude Code — rod-cli Onboarding

Claude Code onboards rod-cli as an **Agent Skill** — a `SKILL.md` file that teaches Claude how and when to shell out to the `rod-cli` binary.

## Install

### 1. Place the skill

Copy the shipped skill directory to Claude Code's skills path:

```bash
# User-wide (recommended)
mkdir -p ~/.claude/skills/rod-cli
cp skills/rod-cli/SKILL.md ~/.claude/skills/rod-cli/

# Project-scoped
mkdir -p .claude/skills/rod-cli
cp skills/rod-cli/SKILL.md .claude/skills/rod-cli/
```

**Important:** If the `~/.claude/skills/` directory didn't exist when your Claude Code session started, you must **restart Claude** (`/exit` then relaunch) for the new skill to be discovered. Claude scans the skills directory at startup only.

### 2. Verify

Start a Claude Code session and ask:

```
/rod-cli --version
```

If rod-cli is on your `PATH` and the skill loaded correctly, Claude should invoke `rod-cli --version` and report the version. You can also test a real browse:

```
/rod-cli goto https://example.com
/rod-cli snapshot
```

## How it works

- The skill name is the directory name (`rod-cli`) — that's what you type after `/`.
- The `SKILL.md` frontmatter (`name`, `description`) tells Claude what the skill does.
- The skill teaches Claude the rod-cli command surface and when to reach for it (web browsing, scraping, form interaction, etc.).
- Every command shells out to the `rod-cli` binary on `PATH`.

## References

- [Claude Code Skills documentation](https://code.claude.com/docs/en/skills)
- [Claude Code custom slash commands](https://code.claude.com/docs/en/slash-commands)
