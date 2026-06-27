---
author: anvil-researcher
responsible: architect
emitted_at: 2026-06-27T18:00:00Z
phase: null
artifact_kind: research
parent_artifacts:
  - .planning/PROJECT.md
status: draft
---

# Research — Pi Extension Stack

**Question:** What npm package structure, TypeScript config, build tooling, and dependencies are needed for a Pi extension that shells out to an external Go binary (rod-cli)?

**Answer:** Pi extensions are TypeScript source files loaded directly by Pi — no compilation or build step is required. The extension lives in `extensions/pi/` with a `package.json` declaring `"pi": { "extensions": ["./src/index.ts"] }` and peer dependencies on `@earendil-works/pi-coding-agent`, `@earendil-works/pi-ai`, and `typebox`. Binary dependencies (rod-cli) are handled by documenting the prerequisite and verifying the binary on `session_start` via `pi.exec()`. **Confidence: HIGH** (verified against official docs, real published packages, and the Pi source repo).

**Recommendation:** Scaffold the extension as an npm-publishable TypeScript package with zero runtime dependencies beyond the three Pi peer deps. Use `pi.exec("rod-cli", [...args])` for all shell-outs. Verify rod-cli presence on PATH at startup and notify the user if missing. Keep `"noEmit": true` in tsconfig — Pi loads `.ts` directly.

---

## Findings

### 1. Pi Extension API Surface

- **finding:** Extensions export a default factory function receiving `ExtensionAPI`. Tools are registered via `pi.registerTool()`, commands via `pi.registerCommand()`, and lifecycle hooks via `pi.on()`.
- **source:** https://pi.dev/docs/latest/extensions · **quality:** HIGH official · **as of:** pi.dev docs live (2026-06-27)
- **source:** https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/extensions.md · **quality:** HIGH source repo · **as of:** main branch (2026-06-27)
- **bears on:** Defines the exact API contract the extension must implement.

#### Key API details:

**Import and export shape:**
```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";

export default function (pi: ExtensionAPI) {
  // register tools, commands, hooks here
}
```

**`pi.registerTool()` full signature:**

| Field | Type | Required | Purpose |
|-------|------|----------|---------|
| `name` | `string` | Yes | Unique tool identifier (e.g. `"browse_goto"`) |
| `label` | `string` | Yes | Display name (e.g. `"Browse Goto"`) |
| `description` | `string` | Yes | What the tool does (shown to LLM) |
| `promptSnippet` | `string` | No | One-line summary for "Available tools" list |
| `promptGuidelines` | `string[]` | No | Guidelines appended to LLM system prompt (must name the tool explicitly) |
| `parameters` | TypeBox schema | Yes | Arguments schema (use `StringEnum` for enums, not `Type.Enum`) |
| `prepareArguments` | `(args) => args` | No | Preprocess args before schema validation (compatibility shim) |
| `execute` | `async (toolCallId, params, signal, onUpdate, ctx) => ToolResult` | Yes | Main execution logic |
| `renderCall` | function | No | Custom rendering of tool call in conversation |
| `renderResult` | function | No | Custom rendering of tool result in conversation |

**`execute` return type (`ToolResult`):**
```typescript
{
  content: [{ type: "text", text: "..." }],  // sent to LLM
  details?: Record<string, any>,              // for UI rendering & state
  terminate?: boolean,                         // stop batch if all results also terminate
}
```

**`pi.on()` event types:** `"session_start"`, `"session_end"`, `"session_shutdown"`, `"tool_call"`, `"agent_end"`, `"before_agent_start"`.

**`pi.exec()` — the shell-out primitive:**
```typescript
const result = await pi.exec("rod-cli", ["goto", url], { signal, timeout: 30000 });
// result: { stdout: string, stderr: string, code: number, killed: boolean }
```

- **source:** https://pi.dev/docs/latest/extensions (API reference + pi.exec APIDOC) · **quality:** HIGH official
- **bears on:** The execute function in every tool will use `pi.exec()` to shell out; hooks use `pi.on()` for daemon lifecycle.

