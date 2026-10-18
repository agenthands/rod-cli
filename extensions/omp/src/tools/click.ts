import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildClickArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseClick(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().describe("CSS selector (e.g. \"#submit-btn\") or a snapshot ref copied from browse_snapshot's [ref=sNeM] output (e.g. \"s1e5\")"),
    doubleClick: z.boolean().optional().default(false).describe("Perform a double click instead of single click"),
    session: z.string().optional().describe(SESSION_DESC),
  }).strict();
  pi.registerTool({
    name: "browse_click",
    label: "Browse Click",
    description:
      "browse_click: Click an element by CSS selector, or by a snapshot ref (e.g. \"s1e5\") " +
      "copied from browse_snapshot's [ref=...] output. " +
      'Usage: browse_click(selector="#submit-btn") or by ref: browse_click(selector="s1e5") (ref from browse_snapshot) or browse_click(selector="a.link", doubleClick=true). ' +
      "Use browse_click for clicking buttons, links, and other interactive elements. " +
      "The selector parameter accepts either a CSS selector (works immediately, no snapshot needed) or a ref token copied verbatim from the most recent browse_snapshot's [ref=sNeM] annotation (e.g. \"s1e5\") -- never a bare \"eN\". " +
      "Prefer browse_click over browse_eval for clicking -- browse_click dispatches a real, trusted, humanized CDP click, while browse_eval-driven clicks (e.g. el.click()) are untrusted synthetic events that defeat rod-cli's stealth/humanization design. " +
      "Use the doubleClick parameter for double-click actions (maps to rod-cli dblclick). " +
      "After clicking, use browse_snapshot or browse_wait to confirm the page updated.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildClickArgs({ selector: params.selector as string, doubleClick: params.doubleClick as boolean | undefined, session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
