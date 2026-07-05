import { validateInput } from "./validate";
import { timeoutFor } from "./timeouts";

export interface ExecResult {
  stdout: string;
  stderr: string;
  /** Exit code. Always 0 — non-zero exits cause execRodCli to throw. */
  code: number;
}

export interface ExecOptions {
  signal?: AbortSignal;
  timeout?: number;
}

export type ExecFn = (args: string[], opts?: ExecOptions) => Promise<ExecResult>;

/**
 * Execute rod-cli with the given arguments via the injected ExecFn.
 *
 * - Validates inputs before execution.
 * - Applies per-command timeouts (see TIMEOUTS table).
 * - Prepends `--raw` (harmless: a no-op for every command this extension
 *   issues — goto/eval/click/etc. all ignore it identically to bare output;
 *   it only changes anything for `install`/`stealth-check`, which this
 *   extension never calls. Kept for forward-compat should either land in
 *   the typed tool surface later, not because it currently does anything).
 * - Throws on non-zero exit status.
 * - Propagates AbortSignal.
 */
export async function execRodCli(
  execFn: ExecFn,
  args: string[],
  opts?: ExecOptions,
): Promise<ExecResult> {
  validateInput(args);

  const timeout = timeoutFor(args);
  const result = await execFn(["--raw", ...args], {
    signal: opts?.signal,
    timeout,
  });

  if (result.code !== 0) {
    const stderr = result.stderr?.trim() || "(no stderr)";
    throw new Error(`rod-cli exited ${result.code}: ${stderr}`);
  }

  return { stdout: result.stdout, stderr: result.stderr, code: result.code };
}
