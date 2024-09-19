import { connectDatabase, eq, gt, lte, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Tinybird } from "@/lib/tinybird";
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
    const date = new Date(Date.now());
    const today = date.getDate();
    const db = connectDatabase();
    const tb = new Tinybird(env().TINYBIRD_TOKEN);
    const tDay = new Date();
    const tMonth = new Date();
    tDay.setUTCHours(tDay.getUTCHours() - 24);
    tMonth.setUTCMonth(tMonth.getUTCMonth() - 1);

    const dailyKeys = await io.runTask("list keys Daily", () =>
      db.query.keys.findMany({
        where: (table, { isNotNull, isNull, eq, and, gt, or }) =>
          and(
            isNull(table.deletedAt),
            isNotNull(table.refillInterval),
            isNotNull(table.refillAmount),
            eq(table.refillInterval, "daily"),
            gt(table.refillAmount, table.remaining),
            or(
              isNull(table.lastRefillAt),
              lte(table.lastRefillAt, tDay), // Check if more than 24 hours have passed
            ),
          ),
      }),
    );
    const monthlyKeys = await io.runTask("list keys Monthly", () =>
      db.query.keys.findMany({
        where: (table, { isNotNull, isNull, eq, and, or }) =>
          and(
            isNull(table.deletedAt),
            isNotNull(table.refillInterval),
            isNotNull(table.refillAmount),
            eq(table.refillInterval, "monthly"),
            eq(table.refillDay, today),
            gt(table.refillAmount, table.remaining),
            or(
              isNull(table.lastRefillAt),
              lte(table.lastRefillAt, tMonth), // Check if more than 1 Month has passed
            ),
          ),
      }),
    );
    io.logger.info(`found ${dailyKeys.length} keys with daily refill set`);
    io.logger.info(`found ${monthlyKeys.length} keys with monthly refill set`);
    // const keysWithRefill = keys.length;
    for (const key of dailyKeys) {
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
    for (const key of monthlyKeys) {
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
      daileyKeyIds: dailyKeys.map((k) => k.id),
      monthlyKeyIds: monthlyKeys.map((k) => k.id),
    };
  },
});
