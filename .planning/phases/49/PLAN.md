---
author: architect
responsible: architect
phase: 49
status: ready
parent_artifacts:
  - .planning/phases/49/CONTEXT.md
  - .planning/REQUIREMENTS.md
---

# Phase 49: Extended Tools — PLAN

## Shared pattern (all tools)

Every tool:
1. Imports `execRodCli` from `"../cli"`, `SessionParam` from `"../types"`, `StringEnum` from `"@earendil-works/pi-ai"` (if enum params)
2. Calls `pi.registerTool({ name: "browse_*", ... })`
3. In `execute`: builds args array, calls `execRodCli(args, { signal })`, returns `{ content: [{ type: "text", text: result.stdout }] }`
4. `--` inserted before all user-controlled positional args (I6 — argv injection fix)
5. Session param forwarded via `-s <name>` if set
6. `promptGuidelines` name the tool explicitly

## §1 — browse_tabs (TOOLS-08)

Write `extensions/pi/src/tools/tabs.ts`:

- **params:** `action: StringEnum(["list", "new", "close", "select"] as const)`, `url` (optional, for new), `index` (optional, for close/select), `session`
- **list:** `["tab-list"]`
- **new:** `["tab-new", "--", url]`
- **close:** `["tab-close", String(index)]`
- **select:** `["tab-select", String(index)]`
- If optional param missing when required (new needs url, close/select need index): throw `Error("action 'new' requires url")`

## §2 — browse_navigate (TOOLS-09)

Write `extensions/pi/src/tools/navigate.ts`:

- **params:** `action: StringEnum(["reload", "back", "forward"] as const)`, `session`
- **reload:** `["reload"]`
- **back:** `["go-back"]`
- **forward:** `["go-forward"]`

## §3 — browse_scroll (TOOLS-10)

Write `extensions/pi/src/tools/scroll.ts`:

- **params:** `direction: StringEnum(["down", "up"] as const)` (default `"down"`), `distance` (optional number, default 300), `selector` (optional — scroll specific element), `session`
- Maps to `mousewheel <dx> <dy>`: dx=0, dy=`distance` (down) or `-distance` (up)
- `["mousewheel", "0", String(dy)]`
- If `selector`: throw with message pointing to `browse_eval` for element-level scrolling (mousewheel is viewport-level)

## §4 — browse_cookies (TOOLS-11)

Write `extensions/pi/src/tools/cookies.ts`:

- **params:** `action: StringEnum(["get", "set", "delete", "clear"] as const)`, `name` (optional), `value` (optional, for set), `url` (optional, for set), `session`
- **get:** `["cookie-get"]` (returns all cookies)
- **set:** `["cookie-set", "--", name, value]` — throw if name or value missing
- **delete:** `["cookie-delete", "--", name]` — throw if name missing
- **clear:** `["cookie-clear"]`

## §5 — browse_storage (TOOLS-12)

Write `extensions/pi/src/tools/storage.ts`:

- **params:** `action: StringEnum(["get", "set", "delete", "clear"] as const)`, `storageType: StringEnum(["local", "session"] as const)` (default `"local"`), `key` (optional), `value` (optional, for set), `session`
- Build command prefix from storageType: `"local"` → `"localstorage-"`, `"session"` → `"sessionstorage-"`
- **get:** `[prefix + "get"]` or `[prefix + "get", "--", key]` if key specified
- **set:** `[prefix + "set", "--", key, value]` — throw if key or value missing
- **delete:** `[prefix + "delete", "--", key]` — throw if key missing
- **clear:** `[prefix + "clear"]`

## §6 — browse_fill_form (TOOLS-13)

Write `extensions/pi/src/tools/fill_form.ts`:

- **params:** `selector: Type.String`, `text: Type.String`, `submit: Type.Optional(Type.Boolean)` (default false), `session`
- Maps 1:1 to `rod-cli fill <sel> <text> [--submit]`
- `["fill", "--", params.selector, params.text]` + conditional `"--submit"` if submit
- Different from `browse_type`: fill is instant (no humanized typing), fill supports `--submit`

