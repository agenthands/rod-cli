import { describe, it, expect, vi } from "vitest";
import { execRodCli, type ExecFn } from "../exec";

/** A mock ExecFn that always succeeds. */
function mockExecFn(): ExecFn {
  return vi.fn().mockResolvedValue({ code: 0, stdout: "", stderr: "" });
}

describe("validateInput (through execRodCli rejection before exec)", () => {
  // --- B51-1 regression: -- separator is now skipped ---

  it("B51-1: goto with -- separator and https resolves (no throw)", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["goto", "--", "https://example.com"]);
    expect(fn).toHaveBeenCalled();
  });

  it("B51-1: goto with -- separator and http resolves", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["goto", "--", "http://example.com"]);
    expect(fn).toHaveBeenCalled();
  });

  it("B51-1: goto with -- separator and ftp throws (validates real arg)", async () => {
    const fn = mockExecFn();
    await expect(
      execRodCli(fn, ["goto", "--", "ftp://example.com"]),
    ).rejects.toThrow("got: ftp://example.com");
    expect(fn).not.toHaveBeenCalled();
  });

  it("B51-1: click with -- separator and empty selector throws", async () => {
    const fn = mockExecFn();
    await expect(
      execRodCli(fn, ["click", "--", ""]),
    ).rejects.toThrow("click requires a non-empty CSS selector");
    expect(fn).not.toHaveBeenCalled();
  });

  it("B51-1: fill with -- separator and empty text throws", async () => {
    const fn = mockExecFn();
    await expect(
      execRodCli(fn, ["fill", "--", "#s", ""]),
    ).rejects.toThrow("fill requires non-empty text as the second argument");
    expect(fn).not.toHaveBeenCalled();
  });

  it("B51-1: eval with -- separator and 10001 chars throws", async () => {
    const fn = mockExecFn();
    await expect(
      execRodCli(fn, ["eval", "--", "x".repeat(10001)]),
    ).rejects.toThrow("eval expression too long");
    expect(fn).not.toHaveBeenCalled();
  });

  it("B51-1: eval with -- separator and 10000 chars resolves", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["eval", "--", "x".repeat(10000)]);
    expect(fn).toHaveBeenCalled();
  });

  it("B51-1: -s prefixed goto with -- separator resolves", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["-s", "admin", "goto", "--", "https://example.com"]);
    expect(fn).toHaveBeenCalled();
  });

  // --- Original no--- separator cases (preserved) ---

  it("rejects goto without URL", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["goto"])).rejects.toThrow("goto requires a URL argument");
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects goto with empty URL", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["goto", ""])).rejects.toThrow("goto requires a URL argument");
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects goto with ftp:// scheme", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["goto", "ftp://example.com"])).rejects.toThrow(
      "requires an http:// or https:// URL",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects goto with ws:// scheme", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["goto", "ws://example.com"])).rejects.toThrow(
      "requires an http:// or https:// URL",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects goto with file:// scheme", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["goto", "file:///etc/passwd"])).rejects.toThrow(
      "requires an http:// or https:// URL",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("accepts goto with valid https:// URL", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["goto", "https://example.com"]);
    expect(fn).toHaveBeenCalled();
  });

  it("accepts goto with valid http:// URL", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["goto", "http://example.com"]);
    expect(fn).toHaveBeenCalled();
  });

  it("rejects click without selector", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["click"])).rejects.toThrow(
      "click requires a non-empty CSS selector",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects click with empty selector", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["click", ""])).rejects.toThrow(
      "click requires a non-empty CSS selector",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects click with whitespace-only selector", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["click", "   \t  "])).rejects.toThrow(
      "click requires a non-empty CSS selector",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("accepts click with valid selector", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["click", "#myid"]);
    expect(fn).toHaveBeenCalled();
  });

  it("rejects fill without selector", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["fill"])).rejects.toThrow(
      "fill requires a non-empty CSS selector",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects fill with selector but no text", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["fill", "#sel"])).rejects.toThrow(
      "fill requires non-empty text as the second argument",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects fill with empty text", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["fill", "#sel", ""])).rejects.toThrow(
      "fill requires non-empty text as the second argument",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("accepts fill with selector and text", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["fill", "#sel", "hello"]);
    expect(fn).toHaveBeenCalled();
  });

  it("rejects type without selector", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["type"])).rejects.toThrow(
      "type requires a non-empty CSS selector",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("rejects type with empty selector", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, ["type", ""])).rejects.toThrow(
      "type requires a non-empty CSS selector",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("accepts type with valid selector", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["type", "#sel"]);
    expect(fn).toHaveBeenCalled();
  });

  it("accepts eval with expression at exactly 10000 chars (boundary)", async () => {
    const fn = mockExecFn();
    const expr = "x".repeat(10_000);
    await execRodCli(fn, ["eval", expr]);
    expect(fn).toHaveBeenCalled();
  });

  it("rejects eval with expression at 10001 chars (boundary + 1)", async () => {
    const fn = mockExecFn();
    const expr = "x".repeat(10_001);
    await expect(execRodCli(fn, ["eval", expr])).rejects.toThrow(
      "eval expression too long",
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("accepts eval with no expression (no second arg)", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["eval"]);
    expect(fn).toHaveBeenCalled();
  });

  it("passes through unknown subcommands without validation", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["unknowncmd", "somearg"]);
    expect(fn).toHaveBeenCalled();
  });

  it("passes through flags-only invocation (--version)", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["--version"]);
    expect(fn).toHaveBeenCalled();
  });

  it("correctly validates when flags precede subcommand", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["--foo", "goto", "https://example.com"]);
    expect(fn).toHaveBeenCalled();
  });

  it("does not throw on empty args array", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, []);
    expect(fn).toHaveBeenCalled();
  });

  it("truncates long URL in error message", async () => {
    const fn = mockExecFn();
    const longUrl = "ftp://" + "x".repeat(200);
    try {
      await execRodCli(fn, ["goto", longUrl]);
      expect.fail("should have thrown");
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      expect(msg.length).toBeLessThan(longUrl.length + 50);
      expect(msg).toContain("ftp://" + "x".repeat(74));
    }
  });
});
