import type { FieldPacket, QueryOptions, QueryResult } from "mysql2";
import { expect, it, vi } from "vitest";
import { loadWorkspace } from "./load-workspace";

const mocks = vi.hoisted(() => ({
  query:
    vi.fn<(options: QueryOptions, values: unknown[]) => Promise<[QueryResult, FieldPacket[]]>>(),
}));

vi.mock("@/lib/db", async () => {
  const { drizzle, schema } = await import("@unkey/db");
  const { createPool } = await import("mysql2");
  const pool = createPool({}).promise();
  vi.spyOn(pool, "query").mockImplementation(mocks.query);
  return { db: drizzle(pool, { schema, mode: "default" }) };
});

it("loads the session workspace and scoped flags with one regular joined query", async () => {
  mocks.query.mockResolvedValue([[], []]);
  expect(await loadWorkspace("org_owned")).toBeUndefined();
  expect(mocks.query).toHaveBeenCalledTimes(1);
  const call = mocks.query.mock.calls[0];
  expect(call?.[1]).toEqual(["org_owned"]);
  const sql = call?.[0].sql;
  expect(sql).toContain("left join `flags` on true");
  expect(sql).toContain("`workspace_flag_overrides`.`workspace_id` = `workspaces`.`id`");
  expect(sql).toContain("`workspace_flag_overrides`.`flag_id` = `flags`.`id`");
  expect(sql).toContain("`workspaces`.`org_id` = ?");
  expect(sql).toContain("`workspaces`.`deleted_at_m` is null");
  expect(sql).not.toContain("JSON_OBJECTAGG");
});
