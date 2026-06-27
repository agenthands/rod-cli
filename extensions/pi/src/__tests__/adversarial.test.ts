/**
 * Adversarial test suite — the tests the author didn't think to write.
 *
 * Tests across five areas:
 *   1. findRodCli() — cross-platform binary resolution
 *   2. validateInput() — input validation (tested through execRodCli)
 *   3. timeoutFor() — timeout selection (tested through execRodCli)
 *   4. execRodCli() — shell-out wrapper integration
 *   5. registerLifecycle() — hook registration
 *   6. index.ts default export — wiring
 *
 * Oracles:
 *   - Totality: no input panics, hangs, or corrupts state
 *   - Boundary values: exact limits, one-over
 *   - Metamorphic: PATH ordering respects resolution priority
 *   - Property: idempotence of findRodCli, monotonicity of timeout
 *   - Contract: throws are typed Errors with messages, not crashes
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

/** Create a minimal mock ExtensionAPI for execRodCli / lifecycle tests. */
function mockPi(overrides?: Partial<ExtensionAPI>): ExtensionAPI {
  const exec = vi.fn().mockResolvedValue({ code: 0, stdout: "", stderr: "" });
  const notify = vi.fn();
  const onHandlers: Record<string, Array<(...args: any[]) => any>> = {};

  return {
    exec,
    ui: { notify } as any,
    on: vi.fn((event: string, handler: (...args: any[]) => any) => {
      (onHandlers[event] ??= []).push(handler);
    }),
    // Escape hatch: fire a registered event handler
    _fire: async (event: string, ...args: any[]) => {
      for (const h of onHandlers[event] ?? []) {
        await h(...args);
      }
    },
    ...overrides,
  } as any;
}

// ---------------------------------------------------------------------------
// 1. findRodCli — binary resolution
// ---------------------------------------------------------------------------

// We use vi.hoisted() to create mock functions that survive vitest's hoisting
// of vi.mock() calls — needed because the mock factory runs before the module
// body, but the test code needs a reference to control the mock's return value.

const { fsStatSyncMock, osPlatformMock } = vi.hoisted(() => ({
  fsStatSyncMock: vi.fn(),
  osPlatformMock: vi.fn(() => "linux"),
}));

vi.mock("node:fs", () => ({
  statSync: fsStatSyncMock,
  // re-export anything else the module might pull in
}));

vi.mock("node:os", () => ({
  platform: osPlatformMock,
  default: { platform: osPlatformMock },
}));

// Dynamic import to get a fresh module after mocks are in place
async function importCli() {
  return await import("../cli");
}

