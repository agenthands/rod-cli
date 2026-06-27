---
author: anvil-researcher
responsible: architect
emitted_at: 2026-06-27T20:00:00Z
phase: null
artifact_kind: research
parent_artifacts:
  - .planning/PROJECT.md
  - .planning/research/STACK.md
status: draft
---

# Research — Pi Extension: FEATURES

**Answer:** A rod-cli Pi extension should expose 18–22 tools under the `browse_` prefix,
delivered in 3 waves: 7 core table-stakes tools (MVP), 8 interaction/lifecycle tools (wave
2), and 4–7 storage/advanced tools (wave 3). The flat `browse_` prefix is the Pi ecosystem
convention — the dominant `pi-browser` extension (Playwright-backed, 50 tools) uses
`browser_<verb>` and no tool uses hierarchical nesting. All tools accept an optional
`session` parameter (named sessions), and the daemon lifecycle is managed via hooks
(`session_end` → `rod-cli close`), not user-facing tools. **Confidence: HIGH** —
ecosystem convention verified against live `pi-browser` source, Pi official docs, and the
`@shog-lab/pi-toolkit` published package.

**Recommendation:** Ship the 7 core tools first (Table Stakes below). The daemon model
is rod-cli's key architectural differentiator — `session_start`/`session_end` hooks
manage it transparently. Add extended tools after the smoke test passes. Keep the flat
`browse_` prefix; Pi tools don't nest.

---

## Findings

### 1. Tool Naming Convention — Flat `browse_` Prefix

- **finding:** The Pi ecosystem uses flat `snake_case` tool names with a function-group
  prefix. The dominant browser extension (`pi-browser`, Playwright-backed, 50 tools) uses
  `browser_<verb>` (e.g., `browser_navigate`, `browser_snapshot`, `browser_click`,
  `browser_cookie_set`). Other extensions follow the same pattern: `steel_<verb>` for
  Steel browser, `<server>_<tool>` for MCP bridges. Pi built-in tools are bare verbs
  (`read`, `bash`, `edit`, `write`, `grep`). No hierarchical nesting (e.g.,
  `browser.tab.switch`) exists in any published Pi extension.
- **source:** https://github.com/larsderidder/pi-browser (raw source, 50 registered
  tools) · **quality:** HIGH real-world · **as of:** 2026-06-27
- **source:** https://www.npmjs.com/package/@shog-lab/pi-toolkit (README: "Tool names
  stay snake_case") · **quality:** HIGH published package · **as of:** 2026-06-03
- **source:** https://pi.dev/docs/latest/extensions (official docs — all examples use
  `snake_case` names) · **quality:** HIGH official
- **bears on:** Every tool name choice. The PROJECT.md already targets `browse_*`; this
  confirms it.

#### Naming rules, synthesized from ecosystem practice:

| Rule | Source | Example |
|------|--------|---------|
| Flat `browse_` prefix | pi-browser, pi-steel, pi-toolkit | `browse_goto`, `browse_click` |
| Verb-first | All browser extensions | `browse_navigate`, `browse_type` |
| `snake_case` | pi-toolkit README explicitly | `browse_fill_form`, not `browseFillForm` |
| `_xy` suffix for coordinate ops | pi-browser mouse tools | `browse_mouse_move_xy` |
| Storage tools: `<prefix>_<storage>_<verb>` | pi-browser | `browse_localstorage_get`, `browse_cookie_set` |
| Multi-action tools use a verb+noun discriminator | pi-browser `browser_tabs` (list/create/close/select) | — |

### 2. Tool Granularity — One Tool Per Command, Not Combined

- **finding:** The pi-browser extension registers 50 individual tools — one per browser
  action. Exceptions that combine multiple actions into one tool are rare and use an
  explicit `action` parameter: `browser_tabs` takes `action: "list"|"new"|"close"|"select"`,
  `browser_handle_dialog` takes `accept: boolean`. The Pi docs show tools like
  `db_connect`, `db_query`, `db_close` — each a separate registration, sharing state
  through a closure variable. **No extension combines unrelated actions into one tool.**
- **source:** pi-browser `src/index.ts` (50 `defineTool` calls, one per action) ·
  **quality:** HIGH real-world
- **source:** https://pi.dev/docs/latest/extensions (db_connect/db_query/db_close
  example) · **quality:** HIGH official
