import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli, buildCookiesArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseCookies(pi: ExtensionAPI, execFn: ExecFn) {
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
    }, { additionalProperties: false }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildCookiesArgs({
        action: params.action,
        name: params.name,
        value: params.value,
        session: params.session,
      }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
