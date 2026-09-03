import type { TRPCError } from "@trpc/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

type Resolver = (opts: {
  ctx: {
    workspace: { id: string };
    user: { id: string };
    audit: { location: string; userAgent: string };
  };
  input: { alertId: string; message: string };
}) => Promise<unknown>;

const state = vi.hoisted(() => ({
  resolver: null as Resolver | null,
  alert: {
    id: "alert_1",
    status: "open" as "open" | "resolved",
    appId: "app_1",
    environmentId: "env_1",
    metric: "error_5xx",
  },
  update: null as Record<string, unknown> | null,
  audit: null as Record<string, unknown> | null,
}));

vi.mock("@/lib/db", async () => {
  const actual = await vi.importActual<typeof import("@unkey/db")>("@unkey/db");
  const tx = {
    select: () => ({
      from: () => ({
        where: () => ({
          for: async () => [state.alert],
        }),
      }),
    }),
    update: () => ({
      set: (values: Record<string, unknown>) => {
        state.update = values;
        return { where: async () => undefined };
      },
    }),
  };

  return {
    ...actual,
    db: {
      transaction: async <T>(fn: (transaction: typeof tx) => Promise<T>) => fn(tx),
    },
  };
});

vi.mock("@/lib/audit", () => ({
  insertAuditLogs: async (_tx: unknown, audit: Record<string, unknown>) => {
    state.audit = audit;
  },
}));

vi.mock("../../trpc", () => ({
  workspaceProcedure: {
    input: () => ({
      mutation: (resolver: Resolver) => {
        state.resolver = resolver;
        return {};
      },
    }),
  },
}));

import "./resolve";

function resolve() {
  if (!state.resolver) {
    throw new Error("resolveAlert resolver was not registered");
  }
  return state.resolver({
    ctx: {
      workspace: { id: "ws_1" },
      user: { id: "user_1" },
      audit: { location: "127.0.0.1", userAgent: "vitest" },
    },
    input: { alertId: "alert_1", message: "Raised instance memory" },
  });
}

describe("resolveAlert", () => {
  beforeEach(() => {
    state.alert.status = "open";
    state.update = null;
    state.audit = null;
  });

  it("moves an open alert to resolved and records the actor", async () => {
    await expect(resolve()).resolves.toMatchObject({
      id: "alert_1",
      status: "resolved",
      resolvedBy: "user_1",
      resolutionMessage: "Raised instance memory",
    });
    expect(state.update).toMatchObject({
      status: "resolved",
      resolvedBy: "user_1",
      resolutionMessage: "Raised instance memory",
    });
    expect(state.audit).toMatchObject({ event: "alert.resolve" });
  });

  it("rejects an alert that is already resolved", async () => {
    state.alert.status = "resolved";

    await expect(resolve()).rejects.toMatchObject({
      code: "PRECONDITION_FAILED",
    } satisfies Partial<TRPCError>);
    expect(state.update).toBeNull();
    expect(state.audit).toBeNull();
  });
});
