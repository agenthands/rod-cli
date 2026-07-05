# OMP Host API Reference — building `@agenthands/rod-cli-omp`

Host: `@oh-my-pi/pi-coding-agent`, source at `/home/john/node_modules/@oh-my-pi/pi-coding-agent/src`.
All citations are `path:line` into that tree. **Source is truth.**

---

## 1. exec contract — `ExecOptions` / `ExecResult`

`src/exec/exec.ts:9-51`
```ts
export interface ExecOptions {
	signal?: AbortSignal;   // AbortSignal to cancel
	timeout?: number;       // ms
	cwd?: string;           // working dir
}
export interface ExecResult {
	stdout: string;
	stderr: string;
	code: number;           // <-- `code`, NOT `exitCode`
	killed: boolean;        // true when aborted (result.exitError?.aborted)
}
export async function execCommand(command, args, cwd, options?): Promise<ExecResult> {
	const result = await ptree.exec([command, ...args], {
		cwd, signal: options?.signal, timeout: options?.timeout,
		allowNonZero: true,   // <-- does NOT throw on non-zero exit
		allowAbort: true, stderr: "full",
	});
	return { stdout, stderr, code: result.exitCode ?? 0, killed: Boolean(result.exitError?.aborted) };
}
```
- Field is **`code`** (a `number`), never `exitCode`. Also `stdout`, `stderr`, `killed`.
- **Non-zero exit RESOLVES** (does not throw) — `allowNonZero: true`. You must inspect `result.code` yourself.
- The `pi.exec` bound on the ExtensionAPI wraps this: `src/extensibility/extensions/loader.ts:223` →
  `exec(command, args, options) { return execCommand(command, args, options?.cwd ?? this.cwd, options); }`

## 2. session events — `SessionShutdownEvent` / `SessionStartEvent`

`src/extensibility/shared-events.ts:28-30` and `:95-97`
```ts
/** Fired on initial session load */
export interface SessionStartEvent { type: "session_start"; }

/** Fired on process exit (SIGINT/SIGTERM) */
export interface SessionShutdownEvent { type: "session_shutdown"; }
```
- `SessionShutdownEvent` has **only** `type: "session_shutdown"`. **NO `reason` field.** No 'quit' / no enum of reasons exists.
- (Contrast: `SessionBeforeSwitchEvent`/`SessionSwitchEvent` DO carry `reason: "new"|"resume"|"fork"` — `shared-events.ts:36,45` — but shutdown does not.)
- `SessionStartEvent` is likewise just `{ type: "session_start" }`.
- Cleanup gate implication: shutdown carries no discriminator, so make teardown idempotent and unconditional.

## 3. `ToolDefinition` + `AgentToolResult`

`src/extensibility/extensions/types.ts:428-472`
```ts
export interface ToolDefinition<TParams extends TSchema = TSchema, TDetails = unknown> {
	name: string;
	label: string;         // UI label
	description: string;   // LLM-facing
	parameters: TParams;   // Zod OR TypeBox-shim schema
	hidden?: boolean;
	defaultInactive?: boolean;
	deferrable?: boolean;
	approval?: ToolApproval;         // "read" | "write" | "exec" (default "exec")
	mcpServerName?: string; mcpToolName?: string;
	execute(
		toolCallId: string,
		params: Static<TParams>,
		signal: AbortSignal | undefined,
		onUpdate: AgentToolUpdateCallback<TDetails> | undefined,
		ctx: ExtensionContext,          // <-- 5th arg is the ExtensionContext
	): Promise<AgentToolResult<TDetails>>;
	onSession?(event, ctx): void | Promise<void>;
	renderCall?(...); renderResult?(...);
}
```
Return type `AgentToolResult` — `@oh-my-pi/pi-agent-core/dist/types/types.d.ts:479-485`
```ts
export interface AgentToolResult<T = any, _TInput = unknown> {
	content: (TextContent | ImageContent)[];   // e.g. [{ type: "text", text }]
	details?: T;
	isError?: boolean;   // non-throwing failure flag
	useless?: boolean;   // compaction may elide
}
```
- `content` is an array of `{type:"text", text}` (or image) blocks. `details` is optional arbitrary payload.
- **`promptSnippet` and `promptGuidelines` DO NOT EXIST** on `ToolDefinition` (grep confirmed zero hits in types.ts). Fold both into `description`.
- Error convention (`pi-agent-core/README.md:312`): **throw** on failure, or set `isError:true`; don't return error text as normal content.
- Note arg ORDER differs from legacy Pi custom-tools (`custom-tools/hello/index.ts` uses `(id, params, onUpdate, ctx, signal)`), but the **extension** `registerTool` signature is `(toolCallId, params, signal, onUpdate, ctx)` — use the extension order.

