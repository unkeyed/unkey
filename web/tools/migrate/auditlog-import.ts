import type { AuditLog, AuditLogTarget } from "@unkey/db";
import { newId } from "@unkey/id";
import ms from "ms";

export type Row = {
  workspaceId: string;
  namespaceId: string;
  requestId: string;
  identifier: string;
  time: string;
  servicelatency: number;
  success: number;
};

async function main() {
  const decoder = new TextDecoder();
  let buffer = "";

  const exportFile = Bun.file("./export_ratelimits.json");
  let lineCount = 0;

  const stream = exportFile.stream();
  const start = Date.now();
  for await (const chunk of stream) {
    buffer += decoder.decode(chunk);

    const auditLogs: AuditLog[] = [];
    const auditLogTargets: AuditLogTarget[] = [];

    const lines = buffer.split("\n");
    while (lines.length > 1) {
      const line = lines.shift();
      lineCount++;
      if (lineCount < 930583) {
        continue;
      }
      const timePerRow = Math.round((Date.now() - start) / lineCount);
      const _remainingTime = ms(timePerRow * (totalLines - lineCount));
      const log = JSON.parse(line!) as {
        workspaceId: string;
        bucket: string;
        auditLogId: string;
        event: string;
        time: number;
        actorType: string;
        actorId: string;
        actorName: string | null;
        actorMeta: string | null;
        description: string;
        resources: string;
        userAgent: string;
        location: string;
        meta: string | null;
      };

      if (log.workspaceId.startsWith("test_") || log.time < Date.now() - 90 * 24 * 60 * 60 * 1000) {
        continue;
      }
      let bucketId = "";
      const key: Key = `${log.workspaceId}::${log.bucket}`;
      const cachedBucketId = bucketCache.get(key);
      if (cachedBucketId) {
        bucketId = cachedBucketId;
      } else {
        const bucket = await db.query.auditLogBucket.findFirst({
          where: (table, { eq, and }) =>
            and(eq(table.workspaceId, log.workspaceId), eq(table.name, "unkey_mutations")),
        });
        if (bucket) {
          bucketId = bucket.id;
        } else {
          bucketId = newId("auditLogBucket");
          await db.insert(schema.auditLogBucket).values({
            id: bucketId,
            workspaceId: log.workspaceId,
            name: "unkey_mutations",
          });
        }
      }
      bucketCache.set(key, bucketId);

      const auditLogId = newId("auditLog");
      auditLogs.push({
        id: auditLogId,
        workspaceId: log.workspaceId,
        bucketId,
        event: log.event,
        time: log.time,
        // A human readable description of the event
        display: log.description,
        remoteIp: log.location,
        userAgent: log.userAgent,
        actorType: log.actorType,
        actorId: log.actorId,
        actorName: log.actorName,
        actorMeta: log.actorMeta ? JSON.parse(log.actorMeta) : null,
        createdAt: Date.now(),
        updatedAtM: null,
      });

      const resources: Array<{
        type: string;
        id: string;
        name: string | null;
        meta: unknown | null;
      }> = log.resources ? JSON.parse(log.resources) : [];

      for (const r of resources) {
        auditLogTargets.push({
          workspaceId: log.workspaceId,
          bucketId,
          auditLogId,
          displayName: r.name ?? "",
          type: r.type,
          id: r.id,
          name: r.name,
          meta: r.meta,
          createdAt: Date.now(),
          updatedAtM: null,
        });
      }
    }

    if (auditLogs.length > 0) {
      await db
        .insert(schema.auditLog)
        .values(auditLogs)
        .onDuplicateKeyUpdate({ set: { updatedAt: Date.now() } });
    }
    if (auditLogTargets.length > 0) {
      await db
        .insert(schema.auditLogTarget)
        .values(auditLogTargets)
        .onDuplicateKeyUpdate({
          set: {
            updatedAt: Date.now(),
          },
        });
    }

    buffer = lines[0];
  }
}

main();
