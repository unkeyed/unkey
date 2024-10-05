import type { Context } from "@/pkg/hono/app";
import { auditLogSchemaV1, unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { type Transaction, schema } from "./db";

import { newId } from "@unkey/id";
import { z } from "zod";
import type { UnkeyAuditLog } from "./analytics";

export async function insertUnkeyAuditLog(
  c: Context,
  tx: Transaction | undefined,
  auditLogs: UnkeyAuditLog | Array<UnkeyAuditLog>,
): Promise<void> {
  const schema = auditLogSchemaV1.merge(
    z.object({
      event: unkeyAuditLogEvents,
      auditLogId: z.string().default(newId("auditLog")),
      bucket: z.string().default("unkey_mutations"),
      time: z.number().default(Date.now()),
    }),
  );

  const arr = Array.isArray(auditLogs) ? auditLogs : [auditLogs];
  return insertGenericAuditLogs(
    c,
    tx,
    arr.map((l) => schema.parse(l)),
  );
}

export async function insertGenericAuditLogs(
  c: Context,
  tx: Transaction | undefined,
  auditLogs: z.infer<typeof auditLogSchemaV1> | z.infer<typeof auditLogSchemaV1>[],
): Promise<void> {
  const arr = Array.isArray(auditLogs) ? auditLogs : [auditLogs];

  if (arr.length === 0) {
    return;
  }

  const { cache, logger, db } = c.get("services");

  for (const log of arr) {
    let { val: bucket, err } = await cache.auditLogBucketByWorkspaceIdAndName.swr(
      [log.workspaceId, log.bucket].join(":"),
      async () => {
        const bucket = await (tx ?? db.primary).query.auditLogBucket.findFirst({
          where: (table, { eq, and }) =>
            and(eq(table.workspaceId, log.workspaceId), eq(table.name, log.bucket)),
        });
        if (!bucket) {
          return undefined;
        }
        return {
          id: bucket.id,
        };
      },
    );
    if (err) {
      logger.error("Could not find audit log bucket for workspace", {
        workspaceId: log.workspaceId,
        error: err.message,
      });
      continue;
    }

    if (!bucket) {
      const bucketId = newId("auditLogBucket");
      await (tx ?? db.primary).insert(schema.auditLogBucket).values({
        id: bucketId,
        workspaceId: log.workspaceId,
        name: log.bucket,
        retentionDays: 90,
      });
      bucket = { id: bucketId };
    }

    const auditLogId = newId("auditLog");
    await (tx ?? db.primary).insert(schema.auditLog).values({
      id: auditLogId,
      workspaceId: log.workspaceId,
      bucketId: bucket.id,
      event: log.event,
      time: log.time,

      display: log.description ?? "",

      remoteIp: log.context?.location,

      userAgent: log.context?.userAgent,
      actorType: log.actor.type,
      actorId: log.actor.id,
      actorName: log.actor.name,
      actorMeta: log.actor.meta,
    });
    await (tx ?? db.primary).insert(schema.auditLogTarget).values(
      log.resources.map((r) => ({
        workspaceId: log.workspaceId,
        bucketId: bucket.id,
        auditLogId,
        displayName: r.name ?? "",
        type: r.type,
        id: r.id,
        name: r.name,
        meta: r.meta,
      })),
    );
  }
}