## 4. schemas — `pi.typebox`, `pi.zod`, enums

`src/extensibility/extensions/types.ts:956-962`
```ts
typebox: typeof TypeBox;   // injected zod-backed TypeBox shim (legacy Type.Object authoring)
arktype: typeof arktype;   // canonical going forward
zod: typeof zod;           // zod/v4  (import * as zod from "zod/v4"  — types.ts:36)
```
TypeBox shim `Type.*` surface — `src/extensibility/typebox.ts:915-935`:
```
Type = { String, Number, Integer, Boolean, Any, Unknown, Never, Literal, Union,
         Intersect, Enum, Array, Tuple, Object, Record, Optional, Nullable,
         Readonly, Partial, ... }
```
- Full builder set present: `Object/String/Number/Boolean/Optional/Union/Literal/Array` all exist.
- **Enum options:** two paths, both real:
  - `Type.Union([Type.Literal("a"), Type.Literal("b")])` → emits `{anyOf:[{const...}]}` (`typebox.ts:590-598, 584-587`).
  - `Type.Enum(["a","b"])` → `tEnum` collapses to a literal-union (`typebox.ts:627-643`). There is **no `StringEnum` export**.
  - Access as `pi.typebox.Type.*` (shim exports `Type`).
- **zod path (what examples actually use):** `const { z } = pi.zod;` then `z.enum([...])`:
  - `examples/extensions/api-demo.ts:11,20-22` → `parameters: z.object({ message: z.string()..., logLevel: z.enum(["error","warn","debug"]).default("debug") })`
  - `examples/extensions/hello.ts:8-9` → `const { z } = pi.zod`
  - `examples/custom-tools/hello/index.ts:7-8` → `parameters: pi.zod.object({ name: pi.zod.string()... })`
- Reusing existing Pi `Type.Object(...)` TypeBox schemas works unchanged via `pi.typebox.Type`.

## 5. notify / context / handler args

`ExtensionContext` — `src/extensibility/extensions/types.ts:333-335`
```ts
export interface ExtensionContext {
	ui: ExtensionUIContext;   // UI methods
	getContextUsage(): ...; sessionManager; newSession(); switchSession(); reload(); ...
}
```
`ExtensionUIContext.notify` — `types.ts` (ExtensionUIContext block):
```ts
/** Show a notification to the user. */
notify(message: string, type?: "info" | "warning" | "error"): void;
```
- So `ctx.ui.notify(msg, "info"|"warning"|"error")`. Levels are **`info|warning|error`** (NOT the Pi `level` string — verify names).
- Handler signature — `types.ts:943`:
  ```ts
  export type ExtensionHandler<E, R = undefined> = (event: E, ctx: ExtensionContext) => Promise<R|void> | R | void;
  ```
  i.e. `pi.on("session_start", (event, ctx) => {...})` — handler gets **`(event, ctx)`**, ctx exposes `ctx.ui.notify`.
  Example: `examples/extensions/tools.ts:124` → `pi.on("session_start", async (_event, ctx) => {...})`.
