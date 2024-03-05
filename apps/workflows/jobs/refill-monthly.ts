import { connectDatabase, eq, gt, lte, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Tinybird } from "@/lib/tinybird";
import { client } from "@/trigger";
import { cronTrigger } from "@trigger.dev/sdk";

client.defineJob({
  id: "refill.monthly",
  name: "Monthly refill",
  version: "0.0.1",
  trigger: cronTrigger({
    cron: "0 0 0 1 1/1 ? *", // First of each month at 00:00 UTC
  }),

  run: async (_payload, io, _ctx) => {
    const db = connectDatabase();
    const tb = new Tinybird(env().TINYBIRD_TOKEN);
    const t = new Date();
    t.setUTCMonth(t.getUTCMonth() - 1);

    const keys = await io.runTask("list keys", () =>
      db.query.keys.findMany({
        where: (table, { isNotNull, isNull, eq, and, or }) =>
          and(
            isNull(table.deletedAt),
            isNotNull(table.refillInterval),
            isNotNull(table.refillAmount),
            eq(table.refillInterval, "monthly"),
            gt(table.refillAmount, table.remaining),
            or(
              isNull(table.lastRefillAt),
              lte(table.lastRefillAt, t), // Check if more than 1 Month has passed
            ),
          ),
      }),
    );
    io.logger.info(`found ${keys.length} keys with monthly refill set`);
    // const keysWithRefill = keys.length;
    for (const key of keys) {
      await io.runTask(`refill for ${key.id}`, async () => {
        await db
          .update(schema.keys)
          .set({
            remaining: key.refillAmount,
            lastRefillAt: new Date(),
          })
          .where(eq(schema.keys.id, key.id));
      });
      await io.runTask(`create audit log refilling ${key.id}`, async () => {
        await tb.ingestAuditLogs({
          workspaceId: key.workspaceId,
          event: "key.update",
          actor: {
            type: "system",
            id: "trigger",
          },
          description: `Refilled ${key.id} to ${key.refillAmount}`,
          resources: [
            {
              type: "workspace",
              id: key.workspaceId,
            },
            {
              type: "key",
              id: key.id,
            },
          ],
          context: {
            location: "trigger",
          },
        });
      });
    }
    return {
      keyIds: keys.map((k) => k.id),
    };
  },
});
