# Installation

`rod-cli` is a compiled Go binary, which means it requires zero Node.js or Python dependencies.

Installation has **two parts**:

1. **Install the binary** (and its Chromium) — so the `rod-cli` command exists on your machine (Options 1–2 below).
2. **Install the agent skill** — so your LLM agent knows how to *use* the binary ([As an Agent Skill](#as-an-agent-skill) below).

Installing only the binary gives you a CLI you can run by hand; installing the skill is what turns it into a drop-in capability your agent can call autonomously. For agent use, do both.

## Option 1: Install via Go (Recommended)

If you have Go 1.25+ installed, you can install it globally:

```bash
go install github.com/agenthands/rod-cli@latest
```

This will place the `rod-cli` binary in your `$GOPATH/bin` directory (usually `~/go/bin`). Ensure this directory is in your system's `$PATH`.

Once the binary is installed, you must install the local Chromium browser that `rod-cli` relies on:

```bash
rod-cli install
```

## Option 2: Build from Source

You can also clone the repository and build it manually:

```bash
git clone https://github.com/agenthands/rod-cli.git
cd rod-cli
go build -o rod-cli
sudo mv rod-cli /usr/local/bin/
```

## As an Agent Skill

`rod-cli` is designed to be driven by an LLM agent. Installing the binary (above) is necessary but **not sufficient** — your agent also needs the skill bundle so it knows the commands, the daemon model, and the token-efficient conventions. The skill lives in this repo at [`skills/rod-cli/`](skills/rod-cli/) (a `SKILL.md` plus a `references/` tree).

### Claude Code (and Claude Code–compatible agents)

Copy the skill bundle into a skills directory the agent discovers:

```bash
# Personal (available in every project):
mkdir -p ~/.claude/skills
cp -r skills/rod-cli ~/.claude/skills/

# — or — project-scoped (checked into a specific repo):
mkdir -p .claude/skills
cp -r skills/rod-cli .claude/skills/
```

The agent then auto-discovers the `rod-cli` skill by its `SKILL.md` frontmatter and loads it on demand. Keep the whole directory together — the `references/` files are progressively loaded as needed.

### Other agents (Gemini, Codex, custom harnesses)

For agents without a `skills/` discovery mechanism, supply [`skills/rod-cli/SKILL.md`](skills/rod-cli/SKILL.md) as system/prompt context so the model knows how to invoke the CLI, and point it at the [`references/`](skills/rod-cli/references/) guides for advanced flows (request mocking, code evaluation, tabs, storage/session state, plugins).

### Verify the skill is wired up

With the binary installed and the skill loaded, your agent should be able to run a one-shot command and clean up:

```bash
rod-cli goto https://example.com
rod-cli snapshot
rod-cli close
```

## Verifying Installation

Run the following command to verify the binary installation:

```bash
rod-cli --version
```
