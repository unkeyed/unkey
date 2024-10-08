import { connectDatabase, eq, lte, schema } from "@/lib/db";
import { client } from "@/trigger";
import { cronTrigger } from "@trigger.dev/sdk";
import { newId } from "@unkey/id";

client.defineJob({
  id: "refill.daily",
  name: "Daily refill",
  version: "0.0.1",
  trigger: cronTrigger({
    cron: "0 0 * * *", // Daily at midnight UTC
  }),

  run: async (payload, io, _ctx) => {
    const date = payload.ts;
    // Set up last day of month so if refillDay is after last day of month, Key will be refilled today.
    const lastDayOfMonth = new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate();
    const today = date.getUTCDate();
    const db = connectDatabase();

    // If refillDay is after last day of month, refillDay will be today.
    const keys = await db.query.keys.findMany({
      where: (table, { isNotNull, isNull, and, gt, or, eq }) => {
        const baseConditions = and(
          isNull(table.deletedAt),
          isNotNull(table.refillAmount),
          gt(table.refillAmount, table.remaining),
          or(isNull(table.refillDay), eq(table.refillDay, today)),
        );

        if (today === lastDayOfMonth) {
          return and(baseConditions, gt(table.refillDay, today));
        }
        return baseConditions;
      },
    });

    io.logger.info(`found ${keys.length} keys with refill set for today`);
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
      refillKeyIds: keys.map((k) => k.id),
    };
  },
});
