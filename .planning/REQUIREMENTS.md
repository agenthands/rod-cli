# Requirements: v1.5 Plugin Ecosystem Documentation

## 1. Plugin Documentation

- [x] **PDOC-01**: A plugin author can follow an authoring guide to write, load, and run their first plugin end-to-end.
- [x] **PDOC-02**: A plugin author can look up every lifecycle hook (`OnRequest`, `OnResponse`, `OnLoad`, `OnDOMNodeInserted`) with its signature and payload shape in a reference page.
- [x] **PDOC-03**: A plugin author can look up the state/context API (token-optimized snapshot, cookies, localStorage, network context) with usage examples.
- [x] **PDOC-04**: A user can look up every plugin CLI command (`plugin load`, `plugin list`, `plugin run`) with its flags and exit codes in a reference page.
- [ ] **PDOC-05**: A reader can discover all plugin documentation from a `docs/plugins/` index linked from the README.

## 2. Example Plugins

- [x] **PEX-01**: A user can read and run the documented XSS scanner as a complete, polished worked example.
- [x] **PEX-02**: A user can read and run a small recipe plugin demonstrating each lifecycle hook.
- [x] **PEX-03**: A plugin author can copy a starter/template plugin to scaffold a new plugin.

## Future Requirements

*(Deferred to a later milestone)*

- A hosted/searchable docs site (e.g. generated static site) beyond in-repo Markdown.
- A plugin registry or marketplace for sharing third-party plugins.

## Out of Scope

- New engine capabilities, lifecycle hooks, or CLI commands — this milestone documents and exemplifies the v1.4 plugin system as built. Any gaps surfaced during documentation are treated as small corrective fixes, not new features.
- Vulnerability scanning logic or exploit payloads baked into the binary — consistent with v1.4, specific functional/exploit logic stays in user-written example scripts, not the core tool.

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| PDOC-02 | Phase 21 — Reference Documentation | Complete |
| PDOC-03 | Phase 21 — Reference Documentation | Complete |
| PDOC-04 | Phase 21 — Reference Documentation | Complete |
| PEX-01 | Phase 22 — Example Plugins | Complete |
| PEX-02 | Phase 22 — Example Plugins | Complete |
| PEX-03 | Phase 22 — Example Plugins | Complete |
| PDOC-01 | Phase 23 — Authoring Guide & Docs Index | Complete |
| PDOC-05 | Phase 23 — Authoring Guide & Docs Index | Pending |

**Coverage:** 8/8 requirements mapped to exactly one phase. No orphans, no duplicates.
