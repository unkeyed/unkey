import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { createWorkspace } from "./create";

const state = vi.hoisted(() => ({
  inTransaction: false,
  transactions: 0,
  createTenant: vi.fn(),
  findFirst: vi.fn(),
  insert: vi.fn(),
  audit: vi.fn(),
}));
vi.mock("@/lib/auth/server", () => ({ auth: { createTenant: state.createTenant } }));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: state.audit }));
vi.mock("@/lib/env", () => ({ env: () => ({ AUTH_PROVIDER: "workos" }) }));
vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  return { protectedProcedure: initTRPC.create().procedure };
});
vi.mock("@/lib/db", () => {
  const tx = {
    insert: (table: string) => ({ values: async (values: unknown) => state.insert(table, values) }),
  };
  return {
    schema: { workspaces: "workspaces", limits: "limits", workspaceBilling: "billing" },
    db: {
      query: { workspaces: { findFirst: state.findFirst } },
      transaction: async (fn: (connection: typeof tx) => Promise<unknown>) => {
        state.transactions++;
        state.inTransaction = true;
        try {
          return await fn(tx);
        } finally {
          state.inTransaction = false;
        }
      },
    },
  };
});
const caller = initTRPC
  .context<{ user: { id: string }; audit: { location: string } }>()
  .create()
  .router({ create: createWorkspace })
  .createCaller({ user: { id: "user" }, audit: { location: "test" } });

beforeEach(() => {
  state.transactions = 0;
  state.insert.mockReset();
  state.audit.mockReset();
  state.findFirst.mockReset().mockResolvedValue(undefined);
  state.createTenant.mockReset().mockImplementation(async () => {
    expect(state.inTransaction).toBe(false);
    return "org";
  });
});

it("creates the WorkOS tenant before opening the MySQL transaction", async () => {
  expect(await caller.create({ name: "Workspace", slug: "workspace" })).toEqual({
    orgId: "org",
    slug: "workspace",
  });
  expect(state.transactions).toBe(1);
  expect(state.insert).toHaveBeenCalledTimes(3);
  expect(state.audit).toHaveBeenCalledOnce();
});

it("does not open a transaction if WorkOS fails", async () => {
  const log = vi.spyOn(console, "error").mockImplementation(() => {});
  state.createTenant.mockRejectedValue(new Error("WorkOS unavailable"));
  try {
    await expect(caller.create({ name: "Workspace", slug: "workspace" })).rejects.toThrow();
    expect(state.transactions).toBe(0);
    expect(state.insert).not.toHaveBeenCalled();
  } finally {
    log.mockRestore();
  }
});

it("rejects an existing slug without creating a WorkOS tenant", async () => {
  state.findFirst.mockResolvedValue({ id: "existing" });
  await expect(caller.create({ name: "Workspace", slug: "workspace" })).rejects.toThrow(
    "already exists",
  );
  expect(state.createTenant).not.toHaveBeenCalled();
});
