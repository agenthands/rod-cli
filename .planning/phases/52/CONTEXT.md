---
author: architect
responsible: architect
phase: 52
phase_type: implementation
hard_bar: true
security_relevant: false
design_fork: false
status: locked
parent_artifacts:
  - .planning/REQUIREMENTS.md (OMP-01..04)
  - .planning/ROADMAP.md (Phase 52 Success Criteria + host-API verifications)
  - .planning/research/omp-host-api.md (source-verified OMP host API)
  - .planning/phases/51/SUMMARY.md (the core + ExecFn contract this consumes)
---

# Phase 52: Native OMP Extension — CONTEXT

**Gathered:** 2026-07-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Build `@agenthands/rod-cli-omp` — a native OMP (`@oh-my-pi/pi-coding-agent`) extension
on the shared `@agenthands/rod-cli-core`: all 13 `browse_*` tools with argv identical to
Pi (shared core), the two lifecycle hooks, parameter schemas via the injected `pi.zod`
(no runtime `typebox`/`StringEnum`), an `omp` manifest, and a `bun:test` suite — verified
loading in a real OMP session and driving goto→snapshot.

**In scope:** OMP-01..04 (the new adapter + its bun tests + real-session load) and adding
the OMP `bun:test` lane to CI (deferred from Phase 51).
**Out of scope:** publish-readiness/LICENSE/`npm pack` (Phase 53), docs (Phase 54), any
core change (the core is frozen for this phase — if a core gap appears, it routes UP as a
finding), any new browser capability.
</domain>

## {P} What the phase can assume (host API — source-verified)

All from `.planning/research/omp-host-api.md` (cited to `@oh-my-pi/pi-coding-agent` source):
- **exec contract is compatible with the core unchanged.** `pi.exec(command, args, {signal,timeout,cwd?})` → `ExecResult {stdout, stderr, code: number, killed}` (field is **`code`**, not `exitCode`) and **resolves on non-zero** (`allowNonZero`). This is structurally the core's `ExecResult {stdout,stderr,code}` and matches Pi's throw-on-non-zero handling. **No OMP-specific normalization needed** — the OMP `ExecFn` is the same `(args, opts) => pi.exec(bin, args, opts)` as Pi.
- **`ToolDefinition`** = `{name, label, description, parameters, hidden?, defaultInactive?, deferrable?, approval?, execute, onSession?, renderCall?, renderResult?}`. **No `promptSnippet`, no `promptGuidelines`** — fold both into `description`. `execute(toolCallId, params, signal, onUpdate, ctx)` → `AgentToolResult {content:(TextContent|ImageContent)[], details?, isError?, useless?}` (content is `[{type:"text",text}]`). The AbortSignal is the **3rd** positional arg. Throw (or `isError:true`) on failure.
- **Schemas:** `pi.zod` is `zod/v4` accessed as `const { z } = pi.zod` (`pi.zod.z.*`); `pi.typebox` is a zod-backed TypeBox shim (`pi.typebox.Type.*`, **no `StringEnum`**); `pi.arktype` also injected. Shipped examples use `pi.zod` (`z.object`, `z.enum([...])`).
- **Lifecycle:** `SessionStartEvent`/`SessionShutdownEvent` are `{type}` only — **`session_shutdown` has NO `reason` field** (no `"quit"`). Handlers get `(event, ctx)`; `ctx.ui.notify(message, "info"|"warning"|"error")`; `pi.logger` for logging.
- **Discovery/manifest:** the loader reads the package.json **`omp`** key (falls back to `pi`): `{ extensions?: string[], themes?, skills? }`. Entry convention: `export default function (pi: ExtensionAPI) { … }`. `pi.exec`'s command may be an absolute path.
- **Types** (`ExtensionAPI`, `ExtensionContext`) are exported from the package root `@oh-my-pi/pi-coding-agent` and are **type-only** (erased at runtime — the host injects the real object).
- The core (`@agenthands/rod-cli-core`) is published raw-TS with: `findRodCli`, `execRodCli`, `ExecFn`/`ExecResult`/`ExecOptions`, 13 argv-builders, `waitForSelector`, `sleep`. The repo is an npm workspace (`extensions/*`).

