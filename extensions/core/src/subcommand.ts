/**
 * Find the index of the first non-flag argument (the subcommand).
 * Skips -s <value> and --session <value> pairs, and any other flag.
 */
export function findSubcommand(args: string[]): number {
  let i = 0;
  while (i < args.length) {
    const arg = args[i];
    if (arg === undefined) break; // safety net for noUncheckedIndexedAccess
    // Skip -s <name> and --session <name> (the value is never the subcommand)
    if ((arg === "-s" || arg === "--session") && i + 1 < args.length) {
      i += 2;
      continue;
    }
    if (arg.startsWith("-")) {
      i++;
      continue;
    }
    return i;
  }
  return -1;
}
