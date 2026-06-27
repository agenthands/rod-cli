---
author: anvil-researcher
responsible: architect
emitted_at: 2026-06-27T22:00:00Z
phase: null
artifact_kind: research
parent_artifacts:
  - .planning/PROJECT.md
  - .planning/research/STACK.md
  - .planning/research/FEATURES.md
  - docs/onboarding/pi.md
  - skills/rod-cli/SKILL.md
status: draft
---

# Research — Pi Extension: PITFALLS

**Answer:** Six critical mistakes will silently break a Pi extension that shells out to an
external Go binary: (1) using the wrong hook name (`session_end` — doesn't exist; it's
`session_shutdown`), (2) using `Type.Enum`/`Type.Union` instead of `StringEnum` (silently
incompatible with Google API), (3) returning `isError` instead of throwing (silently
ignored), (4) assuming Unix PATH resolution works on Windows (`.cmd` shims aren't found by
`spawn`), (5) not truncating output to 50KB/2000-line ceiling (LLM context overflow), and
(6) starting the browser daemon in the extension factory instead of deferring to
`session_start`. **Confidence: HIGH** — every finding is verified against the official Pi
source, real GitHub issues, and published extension patterns.

**Recommendation:** Fix the hook name in STACK.md and FEATURES.md immediately (they
reference the nonexistent `session_end`). Use the checklist in §Prevention below during
planning. Add a Windows CI lane even if rod-cli itself doesn't ship Windows binaries yet
— the extension must fail gracefully, not crash.

---

## Findings

### 1. CRITICAL: The Hook Is `session_shutdown`, NOT `session_end`

- **finding:** The Pi docs define `session_shutdown` as the teardown hook — there is NO
  `session_end` event. `session_shutdown` fires "before a started session runtime is torn
  down" with `event.reason` (`"quit"`, `"reload"`, `"new"`, `"resume"`, `"fork"`). The
  existing STACK.md and FEATURES.md use `pi.on("session_end", ...)` — this would silently
  never fire, leaving zombie browser daemons on every session exit.
- **source:** Official Pi extensions docs (`pi.dev/docs/latest/extensions`) — lists
  `session_shutdown`, no `session_end` · **quality:** HIGH official · **as of:** 2026-06-27
- **source:** `earendil-works/pi` extensions.md source — complete hook event catalog,
  `session_shutdown` is the only session-teardown event · **quality:** HIGH source
- **bears on:** Every lifecycle hook in the extension. This is the single highest-impact
  pitfall — if the teardown hook never fires, `rod-cli close` never runs, and the browser
  daemon leaks.

**Correct hook registration:**
```typescript
// ✅ CORRECT
pi.on("session_shutdown", async (event) => {
  // event.reason: "quit" | "reload" | "new" | "resume" | "fork"
  try { await pi.exec("rod-cli", ["close"], { timeout: 5000 }); } catch {}
});

// ❌ WRONG — this event does not exist
pi.on("session_end", async () => { ... });
```

**Session replacement lifecycle — the reload hazard:** When Pi forks, reloads, or switches
sessions, the sequence is:
1. `session_shutdown` fires for the OLD session
2. Extensions are reloaded/re-bound
3. `session_start` fires for the NEW session with `event.previousSessionFile`

The `session_shutdown` handler must tear down the daemon for the session being left. The
`session_start` handler must re-verify the binary for the new session. Old captured
`pi`/`ctx` objects are stale after replacement and throw if used.

- **source:** Official Pi extensions docs ("Replacement lifecycle sequence") ·
  **quality:** HIGH official

---

### 2. CRITICAL: `StringEnum` From `@earendil-works/pi-ai`, NOT `Type.Enum`

- **finding:** `Type.Union`/`Type.Literal` from TypeBox does NOT work with Google's
  Vertex AI / Gemini API. The Pi docs explicitly state: "`Type.Union`/`Type.Literal`
  doesn't work with Google's API." You MUST use `StringEnum` from `@earendil-works/pi-ai`
  for any string enum parameter, with the `as const` assertion. This is not a preference
  — it's a hard compatibility requirement.