- `pi.on` overloads incl. `session_start` (`types.ts:972`) and `session_shutdown` (`types.ts:989`).
- Extension-level logging also available via `pi.logger` (`types.ts:953-954`, `typeof PiLogger`).

## 6. discovery / manifest

`src/extensibility/extensions/loader.ts:361-394`
```ts
interface ExtensionManifest { extensions?: string[]; themes?: string[]; skills?: string[]; }
const pkg = ... as { omp?: ExtensionManifest; pi?: ExtensionManifest };
const manifest = pkg.omp ?? pkg.pi;          // <-- reads "omp" key, falls back to "pi"
...
if (manifest?.extensions?.length) { for (const extPath of manifest.extensions) { resolve(dir, extPath) ... } }
```
Real example — `examples/extensions/with-deps/package.json`:
```json
{ "name": "pi-extension-with-deps", "type": "module",
  "omp": { "extensions": ["./index.ts"] },
  "dependencies": { "ms": "^2.1.3" } }
```
- **Manifest key: `omp.extensions` (array of module paths); legacy `pi.extensions` also accepted.** Use `omp`.
- Discovery rules (`loader.ts:440-445`): direct `extensions/*.ts`, or `<ext>/index.ts`, or `<ext>/package.json` with `omp`/`pi` field. One level, no recursion — complex packages must use the package.json manifest.
- **Entry convention: `export default function (pi: ExtensionAPI) { ... }`** — the module's default export is a factory receiving the ExtensionAPI (`examples/extensions/{hello,api-demo,tools}.ts` all `export default function toolsExtension(pi)`). Inside it you call `pi.registerTool(...)` and `pi.on(...)`.

## 7. exec + binary

`pi.exec(command, args, options)` — `src/extensibility/extensions/types.ts:1108`
```ts
exec(command: string, args: string[], options?: ExecOptions): Promise<ExecResult>;
```
- **First arg is the command/binary name.** Bound impl (`loader.ts:223-225`) forwards straight to `ptree.exec([command, ...args], ...)`.
- It is passed verbatim to ptree/`spawn`, so an **absolute path** (e.g. `/home/john/bin/rod-cli`) works as `command`; a bare name resolves via PATH.

---

## IMPLICATIONS FOR rod-cli-omp
- **Package manifest:** ship `"omp": { "extensions": ["./index.ts"] }` in package.json; entry file `export default function (pi: ExtensionAPI) {…}`. Register all 13 `browse_*` tools + `pi.on` inside it.
- **Reuse existing TypeBox schemas** unchanged via `pi.typebox.Type.*` (Object/String/Union/Literal/Optional/Array all present). Enum params: keep `Type.Union([Type.Literal(...)])` — there is no `StringEnum`. Or migrate to `pi.zod` (`const {z}=pi.zod; z.enum([...])`) which is `zod/v4` and is what the shipped examples use.
- **Fold `promptSnippet` + `promptGuidelines` into `description`** — those fields do not exist on OMP `ToolDefinition`. Keep `name`, `label`, `description`, `parameters`, `execute`.
- **execute signature is `(toolCallId, params, signal, onUpdate, ctx)`** — the AbortSignal is the 3rd positional arg (pass it straight into `pi.exec(bin, args, { signal, timeout })`). Return `{ content:[{type:"text",text}], details:{} }`; **throw** (or set `isError:true`) on failure.
- **exec result uses `.code` not `.exitCode`**, and **non-zero exit resolves** (no throw) — branch on `result.code !== 0` and surface `result.stderr`. `result.killed` marks abort.
- **`pi.exec` command can be an absolute path** to the rod-cli binary; else resolves via PATH.
- **session_shutdown has NO reason field** (and no 'quit') — make any daemon/browser teardown idempotent and unconditional; don't gate on a reason. `session_start`/`session_shutdown` payloads are just `{type}`; the handler's 2nd arg `ctx` gives you `ctx.ui.notify(msg, "info"|"warning"|"error")` and `pi.logger`.
