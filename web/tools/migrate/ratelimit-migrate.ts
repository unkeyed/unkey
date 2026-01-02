import { mysqlDrizzle, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  let cursor = "";
  do {
    const keys = await db.query.keys.findMany({
      where: (table, { isNotNull, gt, and }) =>
        and(
          gt(table.id, cursor),
          isNotNull(table.ratelimitLimit),
          isNotNull(table.ratelimitDuration),
        ),
      with: {
        ratelimits: true,
      },
      limit: 1000,
      orderBy: (table, { asc }) => asc(table.id),
    });

    cursor = keys.at(-1)?.id ?? "";
    console.info({ cursor, keys: keys.length });

    if (keys.length === 0) {
      break;
    }

    for (const key of keys) {
      if (key.ratelimits.find((rl) => rl.name === "default" && rl.autoApply)) {
        break;
      }

      console.info("Updating", key.id, key.ratelimitLimit, key.ratelimitDuration);
      await db
        .insert(schema.ratelimits)
        .values({
          id: newId("ratelimit"),
          keyId: key.id,
          limit: key.ratelimitLimit!,
          duration: key.ratelimitDuration!,
          workspaceId: key.workspaceId,
          name: "default",
          autoApply: true,
        })
        .onDuplicateKeyUpdate({
          set: {
            limit: key.ratelimitLimit!,
            duration: key.ratelimitDuration!,
          },
        });
    }
  } while (cursor);
  await conn.end();
  console.info("Migration completed. Keys Changed");
}

main();
