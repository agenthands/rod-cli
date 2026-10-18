import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildNavigateArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseNavigate(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    action: z.enum(["reload", "back", "forward"]).describe("Navigation action to perform"),
    session: z.string().optional().describe(SESSION_DESC),
  }).strict();
  pi.registerTool({
    name: "browse_navigate",
    label: "Browse Navigate",
    description:
      "browse_navigate: Navigate the page history — reload, go back, or go forward. " +
      'Usage: browse_navigate(action="reload") or go back: browse_navigate(action="back"). ' +
      "Use browse_navigate for standard browser navigation actions. " +
      "Prefer browse_goto for explicit URL navigation. " +
      "Use action='back' and action='forward' for history traversal.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildNavigateArgs({ action: params.action as string, session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
