import type { TRPCError } from "@trpc/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

type Resolver = (opts: {
  ctx: {
    user: { id: string };
    tenant: { id: string; role: string };
    audit: { location: string; userAgent: string };
  };
  input: { name: string; slug: string };
}) => Promise<{ orgId: string; slug: string }>;

const state = vi.hoisted(() => ({
  resolver: null as Resolver | null,
  provider: "workos" as "workos" | "local",
  /** Whether the session organization already has a workspace. */
  tenantHasWorkspace: false,
  claimed: false,
  claimTenant: vi.fn(),
  createTenant: vi.fn(),
  insertedOrgIds: [] as string[],
}));

vi.mock("@/lib/db", async () => {
  const actual = await vi.importActual<typeof import("@unkey/db")>("@unkey/db");

  return {
    ...actual,
    db: {
      transaction: async <T>(fn: (tx: unknown) => Promise<T>) => {
        let lookups = 0;
        const tx = {
          query: {
            workspaces: {
              // The first lookup is the session organization, the second the slug.
              findFirst: async () => {
                lookups++;
                return lookups === 1 && state.tenantHasWorkspace ? { id: "ws_tenant" } : undefined;
              },
            },
          },
          insert: (table: unknown) => ({
            values: async (row: { orgId?: string }) => {
              if (table === actual.schema.workspaces && row.orgId) {
                state.insertedOrgIds.push(row.orgId);
              }
            },
          }),
        };
        return fn(tx);
      },
    },
  };
});

vi.mock("@/lib/env", () => ({
  env: () => ({ AUTH_PROVIDER: state.provider }),
}));

vi.mock("@/lib/audit", () => ({
  insertAuditLogs: async () => {},
}));

vi.mock("@/lib/auth/server", () => ({
  auth: {
    claimTenant: state.claimTenant,
    createTenant: state.createTenant,
  },
}));

vi.mock("../../trpc", () => ({
  protectedProcedure: {
    input: () => ({
      mutation: (fn: Resolver) => {
        state.resolver = fn;
        return {};
      },
    }),
  },
}));

import "./create";

const input = { name: "Acme", slug: "acme" };

function createWorkspace(tenantId = "org_session") {
  if (!state.resolver) {
    throw new Error("createWorkspace resolver was not registered");
  }

  return state.resolver({
    ctx: {
      user: { id: "user_1" },
      tenant: { id: tenantId, role: "member" },
      audit: { location: "127.0.0.1", userAgent: "vitest" },
    },
    input,
  });
}

describe("createWorkspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    state.provider = "workos";
    state.tenantHasWorkspace = false;
    state.claimed = false;
    state.insertedOrgIds = [];
    state.claimTenant.mockImplementation(async () => state.claimed);
    state.createTenant.mockResolvedValue("org_new");
  });

  it("puts the workspace on the session organization when the user can claim it", async () => {
    state.claimed = true;

    await expect(createWorkspace()).resolves.toEqual({ orgId: "org_session", slug: "acme" });

    expect(state.claimTenant).toHaveBeenCalledWith({
      orgId: "org_session",
      userId: "user_1",
      name: "Acme",
    });
    expect(state.createTenant).not.toHaveBeenCalled();
    expect(state.insertedOrgIds).toEqual(["org_session"]);
  });

  it("creates a new organization when the user cannot claim the session organization", async () => {
    await expect(createWorkspace()).resolves.toEqual({ orgId: "org_new", slug: "acme" });

    expect(state.claimTenant).toHaveBeenCalledOnce();
    expect(state.createTenant).toHaveBeenCalledWith({ name: "Acme", userId: "user_1" });
    expect(state.insertedOrgIds).toEqual(["org_new"]);
  });

  it("never claims a session organization that already has a workspace", async () => {
    state.tenantHasWorkspace = true;

    await expect(createWorkspace()).resolves.toEqual({ orgId: "org_new", slug: "acme" });

    expect(state.claimTenant).not.toHaveBeenCalled();
    expect(state.insertedOrgIds).toEqual(["org_new"]);
  });

  it("never claims when the session has no organization", async () => {
    await expect(createWorkspace("user_1")).resolves.toEqual({ orgId: "org_new", slug: "acme" });

    expect(state.claimTenant).not.toHaveBeenCalled();
  });

  it("still allows only one workspace in local mode", async () => {
    state.provider = "local";
    state.tenantHasWorkspace = true;

    await expect(createWorkspace()).rejects.toMatchObject({
      code: "METHOD_NOT_SUPPORTED",
    } satisfies Partial<TRPCError>);

    expect(state.claimTenant).not.toHaveBeenCalled();
    expect(state.createTenant).not.toHaveBeenCalled();
  });
});
