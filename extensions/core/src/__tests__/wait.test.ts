import { describe, it, expect } from "vitest";
import { getEventListeners } from "node:events";
import { sleep } from "../wait";

describe("sleep", () => {
  it("leaves no abort listener on the signal once it resolves", async () => {
    const ac = new AbortController();
    await sleep(1, ac.signal);
    expect(getEventListeners(ac.signal, "abort")).toHaveLength(0);
  });
});
