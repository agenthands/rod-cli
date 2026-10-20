import { describe, it, expect, vi, afterEach } from "vitest";
import { getEventListeners } from "node:events";
import { sleep, waitForSelector } from "../wait";

describe("sleep", () => {
  it("leaves no abort listener on the signal once it resolves", async () => {
    const ac = new AbortController();
    await sleep(1, ac.signal);
    expect(getEventListeners(ac.signal, "abort")).toHaveLength(0);
  });
});

const POLL_ARGV = ["--raw", "--max-output", "51200", "eval", "--", '!!document.querySelector("#x")'];

function stdoutSeq(...outs: string[]) {
  const execFn = vi.fn();
  for (const stdout of outs) execFn.mockResolvedValueOnce({ code: 0, stdout, stderr: "" });
  return execFn;
}

describe("waitForSelector", () => {
  afterEach(() => vi.useRealTimers());

  it("polls every 500ms until eval prints true", async () => {
    vi.useFakeTimers();
    const execFn = stdoutSeq("false\n", "false\n", "true\n");
    const done = waitForSelector(execFn, { selector: "#x", timeoutMs: 5000 });
    await vi.advanceTimersByTimeAsync(1000);
    await done;
    expect(execFn).toHaveBeenCalledTimes(3);
    for (const call of execFn.mock.calls) expect(call[0]).toEqual(POLL_ARGV);
  });

  it("rejects with the exact timeout message when the selector never appears", async () => {
    vi.useFakeTimers();
    const execFn = vi.fn().mockResolvedValue({ code: 0, stdout: "false\n", stderr: "" });
    const done = expect(waitForSelector(execFn, { selector: "#x", timeoutMs: 1000 })).rejects.toThrow(
      new Error("Timed out waiting 1000ms for selector: #x"),
    );
    await vi.advanceTimersByTimeAsync(1500);
    await done;
  });

  it.each([0, -1])("with timeoutMs %i checks exactly once and does not poll", async (timeoutMs) => {
    await waitForSelector(stdoutSeq("true\n"), { selector: "#x", timeoutMs });
    const miss = stdoutSeq("false\n", "true\n");
    await expect(waitForSelector(miss, { selector: "#x", timeoutMs })).rejects.toThrow(
      new Error(`Timed out waiting ${timeoutMs}ms for selector: #x`),
    );
    expect(miss).toHaveBeenCalledTimes(1);
  });
});