- **bears on:** Whether rod-cli's `fill` + `type` should be one tool or two; whether tab
  management is one tool or five. Answer: **separate registration per concept, combined
  only when an explicit `action` enum parameter discriminates.**

#### Granularity decision matrix for rod-cli:

| rod-cli commands | Pi tool strategy | Rationale |
|---|---|---|
| `goto [url]` | `browse_goto` (1 tool) | Single clear action |
| `snapshot` | `browse_snapshot` (1 tool) | Single action; optional `selector` param |
| `click [sel]`, `dblclick [sel]` | `browse_click` (1 tool, `doubleClick` param) | Same concept, boolean discriminator |
| `type [sel] [text]`, `fill [sel] [text]` | `browse_type` (1 tool, `submit` param) | `fill` is `type` + submit; use `slowly` boolean |
| `eval [script]` | `browse_eval` (1 tool) | Single action |
| `screenshot`, `pdf` | `browse_screenshot` (1 tool, `format` param) | Same concept (page capture), format discriminator |
| `wait [sel]` | `browse_wait` (1 tool) | Single action; also accept `time` and `text` params |
| `tab-list`, `tab-new`, `tab-select`, `tab-close` | `browse_tabs` (1 tool, `action` StringEnum) | Grouped by target (tabs); follows pi-browser convention |
| `reload`, `go-back`, `go-forward` | `browse_navigate` (1 tool, `action` StringEnum) | Grouped by target (navigation); follows pi-browser |
| `cookie-get/set/delete/clear` | `browse_cookie_get`, `browse_cookie_set`, `browse_cookie_delete`, `browse_cookie_clear` (4 tools) | Follows pi-browser storage tool pattern exactly |
| `localstorage-*`, `sessionstorage-*` | Same 4-verb pattern per storage (8 tools) | Parallel structure, like pi-browser |
| `press [key]` | `browse_press_key` (1 tool) | Follows pi-browser naming |
| `hover [sel]` | `browse_hover` (1 tool) | Single action |
| `select [sel] [values]` | `browse_select_option` (1 tool) | Follows pi-browser naming |
| `scroll` | `browse_scroll` (1 tool, `deltaX`/`deltaY` params) | Single action |
| `upload [sel] [file]` | `browse_file_upload` (1 tool) | Follows pi-browser naming |
| `drag [start] [end]` | `browse_drag` (1 tool) | Single action |
| `resize [w] [h]` | `browse_resize` (1 tool) | Single action |
| `console` | `browse_console` (1 tool) | Single action |
| `dialog-accept`, `dialog-dismiss` | `browse_handle_dialog` (1 tool, `accept` boolean) | Follows pi-browser pattern |
| `route`, `route-list`, `unroute`, `requests`, `request` | `browse_route_set`, `browse_route_list`, `browse_route_clear`, `browse_network_requests` (4 tools) | Network tools, split by action |
| `sessions`, `close` | **NOT exposed as tools** — managed via hooks | Daemon lifecycle is transparent to the LLM |

### 3. Table Stakes — The 7 Tools Every Browser Extension Must Have

- **finding:** Across pi-browser (50 tools), `@false00/pi-steel`, and MCP browser tools
  (Playwright MCP, Puppeteer MCP), a core set of 7 capabilities appears in every
  implementation. These are the minimum for an agent to navigate, observe, interact,
  extract, and wait — the browser automation primitives.

| # | Tool | rod-cli command | Why table stakes |
|---|------|----------------|------------------|
| 1 | `browse_goto` | `goto <url>` | Navigation is the entry point for every workflow |
| 2 | `browse_snapshot` | `snapshot` | The agent's "eyes" — without it, it's blind |
| 3 | `browse_click` | `click <sel>` | The primary interaction primitive |
| 4 | `browse_type` | `fill <sel> <text>` | Form input — the other half of interaction |
| 5 | `browse_eval` | `eval <js>` | Escape hatch for anything the DSL doesn't cover |
| 6 | `browse_screenshot` | `screenshot` | Visual verification for the human-in-the-loop |
| 7 | `browse_wait` | `wait <sel>` | The internet is async; without it, every script races |

- **source:** pi-browser tool catalog (README groups these as Navigation + Observation +
  Interaction + Waiting) · **quality:** HIGH real-world
- **source:** Playwright MCP tool list (browser_navigate, browser_snapshot, browser_click,
  browser_type, browser_evaluate, browser_take_screenshot, browser_wait_for) ·
  **quality:** HIGH reference
