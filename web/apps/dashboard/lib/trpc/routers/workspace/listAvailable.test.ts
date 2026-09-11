import { drizzle, schema } from "@unkey/db";
import { createPool } from "mysql2";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { router } from "../../trpc";
import { listAvailable } from "./listAvailable";

const mocks = vi.hoisted(() => ({
  memberships: vi.fn(),
  findMany: vi.fn(),
}));

vi.mock("@/lib/auth/server", () => ({
  auth: { listMemberships: mocks.memberships },
}));
vi.mock("@/lib/db", () => ({ db: { query: { workspaces: { findMany: mocks.findMany } } } }));
vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  const t = initTRPC.context<{ user: { id: string } }>().create();
  return { protectedProcedure: t.procedure, router: t.router };
});

const caller = router({ listAvailable }).createCaller({
  user: { id: "session-user", profile: null },
  req: new Request("http://localhost"),
  audit: { userAgent: undefined, location: "test" },
  workspace: undefined,
  tenant: null,
});

function membership(orgId: string, status = "active") {
  return { organization: { id: orgId }, status };
}

beforeEach(() => {
  vi.resetAllMocks();
  vi.spyOn(console, "error").mockImplementation(() => {});
  mocks.memberships.mockResolvedValue({
    data: [
      membership("org_live"),
      membership("org_disabled"),
      membership("org_deleted"),
      membership("org_missing"),
      membership("org_inactive", "inactive"),
      membership("org_pending", "pending"),
    ],
  });
  mocks.findMany.mockResolvedValue([
    { orgId: "org_live", name: "Live" },
    { orgId: "org_disabled", name: "Disabled" },
  ]);
});

describe("available workspaces", () => {
  it("derives membership from the session and queries only active member IDs, excluding deleted rows but not disabled rows", async () => {
    const pool = createPool({});
    const database = drizzle(pool, { schema, mode: "default" });
    let query: { sql: string; params: unknown[] } | undefined;
    mocks.findMany.mockImplementation(
      (config: Parameters<typeof database.query.workspaces.findMany>[0]) => {
        query = database.query.workspaces.findMany(config).toSQL();
        return [{ orgId: "org_disabled", name: "Disabled" }];
      },
    );
    expect(await caller.listAvailable()).toEqual([{ orgId: "org_disabled", name: "Disabled" }]);
    expect(mocks.memberships).toHaveBeenCalledWith("session-user");
    expect(query?.params).toEqual(["org_live", "org_disabled", "org_deleted", "org_missing"]);
    expect(query?.sql).toContain("`org_id` in (?, ?, ?, ?)");
    expect(query?.sql).toContain("`deleted_at_m` is null");
    expect(query?.sql).not.toContain("enabled");
    pool.end();
  });

  it("does not query workspace details without active memberships", async () => {
    mocks.memberships.mockResolvedValue({ data: [membership("org_inactive", "inactive")] });
    expect(await caller.listAvailable()).toEqual([]);
    expect(mocks.findMany).not.toHaveBeenCalled();
  });

  it.each(["memberships", "findMany"] as const)(
    "does not hide %s failures as empty lists",
    async (operation) => {
      mocks[operation].mockRejectedValue(new Error("unavailable"));
      await expect(caller.listAvailable()).rejects.toThrow("unavailable");
    },
  );
});
