import { eq, mysqlDrizzle, schema } from "@unkey/db";
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
      where: (table, { isNotNull, gt, and }) =>
        and(
          gt(table.id, cursor),
          isNotNull(table.refillInterval),
          isNotNull(table.refillAmount),
          isNotNull(table.remaining),
        ),
      limit: 1000,
      orderBy: (table, { asc }) => asc(table.id),
    });

    cursor = keys.at(-1)?.id ?? "";
    console.info({ cursor, keys: keys.length });

    for (const key of keys) {
      if (key.refillInterval === "monthly") {
        if (key.refillDay === null) {
          key.refillDay = 1;
        }
      }
      if (key.refillInterval === "daily") {
        key.refillDay = null;
      }
      const changed = await db
        .update(schema.keys)
        .set({ refillDay: key.refillDay, refillInterval: key.refillInterval })
        .where(eq(schema.keys.id, key.id));
      if (changed) {
        keyChanges++;
      }
    }
  } while (cursor);
  await conn.end();
  console.info("Migration completed. Keys Changed", keyChanges);
}

main();
