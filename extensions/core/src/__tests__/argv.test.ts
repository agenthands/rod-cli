import { describe, it, expect } from "vitest";
import fc from "fast-check";
import { builderCall, modelString, SCREENSHOT_OUT } from "./builder-arbs";
import {
  withSession,
  buildGotoArgs,
  buildSnapshotArgs,
  buildClickArgs,
  buildTypeArgs,
  buildEvalArgs,
  buildScreenshotArgs,
  buildScrollArgs,
  buildTabsArgs,
  buildNavigateArgs,
  buildCookiesArgs,
  buildStorageArgs,
  buildFillFormArgs,
  buildPressArgs,
  buildSelectArgs,
  buildHoverArgs,
} from "../argv";

describe("withSession", () => {
  it("prepends -s when session is provided", () => {
    expect(withSession("admin", ["goto", "https://x"])).toEqual([
      "-s", "admin", "goto", "https://x",
    ]);
  });

  it("returns args unchanged when session is undefined", () => {
    expect(withSession(undefined, ["goto", "https://x"])).toEqual([
      "goto", "https://x",
    ]);
  });
});

describe("buildGotoArgs", () => {
  it("builds goto argv with -- separator", () => {
    expect(buildGotoArgs({ url: "https://example.com" })).toEqual([
      "goto", "--", "https://example.com",
    ]);
  });

  it("includes session prefix", () => {
    expect(buildGotoArgs({ url: "https://x", session: "admin" })).toEqual([
      "-s", "admin", "goto", "--", "https://x",
    ]);
  });
});

describe("buildSnapshotArgs", () => {
  it("builds snapshot argv", () => {
    expect(buildSnapshotArgs({})).toEqual(["snapshot"]);
  });

  it("includes session prefix", () => {
    expect(buildSnapshotArgs({ session: "admin" })).toEqual(["-s", "admin", "snapshot"]);
  });

  it("passes interactive before -- and the scope after it", () => {
    expect(buildSnapshotArgs({ selector: "s1e3", interactive: true })).toEqual(["snapshot", "-i", "--", "s1e3"]);
  });
});

describe("buildClickArgs", () => {
  it("builds click argv with -- separator", () => {
    expect(buildClickArgs({ selector: "#btn" })).toEqual([
      "click", "--", "#btn",
    ]);
  });

  it("builds dblclick argv when doubleClick is true", () => {
    expect(buildClickArgs({ selector: "#btn", doubleClick: true })).toEqual([
      "dblclick", "--", "#btn",
    ]);
  });

  it("includes session prefix", () => {
    expect(buildClickArgs({ selector: "#x", session: "admin" })).toEqual([
      "-s", "admin", "click", "--", "#x",
    ]);
  });
});

describe("buildTypeArgs", () => {
  it("builds type argv with -- separator", () => {
    expect(buildTypeArgs({ selector: "#input", text: "hello" })).toEqual([
      "type", "--", "#input", "hello",
    ]);
  });

  it("includes session prefix", () => {
    expect(buildTypeArgs({ selector: "#x", text: "hi", session: "admin" })).toEqual([
      "-s", "admin", "type", "--", "#x", "hi",
    ]);
  });
});

describe("buildEvalArgs", () => {
  it("builds eval argv with -- separator", () => {
    expect(buildEvalArgs({ expression: "1+1" })).toEqual([
      "eval", "--", "1+1",
    ]);
  });

  it("includes session prefix", () => {
    expect(buildEvalArgs({ expression: "1+1", session: "admin" })).toEqual([
      "-s", "admin", "eval", "--", "1+1",
    ]);
  });
});

describe("buildScreenshotArgs", () => {
  it("builds screenshot argv without selector", () => {
    expect(buildScreenshotArgs({})).toEqual(["screenshot"]);
  });

  it("builds screenshot argv with --selector", () => {
    expect(buildScreenshotArgs({ selector: "#chart" })).toEqual([
      "screenshot", "--selector=#chart",
    ]);
  });

  it("keeps a flag-like selector inside its own --selector= token", () => {
    expect(buildScreenshotArgs({ selector: "--out", out: "/tmp/x.png", session: "admin" })).toEqual([
      "-s", "admin", "screenshot", "--selector=--out", "--out", "/tmp/x.png",
    ]);
  });

  it("includes session prefix", () => {
    expect(buildScreenshotArgs({ session: "admin" })).toEqual(["-s", "admin", "screenshot"]);
  });
});

