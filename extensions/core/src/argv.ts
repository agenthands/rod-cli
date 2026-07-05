/**
 * Prepend session flags if a session is specified.
 */
export function withSession(session: string | undefined, args: string[]): string[] {
  return session ? ["-s", session, ...args] : args;
}

// ---------------------------------------------------------------------------
// 13 pure argv-builders
// ---------------------------------------------------------------------------

export function buildGotoArgs(p: { url: string; session?: string }): string[] {
  return withSession(p.session, ["goto", "--", p.url]);
}

export function buildSnapshotArgs(p: { selector?: string; session?: string }): string[] {
  // Note: rod-cli snapshot currently has no --selector flag.
  // The selector param is accepted for future compatibility.
  return withSession(p.session, ["snapshot"]);
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
  session?: string;
}): string[] {
  const args = ["screenshot"];
  if (p.selector) args.push("--selector", p.selector);
  return withSession(p.session, args);
}

export function buildScrollArgs(p: {
  direction?: string;
  distance?: number;
  selector?: string;
  session?: string;
}): string[] {
  if (p.selector) {
    throw new Error(
      "browse_scroll does not support element-level scrolling. " +
        "Use browse_eval with element.scrollBy() for element-level scrolling.",
    );
  }
  const direction = p.direction ?? "down";
  const distance = p.distance ?? 300;
  const dy = direction === "down" ? distance : -distance;
  return withSession(p.session, ["mousewheel", "--", "0", String(dy)]);
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
