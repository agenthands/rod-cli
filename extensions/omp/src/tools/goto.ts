import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildGotoArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseGoto(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    url: z.string().describe("Full URL to navigate to (https://...)"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_goto",
    label: "Browse Goto",
    description:
      "browse_goto: Navigate the browser to a URL. The browser daemon starts automatically on first use. " +
      'Usage: browse_goto(url="https://example.com"). ' +
      "Use browse_goto to navigate to any URL before interacting with the page. " +
      "The browser starts automatically on the first browse_goto call -- no manual setup needed. " +
      "Use the session parameter for multi-tab or multi-identity workflows.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildGotoArgs({ url: params.url as string, session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
