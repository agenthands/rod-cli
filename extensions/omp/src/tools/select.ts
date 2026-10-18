import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildSelectArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseSelect(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().describe("CSS selector (e.g. \"#country\") or a snapshot ref copied from browse_snapshot's [ref=sNeM] output (e.g. \"s1e7\")"),
    values: z.array(z.string()).min(1).describe("Options to select, by visible text or value"),
    session: z.string().optional().describe(SESSION_DESC),
  }).strict();
  pi.registerTool({
    name: "browse_select",
    label: "Browse Select",
    description:
      "browse_select: Choose options in a <select> element. Each value matches an option's visible text, " +
      "then its value attribute, then a substring of its text. " +
      'Usage: browse_select(selector="s1e7", values=["Canada"]) or browse_select(selector="#tags", values=["red", "blue"]). ' +
      "Use browse_select for <select> dropdowns instead of clicking options or setting .value with browse_eval. " +
      "Pass several values only for a multi-select.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildSelectArgs({ selector: params.selector as string, values: params.values as string[], session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
