---
author: architect
responsible: architect
phase: 51
phase_type: implementation
hard_bar: true
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (CORE-01, CORE-02, PI-01, CI-01)
  - .planning/ROADMAP.md (Phase 51 Success Criteria; Records — red-team fold B1/M3/M4/M5/M6)
  - .planning/STATE.md (v2.3 locked decisions)
  - extensions/pi/src/cli.ts (source of the logic being extracted)
---

# Phase 51: Shared Core Extraction + Pi Refactor + CI Gate — CONTEXT

**Gathered:** 2026-07-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Extract a **runtime-agnostic core** as its own raw-TS package `@agenthands/rod-cli-core`
(binary discovery, a normalized injected-`ExecFn`/`ExecResult` shell-out, input
validation, per-subcommand timeout table, subcommand detection, and one **pure**
argv-builder per `browse_*` tool), refactor `@agenthands/rod-cli-pi` to consume it
with **zero behavior change** (one deliberate exception: the pre-existing `validateInput` `--`-separator bug is FIXED — finding B51-1 / D-16), and land the **blocking CI typecheck+test gate first**.

**In scope:** the extraction + Pi rewire + CI gate (CORE-01, CORE-02, PI-01, CI-01).
**Out of scope:** the OMP adapter (Phase 52), publish-readiness/LICENSE/`npm pack`
(Phase 53), docs (Phase 54), any new browser capability, any Go binary change.
</domain>

## {P} What the phase can assume

- `extensions/pi/` (`@agenthands/rod-cli-pi` v0.1.0) exists: `src/{index,cli,lifecycle,types}.ts`,
  `src/tools/*.ts` (13 tools), 3 vitest files under `src/__tests__/`, `vitest.config.ts`.
- Current `execRodCli` closes over a module-level `_pi` singleton (`setPi`) and calls
  `_pi.exec("rod-cli", ["--raw", ...args], {signal, timeout})` → `{stdout, stderr, code}`.
- Each tool's `execute()` builds argv **and** shells out in one function; 5 tools
  runtime-import `StringEnum` (`@earendil-works/pi-ai`); every tool imports `Type` (`typebox`).
- Root `package.json` is **not** a workspace today (only a `terser` dev script).
- The live dev path is `~/.omp/plugins/node_modules/@agenthands/rod-cli-pi → extensions/pi`.
- Toolchain: Node 24, npm 11, Bun 1.3.14. Git clean on `main`.

## {Q} What the phase must establish

1. **CORE-01** — `@agenthands/rod-cli-core` exists as its own raw-TS package with
   **zero host-package imports** and **zero schema-lib runtime imports**
   (`grep` proves `typebox`/`StringEnum`/`pi.zod` appear only in adapters). It defines
   a normalized `ExecFn`/`ExecResult` contract; each argv-builder is pure `(params) → string[]`.
2. **CORE-02** — core-only unit tests (single vitest runner, Node lane, dir-scoped
   include) exercise all 13 argv-builders (enum, positional, `--` separator),
   validation caps (URL/selector/expression/empty-text), timeout resolution, and
   binary-discovery fallback — **with no `pi` object**.
3. **PI-01** — `@agenthands/rod-cli-pi` consumes the core as a real dependency; its
   adapter owns only the TypeBox/`StringEnum` schemas + `pi.registerTool` wiring; the
   existing vitest tests and `tsc --noEmit` stay green. **One deliberate behavior change:**
   the pre-existing `validateInput` `--`-separator bug (finding B51-1) is fixed, and new
   regression tests cover the `--`-separated tool argv so goto works and client-side
   validation actually enforces.
4. **Success Criterion 4** — the refactored Pi extension reloads and drives
   goto→snapshot **through the `~/.omp/plugins` symlink load path**, core resolvable
   in that `node_modules`, not only from the repo checkout.
5. **CI-01** — CI runs a **blocking** `tsc --noEmit` + tests gate (core + Pi) on
   push/PR, **proven by a deliberately failing test blocking the job**.

<decisions>
## Implementation Decisions

