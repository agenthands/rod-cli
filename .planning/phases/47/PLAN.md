---
author: architect
responsible: architect
phase: 47
status: ready
parent_artifacts:
  - .planning/phases/47/CONTEXT.md
  - .planning/REQUIREMENTS.md
---

# Phase 47: Extension Foundation + Lifecycle — PLAN

## §1 — Package scaffold (FOUND-01)

Create `extensions/pi/package.json`:
```json
{
  "name": "@agenthands/rod-cli-pi",
  "version": "0.1.0",
  "description": "Pi extension for rod-cli — native browser automation tools for the Pi coding agent",
  "license": "MIT",
  "pi": { "extensions": ["./src/index.ts"] },
  "peerDependencies": {
    "@earendil-works/pi-coding-agent": "*",
    "@earendil-works/pi-ai": "*"
  },
  "devDependencies": {
    "typebox": "*",
    "vitest": "^1"
  },
  "scripts": {
    "test": "vitest run"
  }
}
```

Create `extensions/pi/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "noEmit": true,
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true
  },
  "include": ["src"]
}
```

Create `extensions/pi/.gitignore`:
```
node_modules/
```

Grounding: no SMTC needed (pure scaffold). Verification: `cat package.json | jq '.pi.extensions'` returns `["./src/index.ts"]`.

## §2 — TypeBox schemas (shared types) (FOUND-02 prep)

Write `extensions/pi/src/types.ts`:
```typescript
import { Type } from "typebox";

/** Shared session parameter — optional on every tool. */
export const SessionParam = Type.Optional(Type.String({
  description: "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). " +
    "Omit to use the default session.",
}));
```

## §3 — Binary discovery + shell-out wrapper (FOUND-03, FOUND-04)

Write `extensions/pi/src/cli.ts`:

### `findRodCli(): string | null`
```typescript
import { platform } from "node:os";

const IS_WINDOWS = platform() === "win32";
const BINARY_NAME = IS_WINDOWS ? "rod-cli.exe" : "rod-cli";
const PATH_SEP = IS_WINDOWS ? ";" : ":";

export function findRodCli(): string | null {
  // 1. ROD_CLI_PATH env var
  const envPath = process.env.ROD_CLI_PATH;
  if (envPath) return envPath;

  // 2. Scan PATH
  const pathDirs = (process.env.PATH || "").split(PATH_SEP);
  for (const dir of pathDirs) {
    const candidate = `${dir}/${BINARY_NAME}`;
    try {
      // On Windows, just check if the file exists without .exe too
      // (Node's spawn handles PATHEXT)
      const { statSync } = require("node:fs");
      if (statSync(candidate, { throwIfNoEntry: false })?.isFile()) return candidate;
    } catch { /* continue */ }
  }

  // 3. Go install locations
  const home = IS_WINDOWS ? process.env.USERPROFILE : process.env.HOME;
  if (home) {
    const goBin = process.env.GOBIN || `${home}/go/bin`;
    const candidate = `${goBin}/${BINARY_NAME}`;
    try {
      const { statSync } = require("node:fs");
      if (statSync(candidate, { throwIfNoEntry: false })?.isFile()) return candidate;
    } catch { /* continue */ }
  }

  return null;
}
```

### `execRodCli(args, opts): Promise<{stdout: string, stderr: string, code: number}>`

Timeout table (from FOUND-04):
| Command pattern | Timeout (ms) |
|---|---|
| `goto` | 60000 |
| `snapshot`, `click`, `fill`, `type`, `eval` | 15000 |
| `screenshot`, `wait` | 30000 |
| `close`, `--version` | 5000 |
| default | 30000 |

Input validation (red-team recommendation):
- `goto <url>`: URL must start with `http://` or `https://`
- `click <selector>`, `type <selector>`, `fill <selector>`: selector must be non-empty
- `eval <expression>`: expression must be ≤10KB
- `fill <selector> <text>`: text must be non-empty
- Any required param: must be non-empty string

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

let _pi: ExtensionAPI | null = null;
export function setPi(pi: ExtensionAPI) { _pi = pi; }

const TIMEOUTS: Record<string, number> = {
  goto: 60_000, snapshot: 15_000, click: 15_000, fill: 15_000,
  type: 15_000, eval: 15_000, screenshot: 30_000, wait: 30_000,
  close: 5_000, "--version": 5_000,
};
const DEFAULT_TIMEOUT = 30_000;

function timeoutFor(args: string[]): number {
  for (const [cmd, ms] of Object.entries(TIMEOUTS)) {
    if (args.includes(cmd)) return ms;
  }
  return DEFAULT_TIMEOUT;
}

