import { and, asc, eq, gt, isNotNull, mysqlDrizzle, schema } from "@unkey/db";
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
      where: (table, { isNotNull, gt, and, isNull }) =>
        and(gt(table.id, cursor), isNotNull(table.ownerId), isNull(table.identityId)),
      limit: 1000,
      orderBy: (table, { asc }) => asc(table.id),
    });

    cursor = keys.at(-1)?.id ?? "";
    console.info({ cursor, keys: keys.length });

    for (const key of keys) {
      let identity: { id: string } | undefined = await db.query.identities.findFirst({
        where: (table, { eq }) => eq(table.externalId, key.ownerId!),
      });
      if (!identity) {
        const id = newId("identity");
        console.log("Creating new identity", id, key.ownerId);
        await db.insert(schema.identities).values({
          id,
          workspaceId: key.workspaceId,
          externalId: key.ownerId!,
        });
        identity = {
          id,
        };
      }
      console.log("connecting", identity.id, key.id);
      await db
        .update(schema.keys)
        .set({ identityId: identity.id })
        .where(eq(schema.keys.id, key.id));
    }
  } while (cursor);
}

main();
