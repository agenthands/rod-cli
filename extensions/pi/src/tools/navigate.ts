import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

const ACTION_TO_COMMAND: Record<string, string> = {
  reload: "reload",
  back: "go-back",
  forward: "go-forward",
};

export function registerBrowseNavigate(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_navigate",
    label: "Browse Navigate",
    description: "Navigate the page history — reload, go back, or go forward.",
    promptSnippet:
      'Reload: browse_navigate(action="reload") or go back: browse_navigate(action="back")',
    promptGuidelines: [
      "Use browse_navigate for standard browser navigation actions.",
      "Prefer browse_goto for explicit URL navigation.",
      "Use action='back' and action='forward' for history traversal.",
    ],
    parameters: Type.Object({
      action: StringEnum(["reload", "back", "forward"] as const, {
        description: "Navigation action to perform",
      }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const cmd = ACTION_TO_COMMAND[params.action];
      if (!cmd) throw new Error(`Unknown navigation action: ${params.action}`);

      const args: string[] = [cmd];
      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
