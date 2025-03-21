import type { Context } from "@/pkg/hono/app";
import { auditLogSchemaV1, unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { type Transaction, schema } from "./db";

import { newId } from "@unkey/id";
import { z } from "zod";

const unkeyAuditLogSchema = auditLogSchemaV1.merge(
  z.object({
    event: unkeyAuditLogEvents,
    auditLogId: z.string().default(() => newId("auditLog")),
    bucket: z.string().default("unkey_mutations"),
    time: z.number().default(() => Date.now()),
  }),
);
export type UnkeyAuditLog = z.input<typeof unkeyAuditLogSchema>;

export async function insertUnkeyAuditLog(
  c: Context,
  tx: Transaction | undefined,
  auditLogs: UnkeyAuditLog | Array<UnkeyAuditLog>,
): Promise<void> {
  const schema = auditLogSchemaV1.merge(
    z.object({
      event: unkeyAuditLogEvents,
      auditLogId: z.string().default(() => newId("auditLog")),
      bucket: z.string().default("unkey_mutations"),
      time: z.number().default(() => Date.now()),
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

  const { db } = c.get("services");

  const auditLogsInserts = [];
  const auditLogTargetInserts = [];

  for (const log of arr) {
    const auditLogId = newId("auditLog");
    auditLogsInserts.push({
      id: auditLogId,
      workspaceId: log.workspaceId,
      bucketId: "DEPRECATED",
      bucket: "unkey_mutations",
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

    auditLogTargetInserts.push(
      ...log.resources.map((r) => ({
        workspaceId: log.workspaceId,
        bucketId: "DEPRECATED",
        bucket: "unkey_mutations",
        auditLogId,
        displayName: r.name ?? "",
        type: r.type,
        id: r.id,
        name: r.name,
        meta: r.meta,
      })),
    );
  }

  await (tx ?? db.primary).insert(schema.auditLog).values(auditLogsInserts);

  await (tx ?? db.primary).insert(schema.auditLogTarget).values(auditLogTargetInserts);
}