- **bears on:** The MVP scope. These 7 are what the PROJECT.md already lists. Ship these
  first.

### 4. Differentiators — What rod-cli Brings That pi-browser Doesn't

rod-cli's architectural differences create genuine differentiators in Pi tool design:

| Differentiator | rod-cli capability | Pi tool impact |
|---------------|-------------------|----------------|
| **Zero cold-start** | Daemon keeps browser alive; first command starts it, subsequent commands reuse it | No `browse_connect` or `/browser launch` command needed. The `session_start` hook verifies the binary; the first `browse_goto` implicitly starts the daemon. The LLM never manages browser lifecycle. |
| **Named sessions** | `-s <name>` flag on every command | Every tool accepts `session: Type.Optional(Type.String())`. An agent can run two independent browsers (`-s=admin`, `-s=guest`) simultaneously. pi-browser has no equivalent — it's single-session. |
| **Stealth by default** | godoll fingerprinting, no `navigator.webdriver`, humanized input | pi-browser is raw Playwright — detectable as automation. rod-cli tools don't need "stealth mode" flags; it's always on. |
| **Token-efficient output** | Markdown snapshots, `--raw` passthrough, no DOM dump | `browse_snapshot` returns accessibility-tree markdown by default — ~10–50KB instead of pi-browser's full ARIA tree. The LLM gets more signal per token. |
| **CDP proxy transparency** | `cdp-traffic` command, `--no-cdp-proxy` bypass | A `browse_cdp_traffic` tool could expose the raw CDP log for debugging — no other browser Pi extension offers this. |
| **Profile switching** | `--profile=<name>`, 6 embedded Chrome profiles | A `browse_profile` tool could let agents switch device profiles mid-session. pi-browser requires manual browser restart with different args. |
| **Plugin system** | `plugin load/list/run` | `browse_plugin_load`, `browse_plugin_list`, `browse_plugin_run` — extensibility from within the agent session. Unique to rod-cli. |
| **Single Go binary** | Zero Node.js, zero Python, zero npm install scripts | The rod-cli install is `go install` + `rod-cli install` (Chromium). pi-browser requires `npx playwright install` and a running browser with `--remote-debugging-port`. |

- **source:** rod-cli SKILL.md (command catalog, daemon architecture, named sessions,
  profiles, plugins) · **quality:** HIGH primary
- **source:** pi-browser README (requires CDP connection, manual browser launch,
  `--remote-debugging-port`) · **quality:** HIGH real-world
- **source:** .planning/PROJECT.md (stealth stack, CDP proxy, godoll integration) ·
  **quality:** HIGH project
- **bears on:** Which tools to prioritize beyond table stakes. Named sessions + stealth
  are the headline differentiators; the specific tools that expose them (`session` param
  on every tool, `browse_profile`, `browse_cdp_traffic`) should ship in wave 2.

### 5. Full Tool Catalog — 3-Wave Delivery Plan

#### Wave 1 — Table Stakes (MVP, 7 tools)

```typescript
// Every tool shares this base parameter
const SessionParam = Type.Optional(Type.String({
  description: "Named session identifier for multi-target workflows (e.g. 'admin', 'guest')"
}));

// 1. browse_goto
Type.Object({
  url: Type.String({ description: "URL to navigate to" }),
  session: SessionParam,
  timeout: Type.Optional(Type.Number({
    description: "Navigation timeout in milliseconds",
    default: 30000
  })),
})

// 2. browse_snapshot
Type.Object({
  selector: Type.Optional(Type.String({
    description: "CSS selector to scope the snapshot to a specific element"
  })),
  session: SessionParam,
})

// 3. browse_click
Type.Object({
  selector: Type.String({ description: "CSS selector of the element to click" }),
  doubleClick: Type.Optional(Type.Boolean({
    description: "Perform a double click instead of single click",
    default: false
  })),
  session: SessionParam,
})

// 4. browse_type
Type.Object({
  selector: Type.String({ description: "CSS selector of the input element" }),
  text: Type.String({ description: "Text to type into the element" }),
  submit: Type.Optional(Type.Boolean({
    description: "Press Enter after typing",
    default: false
  })),
  slowly: Type.Optional(Type.Boolean({
    description: "Type one character at a time (humanized)",
    default: true
  })),
  session: SessionParam,
})

// 5. browse_eval
Type.Object({
  script: Type.String({ description: "JavaScript code to evaluate in the page context" }),
  session: SessionParam,
})

// 6. browse_screenshot
Type.Object({
  name: Type.Optional(Type.String({ description: "Filename for the screenshot (PNG)" })),
  fullPage: Type.Optional(Type.Boolean({
    description: "Capture the full scrollable page",
    default: false
  })),
  selector: Type.Optional(Type.String({
    description: "CSS selector to screenshot a specific element"
  })),
  session: SessionParam,
})

// 7. browse_wait
Type.Object({
  selector: Type.Optional(Type.String({
    description: "CSS selector to wait for"
  })),
  text: Type.Optional(Type.String({
    description: "Text to wait for on the page"
  })),
  time: Type.Optional(Type.Number({
    description: "Time to wait in milliseconds"
  })),
  session: SessionParam,
})
```

