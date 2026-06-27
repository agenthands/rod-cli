import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli } from "../cli";
import { SessionParam } from "../types";

/**
 * Sleep helper that respects AbortSignal — resolves after `ms` milliseconds
 * or rejects immediately when the signal is aborted.
 */
function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    if (signal?.aborted) return reject(signal.reason);
    const id = setTimeout(resolve, ms);
    if (signal) {
      signal.addEventListener(
        "abort",
        () => {
          clearTimeout(id);
          reject(signal.reason);
        },
        { once: true },
      );
    }
  });
}

/**
 * Poll for a CSS selector using rod-cli eval since no dedicated wait command
 * exists. Evaluates `document.querySelector(sel) !== null` on a 500ms interval.
 * Times out after `timeoutMs` with an error. Respects session and AbortSignal.
 */
async function waitForSelector(
  selector: string,
  timeoutMs: number,
  session: string | undefined,
  signal?: AbortSignal,
): Promise<void> {
  const pollScript = JSON.stringify(selector);
  const deadline = Date.now() + timeoutMs;
  const evalArgs = ["eval", "--", `!!document.querySelector(${pollScript})`];
  if (session) evalArgs.unshift("-s", session);

  while (Date.now() < deadline) {
    const result = await execRodCli(evalArgs, { signal });
    if (result.stdout.trim() === "true") return;
    await sleep(500, signal);
  }

  throw new Error(
    `Timed out waiting ${timeoutMs}ms for selector: ${selector}`,
  );
}

export function registerBrowseWait(pi: ExtensionAPI) {
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
    }),
    async execute(_toolCallId, params, signal) {
      const timeoutMs = params.timeout ?? 30000;

      if (params.selector) {
        await waitForSelector(params.selector, timeoutMs, params.session, signal);
        return {
          content: [
            {
              type: "text",
              text: `Found selector: ${params.selector}`,
            },
          ],
          details: {},
        };
      }

      // No selector -- just wait for the specified duration.
      await sleep(timeoutMs, signal);
      return { content: [{ type: "text", text: `Waited ${timeoutMs}ms` }], details: {} };
    },
  });
}
