---
author: architect
responsible: architect
phase: 51
status: ready
parent_artifacts:
  - .planning/phases/51/CONTEXT.md
  - .planning/REQUIREMENTS.md (CORE-01, CORE-02, PI-01, CI-01)
  - .planning/ROADMAP.md (Phase 51 Success Criteria)
---

# Phase 51: Shared Core Extraction + Pi Refactor + CI Gate — PLAN

## {P} Precondition
`extensions/pi/` is the shipped v2.2 extension (13 tools, 116 green vitest tests, `tsc --noEmit`
green). `execRodCli` closes over a `_pi` singleton. Root `package.json` is not a workspace.
`rod-cli` is at `~/bin/rod-cli` (in PATH). Toolchain Node 24 / npm 11 / Bun 1.3.14. Tree clean.

## {Q} Postcondition
`@agenthands/rod-cli-core` exists at `extensions/core/` (host-agnostic, no schema-lib runtime
imports); `@agenthands/rod-cli-pi` consumes it via the workspace; the injected binary-bound
`ExecFn` contract is the sole exec seam; `browse_goto` works and validation enforces (B51-1
fixed); all Pi + core tests + `tsc --noEmit` are green; the refactored Pi reloads through the
`~/.omp/plugins` symlink and drives goto→snapshot; a blocking `.github/workflows/extensions.yml`
runs `tsc --noEmit` + vitest for core + Pi, proven to block on a deliberately failing test.

## must_haves.truths (constraints — non-overridable, verified at the gate)
1. **Host-agnostic core:** `grep -rn "typebox\|StringEnum\|pi-coding-agent\|pi-ai\|pi\.zod" extensions/core/src` returns **nothing**. The core imports only Node built-ins.
2. **Pure argv-builders:** each of the 13 builders is `(params) → string[]` with no I/O, no `pi`, no exec. (Builders MAY throw synchronously on invalid params — preserves current pre-exec throw timing.)
3. **`--` injection separator preserved:** every builder that emits user-controlled positional values keeps the `--` separator exactly where the current tool places it (see the argv table). This is the I6 protection — do not drop or reorder it.
4. **B51-1 fixed (the one intended behavior change):** `validateInput` locates the real argument **past** a leading `--` in the post-subcommand args, so `browse_goto` works and click/type/fill/eval validation inspects the real value. New regression tests assert the `--`-separated argv for every validated tool.
5. **Everything else byte-identical:** argv emitted to `rod-cli` (modulo `--raw` prepend, unchanged), timeout table, error messages, session threading, `--raw` prepend — all preserved verbatim.
6. **Green gate:** `tsc --noEmit` + `vitest run` green for **both** packages; the pre-existing Pi behavioral coverage stays green (test count MAY shift as logic tests relocate).

---

## §1 — Root workspace (D-08)

Edit root `package.json` (additive; keep the existing `terser` dev script + dependency):
- add `"private": true`
- add `"workspaces": ["extensions/*"]`

Then run `npm install` at repo root → creates `node_modules/@agenthands/rod-cli-{core,pi}` symlinks to the workspace dirs. (Core dir must exist first; do §2 before the install, or run install after §2/§5.)

## §2 — Core package scaffold (D-05)

Create `extensions/core/`:

**`extensions/core/package.json`:**
```json
{
  "name": "@agenthands/rod-cli-core",
  "version": "0.1.0",
  "description": "Runtime-agnostic core for rod-cli host extensions — binary discovery, normalized exec contract, input validation, timeout table, and pure argv-builders for the browse_* tools",
  "license": "MIT",
  "type": "module",
  "main": "src/index.ts",
  "keywords": ["browser-automation", "rod-cli"],
  "files": ["src/**/*.ts", "!src/__tests__/**", "README.md", "LICENSE"],
  "scripts": {
    "test": "vitest run",
    "typecheck": "tsc --noEmit"
  },
  "devDependencies": {
    "@types/node": "^22.0.0",
    "typescript": "^5.8.0",
    "vitest": "^3.0.0"
  }
}
```
(No runtime `dependencies` — the core is host-agnostic. LICENSE/publishConfig/repository are Phase 53, deferred.)