### 2. Package Structure & npm Conventions

- **finding:** Pi extensions published to npm follow a consistent pattern: `"type": "module"` (ESM), `"main": "src/index.ts"`, `"pi": { "extensions": ["./src/index.ts"] }`, and `"keywords": ["pi-package"]`. Peer dependencies are declared with `"*"` version ranges. TypeScript is loaded directly — no build step.
- **source:** `@harms-haus/pi-til-done` v1.2.0 package.json (published npm package) · **quality:** HIGH real-world · **as of:** 2026-06-27
- **source:** https://pi.dev/docs/latest/packages · **quality:** HIGH official · **as of:** 2026-06-27
- **bears on:** The `package.json` template for `extensions/pi/`.

#### Canonical package.json for a Pi extension:

```json
{
  "name": "@agenthands/rod-cli-pi",
  "version": "0.1.0",
  "description": "Pi extension wrapping rod-cli for native browser automation",
  "type": "module",
  "main": "src/index.ts",
  "keywords": ["pi-package", "pi-extension", "browser-automation", "rod-cli"],
  "files": [
    "src/**/*.ts",
    "!src/__tests__/**",
    "README.md",
    "CHANGELOG.md",
    "LICENSE"
  ],
  "scripts": {
    "typecheck": "tsc --noEmit",
    "test": "vitest",
    "lint": "eslint src/",
    "format": "prettier --write src/"
  },
  "peerDependencies": {
    "@earendil-works/pi-coding-agent": "*",
    "@earendil-works/pi-ai": "*",
    "typebox": "*"
  },
  "devDependencies": {
    "@types/node": "^22.0.0",
    "typescript": "^5.8.0",
    "vitest": "^3.0.0",
    "prettier": "^3.0.0",
    "eslint": "^9.0.0"
  },
  "engines": {
    "node": ">=22.0.0"
  },
  "pi": {
    "extensions": ["./src/index.ts"]
  },
  "repository": {
    "type": "git",
    "url": "git+https://github.com/agenthands/rod-cli.git",
    "directory": "extensions/pi"
  },
  "license": "MIT"
}
```

**Key conventions observed across published Pi extensions:**

| Convention | Detail |
|---|---|
| Scoped name | `@org/name` — matches npm registry convention |
| `"type": "module"` | ESM, universal across extensions |
| `"main": "src/index.ts"` | Points to TypeScript source, not compiled JS |
| `pi.extensions` | Array of entry point paths, Pi reads this at install time |
| `"keywords": ["pi-package"]` | Discoverability marker |
| `peerDependencies` with `"*"` | Pi provides these at runtime; no version pinning needed |
| `"noEmit": true` in tsconfig | Extensions are NOT compiled; Pi loads `.ts` directly |
| Zero runtime deps beyond peers | Keeps supply chain surface minimal |

- **source:** https://pi.dev/docs/latest/extensions · **quality:** HIGH official
- **source:** https://pi.dev/docs/latest/packages · **quality:** HIGH official
- **bears on:** The exact shape of the extension's `package.json`.

### 3. TypeScript Configuration

- **finding:** Pi extensions use `tsconfig.json` with `"noEmit": true`, targeting ES2022, ESNext modules, bundler resolution, and strict mode. TypeScript is used for type-checking at dev time only — Pi loads `.ts` files directly at runtime without compilation.
- **source:** `@harms-haus/pi-til-done` tsconfig.json (raw GitHub) · **quality:** HIGH real-world · **as of:** 2026-06-27
- **bears on:** The `tsconfig.json` for `extensions/pi/`.

