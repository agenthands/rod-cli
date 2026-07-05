import { describe, it, expect } from "vitest";
import { timeoutFor } from "../timeouts";

describe("timeoutFor", () => {
  it("returns goto timeout of 60000ms", () => {
    expect(timeoutFor(["goto", "https://example.com"])).toBe(60_000);
  });

  it("returns snapshot timeout of 15000ms", () => {
    expect(timeoutFor(["snapshot"])).toBe(15_000);
  });

  it("returns click timeout of 15000ms", () => {
    expect(timeoutFor(["click", "#btn"])).toBe(15_000);
  });

  it("returns fill timeout of 15000ms", () => {
    expect(timeoutFor(["fill", "#input", "text"])).toBe(15_000);
  });

  it("returns type timeout of 15000ms", () => {
    expect(timeoutFor(["type", "#input"])).toBe(15_000);
  });

  it("returns eval timeout of 15000ms", () => {
    expect(timeoutFor(["eval", "document.title"])).toBe(15_000);
  });

  it("returns screenshot timeout of 30000ms", () => {
    expect(timeoutFor(["screenshot"])).toBe(30_000);
  });

  it("returns wait timeout of 30000ms", () => {
    expect(timeoutFor(["wait", "#el"])).toBe(30_000);
  });

  it("returns close timeout of 5000ms", () => {
    expect(timeoutFor(["close"])).toBe(5_000);
  });

  it("returns --version flag timeout of 5000ms", () => {
    expect(timeoutFor(["--version"])).toBe(5_000);
  });

  it("uses DEFAULT_TIMEOUT (30000ms) for unknown subcommand", () => {
    expect(timeoutFor(["unknown_cmd"])).toBe(30_000);
  });

  it("uses DEFAULT_TIMEOUT for empty args", () => {
    expect(timeoutFor([])).toBe(30_000);
  });

  it("finds subcommand timeout even with flags before subcommand", () => {
    expect(timeoutFor(["--foo", "goto", "https://example.com"])).toBe(60_000);
  });

  it("finds subcommand timeout with -s session prefix", () => {
    expect(timeoutFor(["-s", "admin", "goto", "https://example.com"])).toBe(60_000);
  });

  it("finds subcommand timeout with --session prefix", () => {
    expect(timeoutFor(["--session", "admin", "goto", "https://example.com"])).toBe(60_000);
  });
});