**`extensions/core/tsconfig.json`:** copy `extensions/pi/tsconfig.json` verbatim (same strict flags, `include: ["src"]`).

**`extensions/core/vitest.config.ts`:** copy `extensions/pi/vitest.config.ts` verbatim (`include: ["src/__tests__/**/*.test.ts"]` — dir-scoped, M5).

## §3 — Core modules (D-06, D-07)

Move logic from `extensions/pi/src/cli.ts` (verbatim except the B51-1 fix and the `_pi`→`ExecFn` reshape).

**`src/binary.ts`** — platform constants (`IS_WINDOWS`, `BINARY_NAME`, `PATH_SEP` from `node:os`/`node:path`) + `findRodCli(): string | null` (lines 21-57 of cli.ts, verbatim).

**`src/subcommand.ts`** — `findSubcommand(args: string[]): number` (lines 72-89, verbatim; skips `-s`/`--session <value>` and flags). Exported (used by validate + timeouts + tests).

**`src/timeouts.ts`** — `TIMEOUTS` table + `DEFAULT_TIMEOUT` + `timeoutFor(args): number` (lines 94-123, verbatim), importing `findSubcommand`.

**`src/validate.ts`** — `validateInput(args: string[]): void` (lines 129-161) **WITH the B51-1 fix**:
```ts
import { findSubcommand } from "./subcommand";

export function validateInput(args: string[]): void {
  const cmdIndex = findSubcommand(args);
  if (cmdIndex < 0) return;
  const cmd = args[cmdIndex];
  let rest = args.slice(cmdIndex + 1);
  if (rest[0] === "--") rest = rest.slice(1);   // B51-1: skip the injection separator
  // ...the existing goto / click|fill|type / fill-text / eval checks, UNCHANGED,
  //    now operate on the real positional args in `rest`.
}
```
Keep every existing check and error message string identical.

**`src/exec.ts`** — the contract + orchestration (D-01/D-02/D-04):
```ts
export interface ExecResult { stdout: string; stderr: string; code: number; }
export interface ExecOptions { signal?: AbortSignal; timeout?: number; }
export type ExecFn = (args: string[], opts?: ExecOptions) => Promise<ExecResult>;

import { validateInput } from "./validate";
import { timeoutFor } from "./timeouts";

export async function execRodCli(execFn: ExecFn, args: string[], opts?: ExecOptions): Promise<ExecResult> {
  validateInput(args);
  const timeout = timeoutFor(args);
  const result = await execFn(["--raw", ...args], { signal: opts?.signal, timeout });
  if (result.code !== 0) {
    const stderr = result.stderr?.trim() || "(no stderr)";
    throw new Error(`rod-cli exited ${result.code}: ${stderr}`);
  }
  return { stdout: result.stdout, stderr: result.stderr, code: result.code };
}
```
(No `_pi`, no `setPi`. `--raw` prepend + throw semantics preserved verbatim.)

**`src/wait.ts`** — the wait orchestration (browse_wait is not a single-shot):
```ts
export function sleep(ms: number, signal?: AbortSignal): Promise<void> { /* cli.ts wait.ts sleep(), verbatim */ }
export function buildWaitPollArgs(p: { selector: string; session?: string }): string[] {
  return withSession(p.session, ["eval", "--", `!!document.querySelector(${JSON.stringify(p.selector)})`]);
}
export async function waitForSelector(execFn: ExecFn, p: { selector: string; timeoutMs: number; session?: string }, opts?: { signal?: AbortSignal }): Promise<void> {
  const deadline = Date.now() + p.timeoutMs;
  while (Date.now() < deadline) {
    const r = await execRodCli(execFn, buildWaitPollArgs({ selector: p.selector, session: p.session }), { signal: opts?.signal });
    if (r.stdout.trim() === "true") return;
    await sleep(500, opts?.signal);
  }
  throw new Error(`Timed out waiting ${p.timeoutMs}ms for selector: ${p.selector}`);
}
```
(Poll interval 500ms, script `!!document.querySelector(<json>)`, timeout message — all verbatim from the current `wait.ts`.)

