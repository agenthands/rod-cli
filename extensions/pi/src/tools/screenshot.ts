import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli, buildScreenshotArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseScreenshot(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_screenshot",
    label: "Browse Screenshot",
    description:
      "Capture a screenshot of the current page or a specific element.",
    promptSnippet:
      "Screenshot: browse_screenshot() or browse_screenshot(selector=\"#chart\")",
    promptGuidelines: [
      "Use browse_screenshot for visual verification of page state.",
      "Prefer browse_snapshot for reading text content -- screenshots are larger and slower.",
    ],
    parameters: Type.Object({
      selector: Type.Optional(
        Type.String({
          description: "CSS selector to screenshot a specific element",
        }),
      ),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildScreenshotArgs({ selector: params.selector, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
