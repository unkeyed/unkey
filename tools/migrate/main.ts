import { and, asc, eq, gt, isNotNull, mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  const tables = [
    schema.keys,
    schema.keyAuth,
    schema.apis,
    schema.gateways,
    schema.gatewayHeaderRewrites,
    schema.ratelimitOverrides,
    schema.ratelimitNamespaces,
    schema.vercelBindings,
    schema.vercelIntegrations,
    schema.workspaces,
  ];

  for (const table of tables) {
    let cursor: string | undefined = "";
    do {
      const rows = await db
        .select()
        .from(table)
        .where(and(gt(table.id, cursor), isNotNull(table.deletedAt)))
        .limit(10)
        .orderBy(asc(table.id))
        .execute();
      cursor = rows.at(-1)?.id;
      console.info({ cursor, rows: rows.length });

      for (const row of rows) {
        const start = performance.now();
        await db.delete(table).where(eq(table.id, row.id));
        const latency = Math.round(performance.now() - start);
        console.info(`${latency} ms`);
      }
    } while (cursor);
  }
}

main();
