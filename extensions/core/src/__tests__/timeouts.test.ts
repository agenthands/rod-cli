import { describe, it, expect } from "vitest";
import fc from "fast-check";
import { DEFAULT_TIMEOUT, timeoutFor } from "../timeouts";
import { validateInput } from "../validate";
import { withSession } from "../argv";
import { builderCall } from "./builder-arbs";

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

  it("ignores table keys that appear as user values or session names", () => {
    expect(timeoutFor(["hover", "--", "close"])).toBe(DEFAULT_TIMEOUT);
    expect(timeoutFor(["-s", "goto", "tab-list"])).toBe(DEFAULT_TIMEOUT);
  });
});

// gap-plan ext:EXT-5: a session name never changes how a builder argv is timed or validated.
describe("withSession (property)", () => {
  const argvArb = builderCall(fc.constant(undefined)).map((c) => c.argv);
  const sessionArb = fc.oneof(fc.string({ minLength: 1 }), fc.constantFrom("close", "goto", "--version"));
  // Names outside the session charset are rejected on their own (EXT-12), so validation is compared on valid names.
  const validSessionArb = fc.oneof(
    fc.stringMatching(/^[A-Za-z0-9._-]{1,64}$/).filter((n) => n !== "." && n !== ".."),
    fc.constantFrom("close", "goto", "--version"),
  );
  const thrown = (args: string[]) => {
    try { validateInput(args); return undefined; } catch (e) { return (e as Error).message; }
  };

  it("timeoutFor(withSession(s, a)) equals timeoutFor(a)", () => {
    fc.assert(fc.property(sessionArb, argvArb, (s, a) => {
      expect(timeoutFor(withSession(s, a))).toBe(timeoutFor(a));
    }));
  });

  it("validateInput throws for withSession(s, a) exactly when it throws for a", () => {
    fc.assert(fc.property(validSessionArb, argvArb, (s, a) => {
      expect(thrown(withSession(s, a))).toBe(thrown(a));
    }));
  });
});
