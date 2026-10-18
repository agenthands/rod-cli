import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildScrollArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseScroll(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    direction: z.enum(["up", "down", "left", "right"]).optional().describe("Scroll direction (default down)"),
    distance: z.number().optional().describe("Scroll distance in pixels; omit for one viewport down"),
    selector: z.string().optional().describe("Snapshot ref (e.g. \"s1e4\") or CSS selector of an inner scrollable pane; omit to scroll the page"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_scroll",
    label: "Browse Scroll",
    description:
      'browse_scroll: Scroll with real wheel input. Returns what moved: "scrolled dy=800 y=1600/4600"; " end" means nothing more in that direction. Pass selector (ref or CSS) to scroll an inner pane. ' +
      'Usage: browse_scroll() (one viewport down), browse_scroll(direction="up", distance=500), or browse_scroll(selector="s1e4"). ' +
      "With no distance, browse_scroll scrolls one viewport down; up, left and right need a distance. " +
      "Stop scrolling in a direction once the result ends with \" end\".",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildScrollArgs({
        direction: params.direction as string | undefined,
        distance: params.distance as number | undefined,
        selector: params.selector as string | undefined,
        session: params.session as string | undefined,
      }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
