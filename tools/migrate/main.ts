import { eq, schema } from "@unkey/db";

import { Client } from "@planetscale/database";
import { drizzle } from "drizzle-orm/planetscale-serverless";

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

  const keys = await db.query.keys.findMany({
    // where: (table, { eq, isNotNull }) => isNotNull(table.ratelimitType),
  });

  console.info("found", keys.length, "keys");
  while (keys.length > 0) {
    console.info(keys.length);

    const chunk = keys.splice(0, 100);
    await Promise.all(
      chunk.map(async (key) => {
        await db
          .update(schema.keys)
          .set({
            // ratelimitAsync: key.ratelimitType === "fast",
            // ratelimitLimit: key.ratelimitRefillRate,
            // ratelimitDuration: key.ratelimitRefillInterval,
          })
          .where(eq(schema.keys.id, key.id));
      }),
    );
  }
}

main();
