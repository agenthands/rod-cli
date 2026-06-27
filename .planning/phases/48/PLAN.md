---
author: architect
responsible: architect
phase: 48
status: ready
parent_artifacts:
  - .planning/phases/48/CONTEXT.md
  - .planning/REQUIREMENTS.md
---

# Phase 48: Core Browser Tools + Integration Test — PLAN

## §1 — Barrel export + shared pattern (TOOLS-01..07 prep)

Write `extensions/pi/src/tools/index.ts`:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { registerBrowseGoto } from "./goto";
import { registerBrowseSnapshot } from "./snapshot";
import { registerBrowseClick } from "./click";
import { registerBrowseType } from "./type";
import { registerBrowseEval } from "./eval";
import { registerBrowseScreenshot } from "./screenshot";
import { registerBrowseWait } from "./wait";

export function registerAllCoreTools(pi: ExtensionAPI) {
  registerBrowseGoto(pi);
  registerBrowseSnapshot(pi);
  registerBrowseClick(pi);
  registerBrowseType(pi);
  registerBrowseEval(pi);
  registerBrowseScreenshot(pi);
  registerBrowseWait(pi);
}
```

Update `extensions/pi/src/index.ts` to call `registerAllCoreTools(pi)` after lifecycle.

## §2 — browse_goto (TOOLS-01)

Write `extensions/pi/src/tools/goto.ts`:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseGoto(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_goto",
    label: "Browse Goto",
    description: "Navigate the browser to a URL. The browser daemon starts automatically on first use.",
    promptSnippet: "Navigate to a URL: browse_goto(url=\"https://example.com\")",
    promptGuidelines: [
      "Use browse_goto to navigate to any URL before interacting with the page.",
      "The browser starts automatically on the first browse_goto call — no manual setup needed.",
      "Use the session parameter for multi-tab or multi-identity workflows.",
    ],
    parameters: Type.Object({
      url: Type.String({ description: "Full URL to navigate to (https://...)" }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["goto", params.url];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }] };
    },
  });
}
```

## §3 — browse_snapshot (TOOLS-02)

Write `extensions/pi/src/tools/snapshot.ts`:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseSnapshot(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_snapshot",
    label: "Browse Snapshot",
    description: "Capture a token-efficient accessibility-tree markdown snapshot of the current page. Output is truncated to 50KB/2000 lines.",
    promptSnippet: "Get page content: browse_snapshot() or browse_snapshot(selector=\"#main\")",
    promptGuidelines: [
      "Prefer browse_snapshot over browse_screenshot for reading page structure and text content.",
      "Do NOT call browse_snapshot after every action to confirm results — use it when you need to read the page.",
      "Use the optional selector parameter to scope the snapshot to a specific element.",
    ],
    parameters: Type.Object({
      selector: Type.Optional(Type.String({ description: "CSS selector to scope snapshot to a specific element" })),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["snapshot"];
      if (params.selector) args.push("--selector", params.selector);
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }] };
    },
  });
}
```

## §4 — browse_click (TOOLS-03)

Write `extensions/pi/src/tools/click.ts`:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseClick(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_click",
    label: "Browse Click",
    description: "Click an element by CSS selector.",
    promptSnippet: "Click an element: browse_click(selector=\"#submit-btn\") or browse_click(selector=\"a.link\", doubleClick=true)",
    promptGuidelines: [
      "Use browse_click for clicking buttons, links, and other interactive elements.",
      "Use the doubleClick parameter for double-click actions.",
      "After clicking, use browse_snapshot or browse_wait to confirm the page updated.",
    ],
    parameters: Type.Object({
      selector: Type.String({ description: "CSS selector of the element to click" }),
      doubleClick: Type.Optional(Type.Boolean({ description: "Perform a double click instead of single click", default: false })),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["click", params.selector];
      if (params.doubleClick) args.push("--double");
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }] };
    },
  });
}
```

## §5 — browse_type (TOOLS-04)

Write `extensions/pi/src/tools/type.ts`:

Maps 1:1 to `rod-cli type <sel> <text>` (humanized typing, sequential keystrokes, no submit).
`browse_fill_form` in Phase 49 handles instant fill + submit via `rod-cli fill`.

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseType(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_type",
    label: "Browse Type",
    description: "Type text into an input or textarea with humanized keystroke timing.",
    promptSnippet: "Type text: browse_type(selector=\"#input\", text=\"hello world\")",
    promptGuidelines: [
      "Use browse_type for typing into text inputs and textareas with human-like timing.",
      "browse_type does NOT submit forms — use browse_click on the submit button or browse_fill_form (instant fill with submit option).",
      "For filling multiple fields at once, prefer browse_fill_form.",
    ],
    parameters: Type.Object({
      selector: Type.String({ description: "CSS selector of the input or textarea element" }),
      text: Type.String({ description: "Text to type into the element" }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["type", params.selector, params.text];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }] };
    },
  });
}
```

## §6 — browse_eval (TOOLS-05)

Write `extensions/pi/src/tools/eval.ts`:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseEval(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_eval",
    label: "Browse Eval",
    description: "Evaluate JavaScript on the page. Use as an escape hatch for data extraction when browse_snapshot is insufficient.",
    promptSnippet: "Run JS: browse_eval(expression=\"document.title\") or browse_eval(expression=\"JSON.stringify(window.__data)\")",
    promptGuidelines: [
      "Use browse_eval as an escape hatch for data extraction that browse_snapshot cannot provide.",
      "Prefer browse_click and browse_type for standard interactions — do not use browse_eval to click or type.",
      "Expressions are capped at 10KB. For larger scripts, break into multiple calls.",
      "Return JSON-serializable values when possible — the result is returned as text.",
    ],
    parameters: Type.Object({
      expression: Type.String({ description: "JavaScript expression to evaluate on the page (max 10KB)" }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["eval", params.expression];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }] };
    },
  });
}
```

## §7 — browse_screenshot (TOOLS-06)

Write `extensions/pi/src/tools/screenshot.ts`:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseScreenshot(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_screenshot",
    label: "Browse Screenshot",
    description: "Capture a screenshot of the current page or a specific element.",
    promptSnippet: "Screenshot: browse_screenshot() or browse_screenshot(selector=\"#chart\", format=\"png\", fullPage=true)",
    promptGuidelines: [
      "Use browse_screenshot for visual verification of page state.",
      "Prefer browse_snapshot for reading text content — screenshots are larger and slower.",
      "Use fullPage=true to capture the entire scrollable page, not just the viewport.",
    ],
    parameters: Type.Object({
      selector: Type.Optional(Type.String({ description: "CSS selector to screenshot a specific element" })),
      fullPage: Type.Optional(Type.Boolean({ description: "Capture the full scrollable page", default: false })),
      format: Type.Optional(StringEnum(["png", "jpeg"] as const, { description: "Image format", default: "png" })),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["screenshot"];
      if (params.selector) args.push("--selector", params.selector);
      if (params.fullPage) args.push("--full-page");
      if (params.format) args.push("--format", params.format);
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }] };
    },
  });
}
```

## §8 — browse_wait (TOOLS-07)

Write `extensions/pi/src/tools/wait.ts`:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseWait(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_wait",
    label: "Browse Wait",
    description: "Wait for a CSS selector to appear on the page or for a fixed duration.",
    promptSnippet: "Wait: browse_wait(selector=\"#loaded\") or browse_wait(timeout=3000)",
    promptGuidelines: [
      "Use browse_wait to wait for async page content to load before taking a snapshot.",
      "Prefer waiting for a specific selector over a fixed timeout — it's more reliable.",
      "Use browse_wait after browse_goto to ensure the page has loaded.",
    ],
    parameters: Type.Object({
      selector: Type.Optional(Type.String({ description: "CSS selector to wait for" })),
      timeout: Type.Optional(Type.Number({ description: "Maximum time to wait in milliseconds" })),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["wait"];
      if (params.selector) args.push(params.selector);
      else if (params.timeout) args.push("--timeout", String(params.timeout));
      else args.push("--timeout", "5000"); // default 5s
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }] };
    },
  });
}
```

