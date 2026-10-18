/**
 * Prepend session flags if a session is specified.
 */
export function withSession(session: string | undefined, args: string[]): string[] {
  return session ? ["-s", session, ...args] : args;
}

// ---------------------------------------------------------------------------
// pure argv-builders
// ---------------------------------------------------------------------------

export function buildGotoArgs(p: { url: string; session?: string }): string[] {
  return withSession(p.session, ["goto", "--", p.url]);
}

export function buildSnapshotArgs(p: { selector?: string; interactive?: boolean; session?: string }): string[] {
  const args = ["snapshot"];
  if (p.interactive) args.push("-i");
  if (p.selector) args.push("--", p.selector);
  return withSession(p.session, args);
}

export function buildClickArgs(p: {
  selector: string;
  doubleClick?: boolean;
  session?: string;
}): string[] {
  // rod-cli has separate click and dblclick commands (no --double flag)
  return withSession(p.session, [
    p.doubleClick ? "dblclick" : "click",
    "--",
    p.selector,
  ]);
}

export function buildTypeArgs(p: {
  selector: string;
  text: string;
  session?: string;
}): string[] {
  return withSession(p.session, ["type", "--", p.selector, p.text]);
}

export function buildEvalArgs(p: {
  expression: string;
  session?: string;
}): string[] {
  return withSession(p.session, ["eval", "--", p.expression]);
}

export function buildScreenshotArgs(p: {
  selector?: string;
  out?: string;
  fullPage?: boolean;
  session?: string;
}): string[] {
  const args = ["screenshot"];
  if (p.selector) args.push("--selector", p.selector);
  if (p.out) args.push("--out", p.out);
  if (p.fullPage) args.push("--full-page");
  return withSession(p.session, args);
}

export function buildScrollArgs(p: {
  direction?: string;
  distance?: number;
  selector?: string;
  session?: string;
}): string[] {
  const direction = p.direction ?? "down";
  const args = ["mousewheel"];
  if (p.distance !== undefined) {
    const d = p.distance;
    const delta: Record<string, [number, number]> = {
      down: [0, d],
      up: [0, -d],
      right: [d, 0],
      left: [-d, 0],
    };
    const xy = delta[direction];
    if (!xy) throw new Error(`Unknown scroll direction: ${direction}`);
    args.push("--", String(xy[0]), String(xy[1]));
  } else if (direction !== "down") {
    // rod-cli's bare mousewheel scrolls one viewport down; other directions need explicit deltas.
    throw new Error(`direction '${direction}' requires a 'distance' parameter`);
  }
  if (p.selector) {
    if (args.length === 1) args.push("--");
    args.push(p.selector);
  }
  return withSession(p.session, args);
}

export function buildTabsArgs(p: {
  action: string;
  url?: string;
  index?: number;
  session?: string;
}): string[] {
  let args: string[];
  switch (p.action) {
    case "list":
      args = ["tab-list"];
      break;
    case "new": {
      if (!p.url) throw new Error("action 'new' requires a 'url' parameter");
      args = ["tab-new", "--", p.url];
      break;
    }
    case "close": {
      if (p.index === undefined)
        throw new Error("action 'close' requires an 'index' parameter");
      args = ["tab-close", "--", String(p.index)];
      break;
    }
    case "select": {
      if (p.index === undefined)
        throw new Error("action 'select' requires an 'index' parameter");
      args = ["tab-select", "--", String(p.index)];
      break;
    }
    default:
      throw new Error(`Unknown tab action: ${p.action}`);
  }
  return withSession(p.session, args);
}

export function buildNavigateArgs(p: {
  action: string;
  session?: string;
}): string[] {
  const cmd =
    p.action === "reload" ? "reload" :
    p.action === "back" ? "go-back" :
    p.action === "forward" ? "go-forward" :
    undefined;
  if (!cmd) throw new Error(`Unknown navigation action: ${p.action}`);
  return withSession(p.session, [cmd]);
}

export function buildCookiesArgs(p: {
  action: string;
  name?: string;
  value?: string;
  session?: string;
}): string[] {
  let args: string[];
  switch (p.action) {
    case "get":
      args = ["cookie-get"];
      break;
    case "set":
      if (!p.name) throw new Error("action 'set' requires a 'name' parameter");
      if (!p.value) throw new Error("action 'set' requires a 'value' parameter");
      args = ["cookie-set", "--", p.name, p.value];
      break;
    case "delete":
      if (!p.name) throw new Error("action 'delete' requires a 'name' parameter");
      args = ["cookie-delete", "--", p.name];
      break;
    case "clear":
      args = ["cookie-clear"];
      break;
    default:
      throw new Error(`Unknown cookie action: ${p.action}`);
  }
  return withSession(p.session, args);
}

export function buildStorageArgs(p: {
  action: string;
  storageType?: string;
  key?: string;
  value?: string;
  session?: string;
}): string[] {
  const STORAGE_PREFIX: Record<string, string> = {
    local: "localstorage",
    session: "sessionstorage",
  };
  const storageType = p.storageType ?? "local";
  const prefix = STORAGE_PREFIX[storageType];
  if (!prefix) throw new Error(`Unknown storage type: ${storageType}`);

  let args: string[];
  switch (p.action) {
    case "get":
      args = p.key ? [`${prefix}-get`, "--", p.key] : [`${prefix}-get`];
      break;
    case "set":
      if (!p.key) throw new Error("action 'set' requires a 'key' parameter");
      if (!p.value) throw new Error("action 'set' requires a 'value' parameter");
      args = [`${prefix}-set`, "--", p.key, p.value];
      break;
    case "delete":
      if (!p.key) throw new Error("action 'delete' requires a 'key' parameter");
      args = [`${prefix}-delete`, "--", p.key];
      break;
    case "clear":
      args = [`${prefix}-clear`];
      break;
    default:
      throw new Error(`Unknown storage action: ${p.action}`);
  }
  return withSession(p.session, args);
}

export function buildFillFormArgs(p: {
  selector: string;
  text: string;
  submit?: boolean;
  session?: string;
}): string[] {
  // --submit is a tool-generated flag, so it goes BEFORE the -- separator.
  // User-controlled values (selector, text) go AFTER -- for I6 protection.
  const args = ["fill"];
  if (p.submit) args.push("--submit");
  args.push("--", p.selector, p.text);
  return withSession(p.session, args);
}

export function buildPressArgs(p: { key: string; session?: string }): string[] {
  return withSession(p.session, ["press", "--", p.key]);
}

export function buildSelectArgs(p: { selector: string; values: string[]; session?: string }): string[] {
  return withSession(p.session, ["select", "--", p.selector, ...p.values]);
}

export function buildHoverArgs(p: { selector: string; session?: string }): string[] {
  return withSession(p.session, ["hover", "--", p.selector]);
}
