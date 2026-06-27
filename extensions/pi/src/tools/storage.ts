import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

const STORAGE_PREFIX: Record<string, string> = {
  local: "localstorage",
  session: "sessionstorage",
};

export function registerBrowseStorage(pi: ExtensionAPI) {
  pi.registerTool({
    name: "browse_storage",
    label: "Browse Storage",
    description:
      "Manage browser web storage (localStorage / sessionStorage) — get, set, delete, or clear.",
    promptSnippet:
      'Get storage: browse_storage(action="get") or browse_storage(action="set", key="k", value="v", storageType="local")',
    promptGuidelines: [
      "Use browse_storage to inspect or manage web storage data.",
      "Prefer localStorage (default) for persistent data, sessionStorage for session-scoped data.",
      "For action='set', both key and value are required.",
    ],
    parameters: Type.Object({
      action: StringEnum(["get", "set", "delete", "clear"] as const, {
        description: "Storage action to perform",
      }),
      storageType: Type.Optional(
        StringEnum(["local", "session"] as const, {
          description: "Storage type",
          default: "local",
        }),
      ),
      key: Type.Optional(
        Type.String({ description: "Storage key (required for set and delete)" }),
      ),
      value: Type.Optional(
        Type.String({ description: "Storage value (required for set)" }),
      ),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const storageType = params.storageType ?? "local";
      const prefix = STORAGE_PREFIX[storageType];
      if (!prefix) throw new Error(`Unknown storage type: ${storageType}`);

      let args: string[];

      switch (params.action) {
        case "get":
          args = params.key
            ? [`${prefix}-get`, "--", params.key]
            : [`${prefix}-get`];
          break;
        case "set":
          if (!params.key) throw new Error("action 'set' requires a 'key' parameter");
          if (!params.value)
            throw new Error("action 'set' requires a 'value' parameter");
          args = [`${prefix}-set`, "--", params.key, params.value];
          break;
        case "delete":
          if (!params.key) throw new Error("action 'delete' requires a 'key' parameter");
          args = [`${prefix}-delete`, "--", params.key];
          break;
        case "clear":
          args = [`${prefix}-clear`];
          break;
        default:
          throw new Error(`Unknown storage action: ${params.action}`);
      }

      if (params.session) args.unshift("-s", params.session);
      const result = await execRodCli(args, { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
