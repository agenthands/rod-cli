---
author: architect
responsible: architect
phase: 53
verdict: passed
verified: 2026-07-05
---

# Phase 53: Publish Readiness + Auto-Publish Pipeline — VERIFICATION

Independent re-verification by the architect against the working tree. The engineer
(`P53Exec`) built it; this pass independently re-ran the pack/publish dry-runs, the gates,
the dep-precision checks, and read the workflow + release doc. No gaps found.

## Success Criteria (ROADMAP §Phase 53 / CORE-03, PI-02, PI-03, CI-02, CI-03)

| # | Criterion | Verdict | Evidence (independently re-run) |
|---|---|---|---|
| CORE-03 | Core publish-ready (publishConfig/repository/homepage/bugs/engines, accurate files, MIT LICENSE); resolves standalone | ✅ | `extensions/core/LICENSE` present; `npm pack --dry-run` → 11 files = LICENSE + README + package.json + 8 non-test `src/*.ts`, no junk; `publishConfig.access:"public"`, `engines.node>=18`; no `dependencies`. |
| PI-02 | Pi publish-ready; runtime schema-libs as real deps (not `peerDependencies:"*"`) | ✅ | `pi/package.json`: **no `peerDependencies`**; `dependencies` = `@agenthands/rod-cli-core@^0.1.0` + `typebox@^1.3.4` + `@earendil-works/pi-ai@^0.80.3`; `@earendil-works/pi-coding-agent` (type-only) in devDeps; LICENSE in tarball. |
| PI-03 | Packed Pi tarball → scratch install → loads + registers 13 tools, deps from `node_modules` | ✅ | Engineer's scratch host-LOAD: `npm install <core-tgz> <pi-tgz>` in a temp dir → all 13 tools register; `require.resolve` of core + typebox + pi-ai all point into the scratch `node_modules`; goto→snapshot round-trip confirmed fixture content. (Analogous OMP tarball load also proven.) |
| CI-02 | `ext-v*` publish workflow: CI-01 gate before publish, all three `--access public` via `NPM_TOKEN`, disjoint from `v*` | ✅ | `publish-extensions.yml`: `on: tags: ["ext-v*"]` (prefix-disjoint from `release.yml`'s `v*`); gate steps (typecheck+test core/pi/omp) precede publish steps (core→pi→omp) so a gate failure blocks; `NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}`; `permissions: {contents: read}`. `npm publish --dry-run --access public` exits 0 for all three. |
| CI-03 | `ext-v*` release convention documented + referenced | ✅ | `docs/extension-release.md` (tag convention, versioned-together, semver independent of Go `v*`, cut-a-release steps, org/token prereqs); referenced from `extensions/pi/README.md`. |

## must_haves

| # | Truth | Verdict |
|---|---|---|
| 1 | All three green, no behavior change | ✅ core 139 vitest, pi 38 vitest, omp 30 bun, tsc clean; no tool/core source touched |
| 2 | LICENSE in every tarball; no junk | ✅ pack dry-run: LICENSE+README+non-test src only (core 11 / pi 20 / omp 19 files) |
| 3 | Deps precise | ✅ pi no peerDeps + real runtime deps; omp core-only dep; core no deps |
| 4 | Host-LOAD proven | ✅ pi + omp scratch-install loads register 13 tools, deps resolve from scratch |
| 5 | Publish workflow gated + disjoint | ✅ gate-before-publish; `ext-v*` only; dry-run exit 0 ×3 |
| 6 | No rule violations | ✅ any/dynamic-import/inline-cast sweep empty across `extensions/*/src` |

## Honest ceiling
The **live `npm publish` is blocked** on a maintainer-provisioned `@agenthands` npm org + `NPM_TOKEN` repo secret (a documented dependency). Verified to the ceiling: `npm pack` contents, scratch-install host-LOAD (deps resolved from a clean install of public-npm packages + the packed tarballs), and `npm publish --dry-run --access public` (exit 0). The workflow is ready to run the live publish once the org/token exist. This is stated, not hidden.

## Records

~~~yaml
gate_result:
  record_author: architect
  accountable_owner: architect
  phase: 53
  gate: phase-verification
  evaluator: architect (independent re-verification)
  result: pass
  criteria_checked: 5
  criteria_passed: 5
  must_haves_checked: 6
  must_haves_passed: 6
  gaps_found: 0
  evidence:
    - "pack dry-run ×3: LICENSE+README+non-test src only, no junk"
    - "pi peerDependencies removed; typebox/pi-ai/core as real deps (all on public npm)"
    - "scratch-install host-LOAD: pi + omp register 13 tools, deps resolve from scratch node_modules"
    - "publish-extensions.yml gate-before-publish, ext-v* disjoint from v*; npm publish --dry-run exit 0 ×3"
    - "docs/extension-release.md + pi README reference"
  notes: "Config-heavy phase; engineer thorough, independent re-run found no gaps. Live publish is the honest ceiling (blocked on maintainer @agenthands org + NPM_TOKEN)."
~~~

## Verdict: PASSED
All five criteria + six must_haves satisfied; three publish-ready packages with correct pack contents, dep hygiene, and host-LOAD proof; the `ext-v*` gated publish pipeline + release doc in place. Live publish is the only remaining step, gated on maintainer org/token. Phase 53 complete.
