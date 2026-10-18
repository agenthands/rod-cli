import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { execRodCli, buildScrollArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseScroll(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_scroll",
    label: "Browse Scroll",
    description:
      'Scroll with real wheel input. Returns what moved: "scrolled dy=800 y=1600/4600"; " end" means nothing more in that direction. Pass selector (ref or CSS) to scroll an inner pane.',
    promptSnippet:
      'Scroll: browse_scroll() (one viewport down), browse_scroll(direction="up", distance=500), or browse_scroll(selector="s1e4")',
    promptGuidelines: [
      "With no distance, browse_scroll scrolls one viewport down; up, left and right need a distance.",
      "Stop scrolling in a direction once the result ends with \" end\".",
    ],
    parameters: Type.Object({
      direction: Type.Optional(
        StringEnum(["up", "down", "left", "right"] as const, {
          description: "Scroll direction (default down)",
        }),
      ),
      distance: Type.Optional(
        Type.Number({
          description: "Scroll distance in pixels; omit for one viewport down",
        }),
      ),
      selector: Type.Optional(
        Type.String({
          description: "Snapshot ref (e.g. \"s1e4\") or CSS selector of an inner scrollable pane; omit to scroll the page",
        }),
      ),
      session: SessionParam,
    }, { additionalProperties: false }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildScrollArgs({
        direction: params.direction,
        distance: params.distance,
        selector: params.selector,
        session: params.session,
      }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