#### Canonical tsconfig.json:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "esModuleInterop": true,
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "skipLibCheck": true,
    "noEmit": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules"]
}
```

**Why `noEmit`: true:** Pi's extension loader handles TypeScript directly (likely via tsx/esbuild-register internally). There is no `tsc` compilation step and no `dist/` directory. The `typecheck` script runs `tsc --noEmit` purely for CI validation.

### 4. Binary Dependency Handling (rod-cli)

- **finding:** Pi extensions handle external binary dependencies through three patterns: (1) document the prerequisite, (2) verify the binary on `session_start` and notify if missing, (3) shell out via `pi.exec()` or Node's `child_process`. No npm dependency on the binary itself — it's a system prerequisite. This is the same pattern used by extensions that depend on `git`, `docker`, or API-key-gated services.
- **source:** https://pi.dev/docs/latest/extensions (pi.exec APIDOC) · **quality:** HIGH official
- **source:** `pi-lovely-web` README (API keys as prerequisite) · **quality:** MEDIUM community pattern · **as of:** 2026-06-27
- **bears on:** How the extension finds, verifies, and invokes the rod-cli binary.

#### Recommended binary dependency pattern:

```typescript
// src/cli.ts — shared rod-cli shell wrapper
import { execSync, exec } from "node:child_process";
import { existsSync } from "node:fs";
import { join } from "node:path";

const ROD_CLI = "rod-cli";

/** Resolve rod-cli binary path. Returns null if not found. */
export function findRodCli(): string | null {
  // 1. Check explicit env var
  if (process.env.ROD_CLI_PATH && existsSync(process.env.ROD_CLI_PATH)) {
    return process.env.ROD_CLI_PATH;
  }
  // 2. Check PATH
  const pathDirs = (process.env.PATH || "").split(":");
  for (const dir of pathDirs) {
    const candidate = join(dir, ROD_CLI);
    if (existsSync(candidate)) return candidate;
  }
  // 3. Check common Go install locations
  const gobin = process.env.GOBIN || join(process.env.HOME || "/root", "go", "bin");
  const goCandidate = join(gobin, ROD_CLI);
  if (existsSync(goCandidate)) return goCandidate;

  return null;
}

/** Execute rod-cli with args. Throws on non-zero exit. */
export async function execRodCli(
  pi: { exec: (cmd: string, args: string[], opts?: any) => Promise<any> },
  args: string[],
  opts?: { signal?: AbortSignal; timeout?: number }
): Promise<{ stdout: string; stderr: string; code: number }> {
  const result = await pi.exec(ROD_CLI, ["--raw", ...args], {
    signal: opts?.signal,
    timeout: opts?.timeout ?? 30_000,
  });
  if (result.code !== 0) {
    throw new Error(`rod-cli exited ${result.code}: ${result.stderr}`);
  }
  return result;
}
```

```typescript
// src/lifecycle.ts — session hooks
export function registerLifecycle(pi: ExtensionAPI, rodCliPath: string | null) {
  pi.on("session_start", async (_event, ctx) => {
    if (!rodCliPath) {
      ctx.ui.notify(
        "rod-cli not found on PATH. Install with: go install github.com/agenthands/rod-cli@latest",
        "warn"
      );
      return;
    }
    // Verify it actually works
    const result = await pi.exec("rod-cli", ["--version"], { timeout: 5000 });
    if (result.code !== 0) {
      ctx.ui.notify(`rod-cli found but failed: ${result.stderr}`, "error");
      return;
    }
    ctx.ui.notify(`rod-cli ${result.stdout.trim()} ready`, "info");
  });

  pi.on("session_end", async () => {
    // Tear down daemon — best-effort, don't throw
    try { await pi.exec("rod-cli", ["close"], { timeout: 5000 }); } catch {}
  });
}
```

**Why `pi.exec` over raw `child_process`:**
- `pi.exec` integrates with Pi's abort signals (the `signal` parameter propagates to the child process)
- `pi.exec` gets consistent PATH resolution within Pi's environment
- It's the documented, supported extension API — raw `child_process` may miss environment setup Pi performs

### 5. Directory Structure

- **finding:** The community convention is `src/index.ts` as the single entry point, with tool implementations in `src/tools/`, shared helpers in sibling files, and tests in a top-level `test/` or `src/__tests__/` directory. Pi discovers the extension via the `pi.extensions` field in `package.json`.
- **source:** `@harms-haus/pi-til-done` repo structure · **quality:** HIGH real-world · **as of:** 2026-06-27
- **source:** https://pi.dev/docs/latest/extensions · **quality:** HIGH official

#### Recommended structure for rod-cli Pi extension:

```
extensions/pi/
├── package.json            # npm metadata + pi.extensions entry
├── tsconfig.json           # TypeScript config (noEmit)
├── README.md               # Install + usage docs
├── CHANGELOG.md
├── LICENSE                 # MIT
├── biome.json              # or .prettierrc / eslint.config.js
│
├── src/
│   ├── index.ts            # Default export: wire everything
│   ├── cli.ts              # findRodCli() + exec() wrapper
│   ├── lifecycle.ts        # session_start/end hooks
│   ├── types.ts            # Shared TypeBox parameter schemas
│   └── tools/
│       ├── goto.ts         # browse_goto
│       ├── snapshot.ts     # browse_snapshot
│       ├── click.ts        # browse_click
│       ├── type.ts         # browse_type
│       ├── eval.ts         # browse_eval
│       ├── screenshot.ts   # browse_screenshot
│       └── wait.ts         # browse_wait
│
└── src/__tests__/
    └── smoke.test.ts       # Integration smoke test
