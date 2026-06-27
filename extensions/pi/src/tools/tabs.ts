import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseTabs(pi: ExtensionAPI) {
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
    }),
    async execute(_toolCallId, params, signal) {
      let args: string[];

      switch (params.action) {
        case "list":
          args = ["tab-list"];
          break;
        case "new": {
          if (!params.url)
            throw new Error("action 'new' requires a 'url' parameter");
          args = ["tab-new", "--", params.url];
          break;
        }
        case "close": {
          if (params.index === undefined)
            throw new Error("action 'close' requires an 'index' parameter");
          args = ["tab-close", "--", String(params.index)];
          break;
        }
        case "select": {
          if (params.index === undefined)
            throw new Error("action 'select' requires an 'index' parameter");
          args = ["tab-select", "--", String(params.index)];
          break;
        }
        default:
          throw new Error(`Unknown tab action: ${params.action}`);
      }

      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
