import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseEval(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_eval",
    label: "Browse Eval",
    description:
      "Evaluate JavaScript on the page. " +
      "Use as an escape hatch for data extraction when browse_snapshot is insufficient.",
    promptSnippet:
      'Run JS: browse_eval(expression="document.title") or browse_eval(expression="JSON.stringify(window.__data)")',
    promptGuidelines: [
      "Use browse_eval as an escape hatch for data extraction that browse_snapshot cannot provide.",
      "Prefer browse_click and browse_type for standard interactions -- do not use browse_eval to click or type.",
      "Expressions are capped at 10KB. For larger scripts, break into multiple calls.",
      "Return JSON-serializable values when possible -- the result is returned as text.",
    ],
    parameters: Type.Object({
      expression: Type.String({
        description:
          "JavaScript expression to evaluate on the page (max 10KB)",
      }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const args = ["eval", params.expression];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
