# Requirements — v2.3 Distribution & Publishing

**Milestone goal:** Make rod-cli's browser-automation extension actually consumable — publish the Pi extension to npm, and build/test/publish a native OMP extension (OMP is the primary coding agent) — on a shared, host-agnostic core with a CI-driven release pipeline.

**Grounding (current state, verified 2026-07-05):**
- `extensions/pi/` (`@agenthands/rod-cli-pi` v0.1.0) exists with 13 `browse_*` tools + 116 vitest tests, but is **not on npm** (`npm view` → 404). Its README's `pi install npm:@agenthands/rod-cli-pi` would fail.
- OMP runs the Pi extension today only via a local `omp plugin link` symlink (`~/.omp/plugins/node_modules/@agenthands/rod-cli-pi → extensions/pi`) backed by a full local `node_modules`. No OMP-scoped package exists.
- The Pi extension's `ExtensionAPI` import is type-only (erased), but 5 enum tools runtime-import `StringEnum` from `@earendil-works/pi-ai` + `Type` from `typebox` — OMP's host dropped `StringEnum` and prefers injected `pi.zod`/`pi.typebox`.
- No `LICENSE` file exists (yet `package.json` lists it in `files`). CI is Go-only: no TS typecheck/test, no npm-publish job. Go releases use `v*` tags via goreleaser.

**Architecture decision (locked):** shared runtime-agnostic **core** + **two thin host adapters** (`@agenthands/rod-cli-pi`, `@agenthands/rod-cli-omp`). Adapters differ only in host scope, schema library, and test runner.

---

## v2.3 Requirements

### Shared Core (CORE)

- [ ] **CORE-01:** A runtime-agnostic core module owns binary discovery (`findRodCli`), the shell-out wrapper (`execRodCli`, accepting an injected `exec` function + `AbortSignal`), input validation, the per-subcommand timeout table, subcommand detection, and one argv-builder per `browse_*` tool. It has **zero host-package imports** and **zero schema-library runtime imports** — `typebox`/`StringEnum`/`pi.zod` appear only in adapters.
- [ ] **CORE-02:** The core is unit-tested in isolation from any host: argv builders for all 13 tools (including enum, positional, and `--` separator handling), input-validation caps (URL/selector/expression), timeout resolution, and binary-discovery fallback — with no `pi` object required.

### Pi Extension (PI)

- [ ] **PI-01:** `@agenthands/rod-cli-pi` is refactored to consume the shared core; its adapter owns only the TypeBox/`StringEnum` parameter schemas and `pi.registerTool` wiring. Behavior is unchanged — the existing 116 vitest tests and `tsc --noEmit` stay green.
- [ ] **PI-02:** The Pi `package.json` is publish-ready: `publishConfig.access: "public"`, `repository`, `homepage`, `bugs`, `engines`, an accurate `files` list, and a real `LICENSE` (MIT) file included in the package.
- [ ] **PI-03:** `npm pack` of `@agenthands/rod-cli-pi` produces a tarball a fresh consumer can install and load with **no reliance on the repo's local `node_modules`**; once published, `pi install npm:@agenthands/rod-cli-pi` resolves the package and registers all 13 tools.

### OMP Extension (OMP)

- [ ] **OMP-01:** A new `@agenthands/rod-cli-omp` package consumes the shared core; its adapter imports `@oh-my-pi/pi-coding-agent` (type-only), builds parameter schemas with the injected `pi.zod`/`pi.typebox` (no runtime `StringEnum`/`typebox` import), and declares an `omp.extensions` manifest entry.
- [ ] **OMP-02:** All 13 `browse_*` tools register under OMP with argv mapping identical to Pi (shared core), plus the two lifecycle hooks: `session_start` binary `--version` check and `session_shutdown` quit-gated daemon close. Prompt guidance is preserved (as `promptGuidelines` if OMP's `ToolDefinition` supports it, otherwise folded into `description` — resolved during planning against the OMP host API).
- [ ] **OMP-03:** A `bun:test` suite covers the OMP adapter — tool-registration presence, per-tool argv mapping, enum/positional handling, and lifecycle gating — and runs green under Bun.
- [ ] **OMP-04:** The OMP extension loads in a real OMP session (via `omp plugin install` or `omp plugin link`), its 13 `browse_*` tools appear in the tool catalog, and a `browse_goto` + `browse_snapshot` round-trip drives a fixture page — replacing the Pi-extension symlink as the OMP-native path.

### CI & Publishing (CI)

- [ ] **CI-01:** CI typechecks and tests both extension packages on push/PR — `tsc --noEmit` for each, `vitest` for Pi, `bun:test` for OMP, plus the shared-core tests — as a blocking gate alongside the existing Go job.
- [ ] **CI-02:** An auto-publish workflow triggers on an `ext-v*` git tag, runs the CI-01 gate, and publishes both packages to npm with `--access public` using an `NPM_TOKEN` secret. A tag whose gate fails does not publish.
- [ ] **CI-03:** The extension release/versioning convention is documented — `ext-v*` tags, semver **independent** of the Go binary `v*` tags, and how to cut a release — in a release doc referenced from the extension READMEs.

### Documentation (DOCS)

- [ ] **DOCS-01:** Each package README (Pi + OMP) carries correct host-specific install instructions, the tool catalog, prerequisites (rod-cli binary + Chromium), and a verify-on-install step; the previously-broken `pi install npm:` claim is now accurate.
- [ ] **DOCS-02:** The top-level README, `docs/onboarding/pi.md`, and `skills/rod-cli/SKILL.md` cross-link both published extensions and clearly distinguish the three integration paths (OMP extension, Pi extension, Agent Skill).

---

## Dependencies & Assumptions

- **NPM_TOKEN / npm org:** Actual npm publication depends on an `@agenthands` npm org and an `NPM_TOKEN` repo secret provisioned by the maintainer. CI-02 builds the pipeline; the live publish executes once the token exists. PI-03/OMP-04 verification uses `npm pack` + local install where a live publish is not yet possible.
- **OMP host API:** The OMP adapter targets `@oh-my-pi/pi-coding-agent`'s injected `pi.zod`/`pi.typebox`. Exact schema shape and `promptGuidelines` support are verified against the host API during planning (OMP-02).

## Out of Scope

- **New browser tools / capabilities** — this milestone packages and ships the existing 13-tool surface; it does not add features. (Rationale: distribution milestone.)
- **Marketplace-catalog distribution** — OMP marketplace installs do not feed extension modules (they surface skills/commands/hooks/tools/MCP only); the extension path is npm / `omp plugin link`.
- **Go binary changes** — the extensions are pure TypeScript shelling out to the existing rod-cli binary.
- **TLS/JA3 spoofing** — permanently out of scope (real Chrome).

## Traceability

| REQ-ID | Phase | Status |
|--------|-------|--------|
| CORE-01 | TBD | pending |
| CORE-02 | TBD | pending |
| PI-01 | TBD | pending |
| PI-02 | TBD | pending |
| PI-03 | TBD | pending |
| OMP-01 | TBD | pending |
| OMP-02 | TBD | pending |
| OMP-03 | TBD | pending |
| OMP-04 | TBD | pending |
| CI-01 | TBD | pending |
| CI-02 | TBD | pending |
| CI-03 | TBD | pending |
| DOCS-01 | TBD | pending |
| DOCS-02 | TBD | pending |
