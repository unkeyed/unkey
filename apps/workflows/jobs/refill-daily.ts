import { connectDatabase, eq, schema } from "@/lib/db";
import { client } from "@/trigger";
import { cronTrigger } from "@trigger.dev/sdk";

client.defineJob({
  id: "refill.daily",
  name: "Daily refill",
  version: "0.0.1",
  trigger: cronTrigger({
    cron: "0 0 * * *", // Daily at midnight UTC
  }),

  run: async (_payload, io, _ctx) => {
    const db = connectDatabase();

    const keys = await io.runTask("list keys", () =>
      db.query.keys.findMany({
        where: (table, { isNotNull, eq, gt, and }) =>
          and(
            isNotNull(table.refillInterval),
            isNotNull(table.refillIncrement),
            eq(table.refillInterval, "daily"),
            gt(table.refillIncrement, 0),
          ),
      }),
    );
    io.logger.info(`found ${keys.length} keys with daily refill set`);
    // const keysWithRefill = keys.length;
    for (const key of keys) {
      await io.runTask(`refill for ${key.id}`, async () => {
        await db
          .update(schema.keys)
          .set({
            remaining:
              key.remaining !== null
                ? key.refillIncrement !== null
                  ? key.remaining + key.refillIncrement
                  : key.refillIncrement
                : null,
            lastRefillAt: new Date(),
          })
          .where(eq(schema.keys.id, key.id));
      });

      return {
        keyIds: keys.map((k) => k.id),
      };
    }
  },
});
