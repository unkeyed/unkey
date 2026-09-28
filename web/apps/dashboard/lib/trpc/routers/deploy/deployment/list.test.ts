import { DatabaseSync } from "node:sqlite";
import { initTRPC } from "@trpc/server";
import { type SQL, drizzle } from "@unkey/db";
import { deployments } from "@unkey/db/src/schema";
import { afterAll, beforeEach, expect, test, vi } from "vitest";
import { z } from "zod";
import type { DeploymentListSelection } from "./deployment-query-helpers";
import { listDeployments } from "./list";

const selectRows = vi.hoisted(() => vi.fn());
vi.mock("@/lib/db", async () => ({
  ...(await import("@unkey/db")),
  db: {
    select: () => ({
      from: () => ({
        where: (condition: SQL) => ({
          orderBy: (...order: SQL[]) => ({
            limit: (limit: number) => selectRows(condition, order, limit),
          }),
        }),
      }),
    }),
  },
}));
vi.mock("@/lib/trpc/trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  const t = initTRPC.context<{ workspace: { id: string } }>().create();
  return {
    workspaceProcedure: t.procedure,
    ratelimit: { read: {} },
    withRatelimit: () => t.middleware(({ next }) => next()),
  };
});
vi.mock("./enrich-deployment-rows", () => ({
  enrichDeploymentRows: async (_workspaceId: string, rows: DeploymentListSelection[]) => rows,
}));

const sqlite = new DatabaseSync(":memory:");
sqlite.exec(`CREATE TABLE deployments (
  id TEXT, workspace_id TEXT, project_id TEXT, app_id TEXT, status TEXT, created_at INTEGER
)`);
afterAll(() => sqlite.close());

const t = initTRPC.context<{ workspace: { id: string } }>().create();
const caller = t.router({ list: listDeployments }).createCaller({ workspace: { id: "workspace" } });

beforeEach(() => {
  sqlite.exec("DELETE FROM deployments");
  const insert = sqlite.prepare("INSERT INTO deployments VALUES (?, ?, 'project', 'app', ?, ?)");
  for (const [id, workspaceId, status, createdAt] of [
    ["recent-failure", "workspace", "failed", 1_001],
    ["boundary-failure", "workspace", "failed", 1_000],
    ["old-failure", "workspace", "failed", 999],
    ["old-ready", "workspace", "ready", 998],
    ["old-building", "workspace", "building", 997],
    ["other-workspace", "other", "ready", 2_000],
  ] as const) {
    insert.run(id, workspaceId, status, createdAt);
  }
  selectRows.mockImplementation((condition: SQL, order: SQL[], limit: number) => {
    const query = drizzle
      .mock()
      .select({ id: deployments.id, createdAt: deployments.createdAt })
      .from(deployments)
      .where(condition)
      .orderBy(...order)
      .limit(limit)
      .toSQL();
    return sqlite
      .prepare(query.sql)
      .all(...query.params.map((param) => z.union([z.string(), z.number()]).parse(param)))
      .map((row) => ({ id: row.id, createdAt: row.created_at }));
  });
});

test("filters only old failures before pagination and includes the cutoff boundary", async () => {
  const input = { projectId: "project", appId: "app", failedSince: 1_000, limit: 2 };
  const first = await caller.list(input);
  expect(first.deployments.map((row) => row.id)).toEqual(["recent-failure", "boundary-failure"]);
  expect(first.nextCursor).toEqual({ id: "boundary-failure", createdAt: 1_000 });

  const second = await caller.list({ ...input, cursor: first.nextCursor });
  expect(second.deployments.map((row) => row.id)).toEqual(["old-ready", "old-building"]);
  expect(second.nextCursor).toBeNull();
});

test("omitting the cutoff preserves older failures for other list callers", async () => {
  const result = await caller.list({ projectId: "project", appId: "app" });
  expect(result.deployments.map((row) => row.id)).toEqual([
    "recent-failure",
    "boundary-failure",
    "old-failure",
    "old-ready",
    "old-building",
  ]);
});
