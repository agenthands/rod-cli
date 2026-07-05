import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildTypeArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseType(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().describe("CSS selector (e.g. \"#input\") or a snapshot ref copied from browse_snapshot's [ref=sNeM] output (e.g. \"s1e5\")"),
    text: z.string().describe("Text to type into the element"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_type",
    label: "Browse Type",
    description:
      "browse_type: Type text into an input or textarea with humanized keystroke timing. " +
      "Accepts a CSS selector or a snapshot ref (e.g. \"s1e5\") from browse_snapshot's [ref=...] output. " +
      'Usage: browse_type(selector="#input", text="hello world") or by ref: browse_type(selector="s1e5", text="hello world"). ' +
      "Use browse_type for typing into text inputs and textareas with human-like timing. " +
      "The selector parameter accepts either a CSS selector (works immediately, no snapshot needed) or a ref token copied verbatim from the most recent browse_snapshot's [ref=sNeM] annotation (e.g. \"s1e5\") -- never a bare \"eN\". " +
      "Prefer browse_type over browse_eval for entering text -- browse_type dispatches real, trusted, humanized keystrokes, while browse_eval-driven input (e.g. setting el.value directly) produces untrusted synthetic events that defeat rod-cli's stealth/humanization design. " +
      "browse_type does NOT submit forms -- use browse_click on the submit button or browse_fill_form. " +
      "For filling multiple fields at once, prefer browse_fill_form.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildTypeArgs({ selector: params.selector as string, text: params.text as string, session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
