# Requirements — v2.3 Distribution & Publishing

**Milestone goal:** Make rod-cli's browser-automation extension actually consumable — publish the Pi extension to npm, and build/test/publish a native OMP extension (OMP is the primary coding agent) — on a shared, host-agnostic core with a CI-driven release pipeline.

**Grounding (current state, verified 2026-07-05):**
- `extensions/pi/` (`@agenthands/rod-cli-pi` v0.1.0) exists with 13 `browse_*` tools + 116 vitest tests, but is **not on npm** (`npm view` → 404). Its README's `pi install npm:@agenthands/rod-cli-pi` would fail.
- OMP runs the Pi extension today only via a local `omp plugin link` symlink (`~/.omp/plugins/node_modules/@agenthands/rod-cli-pi → extensions/pi`) backed by a full local `node_modules`. No OMP-scoped package exists.
- The Pi extension's `ExtensionAPI` import is type-only (erased), but 5 enum tools runtime-import `StringEnum` from `@earendil-works/pi-ai` + `Type` from `typebox` — OMP's host dropped `StringEnum` and prefers injected `pi.zod`/`pi.typebox`.
- No `LICENSE` file exists (yet `package.json` lists it in `files`). CI is Go-only: no TS typecheck/test, no npm-publish job. Go releases use `v*` tags via goreleaser.

**Architecture decision (locked; core-packaging resolved by roadmap red-team):** shared runtime-agnostic **core** + **two thin host adapters**, realized as **three published raw-TS npm packages** — `@agenthands/rod-cli-core` (the shared core) depended on as a real `dependencies` entry by `@agenthands/rod-cli-pi` and `@agenthands/rod-cli-omp`. A relative-sibling core import would be excluded from each `npm pack` tarball, so the core must be its own package (not copied/bundled at pack time — that would break the raw-TS, no-build model). Adapters differ only in host scope, schema library, and test runner.

---

## v2.3 Requirements

### Shared Core (CORE)

- [ ] **CORE-01:** A runtime-agnostic core owns binary discovery (`findRodCli`), a shell-out wrapper (`execRodCli`) that takes a **normalized injected `ExecFn`** and returns a **normalized `ExecResult`** (so each host's `pi.exec` shape is adapted, not assumed), input validation, the per-subcommand timeout table, subcommand detection, and one argv-builder per `browse_*` tool. It has **zero host-package imports** and **zero schema-library runtime imports** — `typebox`/`StringEnum`/`pi.zod` appear only in adapters. Each argv-builder is a pure `(params) → string[]`.
- [ ] **CORE-02:** The core is unit-tested in isolation from any host — argv builders for all 13 tools (including enum, positional, and `--` separator handling), input-validation caps (URL/selector/expression), timeout resolution, and binary-discovery fallback — with no `pi` object required. The core owns **one** test config/runner (vitest, Node lane); each package's test include-glob is scoped to its own directory so runners never cross-pick.
- [ ] **CORE-03:** The core ships as its own publish-ready raw-TS package `@agenthands/rod-cli-core` (own `package.json` with `publishConfig.access: "public"`, `repository`/`homepage`/`bugs`/`engines`, accurate `files`, a real MIT `LICENSE`). It installs standalone and both adapters resolve it from `node_modules` (not the repo), verified via `npm pack` + scratch install.

### Pi Extension (PI)

- [ ] **PI-01:** `@agenthands/rod-cli-pi` is refactored to consume `@agenthands/rod-cli-core` as a real dependency; its adapter owns only the TypeBox/`StringEnum` parameter schemas and `pi.registerTool` wiring. Behavior is unchanged — the existing 116 vitest tests and `tsc --noEmit` stay green.
- [ ] **PI-02:** The Pi `package.json` is publish-ready: `publishConfig.access: "public"`, `repository`, `homepage`, `bugs`, `engines`, accurate `files`, a real `LICENSE` (MIT) file. Runtime schema-lib imports (`typebox`, `@earendil-works/pi-ai`) are declared as real `dependencies` — **not** `peerDependencies: "*"` — unless the Pi host is verified to inject them (verified by the PI-03 host-load).
- [ ] **PI-03:** `npm pack` of `@agenthands/rod-cli-pi` produces a tarball that, installed into a scratch dir with **no reliance on the repo's local `node_modules`**, **loads in a real Pi host and registers all 13 tools** (host-LOAD smoke — not merely "npm install exits 0"), with `@agenthands/rod-cli-core` resolved from `node_modules`.

### OMP Extension (OMP)

- [ ] **OMP-01:** A new `@agenthands/rod-cli-omp` package consumes the shared core (real dependency); its adapter imports `@oh-my-pi/pi-coding-agent` (type-only), builds parameter schemas with the injected `pi.zod`/`pi.typebox` (no runtime `StringEnum`/`typebox` import), and declares an `omp.extensions` manifest entry. `tsc --noEmit` (strict) passes with no excess-property error.
- [ ] **OMP-02:** All 13 `browse_*` tools register under OMP with argv mapping identical to Pi (shared core), plus the two lifecycle hooks. Prompt guidance is folded into each tool's `description` (OMP's `ToolDefinition` does not accept `promptGuidelines`/`promptSnippet`), with the tool named explicitly and the Pi usage example included. The `session_shutdown` daemon-close is **behaviorally** verified to fire on a real OMP shutdown — gated on OMP's actual terminal reason (verified against the host `session_shutdown` payload; OMP's documented reasons are start|switch|branch|tree|shutdown, so a copied `reason === "quit"` gate is likely a dead no-op) or ungated + idempotent.
- [ ] **OMP-03:** A `bun:test` suite — with a mock `ExtensionAPI` exposing `.zod`/`.typebox`/`.registerTool`/`.on`/`.exec` — covers tool-registration presence, per-tool argv mapping, enum/positional handling, and lifecycle behavior, and runs green under Bun.
- [ ] **OMP-04:** The OMP extension loads in a real OMP session (via `omp plugin install` or `omp plugin link`), its 13 `browse_*` tools appear in the tool catalog, and a `browse_goto` + `browse_snapshot` round-trip drives a fixture page — replacing the Pi-extension symlink as the OMP-native path.

