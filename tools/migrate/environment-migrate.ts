import { eq, mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={} `,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  let cursor = "";
  do {
    const keys = await db.query.keys.findMany({
      where: (table, { isNotNull, gt, eq, and }) =>
        and(
          gt(table.id, cursor),
          eq(table.workspaceId, "ws_39g5eLLQTX8bVdbsGK9Dke"),
          isNotNull(table.environment),
        ),
      limit: 1000,
      orderBy: (table, { asc }) => asc(table.id),
    });

    cursor = keys.at(-1)?.id ?? "";
    console.info({ cursor, keys: keys.length });

    if (keys.length === 0) {
      break;
    }

    for (const key of keys) {
      if (!key.environment) {
        continue;
      }

      console.log(JSON.stringify(key));

      const meta = key.meta ? JSON.parse(key.meta) : {};
      if (meta.environment) {
        console.error(`Key ${key.id} has environment ${meta.environment}`);
        continue;
      }

      meta.environment = key.environment;

      const newMeta = JSON.stringify(meta);

      await db.update(schema.keys).set({ meta: newMeta }).where(eq(schema.keys.id, key.id));
    }
  } while (cursor);
  await conn.end();
  console.info("Migration completed. Keys Changed");
}

main();
