import { drizzle, eq, schema } from "@unkey/db";

import { Client } from "@planetscale/database";
import { newId } from "@unkey/id";

async function main() {
  const db = drizzle(
    new Client({
      host: process.env.DATABASE_HOST,
      username: process.env.DATABASE_USERNAME,
      password: process.env.DATABASE_PASSWORD,
    }),
    {
      schema,
    },
  );

  let cursor: string | undefined = undefined;
  do {
    const keys = await db.query.keys.findMany({
      where: (table, { and, isNull, isNotNull }) =>
        and(isNotNull(table.ownerId), isNull(table.identityId)),
      columns: {
        id: true,
        ownerId: true,
        workspaceId: true,
      },
      limit: 10000,
      orderBy: (table, { asc }) => asc(table.id),
    });
    cursor = keys.at(-1)?.id;
    console.info({ cursor, keys: keys.length });

    for (const key of keys) {
      let identity = await db.query.identities.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, key.workspaceId), eq(table.externalId, key.ownerId!)),
      });
      if (!identity) {
        console.info("identity did not exist", key.workspaceId, key.ownerId);
        identity = {
          id: newId("identity"),
          createdAt: Date.now(),
          workspaceId: key.workspaceId,
          externalId: key.ownerId!,
          updatedAt: null,
          meta: {},
          environment: "default",
        };
        await db.insert(schema.identities).values(identity);
      }
      await db
        .update(schema.keys)
        .set({
          identityId: identity.id,
        })
        .where(eq(schema.keys.id, key.id));
    }
  } while (cursor);
}

main();
