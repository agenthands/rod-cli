import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseFillForm(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_fill_form",
    label: "Browse Fill Form",
    description:
      "Fill a form field instantly and optionally submit. " +
      "Unlike browse_type (humanized typing), fill is instantaneous.",
    promptSnippet:
      'Fill form: browse_fill_form(selector="#name", text="John") or browse_fill_form(selector="#search", text="query", submit=true)',
    promptGuidelines: [
      "Use browse_fill_form for instant form filling — faster than browse_type.",
      "Use submit=true to press Enter after filling (e.g. for search bars and single-field forms).",
      "For multi-field forms, call browse_fill_form once per field, then browse_click the submit button.",
    ],
    parameters: Type.Object({
      selector: Type.String({
        description: "CSS selector of the input element to fill",
      }),
      text: Type.String({ description: "Text to fill into the element" }),
      submit: Type.Optional(
        Type.Boolean({
          description: "Press Enter after filling to submit the form",
          default: false,
        }),
      ),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      // --submit is a tool-generated flag, so it goes BEFORE the -- separator.
      // User-controlled values (selector, text) go AFTER -- for I6 protection.
      const args = ["fill"];
      if (params.submit) args.push("--submit");
      args.push("--", params.selector, params.text);
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
