# Requirements — v1.8 Debt Cleanup & Coding-Assistant Onboarding

**Milestone goal:** Retire the three v1.7 follow-ups and ship authoritative install + agent-skill documentation so any of the five major coding assistants can adopt rod-cli.

**Grounding:** rod-cli is a **pure CLI/daemon** (no MCP server — verified at HEAD: no `mcp` subcommand, no JSON-RPC, no MCP dep). All onboarding docs therefore teach the agent to **shell out to the `rod-cli` binary** via each tool's agent-skill / instructions-file mechanism; the MCP path is deliberately not documented for any assistant. Research synthesis: `.planning/research/assistant-onboarding-SUMMARY.md`.

---

## v1.8 Requirements

### Toolchain & Supply-Chain (BUILD) — v1.7 follow-up F1

- [ ] **BUILD-01**: `go.mod` pins the toolchain to **go1.26.1** (add a `toolchain go1.26.1` directive and align the `go` directive); CI and release workflows build on go1.26.1; the full build and existing test suite pass on go1.26.1.
- [ ] **BUILD-02**: `govulncheck` reports **no known-vulnerable called paths** for the 13 stdlib vulns the F1 audit flagged (fixed in go1.26.1); any residual finding is documented with justification. A `govulncheck` run is wired into CI so the gate cannot silently regress.

### CDP-Ledger Coverage (LEDGER) — v1.7 follow-up

- [ ] **LEDGER-01**: CDP domains enabled via the **plugin lifecycle path** (the path the v1.7 ledger did not observe) are recorded in the per-session CDP-domain ledger, identically to the main navigation path — no enabled domain escapes the inventory.
- [ ] **LEDGER-02**: The offline detection harness asserts the plugin-path CDP footprint — a test exercises a plugin that enables a lazy CDP domain and verifies the ledger reflects it — closing the v1.7 coverage hole so the regression cannot reappear unseen.

### Real Font Spoofing (FONT) — v1.7 follow-up

- [ ] **FONT-01**: With `--font-spoof` enabled, the page's **detectable font availability actually changes** — the godoll font-injector no-op is replaced with a real injector so a font-probe reads a spoofed font set, not the host's genuine fonts.
- [ ] **FONT-02**: The spoofed font set is **coherent with the active profile's OS/locale** and **stable within a session** (identical across re-reads on the same session); `--font-spoof=false` restores the genuine host font behavior.
- [ ] **FONT-03**: The offline detection harness asserts FONT-01 and FONT-02 (font set differs from baseline when on, identical across re-reads, restored when off) — the fix is observable, not just claimed.

### Coding-Assistant Onboarding Docs (DOC)

- [ ] **DOC-01**: A binary-install section documents the shared prerequisite for every assistant — getting `rod-cli` on PATH (`go install` and/or a prebuilt binary) plus `rod-cli install` to fetch Chromium — with a verify step.
- [ ] **DOC-02**: The shipped `skills/rod-cli/SKILL.md` is updated to the current cross-tool Agent-Skills standard (valid `name`/`description` frontmatter plus explicit "when to use" trigger phrases) so one skill directory works across Claude Code, Codex, Pi, and opencode.
- [ ] **DOC-03**: **Claude Code** install documented — place the skill at `~/.claude/skills/rod-cli/` (user) or `.claude/skills/rod-cli/` (project), with the restart-on-first-create gotcha and a `/rod-cli` verify step.
- [ ] **DOC-04**: **Codex CLI** install documented — SKILL.md agent-skill path (pinned against the installed Codex version, since the skills directory differs between official `.agents/skills/` and community `~/.codex/skills/`) plus AGENTS.md as the secondary instructions path.
- [ ] **DOC-05**: **Gemini CLI** install documented — using a `GEMINI.md` context file (`~/.gemini/GEMINI.md` global or project `.gemini/GEMINI.md`), explicitly noting Gemini has no skills primitive and that the MCP path does not apply (rod-cli is not an MCP server).
- [ ] **DOC-06**: **Pi** install documented — `npm install -g --ignore-scripts @earendil-works/pi-coding-agent`, the SKILL.md skill under `~/.pi/agent/skills/` (or shared `~/.agents/skills/`), with an explicit "Pi has no MCP support" note and the project-trust gotcha.
- [ ] **DOC-07**: **opencode** install documented — the native skill path (`.opencode/skills/rod-cli/` or `~/.config/opencode/skills/`) and the fact that opencode natively reads `.claude/skills/` and `.agents/skills/`, plus AGENTS.md instructions.
- [ ] **DOC-08**: Each per-assistant section provides a **copy-paste install sequence and a concrete verify step** that confirms the assistant can actually drive rod-cli (not just that files are placed).
- [ ] **DOC-09**: Every documented claim is **accurate against the real rod-cli command surface and current tool docs** — verified (no MCP install path is documented, the shared `~/.agents/skills/` substrate is noted where it applies), with the docs discoverable from the top-level README.

---

## Future Requirements (deferred)

- Heavier first-class **Extensions** that wrap rod-cli as an LLM-callable tool (Pi TypeScript extension, Gemini `gemini-extension.json`) — optional polish beyond shell-out onboarding.
- Deeper CDP signal obfuscation (v2 candidate `CDP-DEEP-01`, carried from v1.7).
- A genuine MCP server mode for rod-cli (would re-open the MCP install path for Codex/Gemini/opencode) — only if a future milestone re-adds MCP.

## Out of Scope

- **TLS/JA3-JA4 spoofing** — rod-cli drives real Chrome; its TLS is authentic by construction (spoofing lives in the separate "munch" project). Permanent exclusion.
- **MCP server install instructions** — rod-cli does not ship an MCP server; documenting one would be false.
- Upstream-godoll hygiene items beyond the font-injector fix (F2/F4 from the v1.7 security review).

---

## Traceability

| REQ-ID | Phase | Status |
|--------|-------|--------|
| BUILD-01 | TBD | pending |
| BUILD-02 | TBD | pending |
| LEDGER-01 | TBD | pending |
| LEDGER-02 | TBD | pending |
| FONT-01 | TBD | pending |
| FONT-02 | TBD | pending |
| FONT-03 | TBD | pending |
| DOC-01 | TBD | pending |
| DOC-02 | TBD | pending |
| DOC-03 | TBD | pending |
| DOC-04 | TBD | pending |
| DOC-05 | TBD | pending |
| DOC-06 | TBD | pending |
| DOC-07 | TBD | pending |
| DOC-08 | TBD | pending |
| DOC-09 | TBD | pending |

*Traceability filled by the roadmap.*
