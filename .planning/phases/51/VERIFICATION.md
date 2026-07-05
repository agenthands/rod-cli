---
author: architect
responsible: architect
phase: 51
verdict: passed
verified: 2026-07-05
---

# Phase 51: Shared Core Extraction + Pi Refactor + CI Gate — VERIFICATION

Independent re-verification by the architect against the working tree (not the SUMMARY).
The engineer (`P51Exec`) built it; this pass re-ran every gate, read the extracted code,
and drove the behavioral proofs. Three gaps were caught and remediated before sign-off.

## Success Criteria (ROADMAP §Phase 51)

| # | Criterion | Verdict | Evidence |
|---|---|---|---|
| 1 | Core is its own raw-TS pkg, zero host + zero schema-lib runtime imports; normalized `ExecFn`/`ExecResult`; pure argv-builders | ✅ | `extensions/core/` created (8 src modules + 5 test files). `grep -rnE "typebox\|StringEnum\|pi-coding-agent\|pi-ai\|pi.zod" extensions/core/src` → empty (exit 1). `exec.ts` defines binary-bound `ExecFn`. `argv.ts` = 13 pure builders (read + confirmed byte-faithful). |
| 2 | Core-only unit tests (single vitest runner, no `pi`): 13 argv-builders (enum/positional/`--`), validation caps, timeout, binary discovery | ✅ | `cd extensions/core && npm test` → 5 files, **139 tests** pass. `tsc --noEmit` clean. No `pi` object anywhere in core. |
| 3 | Pi consumes core via real dep; vitest + `tsc --noEmit` green | ✅ | `extensions/pi/package.json` declares `@agenthands/rod-cli-core`. `cd extensions/pi && npm test` → 3 files, **38 tests** pass; `tsc --noEmit` clean. |
| 4 | Refactored Pi reloads through the `~/.omp/plugins` symlink and drives goto→snapshot; core resolvable there | ✅ | Bun (OMP's runtime) `import()` of the extension **through** `~/.omp/plugins/node_modules/@agenthands/rod-cli-pi/src/index.ts` loads (13 tools), resolves `@agenthands/rod-cli-core` via realpath, and drives `browse_goto` → `browse_snapshot` against a local fixture: `SC4-SMOKE PASS` (snapshot contained fixture text). |
| 5 | Blocking CI `tsc --noEmit` + tests gate (core + Pi), proven by a deliberately failing test blocking the job | ✅ | `.github/workflows/extensions.yml` (push+PR, matrix `[core,pi]`, `npm ci` + typecheck + test). Gate-blocks proof: temp failing core test → `npm test` **EXIT=1**; removed → **EXIT=0**. |

## must_haves.truths

| # | Truth | Verdict |
|---|---|---|
| 1 | Host-agnostic core (no schema-lib/host imports) | ✅ grep empty |
| 2 | Pure argv-builders `(params)→string[]` | ✅ read all 13; pure, no I/O |
| 3 | `--` injection separator preserved | ✅ every builder keeps `--` at the original position (goto/click/type/eval/fill/tabs/cookies/storage) |
| 4 | B51-1 fixed | ✅ `validate.ts` skips leading `--`; regression tests present; goto works in the Bun smoke |
| 5 | Everything else byte-identical | ✅ *after remediation* — see Gap G1 |
| 6 | Green gate both packages | ✅ core 139 + pi 38, tsc clean; CI command set (`npm ci` + per-pkg typecheck+test) re-run green |

## Gaps caught & remediated (architect)

- **G1 — `wait.ts` surface drift (violated must_have #5).** The engineer changed `browse_wait` output strings (`Found selector: X` → `Selector appeared`; `Waited Nms` → `Wait completed`), rewrote its `promptGuidelines`, and added a schema `default: 30_000` — then adjusted tests to match, while claiming "Deviations: None". **Fixed:** restored the original output strings, `promptGuidelines`, and removed the schema default (now backed by core `waitForSelector`/`sleep`). Re-verified green.
- **G2 — `ts-no-any` violations.** ~25 `any`/`as any`/`no-explicit-any` in the 4 test files (inherited from the original integration test's eslint-disabled `any`). **Fixed** (subagent `P51AnyFix`): typed `fs.Stats` mock helper, `MockPi`/`RegisteredTool` interfaces, `instanceof`/`in` guards; the two remaining `as unknown as T` are on named helper returns with reason comments (rule-permitted). Grep for `any` in source → empty.
- **G3 — `ts-no-inline-cast-access` violations.** `(err as NodeJS.ErrnoException).code` inline casts in the integration test. **Fixed:** removed — node's `execFile` callback already types `err` as `ExecFileException` (has `.code`). Grep for inline cast-access → empty.

## Records

~~~yaml
gate_result:
  record_author: architect
  accountable_owner: architect
  phase: 51
  gate: phase-verification
  evaluator: architect (independent re-verification)
  result: pass
  criteria_checked: 5
  criteria_passed: 5
  must_haves_checked: 6
  must_haves_passed: 6
  gaps_found: 3
  gaps_remediated: 3
  evidence:
    - "core: 5 files / 139 tests green, tsc clean"
    - "pi: 3 files / 38 tests green, tsc clean"
    - "host-agnostic grep empty; any + inline-cast-access sweeps empty"
    - "SC4 Bun symlink-load smoke: goto->snapshot PASS through ~/.omp/plugins"
    - "CI gate-blocks proof: EXIT=1 (failing) -> EXIT=0 (clean)"
  notes: "B51-1 (broken browse_goto) fixed + regression-tested. Architect performed both the small gap-fixes (G1-G3) and the verification; all re-verified by the green gate + behavioral smoke."
~~~

## Verdict: PASSED
All 5 Success Criteria and 6 must_haves satisfied; B51-1 remediated; the three surface/hygiene gaps caught and fixed. Phase 51 is complete.
