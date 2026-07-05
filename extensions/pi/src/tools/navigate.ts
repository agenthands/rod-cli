import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli, buildNavigateArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseNavigate(pi: ExtensionAPI, execFn: ExecFn) {
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
      const result = await execRodCli(execFn, buildNavigateArgs({ action: params.action, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
