import { createDeploymentOwnershipCache } from "@/lib/trpc/deployment-ownership";
import { TRPCError } from "@trpc/server";
import { drizzle, schema } from "@unkey/db";
import { createPool } from "mysql2";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { requireOwnedDeployment } from "./require-owned-deployment";

const findFirst = vi.hoisted(() => vi.fn());

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      deployments: { findFirst },
    },
  },
}));

const owned = {
  id: "dep_1",
  projectId: "prj_1",
  environmentId: "env_1",
};

describe("requireOwnedDeployment", () => {
  beforeEach(() => {
    findFirst.mockReset();
  });

  it("queries deployments once when the same deployment is checked more than once", async () => {
    findFirst.mockResolvedValue(owned);
    const cache = createDeploymentOwnershipCache();

    const [first, second] = await Promise.all([
      requireOwnedDeployment(cache, "ws_1", "dep_1", "Resource not found"),
      requireOwnedDeployment(cache, "ws_1", "dep_1", "Deployment not found"),
    ]);

    expect(findFirst).toHaveBeenCalledOnce();
    expect(first).toEqual(owned);
    expect(second).toBe(first);
  });

  it("keeps a missing deployment as not found without querying again", async () => {
    findFirst.mockResolvedValue(undefined);
    const cache = createDeploymentOwnershipCache();

    const missing = requireOwnedDeployment(cache, "ws_1", "dep_missing", "Resource not found");
    await expect(missing).rejects.toBeInstanceOf(TRPCError);
    await expect(
      requireOwnedDeployment(cache, "ws_1", "dep_missing", "Deployment not found"),
    ).rejects.toMatchObject({
      code: "NOT_FOUND",
      message: "Deployment not found",
    });
    expect(findFirst).toHaveBeenCalledOnce();
  });

  it("selects the deployment only when it belongs to the workspace", async () => {
    const pool = createPool({});
    const database = drizzle(pool, { schema, mode: "default" });
    let query: { sql: string; params: unknown[] } | undefined;
    findFirst.mockImplementation(
      (config: Parameters<typeof database.query.deployments.findFirst>[0]) => {
        query = database.query.deployments.findFirst(config).toSQL();
        return Promise.resolve(undefined);
      },
    );

    await expect(
      requireOwnedDeployment(
        createDeploymentOwnershipCache(),
        "ws_1",
        "dep_1",
        "Resource not found",
      ),
    ).rejects.toMatchObject({ code: "NOT_FOUND", message: "Resource not found" });

    expect(query?.sql).toContain("select `id`, `project_id`, `environment_id`");
    expect(query?.sql).toContain("`deployments`.`id` = ? and `deployments`.`workspace_id` = ?");
    expect(query?.sql).toContain("limit ?");
    expect(query?.params).toEqual(["dep_1", "ws_1", 1]);
    pool.end();
  });
});
