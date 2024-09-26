import { type Database, type Transaction, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type { UnkeyAuditLog } from "../tinybird";
import type { MaybeArray } from "../types";

const BUCKET_NAME = "unkey_mutations";

type Key = `${string}::${string}`;
type BucketId = string;
const bucketCache = new Map<Key, BucketId>();

export async function insertAuditLogs(
  db: Transaction | Database,
  logOrLogs: MaybeArray<UnkeyAuditLog>,
) {
  const logs = Array.isArray(logOrLogs) ? logOrLogs : [logOrLogs];

  if (logs.length === 0) {
    return Promise.resolve();
  }

  for (const log of logs) {
    // 1. Get the bucketId or create one if necessary
    const key: Key = `${log.workspaceId}::${BUCKET_NAME}`;
    let bucketId = "";
    const cachedBucketId = bucketCache.get(key);
    if (cachedBucketId) {
      bucketId = cachedBucketId;
    } else {
      const bucket = await db.query.auditLogBucket.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, log.workspaceId), eq(table.name, BUCKET_NAME)),
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
          workspaceId: log.workspaceId,
          name: BUCKET_NAME,
        });
      }
    }
    bucketCache.set(key, bucketId);

    // 2. Insert the log

    const auditLogId = newId("auditLog");
    await db.insert(schema.auditLog).values({
      id: auditLogId,
      workspaceId: log.workspaceId,
      bucketId,
      event: log.event,
      time: Date.now(),
      display: log.description,
      remoteIp: log.context.location,
      userAgent: log.context.userAgent,
      actorType: log.actor.type,
      actorId: log.actor.id,
      actorName: log.actor.name,
      actorMeta: log.actor.meta,
    });
    if (log.resources.length > 0) {
      await db.insert(schema.auditLogTarget).values(
        log.resources.map((r) => ({
          workspaceId: log.workspaceId,
          auditLogId,
          bucketId,
          displayName: "",
          type: r.type,
          id: r.id,
          name: "",
          meta: r.meta,
        })),
      );
    }
  }
}
