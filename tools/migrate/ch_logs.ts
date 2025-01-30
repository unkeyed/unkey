import { ClickHouse } from "@unkey/clickhouse";
import { mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";
import { z } from "zod";
async function main() {
  const ch = new ClickHouse({
    url: process.env.CLICKHOUSE_URL,
  });

  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  const keySpaceCache = new Map<string, string>();

  const start = 1724930749353;
  const end = 1738212041696;
  const workspaceId = "ws_wB4SmWrYkhSbWE2rH61S6gMseWw";

  const query = ch.querier.query({
    query: `
    SELECT * FROM metrics.raw_api_requests_v1
    WHERE workspace_id = '${workspaceId}'
    AND time > ${start}
    AND time < ${end}
    AND path = '/v1/keys/verify'
    AND response_status = 200
    `,
    schema: z
      .object({
        request_id: z.string(),
        time: z.number(),
        workspace_id: z.string(),
        response_body: z.string(),
      })
      .passthrough(),
  });
  const res = await query({});

  const logs = res.val!;

  let i = 1;
  const inserts = [];
  for (const log of logs) {
    console.infi(i++, logs.length);

    const body = z
      .object({
        keyId: z.string().optional(),
        valid: z.boolean(),
        ownerId: z.string().optional(),
        remaining: z.number().optional(),
        code: z.string().optional(),
      })
      .safeParse(JSON.parse(log.response_body));
    if (!body.success) {
      console.error(body.error);
      console.error(log.response_body);
      continue;
    }

    if (!body.data.keyId) {
      continue;
    }

    let keySpaceId = keySpaceCache.get(body.data.keyId);
    if (!keySpaceId) {
      const key = await db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, body.data.keyId),
      });
      if (!key) {
        console.error("Key not found", body.data.keyId);
        continue;
      }
      keySpaceId = key.keyAuthId;
      keySpaceCache.set(body.data.keyId, keySpaceId);
    }

    const insert = {
      workspace_id: workspaceId,
      request_id: log.request_id,
      time: log.time,
      key_space_id: keySpaceId,
      key_id: body.data.keyId,
      region: "",
      tags: [],
      outcome: body.data.code ?? "VALID",
    };

    console.info(insert);
    inserts.push(insert);
  }
  await ch.verifications.insert(inserts);
}

main();