## §9 — Integration test (INTEG-01)

Write `extensions/pi/src/__tests__/integration.test.ts`:

```typescript
import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { createServer } from "node:http";
import type { Server } from "node:http";
import { findRodCli, execRodCli } from "../cli";

const FIXTURE_HTML = `<!DOCTYPE html>
<html><head><title>Integration Fixture</title></head>
<body>
  <h1>Fixture Page</h1>
  <p id="msg">Hello from fixture</p>
  <input id="name" type="text" placeholder="Enter name" />
  <button id="greet">Greet</button>
</body></html>`;

describe("integration workflow", () => {
  const rodCliPath = findRodCli();

  describe.skipIf(!rodCliPath)("with rod-cli", () => {
    let server: Server;
    let url: string;

    beforeAll(async () => {
      server = createServer((_req, res) => {
        res.writeHead(200, { "content-type": "text/html" });
        res.end(FIXTURE_HTML);
      });
      await new Promise<void>(r => server.listen(0, () => r()));
      const addr = server.address();
      url = `http://127.0.0.1:${typeof addr === "object" ? addr?.port : ""}`;
    });

    afterAll(async () => {
      try { await execRodCli(["close"]); } catch { /* ok if already closed */ }
      await new Promise<void>(r => server.close(() => r()));
    });

    it("goto navigates and snapshot reads content", async () => {
      const r1 = await execRodCli(["goto", url]);
      expect(r1.code).toBe(0);

      const r2 = await execRodCli(["snapshot"]);
      expect(r2.code).toBe(0);
      expect(r2.stdout).toContain("Fixture Page");
    });

    it("eval returns page title", async () => {
      const r = await execRodCli(["eval", "document.title"]);
      expect(r.code).toBe(0);
      expect(r.stdout).toContain("Integration Fixture");
    });

    it("type + click workflow", async () => {
      const r1 = await execRodCli(["type", "#name", "World"]);
      expect(r1.code).toBe(0);

      const r2 = await execRodCli(["click", "#greet"]);
      expect(r2.code).toBe(0);
    });

    it("screenshot produces output", async () => {
      const r = await execRodCli(["screenshot", "--format", "png"]);
      expect(r.code).toBe(0);
    });

    it("wait finds element", async () => {
      const r = await execRodCli(["wait", "#msg"]);
      expect(r.code).toBe(0);
    });
  });

  it("smoke: rod-cli is available or null", () => {
    expect(rodCliPath === null || typeof rodCliPath === "string").toBe(true);
  });
});
```

## §10 — Build verification gate

1. `npx tsc --noEmit --project extensions/pi/tsconfig.json` — no type errors
2. `npx vitest run --config extensions/pi/vitest.config.ts` — all tests pass
   (integration tests skip if rod-cli not on PATH; existing 107 Phase 47 tests still pass)

## Execution order

1. §1: Write `tools/index.ts` barrel + update `src/index.ts`
2. §2-§8: Write 7 tool files (goto, snapshot, click, type, eval, screenshot, wait)
3. §9: Write integration test
4. §10: tsc + vitest gate

## Files changed

| File | § | Status |
|------|---|--------|
| `extensions/pi/src/index.ts` | §1 | MODIFIED (add registerAllCoreTools call) |
| `extensions/pi/src/tools/index.ts` | §1 | NEW |
| `extensions/pi/src/tools/goto.ts` | §2 | NEW |
| `extensions/pi/src/tools/snapshot.ts` | §3 | NEW |
| `extensions/pi/src/tools/click.ts` | §4 | NEW |
| `extensions/pi/src/tools/type.ts` | §5 | NEW |
| `extensions/pi/src/tools/eval.ts` | §6 | NEW |
| `extensions/pi/src/tools/screenshot.ts` | §7 | NEW |
| `extensions/pi/src/tools/wait.ts` | §8 | NEW |
| `extensions/pi/src/__tests__/integration.test.ts` | §9 | NEW |