## §7 — Barrel update + argv injection fix

Update `extensions/pi/src/tools/index.ts`:
```typescript
import { registerBrowseTabs } from "./tabs";
import { registerBrowseNavigate } from "./navigate";
import { registerBrowseScroll } from "./scroll";
import { registerBrowseCookies } from "./cookies";
import { registerBrowseStorage } from "./storage";
import { registerBrowseFillForm } from "./fill_form";

export function registerAllExtendedTools(pi: ExtensionAPI) {
  registerBrowseTabs(pi);
  registerBrowseNavigate(pi);
  registerBrowseScroll(pi);
  registerBrowseCookies(pi);
  registerBrowseStorage(pi);
  registerBrowseFillForm(pi);
}
```

Update `extensions/pi/src/index.ts`: call `registerAllExtendedTools(pi)` after `registerAllCoreTools(pi)`.

### I6 fix (argv injection): Insert `--` in Phase 48 tools

In each Phase 48 tool's `execute()`, insert `"--"` before the first user-controlled positional argument:

| Tool | Before | After |
|------|--------|-------|
| goto.ts | `["goto", params.url]` | `["goto", "--", params.url]` |
| click.ts | `[cmd, params.selector]` | `[cmd, "--", params.selector]` |
| type.ts | `["type", params.selector, params.text]` | `["type", "--", params.selector, params.text]` |
| eval.ts | `["eval", params.expression]` | `["eval", "--", params.expression]` |
| wait.ts | eval args | insert `"--"` before expression |

## §8 — Integration test additions

Add extended tool tests to `extensions/pi/src/__tests__/integration.test.ts`:

```typescript
it("tabs list works", async () => {
  const r = await execRodCli(["tab-list"]);
  expect(r.code).toBe(0);
});

it("cookies get works", async () => {
  const r = await execRodCli(["cookie-get"]);
  expect(r.code).toBe(0);
});

it("navigate reload works", async () => {
  const r = await execRodCli(["reload"]);
  expect(r.code).toBe(0);
});

it("fill form works", async () => {
  const r = await execRodCli(["fill", "--", "#name", "Test"]);
  expect(r.code).toBe(0);
});

it("storage works", async () => {
  const r1 = await execRodCli(["localstorage-set", "--", "test_key", "test_val"]);
  expect(r1.code).toBe(0);
  const r2 = await execRodCli(["localstorage-get"]);
  expect(r2.code).toBe(0);
});
```

## §9 — Build verification gate

1. `npx tsc --noEmit --project extensions/pi/tsconfig.json` — no type errors
2. `npx vitest run --config extensions/pi/vitest.config.ts` — all tests pass

## Execution order

1. §1-§6: Write 6 extended tool files
2. §7: Update barrel + index + backfill `--` fix in Phase 48 tools
3. §8: Add integration test cases
4. §9: tsc + vitest gate

## Files changed

| File | § | Status |
|------|---|--------|
| `extensions/pi/src/tools/tabs.ts` | §1 | NEW |
| `extensions/pi/src/tools/navigate.ts` | §2 | NEW |
| `extensions/pi/src/tools/scroll.ts` | §3 | NEW |
| `extensions/pi/src/tools/cookies.ts` | §4 | NEW |
| `extensions/pi/src/tools/storage.ts` | §5 | NEW |
| `extensions/pi/src/tools/fill_form.ts` | §6 | NEW |
| `extensions/pi/src/tools/index.ts` | §7 | MODIFIED |
| `extensions/pi/src/index.ts` | §7 | MODIFIED |
| `extensions/pi/src/tools/goto.ts..wait.ts` | §7 | MODIFIED (-- fix) |
| `extensions/pi/src/__tests__/integration.test.ts` | §8 | MODIFIED |
