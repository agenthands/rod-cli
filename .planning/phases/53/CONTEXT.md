---
author: architect
responsible: architect
phase: 53
phase_type: implementation
hard_bar: true
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (CORE-03, PI-02, PI-03, CI-02, CI-03)
  - .planning/ROADMAP.md (Phase 53 Success Criteria)
  - .planning/phases/51/SUMMARY.md, .planning/phases/52/SUMMARY.md
---

# Phase 53: Publish Readiness + Auto-Publish Pipeline — CONTEXT

**Gathered:** 2026-07-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Make all **three** packages (`@agenthands/rod-cli-core`, `-pi`, `-omp`) publish-ready and
prove they load from a clean install; wire the `ext-v*` auto-publish workflow. Live npm
publish is gated on a maintainer-provisioned `@agenthands` org + `NPM_TOKEN` — everything
short of the live push is verified via `npm pack` + scratch install + host-load + `npm publish --dry-run`.

**In scope:** CORE-03, PI-02, PI-03, CI-02, CI-03 — package.json publish-readiness,
LICENSE files, dep hygiene, `npm pack` content correctness, host-LOAD smokes, the
`ext-v*` workflow, and the release-convention doc.
**Out of scope:** README content (Phase 54 — DOCS-01/02; this phase creates minimal
README stubs only so `files` is accurate), any code/behavior change to the tools/core,
the live npm publish (blocked on maintainer token — dry-run verified here).
</domain>

## {P} What the phase can assume (verified 2026-07-05)

- Three raw-TS packages exist in the `extensions/*` npm workspace, all green (core 139 vitest, pi 38 vitest, omp 30 bun; all `tsc --noEmit` clean).
- Current package.json gaps (ALL three): no `publishConfig`, `repository`, `homepage`, `bugs`, `engines`; no `LICENSE` file anywhere (root or packages); core + omp have no README (pi has one from Phase 50). `license: "MIT"` is declared but no file backs it.
- Pi currently declares `peerDependencies: { @earendil-works/pi-coding-agent:"*", @earendil-works/pi-ai:"*", typebox:"*" }` — imprecise. Pi's runtime imports are `typebox` (`Type`) + `@earendil-works/pi-ai` (`StringEnum`); its `@earendil-works/pi-coding-agent` import is **type-only**.
- **Registry facts (verified via `npm view`):** `typebox@1.3.4`, `@earendil-works/pi-ai@0.80.3`, `@earendil-works/pi-coding-agent@0.80.3`, `@oh-my-pi/pi-coding-agent@16.3.7` are ALL on public npm. `@agenthands/rod-cli-pi` → 404 (unpublished — the milestone goal).
- CI: `.github/workflows/release.yml` triggers Go releases on `tags: ['v*']`; `.github/workflows/extensions.yml` runs the blocking core/pi/omp gate on push/PR. No publish job exists.

## {Q} What the phase must establish

1. **CORE-03** — core is a publish-ready package (`publishConfig.access:"public"`, `repository`/`homepage`/`bugs`/`engines`, accurate `files`, real MIT `LICENSE`); installs standalone; both adapters resolve it from `node_modules`. Verified via `npm pack` + scratch install.
2. **PI-02** — Pi package.json publish-ready (same fields + LICENSE). Runtime schema-lib imports (`typebox`, `@earendil-works/pi-ai`) declared as **real `dependencies`** (they are on public npm, so a scratch install resolves them) — NOT `peerDependencies:"*"`. `@earendil-works/pi-coding-agent` (type-only) → `devDependencies`.
3. **PI-03** — `npm pack` of Pi → installed into a scratch dir with **no repo `node_modules`** → **loads + registers all 13 tools**, with `@agenthands/rod-cli-core` (and `typebox`/`pi-ai`) resolved from `node_modules` (host-LOAD smoke, not "install exits 0").
4. **CI-02** — a `publish-extensions` workflow triggers on `ext-v*` (anchored, disjoint from the Go `v*` — different prefix, no cross-trigger), runs the CI-01 gate, and publishes all three (core→pi→omp order) with `--access public` via `NPM_TOKEN`; a gate failure blocks publish. Verified via `npm publish --dry-run` (no live token yet).
5. **CI-03** — the `ext-v*` release convention (core + adapters versioned together per release; semver independent of the Go `v*` tags; how to cut a release) documented in a release doc, referenced from the extension READMEs.

