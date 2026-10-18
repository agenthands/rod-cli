import { describe, it, expect } from "bun:test";
import { createMockApi } from "./mockapi";
import extension from "../index";

describe("tool registration", () => {
  it("registers all 16 browse_* tools", () => {
    const mock = createMockApi();
    extension(mock.api);

    const names = mock.tools.map((t) => t.name).sort();
    expect(names).toEqual([
      "browse_click",
      "browse_cookies",
      "browse_eval",
      "browse_fill_form",
      "browse_goto",
      "browse_hover",
      "browse_navigate",
      "browse_press",
      "browse_screenshot",
      "browse_scroll",
      "browse_select",
      "browse_snapshot",
      "browse_storage",
      "browse_tabs",
      "browse_type",
      "browse_wait",
    ]);
  });
});

describe("TestAllToolSchemasStrict", () => {
  const validArgs: Record<string, Record<string, unknown>> = {
    browse_click: { selector: "#b" },
    browse_cookies: { action: "get" },
    browse_eval: { expression: "1" },
    browse_fill_form: { selector: "#n", text: "x" },
    browse_goto: { url: "https://example.com" },
    browse_hover: { selector: "#m" },
    browse_navigate: { action: "reload" },
    browse_press: { key: "Enter" },
    browse_screenshot: {},
    browse_scroll: {},
    browse_select: { selector: "#c", values: ["a"] },
    browse_snapshot: {},
    browse_storage: { action: "clear" },
    browse_tabs: { action: "list" },
    browse_type: { selector: "#i", text: "x" },
    browse_wait: {},
  };

  it("every tool rejects an unknown key", () => {
    const mock = createMockApi();
    extension(mock.api);

    expect(mock.tools).toHaveLength(16);
    for (const tool of mock.tools as unknown as Array<{
      name: string;
      parameters: { safeParse(v: unknown): { success: boolean } };
    }>) {
      const args = validArgs[tool.name];
      expect(args, tool.name).toBeDefined();
      expect(tool.parameters.safeParse(args).success, `${tool.name} valid`).toBe(true);
      expect(tool.parameters.safeParse({ ...args, bogus: 1 }).success, `${tool.name} bogus`).toBe(false);
    }
  });
});