```

#### Entry point shape (`src/index.ts`):

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { findRodCli } from "./cli.js";
import { registerLifecycle } from "./lifecycle.js";
import { registerBrowseGoto } from "./tools/goto.js";
import { registerBrowseSnapshot } from "./tools/snapshot.js";
// ...etc

export default function (pi: ExtensionAPI) {
  const rodCliPath = findRodCli();

  registerLifecycle(pi, rodCliPath);
  registerBrowseGoto(pi);
  registerBrowseSnapshot(pi);
  registerBrowseClick(pi);
  registerBrowseType(pi);
  registerBrowseEval(pi);
  registerBrowseScreenshot(pi);
  registerBrowseWait(pi);
}
```

### 6. Install & Distribution Flow

- **finding:** End users install via `pi install npm:@agenthands/rod-cli-pi`. Pi reads `package.json` → `pi.extensions` → loads the TypeScript entry point. The rod-cli Go binary must be installed separately (`go install github.com/agenthands/rod-cli@latest`). Extensions can also be used from local paths (`pi install ./extensions/pi`) or git repos.
- **source:** https://pi.dev/docs/latest/packages · **quality:** HIGH official
- **source:** docs/onboarding/pi.md (existing rod-cli Pi skill docs) · **quality:** HIGH project docs
- **bears on:** The README instructions and onboarding flow.

#### Install commands (end-user flow):

```bash
# 1. Install rod-cli Go binary
go install github.com/agenthands/rod-cli@latest
rod-cli install   # download Chromium

# 2. Install the Pi extension
pi install npm:@agenthands/rod-cli-pi

# 3. Verify in a Pi session — ask the agent to browse a page
```

### 7. Parameter Types (TypeBox + StringEnum)

- **finding:** Tools use `Type.Object()` with `Type.String()`, `Type.Optional()`, `Type.Boolean()`, `Type.Number()`, and `Type.Array()`. For enum parameters, use `StringEnum([...] as const)` from `@earendil-works/pi-ai` instead of `Type.Enum()` — the latter generates `anyOf`/`const` patterns unsupported by Google's API.
- **source:** https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/extensions.md · **quality:** HIGH source
- **source:** https://github.com/sinclairzx81/typebox · **quality:** HIGH official · **as of:** TypeBox latest
- **bears on:** Every tool's `parameters` field.

#### Example tool parameter schemas for rod-cli:

