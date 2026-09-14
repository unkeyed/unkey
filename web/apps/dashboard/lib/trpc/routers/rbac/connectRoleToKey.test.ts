import { TRPCError } from "@trpc/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

type Resolver = (opts: {
  ctx: {
    workspace: { id: string };
    user: { id: string };
    tenant: { id: string };
    audit: { location: string; userAgent: string };
  };
  input: { roleId: string; keyId: string };
}) => Promise<void>;

const state = vi.hoisted(() => ({
  resolver: null as Resolver | null,
  roleProjectId: "proj_role",
  keyProjectId: "proj_key",
  transactionCalls: 0,
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      workspaces: {
        findFirst: async () => ({
          id: "ws_1",
          roles: [{ id: "role_1", name: "Role", projectId: state.roleProjectId }],
          keys: [
            {
              id: "key_1",
              name: "Key",
              keyAuth: { projectId: state.keyProjectId },
            },
          ],
        }),
      },
    },
    transaction: async () => {
      state.transactionCalls++;
    },
  },
  schema: {},
}));

vi.mock("@/lib/audit", () => ({ insertAuditLogs: vi.fn() }));

vi.mock("../../trpc", () => ({
  workspaceProcedure: {
    input: () => ({
      mutation: (fn: Resolver) => {
        state.resolver = fn;
        return {};
      },
    }),
  },
}));

import "./connectRoleToKey";

const ctx = {
  workspace: { id: "ws_1" },
  user: { id: "user_1" },
  tenant: { id: "org_1" },
  audit: { location: "127.0.0.1", userAgent: "vitest" },
};

function connectRoleToKey() {
  if (!state.resolver) {
    throw new Error("connectRoleToKey resolver was not registered");
  }

  return state.resolver({ ctx, input: { roleId: "role_1", keyId: "key_1" } });
}

describe("connectRoleToKey", () => {
  beforeEach(() => {
    state.roleProjectId = "proj_role";
    state.keyProjectId = "proj_key";
    state.transactionCalls = 0;
  });

  it("rejects a role and key from different projects before writing", async () => {
    await expect(connectRoleToKey()).rejects.toMatchObject({
      code: "BAD_REQUEST",
    } satisfies Partial<TRPCError>);
    expect(state.transactionCalls).toBe(0);
  });
});
