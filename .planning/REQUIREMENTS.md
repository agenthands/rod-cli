# Requirements: v1.4 Plugin Architecture

## 1. Plugin Engine Integration
- [ ] **PLUG-01**: Select and embed a lightweight scripting engine (e.g., `goja` for JS, or `gopher-lua` for Lua) to avoid native CGO dependencies.
- [ ] **PLUG-02**: Plugins must be dynamically loadable at runtime via file paths without requiring a recompilation of `rod-cli`.

## 2. Lifecycle Event Hooks
- [ ] **PLUG-03**: Implement `OnRequest` hook to allow plugins to observe outbound HTTP requests before they are sent.
- [ ] **PLUG-04**: Implement `OnResponse` hook to allow plugins to observe incoming HTTP responses.
- [ ] **PLUG-05**: Implement `OnLoad` hook to allow plugins to execute logic when a page finishes loading.
- [ ] **PLUG-06**: Implement `OnDOMNodeInserted` hook for observing dynamic DOM mutations natively.

## 3. CLI Management Commands
- [ ] **PLUG-07**: `rod-cli plugin load <path>` - Registers a plugin into the active session's daemon.
- [ ] **PLUG-08**: `rod-cli plugin list` - Displays all active plugins running in the daemon.
- [ ] **PLUG-09**: `rod-cli plugin run <name>` - Manually triggers a named plugin script.

## 4. State & Context Sharing
- [ ] **PLUG-10**: Provide a secure API for plugins to access the cached, token-optimized DOM Snapshot state from the daemon.
- [ ] **PLUG-11**: Provide a secure API for plugins to read active network context (cookies, local storage) without crashing the `godoll` context.

## Out of Scope
- Native Go Plugins (`plugin` standard library) – Excluded because it relies heavily on CGO and breaks cross-compilation reliability.
- Vulnerability scanning logic or specific payloads – `rod-cli` is a dual-use foundational tool; the implementation of specific functional testing or exploit logic must be handled entirely in user-written scripts, not baked into the binary.

## Traceability
*(To be populated by the roadmapper)*
