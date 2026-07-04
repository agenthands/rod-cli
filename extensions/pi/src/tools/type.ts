import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseType(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_type",
    label: "Browse Type",
    description:
      "Type text into an input or textarea with humanized keystroke timing. " +
      "Accepts a CSS selector or a snapshot ref (e.g. \"s1e5\") from browse_snapshot's [ref=...] output.",
    promptSnippet:
      'Type text: browse_type(selector="#input", text="hello world") or by ref: browse_type(selector="s1e5", text="hello world")',
    promptGuidelines: [
      "Use browse_type for typing into text inputs and textareas with human-like timing.",
      "The selector parameter accepts either a CSS selector (works immediately, no snapshot needed) or a ref token copied verbatim from the most recent browse_snapshot's [ref=sNeM] annotation (e.g. \"s1e5\") -- never a bare \"eN\".",
      "Prefer browse_type over browse_eval for entering text -- browse_type dispatches real, trusted, humanized keystrokes, while browse_eval-driven input (e.g. setting el.value directly) produces untrusted synthetic events that defeat rod-cli's stealth/humanization design.",
      "browse_type does NOT submit forms -- use browse_click on the submit button or browse_fill_form (Phase 49).",
      "For filling multiple fields at once, prefer browse_fill_form.",
    ],
    parameters: Type.Object({
      selector: Type.String({
        description:
          "CSS selector (e.g. \"#input\") or a snapshot ref copied from browse_snapshot's [ref=sNeM] output (e.g. \"s1e5\")",
      }),
      text: Type.String({ description: "Text to type into the element" }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["type", "--", params.selector, params.text];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
