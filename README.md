# rod-cli

`rod-cli` is a lightweight, zero-dependency command-line interface (CLI) that gives AI assistants native web browsing, scraping, and interaction capabilities. Built on top of [godoll](https://github.com/agenthands/godoll) (which wraps [go-rod](https://github.com/go-rod/rod)), it replaces bulky Node.js setups with a fast, stealth-enabled compiled Go binary.

Operating as a "Skill" rather than a traditional stateful MCP server, it is designed explicitly for LLMs (Claude, Gemini, and custom autonomous agents). It communicates via standard input/output (stdio) or acts directly as an MCP server. It features aggressive context-window optimization—stripping out DOM noise and converting web pages to LLM-friendly Markdown—so your AI can read the web without burning through token limits.

## Key Benefits:

- **Stealth First:** Powered by `godoll`, `rod-cli` natively masks browser fingerprints and provides seamless network interception capabilities to bypass anti-bot systems.
- **Token-Efficient by Design:** It avoids loading massive accessibility trees or verbose JSON tool schemas into the model context. It relies on concise, purpose-built commands.
- **Zero Dependency Hell:** It’s a single Go binary. No Node.js, no `node_modules`. Just download, run `rod-cli install` to fetch Chromium, and automate.
- **Agent-Ready via Stdio/MCP:** Works seamlessly as a background process using the standard MCP stdio transport, making it a perfect drop-in skill for coding agents.
- **Rock-Solid Stability:** Leverages `godoll/retry` logic for auto-waiting and crash resilience. No more flaky timeouts or zombie browser processes.

## Documentation

- **[Installation](INSTALL.md)**: Instructions for installing `rod-cli` and its dependencies.
- **[Architecture](ARCHITECTURE.md)**: Overview of the persistent daemon and stealth engine.
- **[Usage Guide](USAGE.md)**: Comprehensive CLI reference and examples.
- **[Agent Skill Definition](skills/rod-cli/SKILL.md)**: The prompt context to supply to your LLM agent so it knows how to use the CLI.
- **[Reference Guides](skills/rod-cli/references/)**: Advanced tutorials on request mocking, code evaluation, tab management, and more.
