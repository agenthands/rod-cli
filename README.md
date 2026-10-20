# rod-cli

`rod-cli` is a lightweight, zero-dependency command-line interface (CLI) that gives AI assistants native web browsing, scraping, and interaction capabilities. Built on top of [godoll](https://github.com/agenthands/godoll) (which wraps [go-rod](https://github.com/go-rod/rod)), it replaces bulky Node.js setups with a fast, stealth-enabled compiled Go binary.

Designed as a "Skill" for LLMs (Claude, Gemini, and custom autonomous agents), it is driven entirely through plain command-line invocations — each command runs and exits while a background daemon keeps the browser state alive between calls. It features aggressive context-window optimization—stripping out DOM noise and rendering pages as a compact accessibility snapshot with element refs—so your AI can read the web without burning through token limits.

## Key Benefits:

- **Stealth First:** Powered by `godoll`, `rod-cli` natively masks browser fingerprints and provides seamless network interception capabilities to bypass anti-bot systems.
- **Token-Efficient by Design:** It avoids loading massive accessibility trees or verbose JSON tool schemas into the model context. It relies on concise, purpose-built commands.
- **Zero Dependency Hell:** It’s a single Go binary. No Node.js, no `node_modules`. Just download, run `rod-cli install` to fetch Chromium, and automate.
- **Agent-Ready:** Each command runs as a one-shot invocation backed by a persistent daemon, making it a perfect drop-in skill for coding agents to call directly from a shell.
- **Rock-Solid Stability:** Leverages `godoll/retry` logic for auto-waiting and crash resilience. No more flaky timeouts or zombie browser processes: each session's Chrome profile and temp dir are removed when the session ends, and the next session that launches Chrome removes any that a killed daemon left under the default profile root.

## Documentation

- **[Installation](INSTALL.md)**: Instructions for installing `rod-cli` and its dependencies.
- **[Coding-Assistant Onboarding](docs/onboarding/README.md)**: Per-assistant install + agent-skill guides for Claude Code, Codex CLI, Gemini CLI, Pi, and opencode — with copy-paste sequences and verify steps.
- **[Architecture](ARCHITECTURE.md)**: Overview of the persistent daemon and stealth engine.
- **[Stealth Configuration](docs/stealth-config.md)**: The configurable stealth surface — fingerprint pins, per-session proxy, hardening toggles, and humanize tuning, with their honest constraints.
- **[Stealth Validation](docs/stealth-validation.md)**: What the offline harness deterministically proves vs the best-effort live ceiling — and the TLS/IP/CDP layers rod-cli cannot control (no "undetectable" guarantee).
- **[CDP Footprint](docs/cdp-footprint.md)**: The per-domain Chrome DevTools Protocol footprint inventory — which CDP domains a session enables, when, and how the v1.7 baseline reduction keeps a plain session off Runtime/Network/Fetch.
- **[Usage Guide](USAGE.md)**: CLI reference and examples, including where session profiles, temp files and daemon files live and when they are removed.
- **[Agent Skill Definition](skills/rod-cli/SKILL.md)**: The prompt context to supply to your LLM agent so it knows how to use the CLI.
- **[Pi Extension](extensions/pi/README.md)**: First-class Pi TypeScript extension with 16 typed browser tools, lifecycle hooks, and prompt guidance.
- **[OMP Extension](extensions/omp/README.md)**: First-class OMP TypeScript extension with 16 typed browser tools, lifecycle hooks, and prompt guidance.
- **[Reference Guides](skills/rod-cli/references/)**: Advanced tutorials on request mocking, code evaluation, tab management, and more.
- **[Plugin Development](docs/plugins/README.md)**: Writing, loading, and running rod-cli plugins — start here for the plugin docs index.

## Integration Paths

rod-cli offers three integration paths. All use the same `rod-cli` binary underneath.

| | OMP extension (`@agenthands/rod-cli-omp`) | Pi extension (`@agenthands/rod-cli-pi`) | Agent Skill (`skills/rod-cli/SKILL.md`) |
|---|---|---|---|
| **Host** | OMP (oh-my-pi) | Pi (earendil) | any Agent-Skills assistant (Claude Code, Codex, opencode, ...); Gemini CLI reads it from `GEMINI.md` |
| **Setup** | `omp plugin install npm:@agenthands/rod-cli-omp` (once published) / `omp plugin link ./extensions/omp` (today) | `pi install npm:@agenthands/rod-cli-pi` (once published) / `pi install ./extensions/pi` (today) | copy the `skills/rod-cli/` directory (`SKILL.md` + `references/`) to the assistant's skills dir |
| **Tools** | 16 typed `browse_*` (injected `pi.zod`) | 16 typed `browse_*` (TypeBox) | LLM shells out to `rod-cli` subcommands |
| **Validation / lifecycle** | schema + session_start/shutdown hooks | schema + session_start/shutdown hooks | none |
| **Best for** | OMP users (native path) | Pi users | multi-assistant / zero-npm |

See the per-package READMEs for full details: [Pi Extension](extensions/pi/README.md), [OMP Extension](extensions/omp/README.md), [Agent Skill](skills/rod-cli/SKILL.md).

## Development

Build with `go build -o rod-cli .`; godoll is fetched from GitHub like any other module.

To build against a local godoll checkout, run: go work init . ../godoll
