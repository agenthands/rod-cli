import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildSnapshotArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseSnapshot(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().optional().describe("Snapshot ref (e.g. s1e3) or CSS selector; only that subtree is captured"),
    interactive: z.boolean().optional().describe("Keep only interactive elements and their ancestors"),
    session: z.string().optional().describe(SESSION_DESC),
  }).strict();
  pi.registerTool({
    name: "browse_snapshot",
    label: "Browse Snapshot",
    description:
      "browse_snapshot: Capture a token-efficient accessibility-tree markdown snapshot of the current page, " +
      "or of the subtree under a ref/CSS selector; interactive=true keeps only controls. " +
      "Output is capped at 50KB; an oversized result ends with a [truncated: …] line saying how to narrow it. " +
      'Usage: browse_snapshot(). ' +
      "Prefer browse_snapshot over browse_screenshot for reading page structure and text content. " +
      "Do NOT call browse_snapshot after every action to confirm results -- use it when you need to read the page. " +
      "To read less, pass interactive=true (controls only) or selector=<ref|css> (one subtree).",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildSnapshotArgs({ selector: params.selector as string | undefined, interactive: params.interactive as boolean | undefined, session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
