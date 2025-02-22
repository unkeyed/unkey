import { and, asc, eq, gt, isNull, mysqlDrizzle, schema, sql } from "@unkey/db";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(process.env.DRIZZLE_DATABASE_URL!);

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  for (const table of [
    schema.apis,
    schema.keyAuth,
    schema.keys,
    schema.permissions,
    schema.ratelimitNamespaces,
    schema.ratelimitOverrides,
    schema.roles,
    schema.vercelBindings,
    schema.vercelIntegrations,
    schema.workspaces,
  ]) {
    const count = await db
      .select({ count: sql<string>`count(*)` })
      .from(table)
      .where(isNull(table.createdAtM))
      .then((res) => res.at(0)?.count ?? 0);
    console.log({ count });

    let processed = 0;
    let cursor = "";
    do {
      const rows = await db
        .select()
        .from(table)
        .where(
          and(
            isNull(table.createdAtM),

            gt(table.id, cursor),
          ),
        )
        .orderBy(asc(table.id))
        .limit(100);

      cursor = rows.at(-1)?.id ?? "";
      console.info({ cursor, rows: rows.length, processed, count });

      await Promise.all(
        rows.map(async (row) => {
          await db
            .update(table)
            .set({
              createdAtM: new Date(row.createdAt ?? 0).getTime() ?? 0,
              updatedAtM: "updatedAt" in row ? row.updatedAt?.getTime() ?? null : null,
              deletedAtM: row.deletedAt?.getTime() ?? null,
            })
            .where(eq(table.id, row.id));
        }),
      );
      processed += rows.length;
    } while (cursor);
  }
}

main().then(() => process.exit(0));
