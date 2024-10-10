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

  run: async (_payload, io, _ctx) => {
    const db = connectDatabase();
    const t = new Date();
    t.setUTCHours(t.getUTCHours() - 24);
    const BUCKET_NAME = "unkey_mutations";

    type Key = `${string}::${string}`;
    type BucketId = string;
    const bucketCache = new Map<Key, BucketId>();
    const keys = await io.runTask("list keys", () =>
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
              lte(table.lastRefillAt, t), // Check if more than 24 hours have passed
            ),
          ),
      }),
    );
    io.logger.info(`found ${keys.length} keys with daily refill set`);

    for (const key of keys) {
      const cacheKey: Key = `${key.workspaceId}::${BUCKET_NAME}`;
      let bucketId = "";
      const cachedBucketId = bucketCache.get(cacheKey);
      if (cachedBucketId) {
        bucketId = cachedBucketId;
      } else {
        const bucket = await db.query.auditLogBucket.findFirst({
          where: (table, { eq, and }) =>
            and(eq(table.workspaceId, key.workspaceId), eq(table.name, BUCKET_NAME)),
          columns: {
            id: true,
          },
        });

        if (bucket) {
          bucketId = bucket.id;
        } else {
          bucketId = newId("auditLogBucket");
          await db.insert(schema.auditLogBucket).values({
            id: bucketId,
            workspaceId: key.workspaceId,
            name: BUCKET_NAME,
          });
        }
      }

      bucketCache.set(cacheKey, bucketId);
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
            bucketId: bucketId,
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
              bucketId: bucketId,
              auditLogId,
              displayName: `workspace ${key.workspaceId}`,
            },
            {
              type: "key",
              id: key.id,
              workspaceId: key.workspaceId,
              bucketId: bucketId,
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
