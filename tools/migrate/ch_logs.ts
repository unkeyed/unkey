import { mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";
import { z } from "zod";
import { createClient } from "@clickhouse/client-web";
import { ClickHouse } from "@unkey/clickhouse";

const tables = [
  {
    name: "default.key_verifications_per_minute_v2",
    dt: 24 * 60 * 60 * 1000,
    retention: 30 * 24 * 60 * 60 * 1000,
  },

  {
    name: "default.key_verifications_per_hour_v2",
    dt: 24 * 60 * 60 * 1000,
    retention: 30 * 24 * 60 * 60 * 1000,
  },
  {
    name: "default.key_verifications_per_day_v2",
    dt: 7 * 24 * 60 * 60 * 1000,
    retention: 6 * 30 * 24 * 60 * 60 * 1000,
  },
  {
    name: "default.key_verifications_per_month_v2",
    dt: 30 * 24 * 60 * 60 * 1000,
    retention: 3 * 356 * 24 * 60 * 60 * 1000,
  },
];

if (!process.env.CLICKHOUSE_URL) {
  throw new Error("CLICKHOUSE_URL is not set");
}

const ch = new ClickHouse({
  url: process.env.CLICKHOUSE_URL,
});

const rawCH = createClient({
  url: process.env.CLICKHOUSE_URL,

  clickhouse_settings: {
    output_format_json_quote_64bit_integers: 0,
    output_format_json_quote_64bit_floats: 0,
    http_send_timeout: 60000,
  },
});

const conn = await mysql.createConnection(
  `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`
);

await conn.ping();
const db = mysqlDrizzle(conn, { schema, mode: "default" });

const identityCache = new Map<string, string | null>();

const aggregatedSchema = z.object({
  workspace_id: z.string(),
  key_space_id: z.string(),
  identity_id: z.string(),
  external_id: z.string(),
});

let concurrency = 100;

for (const table of tables) {
  const end = Date.now();
  const start = end - table.retention;

  console.info("start", start, "end", end);
  const semaphore = new Map<string, Promise<void>>();

  for (let t = start; t < end; t += table.dt) {
    console.log(
      `${table.name}: ${new Date(t).toLocaleString()} - ${new Date(
        t + table.dt
      ).toLocaleString()}`
    );
    const query = ch.querier.query({
      query: `
    SELECT
      workspace_id,
      key_space_id,
      identity_id,
      external_id

    FROM
    ${table.name}
    WHERE time >= fromUnixTimestamp64Milli(${t})
    AND time < fromUnixTimestamp64Milli(${t + table.dt})
    AND identity_id != ''
    AND external_id == ''
    GROUP BY workspace_id, key_space_id, identity_id, external_id
   `,
      schema: aggregatedSchema,
    });
    const res = await query({});

    if (res.err) {
      console.error("query error", res.err);
      continue;
    }
    const rows = res.val;

    for (let i = 0; i < rows.length; i++) {
      console.info(
        `${table.name}:`,
        i + 1,
        "/",
        rows.length,
        `Concurrency: ${semaphore.size} / ${Math.floor(concurrency)}`
      );
      const row = rows[i];

      while (semaphore.size > concurrency) {
        await Promise.race(semaphore.values());
      }

      const key = `${t}-${i}`;
      semaphore.set(
        key,
        handleRow(table.name, row)
          .then(() => {
            concurrency = Math.min(500, concurrency + 10 / concurrency);
          })
          .catch((err) => {
            console.error("handleRow error", err);
            concurrency = Math.max(10, concurrency / 2);
          })
          .finally(() => {
            semaphore.delete(key);
          })
      );
    }
  }
  for (const p of semaphore.values()) {
    await p;
  }
}

async function handleRow(
  table: string,
  row: z.infer<typeof aggregatedSchema>
): Promise<void> {
  let externalId = identityCache.get(row.identity_id);
  if (externalId === null) {
    return;
  }
  if (!externalId) {
    const identity = await db.query.identities.findFirst({
      where: (table, { eq }) => eq(table.id, row.identity_id),
    });
    if (!identity) {
      console.error("identity not found", row.identity_id);
      identityCache.set(row.identity_id, null);
      return;
    }
    externalId = identity.externalId;
    identityCache.set(identity.id, identity.externalId);
  }

  await rawCH.query({
    query: `
    ALTER TABLE ${table}
    UPDATE external_id = '${externalId}'
    WHERE
    workspace_id = '${row.workspace_id}'
    AND key_space_id = '${row.key_space_id}'
    AND identity_id = '${row.identity_id}'
    AND external_id = ''
    `,
  });
}

process.exit(0);
