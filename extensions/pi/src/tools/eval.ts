import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli, buildEvalArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseEval(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_eval",
    label: "Browse Eval",
    description:
      "Evaluate JavaScript on the page. " +
      "Use as an escape hatch for data extraction when browse_snapshot is insufficient.",
    promptSnippet:
      'Run JS: browse_eval(expression="document.title") or browse_eval(expression="JSON.stringify(window.__data)")',
    promptGuidelines: [
      "Use browse_eval as an escape hatch for reading page state/data (attributes, computed values, JSON extraction) that browse_snapshot cannot provide.",
      "Do NOT use browse_eval to simulate clicks, typing, or form interaction (e.g. el.click(), setting el.value directly, dispatching synthetic events) -- use browse_click, browse_type, or browse_fill_form instead.",
      "Native actions (browse_click/browse_type/browse_fill_form) dispatch real, trusted, humanized CDP input (isTrusted: true); browse_eval-driven interaction produces untrusted synthetic events (isTrusted: false) with no humanization, undermining rod-cli's stealth/detection-avoidance design.",
      "Expressions are capped at 10KB. For larger scripts, break into multiple calls.",
      "Return JSON-serializable values when possible -- the result is returned as text.",
    ],
    parameters: Type.Object({
      expression: Type.String({
        description:
          "JavaScript expression to evaluate on the page (max 10KB)",
      }),
      session: SessionParam,
    }, { additionalProperties: false }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildEvalArgs({ expression: params.expression, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
