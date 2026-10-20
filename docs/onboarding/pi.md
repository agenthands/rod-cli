# Pi (pi.dev) — rod-cli Onboarding

Pi onboards rod-cli either as an **Agent Skill** — a `SKILL.md` file placed under Pi's skills directory (below) — or through the [first-class TypeScript extension](#first-class-extension-v22), the recommended path for Pi users.

## Install

### 1. Install Pi (if not already)

```bash
npm install -g --ignore-scripts @earendil-works/pi-coding-agent
```

### 2. Place the skill

Copy the shipped skill directory to Pi's skills path:

```bash
# Global (recommended — avoids the project-trust requirement)
mkdir -p ~/.pi/agent/skills
cp -R skills/rod-cli ~/.pi/agent/skills/

# Also works via the shared agents path
mkdir -p ~/.agents/skills
cp -R skills/rod-cli ~/.agents/skills/
```

**Note:** Project-scoped skills (`.pi/skills/` or `.agents/skills/`) require the project to be **trusted** first. Global `~/.pi/agent/skills/` avoids this. See Pi's project-trust documentation for details.

### 3. No MCP

Pi has explicit **"No MCP support"** by design. rod-cli is a pure CLI/daemon, not an MCP server — this is the correct and only integration path. Do NOT attempt to register rod-cli as an MCP tool.

### 4. Verify

Start a Pi session and ask:

```
Browse to https://example.com and give me the page title.
```

Pi should invoke `rod-cli goto` and `rod-cli snapshot` to complete the task. Alternatively, invoke rod-cli directly:

```
rod-cli --version
```

## First-Class Extension (v2.2+)

rod-cli ships a first-class Pi TypeScript extension with typed tools, lifecycle hooks, and prompt guidance. This is the recommended path for Pi users.

### Install

```bash
# Working path today (local checkout):
pi install ./extensions/pi

# Once published to npm (available once published):
pi install npm:@agenthands/rod-cli-pi
```

### What you get

- **16 typed browser tools** — `browse_goto`, `browse_snapshot`, `browse_click`, etc.
- **Automatic daemon lifecycle** — browser starts on first use; quitting Pi runs `rod-cli close`, which closes the default session and removes its profile (named sessions close on the 15-minute idle timeout)
- **Named sessions** — `session` parameter on every tool for multi-browser workflows
- **Token-efficient output** — markdown snapshots, not raw HTML

The [Skill-based path](#install) (SKILL.md) remains available as a zero-install fallback for users who switch between assistants or prefer no npm dependencies.

See [extensions/pi/README.md](../../extensions/pi/README.md) for the full tool catalog.

**OMP users:** Use the native [`@agenthands/rod-cli-omp`](../../extensions/omp/README.md) extension instead -- it provides the same 16 typed browser tools for the OMP host.

## References

- [Pi documentation](https://pi.dev)
- Pi's built-in `help` command for skills configuration
