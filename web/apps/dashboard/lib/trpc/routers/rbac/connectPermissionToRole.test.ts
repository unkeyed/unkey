import { TRPCError } from "@trpc/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

type Resolver = (opts: {
  ctx: {
    workspace: { id: string };
    user: { id: string };
    tenant: { id: string };
    audit: { location: string; userAgent: string };
  };
  input: { roleId: string; permissionId: string };
}) => Promise<void>;

const state = vi.hoisted(() => ({
  resolver: null as Resolver | null,
  roleProjectId: "proj_role",
  permissionProjectId: "proj_permission",
  transactionCalls: 0,
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      workspaces: {
        findFirst: async () => ({
          id: "ws_1",
          roles: [{ id: "role_1", name: "Role", projectId: state.roleProjectId }],
          permissions: [
            {
              id: "perm_1",
              name: "Permission",
              projectId: state.permissionProjectId,
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

import "./connectPermissionToRole";

const ctx = {
  workspace: { id: "ws_1" },
  user: { id: "user_1" },
  tenant: { id: "org_1" },
  audit: { location: "127.0.0.1", userAgent: "vitest" },
};

function connectPermissionToRole() {
  if (!state.resolver) {
    throw new Error("connectPermissionToRole resolver was not registered");
  }

  return state.resolver({
    ctx,
    input: { roleId: "role_1", permissionId: "perm_1" },
  });
}

describe("connectPermissionToRole", () => {
  beforeEach(() => {
    state.roleProjectId = "proj_role";
    state.permissionProjectId = "proj_permission";
    state.transactionCalls = 0;
  });

  it("rejects a role and permission from different projects before writing", async () => {
    await expect(connectPermissionToRole()).rejects.toMatchObject({
      code: "BAD_REQUEST",
    } satisfies Partial<TRPCError>);
    expect(state.transactionCalls).toBe(0);
  });
});
