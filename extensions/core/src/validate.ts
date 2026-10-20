import { findSubcommand } from "./subcommand";

const SESSION_NAME = /^[A-Za-z0-9._-]{1,64}$/;

/** Reject a session name that could escape TMPDIR in the daemon's .port/.lock paths. */
function validateSession(name: string): void {
  if (!SESSION_NAME.test(name) || name === "." || name === "..") {
    throw new Error(
      `invalid session name ${JSON.stringify(name)}: use letters, digits, '.', '_' or '-' (max 64)`,
    );
  }
}

export function validateInput(args: string[]): void {
  const cmdIndex = findSubcommand(args);
  const globals = cmdIndex < 0 ? args : args.slice(0, cmdIndex);
  for (let i = 0; i < globals.length; i++) {
    const a = globals[i]!;
    if (a === "-s" || a === "--session") validateSession(globals[i + 1] ?? "");
    else if (a.startsWith("-s=") || a.startsWith("--session=")) validateSession(a.slice(a.indexOf("=") + 1));
  }
  if (cmdIndex < 0) return; // no subcommand, e.g. --version
  const cmd = args[cmdIndex];
  let rest = args.slice(cmdIndex + 1);
  // B51-1: skip the -- injection separator so we validate the real argument
  if (rest[0] === "--") rest = rest.slice(1);

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
        `eval expression too long (${expr.length} characters, max 10000)`,
      );
    }
  }
}
