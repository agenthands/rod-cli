import { describe, it, expect, vi } from "vitest";
import { existsSync, writeFileSync } from "node:fs";
import { captureScreenshot } from "../screenshot";

const PNG = Buffer.from("not-really-a-png");

/** An execFn that writes the --out file, like rod-cli does, then exits with `code`. */
function writingExec(code: number) {
  let out = "";
  const execFn = vi.fn(async (args: string[]) => {
    out = args[args.indexOf("--out") + 1]!;
    writeFileSync(out, PNG);
    return { code, stdout: "saved", stderr: code ? "boom" : "" };
  });
  return { execFn, outPath: () => out };
}

describe("captureScreenshot", () => {
  it("returns the PNG as base64 and removes the temp file", async () => {
    const { execFn, outPath } = writingExec(0);
    const [text, image] = await captureScreenshot(execFn, {});
    expect(text).toEqual({ type: "text", text: "saved" });
    expect(image).toEqual({ type: "image", data: PNG.toString("base64"), mimeType: "image/png" });
    expect(existsSync(outPath())).toBe(false);
  });

  it("rejects and removes the temp file when rod-cli exits non-zero after writing it", async () => {
    const { execFn, outPath } = writingExec(1);
    await expect(captureScreenshot(execFn, {})).rejects.toThrow("rod-cli exited 1: boom");
    expect(outPath()).not.toBe("");
    expect(existsSync(outPath())).toBe(false);
  });
});
