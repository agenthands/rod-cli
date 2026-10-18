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
