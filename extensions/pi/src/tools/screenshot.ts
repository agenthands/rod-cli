import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { captureScreenshot, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseScreenshot(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_screenshot",
    label: "Browse Screenshot",
    description:
      "Capture a PNG of the viewport, the whole page, or one element, and return the image itself.",
    promptSnippet:
      "Screenshot: browse_screenshot(), browse_screenshot(selector=\"s1e4\") or browse_screenshot(fullPage=true)",
    promptGuidelines: [
      "Use browse_screenshot for visual verification of page state.",
      "Prefer browse_snapshot for reading text content -- screenshots are larger and slower.",
    ],
    parameters: Type.Object({
      selector: Type.Optional(
        Type.String({
          description: "Snapshot ref (e.g. \"s1e4\") or CSS selector of one element to capture",
        }),
      ),
      fullPage: Type.Optional(
        Type.Boolean({
          description: "Capture the whole scrollable page instead of the viewport",
          default: false,
        }),
      ),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const content = await captureScreenshot(execFn, { selector: params.selector, fullPage: params.fullPage, session: params.session }, signal);
      return { content, details: {} };
    },
  });
}
