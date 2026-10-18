import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildHoverArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseHover(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().describe("CSS selector (e.g. \"nav .menu\") or a snapshot ref copied from browse_snapshot's [ref=sNeM] output (e.g. \"s1e3\")"),
    session: z.string().optional().describe(SESSION_DESC),
  }).strict();
  pi.registerTool({
    name: "browse_hover",
    label: "Browse Hover",
    description:
      "browse_hover: Move the pointer over an element with a humanized mouse path, to open menus or tooltips that appear on hover. " +
      'Usage: browse_hover(selector="s1e3") or browse_hover(selector="nav .menu"). ' +
      "Use browse_hover to reveal hover menus and tooltips, then browse_snapshot to read what appeared.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildHoverArgs({ selector: params.selector as string, session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