**All 7 tools map 1:1 to rod-cli CLI commands** and use
`pi.exec("rod-cli", ["--raw", "<cmd>", ...])` as their execution primitive. Output is
always `--raw` for machine parsing.

- **source:** PROJECT.md v2.2 target features · **quality:** HIGH project
- **source:** pi-browser tool catalog (same 7 categories) · **quality:** HIGH real-world
- **bears on:** The exact MVP scope.

#### Wave 2 — Extended Interaction + Lifecycle (8 tools)

| # | Tool | rod-cli | Rationale |
|---|------|---------|-----------|
| 8 | `browse_hover` | `hover <sel>` | Needed for hover-triggered menus/tooltips |
| 9 | `browse_select_option` | `select <sel> <values>` | Dropdown interaction |
| 10 | `browse_press_key` | `press <key>` | Keyboard shortcuts, Enter, Escape |
| 11 | `browse_scroll` | `scroll` | Scroll-driven lazy loading |
| 12 | `browse_file_upload` | `upload <sel> <file>` | File input interaction |
| 13 | `browse_drag` | `drag <start> <end>` | Drag-and-drop workflows |
| 14 | `browse_tabs` | `tab-list/new/select/close` | Multi-tab workflows (action StringEnum: `list|new|select|close`) |
| 15 | `browse_resize` | `resize <w> <h>` | Viewport size changes |

#### Wave 3 — Storage, Network + Advanced (7 tools)

| # | Tool | rod-cli | Rationale |
|---|------|---------|-----------|
| 16–19 | `browse_cookie_get/set/delete/clear` | `cookie-*` | 4 tools following pi-browser pattern |
| 20–23 | `browse_localstorage_get/set/delete/clear` | `localstorage-*` | 4 tools following pi-browser pattern |
| 24–27 | `browse_sessionstorage_get/set/delete/clear` | `sessionstorage-*` | 4 tools following pi-browser pattern |
| 28 | `browse_handle_dialog` | `dialog-accept/dismiss` | Alert/confirm/prompt handling |
| 29 | `browse_console` | `console` | Debug console messages |
| 30 | `browse_navigate` | `reload/go-back/go-forward` | Navigation history (action StringEnum) |
| 31 | `browse_profile` | `--profile=<name>` | **Differentiator:** switch device profiles mid-session |
| 32 | `browse_cdp_traffic` | `cdp-traffic` | **Differentiator:** expose CDP proxy log for debugging |

> **Note on wave 3 counts:** Storage tools can follow pi-browser's 4-tools-per-storage
> pattern (12 tools: cookies 4 + localStorage 4 + sessionStorage 4) or be consolidated
> into `browse_cookie_*` (4 tools) + `browse_storage_*` (4 tools with a `storageType`
> param). The consolidated approach saves 4 tools at the cost of parameter complexity.
> Recommendation: follow pi-browser's explicit pattern — LLMs handle explicit tools
> better than mode-switching params, and storage tools are only registered when needed.

**Why storage tools are wave 3, not wave 1:** pi-browser devotes 18 tools to cookies +
storage. These are high-count but low-complexity — they're simple CRUD wrappers. They
matter for authenticated workflows (login, session persistence) but aren't needed for
the basic browse→snapshot→click→type loop. Ship them after the core loop works.

**Why `browse_profile` is a differentiator:** No other Pi browser extension lets the
agent switch device profiles mid-session. pi-browser requires a browser restart with
different args. rod-cli's `--profile` flag on any command makes this trivial.

- **source:** rod-cli SKILL.md (full command catalog) · **quality:** HIGH primary
- **source:** pi-browser tool catalog (storage tool patterns) · **quality:** HIGH real-world

