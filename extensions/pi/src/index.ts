import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { findRodCli, setPi } from "./cli";
import { registerLifecycle } from "./lifecycle";
import { registerAllCoreTools, registerAllExtendedTools } from "./tools";

export default function (pi: ExtensionAPI) {
  setPi(pi);
  const rodCliPath = findRodCli();
  registerLifecycle(pi, rodCliPath);
  registerAllCoreTools(pi);
  registerAllExtendedTools(pi);
}
