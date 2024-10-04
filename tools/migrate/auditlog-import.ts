import { and, asc, eq, gt, isNotNull, mysqlDrizzle, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  const decoder = new TextDecoder();
  let buffer = "";

  const exportFile = Bun.file("./export_audit_log.json");

  const stream = exportFile.stream();
  for await (const chunk of stream) {
    buffer += decoder.decode(chunk);

    const lines = buffer.split("\n");
    while (lines.length > 1) {
      const line = lines.shift();
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
      let bucketId = "";
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

      const auditLogId = newId("auditLog");
      await db
        .insert(schema.auditLog)
        .values({
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
        })
        .onDuplicateKeyUpdate({ set: { updatedAt: Date.now() } });

      const resources: Array<{
        type: string;
        id: string;
        name: string | null;
        meta: unknown | null;
      }> = log.resources ? JSON.parse(log.resources) : [];

      await db.insert(schema.auditLogTarget).values(
        resources.map((r) => ({
          workspaceId: log.workspaceId,
          bucketId,
          auditLogId,
          displayName: r.name ?? "",
          type: r.type,
          id: r.id,
          name: r.name,
          meta: r.meta,
        })),
      );
    }
    buffer = lines[0];
  }
}

main();