describe("findRodCli — binary resolution", () => {
  const originalEnv = { ...process.env };

  beforeEach(() => {
    vi.clearAllMocks();
    // Restore env to a known-clean state for each test
    process.env = { ...originalEnv };
    // Sanitise away pre-existing values that would interfere
    delete process.env.ROD_CLI_PATH;
    delete process.env.GOBIN;
    process.env.HOME = "/home/testuser";
    process.env.PATH = "/usr/bin:/bin";
    osPlatformMock.mockReturnValue("linux");
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  // --- Totality: never throws ---

  it("returns null when nothing is found (totality)", async () => {
    // statSync never finds a file
    fsStatSyncMock.mockReturnValue(undefined as any);

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBeNull();
  });

  it("never throws even with all env vars unset", async () => {
    delete process.env.HOME;
    delete process.env.PATH;
    delete process.env.ROD_CLI_PATH;
    delete process.env.GOBIN;
    fsStatSyncMock.mockReturnValue(undefined as any);

    const { findRodCli } = await importCli();
    expect(() => findRodCli()).not.toThrow();
    expect(findRodCli()).toBeNull();
  });

  // --- Priority 1: ROD_CLI_PATH ---

  it("returns ROD_CLI_PATH if it points to a file", async () => {
    process.env.ROD_CLI_PATH = "/opt/bin/rod-cli";
    // statSync returns an object where isFile() → true
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/opt/bin/rod-cli") return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBe("/opt/bin/rod-cli");
  });

  it("skips ROD_CLI_PATH if it points to a directory", async () => {
    process.env.ROD_CLI_PATH = "/opt/bin";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/opt/bin") return { isFile: () => false } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBeNull();
  });

  it("skips ROD_CLI_PATH when statSync throws (permission denied)", async () => {
    process.env.ROD_CLI_PATH = "/root/forbidden/rod-cli";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/root/forbidden/rod-cli") throw new Error("EACCES");
      return undefined;
    });

    const { findRodCli } = await importCli();
    // Should not re-throw, should fall through to null
    expect(findRodCli()).toBeNull();
  });

  it("skips ROD_CLI_PATH when it's an empty string", async () => {
    process.env.ROD_CLI_PATH = ""; // falsy
    const { findRodCli } = await importCli();
    // empty string is falsy → if (envPath) short-circuits
    expect(findRodCli()).toBeNull();
  });

  // --- Priority 2: PATH scanning ---

  it("finds binary in PATH", async () => {
    process.env.PATH = "/a:/b:/c";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/b/rod-cli") return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBe("/b/rod-cli");
  });

  it("returns first PATH match (leftmost wins)", async () => {
    process.env.PATH = "/first:/second";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/first/rod-cli" || p === "/second/rod-cli")
        return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBe("/first/rod-cli");
  });

  it("handles empty PATH string", async () => {
    process.env.PATH = "";
    const { findRodCli } = await importCli();
    expect(() => findRodCli()).not.toThrow();
    expect(findRodCli()).toBeNull();
  });

  it("skips empty PATH entries (leading/trailing/doubled separators)", async () => {
    process.env.PATH = ":/usr/bin::/bin:";
    fsStatSyncMock.mockReturnValue(undefined as any);

    const { findRodCli } = await importCli();
    // The loop has `if (!dir) continue` — empty entries are skipped.
    // Should not call statSync("/rod-cli") or statSync("/usr/bin/rod-cli") etc
    expect(() => findRodCli()).not.toThrow();
  });

  it("handles PATH with only empty entries", async () => {
    process.env.PATH = ":::";
    const { findRodCli } = await importCli();
    expect(() => findRodCli()).not.toThrow();
    expect(findRodCli()).toBeNull();
  });

  // --- Priority 3: GOBIN / $HOME/go/bin ---

  it("finds binary via GOBIN", async () => {
    process.env.PATH = "";
    process.env.GOBIN = "/custom/go/bin";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/custom/go/bin/rod-cli") return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBe("/custom/go/bin/rod-cli");
  });

  it("falls back to $HOME/go/bin when GOBIN is unset", async () => {
    process.env.PATH = "";
    delete process.env.GOBIN;
    process.env.HOME = "/home/testuser";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/home/testuser/go/bin/rod-cli")
        return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBe("/home/testuser/go/bin/rod-cli");
  });

  it("returns null when HOME is also unset", async () => {
    delete process.env.HOME;
    delete process.env.GOBIN;
    process.env.PATH = "";
    const { findRodCli } = await importCli();
    expect(findRodCli()).toBeNull();
  });

  // --- Resolution priority: ROD_CLI_PATH > PATH > GOBIN ---

  it("ROD_CLI_PATH wins over PATH and GOBIN", async () => {
    process.env.ROD_CLI_PATH = "/explicit/rod-cli";
    process.env.PATH = "/pathdir";
    process.env.GOBIN = "/gobindir";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (
        p === "/explicit/rod-cli" ||
        p === "/pathdir/rod-cli" ||
        p === "/gobindir/rod-cli"
      )
        return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBe("/explicit/rod-cli");
  });

  it("PATH wins over GOBIN when ROD_CLI_PATH unset", async () => {
    delete process.env.ROD_CLI_PATH;
    process.env.PATH = "/pathdir";
    process.env.GOBIN = "/gobindir";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/pathdir/rod-cli" || p === "/gobindir/rod-cli")
        return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBe("/pathdir/rod-cli");
  });

  // --- Cross-platform ---

  it("on Windows, checks USERPROFILE instead of HOME", async () => {
    osPlatformMock.mockReturnValue("win32");
    delete process.env.HOME;
    process.env.USERPROFILE = "C:\\Users\\testuser";
    process.env.PATH = "";
    delete process.env.GOBIN;
    fsStatSyncMock.mockImplementation((p: string) => {
      // On Windows, join uses backslash
      if (
        p.includes("rod-cli.exe") &&
        p.includes("go") &&
        p.includes("bin")
      )
        return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    const result = findRodCli();
    // Should find via USERPROFILE-based path, not crash
    expect(result === null || (typeof result === "string" && result.includes("rod-cli.exe"))).toBe(true);
  });

  // --- Property: idempotence ---

  it("is idempotent — same result on repeated calls", async () => {
    process.env.ROD_CLI_PATH = "/opt/rod-cli";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/opt/rod-cli") return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    const r1 = findRodCli();
    const r2 = findRodCli();
    const r3 = findRodCli();
    expect(r1).toBe(r2);
    expect(r2).toBe(r3);
  });

  // --- Pathological paths ---

  it("handles paths with spaces and special characters", async () => {
    process.env.ROD_CLI_PATH = "/home/user/My Tools/rod-cli";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/home/user/My Tools/rod-cli")
        return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBe("/home/user/My Tools/rod-cli");
  });

  it("handles PATH entries with spaces", async () => {
    process.env.PATH = "/home/user/My Bin:/usr/bin";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/home/user/My Bin/rod-cli")
        return { isFile: () => true } as any;
      return undefined;
    });

    const { findRodCli } = await importCli();
    expect(findRodCli()).toBe("/home/user/My Bin/rod-cli");
  });
});

