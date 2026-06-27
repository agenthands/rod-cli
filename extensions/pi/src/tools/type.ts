import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseType(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_type",
    label: "Browse Type",
    description:
      "Type text into an input or textarea with humanized keystroke timing.",
    promptSnippet:
      'Type text: browse_type(selector="#input", text="hello world")',
    promptGuidelines: [
      "Use browse_type for typing into text inputs and textareas with human-like timing.",
      "browse_type does NOT submit forms -- use browse_click on the submit button or browse_fill_form (Phase 49).",
      "For filling multiple fields at once, prefer browse_fill_form.",
    ],
    parameters: Type.Object({
      selector: Type.String({
        description: "CSS selector of the input or textarea element",
      }),
      text: Type.String({ description: "Text to type into the element" }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["type", params.selector, params.text];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
