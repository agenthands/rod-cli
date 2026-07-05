import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli, buildClickArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseClick(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_click",
    label: "Browse Click",
    description:
      "Click an element by CSS selector, or by a snapshot ref (e.g. \"s1e5\") " +
      "copied from browse_snapshot's [ref=...] output.",
    promptSnippet:
      'Click by selector: browse_click(selector="#submit-btn") or by ref: browse_click(selector="s1e5") (ref from browse_snapshot) or browse_click(selector="a.link", doubleClick=true)',
    promptGuidelines: [
      "Use browse_click for clicking buttons, links, and other interactive elements.",
      "The selector parameter accepts either a CSS selector (works immediately, no snapshot needed) or a ref token copied verbatim from the most recent browse_snapshot's [ref=sNeM] annotation (e.g. \"s1e5\") -- never a bare \"eN\".",
      "Prefer browse_click over browse_eval for clicking -- browse_click dispatches a real, trusted, humanized CDP click, while browse_eval-driven clicks (e.g. el.click()) are untrusted synthetic events that defeat rod-cli's stealth/humanization design.",
      "Use the doubleClick parameter for double-click actions (maps to rod-cli dblclick).",
      "After clicking, use browse_snapshot or browse_wait to confirm the page updated.",
    ],
    parameters: Type.Object({
      selector: Type.String({
        description:
          "CSS selector (e.g. \"#submit-btn\") or a snapshot ref copied from browse_snapshot's [ref=sNeM] output (e.g. \"s1e5\")",
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
      const result = await execRodCli(execFn, buildClickArgs({ selector: params.selector, doubleClick: params.doubleClick, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
