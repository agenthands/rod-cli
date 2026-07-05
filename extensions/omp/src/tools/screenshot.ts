import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildScreenshotArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseScreenshot(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().optional().describe("CSS selector to screenshot a specific element"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_screenshot",
    label: "Browse Screenshot",
    description:
      "browse_screenshot: Capture a screenshot of the current page or a specific element. " +
      'Usage: browse_screenshot() or browse_screenshot(selector="#chart"). ' +
      "Use browse_screenshot for visual verification of page state. " +
      "Prefer browse_snapshot for reading text content -- screenshots are larger and slower.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildScreenshotArgs({ selector: params.selector as string | undefined, session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
