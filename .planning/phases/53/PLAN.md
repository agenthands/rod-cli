---
author: architect
responsible: architect
phase: 53
status: ready
parent_artifacts:
  - .planning/phases/53/CONTEXT.md
  - .planning/REQUIREMENTS.md (CORE-03, PI-02, PI-03, CI-02, CI-03)
---

# Phase 53: Publish Readiness + Auto-Publish Pipeline — PLAN

## {P} Precondition
Three green raw-TS packages in the `extensions/*` workspace; no LICENSE/publish-fields; Pi has imprecise `peerDependencies:"*"`; `typebox`/`@earendil-works/pi-ai`/`@earendil-works/pi-coding-agent`/`@oh-my-pi/pi-coding-agent` all on public npm; rod-cli at `~/bin`.

## {Q} Postcondition
All three package.json are publish-ready (publishConfig/repository/homepage/bugs/engines, accurate files, MIT LICENSE in each tarball, precise deps); `npm pack --dry-run` for each lists exactly the intended `.ts`+README+LICENSE; the packed Pi tarball installs into a scratch dir (no repo node_modules) and loads+registers 13 tools with core+typebox+pi-ai resolved from node_modules; the packed OMP tarball loads similarly; a `publish-extensions` workflow on `ext-v*` runs the CI-01 gate then publishes all three (`--access public`, `NPM_TOKEN`, dependency order), verified by `npm publish --dry-run`; the `ext-v*` release convention is documented and referenced.

## must_haves.truths
1. **All three still green + no behavior change.** core 139 vitest, pi 38 vitest, omp 30 bun, all `tsc --noEmit` clean, after the package.json/dep edits. No tool/core source change.
2. **LICENSE in every tarball.** `npm pack --dry-run` for each shows `LICENSE` + `README.md` + `package.json` + only non-test `src/**/*.ts`; NO `__tests__`/`.smtc-cache`/`tmp`/`log`/`rod`/`node_modules`.
3. **Deps precise.** Pi: `typebox`+`@earendil-works/pi-ai`+`@agenthands/rod-cli-core` as real `dependencies`; `@earendil-works/pi-coding-agent` in `devDependencies`; NO `peerDependencies` block. OMP: core real dep, host type-only in devDeps. Core: no `dependencies`.
4. **Host-LOAD proven** (PI-03): packed Pi tarball → scratch install → loads + registers 13 tools, deps resolved from the scratch `node_modules` (not the repo).
5. **Publish workflow gated + disjoint** (CI-02): `on: tags: ['ext-v*']`; runs the CI-01 gate before any publish; a gate failure blocks; publishes core→pi→omp; verified via `npm publish --dry-run`.
6. **No rule violations** (`ts-no-any`, `ts-no-dynamic-import`, `ts-no-inline-cast-access`, `ts-import-type`) in any new/edited `.ts` (the scratch-load smoke scripts live in `/tmp`, outside the repo).

## §1 — LICENSE files (D-04)
Create identical MIT `LICENSE` at repo root and in `extensions/core/`, `extensions/pi/`, `extensions/omp/`. Standard MIT text, `Copyright (c) 2026 agenthands`.

## §2 — package.json publish-readiness

**All three** — add `publishConfig`, `repository` (with `directory`), `homepage`, `bugs`, `engines` per CONTEXT D-01; keep `files` (D-02); confirm `license:"MIT"`.

**core/package.json:** add the fields; no `dependencies`.

**pi/package.json (D-05):** set
```json
"dependencies": {
  "@agenthands/rod-cli-core": "^0.1.0",
  "typebox": "^1.3.4",
  "@earendil-works/pi-ai": "^0.80.3"
},
"devDependencies": {
  "@earendil-works/pi-coding-agent": "^0.80.3",
  "@types/node": "^22.0.0",
  "typescript": "^5.8.0",
  "vitest": "^3.0.0"
}
```
Delete the `peerDependencies` block.

**omp/package.json (D-06):** `dependencies: { "@agenthands/rod-cli-core": "^0.1.0" }`; devDeps keep `@oh-my-pi/pi-coding-agent` (^16.3.6), `@types/bun`, `typescript`.

