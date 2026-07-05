import type { ExtensionAPI, ExtensionContext } from "@oh-my-pi/pi-coding-agent";
import * as z from "zod/v4";

interface RegisteredTool {
  name: string;
  execute: (id: string, params: Record<string, unknown>, signal?: AbortSignal) => Promise<{ content: Array<{ type: string; text: string }>; details?: unknown }>;
}

interface MockApiState {
  tools: RegisteredTool[];
  handlers: Record<string, Array<(e: unknown, ctx: unknown) => unknown>>;
  execCalls: string[][];
  notifications: Array<{ msg: string; level?: string }>;
}

export function createMockApi(opts?: { execCode?: number }): MockApiState & { api: ExtensionAPI; fire(event: string, e?: unknown): Promise<void> } {
  const state: MockApiState = {
    tools: [],
    handlers: {},
    execCalls: [],
    notifications: [],
  };

  const ctx: ExtensionContext = {
    ui: {
      notify(msg: string, level?: "info" | "warning" | "error") {
        state.notifications.push({ msg, level });
      },
    },
  } as unknown as ExtensionContext;

  // Build a minimal TypeBox shim for the mock — just enough for schema construction.
  const T = {
    Object: (shape: Record<string, unknown>) => shape,
    String: (opts?: { description?: string; default?: string }) => ({ type: "string", ...opts }),
    Number: (opts?: { description?: string; default?: number }) => ({ type: "number", ...opts }),
    Boolean: (opts?: { description?: string; default?: boolean }) => ({ type: "boolean", ...opts }),
    Optional: (schema: unknown) => schema,
    Union: (variants: unknown[]) => ({ anyOf: variants }),
    Literal: (value: string) => ({ const: value }),
  };

  const api = {
    zod: { z },
    typebox: { Type: T },
    registerTool(def: RegisteredTool) {
      state.tools.push(def);
    },
    on(event: string, handler: (e: unknown, ctx: unknown) => unknown) {
      if (!state.handlers[event]) state.handlers[event] = [];
      state.handlers[event]!.push(handler);
    },
    exec(_cmd: string, args: string[], _opts?: unknown) {
      state.execCalls.push(args);
      return Promise.resolve({ stdout: "ok", stderr: "", code: opts?.execCode ?? 0, killed: false });
    },
    logger: { info: () => {}, warn: () => {}, error: () => {}, debug: () => {} },
  } as unknown as ExtensionAPI;

  return {
    ...state,
    api,
    async fire(event: string, e?: unknown) {
      const handlers = state.handlers[event];
      if (!handlers) return;
      for (const h of handlers) {
        await h(e ?? { type: event }, ctx);
      }
    },
  };
}
