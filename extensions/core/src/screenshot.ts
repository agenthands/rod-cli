import { randomUUID } from "node:crypto";
import { readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { execRodCli, type ExecFn } from "./exec";
import { buildScreenshotArgs } from "./argv";

export type ScreenshotContent = [
  { type: "text"; text: string },
  { type: "image"; data: string; mimeType: "image/png" },
];

/**
 * Take a screenshot into a private temp file and return rod-cli's text plus the
 * PNG as base64 image content, so the model sees the image itself. The temp
 * file is always removed.
 */
export async function captureScreenshot(
  execFn: ExecFn,
  p: { selector?: string; fullPage?: boolean; session?: string },
  signal?: AbortSignal,
): Promise<ScreenshotContent> {
  const out = join(tmpdir(), `rod-cli-${randomUUID()}.png`);
  try {
    const result = await execRodCli(execFn, buildScreenshotArgs({ ...p, out }), { signal });
    const data = (await readFile(out)).toString("base64");
    return [
      { type: "text", text: result.stdout },
      { type: "image", data, mimeType: "image/png" },
    ];
  } finally {
    await rm(out, { force: true });
  }
}
