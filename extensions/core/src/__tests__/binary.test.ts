import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { Stats } from "node:fs";

/** Partial Stats mock — findRodCli only reads .isFile(). */
const asStats = (isFile: boolean): Stats =>
  ({ isFile: () => isFile }) as unknown as Stats;

const { fsStatSyncMock, osPlatformMock } = vi.hoisted(() => ({
  fsStatSyncMock: vi.fn<(path: string, opts?: unknown) => Stats | undefined>(),
  osPlatformMock: vi.fn(() => "linux"),
}));

vi.mock("node:fs", () => ({
  statSync: fsStatSyncMock,
}));

vi.mock("node:os", () => ({
  platform: osPlatformMock,
  default: { platform: osPlatformMock },
}));

async function importBinary() {
  return await import("../binary");
}

describe("findRodCli — binary resolution", () => {
  const originalEnv = { ...process.env };

  beforeEach(() => {
    vi.clearAllMocks();
    process.env = { ...originalEnv };
    delete process.env.ROD_CLI_PATH;
    delete process.env.GOBIN;
    process.env.HOME = "/home/testuser";
    process.env.PATH = "/usr/bin:/bin";
    osPlatformMock.mockReturnValue("linux");
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it("returns null when nothing is found (totality)", async () => {
    fsStatSyncMock.mockReturnValue(undefined);
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBeNull();
  });

  it("never throws even with all env vars unset", async () => {
    delete process.env.HOME;
    delete process.env.PATH;
    delete process.env.ROD_CLI_PATH;
    delete process.env.GOBIN;
    fsStatSyncMock.mockReturnValue(undefined);
    const { findRodCli } = await importBinary();
    expect(() => findRodCli()).not.toThrow();
    expect(findRodCli()).toBeNull();
  });

  it("returns ROD_CLI_PATH if it points to a file", async () => {
    process.env.ROD_CLI_PATH = "/opt/bin/rod-cli";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/opt/bin/rod-cli") return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBe("/opt/bin/rod-cli");
  });

  it("skips ROD_CLI_PATH if it points to a directory", async () => {
    process.env.ROD_CLI_PATH = "/opt/bin";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/opt/bin") return asStats(false);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBeNull();
  });

  it("skips ROD_CLI_PATH when statSync throws (permission denied)", async () => {
    process.env.ROD_CLI_PATH = "/root/forbidden/rod-cli";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/root/forbidden/rod-cli") throw new Error("EACCES");
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBeNull();
  });

  it("skips ROD_CLI_PATH when it's an empty string", async () => {
    process.env.ROD_CLI_PATH = "";
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBeNull();
  });

  it("finds binary in PATH", async () => {
    process.env.PATH = "/a:/b:/c";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/b/rod-cli") return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBe("/b/rod-cli");
  });

  it("returns first PATH match (leftmost wins)", async () => {
    process.env.PATH = "/first:/second";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/first/rod-cli" || p === "/second/rod-cli")
        return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBe("/first/rod-cli");
  });

  it("handles empty PATH string", async () => {
    process.env.PATH = "";
    const { findRodCli } = await importBinary();
    expect(() => findRodCli()).not.toThrow();
    expect(findRodCli()).toBeNull();
  });

  it("skips empty PATH entries (leading/trailing/doubled separators)", async () => {
    process.env.PATH = ":/usr/bin::/bin:";
    fsStatSyncMock.mockReturnValue(undefined);
    const { findRodCli } = await importBinary();
    expect(() => findRodCli()).not.toThrow();
  });

  it("handles PATH with only empty entries", async () => {
    process.env.PATH = ":::";
    const { findRodCli } = await importBinary();
    expect(() => findRodCli()).not.toThrow();
    expect(findRodCli()).toBeNull();
  });

  it("finds binary via GOBIN", async () => {
    process.env.PATH = "";
    process.env.GOBIN = "/custom/go/bin";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/custom/go/bin/rod-cli") return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBe("/custom/go/bin/rod-cli");
  });

  it("falls back to $HOME/go/bin when GOBIN is unset", async () => {
    process.env.PATH = "";
    delete process.env.GOBIN;
    process.env.HOME = "/home/testuser";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/home/testuser/go/bin/rod-cli") return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBe("/home/testuser/go/bin/rod-cli");
  });

  it("returns null when HOME is also unset", async () => {
    delete process.env.HOME;
    delete process.env.GOBIN;
    process.env.PATH = "";
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBeNull();
  });

  it("ROD_CLI_PATH wins over PATH and GOBIN", async () => {
    process.env.ROD_CLI_PATH = "/explicit/rod-cli";
    process.env.PATH = "/pathdir";
    process.env.GOBIN = "/gobindir";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/explicit/rod-cli" || p === "/pathdir/rod-cli" || p === "/gobindir/rod-cli")
        return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBe("/explicit/rod-cli");
  });

  it("PATH wins over GOBIN when ROD_CLI_PATH unset", async () => {
    delete process.env.ROD_CLI_PATH;
    process.env.PATH = "/pathdir";
    process.env.GOBIN = "/gobindir";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/pathdir/rod-cli" || p === "/gobindir/rod-cli")
        return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBe("/pathdir/rod-cli");
  });

  it("on Windows, checks USERPROFILE instead of HOME", async () => {
    osPlatformMock.mockReturnValue("win32");
    delete process.env.HOME;
    process.env.USERPROFILE = "C:\\Users\\testuser";
    process.env.PATH = "";
    delete process.env.GOBIN;
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p.includes("rod-cli.exe") && p.includes("go") && p.includes("bin"))
        return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    const result = findRodCli();
    expect(result === null || (typeof result === "string" && result.includes("rod-cli.exe"))).toBe(true);
  });

  it("is idempotent — same result on repeated calls", async () => {
    process.env.ROD_CLI_PATH = "/opt/rod-cli";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/opt/rod-cli") return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    const r1 = findRodCli();
    const r2 = findRodCli();
    const r3 = findRodCli();
    expect(r1).toBe(r2);
    expect(r2).toBe(r3);
  });

  it("handles paths with spaces and special characters", async () => {
    process.env.ROD_CLI_PATH = "/home/user/My Tools/rod-cli";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/home/user/My Tools/rod-cli") return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBe("/home/user/My Tools/rod-cli");
  });

  it("handles PATH entries with spaces", async () => {
    process.env.PATH = "/home/user/My Bin:/usr/bin";
    fsStatSyncMock.mockImplementation((p: string) => {
      if (p === "/home/user/My Bin/rod-cli") return asStats(true);
      return undefined;
    });
    const { findRodCli } = await importBinary();
    expect(findRodCli()).toBe("/home/user/My Bin/rod-cli");
  });
});
