import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildScrollArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseScroll(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    direction: z.enum(["down", "up"]).optional().default("down").describe("Scroll direction"),
    distance: z.number().optional().default(300).describe("Scroll distance in pixels"),
    selector: z.string().optional().describe("CSS selector (not yet supported -- use browse_eval for element scrolling)"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_scroll",
    label: "Browse Scroll",
    description:
      "browse_scroll: Scroll the page viewport using the mouse wheel. " +
      "Use browse_eval for element-level scrolling. " +
      'Usage: browse_scroll(direction="down", distance=500) or browse_scroll(direction="up"). ' +
      "Use browse_scroll to scroll the page viewport up or down. " +
      "The scroll is viewport-level via mousewheel -- for element-level scrolling, use browse_eval with element.scrollBy(). " +
      "Default distance is 300px. Increase for longer scrolls.",
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
