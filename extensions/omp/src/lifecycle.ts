import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";

export function registerLifecycle(pi: ExtensionAPI, rodCliPath: string | null) {
  const bin = rodCliPath ?? "rod-cli";
  pi.on("session_start", async (_event, ctx) => {
    if (!rodCliPath) {
      ctx.ui.notify("rod-cli not found. Install: go install github.com/agenthands/rod-cli@latest", "warning");
      return;
    }
    try {
      const result = await pi.exec(bin, ["--version"], { timeout: 5000 });
      if (result.code === 0) {
        ctx.ui.notify(`rod-cli ${result.stdout.trim() || "unknown"} ready`, "info");
      } else {
        // OMP's pi.exec resolves on non-zero; a missing/failed --version must not be reported as "ready".
        ctx.ui.notify("rod-cli found but --version failed. Check your installation.", "warning");
      }
    } catch {
      ctx.ui.notify("rod-cli found but --version failed. Check your installation.", "warning");
    }
  });
  pi.on("session_shutdown", async () => {
    // Ungated + idempotent: OMP's session_shutdown carries no reason.
    try { await pi.exec(bin, ["close"], { timeout: 5000 }); } catch { /* best-effort */ }
  });
}
