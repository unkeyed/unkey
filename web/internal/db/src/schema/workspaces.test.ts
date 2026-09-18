import { drizzle } from "drizzle-orm/mysql2";
import { createPool } from "mysql2";
import { describe, expect, it } from "vitest";
import { workspaces } from "./workspaces";

describe("workspaces", () => {
  it("assigns a Kubernetes namespace when one is not provided", () => {
    const pool = createPool({});
    const database = drizzle(pool);

    try {
      const query = database
        .insert(workspaces)
        .values({
          id: "ws_test",
          orgId: "org_test",
          name: "Test Workspace",
          slug: "test-workspace",
          betaFeatures: {},
        })
        .toSQL();

      expect(query.sql).toContain("`k8s_namespace`");
      expect(query.params).toEqual(
        expect.arrayContaining([expect.stringMatching(/^[a-z][a-z0-9]{21}$/)]),
      );
    } finally {
      pool.end();
    }
  });
});