## {Q} What the phase must establish

1. **OMP-01** — `@agenthands/rod-cli-omp` consumes the core (real dep); imports `@oh-my-pi/pi-coding-agent` **type-only**; builds parameter schemas with **injected `pi.zod`** (no runtime `StringEnum`/`typebox` import); declares an `omp.extensions` manifest. `tsc --noEmit` (strict) passes with no excess-property error.
2. **OMP-02** — all 13 `browse_*` tools register with **argv mapping identical to Pi** (shared core builders) + the two lifecycle hooks. Prompt guidance folded into each `description` (tool named explicitly + Pi usage example). `session_shutdown` daemon-close is **behaviorally verified** to fire on a real OMP shutdown — **ungated + idempotent** (OMP's shutdown carries no reason).
3. **OMP-03** — a `bun:test` suite with a mock `ExtensionAPI` (exposing `.zod`/`.typebox`/`.registerTool`/`.on`/`.exec`) covers tool-registration presence, per-tool argv mapping (enum/positional), and lifecycle behavior; green under `bun test`.
4. **OMP-04** — the extension loads in a real OMP session (`omp plugin install`/`link`); the 13 `browse_*` tools appear in the tool catalog; a `browse_goto` + `browse_snapshot` round-trip drives a fixture page — becoming the OMP-native path (replacing the Pi-symlink).
5. **CI** — the OMP `bun:test` lane is added to `.github/workflows/extensions.yml` (the lane deferred from Phase 51), blocking.

<decisions>
## Implementation Decisions

### Package + wiring
- **D-01:** New package at **`extensions/omp/`**, name `@agenthands/rod-cli-omp`, raw-TS (`"type": "module"`, `main: "src/index.ts"`, no build), in the existing `extensions/*` workspace.
- **D-02:** `dependencies`: `@agenthands/rod-cli-core: "*"`. `devDependencies`: `@oh-my-pi/pi-coding-agent` (type-only, for `tsc`; the host injects it at runtime), `typescript`, `@types/bun` (or bun types) for the test lane. **No runtime `typebox`/`@earendil-works/*` import anywhere.**
- **D-03:** Manifest: `"omp": { "extensions": ["./src/index.ts"] }` in package.json.
- **D-04:** Entry `src/index.ts`: `export default function (pi: ExtensionAPI) { const bin = findRodCli() ?? "rod-cli"; const execFn: ExecFn = (args, opts) => pi.exec(bin, args, opts); registerLifecycle(pi, rodCliPath); registerAllTools(pi, execFn); }` — mirrors the Pi factory. `execFn` is the SAME shape as Pi (OMP exec is contract-compatible).

### Schemas (design decision — resolved)
- **D-05:** Author parameter schemas with **injected `pi.zod`** (`const { z } = pi.zod`): `z.object({...})`, `z.string()`, `z.number()`, `z.boolean()`, `z.enum([...])` for the 5 enum tools, `.optional()`/`.default(...)` as the Pi schemas had. This is OMP's canonical/idiomatic path (matches shipped examples), gives cleaner enums than TypeBox union-of-literals, and satisfies OMP-01's "no runtime `StringEnum`/`typebox` import" (`pi.zod` is injected, not imported). A shared `SessionParam` = `z.string().optional()` (described) mirrors Pi's `SessionParam`.
- **D-06:** The schema is the ONLY thing authored per-tool in the adapter. The argv comes from the **core builders** (`buildGotoArgs`, …, `buildFillFormArgs`, `buildWaitPollArgs`/`waitForSelector`) — byte-identical to Pi. No argv logic is re-implemented in the OMP adapter.

### Tool definitions
- **D-07:** Each tool: `{ name, label, description, parameters: z.object({...}), async execute(toolCallId, params, signal) { const result = await execRodCli(execFn, buildXArgs({...params}), { signal }); return { content: [{ type: "text", text: result.stdout }], details: {} }; } }`. Only `signal` (3rd arg) is used from the execute signature; `onUpdate`/`ctx` are ignored.
- **D-08:** **Fold `promptSnippet` + `promptGuidelines` into `description`** (OMP has no such fields). Each `description` = the Pi `description` + the tool named explicitly + the Pi `promptSnippet` usage example + the key guidance bullets, as prose. (OMP-02 acknowledges this is a second source of truth vs Pi's structured fields — accepted divergence.)

### Lifecycle
- **D-09:** `session_start`: verify the binary via `pi.exec(bin, ["--version"], {timeout:5000})`; `ctx.ui.notify(\`rod-cli \${version} ready\`, "info")` on success, a `"warning"` notify if missing/failed (mirrors Pi's lifecycle intent).
- **D-10:** `session_shutdown`: **unconditional + idempotent** — `try { await pi.exec(bin, ["close"], {timeout:5000}); } catch {}`. **No `reason` gate** (the event has no reason; a copied `reason==="quit"` gate would be a dead no-op — M1). This is the behavioral fix vs a naive Pi copy.

### Tests (bun:test)
- **D-11:** `src/__tests__/*.test.ts` run under **`bun test`** (OMP-03). A `mockExtensionAPI()` helper exposes `.zod` (real `zod/v4`), `.typebox`, `.registerTool` (captures defs), `.on` (captures handlers + a `_fire`), `.exec` (a mock capturing argv / returning a canned `{stdout,stderr,code,killed}`), `.logger`, and handlers receive a mock `ctx.ui.notify`. Cover: all 13 tools registered (names), per-tool argv via the captured `execFn` (enum branches, positional + `--`, session prefix), and lifecycle (session_start notify, session_shutdown fires `close` **unconditionally**). No `any` (project rule); type the mock with an interface + `as unknown as ExtensionAPI` (reason comment) at the single boundary.
- **D-12:** `package.json` scripts: `"typecheck": "tsc --noEmit"`, `"test": "bun test"`. `tsconfig.json` mirrors the core/Pi strict config (add `"types": ["bun"]` or `@types/bun` as needed).

### CI
- **D-13:** Add `omp` to the `.github/workflows/extensions.yml` matrix and a `oven-sh/setup-bun@v2` step to the job, so `omp`'s `npm test` (→ `bun test`) runs on the bun lane while `core`/`pi` stay on the Node/vitest lane. Blocking, alongside the Go job.

</decisions>

<disciplines>
## Lenses surfaced for this phase

~~~yaml
disciplines:
  - section: "inline"
    name: "core-is-the-single-source-of-argv"
    scope: ["extensions/omp/src/tools/**"]
    guidance: "The OMP adapter authors ONLY schema + registration + result-shaping. Every argv comes from a core builder; if a tool seems to need bespoke argv logic in the adapter, that is a signal the core is missing something — route it UP as a finding, do not fork the argv logic into the adapter."
    rationale: "The whole point of the shared core (Phase 51) is one argv source of truth across hosts; re-implementing argv in the OMP adapter silently recreates the drift the core exists to prevent."
    override_signals:
      - "A genuinely OMP-only surface with no Pi equivalent (none exists in this phase's 13-tool scope) would legitimately need adapter-local mapping."
      - "A host-API shape mismatch that the core contract cannot express (e.g. OMP needed a different exec signature) — but here it does not; exec is contract-compatible."
~~~

</disciplines>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Host API (source-verified — READ FIRST)
- `.planning/research/omp-host-api.md` — the definitive OMP host-API reference (exec contract, ToolDefinition/AgentToolResult, pi.zod/pi.typebox, lifecycle events, manifest, all cited to `@oh-my-pi/pi-coding-agent` source). This supersedes any assumption copied from Pi.
- `/home/john/node_modules/@oh-my-pi/pi-coding-agent/src/extensibility/extensions/types.ts` — `ExtensionAPI`, `ToolDefinition`, `ExtensionContext` (authoritative types).
- `/home/john/node_modules/@oh-my-pi/pi-coding-agent/examples/extensions/{api-demo,hello,tools}.ts` — real extension patterns (`export default function(pi)`, `const {z}=pi.zod`, `pi.on`, `ctx.ui.notify`).
- `/home/john/node_modules/@oh-my-pi/pi-coding-agent/examples/extensions/with-deps/package.json` — real `"omp": {"extensions": [...]}` manifest.

### Core + Pi (the shared contract + the surface to mirror)
- `extensions/core/src/index.ts` + `argv.ts` + `exec.ts` + `binary.ts` + `wait.ts` — the core API the OMP adapter consumes (findRodCli, execRodCli, ExecFn, 13 builders, waitForSelector, sleep).
- `extensions/pi/src/tools/*.ts` — the per-tool `description`/`promptSnippet`/`promptGuidelines` PROSE to fold into each OMP `description`, and the schema shapes to translate to `pi.zod` (StringEnum → `z.enum`).
- `extensions/pi/src/lifecycle.ts` — the session_start/shutdown intent to mirror (OMP shutdown is ungated).
- `extensions/pi/package.json`, `.github/workflows/extensions.yml` — the raw-TS package shape + CI to extend.

### Contract
- `.planning/REQUIREMENTS.md` (OMP-01..04), `.planning/ROADMAP.md` §"Phase 52" (Success Criteria + host-API verifications).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reused unchanged from the core (zero re-implementation)
- All 13 argv-builders + `withSession`, `execRodCli`, `findRodCli`, `waitForSelector`, `sleep`, `ExecFn`/`ExecResult`/`ExecOptions` — imported from `@agenthands/rod-cli-core`.

### Mirror-from-Pi (translate, don't copy verbatim)
- Tool `description`/`promptSnippet`/`promptGuidelines` prose → one folded `description` per tool (D-08).
- Schemas: `Type.Object`/`Type.String`/`Type.Optional`/`Type.Number`/`Type.Boolean` → `z.object`/`z.string`/`.optional()`/`z.number`/`z.boolean`; `StringEnum([...])` → `z.enum([...])`.
- Factory + lifecycle shape → mirror `extensions/pi/src/{index,lifecycle}.ts`, adjusting: type-only OMP import, `ctx.ui.notify` levels (`info|warning|error`), ungated shutdown.

### Divergences from Pi (host-driven — verified)
- `execute` signature `(toolCallId, params, signal, onUpdate, ctx)` (Pi: `(toolCallId, params, signal)`) — use `signal` only.
- No `promptSnippet`/`promptGuidelines` fields.
- `session_shutdown` ungated (no `reason`).
- Test runner `bun:test` (Pi: vitest).

</code_context>

<specifics>
## Specific Ideas
- The `ExecFn` contract cut in Phase 51 was validated by this phase's host research: OMP's `pi.exec` return shape (`code`, resolves-on-non-zero) matches the contract with zero adaptation. This is the payoff of M3.
- OMP-04's "real OMP session" bar: verify via (a) a Bun behavioral smoke that loads the extension through its manifest with a real-exec mock ExtensionAPI and drives goto→snapshot (deterministic, like Phase 51's SC4 proof), AND (b) `omp plugin link` + confirm catalog/round-trip in an OMP session where feasible.

</specifics>

<deferred>
## Deferred Ideas
- LICENSE/`publishConfig`/`repository`/`engines`/`files` accuracy + `npm pack` host-LOAD — Phase 53 (CORE-03/PI-02/PI-03 + the OMP tarball load).
- `ext-v*` auto-publish of all three packages — Phase 53 (CI-02/CI-03).
- READMEs / three-path docs — Phase 54.

</deferred>

---

*Phase: 52-native-omp-extension*
*Context gathered: 2026-07-05*
