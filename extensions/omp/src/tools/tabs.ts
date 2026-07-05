import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildTabsArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseTabs(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    action: z.enum(["list", "new", "close", "select"]).describe("Tab action to perform"),
    url: z.string().optional().describe("URL to open in a new tab (required for action 'new')"),
    index: z.number().optional().describe("Tab index to close or select (required for actions 'close' and 'select')"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_tabs",
    label: "Browse Tabs",
    description:
      "browse_tabs: Manage browser tabs — list, create, close, or switch tabs. " +
      'Usage: browse_tabs(action="list") or browse_tabs(action="new", url="https://example.com"). ' +
      "Use browse_tabs for tab management across different actions. " +
      "For 'new' action, provide a URL. For 'close' or 'select', provide the tab index. " +
      "Tab indices are zero-based and can be discovered with action='list'.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildTabsArgs({
        action: params.action as string,
        url: params.url as string | undefined,
        index: params.index as number | undefined,
        session: params.session as string | undefined,
      }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