describe("buildScrollArgs", () => {
  it("with no distance scrolls one viewport down", () => {
    expect(buildScrollArgs({})).toEqual(["mousewheel"]);
    expect(buildScrollArgs({ direction: "down" })).toEqual(["mousewheel"]);
  });

  it("requires a distance for any direction but down", () => {
    expect(() => buildScrollArgs({ direction: "up" })).toThrow("requires a 'distance'");
  });

  it("maps direction and distance to dx/dy after --", () => {
    expect(buildScrollArgs({ direction: "left", distance: 200 })).toEqual(["mousewheel", "--", "-200", "0"]);
    expect(buildScrollArgs({ direction: "right", distance: 200 })).toEqual(["mousewheel", "--", "200", "0"]);
    expect(buildScrollArgs({ direction: "up", distance: 300 })).toEqual(["mousewheel", "--", "0", "-300"]);
    expect(buildScrollArgs({ direction: "down", distance: 500 })).toEqual(["mousewheel", "--", "0", "500"]);
  });

  it("passes the selector as the trailing ref", () => {
    expect(buildScrollArgs({ selector: "s1e4" })).toEqual(["mousewheel", "--", "s1e4"]);
    expect(buildScrollArgs({ selector: "s1e4", direction: "up", distance: 100 })).toEqual([
      "mousewheel", "--", "0", "-100", "s1e4",
    ]);
  });

  it("includes session prefix", () => {
    expect(buildScrollArgs({ session: "admin" })).toEqual(["-s", "admin", "mousewheel"]);
  });
});

describe("buildTabsArgs", () => {
  it("builds list argv", () => {
    expect(buildTabsArgs({ action: "list" })).toEqual(["tab-list"]);
  });

  it("builds new argv with -- separator", () => {
    expect(buildTabsArgs({ action: "new", url: "https://x" })).toEqual([
      "tab-new", "--", "https://x",
    ]);
  });

  it("throws new without url", () => {
    expect(() => buildTabsArgs({ action: "new" })).toThrow(
      "action 'new' requires a 'url' parameter",
    );
  });

  it("builds close argv with -- separator", () => {
    expect(buildTabsArgs({ action: "close", index: 2 })).toEqual([
      "tab-close", "--", "2",
    ]);
  });

  it("throws close without index", () => {
    expect(() => buildTabsArgs({ action: "close" })).toThrow(
      "action 'close' requires an 'index' parameter",
    );
  });

  it("builds select argv with -- separator", () => {
    expect(buildTabsArgs({ action: "select", index: 0 })).toEqual([
      "tab-select", "--", "0",
    ]);
  });

  it("throws select without index", () => {
    expect(() => buildTabsArgs({ action: "select" })).toThrow(
      "action 'select' requires an 'index' parameter",
    );
  });

  it("throws unknown action", () => {
    expect(() => buildTabsArgs({ action: "bad" })).toThrow("Unknown tab action");
  });

  it("includes session prefix", () => {
    expect(buildTabsArgs({ action: "list", session: "admin" })).toEqual([
      "-s", "admin", "tab-list",
    ]);
  });
});

describe("buildNavigateArgs", () => {
  it("builds reload argv", () => {
    expect(buildNavigateArgs({ action: "reload" })).toEqual(["reload"]);
  });

  it("builds back argv", () => {
    expect(buildNavigateArgs({ action: "back" })).toEqual(["go-back"]);
  });

  it("builds forward argv", () => {
    expect(buildNavigateArgs({ action: "forward" })).toEqual(["go-forward"]);
  });

  it("throws unknown action", () => {
    expect(() => buildNavigateArgs({ action: "bad" })).toThrow(
      "Unknown navigation action",
    );
  });

  it("includes session prefix", () => {
    expect(buildNavigateArgs({ action: "reload", session: "admin" })).toEqual([
      "-s", "admin", "reload",
    ]);
  });
});

describe("buildCookiesArgs", () => {
  it("builds get argv", () => {
    expect(buildCookiesArgs({ action: "get" })).toEqual(["cookie-get"]);
  });

  it("builds set argv with -- separator", () => {
    expect(buildCookiesArgs({ action: "set", name: "k", value: "v" })).toEqual([
      "cookie-set", "--", "k", "v",
    ]);
  });

  it("throws set without name", () => {
    expect(() => buildCookiesArgs({ action: "set", value: "v" })).toThrow(
      "action 'set' requires a 'name' parameter",
    );
  });

  it("throws set without value", () => {
    expect(() => buildCookiesArgs({ action: "set", name: "k" })).toThrow(
      "action 'set' requires a 'value' parameter",
    );
  });

  it("builds delete argv with -- separator", () => {
    expect(buildCookiesArgs({ action: "delete", name: "k" })).toEqual([
      "cookie-delete", "--", "k",
    ]);
  });

  it("throws delete without name", () => {
    expect(() => buildCookiesArgs({ action: "delete" })).toThrow(
      "action 'delete' requires a 'name' parameter",
    );
  });

  it("builds clear argv", () => {
    expect(buildCookiesArgs({ action: "clear" })).toEqual(["cookie-clear"]);
  });

  it("throws unknown action", () => {
    expect(() => buildCookiesArgs({ action: "bad" })).toThrow("Unknown cookie action");
  });

  it("includes session prefix", () => {
    expect(buildCookiesArgs({ action: "get", session: "admin" })).toEqual([
      "-s", "admin", "cookie-get",
    ]);
  });
});

