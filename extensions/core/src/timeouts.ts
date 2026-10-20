import { findSubcommand } from "./subcommand";

export const TIMEOUTS: Record<string, number> = {
  goto: 60_000,
  snapshot: 15_000,
  click: 15_000,
  fill: 15_000,
  type: 15_000,
  eval: 15_000,
  screenshot: 30_000,
  wait: 30_000,
  close: 5_000,
  "--version": 5_000,
};
export const DEFAULT_TIMEOUT = 30_000;

export function timeoutFor(args: string[]): number {
  // Find the first non-flag argument -- that's the subcommand.
  const cmdIndex = findSubcommand(args);
  if (cmdIndex >= 0) return TIMEOUTS[args[cmdIndex]!] ?? DEFAULT_TIMEOUT;
  // Flags-only call (e.g. `--version`): fall back to scanning TIMEOUTS keys.
  // Never scan when there is a subcommand: user values and session names could match a key.
  for (const [cmd, ms] of Object.entries(TIMEOUTS)) {
    if (args.includes(cmd)) return ms;
  }
  return DEFAULT_TIMEOUT;
}
