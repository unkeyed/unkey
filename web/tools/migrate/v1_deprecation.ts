import { ClickHouse } from "@unkey/clickhouse";
import { type Workspace, mysqlDrizzle, schema } from "@unkey/db";
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
    AND time >= (now() - INTERVAL 1 DAY)
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

  const workspaces: Record<string, Workspace> = {};

  for (const row of rows.val) {
    if (workspaces[row.workspace_id]) {
      continue;
    }

    const workspace = await db.query.workspaces.findFirst({
      where: (table, { eq }) => eq(table.id, row.workspace_id),
    });
    if (!workspace) {
      console.error(`Workspace ${row.workspace_id} not found`);
      continue;
    }
    workspaces[workspace.id] = workspace;
  }

  console.table(
    Object.values(workspaces).map((ws) => ({
      id: ws.id,
      name: ws.name,
      org: ws.orgId,
      sub: ws.stripeSubscriptionId,
    })),
  );
}

main();
