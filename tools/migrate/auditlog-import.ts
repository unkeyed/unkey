import {
  type AuditLog,
  type AuditLogTarget,
  and,
  asc,
  eq,
  gt,
  isNotNull,
  mysqlDrizzle,
  schema,
} from "@unkey/db";
import { newId } from "@unkey/id";
import ms from "ms";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={"rejectUnauthorized":true}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  const decoder = new TextDecoder();
  let buffer = "";

  const exportFile = Bun.file("./export_audit_log.json");
  const totalLines = 8991960;
  let lineCount = 0;

  type Key = `${string}::${string}`;
  type BucketId = string;
  const bucketCache = new Map<Key, BucketId>();

  for (const workspace of await db.query.workspaces.findMany({
    with: {
      auditLogBuckets: true,
    },
  })) {
    console.log("workspace", workspace.id);
    if (workspace.auditLogBuckets.some((bucket) => bucket.name === "unkey_mutations")) {
      continue;
    }
    await db
      .insert(schema.auditLogBucket)
      .values({
        id: newId("auditLogBucket"),
        workspaceId: workspace.id,
        name: "unkey_mutations",
      })
      .onDuplicateKeyUpdate({ set: { updatedAt: Date.now() } });
  }

  for (const bucket of await db.query.auditLogBucket.findMany()) {
    bucketCache.set(`${bucket.workspaceId}::${bucket.name}`, bucket.id);
  }

  console.log("buckets cached", bucketCache.size);

  const stream = exportFile.stream();
  const start = Date.now();
  for await (const chunk of stream) {
    console.log("chunk");
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
      const remainingTime = ms(timePerRow * (totalLines - lineCount));
      console.log(
        `Line ${lineCount} /  ${totalLines} [ ${Intl.NumberFormat().format(
          (lineCount / 8991960) * 100,
        )}% ] - ${timePerRow} ms - ${remainingTime} remaining`,
      );
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
        console.log("uncached bucket", key);
        const bucket = await db.query.auditLogBucket.findFirst({
          where: (table, { eq, and }) =>
            and(eq(table.workspaceId, log.workspaceId), eq(table.name, "unkey_mutations")),
        });
        console.log({ bucket });
        if (bucket) {
          bucketId = bucket.id;
        } else {
          console.log("creating bucket");
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
        updatedAt: null,
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
          updatedAt: null,
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

  console.log("END");
}

main();