```typescript
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";

// browse_goto parameters
const GotoParams = Type.Object({
  url: Type.String({ description: "URL to navigate to" }),
  session: Type.Optional(Type.String({ description: "Named session identifier" })),
  timeout: Type.Optional(Type.Number({ description: "Navigation timeout in milliseconds", default: 30000 })),
});

// browse_click parameters
const ClickParams = Type.Object({
  selector: Type.String({ description: "CSS selector of the element to click" }),
  session: Type.Optional(Type.String({ description: "Named session identifier" })),
});

// browse_screenshot parameters
const ScreenshotParams = Type.Object({
  name: Type.Optional(Type.String({ description: "Filename for the screenshot (PNG)" })),
  fullPage: Type.Optional(Type.Boolean({ description: "Capture full scrollable page", default: false })),
  session: Type.Optional(Type.String({ description: "Named session identifier" })),
});
```

### 8. Testing Approach

- **finding:** Community Pi extensions use `vitest` for testing (fast, ESM-native, TypeScript-native). Tests are type-checked with `tsc --noEmit`. A smoke test that shells out to the real rod-cli binary against a local fixture page is the standard integration test pattern.
- **source:** `@harms-haus/pi-til-done` package.json scripts · **quality:** HIGH real-world · **as of:** 2026-06-27
- **bears on:** The test infrastructure and CI pipeline.

#### Recommended test setup:

```typescript
// src/__tests__/smoke.test.ts
import { describe, it, expect } from "vitest";

describe("rod-cli binary", () => {
  it("is on PATH", () => {
    // Verify findRodCli resolves
  });

  it("can navigate and snapshot", async () => {
    // Integration: start daemon, goto fixture page, snapshot, close
  });
});
```

---

## Tradeoffs / Alternatives

- **`pi.exec` vs `child_process.spawn`:** `pi.exec` is the documented API and integrates with Pi's abort signals. Using raw `child_process` gives more control over stdio streaming but loses signal integration. **Recommendation:** use `pi.exec` for all tool invocations; it covers the rod-cli use case (short-lived commands with text output) perfectly.
- **`StringEnum` vs `Type.Enum`:** `StringEnum` is explicitly documented as required for Google API compatibility. Even though Pi may not always use Google, this is the convention in all official examples. **Recommendation:** always use `StringEnum`.
- **`noEmit` vs compiled JS:** Pi loads TypeScript directly; compiling to JS would add a build step with no benefit. The community standard is `noEmit: true`. **Recommendation:** no compilation, ship `.ts` source.
- **Scoped vs unscoped npm name:** All community extensions use scoped names (`@org/name`). An unscoped name (`rod-cli-pi`) would also work but is less conventional. **Recommendation:** `@agenthands/rod-cli-pi`.

---

## Unknowns

- **Pi's TypeScript loader internals:** Pi loads `.ts` files directly, but the exact loader (tsx, esbuild-register, swc, etc.) is not documented. This could affect whether certain TypeScript features (decorators, const enum, etc.) work. **Mitigation:** stick to standard TypeScript features (no decorators, no `const enum`); if issues arise, test against a local `pi -e ./extensions/pi/src/index.ts`.
- **AbortSignal propagation through `pi.exec`:** The docs show `signal` is passed to `pi.exec`, but the exact behaviour (SIGTERM vs SIGKILL, child process tree cleanup) is not specified. **How to settle:** write a quick test — spawn `pi.exec("sleep", ["60"])` with a 1s AbortSignal timeout and verify the process is killed.
- **`terminate: true` semantics in tool results:** The docs mention `terminate: true` stops the batch "when every finalized tool result in the batch also returns terminate: true." The exact batch semantics may affect how rod-cli tools interact with other tools in the same turn. **How to settle:** test with multiple rod-cli tools in a single prompt and observe batching behaviour.
- **npm package discovery via `pi-package` keyword:** It's unclear whether Pi has a registry search that indexes the `pi-package` keyword, or if discovery is purely through word-of-mouth / GitHub. **Impact:** low — the extension is distributed via the rod-cli repo and install instructions; npm is the transport, not the discovery mechanism.
