import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { captureScreenshot, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseScreenshot(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().optional().describe("Snapshot ref (e.g. \"s1e4\") or CSS selector of one element to capture"),
    fullPage: z.boolean().optional().default(false).describe("Capture the whole scrollable page instead of the viewport"),
    session: z.string().optional().describe(SESSION_DESC),
  }).strict();
  pi.registerTool({
    name: "browse_screenshot",
    label: "Browse Screenshot",
    description:
      "browse_screenshot: Capture a PNG of the viewport, the whole page, or one element, and return the image itself. " +
      'Usage: browse_screenshot(), browse_screenshot(selector="s1e4") or browse_screenshot(fullPage=true). ' +
      "Use browse_screenshot for visual verification of page state. " +
      "Prefer browse_snapshot for reading text content -- screenshots are larger and slower.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const content = await captureScreenshot(execFn, {
        selector: params.selector as string | undefined,
        fullPage: params.fullPage as boolean | undefined,
        session: params.session as string | undefined,
      }, signal);
      return { content, details: {} };
    },
  } as never);
}