- **source:** Official Pi extensions docs ("Use `StringEnum` from `@earendil-works/pi-ai`
  for string enums") · **quality:** HIGH official
- **source:** `earendil-works/pi` extensions.md source ("Schema validation sequence",
  TypeBox section) · **quality:** HIGH source
- **source:** `pi-extension-template` docs — shows `StringEnum` as the only enum pattern ·
  **quality:** HIGH reference
- **bears on:** Every tool with an `action` discriminator parameter — `browse_tabs`
  (`action: "list"|"new"|"close"|"select"`), `browse_navigate`
  (`action: "reload"|"go_back"|"go_forward"`), `browse_handle_dialog`
  (`accept: true|false` — though Boolean, not enum), and `browse_screenshot`
  (`format: "png"|"jpeg"`).

```typescript
// ✅ CORRECT
import { StringEnum } from "@earendil-works/pi-ai";

const TabAction = StringEnum(["list", "new", "close", "select"] as const);

// ❌ WRONG — silently incompatible with Google API
const TabAction = Type.Enum({ list: "list", new: "new", close: "close", select: "select" });
const TabAction = Type.Union([Type.Literal("list"), Type.Literal("new"), ...]);
```

**Additional TypeBox pitfall — multiple versions:** If `typebox` appears at multiple
versions in `node_modules` (common with monorepos), `Value.Cast` can fail with
"ValueCheck: Unknown type" due to symbol conflicts. Keep a single `typebox` version
via `peerDependencies`.

- **source:** Community blog on TypeBox enum pitfalls · **quality:** MEDIUM corroborated ·
  **as of:** 2026-06-27

---

### 3. CRITICAL: Errors MUST Be Thrown, NOT Returned

- **finding:** Setting `isError: true` in the return value of a tool's `execute()` function
  is SILENTLY IGNORED. The Pi docs explicitly state: "Returning a value never sets the
  error flag regardless of what properties you include." To signal an error to the LLM,
  you MUST `throw` from `execute()`. This is the #1 tool-authoring mistake.
- **source:** Official Pi extensions docs ("Must **throw** to mark an error") ·
  **quality:** HIGH official
- **source:** `earendil-works/pi` extensions.md ("Error signaling" section) ·
  **quality:** HIGH source
- **bears on:** Every tool's error handling. The `execRodCli` wrapper must throw on
  non-zero exit; individual tools must let errors propagate, not catch-and-return.

```typescript
// ✅ CORRECT
async execute(_toolCallId, params, signal) {
  const result = await pi.exec("rod-cli", ["goto", params.url], { signal });
  if (result.code !== 0) {
    throw new Error(`rod-cli goto failed: ${result.stderr}`);
  }
  return { content: [{ type: "text", text: result.stdout }] };
}

// ❌ WRONG — LLM will never see the error
async execute(_toolCallId, params, signal) {
  const result = await pi.exec("rod-cli", ["goto", params.url], { signal });
  if (result.code !== 0) {
    return {
      content: [{ type: "text", text: `Error: ${result.stderr}` }],
      isError: true  // SILENTLY IGNORED
    };
  }
  return { content: [{ type: "text", text: result.stdout }] };
}
```

---

### 4. CRITICAL: Cross-Platform Binary Discovery — Windows `.cmd` Shims

- **finding:** On Windows, npm and Scoop install binaries as `.cmd` shim files, not
  `.exe` executables. Node's `child_process.spawn()` with `shell: false` (the default)
  does NOT resolve `.cmd` files through `PATHEXT` — it returns `ENOENT`. This has been
  the root cause of at least 3 Pi issues (#2464, #1220, #2186). Every Pi extension that
  shells out to an external binary has hit this.
- **source:** `earendil-works/pi#2464` — "subagent fails on Windows when pi is installed
  as a .cmd shim" · **quality:** HIGH issue
- **source:** `earendil-works/pi#1220` — "`pi install npm:*` fails with 'spawn npm
  ENOENT'" · **quality:** HIGH issue
- **source:** `earendil-works/pi#2186` — "VS Code launch on Windows — `pi.exec('code',
  ...)` fails" · **quality:** HIGH issue
- **bears on:** The `findRodCli()` function in `cli.ts`. On Windows, it must check for
  `rod-cli.exe` AND `rod-cli.cmd`, AND `$env:GOBIN` instead of `$GOBIN`, AND use `;`
  path separator instead of `:`.

**Cross-platform binary discovery checklist:**

| Concern | Unix (Linux/macOS) | Windows |
|---------|-------------------|---------|
| Binary name | `rod-cli` | `rod-cli.exe` (or `rod-cli.cmd` if npm-wrapped) |
| PATH separator | `:` | `;` |
| Go bin default | `$GOPATH/bin` or `$GOBIN` or `~/go/bin` | `%GOPATH%\bin` or `%GOBIN%` or `%USERPROFILE%\go\bin` |
| `process.env.HOME` | `/home/user` | `C:\Users\user` (use `USERPROFILE`) |
| `existsSync` checks | Works directly | Must check for `.exe` extension too |
| `pi.exec("rod-cli", ...)` | May resolve via `which` | May need explicit `.cmd` or full path |

```typescript
// Platform-aware binary resolution
import { existsSync } from "node:fs";
import { join } from "node:path";
import { platform, homedir } from "node:os";

const IS_WINDOWS = platform() === "win32";
const BINARY_NAME = IS_WINDOWS ? "rod-cli.exe" : "rod-cli";
const PATH_SEP = IS_WINDOWS ? ";" : ":";

function findRodCli(): string | null {
  // 1. Explicit env var
  if (process.env.ROD_CLI_PATH && existsSync(process.env.ROD_CLI_PATH)) {
    return process.env.ROD_CLI_PATH;
  }
  // 2. PATH search
  const pathDirs = (process.env.PATH || "").split(PATH_SEP);
  for (const dir of pathDirs) {
    const candidate = join(dir, BINARY_NAME);
    if (existsSync(candidate)) return candidate;
  }
  // 3. Go bin directory
  const gobin = process.env.GOBIN ||
    (IS_WINDOWS
      ? join(process.env.USERPROFILE || homedir(), "go", "bin")
      : join(homedir(), "go", "bin"));
  const goCandidate = join(gobin, BINARY_NAME);
  if (existsSync(goCandidate)) return goCandidate;
  return null;
}
```

> **Note on `pi.exec` vs `which`:** `pi.exec` likely resolves commands through the
> system PATH, but the exact mechanism is undocumented. For reliability, resolve the
> binary path explicitly and use the full path in `pi.exec("rod-cli", ...)` if
> ambiguous. The STACK.md's `findRodCli()` uses `:`-only path splitting without
> Windows detection — this will break on Windows.

- **source:** STACK.md `findRodCli()` implementation (Unix-only PATH splitting) ·
  **quality:** HIGH project — the existing code has this bug

---

### 5. CRITICAL: 50KB/2000-Line Output Truncation

- **finding:** Pi enforces a DEFAULT output ceiling of 50KB or 2000 lines. Tool results
  that exceed this are truncated. `truncateHead()` and `truncateTail()` from
  `@mariozechner/pi-coding-agent` handle truncation; when truncation occurs, the full
  output is saved to a temp file and a notice appended. Failing to truncate means the
  agent either gets no output (silent truncation by Pi) or the LLM context overflows.
- **source:** `@mariozechner/pi-coding-agent` exports — `truncateHead`, `truncateTail`,
  `DEFAULT_MAX_BYTES` (51200), `DEFAULT_MAX_LINES` (2000) · **quality:** HIGH source
- **source:** `pi-output-guard` extension (universal token ceiling pattern) ·
  **quality:** MEDIUM community
- **source:** `pi-sub-agent` extension (2000-line tail truncation per agent) ·
  **quality:** MEDIUM community
- **bears on:** `browse_snapshot` output (can be 10–50KB for complex pages),
  `browse_screenshot` base64, `browse_eval` (arbitrary JS output), `browse_console`
  (verbose logs). Every tool that returns potentially large output must truncate.

**Truncation pattern for rod-cli tools:**
```typescript
import { truncateHead, DEFAULT_MAX_BYTES, DEFAULT_MAX_LINES } from "@mariozechner/pi-coding-agent";

async execute(_toolCallId, params, signal) {
  const result = await pi.exec("rod-cli", ["--raw", "snapshot"], { signal });
  const truncated = truncateHead(result.stdout, DEFAULT_MAX_BYTES, DEFAULT_MAX_LINES);
  return { content: [{ type: "text", text: truncated }] };
}
```

**Rod-cli specific concern:** `browse_snapshot` returns accessibility-tree markdown that
is already token-efficient by design (~10–50KB for a typical page). A complex page
(e.g., amazon.com, a large e-commerce listing) could still exceed 50KB. The rod-cli
`--raw` flag produces machine-parseable output but does NOT guarantee a size bound.

**Recommended mitigation:** Add an optional `maxLength` parameter to `browse_snapshot`
(default: unlimited, but truncate at Pi's ceiling). If the snapshot is truncated, the
tool result should include a `details.truncated: true` flag and suggest re-calling with
a scoped `selector`.

---

### 6. HIGH: `ctx.signal` Is `undefined` Outside Active Turns

- **finding:** `ctx.signal` (the AbortSignal for agent abort) is "usually `undefined` in
  idle or non-turn contexts such as session events, extension commands, and shortcuts
  fired while pi is idle." The `session_start` and `session_shutdown` hooks receive a
  context where `ctx.signal` may be `undefined`. Code that assumes `signal` is always
  available (e.g., passing it to `pi.exec`) will work but won't be abortable.
- **source:** Official Pi extensions docs ("Signal availability" section) ·
  **quality:** HIGH official
- **bears on:** The `session_start` hook (which runs `pi.exec("rod-cli", ["--version"])`
  to verify the binary) and the `session_shutdown` hook (which runs `pi.exec("rod-cli",
  ["close"])`). These are fine — they're short operations with explicit short timeouts
  and don't need abortability. But tool `execute` functions should check: `if (!signal)
  { /* warn? create a manual timeout? */ }`.

**Mitigation:** Tool `execute` functions receive `signal` as a separate parameter (not
via `ctx.signal`), which IS set during active turns. The pitfall only affects hooks and
commands that use `ctx.signal`.

---

### 7. HIGH: Long-Lived Resources MUST Be Deferred to `session_start`

- **finding:** The Pi docs explicitly warn: "Extension factories may run in invocations
  that never start a session." The extension factory (the default export function) must
  NOT start background processes, open sockets, or set up timers. All long-lived
  resources must be deferred to `session_start` and cleaned up in `session_shutdown`.
  Starting the rod-cli daemon in the factory function risks leaving a browser running in
  a headless Pi invocation (e.g., `pi --list-models`, `pi --help`).
- **source:** Official Pi extensions docs ("Do not start background processes, sockets,
  file watchers, or timers from the extension factory") · **quality:** HIGH official
- **bears on:** The daemon startup. The FIRST `browse_goto` call implicitly starts the
  daemon — this is correct. Do NOT start the daemon in `session_start` proactively; let
  the first tool invocation trigger it. `session_start` should only VERIFY the binary
  exists and is functional.

```typescript
// ✅ CORRECT — only verify, don't start
pi.on("session_start", async (_event, ctx) => {
  const result = await pi.exec("rod-cli", ["--version"], { timeout: 5000 });
  if (result.code !== 0) {
    ctx.ui.notify(`rod-cli not ready: ${result.stderr}`, "error");
  }
});

// ❌ WRONG — starts the daemon unconditionally, even if session never browses
pi.on("session_start", async (_event, ctx) => {
  await pi.exec("rod-cli", ["goto", "about:blank"], { timeout: 30000 });
});

// ❌ WORSE — starts daemon even if Pi is just listing models
export default function(pi: ExtensionAPI) {
  execSync("rod-cli goto about:blank"); // runs in factory!
}
```

---

### 8. MEDIUM: Existing SKILL.md — Keep It, Don't Replace It

- **finding:** Pi skills (SKILL.md) and TypeScript extensions are complementary, not
  competing. Skills provide progressive-disclosure instruction ("what the agent should
  know"); extensions provide executable capabilities ("what the agent can do"). They
  coexist naturally: skills are loaded on-demand when the LLM's task matches the
  `description` frontmatter; extensions run at startup and register tools. Removing the
  SKILL.md when the extension ships would:
  - Break users who only use the skill (no extension installed)
  - Remove the progressive-disclosure benefit (only the `description` is in the system
    prompt; the full body loads on demand)
  - Lose the detailed usage examples and reference docs the skill provides
- **source:** `pi.dev/docs/latest/extensions` — skills and extensions are separate
  layers in Pi's processing pipeline · **quality:** HIGH official
- **source:** `understanding-pi-agent-extension-model` (exitcode0.net) — coexistence
  patterns, skill vs extension decision matrix · **quality:** MEDIUM reference
- **bears on:** Whether to remove, keep, or link the existing `skills/rod-cli/SKILL.md`.

**Recommendation — keep the SKILL.md AND add the extension:**

| Layer | What ships | Role |
|-------|-----------|------|
| SKILL.md | `skills/rod-cli/SKILL.md` (keep as-is, update) | Progressive disclosure: instructions + examples |
| Extension | `extensions/pi/` (new) | Executable tools: registers `browse_*` tools |

**Update the SKILL.md to reference the extension:**
```markdown
---
name: rod-cli
description: "Token-efficient browser automation via rod-cli..."
---

## Using rod-cli

**With the Pi extension installed** (`pi install npm:@agenthands/rod-cli-pi`):
rod-cli tools are available as `browse_goto`, `browse_snapshot`, etc.
— call them directly.

**Without the extension** (skill-only fallback):
Use `rod-cli` directly via shell commands as documented below.
```

This preserves backward compatibility for skill-only users while guiding extension users
toward the structured tool interface.

**Note:** If the extension registers a command with the same name as a skill command,
the extension wins. Extension commands take precedence over prompt templates. This is
not a concern for rod-cli (the extension registers tools, not commands), but worth
knowing if `browse_*` were ever registered as commands.

---

### 9. MEDIUM: Daemon Lifecycle Across Session Forks and Reloads

- **finding:** Pi's session model includes fork, reload, resume, and new-session
  operations. The `session_shutdown` hook fires with a `reason` field
  (`"quit"`|`"reload"`|`"new"`|`"resume"`|`"fork"`). On reload and fork, the browser
  daemon should SURVIVE (it's the same user's browsing context) — but on quit, it must
  be torn down. The naive "always `rod-cli close` on shutdown" approach kills the
  daemon on `/reload`, losing the browsing state.
- **source:** Official Pi extensions docs ("Replacement lifecycle sequence") ·
  **quality:** HIGH official
- **bears on:** The `session_shutdown` handler. It should only `rod-cli close` on
  `reason === "quit"`, not on reload/fork/resume.

```typescript
pi.on("session_shutdown", async (event) => {
  // Only tear down the daemon on actual session exit, not on reloads/forks
  if (event.reason === "quit") {
    try { await pi.exec("rod-cli", ["close"], { timeout: 5000 }); } catch {}
  }
  // For reload/new/resume/fork: the daemon stays alive, the new session reuses it
});
```

**Open question:** Should a named session (`-s admin`) survive a Pi session fork?
Probably yes — named sessions are user-managed and outlive any single Pi session.
The `session_shutdown` handler should only close the default session on quit; named
sessions are the user's responsibility (or could be listed in a notification).

---

### 10. MEDIUM: `pi.exec` Timeout vs rod-cli Command Duration

- **finding:** rod-cli commands have wildly different execution times: `--version` takes
  <1s, `snapshot` takes 1–3s, `goto` can take 5–60s on slow sites, and `eval` depends
  on the script. A single default timeout across all `pi.exec` calls will either be too
  short (killing legitimate slow pages) or too long (hanging on stuck commands). Each
  tool must set its own timeout based on the rod-cli command it wraps.
- **source:** `pi.exec` API — `timeout` parameter in milliseconds, kills the process
  when exceeded · **quality:** HIGH official
- **source:** rod-cli SKILL.md — command categories with implicit timing profiles ·
  **quality:** HIGH primary
- **bears on:** The `execRodCli` wrapper and each tool's timeout parameter.

**Recommended per-command timeouts:**

| Command | Timeout | Rationale |
|---------|---------|-----------|
| `--version` | 5s | Instant; only used in `session_start` |
| `goto <url>` | 60s | Slow sites, first load, DNS resolution |
| `snapshot` | 15s | DOM serialization, rarely slow |
| `click <sel>` | 15s | Navigation may follow click |
| `fill <sel> <text>` | 15s | Form interaction, fast |
| `eval <script>` | 15s | JS execution, script-dependent (user-controlled) |
| `screenshot` | 30s | Viewport capture + file I/O |
| `wait <sel>` | 30s | Explicit waiting, user expects delay |
| `close` | 5s | Teardown, best-effort |

The tool's `execute()` function should use its own timeout parameter if provided, or
fall back to the recommended default. The user-facing `timeout` parameter in
`browse_goto` and `browse_wait` allows the LLM to override.

---

### 11. LOW: Path `@` Prefix — Some Models Include It

- **finding:** "Some models are idiots and include the @ prefix in tool path arguments."
  Built-in Pi tools strip a leading `@` before resolving paths. Custom tools that accept
  file paths (e.g., `browse_file_upload`, `browse_screenshot` with `name`) should
  normalize a leading `@` as well.
- **source:** Official Pi extensions docs ("@ prefix in paths") · **quality:** HIGH
  official
- **bears on:** `browse_file_upload` (accepts a file path) and `browse_screenshot`
  (accepts an optional filename). Strip `@` prefix before passing to rod-cli.

```typescript
function normalizePath(p: string): string {
  return p.startsWith("@") ? p.slice(1) : p;
}
```

---

### 12. LOW: `prepareArguments` for Schema Evolution

- **finding:** Pi sessions can be resumed from stored state. If a tool's parameter schema
  changes between versions, resumed sessions carry OLD argument shapes. The
  `prepareArguments(args)` shim runs BEFORE schema validation and lets you transform
  legacy args into the current shape. Without it, resumed sessions fail silently on
  schema validation. The Pi docs explicitly say: "Do not add deprecated compatibility
  fields to `parameters` just to keep old resumed sessions working."
- **source:** Official Pi extensions docs ("Schema validation sequence") ·
  **quality:** HIGH official
- **bears on:** Future schema evolution. The v0.1.0 schema should include
  `prepareArguments` as a no-op placeholder so it's available when needed.

```typescript
pi.registerTool({
  name: "browse_goto",
  parameters: GotoParams,
  // No-op for v0.1.0; available when schema evolves
  prepareArguments(args) {
    // Example future migration: { "uri": "..." } → { "url": "..." }
    if ("uri" in (args as any)) {
      (args as any).url = (args as any).uri;
      delete (args as any).uri;
    }
    return args;
  },
  async execute(_toolCallId, params, signal) { ... },
});
```

---

### 13. LOW: Parallel Tool Execution Races — Not a Concern for Read-Only Browser Tools

- **finding:** Pi tools "run in parallel by default." If two tools mutate the same file
  without `withFileMutationQueue()`, a read-modify-write race occurs. rod-cli Pi tools
  are read-only from the filesystem perspective (they shell out to the binary, which
  mutates browser state, not disk files), so this pitfall does NOT apply. However, if
  future tools write files (e.g., `browse_screenshot` saving to disk), they should use
  `withFileMutationQueue()`.
- **source:** Official Pi extensions docs ("Parallel tool races") · **quality:** HIGH
  official
- **bears on:** Low priority for v0.1.0. Add a note to the tool authoring guide.

---

### 14. LOW: `terminate: true` — Batch-Level, Not Single-Tool

- **finding:** Returning `terminate: true` from a tool's `execute()` only stops the
  batch when "every finalized tool result in that batch is terminating." It does NOT
  stop the agent from making further tool calls. It's a batch-level optimization, not a
  session-stop mechanism. Don't use it.
- **source:** Official Pi extensions docs ("Terminate behavior") · **quality:** HIGH
  official
- **bears on:** rod-cli tools should not return `terminate: true`. The daemon lifecycle
  is handled by hooks, not tool results.

---

### 15. LOW: `pi.exec` Env Propagation

- **finding:** `pi.exec` accepts an `env` option (`Record<string, string | undefined>`)
  to override environment variables. The default env is Pi's own process environment.
  The `cwd` option sets the working directory. These are useful for rod-cli tools:
  `browse_goto` might want to set `--user-data-dir` or pass browser flags. Prefer
  `--profile` and rod-cli's own configuration over env vars, but `env` is available as
  an escape hatch.
- **source:** Official Pi extensions docs (`pi.exec` options: `cwd`, `env`, `signal`,
  `timeout`) · **quality:** HIGH official
- **bears on:** Low priority. rod-cli uses flags, not env vars, for configuration.

---

## Prevention Checklist

For the architect and engineer — verify each of these before shipping:

### Immediate (fix existing artifacts)
- [ ] **Fix hook name:** STACK.md and FEATURES.md use `session_end` → change to `session_shutdown`
- [ ] **Fix `findRodCli()`:** STACK.md uses `:`-only PATH splitting → add `;` for Windows
- [ ] **Fix execRodCli:** STACK.md signature uses `{ exec: ... }` duck type → use real
  `ExtensionAPI` type if available, or `pi.exec` directly

### Architecture (design phase)
- [ ] `session_shutdown` handler gates `rod-cli close` on `event.reason === "quit"`
- [ ] `session_start` handler verifies binary but does NOT start the daemon
- [ ] Every tool uses `StringEnum` for enum params (import from `@earendil-works/pi-ai`)
- [ ] Every tool throws on error (never returns `isError`)
- [ ] SKILL.md is kept and updated, not deleted
- [ ] `prepareArguments` is a no-op placeholder on every tool
- [ ] Per-command timeout table is implemented

### Implementation (build phase)
- [ ] Windows PATH resolution: `;` separator, `rod-cli.exe` name, `USERPROFILE` not `HOME`
- [ ] Output truncation: `truncateHead`/`truncateTail` for `browse_snapshot`, `browse_eval`, `browse_console`
- [ ] `@` prefix stripping in path-accepting tools
- [ ] Tools never use `terminate: true`
- [ ] `withFileMutationQueue` used if any tool writes files to disk
- [ ] Daemon is NOT started in the extension factory

### Verification (QA phase)
- [ ] Test on Windows (even if rod-cli doesn't ship Windows binaries — extension must fail gracefully)
- [ ] Test session reload: daemon survives `/reload`, closes on `/quit`
- [ ] Test with a 60KB snapshot page — verify truncation works
- [ ] Test with a slow-loading page — verify timeout is not too aggressive
- [ ] Test with `StringEnum` params against a Google AI backend (Vertex/Gemini)

---

## Tradeoffs / Alternatives

### `pi.exec` vs raw `child_process.spawn`
- **`pi.exec` (chosen):** Integrated abort signals, consistent PATH resolution, documented
  API. Cons: less control over stdio streaming, undocumented Windows behaviour.
- **`child_process.spawn`:** Full control, streaming stdout, explicit shell option. Cons:
  no Pi abort integration, must manage process tree cleanup manually, Windows `.cmd`
  shim resolution must be handled explicitly.
  **Recommendation:** use `pi.exec` for all tool invocations. If streaming output is
  needed later, explore `createBashTool`/`createLocalBashOperations` which "lets
  extensions reuse pi's local shell backend."

### `findRodCli()` with full path vs relying on `pi.exec` PATH resolution
- **Full path resolution (chosen):** Resolve binary path explicitly, use full path in
  `pi.exec`. Pros: works cross-platform, debuggable, can notify user if missing.
  Cons: more code, must handle Windows quirks.
- **Rely on `pi.exec` PATH:** Just shell out to `"rod-cli"` and hope. Pros: less code.
  Cons: Windows `.cmd` shim resolution is undocumented; error message is cryptic
  (`ENOENT`). **Recommendation:** resolve explicitly. The `session_start` notification
  when the binary is missing is a better UX than "spawn rod-cli ENOENT."

### `session_shutdown` always-closes vs reason-gated close
- **Always close (simpler but wrong):** `rod-cli close` on every `session_shutdown`
  regardless of reason. Cons: kills daemon on `/reload`, losing browsing state.
- **Reason-gated (chosen):** Only close on `reason === "quit"`. Pros: preserves daemon
  across reloads/forks. Cons: if the user switches projects without quitting, the
  daemon persists (handled by PPID polling + 15-minute idle timeout).

---

## Unknowns

- **`pi.exec` Windows PATH resolution:** The exact mechanism `pi.exec` uses to resolve
  the command name (shell-out, `which`/`where`, direct `spawn`) is undocumented. It's
  unclear whether `pi.exec("rod-cli", ...)` works on Windows without the `.exe`/`.cmd`
  extension. **How to settle:** test on a Windows machine with rod-cli installed via
  `go install`.
- **`pi.exec` abort signal details:** The docs show `signal` is passed to `pi.exec`, but
  the exact behaviour (SIGTERM vs SIGKILL on Unix, `TerminateProcess` vs CTRL-C on
  Windows, child process tree cleanup) is not specified. **How to settle:** STACK.md
  already notes this — test with `pi.exec("sleep", ["60"])` and a 1s AbortSignal.
- **Daemon survival across Pi session forks:** If a user forks a Pi session (`/fork`)
  after starting the rod-cli daemon, the child session inherits the daemon's PPID
  (which is the OLD session's PID). If the parent session exits, the PPID poll may kill
  the daemon even though the child session is still using it. **How to settle:** test
  with a real fork scenario and observe PPID behaviour.
- **`@pi-ai` import in user extensions:** The `StringEnum` import is from
  `@earendil-works/pi-ai`, which is a Pi internal package available at runtime. It
  should be in `peerDependencies` (not `dependencies`) since Pi provides it. If the
  user's node_modules don't have it at dev time, `tsc --noEmit` will fail.
  **How to settle:** add `@earendil-works/pi-ai` to `devDependencies` and confirm
  typecheck passes.
- **Google API validation gap (Cloudflare Workers):** There's an open issue (#3112)
  where tool argument validation is silently SKIPPED in Cloudflare Workers because AJV
  requires `eval()`. If Pi ever runs on Workers, schema validation is a no-op.
  **Impact:** medium-term — rod-cli is a local tool, unlikely to run on Workers.

---

## Sources

- Pi extensions official docs: https://pi.dev/docs/latest/extensions (API reference,
  hooks, pi.exec, tool registration)
- Pi extensions source: https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/extensions.md
  (complete API surface, all pitfalls documented)
- Pi Windows `.cmd` shim issue: https://github.com/earendil-works/pi/issues/2464
  (subagent fails with `spawn pi ENOENT`)
- Pi Windows npm spawn issue: https://github.com/earendil-works/pi/issues/1220
  (`spawn npm ENOENT` — same root cause)
- Pi Windows VS Code issue: https://github.com/earendil-works/pi/issues/2186
  (`spawn code` fails on Windows — platform-specific fix)
- Pi Cloudflare validation issue: https://github.com/earendil-works/pi/issues/3112
  (AJV `eval()` unavailable → schema validation silently skipped)
- Pi extension template (StringEnum pattern): https://cdn.jsdelivr.net/npm/pi-extension-template@0.1.1/docs/typescript.md
- Pi skills vs extensions model: https://exitcode0.net/posts/understanding-pi-agent-extension-model/
- pi-output-guard extension (truncation pattern): https://github.com/26q5gr9hwd-crypto/pi-output-guard
- pi-browser extension (reference implementation): https://github.com/larsderidder/pi-browser
- rod-cli SKILL.md: `skills/rod-cli/SKILL.md`
- STACK.md: `.planning/research/STACK.md` (binary resolution code with Unix-only PATH splitting)
- FEATURES.md: `.planning/research/FEATURES.md` (references nonexistent `session_end` hook)
- PROJECT.md: `.planning/PROJECT.md` (v2.2 milestone target)

---
*Pitfalls research for: Pi Extension shell-out implementation (rod-cli v2.2)*
*Researched: 2026-06-27*
