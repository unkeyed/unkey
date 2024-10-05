import { connectDatabase, eq, gt, lte, schema } from "@/lib/db";
import { client } from "@/trigger";
import { cronTrigger } from "@trigger.dev/sdk";
import { newId } from "@unkey/id";
client.defineJob({
  id: "refill.monthly",
  name: "Monthly refill",
  version: "0.0.1",
  trigger: cronTrigger({
    cron: "0 0 0 1 1/1 ? *", // First of each month at 00:00 UTC
  }),

  run: async (_payload, io, _ctx) => {
    const db = connectDatabase();
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
    for (const key of keys) {
      const bucket = await io.runTask(`get bucket for ${key.workspaceId}`, async () => {
        return await db.query.auditLogBucket.findFirst({
          where: (table, { eq, and }) =>
            and(eq(table.workspaceId, key.workspaceId), eq(table.name, "unkey_mutations")),
        });
      });
      if (!bucket) {
        io.logger.error(`bucket for ${key.workspaceId} does not exist`);
        continue;
      }

      await io.runTask(`refill for ${key.id}`, async () => {
        await db.transaction(async (tx) => {
          await tx
            .update(schema.keys)
            .set({
              remaining: key.refillAmount,
              lastRefillAt: new Date(),
            })
            .where(eq(schema.keys.id, key.id));

          const auditLogId = newId("auditLog");
          await tx.insert(schema.auditLog).values({
            id: auditLogId,
            workspaceId: key.workspaceId,
            bucketId: bucket.id,
            time: Date.now(),
            event: "key.update",
            actorId: "trigger",
            actorType: "system",
            display: `Refilled ${key.id} to ${key.refillAmount}`,
          });
          await tx.insert(schema.auditLogTarget).values([
            {
              type: "workspace",
              id: key.workspaceId,
              workspaceId: key.workspaceId,
              bucketId: bucket.id,
              auditLogId,
              displayName: `workspace ${key.workspaceId}`,
            },
            {
              type: "key",
              id: key.id,
              workspaceId: key.workspaceId,
              bucketId: bucket.id,
              auditLogId,
              displayName: `key ${key.id}`,
            },
          ]);
        });
      });
    }
    return {
      keyIds: keys.map((k) => k.id),
    };
  },
});
