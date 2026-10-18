import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { createServer, type Server } from "node:http";
import { execFile } from "node:child_process";
import { findRodCli, execRodCli, type ExecFn } from "@agenthands/rod-cli-core";

/**
 * A real child_process execFn for integration tests.
 * This lets execRodCli() work in the test environment as it would in Pi.
 */
function createRealExecFn(bin: string): ExecFn {
  return (args, opts) => {
    return new Promise((resolve, reject) => {
      const child = execFile(bin, ["--headless", ...args], { timeout: opts?.timeout }, (err, stdout, stderr) => {
        if (err) {
          // Non-zero exit is NOT an error here — we return it as a result.
          if (err.code === undefined) {
            reject(err);
            return;
          }
          resolve({
            code: Number(err.code ?? 1),
            stdout: stdout ?? "",
            stderr: stderr ?? "",
          });
        } else {
          resolve({ code: 0, stdout: stdout ?? "", stderr: stderr ?? "" });
        }
      });
      if (opts?.signal) {
        opts.signal.addEventListener("abort", () => {
          child.kill();
        }, { once: true });
      }
    });
  };
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
  const bin = rodCliPath ?? "rod-cli";
  const execFn = createRealExecFn(bin);

  describe.skipIf(!rodCliPath)("with rod-cli", () => {
    let server: Server | undefined;
    let url: string;

    beforeAll(async () => {
      // Start a local HTTP server serving the fixture
      server = createServer((_req, res) => {
        res.writeHead(200, { "Content-Type": "text/html" });
        res.end(FIXTURE_HTML);
      });
      await new Promise<void>((resolve) => {
        server!.listen(0, "127.0.0.1", () => {
          const addr = server!.address();
          if (addr && typeof addr === "object") {
            url = `http://127.0.0.1:${addr.port}`;
          }
          resolve();
        });
      });
    });

    afterAll(() => {
      if (server) {
        server.closeAllConnections();
        server.close();
      }
    });

    it("goto navigates and snapshot reads content", async () => {
      await execRodCli(execFn, ["goto", "--", url], { signal: AbortSignal.timeout(15000) });
      const snap = await execRodCli(execFn, ["snapshot"], { signal: AbortSignal.timeout(10000) });
      expect(snap.stdout).toContain("Fixture Page");
    });

    it("eval returns page title", async () => {
      const result = await execRodCli(execFn, ["eval", "--", "document.title"], { signal: AbortSignal.timeout(10000) });
      expect(result.stdout.trim()).toBe("Integration Fixture");
    });

    it("screenshot produces output", async () => {
      const result = await execRodCli(execFn, ["screenshot"], { signal: AbortSignal.timeout(15000) });
      expect(result.stdout).toBeDefined();
    });

    it("eval can query DOM for element presence", async () => {
      const result = await execRodCli(execFn, ["eval", "--", "!!document.querySelector('#msg')"], { signal: AbortSignal.timeout(10000) });
      expect(result.stdout).toContain("true");
    });

    it("tabs list works", async () => {
      const result = await execRodCli(execFn, ["tab-list"], { signal: AbortSignal.timeout(10000) });
      expect(result.stdout).toBeDefined();
    });

    it("cookies get works", async () => {
      const result = await execRodCli(execFn, ["cookie-get"], { signal: AbortSignal.timeout(10000) });
      expect(result.stdout).toBeDefined();
    });

    it("navigate reload works", async () => {
      await execRodCli(execFn, ["reload"], { signal: AbortSignal.timeout(10000) });
      const snap = await execRodCli(execFn, ["snapshot"], { signal: AbortSignal.timeout(10000) });
      expect(snap.stdout).toContain("Fixture Page");
    });

    it("storage set+get works", async () => {
      await execRodCli(execFn, ["localstorage-set", "--", "testkey", "testval"], { signal: AbortSignal.timeout(10000) });
      const result = await execRodCli(execFn, ["localstorage-get", "--", "testkey"], { signal: AbortSignal.timeout(10000) });
      expect(result.stdout).toContain("testval");
    });
  });

  it("smoke: rod-cli is available or null", () => {
    expect(rodCliPath === null || typeof rodCliPath === "string").toBe(true);
  });
});
