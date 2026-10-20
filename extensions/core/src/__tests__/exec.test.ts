import { describe, it, expect, vi } from "vitest";
import { execRodCli, type ExecFn } from "../exec";
import { withSession } from "../argv";

function mockExecFn(result?: Partial<{ code: number; stdout: string; stderr: string }>): ExecFn {
  return vi.fn().mockResolvedValue({
    code: 0,
    stdout: "",
    stderr: "",
    ...result,
  });
}

describe("execRodCli — integration behavior", () => {
  it("TestExecPrependsMaxOutput: prepends --raw --max-output 51200 to args", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, ["goto", "https://example.com"]);
    expect(fn).toHaveBeenCalledWith(
      ["--raw", "--max-output", "51200", "goto", "https://example.com"],
      expect.anything(),
    );
  });

  it("returns stdout/stderr/code on success", async () => {
    const fn = mockExecFn({ stdout: "ok", stderr: "warnings" });
    const result = await execRodCli(fn, ["--version"]);
    expect(result).toEqual({ stdout: "ok", stderr: "warnings", code: 0 });
  });

  it("throws on non-zero exit code", async () => {
    const fn = mockExecFn({ code: 1, stderr: "something broke" });
    await expect(
      execRodCli(fn, ["goto", "https://example.com"]),
    ).rejects.toThrow("rod-cli exited 1: something broke");
  });

  it("includes stderr in non-zero exit error message", async () => {
    const fn = mockExecFn({ code: 2, stderr: "connection refused" });
    await expect(
      execRodCli(fn, ["goto", "https://example.com"]),
    ).rejects.toThrow("rod-cli exited 2: connection refused");
  });

  it("handles missing stderr gracefully in non-zero exit", async () => {
    const fn = mockExecFn({ code: 127, stderr: "" });
    await expect(
      execRodCli(fn, ["goto", "https://example.com"]),
    ).rejects.toThrow("rod-cli exited 127: (no stderr)");
  });

  it("handles undefined stderr in non-zero exit", async () => {
    const fn = mockExecFn({ code: 1, stderr: undefined });
    await expect(
      execRodCli(fn, ["goto", "https://example.com"]),
    ).rejects.toThrow("rod-cli exited 1: (no stderr)");
  });

  it("propagates AbortSignal in options", async () => {
    const fn = mockExecFn();
    const controller = new AbortController();
    await execRodCli(fn, ["goto", "https://example.com"], {
      signal: controller.signal,
    });
    expect(fn).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ signal: controller.signal }),
    );
  });

  it("rethrows unexpected errors from execFn", async () => {
    const fn = vi.fn().mockRejectedValue(new Error("network error"));
    await expect(
      execRodCli(fn, ["goto", "https://example.com"]),
    ).rejects.toThrow("network error");
  });

  it("propagates AbortError from execFn", async () => {
    const abortErr = new DOMException("The operation was aborted", "AbortError");
    const fn = vi.fn().mockRejectedValue(abortErr);
    await expect(
      execRodCli(fn, ["goto", "https://example.com"]),
    ).rejects.toThrow("The operation was aborted");
  });

  it("doesn't hang on synchronous validation path", async () => {
    const fn = mockExecFn();
    const start = Date.now();
    await expect(execRodCli(fn, ["goto"])).rejects.toThrow();
    const elapsed = Date.now() - start;
    expect(elapsed).toBeLessThan(1000);
  });
});

describe("execRodCli — session name charset (EXT-12)", () => {
  it("rejects a path-escaping session before exec", async () => {
    const fn = mockExecFn();
    await expect(execRodCli(fn, withSession("/../../tmp/evil", ["close"]))).rejects.toThrow(
      new Error(`invalid session name "/../../tmp/evil": use letters, digits, '.', '_' or '-' (max 64)`),
    );
    expect(fn).not.toHaveBeenCalled();
  });

  it("passes a plain session name through to exec", async () => {
    const fn = mockExecFn();
    await execRodCli(fn, withSession("admin", ["close"]));
    expect(fn).toHaveBeenCalledWith(["--raw", "--max-output", "51200", "-s", "admin", "close"], expect.anything());
  });
});
