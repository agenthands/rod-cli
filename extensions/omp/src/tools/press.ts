import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { execRodCli, buildPressArgs, type ExecFn } from "@agenthands/rod-cli-core";

const SESSION_DESC = "Named session identifier for multi-target workflows (e.g. 'admin', 'guest'). Omit to use the default session.";

export function registerBrowsePress(pi: ExtensionAPI, execFn: ExecFn) {
  const { z } = pi.zod;
  const parameters = z.object({
    key: z.string().describe("Key name: Enter, Tab, Backspace, Escape, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, or a single character"),
    session: z.string().optional().describe(SESSION_DESC),
  });
  pi.registerTool({
    name: "browse_press",
    label: "Browse Press",
    description:
      "browse_press: Press one keyboard key on the focused element with real key input " +
      "(Enter, Tab, Backspace, Escape, ArrowUp/Down/Left/Right, or a single character). " +
      'Usage: browse_press(key="Enter") or browse_press(key="Escape"). ' +
      "Use browse_press to submit with Enter, move focus with Tab, close dialogs with Escape, or navigate lists with arrow keys. " +
      "Focus the target first (browse_click), then press.",
    parameters,
    async execute(_toolCallId: string, params: Record<string, unknown>, signal: AbortSignal | undefined) {
      const result = await execRodCli(execFn, buildPressArgs({ key: params.key as string, session: params.session as string | undefined }), { signal });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  } as never);
}
