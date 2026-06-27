import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseScroll(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_scroll",
    label: "Browse Scroll",
    description:
      "Scroll the page viewport using the mouse wheel. " +
      "Use browse_eval for element-level scrolling.",
    promptSnippet:
      'Scroll: browse_scroll(direction="down", distance=500) or browse_scroll(direction="up")',
    promptGuidelines: [
      "Use browse_scroll to scroll the page viewport up or down.",
      "The scroll is viewport-level via mousewheel -- for element-level scrolling, use browse_eval with element.scrollBy().",
      "Default distance is 300px. Increase for longer scrolls.",
    ],
    parameters: Type.Object({
      direction: Type.Optional(
        StringEnum(["down", "up"] as const, {
          description: "Scroll direction",
          default: "down",
        }),
      ),
      distance: Type.Optional(
        Type.Number({
          description: "Scroll distance in pixels",
          default: 300,
        }),
      ),
      selector: Type.Optional(
        Type.String({
          description:
            "CSS selector (not yet supported -- use browse_eval for element scrolling)",
        }),
      ),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      if (params.selector) {
        throw new Error(
          "browse_scroll does not support element-level scrolling. " +
            "Use browse_eval with element.scrollBy() for element-level scrolling.",
        );
      }

      const direction = params.direction ?? "down";
      const distance = params.distance ?? 300;
      const dy = direction === "down" ? distance : -distance;

      const args = ["mousewheel", "--", "0", String(dy)];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
