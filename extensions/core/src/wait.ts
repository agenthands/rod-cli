import { execRodCli, type ExecFn } from "./exec";

/**
 * Sleep helper that respects AbortSignal — resolves after `ms` milliseconds
 * or rejects immediately when the signal is aborted.
 */
export function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    if (signal?.aborted) return reject(signal.reason);
    const onAbort = () => {
      clearTimeout(id);
      reject(signal!.reason);
    };
    const id = setTimeout(() => {
      signal?.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    signal?.addEventListener("abort", onAbort, { once: true });
  });
}

/**
 * Build the argv for polling a CSS selector via rod-cli eval.
 */
export function buildWaitPollArgs(p: {
  selector: string;
  session?: string;
}): string[] {
  const pollScript = JSON.stringify(p.selector);
  const args = ["eval", "--", `!!document.querySelector(${pollScript})`];
  if (p.session) args.unshift("-s", p.session);
  return args;
}

/**
 * Poll for a CSS selector using rod-cli eval since no dedicated wait command
 * exists. Evaluates `document.querySelector(sel) !== null` on a 500ms interval.
 * Times out after `timeoutMs` with an error; `timeoutMs <= 0` checks once.
 * Exec failures propagate without retry. Respects session and AbortSignal.
 */
export async function waitForSelector(
  execFn: ExecFn,
  p: { selector: string; timeoutMs: number; session?: string },
  opts?: { signal?: AbortSignal },
): Promise<void> {
  const deadline = Date.now() + p.timeoutMs;
  // Always check at least once (timeoutMs <= 0 checks once and never polls).
  // An exec rejection propagates; it is not retried.
  for (;;) {
    const r = await execRodCli(execFn, buildWaitPollArgs({ selector: p.selector, session: p.session }), { signal: opts?.signal });
    if (r.stdout.trim() === "true") return;
    if (Date.now() >= deadline) break;
    await sleep(500, opts?.signal);
  }
  throw new Error(
    `Timed out waiting ${p.timeoutMs}ms for selector: ${p.selector}`,
  );
}
