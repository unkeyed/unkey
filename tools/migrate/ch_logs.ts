import { ClickHouse } from "@unkey/clickhouse";
import { mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";
import { z } from "zod";
async function main() {
  const ch = new ClickHouse({
    url: process.env.CLICKHOUSE_URL,
  });

  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  const identityCache = new Map<string, string>();

  const aggregatedSchema = z.object({
    workspace_id: z.string(),
    key_space_id: z.string(),
    identity_id: z.string(),
    external_id: z.string(),
    key_id: z.string(),
    outcome: z.string(),
    tags: z.array(z.string()),
    count: z.number(),
    spent_credits: z.number(),
    latency_avg: z.number(),
    latency_p75: z.number(),
    latency_p99: z.number(),
    time: z.number(),
  });

  const insert = ch.inserter.insert({
    table: "default.key_verifications_per_minute_v3",
    schema: aggregatedSchema,
  });

  const start = 1724930749353;
  const end = 1738212041696;

  const dt = 5 * 60 * 1000;

  for (let t = start; t < end; t += dt) {
    console.log(
      `Processing time range: ${new Date(t).toLocaleString()} - ${new Date(
        t + dt
      ).toLocaleString()}`
    );
    const query = ch.querier.query({
      query: `
    SELECT
      workspace_id,
      key_space_id,
      identity_id,
      external_id,
      key_id,
      outcome,
      tags,
      count(*) as count,
      sum(spent_credits) as spent_credits,
      avgState (latency) as latency_avg,
      quantilesTDigestState (0.75) (latency) as latency_p75,
      quantilesTDigestState (0.99) (latency) as latency_p99,
      toStartOfMinute (fromUnixTimestamp64Milli (time)) AS time
    FROM
      key_verifications_raw_v2
    WHERE time >= fromUnixTimestamp64Milli(${t})
    AND time < fromUnixTimestamp64Milli(${t + dt})
    GROUP BY
      workspace_id,
      time,
      key_space_id,
      identity_id,
      external_id,
      key_id,
      outcome,
      tags
    `,
      schema: aggregatedSchema,
    });
    const res = await query({});

    const rows = res.val!;

    let i = 1;
    const inserts: Array<z.infer<typeof aggregatedSchema>> = [];
    for (const row of rows) {
      console.info(i++, rows.length);

      if (row.external_id === "") {
        const externalId = identityCache.get(row.identity_id);
        if (!externalId) {
          const identity = await db.query.identities.findFirst({
            where: (table, { eq }) => eq(table.id, row.identity_id),
          });
          if (!identity) {
            console.error("identity not found", row.identity_id);
            continue;
          }
          row.external_id = identity.externalId;
          identityCache.set(identity.id, identity.externalId);
        }
      }

      inserts.push(row);
    }

    console.log(inserts);
    //const insertRes = insert(inserts);
    //console.log(insertRes);
  }
}
main();