### 6. Prompt Guidelines — What Makes the LLM Use These Correctly

- **finding:** Pi's `promptGuidelines` are string bullets appended flat to the system
  prompt's Guidelines section. Each bullet MUST name the tool it refers to — "Use this
  tool when..." is ambiguous because the LLM can't tell which tool "this" refers to.
  pi-browser only provides guidelines for `browser_snapshot` (3 items). The Pi docs show
  guidelines as a `string[]` field on the tool registration.
- **source:** https://pi.dev/docs/latest/extensions ("Each guideline must name the tool
  it refers to") · **quality:** HIGH official
- **source:** pi-browser `src/index.ts` (only `browser_snapshot` has guidelines) ·
  **quality:** HIGH real-world
- **bears on:** Every tool's registration. Guidelines are the difference between an LLM
  that uses tools correctly and one that calls `browse_snapshot` after every click.

#### Recommended promptGuidelines per tool:

**`browse_snapshot`** (most important — LLMs over-call this):
- "Use browse_snapshot to find element selectors before interacting with the page. Call it before browse_click, browse_type, or browse_select_option."
- "Do NOT call browse_snapshot after every action to confirm results — browse_click, browse_type, and browse_fill_form already include a snapshot in their response."
- "If the snapshot is too large, re-call browse_snapshot with a selector scoped to the relevant section of the page."
- "Prefer browse_snapshot over browse_screenshot for understanding page structure — it returns structured text, not an image."

**`browse_click`:**
- "Use browse_click to interact with buttons, links, and other clickable elements identified by a CSS selector from browse_snapshot."
- "Set doubleClick to true when the target requires a double-click (e.g., to activate inline editing)."

**`browse_type`:**
- "Use browse_type to enter text into form fields, search boxes, and textareas. Provide the CSS selector from browse_snapshot and the text to type."
- "Set submit to true to press Enter after typing (e.g., for search boxes that lack a submit button)."
- "Set slowly to false only when filling large blocks of text — the default humanized typing is preferred for realism."

**`browse_eval`:**
- "Use browse_eval as an escape hatch when no other tool can accomplish the task. Prefer browse_click, browse_type, or browse_snapshot for standard interactions."
- "Use browse_eval to extract structured data from the page (e.g., document.title, JSON from a script tag) that browse_snapshot cannot capture."
- "Keep scripts short — browse_eval returns stdout; complex logic belongs in your reasoning, not in the page context."

**`browse_screenshot`:**
- "Use browse_screenshot for visual verification — when you need to see what the page looks like, not what it contains."
- "Prefer browse_snapshot for understanding page structure and finding elements. browse_screenshot is for visual confirmation."
- "Set fullPage to true to capture content below the fold (e.g., long articles, full-page designs)."

**`browse_wait`:**
- "Use browse_wait after navigation or after an action that triggers a page update (form submission, lazy load, SPA route change)."
- "Prefer waiting for a specific selector or text over a fixed time — it's more reliable and faster."
- "If the page uses heavy JavaScript rendering, call browse_wait before browse_snapshot to let the DOM settle."

**When to add guidelines vs leave them out:** The Pi docs say guidelines are optional.
Only add them when the LLM is likely to misuse the tool without explicit instruction.
`browse_goto` doesn't need them (navigation is obvious); `browse_snapshot` needs them
most (LLMs tend to over-call it, burning tokens). Rule of thumb: if a tool has a
"prefer X over Y" or "don't call this after Z" rule, encode it in guidelines.

- **source:** pi-browser snapshot guidelines (the only tool with guidelines — and it's
  the one LLMs misuse most) · **quality:** HIGH real-world

### 7. Tools NOT Exposed — What Stays Behind the Curtain

Several rod-cli commands should NOT be exposed as Pi tools. They're lifecycle concerns
managed transparently by hooks, or they're user-setup commands irrelevant to the LLM:

| rod-cli command | Why NOT a Pi tool |
|----------------|-------------------|
| `install` | One-time setup; documented in README, verified by `session_start` hook |
| `sessions` | Daemon introspection; the LLM doesn't manage daemons |
| `close` | Teardown; handled by `session_end` hook |
| `show --annotate` | Human-in-the-loop interactive mode; requires a real human viewing the browser |
| `highlight`, `highlight-clear` | Debugging aids for human developers, not LLM agents |
| `state-save`, `state-load` | Covered by `browse_cookie_*` + `browse_storage_*`; full state serialization is fragile |
| `mousemove`, `mousedown`, `mouseup`, `mousewheel` | Low-level mouse primitives; LLMs work at the element level (`browse_click`, `browse_hover`, `browse_scroll`) |
| `plugin load/list/run` | Plugin management is a power-user feature; expose only if demand exists |