**`src/argv.ts`** — 13 pure builders + a `withSession` helper. Each returns argv **without** `--raw` (exec prepends it) but **with** the `--` separator and the `-s <session>` prefix exactly as the current tools produce:

```ts
export function withSession(session: string | undefined, args: string[]): string[] {
  return session ? ["-s", session, ...args] : args;
}
```

| Builder | params | argv (before `--raw`) |
|---|---|---|
| `buildGotoArgs` | `{url, session?}` | `withSession(session, ["goto", "--", url])` |
| `buildSnapshotArgs` | `{selector?, session?}` | `withSession(session, ["snapshot"])` (selector accepted, ignored — no `--selector` in rod-cli snapshot; keep comment) |
| `buildClickArgs` | `{selector, doubleClick?, session?}` | `withSession(session, [doubleClick ? "dblclick" : "click", "--", selector])` |
| `buildTypeArgs` | `{selector, text, session?}` | `withSession(session, ["type", "--", selector, text])` |
| `buildEvalArgs` | `{expression, session?}` | `withSession(session, ["eval", "--", expression])` |
| `buildScreenshotArgs` | `{selector?, session?}` | `withSession(session, selector ? ["screenshot", "--selector", selector] : ["screenshot"])` |
| `buildScrollArgs` | `{direction?, distance?, selector?, session?}` | if `selector` → throw (same message as scroll.ts); else `dy = (direction ?? "down") === "down" ? (distance ?? 300) : -(distance ?? 300)` → `withSession(session, ["mousewheel", "--", "0", String(dy)])` |
| `buildTabsArgs` | `{action, url?, index?, session?}` | `list`→`["tab-list"]`; `new`→(url req, throw)`["tab-new","--",url]`; `close`→(index req, throw)`["tab-close","--",String(index)]`; `select`→`["tab-select","--",String(index)]`; then `withSession` |
| `buildNavigateArgs` | `{action, session?}` | map `reload→reload`, `back→go-back`, `forward→go-forward` (throw unknown) → `withSession(session, [cmd])` |
| `buildCookiesArgs` | `{action, name?, value?, session?}` | `get`→`["cookie-get"]`; `set`→(name+value req)`["cookie-set","--",name,value]`; `delete`→(name req)`["cookie-delete","--",name]`; `clear`→`["cookie-clear"]`; then `withSession` |
| `buildStorageArgs` | `{action, storageType?, key?, value?, session?}` | prefix `local→localstorage`/`session→sessionstorage` (throw unknown); `get`→ key? `[p-get,"--",key] : [p-get]`; `set`→(key+value req)`[p-set,"--",key,value]`; `delete`→(key req)`[p-delete,"--",key]`; `clear`→`[p-clear]`; then `withSession` |
| `buildFillFormArgs` | `{selector, text, submit?, session?}` | base `["fill"]`; if `submit` push `"--submit"`; push `"--", selector, text`; then `withSession` |
| `buildWaitPollArgs` | `{selector, session?}` | (in `src/wait.ts`) `withSession(session, ["eval","--",\`!!document.querySelector(${JSON.stringify(selector)})\`])` |

Copy every throw message verbatim from the corresponding current tool.

**`src/index.ts`** — barrel re-exporting: `findRodCli` (binary), `execRodCli` + `ExecFn`/`ExecResult`/`ExecOptions` (exec), the 13 builders + `withSession` (argv), `buildWaitPollArgs` + `waitForSelector` + `sleep` (wait). Also re-export the testable helpers `validateInput`, `timeoutFor`, `TIMEOUTS`, `findSubcommand`.

## §4 — Core tests (CORE-02) → `extensions/core/src/__tests__/`

