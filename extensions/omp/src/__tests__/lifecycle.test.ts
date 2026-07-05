import { describe, it, expect } from "bun:test";
import { createMockApi } from "./mockapi";
import extension from "../index";

describe("lifecycle", () => {
  it("session_start fires --version exec and info notify", async () => {
    const mock = createMockApi();
    extension(mock.api);

    await mock.fire("session_start");

    // Should have called --version
    const versionCall = mock.execCalls.find((args) => args.includes("--version"));
    expect(versionCall).toBeDefined();

    // Should have an info notification
    const infoNotify = mock.notifications.find((n) => n.level === "info");
    expect(infoNotify).toBeDefined();
    expect(infoNotify!.msg).toMatch(/rod-cli.*ready/);
  });

  it("session_start notifies ready (info), not a warning/usage-dump, when --version exits non-zero", async () => {
    const mock = createMockApi({ execCode: 1 });
    extension(mock.api);

    await mock.fire("session_start");

    const infoNotify = mock.notifications.find((n) => n.level === "info");
    expect(infoNotify).toBeDefined();
    expect(infoNotify!.msg).toBe("rod-cli ready");
    const warnNotify = mock.notifications.find((n) => n.level === "warning");
    expect(warnNotify).toBeUndefined();
  });

  it("session_shutdown fires close unconditionally", async () => {
    const mock = createMockApi();
    extension(mock.api);

    await mock.fire("session_shutdown");

    const closeCall = mock.execCalls.find((args) => args.includes("close"));
    expect(closeCall).toBeDefined();
  });
});
