import { eq, isNull, mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  let cursor = "";
  let keyChanges = 0;
  do {
    const keys = await db.query.keys.findMany({
      where: (table, { isNotNull, gt, and, or }) =>
        and(
          gt(table.id, cursor),
          isNotNull(table.refillAmount),
          isNotNull(table.remaining),
          or(
            and(eq(table.refillInterval, "monthly"), isNull(table.refillDay)),
            and(eq(table.refillInterval, "daily"), isNotNull(table.refillDay)),
          ),
        ),
      limit: 1000,
      orderBy: (table, { asc }) => asc(table.id),
    });

    cursor = keys.at(-1)?.id ?? "";
    console.info({ cursor, keys: keys.length });

    for (const key of keys) {
      if (key.refillInterval === "monthly" && key.refillDay === null) {
        const changed = await db
          .update(schema.keys)
          .set({ refillDay: 1 })
          .where(eq(schema.keys.id, key.id));
        if (changed) {
          keyChanges++;
        }
      } else if (key.refillInterval === "daily" && key.refillDay !== null) {
        const changed = await db
          .update(schema.keys)
          .set({ refillDay: null })
          .where(eq(schema.keys.id, key.id));
        if (changed) {
          keyChanges++;
        }
      }
    }
  } while (cursor);
  await conn.end();
  console.info("Migration completed. Keys Changed", keyChanges);
}

main();
