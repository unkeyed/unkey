import assert from "node:assert/strict";
import test from "node:test";
import { drizzle, schema } from "@unkey/db";
import { createPool } from "mysql2";
import { missingK8sNamespace } from "./workspace-k8s-namespace";

test("selects null and empty Kubernetes namespaces", () => {
  const pool = createPool({});

  try {
    const query = drizzle(pool)
      .select({ pk: schema.workspaces.pk })
      .from(schema.workspaces)
      .where(missingK8sNamespace)
      .toSQL();

    assert.match(query.sql, /`k8s_namespace` is null or .*`k8s_namespace` = \?/);
    assert.deepEqual(query.params, [""]);
  } finally {
    pool.end();
  }
});
