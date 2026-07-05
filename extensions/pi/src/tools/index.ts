import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import type { ExecFn } from "@agenthands/rod-cli-core";
import { registerBrowseGoto } from "./goto";
import { registerBrowseSnapshot } from "./snapshot";
import { registerBrowseClick } from "./click";
import { registerBrowseType } from "./type";
import { registerBrowseEval } from "./eval";
import { registerBrowseScreenshot } from "./screenshot";
import { registerBrowseWait } from "./wait";
import { registerBrowseTabs } from "./tabs";
import { registerBrowseNavigate } from "./navigate";
import { registerBrowseScroll } from "./scroll";
import { registerBrowseCookies } from "./cookies";
import { registerBrowseStorage } from "./storage";
import { registerBrowseFillForm } from "./fill_form";

export function registerAllCoreTools(pi: ExtensionAPI, execFn: ExecFn) {
  registerBrowseGoto(pi, execFn);
  registerBrowseSnapshot(pi, execFn);
  registerBrowseClick(pi, execFn);
  registerBrowseType(pi, execFn);
  registerBrowseEval(pi, execFn);
  registerBrowseScreenshot(pi, execFn);
  registerBrowseWait(pi, execFn);
}

export function registerAllExtendedTools(pi: ExtensionAPI, execFn: ExecFn) {
  registerBrowseTabs(pi, execFn);
  registerBrowseNavigate(pi, execFn);
  registerBrowseScroll(pi, execFn);
  registerBrowseCookies(pi, execFn);
  registerBrowseStorage(pi, execFn);
  registerBrowseFillForm(pi, execFn);
}
