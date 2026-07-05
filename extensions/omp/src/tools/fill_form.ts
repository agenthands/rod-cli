import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildFillFormArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseFillForm(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().describe("CSS selector (e.g. \"#name\") or a snapshot ref copied from browse_snapshot's [ref=sNeM] output (e.g. \"s1e5\")"),
    text: z.string().describe("Text to fill into the element"),
    submit: z.boolean().optional().default(false).describe("Press Enter after filling to submit the form"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_fill_form",
    label: "Browse Fill Form",
    description:
      "browse_fill_form: Fill a form field instantly and optionally submit. " +
      "Unlike browse_type (humanized typing), fill is instantaneous. " +
      "Accepts a CSS selector or a snapshot ref (e.g. \"s1e5\") from browse_snapshot's [ref=...] output. " +
      'Usage: browse_fill_form(selector="#name", text="John") or by ref: browse_fill_form(selector="s1e5", text="John") or browse_fill_form(selector="#search", text="query", submit=true). ' +
      "Use browse_fill_form for instant form filling — faster than browse_type. " +
      "The selector parameter accepts either a CSS selector (works immediately, no snapshot needed) or a ref token copied verbatim from the most recent browse_snapshot's [ref=sNeM] annotation (e.g. \"s1e5\") -- never a bare \"eN\". " +
      "Prefer browse_fill_form over browse_eval for setting field values -- browse_fill_form dispatches real, trusted input, while browse_eval-driven form filling (e.g. setting el.value directly) produces untrusted synthetic events that defeat rod-cli's stealth/humanization design. " +
      "Use submit=true to press Enter after filling (e.g. for search bars and single-field forms). " +
      "For multi-field forms, call browse_fill_form once per field, then browse_click the submit button.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildFillFormArgs({
        selector: params.selector as string,
        text: params.text as string,
        submit: params.submit as boolean | undefined,
        session: params.session as string | undefined,
      }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
