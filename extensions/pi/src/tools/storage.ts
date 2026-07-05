import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli, buildStorageArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseStorage(pi: ExtensionAPI, execFn: ExecFn) {
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
      const result = await execRodCli(execFn, buildStorageArgs({
        action: params.action,
        storageType: params.storageType,
        key: params.key,
        value: params.value,
        session: params.session,
      }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
