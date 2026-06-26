# opencode — rod-cli Onboarding

opencode onboards rod-cli as a **native skill** — plus it natively reads `.claude/skills/` for Claude compatibility.

## Install

### 1. Place the skill (native path)

```bash
# Global
mkdir -p ~/.config/opencode/skills/rod-cli
cp skills/rod-cli/SKILL.md ~/.config/opencode/skills/rod-cli/

# Project-scoped
mkdir -p .opencode/skills/rod-cli
cp skills/rod-cli/SKILL.md .opencode/skills/rod-cli/
```

### 2. Or: use Claude/agents compat (zero extra work)

opencode natively reads skills from these paths too — so if you've already installed rod-cli for Claude Code or Codex, opencode picks it up automatically:

| Path | Source |
|------|--------|
| `~/.claude/skills/rod-cli/` | Claude Code (user) |
| `.claude/skills/rod-cli/` | Claude Code (project) |
| `~/.agents/skills/rod-cli/` | Codex/Pi shared |
| `.agents/skills/rod-cli/` | Codex/Pi shared (project) |

To disable Claude compat (if you prefer opencode-native only):
```bash
export OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1
```

### 3. Optional: AGENTS.md instructions

Add to `AGENTS.md` for additional project context:

```markdown
## rod-cli

This project uses rod-cli for web automation. Commands: goto, snapshot, click,
fill, eval, close.
```

### 4. Verify

Start an opencode session and ask:

```
Go to https://example.com and tell me the page title.
```

opencode should invoke `rod-cli goto` followed by `rod-cli snapshot` or `rod-cli eval`.

## References

- [opencode documentation](https://opencode.ai/docs)
- Install: `curl -fsSL https://opencode.ai/install | bash`
