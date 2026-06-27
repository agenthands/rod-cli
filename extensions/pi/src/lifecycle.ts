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

    try {
      const result = await pi.exec("rod-cli", ["--version"], { timeout: 5000 });
      const version = result.stdout.trim() || "unknown";
      ctx.ui.notify(`rod-cli ${version} ready`, "info");
    } catch {
      ctx.ui.notify(
        "rod-cli found but --version failed. Check your installation.",
        "warning",
      );
    }
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
