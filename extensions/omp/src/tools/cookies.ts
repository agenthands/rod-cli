import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildCookiesArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseCookies(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    action: z.enum(["get", "set", "delete", "clear"]).describe("Cookie action to perform"),
    name: z.string().optional().describe("Cookie name (required for set and delete)"),
    value: z.string().optional().describe("Cookie value (required for set)"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_cookies",
    label: "Browse Cookies",
    description:
      "browse_cookies: Manage browser cookies — get, set, delete, or clear all cookies. " +
      'Usage: browse_cookies(action="get") or set: browse_cookies(action="set", name="key", value="val"). ' +
      "Use browse_cookies to inspect or manage cookies for the current page. " +
      "For action='set', both name and value are required. " +
      "For action='delete', the name parameter is required.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildCookiesArgs({
        action: params.action as string,
        name: params.name as string | undefined,
        value: params.value as string | undefined,
        session: params.session as string | undefined,
      }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
