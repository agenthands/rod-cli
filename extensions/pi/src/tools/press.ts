import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { execRodCli, buildPressArgs, type ExecFn } from "@agenthands/rod-cli-core";
import { SessionParam } from "../types";

export function registerBrowsePress(pi: ExtensionAPI, execFn: ExecFn) {
  pi.registerTool({
    name: "browse_press",
    label: "Browse Press",
    description:
      "Press one keyboard key on the focused element with real key input " +
      "(Enter, Tab, Backspace, Escape, ArrowUp/Down/Left/Right, or a single character).",
    promptSnippet:
      'Press a key: browse_press(key="Enter") or browse_press(key="Escape")',
    promptGuidelines: [
      "Use browse_press to submit with Enter, move focus with Tab, close dialogs with Escape, or navigate lists with arrow keys.",
      "Focus the target first (browse_click), then press.",
    ],
    parameters: Type.Object({
      key: Type.String({
        description: "Key name: Enter, Tab, Backspace, Escape, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, or a single character",
      }),
      session: SessionParam,
    }),
    async execute(_toolCallId, params, signal) {
      const result = await execRodCli(execFn, buildPressArgs({ key: params.key, session: params.session }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
