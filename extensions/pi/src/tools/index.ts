import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
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

export function registerAllCoreTools(pi: ExtensionAPI) {
  registerBrowseGoto(pi);
  registerBrowseSnapshot(pi);
  registerBrowseClick(pi);
  registerBrowseType(pi);
  registerBrowseEval(pi);
  registerBrowseScreenshot(pi);
  registerBrowseWait(pi);
}

export function registerAllExtendedTools(pi: ExtensionAPI) {
  registerBrowseTabs(pi);
  registerBrowseNavigate(pi);
  registerBrowseScroll(pi);
  registerBrowseCookies(pi);
  registerBrowseStorage(pi);
  registerBrowseFillForm(pi);
}
