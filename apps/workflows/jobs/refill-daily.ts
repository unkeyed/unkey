import { connectDatabase, eq, schema } from "@/lib/db";
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
    const today = date.getUTCDate();
    const db = connectDatabase();
    const tb = new Tinybird(env().TINYBIRD_TOKEN);

    const keys = await io.runTask("list keys for refill", () =>
      db.query.keys.findMany({
        where: (table, { isNotNull, isNull, eq, and, gt, or }) =>
          and(
            isNull(table.deletedAt),
            isNotNull(table.refillAmount),
            gt(table.refillAmount, table.remaining),
            or(
              isNull(table.refillDay),
              eq(table.refillDay, today),
            )
          ),
      }),
    );

    io.logger.info(`found ${keys.length} keys with refill set for today`);
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
      refillKeyIds: keys.map((k) => k.id),
    };
  },
});
