import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli, buildGotoArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseGoto(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_goto",
    label: "Browse Goto",
    description:
      "Navigate the browser to a URL. " +
      "The browser daemon starts automatically on first use.",
    promptSnippet:
      'Navigate to a URL: browse_goto(url="https://example.com")',
    promptGuidelines: [
      "Use browse_goto to navigate to any URL before interacting with the page.",
      "The browser starts automatically on the first browse_goto call -- no manual setup needed.",
      "Use the session parameter for multi-tab or multi-identity workflows.",
    ],
    parameters: Type.Object({
      url: Type.String({ description: "Full URL to navigate to (https://...)" }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildGotoArgs({ url: params.url, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
