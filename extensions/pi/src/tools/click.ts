import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseClick(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_click",
    label: "Browse Click",
    description: "Click an element by CSS selector.",
    promptSnippet:
      'Click an element: browse_click(selector="#submit-btn") or browse_click(selector="a.link", doubleClick=true)',
    promptGuidelines: [
      "Use browse_click for clicking buttons, links, and other interactive elements.",
      "Use the doubleClick parameter for double-click actions (maps to rod-cli dblclick).",
      "After clicking, use browse_snapshot or browse_wait to confirm the page updated.",
    ],
    parameters: Type.Object({
      selector: Type.String({
        description: "CSS selector of the element to click",
      }),
      doubleClick: Type.Optional(
        Type.Boolean({
          description: "Perform a double click instead of single click",
          default: false,
        }),
      ),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      // rod-cli has separate click and dblclick commands (no --double flag)
      const args = [params.doubleClick ? "dblclick" : "click", "--", params.selector];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
