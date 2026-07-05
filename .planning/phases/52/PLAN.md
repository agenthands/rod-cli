---
author: architect
responsible: architect
phase: 52
status: ready
parent_artifacts:
  - .planning/phases/52/CONTEXT.md
  - .planning/research/omp-host-api.md
  - .planning/REQUIREMENTS.md (OMP-01..04)
---

# Phase 52: Native OMP Extension — PLAN

## {P} Precondition
`@agenthands/rod-cli-core` exists (raw-TS, workspace, exports findRodCli/execRodCli/ExecFn/13 builders/waitForSelector/sleep). `@oh-my-pi/pi-coding-agent` is installed (hoisted at `/home/john/node_modules`). Bun 1.3.14 present. Host API is source-verified in `.planning/research/omp-host-api.md`. Pi adapter is the surface to mirror.

## {Q} Postcondition
`@agenthands/rod-cli-omp` at `extensions/omp/` registers all 13 `browse_*` tools (argv from core builders, byte-identical to Pi) + 2 lifecycle hooks, schemas via injected `pi.zod`, guidance folded into `description`, `omp` manifest, `bun:test` green, `tsc --noEmit` strict clean, no runtime `typebox`/`StringEnum` import, ungated idempotent shutdown; loads in a real OMP session and drives goto→snapshot; CI runs the OMP bun lane blocking.

## must_haves.truths (constraints — verified at the gate)
1. **argv from core only.** Every tool's argv comes from a `@agenthands/rod-cli-core` builder; grep shows no argv literals (`"goto"`, `"mousewheel"`, `"cookie-get"`, …) in `extensions/omp/src`. If a tool appears to need bespoke argv → route UP as a core finding (do NOT fork argv into the adapter).
2. **No runtime schema-lib/host import.** `grep -rnE "from \"typebox\"|StringEnum|@earendil-works|from \"zod\"" extensions/omp/src` → empty. Schemas use injected `pi.zod`; the ONLY `@oh-my-pi/pi-coding-agent` import is `import type` (erased).
3. **Ungated idempotent shutdown.** `session_shutdown` handler calls `rod-cli close` unconditionally in a try/catch with no `reason` gate.
4. **Descriptions fold guidance.** Each tool's `description` names the tool and includes the Pi usage example + key guidance (since OMP has no `promptSnippet`/`promptGuidelines`).
5. **No `any`** (project `ts-no-any` + `ts-no-inline-cast-access` rules). Mock typed via an interface; a single `as unknown as ExtensionAPI` with a reason comment is the only permitted cast.
6. **Green gates.** `tsc --noEmit` (strict) clean; `bun test` green; CI bun lane added + blocking.

---

## §1 — Package scaffold (D-01/D-02/D-03/D-12)

Create `extensions/omp/`:

**`package.json`:**
```json
{
  "name": "@agenthands/rod-cli-omp",
  "version": "0.1.0",
  "description": "OMP (oh-my-pi) extension for rod-cli — native browser automation tools for the OMP coding agent",
  "license": "MIT",
  "type": "module",
  "main": "src/index.ts",
  "keywords": ["omp-extension", "browser-automation", "rod-cli"],
  "files": ["src/**/*.ts", "!src/__tests__/**", "README.md", "LICENSE"],
  "scripts": { "test": "bun test", "typecheck": "tsc --noEmit" },
  "omp": { "extensions": ["./src/index.ts"] },
  "dependencies": { "@agenthands/rod-cli-core": "*" },
  "devDependencies": {
    "@oh-my-pi/pi-coding-agent": "*",
    "@types/bun": "latest",
    "typescript": "^5.8.0"
  }
}
```
(No runtime schema-lib dep. `@oh-my-pi/pi-coding-agent` is dev-only/type-only — the host injects it at runtime. Verify the installed version's spec; if `"*"` won't resolve in the workspace, pin to the installed version.)

**`tsconfig.json`:** copy `extensions/core/tsconfig.json` (strict), add `"types": ["bun"]` (or rely on `@types/bun`) in compilerOptions so `bun:test` globals typecheck.

After scaffolding, run `npm install` at repo root to link the new workspace + install `@oh-my-pi/pi-coding-agent`/`@types/bun`.

