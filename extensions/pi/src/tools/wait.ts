import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { waitForSelector, sleep, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowseWait(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_wait",
    label: "Browse Wait",
    description:
      "Wait for a CSS selector to appear on the page or for a fixed duration.",
    promptSnippet:
      'Wait: browse_wait(selector="#loaded") or browse_wait(timeout=3000)',
    promptGuidelines: [
      "Use browse_wait to wait for async page content to load before taking a snapshot.",
      "Prefer waiting for a specific selector over a fixed timeout -- it's more reliable.",
      "Use browse_wait after browse_goto to ensure the page has loaded.",
    ],
    parameters: Type.Object({
      selector: Type.Optional(
        Type.String({
          description: "CSS selector to wait for",
        }),
      ),
      timeout: Type.Optional(
        Type.Number({
          description: "Maximum time to wait in milliseconds",
        }),
      ),
      session: SessionParam,
    }, { additionalProperties: false }),
    async execute(_toolCallId, params, signal) {
      const timeoutMs = params.timeout ?? 30_000;

      if (params.selector) {
        await waitForSelector(execFn, {
          selector: params.selector,
          timeoutMs,
          session: params.session,
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
  });
}
