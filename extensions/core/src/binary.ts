import { statSync } from "node:fs";
import { platform } from "node:os";
import { join } from "node:path";

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
