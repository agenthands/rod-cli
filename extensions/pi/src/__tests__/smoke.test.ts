import { describe, it, expect } from "vitest";

describe("extension foundation", () => {
  it("index module exports default function", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });

  it("types module exports SessionParam", async () => {
    const { SessionParam } = await import("../types");
    expect(SessionParam).toBeDefined();
  });

  it("lifecycle module exports registerLifecycle", async () => {
    const { registerLifecycle } = await import("../lifecycle");
    expect(typeof registerLifecycle).toBe("function");
  });

  it("@agenthands/rod-cli-core resolves and exports builders", async () => {
    const core = await import("@agenthands/rod-cli-core");
    expect(typeof core.buildGotoArgs).toBe("function");
    expect(typeof core.execRodCli).toBe("function");
    expect(typeof core.findRodCli).toBe("function");
  });
});