## §2 — Entry factory (D-04) `src/index.ts`
```ts
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { findRodCli, type ExecFn } from "@agenthands/rod-cli-core";
import { registerLifecycle } from "./lifecycle";
import { registerAllTools } from "./tools";

export default function (pi: ExtensionAPI) {
  const rodCliPath = findRodCli();
  const bin = rodCliPath ?? "rod-cli";
  const execFn: ExecFn = (args, opts) => pi.exec(bin, args, opts);
  registerLifecycle(pi, rodCliPath);
  registerAllTools(pi, execFn);
}
```
(`pi.exec` returns `{stdout,stderr,code,killed}` — structurally assignable to `ExecResult`; if tsc complains about the extra `killed`, it is still assignable since extra props are allowed on a return value. If a nominal mismatch appears, wrap: `(args, opts) => pi.exec(bin, args, opts).then(r => ({ stdout: r.stdout, stderr: r.stderr, code: r.code }))`.)

## §3 — Lifecycle (D-09/D-10) `src/lifecycle.ts`
```ts
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";

export function registerLifecycle(pi: ExtensionAPI, rodCliPath: string | null) {
  const bin = rodCliPath ?? "rod-cli";
  pi.on("session_start", async (_event, ctx) => {
    if (!rodCliPath) {
      ctx.ui.notify("rod-cli not found. Install: go install github.com/agenthands/rod-cli@latest", "warning");
      return;
    }
    try {
      const result = await pi.exec(bin, ["--version"], { timeout: 5000 });
      ctx.ui.notify(`rod-cli ${result.stdout.trim() || "unknown"} ready`, "info");
    } catch {
      ctx.ui.notify("rod-cli found but --version failed. Check your installation.", "warning");
    }
  });
  pi.on("session_shutdown", async () => {
    // Ungated + idempotent: OMP's session_shutdown carries no reason (M1).
    try { await pi.exec(bin, ["close"], { timeout: 5000 }); } catch { /* best-effort */ }
  });
}
```

## §4 — Tools (D-05/D-06/D-07/D-08)

**`src/tools/index.ts`:** `registerAllTools(pi, execFn)` calls `registerBrowseGoto(pi, execFn)` … all 13 (same 13 as Pi).

**Per-tool shape** (each `src/tools/<name>.ts`):
```ts
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildGotoArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseGoto(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  pi.registerTool({
    name: "browse_goto",
    label: "Browse Goto",
    description: "<folded>",           // see D-08
    parameters: z.object({
      url: z.string().describe("Full URL to navigate to (https://...)"),
      session: z.string().optional().describe(SESSION_DESC),
    }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildGotoArgs({ url: params.url, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
```

**Schema translation table (Pi TypeBox → `pi.zod`):** `Type.Object`→`z.object`; `Type.String`→`z.string()`; `Type.Number`→`z.number()`; `Type.Boolean`→`z.boolean()`; `Type.Optional(x)`→`x.optional()`; `StringEnum([...])`→`z.enum([...])`; defaults→`.default(v)`; descriptions→`.describe(s)`. Build the schema INSIDE the register fn (`pi.zod` is injected, not importable at module scope).

**argv per tool** — call the matching core builder (identical to Pi), passing the params through:
`buildGotoArgs`, `buildSnapshotArgs`, `buildClickArgs`, `buildTypeArgs`, `buildEvalArgs`, `buildScreenshotArgs`, `buildScrollArgs`, `buildTabsArgs`, `buildNavigateArgs`, `buildCookiesArgs`, `buildStorageArgs`, `buildFillFormArgs`; **browse_wait** uses `waitForSelector(execFn, {selector, timeoutMs, session}, {signal})` + `sleep` (import from core), with output text identical to Pi (`Found selector: ${selector}` / `Waited ${timeoutMs}ms`). The scroll selector-guard throw is already in `buildScrollArgs` (core).

**D-08 folded `description`** (per tool): concatenate, as prose, the Pi tool's `description` + a sentence naming the tool + the Pi `promptSnippet` example + the key `promptGuidelines` bullets. Read each Pi tool file (`extensions/pi/src/tools/<name>.ts`) for the exact strings; do NOT invent new guidance. Enum params keep the same value sets as Pi (`z.enum(["list","new","close","select"])` etc.).

## §5 — bun:test suite (D-11) `src/__tests__/`

