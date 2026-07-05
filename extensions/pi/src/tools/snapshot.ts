import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli, buildSnapshotArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseSnapshot(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_snapshot",
    label: "Browse Snapshot",
    description:
      "Capture a token-efficient accessibility-tree markdown snapshot of the current page. " +
      "Output is truncated to 50KB/2000 lines.",
    promptSnippet:
      "Get page content: browse_snapshot()",
    promptGuidelines: [
      "Prefer browse_snapshot over browse_screenshot for reading page structure and text content.",
      "Do NOT call browse_snapshot after every action to confirm results -- use it when you need to read the page.",
      "The snapshot captures the full page; element-scoped snapshots are not yet supported.",
    ],
    parameters: Type.Object({
      selector: Type.Optional(
        Type.String({
          description:
            "CSS selector to scope snapshot (not yet supported -- captures full page)",
        }),
      ),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildSnapshotArgs({ selector: params.selector, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
