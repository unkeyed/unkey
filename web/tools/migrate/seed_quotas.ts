import { mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  let cursor = "";
  do {
    const workspaces = await db.query.workspaces.findMany({
      where: (table, { gt }) => gt(table.id, cursor),

      with: { quotas: true },
      limit: 1000,
      orderBy: (table, { asc }) => asc(table.id),
    });
    cursor = workspaces.at(-1)?.id ?? "";

    for (const workspace of workspaces) {
      if (workspace.quota) {
        continue;
      }

      if (workspace.stripeCustomerId) {
        await db.insert(schema.quotas).values({
          workspaceId: workspace.id,
          team: true,
          requestsPerMonth: 250_000,
          logsRetentionDays: 30,
          auditLogsRetentionDays: 90,
        });
      } else {
        await db.insert(schema.quotas).values({
          workspaceId: workspace.id,
          team: false,
          requestsPerMonth: 150_000,
          logsRetentionDays: 7,
          auditLogsRetentionDays: 30,
        });
      }
    }
  } while (cursor);
  await conn.end();
}

main();
