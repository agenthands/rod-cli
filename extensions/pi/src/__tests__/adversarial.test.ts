/**
 * Adversarial test suite — adapter-side coverage for the Pi extension.
 *
 * Tests:
 *   1. registerLifecycle — hook registration and behavior
 *   2. Tool registration — all 13 tools registered with correct names
 *   3. Per-tool execute() → argv via mock execFn
 *   4. index.ts default export — wiring
 */

import { describe, it, expect, vi } from "vitest";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

interface RegisteredTool {
  name: string;
  execute: (
    id: string,
    params: Record<string, unknown>,
    signal?: AbortSignal,
  ) => Promise<{
    content: Array<{ type: string; text: string }>;
    details: unknown;
  }>;
}

interface MockPi {
  exec: ReturnType<typeof vi.fn>;
  ui: { notify: ReturnType<typeof vi.fn> };
  on: ReturnType<typeof vi.fn>;
  registerTool: ReturnType<typeof vi.fn>;
  _fire: (event: string, ...args: unknown[]) => Promise<void>;
  _tools: RegisteredTool[];
}

function mockPi(overrides?: Record<string, unknown>): MockPi {
  const exec = vi.fn().mockResolvedValue({ code: 0, stdout: "", stderr: "" });
  const notify = vi.fn();
  const onHandlers: Record<string, Array<(...args: unknown[]) => unknown>> = {};
  const registeredTools: RegisteredTool[] = [];

  return {
    exec,
    ui: { notify },
    on: vi.fn((event: string, handler: (...args: unknown[]) => unknown) => {
      (onHandlers[event] ??= []).push(handler);
    }),
    registerTool: vi.fn((tool: RegisteredTool) => {
      registeredTools.push(tool);
    }),
    _fire: async (event: string, ...args: unknown[]) => {
      for (const h of onHandlers[event] ?? []) {
        await h(...args);
      }
    },
    _tools: registeredTools,
    ...overrides,
  } satisfies MockPi;
}

/** Cast a MockPi to ExtensionAPI for passing to extension functions. */
function asExt(pi: MockPi): ExtensionAPI {
  return pi as unknown as ExtensionAPI;
}

// ---------------------------------------------------------------------------
// 1. registerLifecycle — hook registration
// ---------------------------------------------------------------------------

describe("registerLifecycle — hook registration", () => {
  it("registers session_start and session_shutdown handlers", async () => {
    const { registerLifecycle } = await import("../lifecycle");
    const pi = mockPi();

    registerLifecycle(asExt(pi), "/usr/bin/rod-cli");
    expect(pi.on).toHaveBeenCalledWith("session_start", expect.any(Function));
    expect(pi.on).toHaveBeenCalledWith("session_shutdown", expect.any(Function));
  });

  describe("session_start handler", () => {
    it("notifies warning when rodCliPath is null", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(asExt(pi), null);
      await pi._fire("session_start", {}, { ui: pi.ui });

      expect(pi.ui.notify).toHaveBeenCalledWith(
        expect.stringContaining("not found"),
        "warning",
      );
      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("runs --version and notifies info on success", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();
      pi.exec.mockResolvedValueOnce({ code: 0, stdout: "rod-cli v1.0.0\n", stderr: "" });

      registerLifecycle(asExt(pi), "/usr/bin/rod-cli");
      await pi._fire("session_start", {}, { ui: pi.ui });

      expect(pi.exec).toHaveBeenCalledWith("rod-cli", ["--version"], { timeout: 5000 });
      expect(pi.ui.notify).toHaveBeenCalledWith(
        "rod-cli rod-cli v1.0.0 ready",
        "info",
      );
    });

    it("notifies warning when --version fails", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();
      pi.exec.mockRejectedValueOnce(new Error("exec failed"));

      registerLifecycle(asExt(pi), "/usr/bin/rod-cli");
      await pi._fire("session_start", {}, { ui: pi.ui });

      expect(pi.ui.notify).toHaveBeenCalledWith(
        expect.stringContaining("--version failed"),
        "warning",
      );
    });
  });

  describe("session_shutdown handler", () => {
    it("runs close on reason quit", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(asExt(pi), "/usr/bin/rod-cli");
      await pi._fire("session_shutdown", { reason: "quit" });

      expect(pi.exec).toHaveBeenCalledWith("rod-cli", ["close"], { timeout: 5000 });
    });

    it("does NOT run close on reason reload", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(asExt(pi), "/usr/bin/rod-cli");
      await pi._fire("session_shutdown", { reason: "reload" });

      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("does NOT run close on reason fork", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(asExt(pi), "/usr/bin/rod-cli");
      await pi._fire("session_shutdown", { reason: "fork" });

      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("does NOT run close on reason resume", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(asExt(pi), "/usr/bin/rod-cli");
      await pi._fire("session_shutdown", { reason: "resume" });

      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("swallows errors from close (best-effort)", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();
      pi.exec.mockRejectedValueOnce(new Error("daemon already dead"));

      registerLifecycle(asExt(pi), "/usr/bin/rod-cli");
      await pi._fire("session_shutdown", { reason: "quit" });
    });
  });
});

