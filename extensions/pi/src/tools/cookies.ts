import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

export function registerBrowseCookies(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_cookies",
    label: "Browse Cookies",
    description:
      "Manage browser cookies — get, set, delete, or clear all cookies.",
    promptSnippet:
      'Get cookies: browse_cookies(action="get") or set: browse_cookies(action="set", name="key", value="val")',
    promptGuidelines: [
      "Use browse_cookies to inspect or manage cookies for the current page.",
      "For action='set', both name and value are required.",
      "For action='delete', the name parameter is required.",
    ],
    parameters: Type.Object({
      action: StringEnum(["get", "set", "delete", "clear"] as const, {
        description: "Cookie action to perform",
      }),
      name: Type.Optional(
        Type.String({ description: "Cookie name (required for set and delete)" }),
      ),
      value: Type.Optional(
        Type.String({ description: "Cookie value (required for set)" }),
      ),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      let args: string[];

      switch (params.action) {
        case "get":
          args = ["cookie-get"];
          break;
        case "set":
          if (!params.name) throw new Error("action 'set' requires a 'name' parameter");
          if (!params.value) throw new Error("action 'set' requires a 'value' parameter");
          args = ["cookie-set", "--", params.name, params.value];
          break;
        case "delete":
          if (!params.name) throw new Error("action 'delete' requires a 'name' parameter");
          args = ["cookie-delete", "--", params.name];
          break;
        case "clear":
          args = ["cookie-clear"];
          break;
        default:
          throw new Error(`Unknown cookie action: ${params.action}`);
      }

      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
