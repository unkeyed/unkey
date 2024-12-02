import type { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { z } from "zod";

import type { MaybeArray } from "@/lib/types";
import { type Database, type Transaction, schema } from "@unkey/db";
import { newId } from "@unkey/id";

export const auditLogsDataSchema = z
  .object({
    workspaceId: z.string(),
    bucket: z.string(),
    auditLogId: z.string(),
    time: z.number().int(),
    actorType: z.enum(["key", "user", "system"]),
    actorId: z.string(),
    actorName: z.string().nullable(),
    actorMeta: z.string().nullable(),
    event: z.string(),
    description: z.string(),
    resources: z.string().transform((rs) =>
      z
        .array(
          z.object({
            type: z.string(),
            id: z.string(),
            meta: z.record(z.union([z.string(), z.number(), z.boolean(), z.null()])).optional(),
          }),
        )
        .parse(JSON.parse(rs)),
    ),

    location: z.string(),
    userAgent: z.string().nullable(),
  })
  .transform((l) => ({
    workspaceId: l.workspaceId,
    bucket: l.bucket,
    auditLogId: l.auditLogId,
    time: l.time,
    actor: {
      type: l.actorType,
      id: l.actorId,
      name: l.actorName,
      meta: l.actorMeta ? JSON.parse(l.actorMeta) : undefined,
    },
    event: l.event,
    description: l.description,
    resources: l.resources,
    context: {
      location: l.location,
      userAgent: l.userAgent,
    },
  }));

export type UnkeyAuditLog = {
  workspaceId: string;
  event: z.infer<typeof unkeyAuditLogEvents>;
  description: string;
  actor: {
    type: "user" | "key";
    name?: string;
    id: string;
    meta?: Record<string, string | number | boolean | null>;
  };
  resources: Array<{
    type:
      | "key"
      | "api"
      | "workspace"
      | "role"
      | "permission"
      | "keyAuth"
      | "vercelBinding"
      | "vercelIntegration"
      | "ratelimitNamespace"
      | "ratelimitOverride"
      | "gateway"
      | "llmGateway"
      | "webhook"
      | "reporter"
      | "secret"
      | "identity"
      | "auditLogBucket";

    id: string;
    meta?: Record<string, string | number | boolean | null>;
  }>;
  context: {
    userAgent?: string;
    location: string;
  };
};

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
