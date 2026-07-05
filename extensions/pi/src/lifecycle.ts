import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export function registerLifecycle(pi: ExtensionAPI, rodCliPath: string | null) {
  pi.on("session_start", async (_event, ctx) => {
    if (!rodCliPath) {
      ctx.ui.notify(
        "rod-cli not found. Install: go install github.com/agenthands/rod-cli@latest",
        "warning",
      );
      return;
    }

    // Binary exists (findRodCli confirmed it). Best-effort version enrichment;
    // a version-probe failure is not an install problem, so never warn here.
    let msg = "rod-cli ready";
    try {
      const result = await pi.exec("rod-cli", ["--version"], { timeout: 5000 });
      if (result.code === 0 && result.stdout.trim()) msg = `${result.stdout.trim()} ready`;
    } catch {
      /* binary presence already confirmed */
    }
    ctx.ui.notify(msg, "info");
  });

  pi.on("session_shutdown", async (event) => {
    // Only close daemon on actual quit -- not on reload/fork/resume.
    if (event.reason !== "quit") return;

    try {
      await pi.exec("rod-cli", ["close"], { timeout: 5000 });
    } catch {
      // Best-effort -- daemon has its own PPID polling + idle timeout.
    }
  });
}
