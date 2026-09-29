import { beforeEach, describe, expect, it, vi } from "vitest";

const mockConnection = {
  query: vi.fn(async (..._args: unknown[]) => [[]]),
  execute: vi.fn(async (..._args: unknown[]) => [[]]),
  release: vi.fn(),
};

const mockPool = {
  query: vi.fn((..._args: unknown[]) => Promise.resolve([[]])),
  execute: vi.fn((..._args: unknown[]) => Promise.resolve([[]])),
  getConnection: vi.fn(async () => mockConnection),
};

vi.mock("mysql2/promise", () => ({
  default: {
    createPool: () => mockPool,
  },
}));

import { createCommentedPool, runWithSqlCommentTags } from "./commented-pool";

const staticTags = {
  application: "unkey",
  service: "dashboard",
  region: "us-east-1",
  releaseSha: "a1b2c3d",
};

const drizzleSelectKeys =
  "select `keys`.`id`, `keys`.`name` from `keys` where `keys`.`workspace_id` = ? limit ?";

describe("createCommentedPool", () => {
  beforeEach(() => {
    mockConnection.execute.mockClear();
    mockConnection.query.mockClear();
    mockPool.query.mockClear();
    mockPool.getConnection.mockClear();
  });

  it("tags queries on connections from getConnection (drizzle transactions)", async () => {
    const commented = createCommentedPool({}, staticTags);
    const conn = await commented.getConnection();

    await runWithSqlCommentTags({ route: "keys.create", source: "trpc" }, async () => {
      await conn.execute("insert into `keys` values (?)");
    });

    expect(mockPool.getConnection).toHaveBeenCalledOnce();
    expect(mockConnection.execute).toHaveBeenCalledOnce();
    const sql = mockConnection.execute.mock.calls[0]?.[0];
    expect(sql).toContain("insert into `keys` values (?)");
    expect(sql).toContain("route='keys.create'");
    expect(sql).toContain("source='trpc'");
    expect(sql).toContain("service='dashboard'");
  });

  it("tags the sql inside the config object drizzle hands to mysql2", async () => {
    const commented = createCommentedPool({}, staticTags);

    await commented.query({
      sql: drizzleSelectKeys,
      values: ["ws_1", 10],
      rowsAsArray: true,
    });

    expect(mockPool.query).toHaveBeenCalledOnce();
    expect(mockPool.query).toHaveBeenCalledWith(
      expect.objectContaining({
        sql: expect.stringContaining(drizzleSelectKeys),
        values: ["ws_1", 10],
        rowsAsArray: true,
      }),
    );
    const sql = mockPool.query.mock.calls[0]?.[0] as { sql: string };
    expect(sql.sql).toContain("service='dashboard'");
    expect(sql.sql).toContain("release_sha='a1b2c3d'");
  });

  it("keeps driver options and positional params on config-object queries", async () => {
    const commented = createCommentedPool({}, staticTags);
    const typeCast = () => undefined;

    await runWithSqlCommentTags({ route: "deploy.envVars.create", source: "trpc" }, async () => {
      await commented.query({ sql: drizzleSelectKeys, typeCast }, ["ws_1", 10]);
    });

    expect(mockPool.query).toHaveBeenCalledWith(
      expect.objectContaining({
        sql: expect.stringContaining(drizzleSelectKeys),
        typeCast,
      }),
      ["ws_1", 10],
    );
    const sql = mockPool.query.mock.calls[0]?.[0] as { sql: string };
    expect(sql.sql).toContain("route='deploy.envVars.create'");
  });

  it("tags config-object queries on pooled connections", async () => {
    const commented = createCommentedPool({}, staticTags);
    const conn = await commented.getConnection();

    await conn.execute({ sql: "delete from `keys` where `id` = ?", values: ["k_1"] });

    expect(mockConnection.execute).toHaveBeenCalledWith(
      expect.objectContaining({
        sql: expect.stringContaining("delete from `keys` where `id` = ?"),
        values: ["k_1"],
      }),
    );
    const sql = mockConnection.execute.mock.calls[0]?.[0] as { sql: string };
    expect(sql.sql).toContain("service='dashboard'");
  });
});
