import { initTRPC } from "@trpc/server";
import { type SQL, drizzle, schema } from "@unkey/db";
import { createPool } from "mysql2";
import { beforeEach, expect, it, vi } from "vitest";
import { getCurrentWorkspace } from "./getCurrent";

const mocks = vi.hoisted(() => ({
  findFirst: vi.fn(),
  join: vi.fn<
    (
      table: unknown,
      on: SQL,
    ) => Promise<
      {
        slug: string;
        defaultValue: boolean;
        overrideValue: boolean | null;
      }[]
    >
  >(),
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: { workspaces: { findFirst: mocks.findFirst } },
    select: () => ({ from: () => ({ leftJoin: mocks.join }) }),
  },
}));

vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  return { protectedProcedure: initTRPC.create().procedure };
});

const router = initTRPC
  .context<{
    workspace?: { id: string };
    tenant: { id: string } | null;
  }>()
  .create()
  .router({ getCurrent: getCurrentWorkspace });

beforeEach(() => {
  vi.resetAllMocks();
});

it("loads effective flags with the current workspace, preserving false overrides", async () => {
  mocks.join.mockResolvedValue([
    { slug: "preview", defaultValue: true, overrideValue: false },
    { slug: "routing", defaultValue: false, overrideValue: true },
    { slug: "insights", defaultValue: true, overrideValue: null },
    { slug: "private", defaultValue: false, overrideValue: null },
  ]);
  const result = await router
    .createCaller({ workspace: { id: "ws_owned" }, tenant: { id: "org_session" } })
    .getCurrent();
  expect(result).toMatchObject({
    id: "ws_owned",
    flags: { preview: false, routing: true, insights: true, private: false },
  });

  const pool = createPool({});
  try {
    const predicate = mocks.join.mock.calls[0]?.[1];
    expect(predicate).toBeDefined();
    const query = drizzle(pool)
      .select()
      .from(schema.flags)
      .leftJoin(schema.workspaceFlagOverrides, predicate)
      .toSQL();
    expect(query.params).toEqual(["ws_owned"]);
    expect(query.sql).toContain("`workspace_flag_overrides`.`workspace_id` = ?");
    expect(query.sql).toContain("`workspace_flag_overrides`.`flag_id` = `flags`.`id`");
  } finally {
    pool.end();
  }
});

it("includes flags when the workspace is loaded by the fallback query", async () => {
  mocks.findFirst.mockResolvedValue({ id: "ws_fallback" });
  mocks.join.mockResolvedValue([{ slug: "preview", defaultValue: false, overrideValue: true }]);
  const result = await router.createCaller({ tenant: { id: "org_session" } }).getCurrent();
  expect(result).toMatchObject({ id: "ws_fallback", flags: { preview: true } });
});

it("does not turn a failed flag lookup into an empty flag map", async () => {
  mocks.join.mockRejectedValue(new Error("Flags unavailable"));
  const caller = router.createCaller({
    workspace: { id: "ws_owned" },
    tenant: { id: "org_session" },
  });
  await expect(caller.getCurrent()).rejects.toThrow("Flags unavailable");
});
