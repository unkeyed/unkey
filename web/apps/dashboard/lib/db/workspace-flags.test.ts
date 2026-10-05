import { drizzle, schema } from "@unkey/db";
import { createPool } from "mysql2";
import { expect, it } from "vitest";
import { workspaceFlagExtras } from "./workspace-flags";

it("includes default flags and workspace-scoped overrides in the workspace SQL statement", () => {
  const pool = createPool({});
  try {
    const query = drizzle(pool, { schema, mode: "default" })
      .query.workspaces.findFirst({
        columns: { id: true },
        extras: workspaceFlagExtras,
        where: (workspace, { eq }) => eq(workspace.orgId, "org_owned"),
      })
      .toSQL();
    expect(query.sql).toContain("JSON_OBJECTAGG(f.slug, COALESCE(o.value, f.default_value))");
    expect(query.sql).toContain("LEFT JOIN workspace_flag_overrides AS o");
    expect(query.sql).toContain("o.flag_id = f.id AND o.workspace_id = `workspaces`.`id`");
    expect(query.sql).toContain("JSON_OBJECT()");
    expect(query.params).toContain("org_owned");
  } finally {
    pool.end();
  }
});