Relocate from `extensions/pi/src/__tests__/adversarial.test.ts` the host-free suites and rewrite the exec suite to a **mock `ExecFn`** (no `setPi`/mock-`pi`):
- `binary.test.ts` — the full `findRodCli` suite (ROD_CLI_PATH/PATH/GOBIN priority, Windows USERPROFILE, totality, idempotence, pathological paths).
- `validate.test.ts` — the `validateInput` suite via `execRodCli(mockExecFn, argv)`, asserting a throw **before** `mockExecFn` is called. **Keep** the existing no-`--` cases AND **add B51-1 regression cases with the real tool argv:**
  - `execRodCli(fn, ["goto","--","https://x"])` → resolves (no throw), `fn` called.
  - `execRodCli(fn, ["goto","--","ftp://x"])` → throws `got: ftp://x` (not `got: --`).
  - `execRodCli(fn, ["click","--",""])` → throws non-empty-selector.
  - `execRodCli(fn, ["fill","--","#s",""])` → throws non-empty-text.
  - `execRodCli(fn, ["eval","--", "x".repeat(10001)])` → throws too-long; `["eval","--","x".repeat(10000)]` → ok.
  - `-s` prefixed variants: `["-s","admin","goto","--","https://x"]` → ok.
- `timeouts.test.ts` — `timeoutFor` resolution for each mapped subcommand + default + flags-only (`--version`).
- `exec.test.ts` — `execRodCli` prepends `--raw`, forwards timeout+signal, returns `{stdout,stderr,code}`, throws `rod-cli exited N: <stderr>` on non-zero (mock `ExecFn`).
- `argv.test.ts` — **all 13 builders**: enum branches (tabs/cookies/storage/navigate/scroll), positional + `--` (goto/click/type/eval/fill_form), numeric/default handling (scroll dy up/down/default), session prefix, and the missing-required throws.

## §5 — Pi refactor (PI-01)

- **Delete** `extensions/pi/src/cli.ts` (all logic moved to core).
- **`extensions/pi/package.json`:** add `"dependencies": { "@agenthands/rod-cli-core": "*" }`. Leave `peerDependencies`/`devDependencies` as-is (Phase 53 revisits dep hygiene).
- **`extensions/pi/src/index.ts`** (factory): build the binary-bound `execFn` once and thread it:
  ```ts
  import { findRodCli, execRodCli, type ExecFn } from "@agenthands/rod-cli-core";
  export default function (pi: ExtensionAPI) {
    const rodCliPath = findRodCli();
    const bin = rodCliPath ?? "rod-cli";
    const execFn: ExecFn = (args, opts) => pi.exec(bin, args, opts);
    registerLifecycle(pi, rodCliPath);
    registerAllCoreTools(pi, execFn);
    registerAllExtendedTools(pi, execFn);
  }
  ```
- **`extensions/pi/src/tools/*.ts`** (all 13) + **`tools/index.ts`**: each `registerBrowseX(pi, execFn: ExecFn)`; the `execute()` becomes `const result = await execRodCli(execFn, buildXArgs(params), { signal });` importing the builder + `execRodCli`/`ExecFn` from `@agenthands/rod-cli-core`. Keep the TypeBox/`StringEnum` **schemas**, `description`, `promptSnippet`, `promptGuidelines`, and the `return { content: [...], details: {} }` shape **verbatim**. `wait.ts` uses `waitForSelector`/`sleep` from core.
- **`extensions/pi/src/types.ts`** (SessionParam) and **`lifecycle.ts`** — unchanged (adapter-side; lifecycle keeps using `pi.exec` directly).

## §6 — Pi test split (D-10..D-12)

- **`smoke.test.ts`** — update: drop `../cli` import; verify `../index` default export, `../types` SessionParam, `../lifecycle` registerLifecycle, and that `@agenthands/rod-cli-core` resolves (import a builder).
- **`adversarial.test.ts`** — reduce to **adapter-side** coverage: `registerLifecycle` hook behavior (session_start version notify, session_shutdown `reason==="quit"` gate) with a mock `pi`, and tool-registration presence (all 13 `pi.registerTool` names) + per-tool `execute()` → argv via a **mock `execFn`** (assert the argv the tool passes to `execRodCli`, incl. the `--` separator). Host-free logic tests are gone (now in core §4).
- **`integration.test.ts`** — update imports to `@agenthands/rod-cli-core`; replace `setPi(createRealPiExec())` with a real `child_process` **`execFn`** (`(args,opts)=>` execFile `bin`), and call `execRodCli(execFn, argv, {signal})`. Keep the HTTP-fixture + real-binary cases (skipIf no rod-cli).

