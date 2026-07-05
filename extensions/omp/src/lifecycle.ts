import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";

export function registerLifecycle(pi: ExtensionAPI, rodCliPath: string | null) {
  const bin = rodCliPath ?? "rod-cli";
  pi.on("session_start", async (_event, ctx) => {
    if (!rodCliPath) {
      ctx.ui.notify("rod-cli not found. Install: go install github.com/agenthands/rod-cli@latest", "warning");
      return;
    }
    // Binary exists (findRodCli confirmed it). Best-effort version enrichment;
    // a version-probe failure is not an install problem, so never warn here.
    let msg = "rod-cli ready";
    try {
      const result = await pi.exec(bin, ["--version"], { timeout: 5000 });
      if (result.code === 0 && result.stdout.trim()) msg = `${result.stdout.trim()} ready`;
    } catch {
      /* binary presence already confirmed */
    }
    ctx.ui.notify(msg, "info");
  });
  pi.on("session_shutdown", async () => {
    // Ungated + idempotent: OMP's session_shutdown carries no reason.
    try { await pi.exec(bin, ["close"], { timeout: 5000 }); } catch { /* best-effort */ }
  });
}
