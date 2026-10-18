import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli, buildTabsArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseTabs(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_tabs",
    label: "Browse Tabs",
    description: "Manage browser tabs — list, create, close, or switch tabs.",
    promptSnippet:
      'List tabs: browse_tabs(action="list") or browse_tabs(action="new", url="https://example.com")',
    promptGuidelines: [
      "Use browse_tabs for tab management across different actions.",
      "For 'new' action, provide a URL. For 'close' or 'select', provide the tab index.",
      "Tab indices are zero-based and can be discovered with action='list'.",
    ],
    parameters: Type.Object({
      action: StringEnum(["list", "new", "close", "select"] as const, {
        description: "Tab action to perform",
      }),
      url: Type.Optional(
        Type.String({
          description: "URL to open in a new tab (required for action 'new')",
        }),
      ),
      index: Type.Optional(
        Type.Number({
          description:
            "Tab index to close or select (required for actions 'close' and 'select')",
        }),
      ),
      session: SessionParam,
    }, { additionalProperties: false }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildTabsArgs({
        action: params.action,
        url: params.url,
        index: params.index,
        session: params.session,
      }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