<decisions>
## Implementation Decisions

### Package.json publish fields (all three)
- **D-01:** Add to each: `"publishConfig": { "access": "public" }`; `"repository": { "type": "git", "url": "git+https://github.com/agenthands/rod-cli.git", "directory": "extensions/<pkg>" }`; `"homepage": "https://github.com/agenthands/rod-cli/tree/main/extensions/<pkg>#readme"`; `"bugs": "https://github.com/agenthands/rod-cli/issues"`; `"engines": { "node": ">=18" }`.
- **D-02:** Keep `files: ["src/**/*.ts", "!src/__tests__/**", "README.md", "LICENSE"]` (already accurate — the allowlist naturally excludes `__tests__`, `.smtc-cache`, `tmp`, `log`, `rod`; verify via `npm pack --dry-run`).
- **D-03:** Versioning: all three stay `0.1.0` this release (versioned together — CI-03). Adapters depend on the core via a concrete range `"@agenthands/rod-cli-core": "^0.1.0"` (resolves to the local workspace in dev; a real range for published installs) — replace the current `"*"`.

### LICENSE
- **D-04:** Create an identical MIT `LICENSE` in `extensions/core/`, `extensions/pi/`, `extensions/omp/` (each tarball must carry its own — npm pack only includes files within the package dir) AND a root `LICENSE`. Copyright line: `Copyright (c) 2026 agenthands`. Standard MIT text.

### Dependency hygiene (PI-02 — evidence-driven)
- **D-05:** **Pi:** move runtime imports to real `dependencies`: `typebox: "^1.3.4"`, `@earendil-works/pi-ai: "^0.80.3"` (both on public npm — a scratch install resolves them); `@agenthands/rod-cli-core: "^0.1.0"`. Move `@earendil-works/pi-coding-agent: "^0.80.3"` (type-only) to `devDependencies`. **Delete the `peerDependencies` block.**
- **D-06:** **OMP:** `dependencies: { @agenthands/rod-cli-core: "^0.1.0" }`; `devDependencies` keep `@oh-my-pi/pi-coding-agent` (type-only, host-provided), `@types/bun`, `typescript`. **Core:** no `dependencies` (host-agnostic).

### README stubs (so `files`/pack is accurate; full content is Phase 54)
- **D-07:** Create minimal `extensions/core/README.md` and `extensions/omp/README.md` (package name, one-line purpose, "full docs: Phase 54 / repo README", prerequisites pointer). Pi's existing README stays (Phase 54 revises all three). This keeps `npm pack` contents complete without pre-empting DOCS-01.

### Publish workflow (CI-02)
- **D-08:** New `.github/workflows/publish-extensions.yml`, `on: push: tags: ['ext-v*']` (disjoint from `v*` by prefix). Job: checkout → setup-node@20 + setup-bun@2 → `npm ci` → **run the CI-01 gate** (typecheck + test for core, pi, omp — same commands as `extensions.yml`; a failure fails the job BEFORE any publish) → publish **core, then pi, then omp** (dependency order) via `npm publish --access public` with `NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}` + a registry `.npmrc`/setup-node `registry-url`. Least-privilege: `NPM_TOKEN` used only in the publish steps.
- **D-09:** Since no live `NPM_TOKEN`/org exists yet, the workflow is **verified via `npm publish --dry-run`** for each package (per-package, in dependency order) — proving the pack contents + auth wiring shape without a live push. The live publish executes once the maintainer provisions the org + token (documented dependency).

