# Gemini CLI — rod-cli Onboarding

Gemini CLI does **not** have a skills primitive. Instead, rod-cli is onboarded via a **`GEMINI.md` context file** that teaches Gemini how and when to shell out to the `rod-cli` binary.

## Install

### 1. Create GEMINI.md

Add rod-cli instructions to a `GEMINI.md` file:

```bash
# Global (applies to all projects)
cat >> ~/.gemini/GEMINI.md << 'EOF'

## rod-cli — Web Automation

rod-cli is a native web automation CLI installed at `rod-cli`. Use it for any
task involving web browsing, scraping, form interaction, or page evaluation.

### When to use rod-cli
- Browsing web pages: `rod-cli goto <url>`
- Reading page content: `rod-cli snapshot` (token-efficient DOM tree)
- Clicking elements: `rod-cli click <ref>`
- Filling forms: `rod-cli fill <selector> <value> --submit`
- Running JavaScript: `rod-cli eval <script>`
- Taking screenshots: `rod-cli screenshot`

### Important
- rod-cli runs as a background daemon — the first command starts it.
- Always run `rod-cli close` when done to clean up.
- For multi-tab work, use `--session <name>`.
- rod-cli is NOT an MCP server — do not attempt MCP registration. Use it as a CLI.
EOF
```

Or for a specific project:

```bash
mkdir -p .gemini
cat >> .gemini/GEMINI.md << 'EOF'
... (same content as above) ...
EOF
```

### 2. Verify

Start a Gemini CLI session and ask:

```
Browse to https://example.com and tell me the page title.
```

Gemini should invoke `rod-cli goto` and `rod-cli snapshot` to complete the task.

## Why not MCP?

rod-cli is a **pure CLI/daemon** — it has no MCP server mode, no JSON-RPC endpoint, and no MCP dependency. The `gemini mcp add` command does not apply. Gemini onboards rod-cli by learning its CLI surface from `GEMINI.md`.

## References

- [Gemini CLI documentation](https://github.com/google-gemini/gemini-cli)
- `gemini --help` for context-file configuration