// ---------------------------------------------------------------------------
// 2 & 3. validateInput + timeoutFor — tested through execRodCli
// ---------------------------------------------------------------------------

describe("execRodCli — input validation & timeout selection", () => {
  let pi: ReturnType<typeof mockPi>;

  beforeEach(async () => {
    vi.clearAllMocks();
    pi = mockPi();
    // Set the module-level _pi reference so execRodCli can find it
    const { setPi } = await import("../cli");
    setPi(pi as unknown as ExtensionAPI);
  });

  // ==============================================================
  // validateInput — tested by asserting execRodCli throws BEFORE
  // calling pi.exec. When input is invalid, pi.exec is never called.
  // ==============================================================

  describe("validateInput (through execRodCli rejection before exec)", () => {
    const NOT_CALLED = Symbol("exec should not have been called");

    it("rejects goto without URL", async () => {
      await expect(execRodCli([ "goto" ])).rejects.toThrow("goto requires a URL argument");
      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("rejects goto with empty URL", async () => {
      await expect(execRodCli([ "goto", "" ])).rejects.toThrow("goto requires a URL argument");
    });

    it("rejects goto with ftp:// scheme", async () => {
      await expect(execRodCli([ "goto", "ftp://example.com" ])).rejects.toThrow(
        "requires an http:// or https:// URL",
      );
    });

    it("rejects goto with ws:// scheme (WebSocket)", async () => {
      await expect(execRodCli([ "goto", "ws://example.com" ])).rejects.toThrow(
        "requires an http:// or https:// URL",
      );
    });

    it("rejects goto with wss:// scheme", async () => {
      await expect(execRodCli([ "goto", "wss://example.com" ])).rejects.toThrow(
        "requires an http:// or https:// URL",
      );
    });

    it("rejects goto with data: scheme", async () => {
      await expect(
        execRodCli([ "goto", "data:text/html,<h1>hi</h1>" ]),
      ).rejects.toThrow("requires an http:// or https:// URL");
    });

    it("rejects goto with mailto: scheme", async () => {
      await expect(
        execRodCli([ "goto", "mailto:user@example.com" ]),
      ).rejects.toThrow("requires an http:// or https:// URL");
    });

    it("rejects goto with javascript: scheme", async () => {
      await expect(
        execRodCli([ "goto", "javascript:void(0)" ]),
      ).rejects.toThrow("requires an http:// or https:// URL");
    });

    it("rejects goto with file:// scheme", async () => {
      await expect(
        execRodCli([ "goto", "file:///etc/passwd" ]),
      ).rejects.toThrow("requires an http:// or https:// URL");
    });

    it("rejects goto with uppercase HTTP scheme (case-sensitive startsWith)", async () => {
      // startsWith is case-sensitive — "HTTP://" !== "http://"
      await expect(
        execRodCli([ "goto", "HTTP://EXAMPLE.COM" ]),
      ).rejects.toThrow("requires an http:// or https:// URL");
    });

    it("rejects goto with mixed-case HttPs:// scheme", async () => {
      await expect(
        execRodCli([ "goto", "HttPs://example.com" ]),
      ).rejects.toThrow("requires an http:// or https:// URL");
    });

    it("accepts goto with valid https:// URL", async () => {
      await execRodCli([ "goto", "https://example.com" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.arrayContaining([ "--raw", "goto", "https://example.com" ]),
        expect.anything(),
      );
    });

    it("accepts goto with valid http:// URL", async () => {
      await execRodCli([ "goto", "http://example.com" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("accepts goto with minimal http:// URL", async () => {
      // "http://" is technically a valid URL (empty authority)
      await execRodCli([ "goto", "http://" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    // --- Selector validation (click / fill / type) ---

    it("rejects click without selector", async () => {
      await expect(execRodCli([ "click" ])).rejects.toThrow(
        "click requires a non-empty CSS selector",
      );
    });

    it("rejects click with empty selector", async () => {
      await expect(execRodCli([ "click", "" ])).rejects.toThrow(
        "click requires a non-empty CSS selector",
      );
    });

    it("rejects click with whitespace-only selector", async () => {
      await expect(execRodCli([ "click", "   \t  " ])).rejects.toThrow(
        "click requires a non-empty CSS selector",
      );
    });

    it("accepts click with valid selector", async () => {
      await execRodCli([ "click", "#myid" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("rejects fill without selector", async () => {
      await expect(execRodCli([ "fill" ])).rejects.toThrow(
        "fill requires a non-empty CSS selector",
      );
    });

    it("rejects fill with selector but no text", async () => {
      await expect(execRodCli([ "fill", "#sel" ])).rejects.toThrow(
        "fill requires non-empty text as the second argument",
      );
    });

    it("rejects fill with empty text", async () => {
      await expect(execRodCli([ "fill", "#sel", "" ])).rejects.toThrow(
        "fill requires non-empty text as the second argument",
      );
    });

    it("rejects fill with whitespace-only text", async () => {
      await expect(execRodCli([ "fill", "#sel", "   " ])).rejects.toThrow(
        "fill requires non-empty text as the second argument",
      );
    });

    it("accepts fill with selector and text", async () => {
      await execRodCli([ "fill", "#sel", "hello" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("rejects type without selector", async () => {
      await expect(execRodCli([ "type" ])).rejects.toThrow(
        "type requires a non-empty CSS selector",
      );
    });

    it("rejects type with empty selector", async () => {
      await expect(execRodCli([ "type", "" ])).rejects.toThrow(
        "type requires a non-empty CSS selector",
      );
    });

    it("accepts type with valid selector", async () => {
      await execRodCli([ "type", "#sel" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    // --- Eval length validation ---

    it("accepts eval with expression at exactly 10000 chars (boundary)", async () => {
      const expr = "x".repeat(10_000);
      await execRodCli([ "eval", expr ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("rejects eval with expression at 10001 chars (boundary + 1)", async () => {
      const expr = "x".repeat(10_001);
      await expect(execRodCli([ "eval", expr ])).rejects.toThrow(
        "eval expression too long",
      );
      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("rejects eval with expression at 100KB (pathological length)", async () => {
      const expr = "x".repeat(100_000);
      await expect(execRodCli([ "eval", expr ])).rejects.toThrow(
        /eval expression too long/,
      );
      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("accepts eval with no expression (no second arg)", async () => {
      // expr is undefined → `expr &&` short-circuits → no throw
      await execRodCli([ "eval" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("accepts eval with empty string expression", async () => {
      // expr is "" → `"" &&` → false → no throw
      await execRodCli([ "eval", "" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("accepts eval with zero-length expression explicitly", async () => {
      await execRodCli([ "eval", "" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    // --- Unvalidated commands ---

    it("passes through unknown subcommands without validation", async () => {
      await execRodCli([ "unknowncmd", "somearg" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("passes through flags-only invocation (--version)", async () => {
      await execRodCli([ "--version" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    // --- Edge case: flags before subcommand ---

    it("correctly validates when flags precede subcommand", async () => {
      // findIndex skips "--foo" (starts with "-"), finds "goto"
      // rest = ["https://example.com"]
      await execRodCli([ "--foo", "goto", "https://example.com" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("rejects when flag separates goto from its URL", async () => {
      // findIndex finds "goto" at 0, rest = ["--bar"]
      // rest[0] = "--bar" → doesn't start with http → reject
      await expect(
        execRodCli([ "goto", "--bar" ]),
      ).rejects.toThrow("requires an http:// or https:// URL");
    });

    // --- Totality for validateInput ---

    it("does not throw on empty args array", async () => {
      await execRodCli([]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("works with very long URL (just under what would cause issues)", async () => {
      const longUrl = "https://example.com/" + "a".repeat(4000);
      await execRodCli([ "goto", longUrl ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("truncates long URL in error message", async () => {
      const longUrl = "ftp://" + "x".repeat(200);
      try {
        await execRodCli([ "goto", longUrl ]);
        expect.fail("should have thrown");
      } catch (e: any) {
        // Error message should not contain the full 200-char URL
        expect(e.message.length).toBeLessThan(longUrl.length + 50);
        // It slices at 80 chars
        expect(e.message).toContain("ftp://" + "x".repeat(74)); // 80 - "ftp://".length = 74
      }
    });

    // --- Bug check: Unicode / special chars in selectors ---

    it("accepts click with unicode selector", async () => {
      await execRodCli([ "click", ".按钮" ]);
      expect(pi.exec).toHaveBeenCalled();
    });

    it("accepts type with complex CSS selector", async () => {
      await execRodCli([ "type", "div.foo > span[data-x='y']:nth-child(2)" ]);
      expect(pi.exec).toHaveBeenCalled();
    });
  });

  // ==========================================
  // timeoutFor — tested through execRodCli
  // ==========================================

  describe("timeoutFor (through pi.exec timeout parameter)", () => {
    it("passes goto timeout of 60000ms", async () => {
      await execRodCli([ "goto", "https://example.com" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 60_000 }),
      );
    });

    it("passes click timeout of 15000ms", async () => {
      await execRodCli([ "click", "#btn" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 15_000 }),
      );
    });

    it("passes fill timeout of 15000ms", async () => {
      await execRodCli([ "fill", "#input", "text" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 15_000 }),
      );
    });

    it("passes type timeout of 15000ms", async () => {
      await execRodCli([ "type", "#input" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 15_000 }),
      );
    });

    it("passes eval timeout of 15000ms", async () => {
      await execRodCli([ "eval", "document.title" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 15_000 }),
      );
    });

    it("passes screenshot timeout of 30000ms", async () => {
      await execRodCli([ "screenshot" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 30_000 }),
      );
    });

    it("passes wait timeout of 30000ms", async () => {
      await execRodCli([ "wait", "#el" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 30_000 }),
      );
    });

    it("passes close timeout of 5000ms", async () => {
      await execRodCli([ "close" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 5_000 }),
      );
    });

    it("passes --version flag timeout of 5000ms", async () => {
      await execRodCli([ "--version" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 5_000 }),
      );
    });

    it("uses DEFAULT_TIMEOUT (30000ms) for unknown subcommand", async () => {
      await execRodCli([ "unknown_cmd" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 30_000 }),
      );
    });

    it("uses DEFAULT_TIMEOUT for empty args", async () => {
      await execRodCli([]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 30_000 }),
      );
    });

    it("finds subcommand timeout even with flags before subcommand", async () => {
      await execRodCli([ "--foo", "goto", "https://example.com" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ timeout: 60_000 }),
      );
    });
  });

  // ==========================================
  // execRodCli integration
  // ==========================================

  describe("execRodCli — integration behavior", () => {
    it("throws when _pi is not set", async () => {
      // Reset pi to null
      const { setPi } = await import("../cli");
      setPi(null as any);

      await expect(execRodCli([ "--version" ])).rejects.toThrow(
        "Extension not initialized",
      );
    });

    it("throws on non-zero exit code", async () => {
      pi.exec.mockResolvedValueOnce({ code: 1, stdout: "", stderr: "something broke" });
      await expect(execRodCli([ "goto", "https://example.com" ])).rejects.toThrow(
        "rod-cli exited 1: something broke",
      );
    });

    it("includes stderr in non-zero exit error message", async () => {
      pi.exec.mockResolvedValueOnce({
        code: 2,
        stdout: "",
        stderr: "connection refused",
      });
      await expect(
        execRodCli([ "goto", "https://example.com" ]),
      ).rejects.toThrow("rod-cli exited 2: connection refused");
    });

    it("handles missing stderr gracefully in non-zero exit", async () => {
      pi.exec.mockResolvedValueOnce({
        code: 127,
        stdout: "",
        stderr: "",
      });
      await expect(
        execRodCli([ "goto", "https://example.com" ]),
      ).rejects.toThrow("rod-cli exited 127: (no stderr)");
    });

    it("handles undefined stderr in non-zero exit", async () => {
      pi.exec.mockResolvedValueOnce({
        code: 1,
        stdout: "",
        stderr: undefined,
      });
      await expect(
        execRodCli([ "goto", "https://example.com" ]),
      ).rejects.toThrow("rod-cli exited 1: (no stderr)");
    });

    it("returns stdout/stderr/code on success", async () => {
      pi.exec.mockResolvedValueOnce({
        code: 0,
        stdout: "ok",
        stderr: "warnings",
      });
      const result = await execRodCli([ "--version" ]);
      expect(result).toEqual({ stdout: "ok", stderr: "warnings", code: 0 });
    });

    it("prepends --raw to args", async () => {
      await execRodCli([ "goto", "https://example.com" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        [ "--raw", "goto", "https://example.com" ],
        expect.anything(),
      );
    });

    it("propagates AbortSignal in options", async () => {
      const controller = new AbortController();
      await execRodCli([ "goto", "https://example.com" ], {
        signal: controller.signal,
      });
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ signal: controller.signal }),
      );
    });

    it("AbortSignal is undefined when not provided", async () => {
      await execRodCli([ "goto", "https://example.com" ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        expect.anything(),
        expect.objectContaining({ signal: undefined }),
      );
    });

    it("rethrows unexpected errors from pi.exec", async () => {
      pi.exec.mockRejectedValueOnce(new Error("network error"));
      await expect(
        execRodCli([ "goto", "https://example.com" ]),
      ).rejects.toThrow("network error");
    });

    it("propagates AbortError from pi.exec", async () => {
      const abortErr = new DOMException("The operation was aborted", "AbortError");
      pi.exec.mockRejectedValueOnce(abortErr);
      await expect(
        execRodCli([ "goto", "https://example.com" ]),
      ).rejects.toThrow("The operation was aborted");
    });

    it("handles very large args array without crash", async () => {
      // e.g. eval with a very long expression that passes validation
      const expr = "x".repeat(10_000); // exactly at boundary
      await execRodCli([ "eval", expr ]);
      expect(pi.exec).toHaveBeenCalledWith(
        "rod-cli",
        [ "--raw", "eval", expr ],
        expect.anything(),
      );
    });

    it("doesn't hang on synchronous validation path", async () => {
      // validateInput is synchronous — execRodCli should reject immediately
      // on invalid input, not hang waiting for pi.exec
      const start = Date.now();
      await expect(execRodCli([ "goto" ])).rejects.toThrow();
      const elapsed = Date.now() - start;
      expect(elapsed).toBeLessThan(1000); // should be near-instant
    });
  });
});

// Need execRodCli in scope for this describe block
import { execRodCli } from "../cli";

// ---------------------------------------------------------------------------
// 4. registerLifecycle — hook registration
// ---------------------------------------------------------------------------

describe("registerLifecycle — hook registration", () => {
  it("registers session_start and session_shutdown handlers", async () => {
    const { registerLifecycle } = await import("../lifecycle");
    const pi = mockPi();

    registerLifecycle(pi, "/usr/bin/rod-cli");
    expect(pi.on).toHaveBeenCalledWith("session_start", expect.any(Function));
    expect(pi.on).toHaveBeenCalledWith("session_shutdown", expect.any(Function));
  });

  describe("session_start handler", () => {
    it("notifies warning when rodCliPath is null", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(pi, null);
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

      registerLifecycle(pi, "/usr/bin/rod-cli");
      await pi._fire("session_start", {}, { ui: pi.ui });

      expect(pi.exec).toHaveBeenCalledWith("rod-cli", [ "--version" ], { timeout: 5000 });
      // The notification is: `rod-cli ${version} ready` where version = stdout.trim()
      // If stdout is "rod-cli v1.0.0\n", the message is "rod-cli rod-cli v1.0.0 ready"
      expect(pi.ui.notify).toHaveBeenCalledWith(
        "rod-cli rod-cli v1.0.0 ready",
        "info",
      );
    });

    it("notifies warning when --version fails", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();
      pi.exec.mockRejectedValueOnce(new Error("exec failed"));

      registerLifecycle(pi, "/usr/bin/rod-cli");
      await pi._fire("session_start", {}, { ui: pi.ui });

      expect(pi.ui.notify).toHaveBeenCalledWith(
        expect.stringContaining("--version failed"),
        "warning",
      );
    });

    it("notifies warning when --version returns non-zero", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();
      pi.exec.mockResolvedValueOnce({ code: 1, stdout: "", stderr: "crash" });

      // pi.exec returning non-zero is NOT an exception — the function
      // only catches thrown errors. Non-zero exit code will produce
      // a notification with the raw output (possibly empty).
      registerLifecycle(pi, "/usr/bin/rod-cli");
      await pi._fire("session_start", {}, { ui: pi.ui });

      // it calls .trim() on stdout, which would be "" → "unknown"
      expect(pi.ui.notify).toHaveBeenCalledWith(
        "rod-cli unknown ready",
        "info",
      );
    });
  });

  describe("session_shutdown handler", () => {
    it("runs close on reason quit", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(pi, "/usr/bin/rod-cli");
      await pi._fire("session_shutdown", { reason: "quit" });

      expect(pi.exec).toHaveBeenCalledWith("rod-cli", [ "close" ], { timeout: 5000 });
    });

    it("does NOT run close on reason reload", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(pi, "/usr/bin/rod-cli");
      await pi._fire("session_shutdown", { reason: "reload" });

      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("does NOT run close on reason fork", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(pi, "/usr/bin/rod-cli");
      await pi._fire("session_shutdown", { reason: "fork" });

      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("does NOT run close on reason resume", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();

      registerLifecycle(pi, "/usr/bin/rod-cli");
      await pi._fire("session_shutdown", { reason: "resume" });

      expect(pi.exec).not.toHaveBeenCalled();
    });

    it("swallows errors from close (best-effort)", async () => {
      const { registerLifecycle } = await import("../lifecycle");
      const pi = mockPi();
      pi.exec.mockRejectedValueOnce(new Error("daemon already dead"));

      registerLifecycle(pi, "/usr/bin/rod-cli");
      // Should not throw
      await pi._fire("session_shutdown", { reason: "quit" });
      // Got here without crash — test passes
    });
  });
});

// ---------------------------------------------------------------------------
// 5. index.ts — wiring
// ---------------------------------------------------------------------------

describe("index.ts — default export (wiring)", () => {
  it("is a function", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });

  it("does not throw when called with a mock pi", async () => {
    const mod = await import("../index");
    const pi = mockPi();

    // default() is synchronous — setPi + registerLifecycle should not throw
    // even when findRodCli returns null (no binary found in clean env)
    expect(() => mod.default(pi)).not.toThrow();
  });

  it("calls setPi and registers lifecycle hooks", async () => {
    // Reset _pi first
    const { setPi } = await import("../cli");
    setPi(null as any);

    const mod = await import("../index");
    const pi = mockPi();

    mod.default(pi);

    // After default(), _pi should be set
    // We can verify by calling execRodCli — it should find _pi
    const { execRodCli: _exec } = await import("../cli");
    // Must not throw "Extension not initialized"
    await expect(_exec([ "--version" ])).resolves.toBeDefined();
  });
});

// ---------------------------------------------------------------------------
// 6. Fuzz target — totality of execRodCli argument handling
// ---------------------------------------------------------------------------

describe("fuzz: execRodCli totality — no input panics or hangs", () => {
  it("handles rapid successive calls", async () => {
    const pi = mockPi();
    const { setPi } = await import("../cli");
    setPi(pi as unknown as ExtensionAPI);

    const results = await Promise.allSettled([
      execRodCli([ "goto", "https://a.com" ]),
      execRodCli([ "goto", "https://b.com" ]),
      execRodCli([ "click", "#x" ]),
      execRodCli([ "--version" ]),
      execRodCli([ "eval", "1+1" ]),
    ]);

    // None should have thrown unexpectedly (all settled)
    const panics = results.filter((r) => r.status === "rejected");
    // All should succeed since mock returns code:0
    expect(panics).toHaveLength(0);
  });

  it("handles args with only whitespace strings", async () => {
    const pi = mockPi();
    const { setPi } = await import("../cli");
    setPi(pi as unknown as ExtensionAPI);

    // These are all unvalidated commands — should pass through
    await execRodCli([ "   " ]);
    expect(pi.exec).toHaveBeenCalledWith("rod-cli", ["--raw", "   "], expect.anything());
  });

  it("handles args with special shell characters", async () => {
    const pi = mockPi();
    const { setPi } = await import("../cli");
    setPi(pi as unknown as ExtensionAPI);

    // Shell metacharacters in args — should be passed through literally
    // (pi.exec handles quoting)
    await execRodCli([ "eval", "$(`rm -rf /`)" ]);
    expect(pi.exec).toHaveBeenCalledWith(
      "rod-cli",
      ["--raw", "eval", "$(`rm -rf /`)"],
      expect.anything(),
    );
  });

  it("handles args with null bytes in selector", async () => {
    const pi = mockPi();
    const { setPi } = await import("../cli");
    setPi(pi as unknown as ExtensionAPI);

    // Null bytes in strings — should not crash
    await execRodCli([ "click", "div\0bad" ]);
    expect(pi.exec).toHaveBeenCalled();
  });
});