After edits: `npm install` at root (relinks workspace with the new `^0.1.0` core range + moved pi deps), then re-run all three gates (must stay green — truth #1).

## §3 — README stubs (D-07)
Create minimal `extensions/core/README.md` + `extensions/omp/README.md` (title = package name; one-line purpose; "Prerequisites: the `rod-cli` binary + Chromium (`rod-cli install`)"; "Full docs + install: see the repo README / Phase 54"). Leave `extensions/pi/README.md` as-is.

## §4 — Publish workflow (D-08/D-09) `.github/workflows/publish-extensions.yml`
```yaml
name: publish-extensions
on:
  push:
    tags: ["ext-v*"]
jobs:
  publish:
    runs-on: ubuntu-latest
    permissions: { contents: read }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          registry-url: "https://registry.npmjs.org"
      - uses: oven-sh/setup-bun@v2
        with: { bun-version: latest }
      - run: npm ci
      # --- CI-01 gate (must pass before any publish) ---
      - run: npm run typecheck && npm test
        working-directory: extensions/core
      - run: npm run typecheck && npm test
        working-directory: extensions/pi
      - run: npm run typecheck && npm test
        working-directory: extensions/omp
      # --- publish in dependency order ---
      - run: npm publish --access public
        working-directory: extensions/core
        env: { NODE_AUTH_TOKEN: "${{ secrets.NPM_TOKEN }}" }
      - run: npm publish --access public
        working-directory: extensions/pi
        env: { NODE_AUTH_TOKEN: "${{ secrets.NPM_TOKEN }}" }
      - run: npm publish --access public
        working-directory: extensions/omp
        env: { NODE_AUTH_TOKEN: "${{ secrets.NPM_TOKEN }}" }
```
(Gate steps precede publish steps; a gate failure fails the job before publish. `ext-v*` is prefix-disjoint from `v*`.)

## §5 — Release doc (D-10) `docs/extension-release.md`
Document: the three packages version together per release; `ext-vX.Y.Z` tags trigger `publish-extensions.yml`; extension semver is **independent** of the Go binary `v*` tags; cut-a-release steps (bump all three `version`s → commit → `git tag ext-vX.Y.Z` → `git push --tags` → workflow gates + publishes core→pi→omp). Add a one-line reference from `extensions/pi/README.md` ("Release process: see `docs/extension-release.md`").

## §6 — Verification (the gate)
1. `npm install` (root) → all three gates green (truth #1): core 139, pi 38, omp 30, tsc clean each.
2. **Pack contents (truth #2):** `cd extensions/<pkg> && npm pack --dry-run` for each → assert the file list = `package.json` + non-test `src/**/*.ts` + `README.md` + `LICENSE`, and NONE of `__tests__`/`.smtc-cache`/`tmp`/`log`/`rod`/`node_modules`. Record each list.
3. **Dep precision (truth #3):** grep the three package.json; confirm Pi has no `peerDependencies` and typebox/pi-ai/core in `dependencies`.
4. **PI-03 host-LOAD (truth #4):** `npm pack` core + pi (produces tarballs) → `mktemp -d` scratch → `npm init -y` → `npm install <core-tgz> <pi-tgz>` (fetches typebox+pi-ai from public npm, installs core+pi from tarballs) → a `/tmp` Bun/node script that imports the installed `@agenthands/rod-cli-pi` (from the scratch `node_modules`, NOT the repo), calls its default factory with a mock ExtensionAPI (real `pi.exec` via child_process, TypeBox `Type`/`StringEnum` come from the extension's own imports which resolve in scratch node_modules), asserts 13 tools register, and drives one `browse_goto`+`browse_snapshot` round-trip (same session) against a local fixture. Assert core+typebox+pi-ai all resolved from the scratch dir. Record output. Repeat an analogous OMP tarball load smoke (Phase 53 SC3).
5. **Publish dry-run (truth #5):** `cd extensions/<pkg> && npm publish --dry-run --access public` for each (core→pi→omp) → exits 0, shows the would-publish contents; record. (No live token — this is the honest ceiling.)
6. **Workflow + disjointness:** confirm `publish-extensions.yml` triggers only on `ext-v*`; note `v*` (release.yml) is prefix-disjoint.
7. **Rule sweep:** `ts-no-any`/`ts-no-dynamic-import`/`ts-no-inline-cast-access` empty across `extensions/*/src` (no regressions).

## Execution order
1. §1 LICENSE files.
2. §2 package.json edits + root `npm install` + re-run gates.
3. §3 README stubs.
4. §4 publish workflow.
5. §5 release doc + Pi README reference.
6. §6 verification (pack dry-run ×3, scratch host-load pi+omp, publish dry-run ×3, greps).

## Files changed
| File | § | Status |
|---|---|---|
| `LICENSE`, `extensions/{core,pi,omp}/LICENSE` | §1 | NEW (MIT) |
| `extensions/core/package.json` | §2 | MODIFIED (publish fields) |
| `extensions/pi/package.json` | §2 | MODIFIED (publish fields + deps hygiene, drop peerDeps) |
| `extensions/omp/package.json` | §2 | MODIFIED (publish fields + core range) |
| `extensions/core/README.md`, `extensions/omp/README.md` | §3 | NEW (stubs) |
| `extensions/pi/README.md` | §5 | MODIFIED (release-doc reference) |
| `.github/workflows/publish-extensions.yml` | §4 | NEW |
| `docs/extension-release.md` | §5 | NEW |
| `package-lock.json` | §2 | MODIFIED |
