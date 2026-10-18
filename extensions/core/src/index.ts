// Binary discovery
export { findRodCli } from "./binary";

// Exec contract
export { execRodCli } from "./exec";
export type { ExecFn, ExecResult, ExecOptions } from "./exec";

// Wait orchestration
export { buildWaitPollArgs, waitForSelector, sleep } from "./wait";

// Argv builders
export {
  withSession,
  buildGotoArgs,
  buildSnapshotArgs,
  buildClickArgs,
  buildTypeArgs,
  buildEvalArgs,
  buildScreenshotArgs,
  buildScrollArgs,
  buildTabsArgs,
  buildNavigateArgs,
  buildCookiesArgs,
  buildStorageArgs,
  buildFillFormArgs,
  buildPressArgs,
  buildSelectArgs,
  buildHoverArgs,
} from "./argv";

// Testable helpers
export { validateInput } from "./validate";
export { timeoutFor, TIMEOUTS } from "./timeouts";
export { findSubcommand } from "./subcommand";
