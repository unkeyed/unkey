import { ClickHouse } from "@unkey/clickhouse";
import { mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";
import { z } from "zod";
async function main() {
  const ch = new ClickHouse({
    url: process.env.CLICKHOUSE_URL,
  });

  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  console.log("starting");

  const rows = await ch.querier.query({
    query: `
    SELECT
      workspace_id,
      splitByChar('?', path, 1)[1] as path
    FROM default.api_requests_per_day_v2
    WHERE startsWith(path, '/v1/')
    AND workspace_id != ''
    AND workspace_id != 'ws_2vUFz88G6TuzMQHZaUhXADNyZWMy' // filter out special workspaces
    AND time >= (now() - INTERVAL 7 DAY)
    GROUP BY workspace_id, path`,
    schema: z.object({
      workspace_id: z.string(),
      path: z.string(),
    }),
  })({});
  if (rows.err) {
    console.error(rows.err);
    process.exit(1);
  }

  console.log(
    `Found ${
      new Set(rows.val.map((r) => r.workspace_id)).size
    } workspaces across ${rows.val.length} paths`,
  );
  const workspaceToPaths = new Map<string, string[]>();
  for (const row of rows.val) {
    if (row.workspace_id.startsWith("test_")) {
      continue;
    }
    const paths = workspaceToPaths.get(row.workspace_id) || [];
    paths.push(row.path);
    workspaceToPaths.set(row.workspace_id, paths);
  }

  const workspaces = [];

  for (const [workspaceId, paths] of workspaceToPaths.entries()) {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { eq }) => eq(table.id, workspaceId),
    });
    if (!workspace) {
      console.error(`Workspace ${workspaceId} not found`);
      continue;
    }
    workspaces.push({
      id: workspace.id,
      name: workspace.name,
      sub: workspace.stripeSubscriptionId,
    });
  }

  console.table(workspaces);
}

main();