describe("buildStorageArgs", () => {
  it("builds local get argv", () => {
    expect(buildStorageArgs({ action: "get" })).toEqual(["localstorage-get"]);
  });

  it("builds local get with key", () => {
    expect(buildStorageArgs({ action: "get", key: "k" })).toEqual([
      "localstorage-get", "--", "k",
    ]);
  });

  it("builds local set argv with -- separator", () => {
    expect(buildStorageArgs({ action: "set", key: "k", value: "v" })).toEqual([
      "localstorage-set", "--", "k", "v",
    ]);
  });

  it("throws set without key", () => {
    expect(() => buildStorageArgs({ action: "set", value: "v" })).toThrow(
      "action 'set' requires a 'key' parameter",
    );
  });

  it("throws set without value", () => {
    expect(() => buildStorageArgs({ action: "set", key: "k" })).toThrow(
      "action 'set' requires a 'value' parameter",
    );
  });

  it("builds local delete argv with -- separator", () => {
    expect(buildStorageArgs({ action: "delete", key: "k" })).toEqual([
      "localstorage-delete", "--", "k",
    ]);
  });

  it("throws delete without key", () => {
    expect(() => buildStorageArgs({ action: "delete" })).toThrow(
      "action 'delete' requires a 'key' parameter",
    );
  });

  it("builds clear argv", () => {
    expect(buildStorageArgs({ action: "clear" })).toEqual(["localstorage-clear"]);
  });

  it("builds session storage argv", () => {
    expect(buildStorageArgs({ action: "get", storageType: "session" })).toEqual([
      "sessionstorage-get",
    ]);
  });

  it("throws unknown storage type", () => {
    expect(() => buildStorageArgs({ action: "get", storageType: "bad" })).toThrow(
      "Unknown storage type",
    );
  });

  it("throws unknown action", () => {
    expect(() => buildStorageArgs({ action: "bad" })).toThrow("Unknown storage action");
  });

  it("includes session prefix", () => {
    expect(buildStorageArgs({ action: "clear", session: "admin" })).toEqual([
      "-s", "admin", "localstorage-clear",
    ]);
  });
});

describe("buildFillFormArgs", () => {
  it("builds fill argv with -- separator", () => {
    expect(buildFillFormArgs({ selector: "#name", text: "John" })).toEqual([
      "fill", "--", "#name", "John",
    ]);
  });

  it("includes --submit flag before -- separator", () => {
    expect(buildFillFormArgs({ selector: "#s", text: "q", submit: true })).toEqual([
      "fill", "--submit", "--", "#s", "q",
    ]);
  });

  it("includes session prefix", () => {
    expect(buildFillFormArgs({ selector: "#x", text: "hi", session: "admin" })).toEqual([
      "-s", "admin", "fill", "--", "#x", "hi",
    ]);
  });
});

describe("press/select/hover builders", () => {
  it("buildPressArgs puts the key after --", () => {
    expect(buildPressArgs({ key: "Enter" })).toEqual(["press", "--", "Enter"]);
    expect(buildPressArgs({ key: "Tab", session: "a" })).toEqual(["-s", "a", "press", "--", "Tab"]);
  });

  it("buildSelectArgs passes the selector then every value", () => {
    expect(buildSelectArgs({ selector: "s1e2", values: ["red", "blue"] })).toEqual([
      "select", "--", "s1e2", "red", "blue",
    ]);
    expect(buildSelectArgs({ selector: "#c", values: ["x"], session: "a" })).toEqual([
      "-s", "a", "select", "--", "#c", "x",
    ]);
  });

  it("buildHoverArgs puts the selector after --", () => {
    expect(buildHoverArgs({ selector: "s1e9" })).toEqual(["hover", "--", "s1e9"]);
    expect(buildHoverArgs({ selector: "s1e9", session: "a" })).toEqual(["-s", "a", "hover", "--", "s1e9"]);
  });
});

// gap-plan ext:EXT-1: a model value must never become a flag or command of its own.
describe("argv builders (property)", () => {
  const LITERALS = new Set([
    "goto", "snapshot", "-i", "click", "dblclick", "type", "eval",
    "screenshot", "--full-page", "--out", SCREENSHOT_OUT, "mousewheel",
    "tab-list", "tab-new", "tab-close", "tab-select", "reload", "go-back", "go-forward",
    "cookie-get", "cookie-set", "cookie-delete", "cookie-clear",
    ...["localstorage", "sessionstorage"].flatMap((p) => ["get", "set", "delete", "clear"].map((a) => `${p}-${a}`)),
    "fill", "--submit", "press", "select", "hover",
  ]);
  const OWN_FLAG = /^--selector=/; // screenshot's only model-valued flag

  it("every token before the first -- is a builder literal or a single --<own-flag>=<v>", () => {
    fc.assert(
      fc.property(builderCall(fc.option(modelString, { nil: undefined })), ({ model, argv }) => {
        let tokens = argv;
        if (model.session) {
          expect(tokens.slice(0, 2)).toEqual(["-s", model.session]);
          tokens = tokens.slice(2);
        }
        const end = tokens.indexOf("--");
        for (const tok of end < 0 ? tokens : tokens.slice(0, end)) {
          expect(LITERALS.has(tok) || OWN_FLAG.test(tok), `stray token ${JSON.stringify(tok)}`).toBe(true);
        }
      }),
    );
  });
});