### Release doc (CI-03)
- **D-10:** Create `docs/extension-release.md`: the `ext-v*` tagging convention, "core + pi + omp versioned together per release," semver **independent** of the Go `v*` binary tags, and the cut-a-release steps (bump all three versions → `ext-vX.Y.Z` tag → push → workflow publishes). Reference it from the extension READMEs (the reference lines land here or in Phase 54's README pass — at minimum the doc exists + Pi's README links it).

</decisions>

<disciplines>
## Lenses surfaced for this phase

~~~yaml
disciplines:
  - section: "inline"
    name: "prove-load-not-install"
    scope: ["extensions/*/", "publish verification"]
    guidance: "A publish-readiness claim is only real if the PACKED tarball, installed into a scratch dir with no repo node_modules, actually LOADS and registers its tools (host-LOAD) — not merely 'npm install exited 0'. Verify with pack + scratch install + a real-exec load, and npm publish --dry-run; never assert publishable from config inspection alone."
    rationale: "The whole milestone's value is a consumable package; a package that installs but fails to resolve its deps at load (the core, typebox, pi-ai) is not publish-ready, and only a load test surfaces that."
    override_signals:
      - "A package with zero runtime dependencies and zero host imports (the core) — a pack-contents check + standalone import is sufficient; a full host-load adds little."
      - "The live publish itself is legitimately blocked on the maintainer's npm org/token — there, npm publish --dry-run + scratch-install load is the honest ceiling, and that limit must be stated, not hidden."
~~~

</disciplines>

<canonical_refs>
## Canonical References
- `.planning/REQUIREMENTS.md` (CORE-03, PI-02, PI-03, CI-02, CI-03) + "Dependencies & Assumptions" (NPM_TOKEN/org gating).
- `.planning/ROADMAP.md` §"Phase 53" (5 Success Criteria) + §"Records" (red-team B2 → host-LOAD verification + peer→dependency resolution).
- `.github/workflows/release.yml` (the `v*` Go trigger — `ext-v*` must stay disjoint) + `.github/workflows/extensions.yml` (the CI-01 gate to reuse in the publish workflow).
- `extensions/{core,pi,omp}/package.json` (the files to make publish-ready).
- npm docs: `publishConfig`, `files`, `repository.directory`, `npm pack --dry-run`, `npm publish --dry-run --access public` (use Context7/`npm help` if a field's exact shape is uncertain).

</canonical_refs>

<code_context>
## Existing Code Insights
- The `files` allowlist is already correct (allowlist model → only listed globs ship). The risk is missing LICENSE/README files (listed but absent) — D-04/D-07 create them.
- Pi's `peerDependencies:"*"` is the one imprecise spot; D-05 replaces it with real deps (all on public npm).
- `extensions.yml` already encodes the CI-01 gate command set (`npm ci` + per-package typecheck + test with setup-node + setup-bun) — the publish workflow reuses exactly this before publishing.
- The `ext-v*` vs `v*` globs are disjoint by prefix (`ext-v1.0.0` does not match `v*`); no extra anchoring needed, but keep the publish trigger scoped to `ext-v*` only.

</code_context>

<specifics>
## Specific Ideas
- Publish ORDER is load-bearing: core first (adapters depend on it), then pi, then omp — so a consumer installing pi/omp can resolve the freshly-published core.
- The honest ceiling: no live `@agenthands` org/`NPM_TOKEN` in this environment → verify with `npm pack` (+ scratch install host-load) + `npm publish --dry-run`. State this limit explicitly in SUMMARY; do NOT claim a live publish occurred.
- `@earendil-works/pi-coding-agent` is TYPE-ONLY in Pi → devDependency, not a runtime/peer dep (erased at runtime; the Pi host provides it).

</specifics>

<deferred>
## Deferred Ideas
- Full README content + the three-path (OMP/Pi/Skill) comparison + cross-links — Phase 54 (DOCS-01/02).
- The actual live `npm publish` — blocked on maintainer `@agenthands` org + `NPM_TOKEN` (execute once provisioned; pipeline + dry-run land here).

</deferred>

---

*Phase: 53-publish-readiness-auto-publish-pipeline*
*Context gathered: 2026-07-05*