- **source:** rod-cli SKILL.md (full command catalog) · **quality:** HIGH primary
- **source:** pi-browser tool design (exposes low-level mouse tools but annotates them
  as advanced) · **quality:** HIGH real-world
- **bears on:** What NOT to build. Keeping the tool surface small (~18–22 tools vs
  pi-browser's 50) is a feature, not a gap — less prompt bloat, faster tool selection.

### 8. Execution Pattern — Every Tool Follows the Same Shape

- **finding:** Every rod-cli Pi tool follows an identical execution pattern: (1) build
  CLI args from validated params, (2) call `pi.exec("rod-cli", ["--raw", cmd, ...args],
  { signal, timeout })`, (3) parse stdout, (4) return `{ content: [{ type: "text", text }],
  details }`. This uniformity means tools can share a `buildArgs()` helper and a
  `wrapExec()` error-handler, keeping each tool file ~20 lines.
- **source:** .planning/research/STACK.md (pi.exec API, execRodCli wrapper pattern) ·
  **quality:** HIGH project
- **bears on:** Implementation cost. The marginal cost of adding tool #8 is minutes,
  not hours.

#### Canonical tool shape:

```typescript
// src/tools/goto.ts
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli.js";

const GotoParams = Type.Object({
  url: Type.String({ description: "URL to navigate to" }),
  session: Type.Optional(Type.String({ description: "Named session identifier" })),
  timeout: Type.Optional(Type.Number({ description: "Navigation timeout in ms", default: 30000 })),
});

export function registerBrowseGoto(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_goto",
    label: "Browse Goto",
    description: "Navigate the browser to a URL. The first call starts the background browser daemon automatically.",
    promptSnippet: "Navigate to a URL: browse_goto",
    parameters: GotoParams,
    async execute(_toolCallId, params, signal) {
      const args = ["goto", params.url];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(pi, args, {
        signal,
        timeout: params.timeout ?? 30_000,
      });
      return {
        content: [{ type: "text", text: result.stdout }],
        details: { url: params.url, session: params.session },
      };
    },
  });
}
```

### 9. Lifecycle Hooks — Daemon Management

- **finding:** Two hooks manage the rod-cli daemon transparently. `session_start`
  verifies the binary is on PATH and functional. `session_end` tears down the daemon
  (`rod-cli close`). The LLM never sees or manages the browser lifecycle — it just
  calls `browse_goto` and the daemon is already there.
- **source:** .planning/research/STACK.md (lifecycle.ts pattern) · **quality:** HIGH project
- **source:** pi-browser `session_shutdown` hook (calls `session.disconnect()`) ·
  **quality:** HIGH real-world
- **bears on:** The tools that DON'T need to be built. No `browse_connect`, no
  `browse_launch`, no `browse_close` — the hooks handle everything.

```typescript
// src/lifecycle.ts
export function registerLifecycle(pi: ExtensionAPI, rodCliPath: string | null) {
  pi.on("session_start", async (_event, ctx) => {
    if (!rodCliPath) {
      ctx.ui.notify(
        "rod-cli not found. Install: go install github.com/agenthands/rod-cli@latest",
        "warn"
      );
      return;
    }
    const result = await pi.exec("rod-cli", ["--version"], { timeout: 5000 });
    if (result.code !== 0) {
      ctx.ui.notify(`rod-cli failed: ${result.stderr}`, "error");
      return;
    }
    ctx.ui.notify(`rod-cli ${result.stdout.trim()} ready`, "info");
  });

  pi.on("session_end", async () => {
    try { await pi.exec("rod-cli", ["close"], { timeout: 5000 }); } catch {}
  });
}
```

---

## Tradeoffs / Alternatives

### Flat `browse_` prefix vs bare verbs (`goto`, `snapshot`, `click`)
- **Flat prefix (chosen):** Every tool is `browse_<verb>`. Pros: no collision with
  built-in tools, clear namespace, matches ecosystem convention (pi-browser uses
  `browser_`, pi-steel uses `steel_`). Cons: more characters in the prompt.
- **Bare verbs:** Tools named `goto`, `snapshot`, `click`. Pros: shorter, reads like
  natural language. Cons: could collide with future built-in tools, harder to
  distinguish from non-browser tools, breaks ecosystem convention.
  **Recommendation:** use `browse_` prefix — it's the ecosystem standard and avoids
  ambiguity.

### `browse_` vs `browser_` prefix
- **`browse_` (chosen):** rod-cli's own identity; shorter by 2 chars. Already in the
  PROJECT.md.
- **`browser_`:** Matches pi-browser exactly. Simpler for users who know pi-browser.
  **Recommendation:** keep `browse_` — it's rod-cli's brand and already committed in
  the milestone plan.

### Explicit storage tools vs consolidated `browse_storage` with `storageType` param
- **Explicit (pi-browser pattern, recommended for wave 3):** `browse_cookie_get`,
  `browse_localstorage_get`, `browse_sessionstorage_get` — 3× the tools but
  unambiguous. Pros: LLM-verifiable, matches ecosystem. Cons: prompt bloat.
- **Consolidated:** `browse_storage_get` with `storageType: "cookie"|"local"|"session"`.
  Pros: fewer tools. Cons: parameter complexity, harder for LLM to discover.
  **Recommendation:** follow pi-browser's explicit pattern. Storage tools are wave 3;
  by then the pattern is proven.

### `promptSnippet` on every tool vs only key tools
- **Every tool (recommended):** The `promptSnippet` is a one-line entry in "Available
  tools." Omitting it hides the tool from that list (though it remains callable).
  pi-browser omits it entirely. **Recommendation:** include a short `promptSnippet` on
  every tool — it improves discoverability at negligible token cost.

---

## Unknowns

- **`browse_snapshot` output size in practice:** rod-cli's markdown snapshots are
  token-efficient by design, but Pi enforces 50KB/2000-line truncation. A complex page
  (e.g., a large e-commerce listing) could exceed this. **How to settle:** test with
  real-world pages (amazon.com, github.com, news.ycombinator.com) and measure snapshot
  sizes. If truncation is frequent, add a `maxLength` parameter to `browse_snapshot`.
- **LLM behaviour with `session` parameter:** The `session` param is optional on every
  tool. Will LLMs correctly use it for multi-target workflows, or will they ignore it
  and use a single session? **How to settle:** test with a multi-session prompt ("log
  into site A as admin and site B as guest simultaneously") and observe whether the LLM
  sets `session: "admin"` and `session: "guest"`.
- **Pi's tool count impact on prompt size:** 22 tools × (~80 chars name + ~200 chars
  description + ~300 chars params + ~150 chars guidelines each) ≈ 16KB of system prompt.
  pi-browser's 50 tools would be ~35KB. **How to settle:** measure the actual prompt
  size with all tools registered and verify it fits within Pi's context window.
- **`pi.exec` timeout behaviour with long-running pages:** rod-cli's `goto` can take
  30s+ on slow pages. Pi's `pi.exec` timeout behaviour (SIGTERM vs SIGKILL, child
  cleanup) isn't specified. **How to settle:** the STACK.md already notes this — test
  with `pi.exec("sleep", ["60"])` and a short AbortSignal.

---

## Sources

- pi-browser source: https://github.com/larsderidder/pi-browser (50-tool browser
  extension, `browser_` prefix convention)
- pi-browser README: https://github.com/larsderidder/pi-browser (tool catalog, groups,
  CDP dependency)
- Pi extensions docs: https://pi.dev/docs/latest/extensions (registerTool API,
  promptGuidelines rules, pi.exec, tool design)
- Pi packages docs: https://pi.dev/docs/latest/packages (package.json conventions,
  `pi install`)
- @shog-lab/pi-toolkit: https://www.npmjs.com/package/@shog-lab/pi-toolkit (snake_case
  naming rule, MCP bridge pattern)
- earendil-works/pi source: https://github.com/earendil-works/pi/blob/main/packages/
  coding-agent/docs/extensions.md (API reference)
- rod-cli SKILL.md: `skills/rod-cli/SKILL.md` (full command catalog, daemon architecture)
- .planning/PROJECT.md (v2.2 milestone target features, godoll stealth stack)
- .planning/research/STACK.md (Pi extension stack, pi.exec, lifecycle pattern, TypeBox
  schemas)

---
*Feature research for: Pi Extension tool catalog, naming, and design (rod-cli v2.2)*
*Researched: 2026-06-27*