function validateInput(args: string[]): void {
  const cmdIndex = args.findIndex(a => !a.startsWith("-"));
  if (cmdIndex < 0) return; // no subcommand, e.g. --version
  const cmd = args[cmdIndex];
  const rest = args.slice(cmdIndex + 1);

  if (cmd === "goto" && rest[0]) {
    if (!rest[0].startsWith("http://") && !rest[0].startsWith("https://")) {
      throw new Error(`browse_goto requires an http:// or https:// URL, got: ${rest[0].slice(0, 80)}`);
    }
  }
  if ((cmd === "click" || cmd === "fill" || cmd === "type") && (!rest[0] || rest[0].trim() === "")) {
    throw new Error(`${cmd} requires a non-empty CSS selector`);
  }
  if (cmd === "eval" && rest[0] && rest[0].length > 10_000) {
    throw new Error(`eval expression too long (${rest[0].length} bytes, max 10000)`);
  }
}

export async function execRodCli(args: string[], opts?: { signal?: AbortSignal }): Promise<{stdout: string, stderr: string, code: number}> {
  if (!_pi) throw new Error("Extension not initialized — pi reference not set");
  validateInput(args);
  const timeout = timeoutFor(args);
  return _pi.exec("rod-cli", args, { signal: opts?.signal, timeout });
}
```

## §4 — Lifecycle hooks (LIFECYCLE-01, LIFECYCLE-02)

Write `extensions/pi/src/lifecycle.ts`:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { findRodCli } from "./cli";

export function registerLifecycle(pi: ExtensionAPI, rodCliPath: string | null) {
  pi.on("session_start", async (_event, ctx) => {
    if (!rodCliPath) {
      ctx.ui.notify(
        "rod-cli not found. Install: go install github.com/agenthands/rod-cli@latest",
        "warn"
      );
      return;
    }
    try {
      const result = await pi.exec("rod-cli", ["--version"], { timeout: 5000 });
      const version = result.stdout.trim() || "unknown";
      ctx.ui.notify(`rod-cli ${version} ready`, "info");
    } catch {
      ctx.ui.notify(
        "rod-cli found but --version failed. Check your installation.",
        "warn"
      );
    }
  });

  pi.on("session_shutdown", async (event) => {
    // Only close daemon on actual quit — not on reload/fork/resume
    if (event.reason !== "quit") return;
    try {
      await pi.exec("rod-cli", ["close"], { timeout: 5000 });
    } catch {
      // Best-effort — daemon has its own PPID polling + idle timeout
    }
  });
}
```

## §5 — Entry point (FOUND-02)

Write `extensions/pi/src/index.ts`:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { findRodCli, setPi } from "./cli";
import { registerLifecycle } from "./lifecycle";

export default function (pi: ExtensionAPI) {
  setPi(pi);
  const rodCliPath = findRodCli();
  registerLifecycle(pi, rodCliPath);
  // Tools will be registered in Phase 48-49
}
```

## §6 — Vitest smoke test (FOUND-05)

Write `extensions/pi/vitest.config.ts`:
```typescript
import { defineConfig } from "vitest/config";
export default defineConfig({ test: { include: ["src/__tests__/**/*.test.ts"] } });
```

Write `extensions/pi/src/__tests__/smoke.test.ts`:
```typescript
import { describe, it, expect } from "vitest";

// Test that the source files parse and export correctly
describe("extension foundation", () => {
  it("findRodCli resolves or returns null", async () => {
    const { findRodCli } = await import("../cli");
    const result = findRodCli();
    // On a dev machine with rod-cli installed: string.
    // In CI without rod-cli: null.
    // Either is acceptable — we just verify it doesn't throw.
    expect(result === null || typeof result === "string").toBe(true);
  });

  it("types module exports SessionParam", async () => {
    const { SessionParam } = await import("../types");
    expect(SessionParam).toBeDefined();
  });

  it("lifecycle module exports registerLifecycle", async () => {
    const { registerLifecycle } = await import("../lifecycle");
    expect(typeof registerLifecycle).toBe("function");
  });

  it("index module exports default function", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });
});
```

## §7 — Build verification gate

After all files are written:
1. `npm install --prefix extensions/pi` (installs vitest)
2. `npx vitest run --config extensions/pi/vitest.config.ts` — all tests pass
3. `npx tsc --noEmit --project extensions/pi/tsconfig.json` — no type errors

## Execution order

1. Write package.json + tsconfig.json + .gitignore (§1)
2. Write src/types.ts (§2)
3. Write src/cli.ts (§3)
4. Write src/lifecycle.ts (§4)
5. Write src/index.ts (§5)
6. Write vitest.config.ts + smoke test (§6)
7. Run npm install + vitest + tsc (§7)

## Files changed (all new)

| File | § |
|------|---|
| `extensions/pi/package.json` | §1 |
| `extensions/pi/tsconfig.json` | §1 |
| `extensions/pi/.gitignore` | §1 |
| `extensions/pi/src/types.ts` | §2 |
| `extensions/pi/src/cli.ts` | §3 |
| `extensions/pi/src/lifecycle.ts` | §4 |
| `extensions/pi/src/index.ts` | §5 |
| `extensions/pi/vitest.config.ts` | §6 |
| `extensions/pi/src/__tests__/smoke.test.ts` | §6 |
