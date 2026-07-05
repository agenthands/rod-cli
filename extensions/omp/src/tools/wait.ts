import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { waitForSelector, sleep, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowseWait(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    selector: z.string().optional().describe("CSS selector to wait for"),
    timeout: z.number().optional().describe("Maximum time to wait in milliseconds"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_wait",
    label: "Browse Wait",
    description:
      "browse_wait: Wait for a CSS selector to appear on the page or for a fixed duration. " +
      'Usage: browse_wait(selector="#loaded") or browse_wait(timeout=3000). ' +
      "Use browse_wait to wait for async page content to load before taking a snapshot. " +
      "Prefer waiting for a specific selector over a fixed timeout -- it's more reliable. " +
      "Use browse_wait after browse_goto to ensure the page has loaded.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const timeoutMs = (params.timeout as number | undefined) ?? 30_000;

      if (params.selector) {
        await waitForSelector(execFn, {
          selector: params.selector as string,
          timeoutMs,
          session: params.session as string | undefined,
        }, { signal });
        return {
          content: [{ type: "text", text: `Found selector: ${params.selector}` }],
          details: {},
        };
      }

      // No selector -- just wait for the specified duration.
      await sleep(timeoutMs, signal);
      return { content: [{ type: "text", text: `Waited ${timeoutMs}ms` }], details: {} };
    },
  } as never);
}
