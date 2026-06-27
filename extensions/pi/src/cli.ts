import { statSync } from "node:fs";
import { platform } from "node:os";
import { join } from "node:path";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

const IS_WINDOWS = platform() === "win32";
const BINARY_NAME = IS_WINDOWS ? "rod-cli.exe" : "rod-cli";
const PATH_SEP = IS_WINDOWS ? ";" : ":";

/**
 * Resolve the rod-cli binary path at extension load time.
 *
 * Resolution order:
 *   1. ROD_CLI_PATH environment variable (explicit path)
 *   2. PATH directories (cross-platform: ; on Windows, : elsewhere)
 *   3. $GOBIN / $HOME/go/bin (Go install default; USERPROFILE on Windows)
 *
 * Returns the absolute path to the binary or `null` if not found.
 * Does NOT throw — the caller handles not-found via user notification.
 */
export function findRodCli(): string | null {
  // 1. ROD_CLI_PATH env var
  const envPath = process.env.ROD_CLI_PATH;
  if (envPath) {
    try {
      if (statSync(envPath, { throwIfNoEntry: false })?.isFile()) return envPath;
    } catch {
      /* continue scanning */
    }
  }

  // 2. Scan PATH
  const pathDirs = (process.env.PATH || "").split(PATH_SEP);
  for (const dir of pathDirs) {
    if (!dir) continue;
    const candidate = join(dir, BINARY_NAME);
    try {
      if (statSync(candidate, { throwIfNoEntry: false })?.isFile()) return candidate;
    } catch {
      /* continue */
    }
  }

  // 3. Go install locations
  const home = IS_WINDOWS ? process.env.USERPROFILE : process.env.HOME;
  if (home) {
    const goBin = process.env.GOBIN || join(home, "go", "bin");
    const candidate = join(goBin, BINARY_NAME);
    try {
      if (statSync(candidate, { throwIfNoEntry: false })?.isFile()) return candidate;
    } catch {
      /* continue */
    }
  }

  return null;
}

// Module-level pi reference, set once by the extension factory.
let _pi: ExtensionAPI | null = null;

/** Internal setter called by the entry point. */
export function setPi(pi: ExtensionAPI) {
  _pi = pi;
}

// ---------------------------------------------------------------------------
// Subcommand detection — skips -s <value> and other flags so validation
// and timeout lookup operate on the actual rod-cli subcommand.
// ---------------------------------------------------------------------------

function findSubcommand(args: string[]): number {
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

// ---------------------------------------------------------------------------
// Timeout table
// ---------------------------------------------------------------------------
const TIMEOUTS: Record<string, number> = {
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
const DEFAULT_TIMEOUT = 30_000;

function timeoutFor(args: string[]): number {
  // Find the first non-flag argument -- that's the subcommand.
  const cmdIndex = findSubcommand(args);
  if (cmdIndex >= 0) {
    const cmd = args[cmdIndex];
    if (cmd !== undefined) {
      const timeout = TIMEOUTS[cmd];
      if (timeout !== undefined) return timeout;
    }
  }
  // Flags-only call (e.g. `--version`): fall back to scanning TIMEOUTS keys.
  for (const [cmd, ms] of Object.entries(TIMEOUTS)) {
    if (args.includes(cmd)) return ms;
  }
  return DEFAULT_TIMEOUT;
}

// ---------------------------------------------------------------------------
// Input validation
// ---------------------------------------------------------------------------

function validateInput(args: string[]): void {
  const cmdIndex = findSubcommand(args);
  if (cmdIndex < 0) return; // no subcommand, e.g. --version
  const cmd = args[cmdIndex];
  const rest = args.slice(cmdIndex + 1);

  if (cmd === "goto") {
    const url = rest[0];
    if (!url) throw new Error("goto requires a URL argument");
    if (!url.startsWith("http://") && !url.startsWith("https://")) {
      throw new Error(
        `goto requires an http:// or https:// URL, got: ${url.slice(0, 80)}`,
      );
    }
  }

  if ((cmd === "click" || cmd === "fill" || cmd === "type") && !rest[0]?.trim()) {
    throw new Error(`${cmd} requires a non-empty CSS selector`);
  }

  if (cmd === "fill" && !rest[1]?.trim()) {
    throw new Error("fill requires non-empty text as the second argument");
  }

  if (cmd === "eval") {
    const expr = rest[0];
    if (expr && expr.length > 10_000) {
      throw new Error(
        `eval expression too long (${expr.length} bytes, max 10000)`,
      );
    }
  }
}

// ---------------------------------------------------------------------------
// Shell-out wrapper
// ---------------------------------------------------------------------------

export interface ExecResult {
  stdout: string;
  stderr: string;
  /** Exit code. Always 0 — non-zero exits cause execRodCli to throw. */
  code: number;
}

export interface ExecOptions {
  signal?: AbortSignal;
}

/**
 * Execute rod-cli with the given arguments via pi.exec().
 *
 * - Prepends `--raw` for clean machine-readable output.
 * - Applies per-command timeouts (see TIMEOUTS table).
 * - Validates inputs before execution.
 * - Throws on non-zero exit status.
 * - Propagates AbortSignal.
 */
export async function execRodCli(
  args: string[],
  opts?: ExecOptions,
): Promise<ExecResult> {
  if (!_pi) throw new Error("Extension not initialized — pi reference not set");

  validateInput(args);

  const timeout = timeoutFor(args);
  const result = await _pi.exec("rod-cli", ["--raw", ...args], {
    signal: opts?.signal,
    timeout,
  });

  if (result.code !== 0) {
    const stderr = result.stderr?.trim() || "(no stderr)";
    throw new Error(`rod-cli exited ${result.code}: ${stderr}`);
  }

  return { stdout: result.stdout, stderr: result.stderr, code: result.code };
}
