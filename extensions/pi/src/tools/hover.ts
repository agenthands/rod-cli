import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli, buildHoverArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseHover(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_hover",
    label: "Browse Hover",
    description:
      "Move the pointer over an element with a humanized mouse path, to open menus or tooltips that appear on hover.",
    promptSnippet:
      'Hover: browse_hover(selector="s1e3") or browse_hover(selector="nav .menu")',
    promptGuidelines: [
      "Use browse_hover to reveal hover menus and tooltips, then browse_snapshot to read what appeared.",
    ],
    parameters: Type.Object({
      selector: Type.String({
        description:
          "CSS selector (e.g. \"nav .menu\") or a snapshot ref copied from browse_snapshot's [ref=sNeM] output (e.g. \"s1e3\")",
      }),
      session: SessionParam,
    }, { additionalProperties: false }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildHoverArgs({ selector: params.selector, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
