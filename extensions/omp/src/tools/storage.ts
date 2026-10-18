import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildStorageArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseStorage(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    action: z.enum(["get", "set", "delete", "clear"]).describe("Storage action to perform"),
    storageType: z.enum(["local", "session"]).optional().default("local").describe("Storage type"),
    key: z.string().optional().describe("Storage key (required for set and delete)"),
    value: z.string().optional().describe("Storage value (required for set)"),
    session: z.string().optional().describe(SESSION_DESC),
  }).strict();
  pi.registerTool({
    name: "browse_storage",
    label: "Browse Storage",
    description:
      "browse_storage: Manage browser web storage (localStorage / sessionStorage) — get, set, delete, or clear. " +
      'Usage: browse_storage(action="get") or browse_storage(action="set", key="k", value="v", storageType="local"). ' +
      "Use browse_storage to inspect or manage web storage data. " +
      "Prefer localStorage (default) for persistent data, sessionStorage for session-scoped data. " +
      "For action='set', both key and value are required.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildStorageArgs({
        action: params.action as string,
        storageType: params.storageType as string | undefined,
        key: params.key as string | undefined,
        value: params.value as string | undefined,
        session: params.session as string | undefined,
      }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
