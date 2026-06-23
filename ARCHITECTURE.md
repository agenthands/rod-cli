# rod-cli Architecture

This document provides a high-level overview of the `rod-cli` system architecture, explaining how a simple CLI command translates into complex, stealth-enabled browser automation.

## 1. The Daemon Model
Most browser automation tools (like Node.js Playwright) require a long-running script to keep the browser instance alive. A CLI, by nature, is ephemeral—it runs and immediately exits.

To solve this, `rod-cli` operates using a **Daemon Architecture**:
1. When you run a command (e.g., `rod-cli open https://example.com`), the CLI first checks if a background daemon process for the current session exists.
2. If it does not exist, the CLI spawns a detached background instance of itself (`rod-cli daemon`).
3. The background daemon launches the Chromium browser via `godoll` and binds to a localized HTTP port.
4. The foreground CLI command then sends its request via a lightweight JSON HTTP payload to the daemon.
5. The daemon executes the command against the persistent browser instance and returns the result.

This allows you to execute sequential CLI commands (or pipe commands in a bash script) while the actual browser state, cookies, and memory remain fully intact in the background.

## 2. Godoll Engine Integration
`rod-cli` is deeply integrated with the `godoll` framework (which itself is a robust wrapper around `go-rod`).

### Stealth and Evasion
When the daemon boots the browser, it leverages `godoll/browser` and `godoll/stealth` to automatically configure:
- Custom user agents
- WebGL and Canvas fingerprint overrides
- WebRTC IP leak protection
- Standard bot-evasion properties (`navigator.webdriver = false`, etc.)

This ensures that `rod-cli` can bypass cloud bot-mitigation systems (like Cloudflare or Datadome) natively without any manual configuration from the user.

### Network Interception
Network mocking (the `rod-cli route` and `rod-cli route-list` commands) is routed through the `godoll/network` package. The daemon maintains an active `types.Context` which holds a list of user-defined routes. When the context applies these rules to the browser, it injects them alongside the default stealth/evasion headers, ensuring that mocking doesn't break anti-bot protection.

### Resilient Actions
Commands like `click` and `goto` do not blindly fire `go-rod` primitives. Instead, they are wrapped in `godoll/retry` exponential backoff logic. This means if an element isn't immediately clickable because of a loading spinner, or a navigation fails due to a temporary network hiccup, the daemon will automatically retry before failing.

## 3. Multiplexing
You can run multiple isolated browser instances simultaneously by using the `--session` flag. Each named session spins up its own dedicated background daemon and Chromium instance. This makes it trivial for an agent to perform multi-user workflows (e.g., User A sending a message, User B receiving it).

## 4. Invoking with no command
Running `rod-cli` with no arguments prints the banner, description, and full command list (see `--help`). The tool is driven entirely through explicit subcommands; there is no long-running server mode to start.
