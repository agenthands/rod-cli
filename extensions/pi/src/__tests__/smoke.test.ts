import { describe, it, expect } from "vitest";
import extensionDefault from "../index";
import { SessionParam } from "../types";
import { registerLifecycle } from "../lifecycle";
import { buildGotoArgs, execRodCli, findRodCli } from "@agenthands/rod-cli-core";

describe("extension foundation", () => {
  it("index module exports default function", () => {
    expect(typeof extensionDefault).toBe("function");
  });

  it("types module exports SessionParam", () => {
    expect(SessionParam).toBeDefined();
  });

  it("lifecycle module exports registerLifecycle", () => {
    expect(typeof registerLifecycle).toBe("function");
  });

  it("@agenthands/rod-cli-core resolves and exports builders", () => {
    expect(typeof buildGotoArgs).toBe("function");
    expect(typeof execRodCli).toBe("function");
    expect(typeof findRodCli).toBe("function");
  });
});
