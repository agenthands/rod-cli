import { describe, it, expect } from "vitest";

// Test that the source files parse and export correctly
describe("extension foundation", () => {
  it("findRodCli resolves or returns null", async () => {
    const { findRodCli } = await import("../cli");
    const result = findRodCli();
    // On a dev machine with rod-cli installed: string.
    // In CI without rod-cli: null.
    // Either is acceptable -- we just verify it doesn't throw.
    expect(result === null || typeof result === "string").toBe(true);
  });

  it("types module exports SessionParam", async () => {
    const { SessionParam } = await import("../types");
    expect(SessionParam).toBeDefined();
  });

  it("lifecycle module exports registerLifecycle", async () => {
    const { registerLifecycle } = await import("../lifecycle");
    expect(typeof registerLifecycle).toBe("function");
  });

  it("index module exports default function", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });
});