## §7 — CI gate (CI-01, D-13/D-14/D-15)

**`.github/workflows/extensions.yml`** (new, does not touch `test.yml`/`release.yml`):
```yaml
name: extensions
on:
  push:
    branches: [main]
  pull_request:
jobs:
  extensions:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        package: [core, pi]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 20
      - run: npm ci
      - run: npm run typecheck
        working-directory: extensions/${{ matrix.package }}
      - run: npm test
        working-directory: extensions/${{ matrix.package }}
```
(Bun/OMP lane deferred to Phase 52 — D-14. `npm ci` at root needs a root `package-lock.json` from the §1 workspace install — commit it.)

## §8 — Verification (the gate)

1. **Build+test both packages** (the exact CI command set):
   `npm ci` (root) → `cd extensions/core && npm run typecheck && npm test` → `cd extensions/pi && npm run typecheck && npm test`. Both green.
2. **Host-agnostic grep:** `grep -rn "typebox\|StringEnum\|pi-coding-agent\|pi-ai" extensions/core/src` → empty (truth #1).
3. **B51-1 proof:** the new core regression tests are present and green; and a direct check that `execRodCli(fn, ["goto","--","https://example.com"])` no longer throws.
4. **Symlink reload (Success Criterion 4):** ensure the workspace install populated the core symlink reachable from `extensions/pi`; reload the Pi extension through `~/.omp/plugins/node_modules/@agenthands/rod-cli-pi` and run a real `browse_goto` → `browse_snapshot` against a fixture (or a scratch `node -e` that imports the extension via the symlink path and asserts the core resolves + a builder runs). Record the outcome.
5. **Gate-blocks proof (Success Criterion 5):** add a deliberately failing test to `extensions/core/src/__tests__/`, run `cd extensions/core && npm test` → assert non-zero exit (gate would block); then remove it and re-run → green. Document the two exit codes in SUMMARY.

## Execution order
1. §1 root workspace edit (defer the `npm install` to after §2/§5 so both dirs exist).
2. §2 core scaffold.
3. §3 core modules (binary → subcommand → timeouts → validate[+B51-1] → exec → wait → argv → index).
4. §4 core tests.
5. §5 Pi refactor (delete cli.ts; index; 13 tools + barrel; package.json dep).
6. §6 Pi test split.
7. `npm install` at root (creates lockfile + workspace symlinks).
8. §7 CI workflow.
9. §8 verification (build/test, grep, B51-1, symlink reload, gate-blocks proof).

## Files changed
| File | § | Status |
|---|---|---|
| `package.json` (root) | §1 | MODIFIED (private + workspaces) |
| `package-lock.json` (root) | §1/§7 | NEW (workspace lockfile) |
| `extensions/core/package.json` | §2 | NEW |
| `extensions/core/tsconfig.json` | §2 | NEW |
| `extensions/core/vitest.config.ts` | §2 | NEW |
| `extensions/core/src/{binary,subcommand,timeouts,validate,exec,wait,argv,index}.ts` | §3 | NEW |
| `extensions/core/src/__tests__/{binary,validate,timeouts,exec,argv}.test.ts` | §4 | NEW |
| `extensions/pi/src/cli.ts` | §5 | DELETED |
| `extensions/pi/package.json` | §5 | MODIFIED (core dep) |
| `extensions/pi/src/index.ts` | §5 | MODIFIED (execFn threading) |
| `extensions/pi/src/tools/*.ts` (13) + `tools/index.ts` | §5 | MODIFIED (builders from core) |
| `extensions/pi/src/__tests__/{smoke,adversarial,integration}.test.ts` | §6 | MODIFIED (split) |
| `.github/workflows/extensions.yml` | §7 | NEW |
