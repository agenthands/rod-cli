import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { findRodCli, type ExecFn } from "@agenthands/rod-cli-core";
import { registerLifecycle } from "./lifecycle";
import { registerAllCoreTools, registerAllExtendedTools } from "./tools";

export default function (pi: ExtensionAPI) {
  const rodCliPath = findRodCli();
  const bin = rodCliPath ?? "rod-cli";
  const execFn: ExecFn = (args, opts) => pi.exec(bin, args, opts);
  registerLifecycle(pi, rodCliPath);
  registerAllCoreTools(pi, execFn);
  registerAllExtendedTools(pi, execFn);
}
