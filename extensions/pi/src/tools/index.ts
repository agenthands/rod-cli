import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { registerBrowseGoto } from "./goto";
import { registerBrowseSnapshot } from "./snapshot";
import { registerBrowseClick } from "./click";
import { registerBrowseType } from "./type";
import { registerBrowseEval } from "./eval";
import { registerBrowseScreenshot } from "./screenshot";
import { registerBrowseWait } from "./wait";

export function registerAllCoreTools(pi: ExtensionAPI) {
  registerBrowseGoto(pi);
  registerBrowseSnapshot(pi);
  registerBrowseClick(pi);
  registerBrowseType(pi);
  registerBrowseEval(pi);
  registerBrowseScreenshot(pi);
  registerBrowseWait(pi);
}
