import { type Key, connectDatabase, eq, schema } from "@/lib/db";
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
    const date = _payload.ts;
    const lastDayOfMonth = new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate();
    const newDate = date.getDate();
    const today = date.getUTCDate();
    const db = connectDatabase();
    const tb = new Tinybird(env().TINYBIRD_TOKEN);
    let keys: Key[];

    if (newDate !== lastDayOfMonth) {
      keys = await io.runTask("list keys for refill", () =>
        db.query.keys
          .findMany({
            where: (table, { isNotNull, isNull, eq, and, gt, or }) =>
              and(
                isNull(table.deletedAt),
                isNotNull(table.refillAmount),
                gt(table.refillAmount, table.remaining),
                or(isNull(table.refillDay), eq(table.refillDay, today)),
              ),
          })
          .catch((error) => {
            console.error(`Query on keys failed, ${error}`);
            throw new Error(`Query on keys failed, ${error}`);
          }),
      );
    } else {
      keys = await io.runTask("list keys for end of month refill", () =>
        db.query.keys
          .findMany({
            where: (table, { isNotNull, isNull, gte, and, gt, or }) =>
              and(
                isNull(table.deletedAt),
                isNotNull(table.refillAmount),
                gt(table.refillAmount, table.remaining),
                or(isNull(table.refillDay), gte(table.refillDay, lastDayOfMonth)),
              ),
          })
          .catch((error) => {
            console.error(`Query on keys failed, ${error}`);
            throw new Error(`Query on keys failed, ${error}`);
          }),
      );
    }

    io.logger.info(`found ${keys.length} keys with refill set for today`);
    for (const key of keys) {
      await io.runTask(`refill for ${key.id}`, async () => {
        await db
          .update(schema.keys)
          .set({
            remaining: key.refillAmount,
            lastRefillAt: new Date(),
          })
          .where(eq(schema.keys.id, key.id))
          .catch((error) => {
            console.error(`Failed to update remaining ${key.id}`);
            console.error(`With error ${error}`);
          });
      });

      await io.runTask(`create audit log refilling ${key.id}`, async () => {
        await tb
          .ingestAuditLogs({
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
          })
          .catch((_error) => {
            console.error(`failed to update Audit Logs for ${key.id}`);
          });
      });
    }
    return {
      refillKeyIds: keys.map((k) => k.id),
    };
  },
});
