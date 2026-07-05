import { describe, it, expect } from "bun:test";
import { createMockApi } from "./mockapi";
import extension from "../index";

describe("tool argv mapping", () => {
  it("browse_goto produces correct argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_goto")!;
    await tool.execute("id", { url: "https://example.com" });

    expect(mock.execCalls[0]).toEqual(["--raw", "goto", "--", "https://example.com"]);
  });

  it("browse_goto with session produces correct argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_goto")!;
    await tool.execute("id", { url: "https://example.com", session: "admin" });

    expect(mock.execCalls[0]).toEqual(["--raw", "-s", "admin", "goto", "--", "https://example.com"]);
  });

  it("browse_click produces correct argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_click")!;
    await tool.execute("id", { selector: "#btn" });

    expect(mock.execCalls[0]).toEqual(["--raw", "click", "--", "#btn"]);
  });

  it("browse_click with doubleClick produces dblclick argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_click")!;
    await tool.execute("id", { selector: "#btn", doubleClick: true });

    expect(mock.execCalls[0]).toEqual(["--raw", "dblclick", "--", "#btn"]);
  });

  it("browse_type produces correct argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_type")!;
    await tool.execute("id", { selector: "#input", text: "hello" });

    expect(mock.execCalls[0]).toEqual(["--raw", "type", "--", "#input", "hello"]);
  });

  it("browse_eval produces correct argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_eval")!;
    await tool.execute("id", { expression: "document.title" });

    expect(mock.execCalls[0]).toEqual(["--raw", "eval", "--", "document.title"]);
  });

  it("browse_fill_form produces correct argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_fill_form")!;
    await tool.execute("id", { selector: "#name", text: "John" });

    expect(mock.execCalls[0]).toEqual(["--raw", "fill", "--", "#name", "John"]);
  });

  it("browse_fill_form with submit produces --submit flag", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_fill_form")!;
    await tool.execute("id", { selector: "#search", text: "query", submit: true });

    expect(mock.execCalls[0]).toEqual(["--raw", "fill", "--submit", "--", "#search", "query"]);
  });

  it("browse_scroll produces mousewheel argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_scroll")!;
    await tool.execute("id", { direction: "down", distance: 500 });

    expect(mock.execCalls[0]).toEqual(["--raw", "mousewheel", "--", "0", "500"]);
  });

  it("browse_scroll direction=up produces negative distance", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_scroll")!;
    await tool.execute("id", { direction: "up", distance: 300 });

    expect(mock.execCalls[0]).toEqual(["--raw", "mousewheel", "--", "0", "-300"]);
  });

  it("browse_tabs action=list produces tab-list argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_tabs")!;
    await tool.execute("id", { action: "list" });

    expect(mock.execCalls[0]).toEqual(["--raw", "tab-list"]);
  });

  it("browse_tabs action=new with url produces tab-new argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_tabs")!;
    await tool.execute("id", { action: "new", url: "https://example.com" });

    expect(mock.execCalls[0]).toEqual(["--raw", "tab-new", "--", "https://example.com"]);
  });

  it("browse_tabs action=close with index produces tab-close argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_tabs")!;
    await tool.execute("id", { action: "close", index: 0 });

    expect(mock.execCalls[0]).toEqual(["--raw", "tab-close", "--", "0"]);
  });

  it("browse_tabs action=select with index produces tab-select argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_tabs")!;
    await tool.execute("id", { action: "select", index: 1 });

    expect(mock.execCalls[0]).toEqual(["--raw", "tab-select", "--", "1"]);
  });

  it("browse_cookies action=get produces cookie-get argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_cookies")!;
    await tool.execute("id", { action: "get" });

    expect(mock.execCalls[0]).toEqual(["--raw", "cookie-get"]);
  });

  it("browse_cookies action=set produces cookie-set argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_cookies")!;
    await tool.execute("id", { action: "set", name: "key", value: "val" });

    expect(mock.execCalls[0]).toEqual(["--raw", "cookie-set", "--", "key", "val"]);
  });

  it("browse_cookies action=delete produces cookie-delete argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_cookies")!;
    await tool.execute("id", { action: "delete", name: "key" });

    expect(mock.execCalls[0]).toEqual(["--raw", "cookie-delete", "--", "key"]);
  });

  it("browse_cookies action=clear produces cookie-clear argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_cookies")!;
    await tool.execute("id", { action: "clear" });

    expect(mock.execCalls[0]).toEqual(["--raw", "cookie-clear"]);
  });

  it("browse_storage action=get produces localstorage-get argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_storage")!;
    await tool.execute("id", { action: "get" });

    expect(mock.execCalls[0]).toEqual(["--raw", "localstorage-get"]);
  });

  it("browse_storage action=set with key/value produces localstorage-set argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_storage")!;
    await tool.execute("id", { action: "set", key: "k", value: "v" });

    expect(mock.execCalls[0]).toEqual(["--raw", "localstorage-set", "--", "k", "v"]);
  });

  it("browse_storage action=delete produces localstorage-delete argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_storage")!;
    await tool.execute("id", { action: "delete", key: "k" });

    expect(mock.execCalls[0]).toEqual(["--raw", "localstorage-delete", "--", "k"]);
  });

  it("browse_storage action=clear produces localstorage-clear argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_storage")!;
    await tool.execute("id", { action: "clear" });

    expect(mock.execCalls[0]).toEqual(["--raw", "localstorage-clear"]);
  });

  it("browse_storage with sessionStorage type produces sessionstorage-* argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_storage")!;
    await tool.execute("id", { action: "get", storageType: "session" });

    expect(mock.execCalls[0]).toEqual(["--raw", "sessionstorage-get"]);
  });

  it("browse_navigate action=reload produces reload argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_navigate")!;
    await tool.execute("id", { action: "reload" });

    expect(mock.execCalls[0]).toEqual(["--raw", "reload"]);
  });

  it("browse_navigate action=back produces go-back argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_navigate")!;
    await tool.execute("id", { action: "back" });

    expect(mock.execCalls[0]).toEqual(["--raw", "go-back"]);
  });

  it("browse_navigate action=forward produces go-forward argv", async () => {
    const mock = createMockApi();
    extension(mock.api);

    const tool = mock.tools.find((t) => t.name === "browse_navigate")!;
    await tool.execute("id", { action: "forward" });

    expect(mock.execCalls[0]).toEqual(["--raw", "go-forward"]);
  });
});
