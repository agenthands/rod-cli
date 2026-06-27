import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { findRodCli, setPi } from "./cli";
import { registerLifecycle } from "./lifecycle";

export default function (pi: ExtensionAPI) {
  setPi(pi);
  const rodCliPath = findRodCli();
  registerLifecycle(pi, rodCliPath);
  // Tools will be registered in Phase 48-49
}
