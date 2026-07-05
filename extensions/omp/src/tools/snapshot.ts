import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildSnapshotArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseSnapshot(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().optional().describe("CSS selector to scope snapshot (not yet supported -- captures full page)"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_snapshot",
    label: "Browse Snapshot",
    description:
      "browse_snapshot: Capture a token-efficient accessibility-tree markdown snapshot of the current page. " +
      "Output is truncated to 50KB/2000 lines. " +
      'Usage: browse_snapshot(). ' +
      "Prefer browse_snapshot over browse_screenshot for reading page structure and text content. " +
      "Do NOT call browse_snapshot after every action to confirm results -- use it when you need to read the page. " +
      "The snapshot captures the full page; element-scoped snapshots are not yet supported.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildSnapshotArgs({ selector: params.selector as string | undefined, session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
