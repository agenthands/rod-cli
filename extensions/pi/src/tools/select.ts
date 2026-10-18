import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli, buildSelectArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseSelect(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_select",
    label: "Browse Select",
    description:
      "Choose options in a <select> element. Each value matches an option's visible text, " +
      "then its value attribute, then a substring of its text.",
    promptSnippet:
      'Select: browse_select(selector="s1e7", values=["Canada"]) or browse_select(selector="#tags", values=["red", "blue"])',
    promptGuidelines: [
      "Use browse_select for <select> dropdowns instead of clicking options or setting .value with browse_eval.",
      "Pass several values only for a multi-select.",
    ],
    parameters: Type.Object({
      selector: Type.String({
        description:
          "CSS selector (e.g. \"#country\") or a snapshot ref copied from browse_snapshot's [ref=sNeM] output (e.g. \"s1e7\")",
      }),
      values: Type.Array(Type.String(), {
        description: "Options to select, by visible text or value",
        minItems: 1,
      }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildSelectArgs({ selector: params.selector, values: params.values, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
