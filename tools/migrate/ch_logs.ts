import { Identity, mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";
import { z } from "zod";
import { createClient } from "@clickhouse/client-web";
import { ClickHouse } from "@unkey/clickhouse";

const tables = [
  // {
  //   name: "default.key_verifications_per_minute_v2",
  //   dt: 7 * 24 * 60 * 60 * 1000,
  //   retention: 40 * 24 * 60 * 60 * 1000,
  // },
  //
  // {
  //   name: "default.key_verifications_per_hour_v2",
  //   dt: 24 * 60 * 60 * 1000,
  //   retention: 40 * 24 * 60 * 60 * 1000,
  // },
  // {
  //   name: "default.key_verifications_per_day_v2",
  //   dt: 7 * 24 * 60 * 60 * 1000,
  //   retention: 7 * 30 * 24 * 60 * 60 * 1000,
  // },
  {
    name: "default.key_verifications_per_month_v2",
    dt: 30 * 24 * 60 * 60 * 1000,
    retention: 4 * 356 * 24 * 60 * 60 * 1000,
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
    mutations_sync: "2",
  },
});

const conn = await mysql.createConnection(
  `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`
);

await conn.ping();
const db = mysqlDrizzle(conn, { schema, mode: "default" });

const CACHE_FILE = "identity_cache.json";

// Load cache from file if it exists
let deletedIdentityCache = new Map<string, string | null>();
let migratedIdentities = new Map<string, boolean>();
try {
  const file = Bun.file(CACHE_FILE);
  if (await file.exists()) {
    const cacheData = await file.json();
    deletedIdentityCache = new Map(Object.entries(cacheData));
    console.info(
      `Loaded ${deletedIdentityCache.size} cached identities from ${CACHE_FILE}`
    );
  }
} catch (err) {
  console.warn("Failed to load cache file:", err);
}

// Function to save cache to file
async function saveCache() {
  try {
    const cacheData = Object.fromEntries(deletedIdentityCache);
    await Bun.write(CACHE_FILE, JSON.stringify(cacheData, null, 2));
    console.info(`Saved ${deletedIdentityCache.size} identities to cache file`);
  } catch (err) {
    console.error("Failed to save cache:", err);
  }
}

// Save cache on exit
process.on("SIGINT", async () => {
  console.info("\nSaving cache before exit...");
  await saveCache();
  process.exit(0);
});

process.on("SIGTERM", async () => {
  console.info("\nSaving cache before exit...");
  await saveCache();
  process.exit(0);
});

const aggregatedSchema = z.object({
  workspace_id: z.string(),
  key_space_id: z.string(),
  identity_id: z.string(),
  external_id: z.string(),
});

let concurrency = 1;

for (const table of tables) {
  const end = Date.now();
  const start = end - table.retention;

  console.info("start", start, "end", end);
  const semaphore = new Map<string, Promise<void>>();

  for (let t = start; t < end; t += table.dt) {
    console.log(
      `${table.name}: ${new Date(t).toLocaleString("de")} - ${new Date(
        t + table.dt
      ).toLocaleString("de")}`
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
    FINAL
    --WHERE workspace_id != 'ws_2vUFz88G6TuzMQHZaUhXADNyZWMy'
    WHERE time >= fromUnixTimestamp64Milli(${t})
    AND time < fromUnixTimestamp64Milli(${t + table.dt})
    AND identity_id != ''
    AND ( external_id = '' OR external_id = 'undefined' )
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

      if (migratedIdentities.has(row.identity_id)) {
        console.log("Identity already migrated");
        continue;
      }

      while (semaphore.size >= concurrency) {
        await Promise.race(semaphore.values());
      }

      const key = `${t}-${i}`;
      semaphore.set(
        key,
        handleRow(table.name, row)
          .then(() => {
            concurrency = Math.min(100, concurrency + 0.1);
          })
          .catch(async (err) => {
            console.error(err.message);
            concurrency = Math.max(1, concurrency / 2);
            await new Promise((resolve) => setTimeout(resolve, 10000));
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

// Save cache after processing all tables
await saveCache();

async function handleRow(
  table: string,
  row: z.infer<typeof aggregatedSchema>
): Promise<void> {
  let externalId = deletedIdentityCache.get(row.identity_id);
  if (externalId === null) {
    return;
  }
  let identity: Identity | undefined = undefined;
  if (!externalId) {
    identity = await db.query.identities.findFirst({
      where: (table, { eq }) => eq(table.id, row.identity_id),
    });
    if (!identity) {
      console.error("identity not found", row.identity_id);
      deletedIdentityCache.set(row.identity_id, null);
      await saveCache();
      return;
    }
    externalId = identity.externalId;
    deletedIdentityCache.set(identity.id, identity.externalId);
  }
  if (!externalId || !identity) {
    console.log({ identity });
    return;
  }
  migratedIdentities.set(identity.id, true);

  await rawCH.exec({
    query: `
    UPDATE ${table}
    SET external_id = '${externalId}'
    WHERE
    workspace_id = '${row.workspace_id}'
    AND key_space_id = '${row.key_space_id}'
    AND identity_id = '${row.identity_id}'
    AND ( external_id = '' OR external_id = 'undefined' )
    `,
  });
}

process.exit(0);
