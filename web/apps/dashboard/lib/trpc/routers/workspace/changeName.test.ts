import { TRPCError } from "@trpc/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

type Resolver = (opts: {
  ctx: {
    workspace: { id: string; name: string };
    user: { id: string };
    tenant: { id: string };
    audit: { location: string; userAgent: string };
  };
  input: { name: string; workspaceId: string };
}) => Promise<void>;

const state = vi.hoisted(() => ({
  resolver: null as Resolver | null,
  /** Errors thrown by the workspace update, one per attempt. */
  updateErrors: [] as unknown[],
  authError: null as unknown,
  /** Order of the side effects. */
  calls: [] as string[],
}));

vi.mock("@/lib/db", async () => {
  const actual = await vi.importActual<typeof import("@unkey/db")>("@unkey/db");

  const tx = {
    update: () => ({
      set: () => ({
        where: async () => {
          state.calls.push("update");
          const err = state.updateErrors.shift();
          if (err) {
            throw err;
          }
        },
      }),
    }),
  };

  return {
    ...actual,
    db: {
      transaction: <T>(fn: (tx: unknown) => Promise<T>) => {
        state.calls.push("begin");
        return fn(tx);
      },
    },
  };
});

vi.mock("@/lib/audit", () => ({
  insertAuditLogs: async () => {
    state.calls.push("audit");
  },
}));

vi.mock("@/lib/auth/server", () => ({
  auth: {
    updateOrg: async () => {
      state.calls.push("auth");
      if (state.authError) {
        throw state.authError;
      }
    },
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

import "./changeName";

/** The mysql2 shape of a deadlock. */
function deadlock(): Error {
  return Object.assign(new Error("Deadlock found when trying to get lock"), {
    errno: 1213,
    code: "ER_LOCK_DEADLOCK",
  });
}

const ctx = {
  workspace: { id: "ws_1", name: "Old name" },
  user: { id: "user_1" },
  tenant: { id: "org_1" },
  audit: { location: "127.0.0.1", userAgent: "vitest" },
};

function changeName(input = { name: "New name", workspaceId: "ws_1" }) {
  if (!state.resolver) {
    throw new Error("changeWorkspaceName resolver was not registered");
  }

  return state.resolver({ ctx, input });
}

describe("changeWorkspaceName", () => {
  beforeEach(() => {
    state.updateErrors = [];
    state.authError = null;
    state.calls = [];
  });

  it("rejects a workspace id that is not the caller's own", async () => {
    await expect(changeName({ name: "New name", workspaceId: "ws_other" })).rejects.toMatchObject({
      code: "BAD_REQUEST",
    } satisfies Partial<TRPCError>);

    expect(state.calls).toEqual([]);
  });

  it("writes the database before the auth provider", async () => {
    await changeName();

    expect(state.calls).toEqual(["begin", "update", "audit", "auth"]);
  });

  it("runs the whole transaction again after a deadlock", async () => {
    state.updateErrors = [deadlock()];

    await changeName();

    expect(state.calls).toEqual(["begin", "update", "begin", "update", "audit", "auth"]);
  });

  it("gives up after three attempts and reports a failure", async () => {
    state.updateErrors = [deadlock(), deadlock(), deadlock()];

    await expect(changeName()).rejects.toMatchObject({
      code: "INTERNAL_SERVER_ERROR",
    } satisfies Partial<TRPCError>);

    expect(state.calls.filter((call) => call === "update")).toHaveLength(3);
  });

  it("does not retry an error that a retry cannot fix", async () => {
    state.updateErrors = [new Error("column is too long")];

    await expect(changeName()).rejects.toThrow(TRPCError);
    expect(state.calls.filter((call) => call === "update")).toHaveLength(1);
  });

  it("keeps the original failure as the cause", async () => {
    const original = new Error("column is too long");
    state.updateErrors = [original];

    await expect(changeName()).rejects.toMatchObject({ cause: original });
  });

  it("fails the transaction when the auth provider rejects the name", async () => {
    state.authError = new Error("workos rejected the organization name");

    await expect(changeName()).rejects.toMatchObject({
      code: "INTERNAL_SERVER_ERROR",
    } satisfies Partial<TRPCError>);
  });
});