### ExecFn / ExecResult contract (Area 1 — user-ratified)
- **D-01:** The injected `ExecFn` is **binary-bound**: `type ExecFn = (args: string[], opts: ExecOptions) => Promise<ExecResult>`. The core hands argv + opts only; the adapter has already bound the rod-cli binary into its closure. The core is fully agnostic of *how* the process is launched.
- **D-02:** `ExecResult = { stdout: string; stderr: string; code: number }` and `ExecOptions = { signal?: AbortSignal; timeout?: number }`. Field name is `code` (matches current Pi). Each **adapter's** `execFn` normalizes its host's shape into this (Pi's `pi.exec` already returns `code`; OMP's `code`-vs-`exitCode` divergence is normalized in the OMP adapter in Phase 52 — the whole point of the injected contract).
- **D-03:** For the actual exec, the adapter binds the **resolved path from `findRodCli()`**, falling back to bare `"rod-cli"` if `findRodCli()` returns `null`. Binary resolution becomes purely the adapter's concern; `findRodCli`'s result stops being lifecycle-only. (More robust than today's always-`"rod-cli"`: honors `ROD_CLI_PATH`/`GOBIN`/`$HOME/go/bin`. Same binary resolves; this is an internal-resolution improvement, not a tool-output change.)
- **D-04:** **All exec orchestration is centralized in the core `execRodCli(execFn, args, opts)`**: it validates input, resolves the timeout, prepends `--raw`, calls `execFn`, and throws on non-zero exit. argv-builders stay pure and `--raw`-free (they emit e.g. `["goto", url]`). Error semantics (validation throw + `rod-cli exited N: <stderr>` throw) are moved verbatim.

### Core package layout & exports (Area 2)
- **D-05:** New package at **`extensions/core/`**, name `@agenthands/rod-cli-core`, raw-TS (`"type": "module"`, `main: "src/index.ts"`, no build). Mirrors `extensions/pi/` shape.
- **D-06:** Internal file split: `src/binary.ts` (`findRodCli` + platform constants), `src/exec.ts` (`ExecFn`/`ExecResult`/`ExecOptions` types + `execRodCli`), `src/subcommand.ts` (`findSubcommand`), `src/validate.ts` (`validateInput`), `src/timeouts.ts` (`TIMEOUTS` + `timeoutFor`), `src/argv.ts` (**13 named pure builders** — `buildGotoArgs`, `buildSnapshotArgs`, …; grouped in one module since each is 2–6 lines and independently testable).
- **D-07:** Public surface is a single barrel **`src/index.ts`** re-exporting `findRodCli`, `execRodCli`, the `ExecFn`/`ExecResult`/`ExecOptions` types, and all 13 argv-builders. The testable helpers (`validateInput`, `timeoutFor`, `findSubcommand`, `TIMEOUTS`) are exported from their modules for the core test suite.

### Local-dev resolution (Area 3 — Success Criterion 4, riskiest)
- **D-08:** Make the repo root an **npm workspace**: add `"private": true` + `"workspaces": ["extensions/*"]` to root `package.json`. A single root `npm install` symlinks `node_modules/@agenthands/rod-cli-{core,pi}` → the workspace dirs, so `extensions/pi` resolves `@agenthands/rod-cli-core` by walking up to the hoisted root `node_modules`. Because node realpaths the `~/.omp/plugins/.../rod-cli-pi` symlink to `extensions/pi`, the OMP load path resolves the core the same way as the checkout.
- **D-09:** Pi declares `"@agenthands/rod-cli-core": "*"` (workspace protocol resolves locally) under `dependencies`. **Verification is behavioral**, not structural: after the refactor, actually reload the extension through the live symlink and run `browse_goto` + `browse_snapshot` against a fixture. If node's realpath resolution ever fails to find the core through the symlink, the fallback is to also ensure `extensions/pi/node_modules/@agenthands/rod-cli-core` — but workspaces should make this unnecessary.

### Test split (Area 4)
- **D-10:** **Move to `extensions/core/src/__tests__/`** the logic tests: all 13 argv-builders (enum/positional/`--` separator), `validateInput` caps, `timeoutFor` resolution, `findSubcommand`, `findRodCli` discovery fallback, and `execRodCli` throw-on-non-zero **driven by a mock `ExecFn` (no `pi`)**. Most of `adversarial.test.ts` relocates here.
- **D-11:** **Stay in Pi:** tool-registration presence (13 registered), schema wiring, the `execute()` → argv-builder → `execRodCli` integration with a mock `execFn`, lifecycle hooks, the `--raw` boundary at the adapter, and the real-binary `integration.test.ts`.
- **D-12:** Each package owns its own `vitest.config.ts` with a **dir-scoped** `include: ["src/__tests__/**/*.test.ts"]`; runners never cross-pick. `@agenthands/rod-cli-pi`'s green-test count may shift as logic tests relocate — the invariant is **PI-01's behavior coverage stays green**, not a frozen literal "116".

### CI gate (Area 5)
- **D-13:** New workflow **`.github/workflows/extensions.yml`**, triggers `push` (main) + `pull_request`, alongside (not replacing) the Go `test.yml`. A matrix over `package: [core, pi]`: `actions/setup-node@v4` (Node 20 LTS), `npm ci` at repo root (installs the workspace), then `cd extensions/${package} && npm run typecheck && npm test`.
- **D-14:** The **Bun/OMP test lane is deferred to Phase 52** (the OMP package does not exist yet); CI-01 in this phase covers core + Pi on the Node/vitest lane. Phase 52 adds the `bun:test` lane for OMP.
- **D-15:** **Gate-blocks proof (Success Criterion 5):** demonstrate by running the exact CI command set locally on the green tree (shows pass), then adding a deliberately failing core test and re-running the same command set to show a non-zero exit (gate would block), then reverting. Documented in SUMMARY. (Avoids pushing a knowingly-broken commit; the command set *is* what CI runs.)

### Bugfix folded in — finding B51-1 (user-ratified 2026-07-05)
- **D-16:** The extraction FIXES the pre-existing `validateInput` bug: since Phase 49 (`9f5d8bd`) added the `--` injection separator to every tool's argv, `validateInput` has been reading `"--"` instead of the real argument — so `browse_goto` always threw (`got: --`) and click/type/fill/eval client-side validation was silently defeated (validating `"--"`). The core's `validateInput`/`findSubcommand` MUST locate the real argument **past** the `--` separator. New core regression tests assert the `--`-separated argv for every validated tool (goto http/https, click/type/fill non-empty, eval 10KB cap). This is the sole intended observable behavior change; everything else is byte-identical.

### the agent's Discretion
- Exact argv-builder function names and the internal helper module boundaries (kept close to the file split in D-06) are the planner/engineer's to finalize.
- Whether `integration.test.ts` gains a host-free core sibling (real binary via a `child_process` ExecFn) is left to execution judgment.

</decisions>

<disciplines>
## Lenses surfaced for this phase

> **Lenses, not rules.** Downstream agents adopt, refine, or reason past these. The hard,
> non-overridable requirements of this phase (zero-behavior-change, host-agnostic core,
> pure argv-builders, preserved `--` injection separator) are **constraints, not lenses** —
> they belong in the PLAN's `must_haves`, and are recorded there, not here.

~~~yaml
disciplines:
  - section: "inline"
    name: "test-locality"
    scope: ["extensions/core/src/__tests__/**", "extensions/pi/src/__tests__/**"]
    guidance: "A test lives with the code it exercises: logic that moved to the core is tested in the core (no pi object); wiring/registration/lifecycle is tested in the adapter. Prefer the package whose contract the test actually defends."
    rationale: "The split is the mechanism that keeps the core host-free and lets each package's runner stay dir-scoped; putting a test in the wrong package silently re-couples the core to a host or leaves a moved behavior untested."
    override_signals:
      - "A genuine end-to-end integration test that spans the argv-build AND the real-binary shell-out may reasonably live in either package (or both) — locality is ambiguous when the test defends the seam itself."
      - "A regression test pinned to a specific historical bug may stay co-located with the original reproduction for provenance even after the code moves."
~~~

</disciplines>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap (the contract)
- `.planning/REQUIREMENTS.md` — CORE-01, CORE-02, PI-01, CI-01 (verbatim acceptance criteria) + "Architecture decision (locked)" + "Dependencies & Assumptions" (live-dev symlink note).
- `.planning/ROADMAP.md` §"Phase 51" — the 5 Success Criteria (what must be TRUE).
- `.planning/ROADMAP.md` §"Records" — the red-team fold: **B1** (core is its own package, no bundle/copy), **M3** (normalized ExecFn contract), **M4** (CI-01 pulled to Phase 51), **M5** (single core test runner), **M6** (live-symlink reload criterion).
- `.planning/STATE.md` §"v2.3 Decisions" — three-package architecture, `ext-v*` publishing (context for later phases).

### Code being extracted / refactored
- `extensions/pi/src/cli.ts` — the exact logic to extract: `findRodCli`, `findSubcommand`, `TIMEOUTS`/`timeoutFor`, `validateInput`, `ExecResult`/`ExecOptions`, `execRodCli` (the `_pi` singleton is what's being removed).
- `extensions/pi/src/tools/*.ts` — the 13 argv patterns (`eval.ts` = `--` separator + positional; `scroll.ts` = `StringEnum` enum + numeric default + mousewheel mapping; etc.).
- `extensions/pi/src/{index,types,lifecycle}.ts` — the factory, `SessionParam`, and the session_start/shutdown hooks (unchanged by this phase, but their imports of the moved logic must be rewired).
- `extensions/pi/package.json`, `vitest.config.ts`, `tsconfig.json` — the raw-TS package shape to mirror in the core.

### Codebase map
- `.planning/codebase/ARCHITECTURE.md` §8 "Pi Extension (`extensions/pi/`)" — the authoritative description of binary resolution, the shell-out wrapper, validation, timeouts, the trust boundary, and the `--`-separator injection protection (a constraint to preserve).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets (moving to core, near-verbatim)
- `findRodCli()` (`cli.ts:21-57`): 3-tier discovery (`ROD_CLI_PATH` > `PATH` > `GOBIN`/`$HOME/go/bin`). Returns path or `null`. → `core/src/binary.ts`.
- `findSubcommand()` (`cli.ts:72-89`): skips `-s`/`--session <value>` and flags. → `core/src/subcommand.ts` (shared by validate + timeout).
- `TIMEOUTS` + `timeoutFor()` (`cli.ts:94-123`): per-subcommand table, `DEFAULT_TIMEOUT` 30s. → `core/src/timeouts.ts`.
- `validateInput()` (`cli.ts:129-161`): goto URL scheme, non-empty selector for click/fill/type, non-empty fill text, eval 10KB cap. → `core/src/validate.ts`.
- `execRodCli()` (`cli.ts:191-211`): the orchestration to re-shape around the injected `ExecFn` (drop `_pi`, keep `--raw` prepend + timeout + throw). → `core/src/exec.ts`.

### Established Patterns (preserve)
- **`--` separator** between flags and user-provided values in `eval`/`type`/`fill` argv (I6 injection protection). argv-builders MUST keep emitting it. **This is a constraint → PLAN `must_haves`.**
- Session flag threading: `args.unshift("-s", params.session)` when `params.session` set — becomes part of each pure argv-builder's output.
- Enum tools use `StringEnum` (pi-ai) — these stay in the **adapter** schema; the core argv-builder receives the already-narrowed string.

### Integration Points
- `extensions/pi/src/index.ts` factory: currently `setPi(pi)` then registers. New shape: the adapter builds `execFn` = `(args, opts) => pi.exec(rodCliPath ?? "rod-cli", args, opts)` once, and each tool's `execute()` becomes `execRodCli(execFn, buildXArgs(params), {signal})`.
- Root `package.json` gains `private` + `workspaces` (D-08) — additive; existing `terser` dev script/dep untouched.

</code_context>

<specifics>
## Specific Ideas

- The `--raw` prepend is currently a documented no-op for every command this extension issues (see `cli.ts:178-190`); keep it centralized in core `execRodCli` for exact behavioral parity and forward-compat, per D-04.
- Do not freeze the literal "116 tests" — the number shifts as logic tests relocate to the core; the real invariant is that Pi's behavioral coverage + `tsc --noEmit` stay green (D-12).

</specifics>

<deferred>
## Deferred Ideas

- **LICENSE file + `publishConfig`/`repository`/`engines`/`files` accuracy** — Phase 53 (CORE-03, PI-02). Not required for the refactor to typecheck/test.
- **OMP adapter + Bun test lane** — Phase 52 (OMP-01..04); the ExecFn contract cut here is what OMP consumes.
- **`ext-v*` auto-publish workflow** — Phase 53 (CI-02/CI-03). This phase lands only the *gate*, not the publish job.

</deferred>

## Records

~~~yaml
finding:
  record_author: architect
  accountable_owner: architect
  id: B51-1
  severity: high
  subject_ref: extensions/pi/src/cli.ts validateInput
  statement: "Since Phase 49 (9f5d8bd) added the -- injection separator to every tool's argv, validateInput reads rest[0] == '--' instead of the real argument. Net effect on shipped v2.2: browse_goto always throws ('got: --'); browse_click/type/fill_form non-empty checks and browse_eval's 10KB cap are silently defeated. The 116 tests miss it because none exercise the real tool argv (with '--') through validateInput."
  evidence:
    - "Probe: execRodCli(['goto','--','https://example.com']) with a mock pi THROWS before pi.exec is called."
    - "git: -- separator added in 9f5d8bd (phase 49); validateInput written in c30a0c7/2e8973c (phase 47/48), no -- handling."
    - "grep: no -- handling in extensions/pi/src/cli.ts."
  status: open
~~~

~~~yaml
decision:
  record_author: architect
  accountable_owner: architect
  id: D51-BUGFIX
  subject_ref: B51-1
  statement: "Fix B51-1 as part of the Phase 51 extraction: core validateInput/findSubcommand locate the real argument past the -- separator, with new core regression tests on the --separated argv for every validated tool. CONTEXT's 'zero behavior change' is amended to 'zero behavior change except this documented bugfix'."
  options_considered:
    - "Fix as part of the extraction (chosen)"
    - "Preserve exact behavior; file a separate bug"
    - "Fix in a dedicated pre-commit before the refactor"
  rationale: "Shipping a broken entry point (browse_goto) contradicts the milestone goal of a consumable extension; 'zero behavior change' protects working behavior, not a broken path. The literal PI-01 gate (tests + tsc green) stays satisfied."
  ratified_by: co-driver
  status: locked
~~~

---

*Phase: 51-shared-core-extraction-pi-refactor-ci-gate*
*Context gathered: 2026-07-05*
