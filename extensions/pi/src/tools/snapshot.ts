import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli, buildSnapshotArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseSnapshot(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_snapshot",
    label: "Browse Snapshot",
    description:
      "Capture a token-efficient accessibility-tree markdown snapshot of the current page, " +
      "or of the subtree under a ref/CSS selector; interactive=true keeps only controls. " +
      "Output is capped at 50KB; an oversized result ends with a [truncated: …] line saying how to narrow it.",
    promptSnippet:
      "Get page content: browse_snapshot()",
    promptGuidelines: [
      "Prefer browse_snapshot over browse_screenshot for reading page structure and text content.",
      "Do NOT call browse_snapshot after every action to confirm results -- use it when you need to read the page.",
      "To read less, pass interactive=true (controls only) or selector=<ref|css> (one subtree).",
    ],
    parameters: Type.Object({
      selector: Type.Optional(
        Type.String({
          description: "Snapshot ref (e.g. s1e3) or CSS selector; only that subtree is captured",
        }),
      ),
      interactive: Type.Optional(
        Type.Boolean({ description: "Keep only interactive elements and their ancestors" }),
      ),
      session: SessionParam,
    }, { additionalProperties: false }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildSnapshotArgs({ selector: params.selector, interactive: params.interactive, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