// ---------------------------------------------------------------------------
// 2. Tool registration — all 13 tools registered with correct names
// ---------------------------------------------------------------------------

describe("tool registration — all 13 tools present", () => {
  it("registers all 13 browse_* tools", async () => {
    const { registerAllCoreTools, registerAllExtendedTools } = await import("../tools");
    const pi = mockPi();
    const execFn = vi.fn().mockResolvedValue({ code: 0, stdout: "", stderr: "" });

    registerAllCoreTools(asExt(pi), execFn);
    registerAllExtendedTools(asExt(pi), execFn);

    const names = pi._tools.map((t: RegisteredTool) => t.name).sort();
    expect(names).toEqual([
      "browse_click",
      "browse_cookies",
      "browse_eval",
      "browse_fill_form",
      "browse_goto",
      "browse_navigate",
      "browse_screenshot",
      "browse_scroll",
      "browse_snapshot",
      "browse_storage",
      "browse_tabs",
      "browse_type",
      "browse_wait",
    ]);
  });
});

// ---------------------------------------------------------------------------
// 3. Per-tool execute() → argv via mock execFn
// ---------------------------------------------------------------------------

describe("per-tool execute() produces correct argv", () => {
  async function getTool(name: string) {
    const { registerAllCoreTools, registerAllExtendedTools } = await import("../tools");
    const pi = mockPi();
    const execFn = vi.fn().mockResolvedValue({ code: 0, stdout: "", stderr: "" });

    registerAllCoreTools(asExt(pi), execFn);
    registerAllExtendedTools(asExt(pi), execFn);

    const tool = pi._tools.find((t: RegisteredTool) => t.name === name);
    expect(tool).toBeDefined();
    return { tool: tool!, execFn };
  }

  it("browse_goto execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_goto");
    await tool.execute("id", { url: "https://example.com" }, undefined);
    // execRodCli prepends --raw, so execFn receives ["--raw", "goto", "--", "https://example.com"]
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "goto", "--", "https://example.com"],
      expect.anything(),
    );
  });

  it("browse_snapshot execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_snapshot");
    await tool.execute("id", {}, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "snapshot"],
      expect.anything(),
    );
  });

  it("browse_click execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_click");
    await tool.execute("id", { selector: "#btn" }, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "click", "--", "#btn"],
      expect.anything(),
    );
  });

  it("browse_type execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_type");
    await tool.execute("id", { selector: "#input", text: "hello" }, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "type", "--", "#input", "hello"],
      expect.anything(),
    );
  });

  it("browse_eval execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_eval");
    await tool.execute("id", { expression: "1+1" }, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "eval", "--", "1+1"],
      expect.anything(),
    );
  });

  it("browse_fill_form execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_fill_form");
    await tool.execute("id", { selector: "#name", text: "John" }, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "fill", "--", "#name", "John"],
      expect.anything(),
    );
  });

  it("browse_scroll execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_scroll");
    await tool.execute("id", {}, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "mousewheel", "--", "0", "300"],
      expect.anything(),
    );
  });

  it("browse_tabs execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_tabs");
    await tool.execute("id", { action: "list" }, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "tab-list"],
      expect.anything(),
    );
  });

  it("browse_navigate execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_navigate");
    await tool.execute("id", { action: "reload" }, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "reload"],
      expect.anything(),
    );
  });

  it("browse_cookies execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_cookies");
    await tool.execute("id", { action: "get" }, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "cookie-get"],
      expect.anything(),
    );
  });

  it("browse_storage execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_storage");
    await tool.execute("id", { action: "clear" }, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "localstorage-clear"],
      expect.anything(),
    );
  });

  it("browse_screenshot execute passes correct argv", async () => {
    const { tool, execFn } = await getTool("browse_screenshot");
    await tool.execute("id", {}, undefined);
    expect(execFn).toHaveBeenCalledWith(
      ["--raw", "screenshot"],
      expect.anything(),
    );
  });
});

// ---------------------------------------------------------------------------
// 4. index.ts — wiring
// ---------------------------------------------------------------------------

describe("index.ts — default export (wiring)", () => {
  it("is a function", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });

  it("does not throw when called with a mock pi", async () => {
    const mod = await import("../index");
    const pi = mockPi();

    expect(() => mod.default(asExt(pi))).not.toThrow();
  });

  it("registers lifecycle hooks and all 13 tools", async () => {
    const mod = await import("../index");
    const pi = mockPi();

    mod.default(asExt(pi));

    expect(pi.on).toHaveBeenCalledWith("session_start", expect.any(Function));
    expect(pi.on).toHaveBeenCalledWith("session_shutdown", expect.any(Function));
    expect(pi.registerTool).toHaveBeenCalledTimes(13);
  });
});
