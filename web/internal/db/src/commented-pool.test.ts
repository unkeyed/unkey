import { beforeEach, describe, expect, it, vi } from "vitest";

const mockConnection = {
  query: vi.fn(async () => [[]]),
  execute: vi.fn(async () => [[]]),
  release: vi.fn(),
};

const mockPool = {
  query: vi.fn(),
  execute: vi.fn(),
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

describe("createCommentedPool", () => {
  beforeEach(() => {
    mockConnection.execute.mockClear();
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
});