`mockapi.ts` — a typed mock factory:
```ts
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import * as z from "zod/v4";                 // real zod for the mock's injected .zod
interface RegisteredTool { name: string; execute: (id: string, params: Record<string, unknown>, signal?: AbortSignal) => Promise<{ content: Array<{ type: string; text: string }>; details?: unknown }>; }
interface MockApi { tools: RegisteredTool[]; handlers: Record<string, Array<(e: unknown, ctx: unknown) => unknown>>; execCalls: string[][]; notifications: Array<{ msg: string; level?: string }>; fire(event: string, e?: unknown): Promise<void>; api: ExtensionAPI; }
```
Build a mock whose `.zod = { z }`, `.typebox`, `.registerTool` pushes to `tools`, `.on` captures handlers, `.exec(cmd,args,opts)` records `args` in `execCalls` and returns a canned `{stdout:"ok",stderr:"",code:0,killed:false}`, `.logger` noop, and a `ctx` with `ui.notify` recording notifications. Expose `api` as `mock as unknown as ExtensionAPI` (single reason-commented cast). Note the mock's `.exec` is what the tools' argv flows through (execFn = `(args,opts)=>pi.exec(bin,args,opts)`), so `execCalls` captures `["--raw", ...builderArgv]`.

Test files:
- `registration.test.ts` — load `default(api)`; assert all 13 tool names present.
- `argv.test.ts` — for representative tools (goto/click/type/eval/fill_form/scroll + one enum each of tabs/cookies/storage/navigate), call `tool.execute("id", params, undefined)` and assert `execCalls` contains the expected argv incl. `--raw` prepend, `--` separator, `-s` session prefix, enum branch mapping. (Mirrors the core argv tests but through the OMP tool surface.)
- `lifecycle.test.ts` — fire `session_start` → assert a `--version` exec + an `info` notify; fire `session_shutdown` → assert a `["close"]` exec fired **unconditionally** (no reason passed).

Run: `cd extensions/omp && bun test`.

## §6 — CI bun lane (D-13)
Edit `.github/workflows/extensions.yml`:
- add `omp` to `matrix.package`.
- add a step (before the per-package run) `- uses: oven-sh/setup-bun@v2` with a recent bun.
- the existing matrix run `cd extensions/${{matrix.package}} && npm run typecheck && npm test` works for omp (its `npm test` → `bun test`), as long as bun is installed. (core/pi `npm test` → vitest, unaffected.)

## §7 — Verification (the gate)
1. **Build+test:** `npm ci` (root) → `cd extensions/omp && npm run typecheck` (0 errors, strict) → `bun test` (green). Also re-run core + pi gates (regression: no core change expected).
2. **Constraint greps:** argv-literals grep empty (truth #1); schema-lib/host-import grep empty except `import type` (truth #2); `any`/inline-cast grep empty (truth #5); `grep -n "reason" extensions/omp/src/lifecycle.ts` → no reason gate (truth #3).
3. **OMP-04 behavioral (Bun smoke, like Phase 51 SC4):** a temp Bun script that loads `extensions/omp/src/index.ts` (or via `omp plugin link` path), passes a mock ExtensionAPI with a REAL `pi.exec` (child_process to the resolved rod-cli) + real `pi.zod` (`import * as z from "zod/v4"`), fires `session_start`, invokes `browse_goto` against a local HTTP fixture then `browse_snapshot`, asserts the snapshot contains the fixture, fires `session_shutdown` and asserts the daemon closed. Delete the script after. Record output.
4. **OMP-04 real session:** `omp plugin link extensions/omp` (creates `~/.omp/plugins/node_modules/@agenthands/rod-cli-omp`), then confirm the 13 `browse_*` tools resolve/appear and a goto→snapshot round-trip works in an OMP context (attempt a fresh `omp -p` E2E; if a fresh session can't be driven from here, the Bun smoke in step 3 + successful `omp plugin link` resolution stands as the behavioral evidence, noted honestly).

## Execution order
1. §1 scaffold + root `npm install`.
2. §2 index, §3 lifecycle.
3. §4 tools (index + 13 files) — argv from core, schemas via pi.zod, folded descriptions from the Pi prose.
4. §5 bun tests.
5. §6 CI edit.
6. §7 verification (typecheck/bun test, greps, Bun behavioral smoke, omp plugin link).

## Files changed
| File | § | Status |
|---|---|---|
| `extensions/omp/package.json` | §1 | NEW |
| `extensions/omp/tsconfig.json` | §1 | NEW |
| `extensions/omp/src/index.ts` | §2 | NEW |
| `extensions/omp/src/lifecycle.ts` | §3 | NEW |
| `extensions/omp/src/tools/index.ts` + 13 `tools/*.ts` | §4 | NEW |
| `extensions/omp/src/__tests__/{mockapi,registration,argv,lifecycle}.test.ts` | §5 | NEW |
| `.github/workflows/extensions.yml` | §6 | MODIFIED (add omp + setup-bun) |
| `package-lock.json` | §1 | MODIFIED (workspace + deps) |