### CI & Publishing (CI)

- [ ] **CI-01:** CI typechecks and tests all extension packages on push/PR — `tsc --noEmit` for each, `vitest` for the core + Pi (Node lane), `bun:test` for OMP — as a **blocking** gate alongside the existing Go job. (Landed early — Phase 51 — so the highest-risk refactor is protected from commit one.)
- [ ] **CI-02:** An auto-publish workflow triggers on an `ext-v*` git tag, runs the CI-01 gate, and publishes **all three** packages (core + Pi + OMP) to npm with `--access public` using an `NPM_TOKEN` secret. A tag whose gate fails does not publish. The `ext-v*` glob is anchored and disjoint from the Go binary `v*` trigger (no cross-trigger).
- [ ] **CI-03:** The extension release/versioning convention is documented — `ext-v*` tags, core + adapters versioned together per release, semver **independent** of the Go binary `v*` tags, and how to cut a release — in a release doc referenced from the extension READMEs.

### Documentation (DOCS)

- [ ] **DOCS-01:** Each package README (Pi + OMP) carries correct host-specific install instructions, the tool catalog, prerequisites (rod-cli binary + Chromium), and a verify-on-install step. `pi install npm:` / `omp plugin install npm:` claims are labelled as working only once a live publish exists — no unpublished-install claim presented as verified.
- [ ] **DOCS-02:** The top-level README, `docs/onboarding/pi.md`, and `skills/rod-cli/SKILL.md` cross-link both published extensions and clearly distinguish the three integration paths (OMP extension, Pi extension, Agent Skill).

---

## Dependencies & Assumptions

- **NPM_TOKEN / npm org:** Actual npm publication depends on an `@agenthands` npm org and an `NPM_TOKEN` repo secret provisioned by the maintainer. CI-02 builds the pipeline; the live publish executes once the token exists. PI-03/OMP-04/CORE-03 verification uses `npm pack` + scratch install + host-load + `npm publish --dry-run` where a live publish is not yet possible.
- **OMP host API:** The OMP adapter targets `@oh-my-pi/pi-coding-agent`'s injected `pi.zod`/`pi.typebox`. `pi.exec` return shape, `session_shutdown` reason vocabulary, `pi.zod` access shape (`pi.zod.object` vs `pi.zod.z.object` — docs contradict), and `ToolDefinition` field support are verified against the host types during Phase 52 planning, not assumed from Pi.
- **Live dev path:** This OMP session drives browsing through the `~/.omp/plugins/node_modules/@agenthands/rod-cli-pi → extensions/pi` symlink; the Phase 51 refactor must keep that path loadable (core resolvable in that node_modules via an `extensions/*` workspace or link).

## Out of Scope

- **New browser tools / capabilities** — this milestone packages and ships the existing 13-tool surface; it does not add features.
- **Marketplace-catalog distribution** — OMP marketplace installs do not feed extension modules (they surface skills/commands/hooks/tools/MCP only); the extension path is npm / `omp plugin link`.
- **Go binary changes** — the extensions are pure TypeScript shelling out to the existing rod-cli binary.
- **TLS/JA3 spoofing** — permanently out of scope (real Chrome).

## Traceability

| REQ-ID | Phase | Status |
|--------|-------|--------|
| CORE-01 | 51 | pending |
| CORE-02 | 51 | pending |
| PI-01 | 51 | pending |
| CI-01 | 51 | pending |
| OMP-01 | 52 | pending |
| OMP-02 | 52 | pending |
| OMP-03 | 52 | pending |
| OMP-04 | 52 | pending |
| CORE-03 | 53 | pending |
| PI-02 | 53 | pending |
| PI-03 | 53 | pending |
| CI-02 | 53 | pending |
| CI-03 | 53 | pending |
| DOCS-01 | 54 | pending |
| DOCS-02 | 54 | pending |
