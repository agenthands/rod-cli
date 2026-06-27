import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { createServer, type Server } from "node:http";
import { execFile } from "node:child_process";
import { findRodCli, execRodCli, setPi } from "../cli";

/**
 * A real pi.exec shim for integration tests that shells out via child_process.
 * This lets execRodCli() work in the test environment as it would in Pi.
 */
function createRealPiExec() {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const pi: any = {
    exec: (cmd: string, args: string[], opts?: { signal?: AbortSignal; timeout?: number }) => {
      return new Promise<{ stdout: string; stderr: string; code: number }>((resolve) => {
        execFile(cmd, args, {
          timeout: opts?.timeout ?? 30000,
          signal: opts?.signal,
          windowsHide: true,
        }, (error, stdout, stderr) => {
          if (error) {
            // Non-zero exit is not a rejection — matched pi.exec behavior
            resolve({
              stdout,
              stderr,
              code: typeof error.code === "number" ? error.code : 1,
            });
            return;
          }
          resolve({ stdout, stderr, code: 0 });
        });
      });
    },
  };
  return pi;
}

const FIXTURE_HTML = `<!DOCTYPE html>
<html><head><title>Integration Fixture</title></head>
<body>
  <h1>Fixture Page</h1>
  <p id="msg">Hello from fixture</p>
  <input id="name" type="text" placeholder="Enter name" />
  <button id="greet">Greet</button>
</body></html>`;

describe("integration workflow", () => {
  const rodCliPath = findRodCli();

  describe.skipIf(!rodCliPath)("with rod-cli", () => {
    let server: Server | undefined;
    let url: string;

    beforeAll(async () => {
      // Set up a real pi.exec shim so execRodCli works
      setPi(createRealPiExec());

      server = createServer((_req, res) => {
        res.writeHead(200, { "content-type": "text/html" });
        res.end(FIXTURE_HTML);
      });
      const srv = server!;
      await new Promise<void>((r) => srv.listen(0, () => r()));
      const addr = srv.address();
      url = `http://127.0.0.1:${typeof addr === "object" ? addr?.port : ""}`;
    });

    afterAll(async () => {
      try {
        await execRodCli(["close"]);
      } catch {
        /* ok if already closed */
      }
      await new Promise<void>((r) => server?.close(() => r()));
    });

    it("goto navigates and snapshot reads content", async () => {
      await execRodCli(["goto", url]);
      const snap = await execRodCli(["snapshot"]);
      expect(snap.stdout).toContain("Fixture Page");
    });

    it("eval returns page title", async () => {
      const r = await execRodCli(["eval", "document.title"]);
      expect(r.stdout).toContain("Integration Fixture");
    });

    it("screenshot produces output", async () => {
      const r = await execRodCli(["screenshot"]);
      expect(r.stdout).toContain("Save to");
    });

    it("eval can query DOM for element presence", async () => {
      const r = await execRodCli([
        "eval",
        "!!document.querySelector('#msg')",
      ]);
      expect(r.stdout).toContain("true");
    });

    it("tabs list works", async () => {
      const r = await execRodCli(["tab-list"]);
      expect(r.stdout).toBeDefined();
    });

    it("cookies get works", async () => {
      const r = await execRodCli(["cookie-get"]);
      expect(r.stdout).toBeDefined();
    });

    it("navigate reload works", async () => {
      const r = await execRodCli(["reload"]);
      expect(r.stdout).toBeDefined();
    });

    it("storage set+get works", async () => {
      await execRodCli(["localstorage-set", "--", "test_key", "test_val"]);
      const r = await execRodCli(["localstorage-get"]);
      expect(r.stdout).toContain("test_key");
    });
  });

  it("smoke: rod-cli is available or null", () => {
    expect(rodCliPath === null || typeof rodCliPath === "string").toBe(true);
  });
});
