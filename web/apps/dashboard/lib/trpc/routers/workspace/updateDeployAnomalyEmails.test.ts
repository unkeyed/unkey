import { beforeEach, describe, expect, it, vi } from "vitest";

type Resolver = (opts: {
  ctx: {
    workspace: {
      id: string;
      name: string;
      betaFeatures: { rbac?: boolean; deploy_anomaly_alerts_muted?: boolean };
    };
    user: { id: string };
    audit: { location: string; userAgent: string };
  };
  input: { muted: boolean };
}) => Promise<{ muted: boolean }>;

const state = vi.hoisted(() => ({
  resolver: null as Resolver | null,
  updatedBetaFeatures: null as Record<string, boolean> | null,
  auditDescription: "",
  calls: [] as string[],
}));

vi.mock("@/lib/db", async () => {
  const actual = await vi.importActual<typeof import("@unkey/db")>("@unkey/db");
  const tx = {
    update: () => ({
      set: (values: { betaFeatures: Record<string, boolean> }) => {
        state.updatedBetaFeatures = values.betaFeatures;
        return {
          where: async () => {
            state.calls.push("update");
          },
        };
      },
    }),
  };
  return {
    ...actual,
    db: {},
    transactionWithRetry: async <T>(_db: unknown, fn: (transaction: typeof tx) => Promise<T>) =>
      fn(tx),
  };
});

vi.mock("@/lib/audit", () => ({
  insertAuditLogs: async (_tx: unknown, log: { description: string }) => {
    state.calls.push("audit");
    state.auditDescription = log.description;
  },
}));

vi.mock("../../trpc", () => ({
  requireWorkspaceAdmin: {},
  workspaceProcedure: {
    use: () => ({
      input: () => ({
        mutation: (fn: Resolver) => {
          state.resolver = fn;
          return {};
        },
      }),
    }),
  },
}));

import "./updateDeployAnomalyEmails";

function update(muted: boolean) {
  if (!state.resolver) {
    throw new Error("updateDeployAnomalyEmails resolver was not registered");
  }
  return state.resolver({
    ctx: {
      workspace: { id: "ws_1", name: "Acme", betaFeatures: { rbac: true } },
      user: { id: "user_1" },
      audit: { location: "127.0.0.1", userAgent: "vitest" },
    },
    input: { muted },
  });
}

describe("updateDeployAnomalyEmails", () => {
  beforeEach(() => {
    state.updatedBetaFeatures = null;
    state.auditDescription = "";
    state.calls = [];
  });

  it("preserves other beta features and audits muting", async () => {
    await expect(update(true)).resolves.toEqual({ muted: true });

    expect(state.updatedBetaFeatures).toEqual({
      rbac: true,
      deploy_anomaly_alerts_muted: true,
    });
    expect(state.auditDescription).toBe("Muted Deploy anomaly emails.");
    expect(state.calls).toEqual(["update", "audit"]);
  });

  it("audits enabling emails", async () => {
    await update(false);

    expect(state.updatedBetaFeatures).toEqual({
      rbac: true,
      deploy_anomaly_alerts_muted: false,
    });
    expect(state.auditDescription).toBe("Enabled Deploy anomaly emails.");
  });
});
